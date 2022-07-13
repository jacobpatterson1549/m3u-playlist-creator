package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/dhowden/tag"
)

type songReader struct {
	fsys         fs.FS
	addHash      bool
	pathSuffixes []string
}

func (sr songReader) readSongs(w io.Writer) ([]song, error) {
	var songs []song
	if err := fs.WalkDir(sr.fsys, ".", sr.walkDir(&songs, w)); err != nil {
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

func (sr songReader) walkDir(songs *[]song, w io.Writer) func(path string, d fs.DirEntry, err error) error {
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
		rs := f.(io.ReadSeeker)
		m, err := tag.ReadFrom(rs)
		if err != nil {
			fmt.Fprintf(w, "parsing tags for %v: %v\n", path, err)
			return nil
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
				return fmt.Errorf("reading file to get hash: %v", err)
			}
			h := md5.Sum(b)
			s.hash = fmt.Sprintf("%x", h)
		}
		*songs = append(*songs, s)
		return nil
	}
}
