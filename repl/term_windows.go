package repl

import (
	"syscall"
	"unsafe"
)

// The terminal on Windows, through the console API in kernel32: key by
// key input with the escape sequences other terminals send
// (ENABLE_VIRTUAL_TERMINAL_INPUT), and color codes understood
// (ENABLE_VIRTUAL_TERMINAL_PROCESSING, Windows 10 and later).

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

const (
	enableProcessedInput            = 0x0001
	enableLineInput                 = 0x0002
	enableEchoInput                 = 0x0004
	enableVirtualTerminalInput      = 0x0200
	enableProcessedOutput           = 0x0001
	enableVirtualTerminalProcessing = 0x0004
)

type termState struct{ mode uint32 }

func handle(fd int) syscall.Handle {
	if fd == 0 {
		h, _ := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
		return h
	}
	h, _ := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	return h
}

func getMode(h syscall.Handle) (uint32, bool) {
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	return mode, r != 0
}

func setMode(h syscall.Handle, mode uint32) bool {
	r, _, _ := procSetConsoleMode.Call(uintptr(h), uintptr(mode))
	return r != 0
}

func isTerminal(fd int) bool {
	_, ok := getMode(handle(fd))
	return ok
}

func makeRaw(fd int) (*termState, error) {
	h := handle(fd)
	old, ok := getMode(h)
	if !ok {
		return nil, syscall.EINVAL
	}
	mode := old &^ (enableEchoInput | enableLineInput | enableProcessedInput)
	mode |= enableVirtualTerminalInput
	if !setMode(h, mode) {
		return nil, syscall.EINVAL
	}
	return &termState{old}, nil
}

func restore(fd int, s *termState) {
	if s != nil {
		setMode(handle(fd), s.mode)
	}
}

func termWidth(fd int) int {
	var info struct {
		size, cursor             struct{ x, y int16 }
		attributes               uint16
		left, top, right, bottom int16
		maxSize                  struct{ x, y int16 }
	}
	r, _, _ := procGetConsoleScreenBufferInfo.Call(uintptr(handle(1)), uintptr(unsafe.Pointer(&info)))
	if r == 0 || info.right <= info.left {
		return 80
	}
	return int(info.right-info.left) + 1
}

// enableColors turns on color codes for the console; false on Windows
// versions too old to understand them.
func enableColors(fd int) bool {
	h := handle(fd)
	mode, ok := getMode(h)
	if !ok {
		return false
	}
	return setMode(h, mode|enableProcessedOutput|enableVirtualTerminalProcessing)
}
