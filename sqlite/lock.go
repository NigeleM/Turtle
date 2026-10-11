// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"time"
)

// Locking follows SQLite's own scheme, on the same bytes of the file, so
// Turtle and other programs using SQLite take turns:
//
//	SHARED     reading; many readers at once
//	RESERVED   about to write; readers can still start (one writer at a time)
//	PENDING    waiting for readers to finish; new readers must wait
//	EXCLUSIVE  writing the file; nobody else reads
//
// The lock bytes sit at 1 GB into the file (the "pending byte" page, which
// SQLite never stores data in), whether or not the file is that big.
const (
	lockNone = iota
	lockShared
	lockReserved
	lockPending
	lockExclusive
)

const (
	pendingByte  = 0x40000000
	reservedByte = pendingByte + 1
	sharedFirst  = pendingByte + 2
	sharedSize   = 510
)

// busyTimeout is how long to keep trying a lock another program holds.
var busyTimeout = 5 * time.Second

func (f *dbFile) retry(try func() bool) bool {
	deadline := time.Now().Add(f.timeout())
	for {
		if try() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func errBusy() error {
	return errorf("database is locked (another program is using it); try again when it's done")
}

// lock raises the lock to level, waiting up to busyTimeout.
func (f *dbFile) lock(level int) error {
	if f.lockLevel >= level {
		return nil
	}
	if f.readonly && level > lockShared {
		return errorf("%s is read-only, so it can't be changed", f.path)
	}
	switch {
	case f.lockLevel == lockNone:
		// SHARED: a read lock on the pending byte keeps out while a
		// writer is waiting, then a read lock on the shared range.
		ok := f.retry(func() bool {
			if !osReadLock(f.f, pendingByte, 1) {
				return false
			}
			got := osReadLock(f.f, sharedFirst, sharedSize)
			osUnlock(f.f, pendingByte, 1)
			return got
		})
		if !ok {
			return errBusy()
		}
		f.lockLevel = lockShared
		if level == lockShared {
			return nil
		}
		fallthrough
	case f.lockLevel == lockShared && level >= lockReserved:
		if !f.retry(func() bool { return osWriteLock(f.f, reservedByte, 1) }) {
			return errBusy()
		}
		f.lockLevel = lockReserved
		if level == lockReserved {
			return nil
		}
		fallthrough
	default:
		if f.lockLevel < lockPending {
			if !f.retry(func() bool { return osWriteLock(f.f, pendingByte, 1) }) {
				return errBusy()
			}
			f.lockLevel = lockPending
		}
		if level == lockExclusive {
			ok := f.retry(func() bool {
				if f.isWindows() {
					osUnlock(f.f, sharedFirst, sharedSize)
					if osWriteLock(f.f, sharedFirst, sharedSize) {
						return true
					}
					osReadLock(f.f, sharedFirst, sharedSize)
					return false
				}
				return osWriteLock(f.f, sharedFirst, sharedSize)
			})
			if !ok {
				return errBusy()
			}
			f.lockLevel = lockExclusive
		}
	}
	return nil
}

// unlock lowers the lock to SHARED or NONE.
func (f *dbFile) unlock(level int) {
	if f.lockLevel <= level {
		return
	}
	if f.lockLevel == lockExclusive && level == lockShared {
		osDowngradeShared(f.f)
	}
	if level == lockNone {
		osUnlock(f.f, sharedFirst, sharedSize)
		f.walLeave()
	}
	if f.lockLevel >= lockPending {
		osUnlock(f.f, pendingByte, 1)
	}
	if f.lockLevel >= lockReserved {
		osUnlock(f.f, reservedByte, 1)
	}
	f.lockLevel = level
}
