package main

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/fstest"
)

func TestOsFSCreateFile(t *testing.T) {
	tests := []struct {
		name     string
		osFS     osFS
		destPath string
		wantErr  bool
	}{
		{
			name:     "might note be not child of executablePath",
			destPath: "/e/g/list.m3u",
			wantErr:  true,
		},
		{
			name:     "error creating file",
			destPath: "list.m3u",
			osFS: osFS{
				FS: fstest.MapFS{},
				createFileFunc: func(name string) (io.WriteCloser, error) {
					return nil, fmt.Errorf("error creating file")
				},
			},
			wantErr: true,
		},
		{
			name:     "file exists",
			destPath: "list.m3u",
			osFS: osFS{
				FS: fstest.MapFS{
					"list.m3u": &fstest.MapFile{},
				},
			},
			wantErr: true,
		},
		{
			name:     "ok",
			destPath: "list.m3u",
			osFS: osFS{
				FS: fstest.MapFS{},
				createFileFunc: func(name string) (io.WriteCloser, error) {
					if want, got := "list.m3u", name; want != got {
						return nil, fmt.Errorf("file names not equal: \n wanted: %v \n got:    %q", want, got)

					}
					w := new(MockWriteCloser)
					return w, nil
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, err := test.osFS.CreateFile(test.destPath)
			switch {
			case test.wantErr:
				if err == nil {
					t.Error("wanted error")
				}
			case err != nil:
				t.Errorf("unwanted error: %v", err)
			case w == nil:
				t.Error("file not created")
			}
		})
	}
}

func TestRunPlaylistCreator(t *testing.T) {
	t.Run("EOF", func(t *testing.T) {
		r := strings.NewReader("")
		w := io.Discard
		var fsys osFS
		var songs []song
		fsys.runPlaylistCreator(songs, r, w, false)
	})
	t.Run("many", func(t *testing.T) {
		songs := []song{
			// songs should be sorted so track 1 is first
			{path: "e.mp3", artist: "b", album: "c", title: "e", track: 2},
			{path: "d.mp3", artist: "b", album: "c", title: "d", track: 1},
		}
		commands := []string{
			"",           // empty command
			"h",          // print help
			"l prev.m3u", // load a playlist
			"c",          // clear the playlist
			"f b",        // filter to "b" (both tracks have artist:b, track 1 should be first)
			"d",          // print filter again
			"a 1",        // add song 'd'
			"a 2",        // add song 'e'
			"m 2 1",      // move 'e' to the top
			"r 2",        // remove 'd'
			"n 1 song",   // rename 'e'
			"p",          // print tracks
			"?",          // invalid command
			"w curr.m3u", // write the playlist
			"5",          // toggle hash mode
			"q",          // stop evaluating commands
		}
		joinedCommands := strings.Join(commands, "\n")
		input := strings.NewReader(joinedCommands)
		var output strings.Builder
		want := "#EXTM3U\r\n#EXTINF:0, song\r\ne.mp3\r\n"
		f := MockWriteCloser{
			Writer: &strings.Builder{},
			CloseFunc: func() error {
				return nil
			},
		}
		fsys := osFS{
			FS: fstest.MapFS{
				"e.mp3": {},
				"d.mp3": {},
				"prev.m3u": &fstest.MapFile{
					Data: []byte("d.mp3"),
				},
			},
			createFileFunc: func(name string) (io.WriteCloser, error) {
				if want, got := "curr.m3u", name; want != got {
					t.Errorf("new playlist names not equal: wanted %q, got %q", want, got)
				}

				return &f, nil
			},
		}
		fsys.runPlaylistCreator(songs, input, &output, false)
		switch {
		case output.Len() == 0:
			t.Errorf("no output written")
		default:
			got := f.Writer.(*strings.Builder).String()
			if want != got {
				t.Errorf("created playlist not equal: \n wanted: %q \n got:    %q", want, got)
			}
		}
	})
}

func TestRunCommands(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		tests := []struct {
			line      string
			wantValid bool
		}{
			{"h", true},
			{"q", true},
			{"a", true},
			{"z", true},
			{"j", false},
		}
		for _, test := range tests {
			t.Run(test.line, func(t *testing.T) {
				cmds := commands{
					{"a", func(command string) {}, ""},
					{"z", func(command string) {}, ""},
				}
				input := strings.NewReader(test.line)
				var output strings.Builder
				cmds.run(input, &output)
				got := output.String()
				gotValid := !strings.Contains(got, "invalid command")
				if test.wantValid != gotValid {
					t.Errorf("wanted valid output: %v, got %q", test.wantValid, got)
				}
			})
		}
	})
}
