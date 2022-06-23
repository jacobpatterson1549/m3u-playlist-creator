package main

import (
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/dhowden/tag"
)

type songReader struct {
	fsys         fs.FS
	pathSuffixes []string
}

func (sr songReader) readSongs() ([]song, error) {
	var songs []song
	if err := fs.WalkDir(sr.fsys, ".", sr.walkDir(&songs)); err != nil {
		return nil, fmt.Errorf("walking directory: %v", err)
	}
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

func (sr songReader) walkDir(songs *[]song) func(path string, d fs.DirEntry, err error) error {
	return func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil, d.IsDir(), !sr.validPath(path):
			return nil
		}
		f, err := sr.fsys.Open(path)
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
		*songs = append(*songs, s)
		return nil
	}
}
