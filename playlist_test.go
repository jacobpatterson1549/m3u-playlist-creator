package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"testing/fstest"
)

func TestNewPlaylist(t *testing.T) {
	songs := []song{
		{artist: "b", album: "c", title: "d"},
		{artist: "a", album: "e", title: "f"},
	}
	want := playlist{
		songs: []song{
			{artist: "a", album: "e", title: "f"},
			{artist: "b", album: "c", title: "d"},
		},
	}
	var fsys playlistFS = osFS{}
	var w bytes.Buffer
	p := newPlaylist(songs, fsys, &w)
	checkPlaylistsEqual(t, want, *p)
	if want, got := len(songs), cap(p.selection); want != got {
		t.Errorf("selection not allocated: wanted %v, got %v", want, got)
	}
}

func TestPlaylistFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		src    playlist
		want   playlist
	}{
		{
			name:   "empty",
			filter: "",
			src: playlist{
				songs: []song{
					{artist: "David Bowie", album: "The Rise and Fall of Ziggy Stardust and Spiders from Mars", title: "Five Years"},
					{artist: "Queen", album: "Greatest Hits", title: "Another One Bites The Dust"},
					{artist: "The Shins", album: "Wincing The Night Away", title: "Australia"},
				},
			},
			want: playlist{
				songs: []song{
					{artist: "David Bowie", album: "The Rise and Fall of Ziggy Stardust and Spiders from Mars", title: "Five Years"},
					{artist: "Queen", album: "Greatest Hits", title: "Another One Bites The Dust"},
					{artist: "The Shins", album: "Wincing The Night Away", title: "Australia"},
				},
				selection: []song{
					{artist: "David Bowie", album: "The Rise and Fall of Ziggy Stardust and Spiders from Mars", title: "Five Years"},
					{artist: "Queen", album: "Greatest Hits", title: "Another One Bites The Dust"},
					{artist: "The Shins", album: "Wincing The Night Away", title: "Australia"},
				},
			},
		},
		{
			name:   "dust",
			filter: "dust",
			src: playlist{
				songs: []song{
					{artist: "David Bowie", album: "The Rise and Fall of Ziggy Stardust and Spiders from Mars", title: "Five Years"},
					{artist: "Queen", album: "Greatest Hits", title: "Another One Bites The Dust"},
					{artist: "The Shins", album: "Wincing The Night Away", title: "Australia"},
				},
			},
			want: playlist{
				songs: []song{
					{artist: "David Bowie", album: "The Rise and Fall of Ziggy Stardust and Spiders from Mars", title: "Five Years"},
					{artist: "Queen", album: "Greatest Hits", title: "Another One Bites The Dust"},
					{artist: "The Shins", album: "Wincing The Night Away", title: "Australia"},
				},
				selection: []song{
					{artist: "David Bowie", album: "The Rise and Fall of Ziggy Stardust and Spiders from Mars", title: "Five Years"},
					{artist: "Queen", album: "Greatest Hits", title: "Another One Bites The Dust"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.src.w = &w
			test.src.filter(test.filter)
			checkPlaylistsEqual(t, test.want, test.src)
			if w.Len() == 0 {
				t.Error("wanted song filter to be printed")
			}
		})
	}
}

func TestPlaylistPrintFilter(t *testing.T) {
	tests := []struct {
		name string
		p    playlist
		want string
	}{
		{
			name: "short list",
			p: playlist{
				selection: []song{
					{artist: "x", album: "y", track: 8, title: "z"},
				},
			},
			want: `ID    Artist    Album    Title
 1    x         y        z
`,
		},
		{
			name: "eleven songs",
			p: playlist{
				selection: []song{
					{artist: "Beck", album: "Guero", track: 4, title: "Missing"},
					{artist: "The Killers", album: "Hot Fuss", track: 1, title: "Jenny Was A Friend Of Mine"},
					{artist: "The Killers", album: "Hot Fuss", track: 2, title: "Mr. Brightside"},
					{artist: "The Killers", album: "Hot Fuss", track: 3, title: "Smile Like You Mean It"},
					{artist: "The Killers", album: "Hot Fuss", track: 4, title: "Somebody Told Me"},
					{artist: "The Killers", album: "Hot Fuss", track: 5, title: "All These Things I've Done"},
					{artist: "The Killers", album: "Hot Fuss", track: 6, title: "Andy, You're A Star"},
					{artist: "The Killers", album: "Hot Fuss", track: 7, title: "On Top"},
					{artist: "The Killers", album: "Hot Fuss", track: 8, title: "Change Your Mind"},
					{artist: "The Killers", album: "Hot Fuss", track: 9, title: "Believe Me Natalie"},
					{artist: "The Killers", album: "Hot Fuss", track: 10, title: "Midnight Show"},
					{artist: "The Killers", album: "Hot Fuss", track: 11, title: "Everything Will Be Alright"},
				},
			},
			want: `ID    Artist         Album       Title
 1    Beck           Guero       Missing
 2    The Killers    Hot Fuss    Jenny Was A Friend Of Mine
 3    The Killers    Hot Fuss    Mr. Brightside
 4    The Killers    Hot Fuss    Smile Like You Mean It
 5    The Killers    Hot Fuss    Somebody Told Me
 6    The Killers    Hot Fuss    All These Things I've Done
 7    The Killers    Hot Fuss    Andy, You're A Star
 8    The Killers    Hot Fuss    On Top
 9    The Killers    Hot Fuss    Change Your Mind
10    The Killers    Hot Fuss    Believe Me Natalie
11    The Killers    Hot Fuss    Midnight Show
12    The Killers    Hot Fuss    Everything Will Be Alright
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.p.w = &w
			test.p.printSongFilter("")
			if want, got := test.want, w.String(); want != got {
				t.Errorf("printed song filters not equal: \n wanted: %q \n got:    %q", want, got)
			}
		})
	}
}

func TestPlaylistAddTrack(t *testing.T) {
	tests := []struct {
		name        string
		selectionID string
		p           playlist
		want        playlist
		wantErr     bool
	}{
		{
			name:    "no selection",
			wantErr: true,
		},
		{
			name:        "non-number selectionID",
			selectionID: "first",
			wantErr:     true,
		},
		{
			name:        "low selectionID",
			selectionID: "0",
			p: playlist{
				selection: []song{{}},
			},
			want: playlist{
				selection: []song{{}},
			},
			wantErr: true,
		},
		{
			name:        "high selectionID",
			selectionID: "2",
			p: playlist{
				selection: []song{{}},
			},
			want: playlist{
				selection: []song{{}},
			},
			wantErr: true,
		},
		{
			name:        "ok",
			selectionID: "2",
			p: playlist{
				selection: []song{
					{},
					{artist: "x", album: "y", title: "z", track: 8},
				},
				tracks: []m3uTrack{
					{},
				},
			},
			want: playlist{
				selection: []song{
					{},
					{artist: "x", album: "y", title: "z", track: 8},
				},
				tracks: []m3uTrack{
					{},
					{
						song:    song{artist: "x", album: "y", title: "z", track: 8},
						display: "x - z",
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.p.w = &w
			test.p.addTrack(test.selectionID)
			checkPlaylistsEqual(t, test.want, test.p)
			if want, got := test.wantErr, w.Len() != 0; want != got {
				t.Errorf("wanted logged error: %v, got %q", want, w.String())
			}
		})
	}
}

func TestPlaylistRemoveTrack(t *testing.T) {
	tests := []struct {
		name    string
		trackID string
		p       playlist
		want    playlist
		wantErr bool
	}{
		{
			name:    "no tracks",
			wantErr: true,
		},
		{
			name:    "non-number trackID",
			trackID: "first",
			wantErr: true,
		},
		{
			name:    "low trackID",
			trackID: "0",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "high trackID",
			trackID: "2",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "first",
			trackID: "1",
			p:       playlist{tracks: []m3uTrack{{display: "x"}, {display: "y"}, {display: "z"}}},
			want:    playlist{tracks: []m3uTrack{{display: "y"}, {display: "z"}}},
		},
		{
			name:    "middle",
			trackID: "2",
			p:       playlist{tracks: []m3uTrack{{display: "x"}, {display: "y"}, {display: "z"}}},
			want:    playlist{tracks: []m3uTrack{{display: "x"}, {display: "z"}}},
		},
		{
			name:    "last",
			trackID: "3",
			p:       playlist{tracks: []m3uTrack{{display: "x"}, {display: "y"}, {display: "z"}}},
			want:    playlist{tracks: []m3uTrack{{display: "x"}, {display: "y"}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.p.w = &w
			test.p.removeTrack(test.trackID)
			checkPlaylistsEqual(t, test.want, test.p)
			if want, got := test.wantErr, w.Len() != 0; want != got {
				t.Errorf("wanted logged error: %v, got %q", want, w.String())
			}
		})
	}
}

func TestPlaylistMoveTrack(t *testing.T) {
	tests := []struct {
		name    string
		command string
		p       playlist
		want    playlist
		wantErr bool
	}{
		{
			name:    "no commands",
			wantErr: true,
		},
		{
			name:    "one argument",
			command: "1",
			wantErr: true,
		},
		{
			name:    "too many arguments",
			command: "1 1 1",
			wantErr: true,
		},
		{
			name:    "non-number trackID",
			command: "first 1",
			wantErr: true,
		},
		{
			name:    "low trackID",
			command: "0 1",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "high trackID",
			command: "2 1 ",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "non-number move index",
			command: "1 first",
			wantErr: true,
		},
		{
			name:    "low move index",
			command: "1 0",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "high move index",
			command: "1 2",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "3-1",
			command: "3 1",
			p:       playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "3"}, {display: "4"}, {display: "5"}}},
			want:    playlist{tracks: []m3uTrack{{display: "3"}, {display: "1"}, {display: "2"}, {display: "4"}, {display: "5"}}},
		},
		{
			name:    "3-2",
			command: "3 2",
			p:       playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "3"}, {display: "4"}, {display: "5"}}},
			want:    playlist{tracks: []m3uTrack{{display: "1"}, {display: "3"}, {display: "2"}, {display: "4"}, {display: "5"}}},
		},
		{
			name:    "3-3",
			command: "3 3",
			p:       playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "3"}, {display: "4"}, {display: "5"}}},
			want:    playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "3"}, {display: "4"}, {display: "5"}}},
		},
		{
			name:    "3-4",
			command: "3 4",
			p:       playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "3"}, {display: "4"}, {display: "5"}}},
			want:    playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "4"}, {display: "3"}, {display: "5"}}},
		},
		{
			name:    "3-5",
			command: "3 5",
			p:       playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "3"}, {display: "4"}, {display: "5"}}},
			want:    playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "4"}, {display: "5"}, {display: "3"}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.p.w = &w
			test.p.moveTrack(test.command)
			checkPlaylistsEqual(t, test.want, test.p)
			if want, got := test.wantErr, w.Len() != 0; want != got {
				t.Errorf("wanted logged error: %v, got %q", want, w.String())
			}
		})
	}
}

func TestPlaylistRenameTrack(t *testing.T) {
	tests := []struct {
		name    string
		command string
		p       playlist
		want    playlist
		wantErr bool
	}{
		{
			name:    "no commands",
			wantErr: true,
		},
		{
			name:    "one argument",
			command: "1",
			wantErr: true,
		},
		{
			name:    "non-number move index",
			command: "one Song!",
			wantErr: true,
		},
		{
			name:    "low move index",
			command: "0 Song!",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "high move index",
			command: "2 Song!",
			p:       playlist{tracks: []m3uTrack{{}}},
			want:    playlist{tracks: []m3uTrack{{}}},
			wantErr: true,
		},
		{
			name:    "ok",
			command: "2 A song with    many spaces!",
			p:       playlist{tracks: []m3uTrack{{display: "1"}, {display: "2"}, {display: "3"}}},
			want:    playlist{tracks: []m3uTrack{{display: "1"}, {display: "A song with    many spaces!"}, {display: "3"}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.p.w = &w
			test.p.renameTrack(test.command)
			checkPlaylistsEqual(t, test.want, test.p)
			if want, got := test.wantErr, w.Len() != 0; want != got {
				t.Errorf("wanted logged error: %v, got %q", want, w.String())
			}
		})
	}
}

func TestPlaylistPrintTracks(t *testing.T) {
	tests := []struct {
		name string
		p    playlist
		want string
	}{
		{
			name: "short list",
			p: playlist{
				tracks: []m3uTrack{
					{song: song{artist: "x", album: "y", track: 8, title: "z", path: "b"}, display: "a"},
				},
			},
			want: `ID    Display    Artist    Album    Title
 1    a          x         y        z
`,
		},
		{
			name: "long track",
			p: playlist{
				tracks: []m3uTrack{
					{song: song{artist: "David Bowie", album: "The Rise and Fall of Ziggy Stardust and the Spiders from Mars", track: 1, title: "Five Years"}, display: "long-title"},
				},
			},
			want: `ID    Display       Artist         Album                                                            Title
 1    long-title    David Bowie    The Rise and Fall of Ziggy Stardust and the Spiders from Mars    Five Years
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.p.w = &w
			test.p.printTracks("")
			if want, got := test.want, w.String(); want != got {
				t.Errorf("printed playlist/tracks not equal: \n wanted: %q \n got:    %q", want, got)
			}
		})
	}
}

func TestPlaylistClearTracks(t *testing.T) {
	p := playlist{
		tracks: []m3uTrack{{}, {}, {}},
	}
	p.clearTracks("")
	if p.tracks != nil {
		t.Error()
	}
}

func TestPlaylistLoad(t *testing.T) {
	tests := []struct {
		name    string
		m3uPath string
		p       playlist
		want    playlist
		wantErr bool
	}{
		{
			name:    "missing file",
			m3uPath: "NOT_FOUND",
			p: playlist{
				tracks: []m3uTrack{{}},
				fsys: osFS{
					FS: fstest.MapFS{},
				},
			},
			want: playlist{
				tracks: []m3uTrack{{}},
			},
			wantErr: true,
		},
		{
			name:    "bad track (first file missing)",
			m3uPath: "a/b/c.m3u",
			p: playlist{
				songs: []song{
					{path: "d/g.mp3", track: 1},
					{path: "d/h.mp3", track: 2},
				},
				tracks: []m3uTrack{{}},
				fsys: osFS{
					FS: fstest.MapFS{
						"a/b/c.m3u": &fstest.MapFile{
							Data: []byte(`#EXT3MU
#EXTINF:0, Track 3 title
UNKNOWN.mp3
#EXTINF:0, Track 1 title
d/g.mp3
#EXTINF:0, Track 2 title
d/h.mp3
`),
						},
					},
				},
			},
			want: playlist{
				songs: []song{
					{path: "d/g.mp3", track: 1},
					{path: "d/h.mp3", track: 2},
				},
				tracks: []m3uTrack{
					{song: song{path: "d/g.mp3", track: 1}, display: "Track 1 title"},
					{song: song{path: "d/h.mp3", track: 2}, display: "Track 2 title"},
				},
			},
			wantErr: true,
		},
		{
			name:    "all files missing",
			m3uPath: "a/b/c.m3u",
			p: playlist{
				fsys: osFS{
					FS: fstest.MapFS{
						"a/b/c.m3u": &fstest.MapFile{
							Data: []byte("x\nx\nx\nx\nx\nx\nx\nx\nx\nx\nx\nx\n"),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "ok",
			m3uPath: "a/b/c.m3u",
			p: playlist{
				songs: []song{
					{path: "d/g.mp3", track: 1},
					{path: "d/h.mp3", artist: "x", title: "y", track: 2},
					{path: "j/e.mp3", track: 3},
					{path: "j/h.mp3", artist: "art", title: "word", track: 4},
				},
				tracks: []m3uTrack{{}},
				fsys: osFS{
					FS: fstest.MapFS{
						"a/b/c.m3u": &fstest.MapFile{
							Data: []byte(`#EXT3MU
#EXTINF:0, Track 1 title
d/g.mp3
#EXTINF:0, Track 2 title
d/h.mp3

j/h.mp3
`),
						},
					},
				},
			},
			want: playlist{
				songs: []song{
					{path: "d/g.mp3", track: 1},
					{path: "d/h.mp3", artist: "x", title: "y", track: 2},
					{path: "j/e.mp3", track: 3},
					{path: "j/h.mp3", artist: "art", title: "word", track: 4},
				},
				tracks: []m3uTrack{
					{song: song{path: "d/g.mp3", track: 1}, display: "Track 1 title"},
					{song: song{path: "d/h.mp3", artist: "x", title: "y", track: 2}, display: "Track 2 title"},
					{song: song{path: "j/h.mp3", artist: "art", title: "word", track: 4}, display: "art - word"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w bytes.Buffer
			test.p.w = &w
			test.p.load(test.m3uPath)
			checkPlaylistsEqual(t, test.want, test.p)
			if want, got := test.wantErr, w.Len() != 0; want != got {
				t.Errorf("wanted logged error: %v, got %q", want, w.String())
			}
		})
	}
}

func TestPlaylistWrite(t *testing.T) {
	var f io.WriteCloser
	tests := []struct {
		name    string
		m3uPath string
		p       playlist
		want    string
		wantErr bool
	}{
		{
			name:    "empty path",
			wantErr: true,
		},
		{
			name:    "bad extension",
			m3uPath: "not-music.mp3",
			wantErr: true,
		},
		{
			name:    "only extension",
			m3uPath: ".m3u",
			wantErr: true,
		},
		{
			name:    "file exists",
			m3uPath: "exists.m3u",
			p: playlist{
				fsys: osFS{
					FS: fstest.MapFS{
						"exists.m3u": &fstest.MapFile{},
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "write file error",
			m3uPath: "new.m3u",
			p: playlist{
				fsys: osFS{
					FS: fstest.MapFS{},
					createFileFunc: func(name string) (io.WriteCloser, error) {
						return nil, fmt.Errorf("create write error")
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "close file error",
			m3uPath: "new.m3u",
			p: playlist{
				fsys: osFS{
					FS: fstest.MapFS{},
					createFileFunc: func(name string) (io.WriteCloser, error) {
						f = &MockWriteCloser{
							closeFunc: func() error {
								return fmt.Errorf("close error")
							},
						}
						return f, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name:    "ok",
			m3uPath: "new.m3u",
			p: playlist{
				tracks: []m3uTrack{
					{display: "track 1", song: song{path: "a/b.mp3"}},
					{display: "track 2", song: song{path: "r/b.mp3"}},
					{display: "Track 1, again :)", song: song{path: "a/b.mp3"}},
				},
				fsys: osFS{
					FS: fstest.MapFS{},
					createFileFunc: func(name string) (io.WriteCloser, error) {
						if want, got := "new.m3u", name; want != got {
							return nil, fmt.Errorf("names not equal: wanted %q, got %q", want, got)
						}
						f = &MockWriteCloser{
							closeFunc: func() error {
								return nil // TODO: test close error
							},
						}
						return f, nil
					},
				},
			},
			want: "#EXTM3U\r\n" +
				"#EXTINF:0, track 1\r\n" +
				"a/b.mp3\r\n" +
				"#EXTINF:0, track 2\r\n" +
				"r/b.mp3\r\n" +
				"#EXTINF:0, Track 1, again :)\r\n" +
				"a/b.mp3\r\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f = new(MockWriteCloser)
			var w bytes.Buffer
			test.p.w = &w
			test.p.write(test.m3uPath)
			gotErr := w.Len() != 0
			switch {
			case test.wantErr:
				if !gotErr {
					t.Error("wanted error")
				}
			case gotErr:
				t.Errorf("unwanted error: %v", w.String())
			case test.want != f.(*MockWriteCloser).String():
				t.Errorf("file populated as desired: \n wanted: %q \n got:    %q", test.want, f.(*MockWriteCloser).String())
			}
		})
	}
}

func checkPlaylistsEqual(t *testing.T, want, got playlist) {
	t.Helper()
	if !playlistsEqual(want, got) {
		t.Errorf("playlists not equal: \n wanted: %v \n got:    %v", want, got)
	}
}

func playlistsEqual(a, b playlist) bool {
	switch {
	case len(a.songs) != len(b.songs),
		len(a.selection) != len(b.selection),
		len(a.tracks) != len(b.tracks):
		return false
	}
	for i := range a.songs {
		if a.songs[i] != b.songs[i] {
			return false
		}
	}
	for i := range a.selection {
		if a.selection[i] != b.selection[i] {
			return false
		}
	}
	for i := range a.tracks {
		if a.tracks[i] != b.tracks[i] {
			return false
		}
	}
	// do not compare w, fsys
	return true
}

func TestSongLess(t *testing.T) {
	songs := []song{
		0: {},
		1: {},
		2: {"pathB", "artist1", "album1", "title1", 1},
		3: {"pathA", "artist0", "album2", "title2", 1},
		4: {"pathA", "artist0", "album3", "title2", 3},
		5: {"pathA", "artist0", "album3", "title1", 5},
		6: {"pathA", "artist0", "album3", "title0", 5},
	}
	tests := []struct {
		name string
		i, j int
		want bool
	}{
		{"same", 0, 0, false},
		{"equal", 0, 1, false},
		{"artist first", 2, 3, false},
		{"artist first swapped", 3, 2, true},
		{"album second", 3, 4, true},
		{"track third", 4, 5, true},
		{"track third swapped", 5, 4, false},
		{"title last", 5, 6, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want != songLess(songs)(test.i, test.j) {
				t.Error()
			}
		})
	}
}

func TestDigitCount(t *testing.T) {
	tests := []struct {
		i, want int
	}{
		{0, 0},
		{1, 1},
		{-1, 0},
		{9, 1},
		{10, 2},
		{11, 2},
		{100, 3},
		{1549, 4},
	}
	for _, test := range tests {
		t.Run(fmt.Sprint(test.i), func(t *testing.T) {
			if want, got := test.want, digitCount(test.i); want != got {
				t.Errorf("wanted %v, got %v", want, got)
			}
		})
	}
}
