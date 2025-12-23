//go:build darwin
// +build darwin

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func flushStdin() error {
	fd := int(os.Stdin.Fd())
	arg := unix.FREAD
	return unix.IoctlSetInt(fd, unix.TIOCFLUSH, arg)
}
