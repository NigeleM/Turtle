// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build darwin || linux

package repl

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

// The terminal on macOS and Linux, through the termios ioctls: raw mode
// (each key as it's typed, no echo, Ctrl-C as a key), and the width.

type termState struct{ t syscall.Termios }

func ioctl(fd, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

// isTerminal reports whether fd is a terminal.
func isTerminal(fd int) bool {
	var t syscall.Termios
	return ioctl(uintptr(fd), ioctlGetTermios, unsafe.Pointer(&t)) == nil
}

// makeRaw puts the terminal in raw mode and gives back how to restore it.
func makeRaw(fd int) (*termState, error) {
	var old syscall.Termios
	if err := ioctl(uintptr(fd), ioctlGetTermios, unsafe.Pointer(&old)); err != nil {
		return nil, err
	}
	t := old
	t.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	t.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	t.Cflag &^= syscall.CSIZE | syscall.PARENB
	t.Cflag |= syscall.CS8
	t.Cc[syscall.VMIN] = 1
	t.Cc[syscall.VTIME] = 0
	if err := ioctl(uintptr(fd), ioctlSetTermios, unsafe.Pointer(&t)); err != nil {
		return nil, err
	}
	return &termState{old}, nil
}

func restore(fd int, s *termState) {
	if s != nil {
		ioctl(uintptr(fd), ioctlSetTermios, unsafe.Pointer(&s.t))
	}
}

// termWidth is the terminal's width in columns (80 if it can't tell).
func termWidth(fd int) int {
	var ws struct{ Row, Col, X, Y uint16 }
	if ioctl(uintptr(fd), syscall.TIOCGWINSZ, unsafe.Pointer(&ws)) != nil || ws.Col == 0 {
		return 80
	}
	return int(ws.Col)
}

// enableColors: macOS and Linux terminals understand color codes as is.
func enableColors(fd int) bool { return true }

// backgroundIsLight asks the terminal for its background color (OSC 11),
// then what it is (DA1), which every terminal answers, so one that
// doesn't know OSC 11 doesn't keep us waiting. ok is false without an
// answer; typed is what was typed meanwhile, for the prompt.
func backgroundIsLight(fd int) (light, ok bool, typed []byte) {
	var old syscall.Termios
	if ioctl(uintptr(fd), ioctlGetTermios, unsafe.Pointer(&old)) != nil {
		return false, false, nil
	}
	t := old
	t.Lflag &^= syscall.ECHO | syscall.ICANON
	t.Cc[syscall.VMIN] = 0
	t.Cc[syscall.VTIME] = 2 // a read gives up after 0.2 seconds
	if ioctl(uintptr(fd), ioctlSetTermios, unsafe.Pointer(&t)) != nil {
		return false, false, nil
	}
	defer ioctl(uintptr(fd), ioctlSetTermios, unsafe.Pointer(&old))
	os.Stdout.WriteString("\x1b]11;?\x1b\\\x1b[c")
	var got []byte
	buf := make([]byte, 256)
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		n, err := syscall.Read(fd, buf)
		if n <= 0 || err != nil {
			break
		}
		got = append(got, buf[:n]...)
		if da1Reply.Match(got) {
			break
		}
	}
	light, ok = lightFromReply(string(got))
	return light, ok, withoutReplies(got)
}
