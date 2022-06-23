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

type osFS struct {
	fs.FS
	createFileFunc func(name string) (io.WriteCloser, error)
}

func (fsys *osFS) CreateFile(name string) (io.WriteCloser, error) {
	if !fs.ValidPath(name) {
		return nil, fmt.Errorf("%q bust be relative to application root", name)
	}
	_, err := fs.Stat(fsys, name)
	if _, ok := err.(*os.PathError); !ok {
		return nil, fmt.Errorf("%q already exists or could not be checked: %v", name, err)
	}
	return fsys.createFileFunc(name)
}

func (fsys *osFS) runPlaylistCreator(songs []song, r io.Reader, w io.Writer) {
	p := newPlaylist(songs, fsys, w)
	cmds := commands{
		{"f", p.filter, "Filter songs with query: f <query>"},
		{"d", p.printSongFilter, "Display filter'd songs by id"},
		{"a", p.addTrack, "Add song song by filter id: a <id>"},
		{"m", p.moveTrack, "Move playlist track: m <old_index> <new_index>"},
		{"r", p.removeTrack, "Remove playlist track: r <index>"},
		{"n", p.renameTrack, "Rename playlist track: n <index> <name>"},
		{"c", p.clearTracks, "Clear playlist tracks"},
		{"p", p.printTracks, "Print playlist tracks and indexes"},
		{"l", p.load, "Loads playlist: l <filename>"},
		{"w", p.write, "Writes playlist: w <filename>"},
	}
	cmds.displayHelp(w)
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

func (cmds commands) displayHelp(w io.Writer) {
	lines := []string{
		"Help for m3u-playlist-create",
		"The application reads commands to create m3u playlists.",
		"First, songs must be selected by a filter.",
		"Then, songs can be added to playlist by filter id.",
		"Playlists tracks are referenced by their index.",
	}
	const tab = "    "
	for _, c := range cmds {
		lines = append(lines, c.key+tab+c.info)
	}
	lines = append(lines, "h"+tab+"Help information is printed.")
	lines = append(lines, "q"+tab+"Quits the application.")
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}

func (cmds commands) run(r io.Reader, w io.Writer) {
	cmdsCap := len(cmds) + 2
	lookup := make(map[string]func(string), cmdsCap)
	for _, c := range cmds {
		lookup[c.key] = c.run
	}
	lookup["h"] = func(s string) {
		cmds.displayHelp(w)
	}
	lookup["q"] = nil
	if len(lookup) != cmdsCap {
		fmt.Fprintf(w, "Error (preparing to run commands): some commands have duplicate keys: wanted %v, got %v", cmdsCap, len(lookup))
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
		if commandTokens = strings.Fields(line); len(commandTokens) != 0 {
			key, args = commandTokens[0], strings.TrimSpace(line[len(commandTokens[0]):])
			cmd, ok = lookup[key]
			if ok {
				cmd(args)
				continue
			}
		}
		fmt.Fprintf(w, "Error (invalid command): %v\n", line)
	}
}
