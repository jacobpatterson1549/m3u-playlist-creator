package main

import "testing"

func TestSongMatches(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		song   song
		want   bool
	}{
		{"empty", "", song{}, true},
		{"path not checked", "c:/", song{path: "c:/song.mp3"}, false},
		{"track not checked", "1", song{artist: "The Who", album: "Tommy", title: "Overture / It's A Boy", track: 1}, false},
		{"title match", "19", song{artist: "The Who", album: "Tommy", title: "1921", track: 2}, true},
		{"album match", "tom", song{artist: "The Who", album: "Tommy", title: "Amazing Journey / Sparks", track: 3}, true},
		{"artist match", "who", song{artist: "The Who", album: "Tommy", title: "Eyesight To The Blind (The Hawker)", track: 3}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want != test.song.matches(test.filter, false) {
				t.Error()
			}
		})
	}
	hashTests := []struct {
		name      string
		checkHash bool
		want      bool
	}{
		{"no check", false, false},
		{"yes check", true, true},
	}
	for _, test := range hashTests {
		t.Run(test.name, func(t *testing.T) {
			filter := "hash"
			s := song{hash: filter}
			if test.want != s.matches(filter, test.checkHash) {
				t.Error()
			}
		})
	}
}

func TestSongDisplay(t *testing.T) {
	tests := []struct {
		name string
		song song
		want string
	}{
		{"empty 1", song{}, "?"},
		{"no artist", song{path: "song.mp3", artist: "", album: "world", title: "music", track: 1}, "music"},
		{"no title", song{path: "song.mp3", artist: "hello", album: "world", title: "", track: 1}, "hello - world - ?"},
		{"no title/album", song{path: "song.mp3", artist: "hello", album: "", title: "", track: 1}, "hello - ?"},
		{"path only", song{path: "path_only.mp3", artist: "", album: "", title: "", track: 1}, "path_only.mp3"},
		{"Simple", song{path: "song.mp3", artist: "Muse", album: "Absolution", title: "Hysteria", track: 8}, "Muse - Hysteria"},
		{"extra spaces", song{path: "song.mp3", artist: "a b  c   d", album: "", title: "  x  ", track: 8}, "a b  c   d -   x  "},
		{"special chars", song{path: "song.mp3", artist: "Macklemore & Ryan Lewis", album: "The Heist", title: "Wing$", track: 8}, "Macklemore & Ryan Lewis - Wing$"},
		{"normal", song{path: "song.mp3", artist: "Muse", album: "Absolution", title: "Hysteria", track: 8}, "Muse - Hysteria"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if want, got := test.want, test.song.display(); want != got {
				t.Errorf("wanted %q, got %q", want, got)
			}
		})
	}
}
