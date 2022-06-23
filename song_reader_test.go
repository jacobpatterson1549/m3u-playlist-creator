package main

import (
	"testing"
	"testing/fstest"
)

func TestSongReaderValidPath(t *testing.T) {
	tests := []struct {
		name          string
		pathSuffixes []string
		path          string
		want          bool
	}{
		{"empty songReader", nil, "song.mp3", false},
		{"empty paths matches nothing 1", nil, "", false},
		{"empty paths matches nothing 2", []string{".mp3"}, "", false},
		{"basic mp3", []string{".mp3"}, "song.mp3", true},
		{"basic mp3", []string{".mp3"}, "song.doc", false},
		{"m4a", []string{".mp3", ".m4a", ".ogg"}, "song.m4a", true},
		{"case-sensitive", []string{".MP3"}, "song.mp3", false},
		{"must be suffix 1", []string{".mp3"}, ".mp3.docx", false},
		{"must be suffix 2", []string{".mp3"}, "secrets.mp3.doc", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sr := songReader{pathSuffixes: test.pathSuffixes}
			if want, got := test.want, sr.validPath(test.path); want != got {
				t.Error()
			}
		})
	}
}

func TestSongReaderReadSongs(t *testing.T) {
	tests := []struct {
		name    string
		sr      songReader
		want    []song
		wantErr bool
	}{
		{
			name: "0 files => 0 songs, ok",
			sr: songReader{
				fsys: fstest.MapFS{},
			},
		},
		{
			name: "ok",
			sr: songReader{
				pathSuffixes: []string{".mp3"},
				fsys: fstest.MapFS{
					"a.mp3": &fstest.MapFile{
						Data: emptyMP3,
					},
					"b/c/d.mp3": &fstest.MapFile{
						Data: mockMp3(song{
							track: 2,
							// these will get padded with spaces
							artist: "Beck",
							album:  "Guero",
							title:  "E-Pro",
						}),
					},
					"b/c/e.mp3": &fstest.MapFile{
						Data: mockMp3(song{
							track: 1,
							/// these will get truncated to 10 characters
							artist: "Eagles Of Death Metal",
							album:  "Peace Love Death Metal",
							title:  "I Only Want You",
						}),
					},
					"b/c/notes.txt": &fstest.MapFile{
						Data: []byte("do not read this file"),
					},
				},
			},
			want: []song{
				{
					path:   "a.mp3",
					artist: emptyMp3Artist,
					album:  emptyMp3Album,
					title:  emptyMp3Title,
					track:  1549,
				},
				{
					path:   "b/c/d.mp3",
					artist: "Beck      ",
					album:  "Guero     ",
					title:  "E-Pro     ",
					track:  2,
				},
				{
					path:   "b/c/e.mp3",
					artist: "Eagles Of ",
					album:  "Peace Love",
					title:  "I Only Wan",
					track:  1,
				},
			},
		},
		{
			name: "bad first song",
			sr: songReader{
				pathSuffixes: []string{".mp3"},
				fsys: fstest.MapFS{
					"bad.mp3": &fstest.MapFile{
						Data: []byte("UNKNOWN"),
					},
					"c.mp3": &fstest.MapFile{
						Data: emptyMP3,
					},
				},
			},
			wantErr: true,
		},
	}
	songsEqual := func(a, b []song) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.sr.readSongs()
			gotErr := err != nil
			switch {
			case test.wantErr != gotErr:
				t.Errorf("wanted error: %v, got: %v", test.wantErr, err)
			case !songsEqual(test.want, got):
				t.Errorf("songs not equal: \n wanted: %v \n got:    %v", test.want, got)
			}
		})
	}
}
