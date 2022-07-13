package main

import "strings"

type song struct {
	path                 string
	hash                 string
	artist, album, title string
	track                int
}

func (s song) matches(filter string, checkHash bool) bool {
	filter = strings.ToLower(filter)
	return strings.Contains(strings.ToLower(s.artist), filter) ||
		strings.Contains(strings.ToLower(s.album), filter) ||
		strings.Contains(strings.ToLower(s.title), filter) ||
		(checkHash && strings.Contains(s.hash, filter))
}

func (s song) display() string {
	switch {
	case len(s.title) != 0 && len(s.artist) != 0:
		return s.artist + " - " + s.title
	case len(s.title) != 0 && len(s.artist) == 0:
		return s.title
	case len(s.title) == 0 && len(s.artist) != 0 && len(s.album) != 0:
		return s.artist + " - " + s.album + " - ?"
	case len(s.title) == 0 && len(s.artist) != 0 && len(s.album) == 0:
		return s.artist + " - ?"
	case len(s.title) == 0 && len(s.artist) == 0 && len(s.album) == 0 && len(s.path) != 0:
		return s.path
	}
	return "?"
}
