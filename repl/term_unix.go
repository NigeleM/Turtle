//go:build darwin || linux

package repl

import (
	"syscall"
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
