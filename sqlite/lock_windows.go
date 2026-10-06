//go:build windows

package sqlite

import (
	"os"
	"syscall"
	"unsafe"
)

// Windows locks on the same bytes SQLite locks (see lock.go), through
// LockFileEx and UnlockFileEx in kernel32.dll.

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	lockfileFailImmediately = 0x1
	lockfileExclusiveLock   = 0x2
)

func winLock(f *os.File, flags uint32, start, length int64) bool {
	var ol syscall.Overlapped
	ol.Offset = uint32(start)
	ol.OffsetHigh = uint32(start >> 32)
	r, _, _ := procLockFileEx.Call(f.Fd(), uintptr(flags|lockfileFailImmediately), 0,
		uintptr(uint32(length)), uintptr(uint32(length>>32)), uintptr(unsafe.Pointer(&ol)))
	return r != 0
}

func osReadLock(f *os.File, start, length int64) bool {
	return winLock(f, 0, start, length)
}

func osWriteLock(f *os.File, start, length int64) bool {
	return winLock(f, lockfileExclusiveLock, start, length)
}

func osUnlock(f *os.File, start, length int64) {
	var ol syscall.Overlapped
	ol.Offset = uint32(start)
	ol.OffsetHigh = uint32(start >> 32)
	procUnlockFileEx.Call(f.Fd(), 0, uintptr(uint32(length)), uintptr(uint32(length>>32)), uintptr(unsafe.Pointer(&ol)))
}

// osCheckReserved reports whether another process holds the RESERVED
// lock: try to take it, and give it back.
func osCheckReserved(f *os.File) bool {
	if osWriteLock(f, reservedByte, 1) {
		osUnlock(f, reservedByte, 1)
		return false
	}
	return true
}

// osDowngradeShared: Windows can't convert a lock in place, so drop the
// exclusive lock on the shared range and take a shared one.
func osDowngradeShared(f *os.File) bool {
	osUnlock(f, sharedFirst, sharedSize)
	return osReadLock(f, sharedFirst, sharedSize)
}

func syncDir(dir string) {}
