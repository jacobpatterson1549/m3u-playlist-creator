package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

type songReader struct {
	fsys         fs.FS
	addHash      bool
	loadThreads  int
	pathSuffixes []string
}

func (sr songReader) readSongs(w io.Writer) ([]song, error) {
	var paths []string
	if err := fs.WalkDir(sr.fsys, ".", sr.walkDir(&paths, w)); err != nil {
		return nil, fmt.Errorf("walking directory: %v", err)
	}
	return sr.readPaths(w, paths)
}

func (sr songReader) readPaths(w io.Writer, paths []string) ([]song, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	pathsC := make(chan string, len(paths))
	for _, path := range paths {
		pathsC <- path
	}
	songs := make([]song, 0, len(paths))
	resultsC := make(chan readResult)
	if sr.loadThreads < 1 {
		sr.loadThreads = 1
	}
	for i := 0; i < sr.loadThreads; i++ {
		go func() {
			for path := range pathsC {
				resultsC <- sr.readSong(path)
			}
		}()
	}
	dc := digitCount(len(paths))
	cursorUp := "\033[1A"
	format := fmt.Sprintf("%v> reading songs: [%%%dd/%d]\n", cursorUp, dc, len(paths))
	resultID := 0
	start := time.Now()
	fmt.Println(w)
	for rr := range resultsC {
		switch {
		case rr.tagErr:
			fmt.Fprintf(w, "%v> parsing tags for %v: %v\n\n", cursorUp, rr.path, rr.err)
		case rr.err != nil:
			return nil, rr.err
		default:
			songs = append(songs, *rr.song)
		}
		resultID++
		fmt.Fprintf(w, format, resultID)
		if resultID == len(paths) {
			break
		}
	}
	d := time.Since(start).Seconds()
	fmt.Fprintf(w, "> loaded %v songs with %v errors in in %0.1f seconds\n", len(songs), len(paths)-len(songs), d)
	return songs, nil
}

func (sr songReader) validPath(p string) bool {
	for _, suffix := range sr.pathSuffixes {
		if strings.HasSuffix(p, suffix) {
			return true
		}
	}
	return false
}

func (sr songReader) walkDir(paths *[]string, w io.Writer) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil, d.IsDir(), !sr.validPath(path):
			// NOOP
		default:
			*paths = append(*paths, path)
		}
		return nil
	}
}

type readResult struct {
	song   *song
	err    error
	path   string
	tagErr bool
}

func (sr songReader) readSong(path string) readResult {
	f, err := sr.fsys.Open(path)
	if err != nil {
		return readResult{err: err, path: path}
	}
	defer f.Close()
	rs := f.(io.ReadSeeker)
	m, err := tag.ReadFrom(rs)
	if err != nil {
		return readResult{err: err, path: path, tagErr: true}
	}
	track, _ := m.Track()
	s := song{
		path:   path,
		album:  m.Album(),
		artist: m.Artist(),
		title:  m.Title(),
		track:  track,
	}
	if sr.addHash {
		rs.Seek(0, io.SeekStart)
		b, err := io.ReadAll(rs)
		if err != nil {
			return readResult{err: err, path: path}
		}
		h := md5.Sum(b)
		s.hash = fmt.Sprintf("%x", h)
	}
	return readResult{song: &s}
}
