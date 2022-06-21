package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"testing"
)

//go:embed empty_audacity.mp3
var emptyMP3 []byte

const (
	emptyMp3Track  = "1549"
	emptyMp3Title  = "MY_TITLE00"
	emptyMp3Album  = "MY_ALBUM00"
	emptyMp3Artist = "MY_ARTIST0"
)

func mockMp3(s song) []byte {
	data := make([]byte, len(emptyMP3))
	copy(data, emptyMP3)
	replaceExactly := func(dst, src string) []byte {
		switch {
		case len(src) > len(dst):
			src = src[:len(dst)]
		case len(src) < len(dst):
			format := fmt.Sprintf("%%-%ds", len(dst))
			src = fmt.Sprintf(format, src)
		}
		return bytes.Replace(data, []byte(dst), []byte(src), 1)
	}
	data = replaceExactly(emptyMp3Track, fmt.Sprintf("%04d", s.track%1000))
	data = replaceExactly(emptyMp3Title, s.title)
	data = replaceExactly(emptyMp3Album, s.album)
	data = replaceExactly(emptyMp3Artist, s.artist)
	return data
}

func checkPlaylistsEqual(t *testing.T, want, got playlist) {
	t.Helper()
	if !playlistsEqual(t, want, got) {
		t.Errorf("playlists not equal: \n wanted: %v \n got:    %v", want, got)
	}
}

func playlistsEqual(t *testing.T, a, b playlist) bool {
	t.Helper()
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

type MockPlaylistFS struct {
	fs.FS
	CreateFileFunc func(name string) (io.WriteCloser, error)
}

func (fsys MockPlaylistFS) CreateFile(name string) (io.WriteCloser, error) {
	return fsys.CreateFileFunc(name)
}

// MockWriteCloser is a Writer that is also a Closer that delegates to a helper function.
type MockWriteCloser struct {
	io.Writer
	CloseFunc func() error
}

func (w MockWriteCloser) Close() error {
	return w.CloseFunc()
}

// MockFixedBuffer is a Writer that writes to the buffer without reallocating only if there is space.
type MockFixedBuffer struct {
	Buf []byte
	P   int
}

func (f *MockFixedBuffer) Write(data []byte) (n int, err error) {
	if f.P+len(data) > len(f.Buf) {
		err = errors.New("MockFixedBuffer overflow")
	} else {
		n = copy(f.Buf[f.P:], data)
		f.P += len(data)
	}
	return
}

func TestMockFixedBufferWrite(t *testing.T) {
	tests := []struct {
		name    string
		f       MockFixedBuffer
		data    []byte
		wantN   int
		wantErr bool
		wantBuf []byte
	}{
		{
			name: "empty write ok on empty buffer",
		},
		{
			name: "empty write ok on filled buffer",
			f: MockFixedBuffer{
				Buf: []byte{0, 0},
				P:   2,
			},
			wantBuf: []byte{0, 0},
		},
		{
			name: "empty write ok on half-filled buffer",
			f: MockFixedBuffer{
				Buf: []byte{0, 0},
				P:   1,
			},
			wantBuf: []byte{0, 0},
		},
		{
			name: "write on front",
			data: []byte("test"),
			f: MockFixedBuffer{
				Buf: []byte{0, 0, 0, 0, 0, 0},
			},
			wantN:   4,
			wantBuf: []byte(string("test\x00\x00")),
		},
		{
			name: "write on middle",
			data: []byte("test"),
			f: MockFixedBuffer{
				Buf: []byte{0, 0, 0, 0, 0, 0},
				P:   1,
			},
			wantN:   4,
			wantBuf: []byte(string("\x00test\x00")),
		},
		{
			name: "write on end",
			data: []byte("test"),
			f: MockFixedBuffer{
				Buf: []byte{0, 0, 0, 0, 0, 0},
				P:   2,
			},
			wantN:   4,
			wantBuf: []byte(string("\x00\x00test")),
		},
		{
			name: "write on end",
			data: []byte("test"),
			f: MockFixedBuffer{
				Buf: []byte{0, 0, 0, 0, 0, 0},
				P:   3,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotN, gotErr := test.f.Write(test.data)
			switch {
			case test.wantErr:
				if gotErr == nil {
					t.Error("wanted error")
				}
			case gotErr != nil:
				t.Errorf("unwanted error: %v", gotErr)
			case test.wantN != gotN:
				t.Errorf("write byte counts not equal: wanted %v, got %v", test.wantN, gotN)
			case string(test.wantBuf) != string(test.f.Buf):
				t.Errorf("buffers not equal after write: \n wanted: %q \n got:    %q", string(test.wantBuf), string(test.f.Buf))
			}
		})
	}
}
