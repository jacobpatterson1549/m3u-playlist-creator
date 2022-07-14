//go:build windows

package main

import (
	"log"
	"os"

	"golang.org/x/sys/windows"
)

// init adds a hack from stack overflow to make ANSI cursor escape sequences work
// https://github.com/sirupsen/logrus/issues/172#issuecomment-353724264
func init() {
	stdoutFileDescriptor := os.Stdout.Fd()
	stdout := windows.Handle(stdoutFileDescriptor)

	var consoleMode uint32
	windows.GetConsoleMode(stdout, &consoleMode)
	consoleMode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(stdout, consoleMode); err != nil {
		log.Printf("could not set console to enable virtual terminal: %v", err)
	}
}
