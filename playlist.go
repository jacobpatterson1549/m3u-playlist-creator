package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

type playlistFS interface {
	fs.FS
	CreateFile(name string) (io.WriteCloser, error)
}

type playlist struct {
	songs     []song
	selection []song
	tracks    []m3uTrack
	fsys      playlistFS
	w         io.Writer
	showHash  bool
}

type m3uTrack struct {
	song
	display string
}

func newPlaylist(songs []song, fsys playlistFS, w io.Writer, showHash bool) *playlist {
	p := playlist{
		songs:     make([]song, len(songs)),
		selection: make([]song, 0, len(songs)),
		fsys:      fsys,
		w:         w,
		showHash:  showHash,
	}
	copy(p.songs, songs)
	sort.Slice(p.songs, songLess(p.songs))
	return &p
}

// filter limits the songs to be displayed and selected
func (p *playlist) filter(command string) {
	p.selection = p.selection[:0]
	for _, s := range p.songs {
		if s.matches(command, p.showHash) {
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
	hashFormat := "%32v    "
	format := fmt.Sprintf("%%%dv    %%-%dv    %%-%dv    %%v\n", maxIDWidth, maxArtistWidth, maxAlbumWidth)
	if p.showHash {
		fmt.Fprintf(p.w, hashFormat, "Hash")
	}
	fmt.Fprintf(p.w, format, "ID", "Artist", "Album", "Title")
	for i, s := range p.selection {
		if p.showHash {
			fmt.Fprintf(p.w, hashFormat, s.hash)
		}
		id := i + 1
		fmt.Fprintf(p.w, format, id, s.artist, s.album, s.title)
	}
}

// addTrack adds a song from the last filter to the playlist by id
func (p *playlist) addTrack(filterID string) {
	if len(p.selection) == 0 {
		fmt.Fprintf(p.w, "Error (add track): no selection\n")
		return
	}
	id, err := strconv.Atoi(filterID)
	if err != nil || id <= 0 || id > len(p.selection) {
		fmt.Fprintf(p.w, "Error (add track): reading song id %q from selection. Must be in (1-%v): %v\n", filterID, len(p.selection), err)
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
func (p *playlist) removeTrack(idx string) {
	if len(p.tracks) == 0 {
		fmt.Fprintf(p.w, "Error (remove track): no selection\n")
		return
	}
	id, err := strconv.Atoi(idx)
	if err != nil || id <= 0 || id > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (remove track): reading track index %q from playlist. Must be in (1-%v): %v\n", idx, len(p.tracks), err)
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
	trackIdx, moveIndex := f[0], f[1]
	id, err := strconv.Atoi(trackIdx)
	if err != nil || id <= 0 || id > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (move track): reading track index %q from playlist. Must be in (1-%v): %v\n", trackIdx, len(p.tracks), err)
		return
	}
	id-- // make 1-indexed
	destIdx, err := strconv.Atoi(moveIndex)
	if err != nil || destIdx <= 0 || destIdx > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (move track): reading track destination index %q from playlist. Must be in (1-%v): %v\n", destIdx, len(p.tracks), err)
		return
	}
	destIdx-- // make 1-indexed
	switch {
	case id == destIdx:
		// NOOP
	case destIdx < id: // move to index, shifting others right
		t := p.tracks[id]
		copy(p.tracks[destIdx+1:], p.tracks[destIdx:id])
		p.tracks[destIdx] = t
	case id < destIdx: // move to index, shifting others left
		t := p.tracks[id]
		copy(p.tracks[id:], p.tracks[id+1:destIdx+1])
		p.tracks[destIdx] = t
	}
}

// renameTrack changes the display name of a track by id
func (p *playlist) renameTrack(command string) {
	f := strings.Fields(command)
	if len(f) < 2 {
		fmt.Fprintf(p.w, "Error (rename track): wanted track id and move index\n")
		return
	}
	trackIdx := f[0]
	id, err := strconv.Atoi(trackIdx)
	if err != nil || id <= 0 || id > len(p.tracks) {
		fmt.Fprintf(p.w, "Error (rename track): reading track id %q from playlist. Must be in (1-%v): %v\n", trackIdx, len(p.tracks), err)
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
	if maxIDWidth < 5 {
		maxIDWidth = 5
	}
	maxDisplayWidth := maxWidth(7, func(t m3uTrack) int { return len(t.display) })
	maxArtistWidth := maxWidth(6, func(t m3uTrack) int { return len(t.artist) })
	maxAlbumWidth := maxWidth(5, func(t m3uTrack) int { return len(t.album) })
	hashFormat := "%32v    "
	format := fmt.Sprintf("%%%dv    %%-%dv    %%-%dv    %%-%dv    %%v\n", maxIDWidth, maxDisplayWidth, maxArtistWidth, maxAlbumWidth)
	if p.showHash {
		fmt.Fprintf(p.w, hashFormat, "Hash")
	}
	fmt.Fprintf(p.w, format, "Index", "Display", "Artist", "Album", "Title")
	for i, t := range p.tracks {
		if p.showHash {
			fmt.Fprintf(p.w, hashFormat, t.hash)
		}
		idx := i + 1
		fmt.Fprintf(p.w, format, idx, t.display, t.artist, t.album, t.title)
	}
}

// clearTracks removes the tracks from the playlist
func (p *playlist) clearTracks(_ string) {
	p.tracks = nil
}

// load imports a playlist by name
func (p *playlist) load(m3uPath string) {
	f, err := p.fsys.Open(m3uPath)
	if err != nil {
		fmt.Fprintf(p.w, "Error (load playlist): loading playlist file: %v\n", err)
		return
	}
	if _, err := p.ReadFrom(f); err != nil {
		fmt.Fprintf(p.w, "Error (load playlist): %v", err)
	}
}

// ReadFrom reads the playlist tracks from the reader, updating the playlist contain all valid songs in the file
func (p *playlist) ReadFrom(r io.Reader) (n int64, err error) {
	songPaths := make(map[string]song, len(p.songs))
	for _, s := range p.songs {
		songPaths[s.path] = s
	}
	s := bufio.NewScanner(r)
	var tracks []m3uTrack
	var t m3uTrack
	var errors []string
	const maxErrors = 10
	display := ""
	for s.Scan() {
		line := s.Text()
		n += int64(len(line))
		switch {
		case len(line) == 0:
			// NOOP
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
			t, err = getTrack(line, songPaths, display)
			switch {
			case err == nil:
				tracks = append(tracks, t)
			case len(errors) < maxErrors:
				errors = append(errors, fmt.Sprintf("song not found: %q", line))
			case len(errors) == maxErrors:
				errors = append(errors, "... additional song load errors not displayed")
			}
			display = ""
		}
	}
	switch {
	case s.Err() != nil:
		err = fmt.Errorf("reading playlist file: %v", err)
	case len(errors) != 0:
		err = fmt.Errorf("loading playlist songs: %v", strings.Join(errors, "\n"))
	}
	p.tracks = tracks
	return
}

func getTrack(line string, songPaths map[string]song, display string) (t m3uTrack, err error) {
	s, ok := songPaths[line]
	if !ok {
		err = fmt.Errorf("song not found: %q", line)
		return
	}
	if len(display) == 0 {
		display = s.display()
	}
	t = m3uTrack{
		song:    s,
		display: display,
	}
	return t, nil
}

// write exports the playlist to a new file by name
func (p *playlist) write(m3uPath string) {
	if !strings.HasSuffix(m3uPath, ".m3u") || len(m3uPath) == 4 {
		fmt.Fprintf(p.w, "Error (write playlist): path must end with .m3u, got %q\n", m3uPath)
		return
	}
	f, err := p.fsys.CreateFile(m3uPath)
	if err != nil {
		fmt.Fprintf(p.w, "Error (write playlist): creating file: %v", err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(p.w, "Error (closing %q): %v", m3uPath, err)
		}
	}()
	if _, err := p.WriteTo(f); err != nil {
		fmt.Fprintf(p.w, "Error (writing tracks): %v", err)
		return
	}
}

// WriteTo writes the tracks of the playlist
func (p playlist) WriteTo(w io.Writer) (n int64, err error) {
	n2, err := fmt.Fprint(w, "#EXTM3U\r\n")
	n += int64(n2)
	for i := 0; err == nil && i < len(p.tracks); i++ {
		t := p.tracks[i]
		n2, err = fmt.Fprintf(w, "#EXTINF:0, %v\r\n%v\r\n", t.display, t.path)
		n += int64(n2)
	}
	return
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
	c := 0
	for i > 0 {
		i /= 10
		c++
	}
	return c
}
