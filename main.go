package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

func main() {
	r := os.Stdin
	w := os.Stdout
	start := time.Now()
	t := time.NewTicker(1 * time.Second) // change to Millisecond when testing
	go func() {
		for range t.C {
			fmt.Fprint(w, ".")
		}
	}()
	fs := os.DirFS(".")
	songs, err := readSongs(fs)
	t.Stop()
	if d := time.Since(start).Seconds(); d > 1 {
		fmt.Fprintf(w, "\n> (loaded in %0.1f seconds)\n", d)
	}
	switch {
	case err != nil:
		fmt.Fprintf(w, "Error (reading songs): %v\n", err)
	case len(songs) == 0:
		fmt.Fprintf(w, "no songs in folder to add to playlists\n")
	default:
		fmt.Fprintf(w, "> loaded %d songs\n", len(songs))
		fsys := osFS{
			FS: fs,
			createFileFunc: func(name string) (io.WriteCloser, error) {
				return os.Create(name)
			},
		}
		fsys.runPlaylistCreator(songs, r, w)
	}
}

type osFS struct {
	fs.FS
	createFileFunc func(name string) (io.WriteCloser, error)
}

func (fsys osFS) CreateFile(name string) (io.WriteCloser, error) {
	if !fs.ValidPath(name) {
		return nil, fmt.Errorf("%q bust be relative to application root", name)
	}
	_, err := fs.Stat(fsys, name)
	if _, ok := err.(*os.PathError); !ok {
		return nil, fmt.Errorf("%q already exists or could not be checked: %v", name, err)
	}
	return fsys.createFileFunc(name)
}

func readSongs(fsys fs.FS) ([]song, error) {
	var songs []song
	// MP3, M4A, M4B, M4P, ALAC, FLAC, OGG, and DSF is supported by github.com/dhowden/tag
	validSuffixes := []string{".mp3", ".m4a"}
	valid := func(path string) bool {
		for _, suffix := range validSuffixes {
			if strings.HasSuffix(path, suffix) {
				return true
			}
		}
		return false
	}
	walkDir := func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil, d.IsDir(), !valid(path):
			return nil
		}
		f, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("reading %v: %v", path, err)
		}
		defer f.Close()
		m, err := tag.ReadFrom(f.(io.ReadSeeker))
		if err != nil {
			return fmt.Errorf("parsing tags for %v: %v", path, err)
		}
		track, _ := m.Track()
		s := song{
			path:   path,
			album:  m.Album(),
			artist: m.Artist(),
			title:  m.Title(),
			track:  track,
		}
		songs = append(songs, s)
		return nil
	}
	if err := fs.WalkDir(fsys, ".", walkDir); err != nil {
		return nil, fmt.Errorf("walking directory: %v", err)
	}
	return songs, nil
}

func (fsys *osFS) runPlaylistCreator(songs []song, r io.Reader, w io.Writer) {
	p := newPlaylist(songs, fsys, w)
	cmds := commands{
		{"f", p.filter, "Filter songs by the trailing text.  Songs are filtered by artist, album, and title, ignoring case.  Example: `F The Beatles` selects songs by The Beatles."},
		{"d", p.printSongFilter, "Displays the filtered songs."},
		{"a", p.addTrack, "Add song to the playlist by filter id.  The song id must be from the previous filter.  Example: `a 5` adds the fifth song from the last filter to the playlist."},
		{"m", p.moveTrack, "Moves song in playlist to other index.  Example: `m 3 1` moves the third song to be first in the playlist."},
		{"r", p.removeTrack, "Removes song at the index from the playlist.  Example: `r 4` removes the fourth song from the playlist."},
		{"n", p.renameTrack, "Renames the song in the playlist by id.  Example: `n 2 Best Ever` Renames the display name of the second song in the playlist to \"Best Ever\"."},
		{"c", p.clearTracks, "Clears all songs in the playlist."},
		{"p", p.printTracks, "Prints all songs in the playlist."},
		{"l", p.load, "Loads the playlist with the specified file name.  The previous playlist is discarded.  Example: `l lists/good.m3u` loads the \"good.m3u\" playlist in the \"lists\" subdirectory."},
		{"w", p.write, "Writes the playlist to the specified file name.  The file must not exist before the playlist is written to it.  Example: `w lists/new.m3u` save the playlist as \"new.m3u\" in the \"lists\" subdirectory."},
	}
	displayHelp(cmds, w)
	cmds.run(r, w)
}

type (
	command struct {
		key  string
		run  func(command string)
		info string
	}
	commands []command
)

func displayHelp(cmds commands, w io.Writer) {
	var lines []string
	lines = append(lines, "Help for m3u-playlist-create", "The program reads commands to create new playlists.")
	for _, c := range cmds {
		lines = append(lines, c.key+"    "+c.info)
	}
	lines = append(lines, "h    Help information is printed.")
	lines = append(lines, "q    Quits the application.")
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}

func (cmds commands) toLookup(w io.Writer) (map[string]func(s string), error) {
	cmdsCap := len(cmds) + 2
	lookup := make(map[string]func(string), cmdsCap)
	for _, c := range cmds {
		lookup[c.key] = c.run
	}
	lookup["h"] = func(s string) {
		displayHelp(cmds, w)
	}
	lookup["q"] = nil
	if len(lookup) != cmdsCap {
		return nil, fmt.Errorf("some commands have duplicate keys: wanted %v, got %v", cmdsCap, len(lookup))
	}
	return lookup, nil
}

func (cmds commands) run(r io.Reader, w io.Writer) {
	lookup, err := cmds.toLookup(w)
	if err != nil {
		fmt.Fprintf(w, "Error (preparing to run commands): %v", err)
		return
	}
	s := bufio.NewScanner(r)
	var (
		line, key, args string
		commandTokens   []string
		cmd             func(s string)
		ok              bool
	)
	for {
		fmt.Fprintf(w, "> ")
		if !s.Scan() {
			return
		}
		line = s.Text()
		if line == "q" {
			return
		}
		ok = false
		if commandTokens = strings.Fields(line); len(commandTokens) != 0 {
			key, args = commandTokens[0], strings.TrimSpace(line[len(commandTokens[0]):])
			cmd, ok = lookup[key]
			if ok {
				cmd(args)
			}
		}
		if !ok {
			fmt.Fprintf(w, "Error (invalid command): %v\n", line)
		}
	}
}
