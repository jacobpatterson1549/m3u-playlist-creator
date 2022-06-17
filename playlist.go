package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

type playlistFS interface {
	fs.StatFS
	fs.ReadFileFS
	WriteFile(name string, data []byte) error
}

type playlist struct {
	songs     []song
	selection []song
	tracks    []m3uTrack
	fsys      playlistFS
	w         io.Writer
}

type m3uTrack struct {
	song
	display string
}

func newPlaylist(songs []song, fsys playlistFS, w io.Writer) *playlist {
	p := playlist{
		songs:     make([]song, len(songs)),
		selection: make([]song, 0, len(songs)),
		fsys:      fsys,
		w:         w,
	}
	copy(p.songs, songs)
	sort.Slice(p.songs, songLess(p.songs))
	return &p
}

// filter limits the songs to be displayed and selected
func (p *playlist) filter(command string) {
	p.selection = p.selection[:0]
	for _, s := range p.songs {
		if s.matches(command) {
			p.selection = append(p.selection, s)
		}
	}
	p.printSongFilter("")
}

// printSongFilter displays the filtered songs
func (p *playlist) printSongFilter(_ string) {
	maxWidth := func(start int, f func(s song) int) int {
		maxW := start
		for _, s := range p.selection {
			if w := f(s); maxW < w {
				maxW = w
			}
		}
		return maxW
	}
	maxIDWidth := digitCount(len(p.selection))
	if maxIDWidth < 2 {
		maxIDWidth = 2
	}
	maxArtistWidth := maxWidth(6, func(s song) int { return len(s.artist) })
	maxAlbumWidth := maxWidth(5, func(s song) int { return len(s.album) })
	format := fmt.Sprintf("%%%dv    %%-%dv    %%-%dv    %%v\n", maxIDWidth, maxArtistWidth, maxAlbumWidth)
	fmt.Fprintf(p.w, format, "ID", "Artist", "Album", "Title")
	for i, s := range p.selection {
		fmt.Fprintf(p.w, format, i+1, s.artist, s.album, s.title)
	}
}

// addTrack adds a song from the last filter to the playlist by id
func (p *playlist) addTrack(selectionID string) {
	if len(p.selection) == 0 {
		fmt.Fprintf(p.w, "Error (add track): no selection\n")
		return
	}
	id, err := strconv.Atoi(selectionID)
	if err != nil || id <= 0 || id > len(p.selection) {
		fmt.Fprintf(p.w, "Error (add track): reading song id %q from selection. Must be in (1-%v): %v\n", selectionID, len(p.selection), err)
		return
	}
	s := p.selection[id-1]
	t := m3uTrack{
		song:    s,
		display: s.display(),
	}
	p.tracks = append(p.tracks, t)
}

// removeTrack removes a song from the playlist by id
func (p *playlist) removeTrack(trackID string) {
	if len(p.tracks) == 0 {
		fmt.Fprintf(p.w, "Error (remove track): no selection\n")
		return
	}
	id, err := strconv.Atoi(trackID)
	if err != nil || id <= 0 || id > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (remove track): reading track id %q from playlist. Must be in (1-%v): %v\n", trackID, len(p.tracks), err)
		return
	}
	id-- // make 1-indexed
	copy(p.tracks[id:], p.tracks[id+1:])
	p.tracks = p.tracks[:len(p.tracks)-1]
}

// moveTrack moves a song in the playlist to have a new id
func (p *playlist) moveTrack(command string) {
	f := strings.Fields(command)
	if len(f) != 2 {
		fmt.Fprintf(p.w, "Error (move track): wanted track id and move index\n")
		return
	}
	trackID, moveIndex := f[0], f[1]
	id, err := strconv.Atoi(trackID)
	if err != nil || id <= 0 || id > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (move track): reading track id %q from playlist. Must be in (1-%v): %v\n", trackID, len(p.tracks), err)
		return
	}
	id-- // make 1-indexed
	idx, err := strconv.Atoi(moveIndex)
	if err != nil || idx <= 0 || idx > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (move track): reading track id %q from playlist. Must be in at least 1: %v\n", idx, err)
		return
	}
	idx-- // make 1-indexed
	switch {
	case id == idx:
		// NOOP
	case idx < id: // move to index, shifting others right
		t := p.tracks[id]
		copy(p.tracks[idx+1:], p.tracks[idx:id])
		p.tracks[idx] = t
	case id < idx: // move to index, shifting others left
		t := p.tracks[id]
		copy(p.tracks[id:], p.tracks[id+1:idx+1])
		p.tracks[idx] = t
	}
}

// renameTrack changes the display name of a track by id
func (p *playlist) renameTrack(command string) {
	f := strings.Fields(command)
	if len(f) < 2 {
		fmt.Fprintf(p.w, "Error (rename track): wanted track id and move index\n")
		return
	}
	trackID := f[0]
	id, err := strconv.Atoi(trackID)
	if err != nil || id <= 0 || id > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (rename track): reading track id %q from playlist. Must be in (1-%v): %v\n", trackID, len(p.tracks), err)
		return
	}
	id-- // make 1-indexed
	displayStart := strings.Index(command, f[1])
	display := command[displayStart:]
	p.tracks[id].display = display
}

// print songs lists the tracks in the playlist
func (p *playlist) printTracks(_ string) {
	maxWidth := func(start int, f func(t m3uTrack) int) int {
		maxW := start
		for _, s := range p.tracks {
			if w := f(s); maxW < w {
				maxW = w
			}
		}
		return maxW
	}
	maxIDWidth := digitCount(len(p.tracks))
	if maxIDWidth < 2 {
		maxIDWidth = 2
	}
	maxDisplayWidth := maxWidth(7, func(t m3uTrack) int { return len(t.display) })
	maxArtistWidth := maxWidth(6, func(t m3uTrack) int { return len(t.artist) })
	maxAlbumWidth := maxWidth(5, func(t m3uTrack) int { return len(t.album) })
	format := fmt.Sprintf("%%%dv    %%-%dv    %%-%dv    %%-%dv    %%v\n", maxIDWidth, maxDisplayWidth, maxArtistWidth, maxAlbumWidth)
	fmt.Fprintf(p.w, format, "ID", "Display", "Artist", "Album", "Title")
	for i, t := range p.tracks {
		fmt.Fprintf(p.w, format, i+1, t.display, t.artist, t.album, t.title)
	}
}

// clearTracks removes the tracks from the playlist
func (p *playlist) clearTracks(_ string) {
	p.tracks = nil
}

// load imports a playlist by name
func (p *playlist) load(m3uPath string) {
	buf, err := p.fsys.ReadFile(m3uPath)
	if err != nil {
		fmt.Fprintf(p.w, "Error (load playlist): loading playlist file: %v\n", err)
		return
	}
	songPaths := make(map[string]song, len(p.songs))
	for _, s := range p.songs {
		songPaths[s.path] = s
	}
	r := bytes.NewBuffer(buf)
	br := bufio.NewReader(r)
	s := bufio.NewScanner(br)
	var tracks []m3uTrack
	var errors []string
	const maxErrors = 10
	display := ""
	for s.Scan() {
		line := s.Text()
		if len(line) == 0 {
			continue
		}
		switch {
		case line[0] == '#':
			// comment
			if strings.HasPrefix(line, "#EXTINF:") {
				// #EXTINF:123, Display
				if commaIndex := strings.Index(line, ","); commaIndex > 0 {
					display = strings.TrimSpace(line[commaIndex+1:])
				}
			}
		default:
			// treat line as path
			if s, ok := songPaths[line]; ok {
				if len(display) == 0 {
					display = s.display()
				}
				t := m3uTrack{
					song:    s,
					display: display,
				}
				tracks = append(tracks, t)
			} else {
				// path not found
				switch {
				case len(errors) < maxErrors:
					errors = append(errors, fmt.Sprintf("Error (load playlist): music file not found: %q", line))
				case len(errors) == maxErrors:
					errors = append(errors, "Error (load playlist): ... additional errors not displayed")
				}
			}
			display = ""
		}
	}
	p.tracks = tracks
	for _, e := range errors {
		fmt.Fprintln(p.w, e)
	}
}

// write exports the playlist to a new file by name
func (p *playlist) write(m3uPath string) {
	if !strings.HasSuffix(m3uPath, ".m3u") || len(m3uPath) == 4 {
		fmt.Fprintf(p.w, "Error (write playlist): path must end with .m3u, got %q\n", m3uPath)
		return
	}
	f, err := p.fsys.Stat(m3uPath)
	if err == nil || f != nil {
		fmt.Fprintf(p.w, "Error (write playlist): m3u file exists or is corrupted: %v\n", err)
		return
	}
	data := []byte("#EXTM3U\r\n")
	for _, t := range p.tracks {
		data = append(data, fmt.Sprintf("#EXTINF:0, %v\r\n%v\r\n", t.display, t.path)...)
	}
	_ = f
	if err := p.fsys.WriteFile(m3uPath, data); err != nil {
		fmt.Fprintf(p.w, "Error (write playlist): writing m3u file: %v\n", err)
	}
}

// songLess creates a song function that compares song indices by artist, album, track, then title
func songLess(s []song) func(i, j int) bool {
	return func(i, j int) bool {
		if s[i].artist != s[j].artist {
			return s[i].artist < s[j].artist
		}
		if s[i].album != s[j].album {
			return s[i].album < s[j].album
		}
		if s[i].track != s[j].track {
			return s[i].track < s[j].track
		}
		return s[i].title < s[j].title
	}
}

// digitCount computes the number of digits in the positive number.
// It is slower but simpler than using log10 with floats, which would use
// int(math.Log10(float64(i) + 1)))
func digitCount(i int) int {
	if i < 0 {
		return 0
	}
	c := 0
	for i != 0 {
		i /= 10
		c++
	}
	return c
}
