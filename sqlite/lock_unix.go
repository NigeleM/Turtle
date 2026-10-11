// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package sqlite

import (
	"os"
	"syscall"
)

// POSIX advisory locks on the same bytes SQLite locks (see lock.go), so
// Turtle and the sqlite3 tool (or any program using SQLite) can't change
// a file at the same time.

func lockRange(f *os.File, kind int16, start, length int64) bool {
	lk := syscall.Flock_t{Type: kind, Whence: 0, Start: start, Len: length}
	return syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &lk) == nil
}

func osReadLock(f *os.File, start, length int64) bool {
	return lockRange(f, syscall.F_RDLCK, start, length)
}

func osWriteLock(f *os.File, start, length int64) bool {
	return lockRange(f, syscall.F_WRLCK, start, length)
}

func osUnlock(f *os.File, start, length int64) {
	lockRange(f, syscall.F_UNLCK, start, length)
}

// osCheckReserved reports whether another process holds the RESERVED
// lock: it is writing, so its journal isn't left over from a crash.
func osCheckReserved(f *os.File) bool {
	lk := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: 0, Start: reservedByte, Len: 1}
	if err := syscall.FcntlFlock(f.Fd(), syscall.F_GETLK, &lk); err != nil {
		return false
	}
	return lk.Type != syscall.F_UNLCK
}

// osDowngradeShared turns the exclusive lock on the shared range back
// into a shared one.
func osDowngradeShared(f *os.File) bool {
	return osReadLock(f, sharedFirst, sharedSize)
}

// syncDir makes a new or removed journal file's name durable.
func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
}
