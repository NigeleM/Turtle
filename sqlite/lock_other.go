//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly || windows)

package sqlite

import "os"

// No file locks on this system: only one program may use a database at a
// time.

func osReadLock(f *os.File, start, length int64) bool  { return true }
func osWriteLock(f *os.File, start, length int64) bool { return true }
func osUnlock(f *os.File, start, length int64)         {}
func osCheckReserved(f *os.File) bool                  { return false }
func osDowngradeShared(f *os.File) bool                { return true }
func syncDir(dir string)                               {}
