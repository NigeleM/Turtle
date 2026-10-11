// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package sqlite

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// WAL mode (PRAGMA journal_mode = WAL). SQLite programs share a WAL
// database through a memory-mapped index (the -shm file); Turtle doesn't
// map it, so it uses a WAL database only while no other program has it
// open. SQLite says "open" with a shared lock on the -shm file's DMS byte;
// Turtle holds that byte exclusively during each statement, so others
// wait for it (and it waits, up to the busy timeout, for them).
//
// Reading: the WAL's committed pages are read in place of the file's.
// Writing: the changed pages are appended to the WAL, the commit is made
// durable there, then copied into the database file, and the WAL is
// emptied. A crash at any point leaves committed changes in the WAL,
// where the next opener (Turtle or SQLite) finds them.

const (
	walMagicBE  = 0x377f0683 // checksums of big-endian words
	walMagicLE  = 0x377f0682
	walHdrSize  = 32
	walFrameHdr = 24
	shmDMS      = 128 // the -shm byte every open connection holds
)

type walState struct {
	shm     *os.File
	pages   map[uint32]int64 // page -> offset of its newest committed copy in the WAL
	dbSize  uint32           // the database size the last commit gave
	walSize int64            // the WAL's size when read
}

func (f *dbFile) walPath() string { return f.path + "-wal" }

// walEnter starts a statement on a WAL database: claim the -shm file's
// DMS byte (no other program has the database open) and read the WAL.
func (f *dbFile) walEnter() error {
	if f.walSt == nil {
		f.walSt = &walState{}
	}
	w := f.walSt
	if w.shm == nil {
		shm, err := os.OpenFile(f.path+"-shm", os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			shm, err = os.Open(f.path + "-shm")
			if err != nil {
				return errorf("%s: can't open %s-shm: %v", f.path, f.path, err)
			}
		}
		if !f.retry(func() bool { return osWriteLock(shm, shmDMS, 1) }) {
			shm.Close()
			return errorf("%s is in WAL mode and open in another program; Turtle can use it once that program closes it", f.path)
		}
		w.shm = shm
	}
	return f.walRead()
}

// walLeave gives the DMS byte back.
func (f *dbFile) walLeave() {
	if f.walSt == nil || f.walSt.shm == nil {
		return
	}
	osUnlock(f.walSt.shm, shmDMS, 1)
	f.walSt.shm.Close()
	f.walSt.shm = nil
}

// walChecksum is SQLite's WAL checksum over 8-byte steps, continuing from
// s1, s2.
func walChecksum(data []byte, s1, s2 uint32, bigEndian bool) (uint32, uint32) {
	word := binary.LittleEndian.Uint32
	if bigEndian {
		word = binary.BigEndian.Uint32
	}
	for i := 0; i+8 <= len(data); i += 8 {
		s1 += word(data[i:]) + s2
		s2 += word(data[i+4:]) + s1
	}
	return s1, s2
}

// walRead finds the committed pages in the WAL: frames whose salts match
// the header and whose running checksum holds, up to the last commit.
func (f *dbFile) walRead() error {
	w := f.walSt
	info, err := os.Stat(f.walPath())
	if err != nil || info.Size() < walHdrSize {
		w.pages, w.dbSize, w.walSize = nil, 0, 0
		return nil
	}
	if w.pages != nil && info.Size() == w.walSize {
		return nil
	}
	data, err := os.ReadFile(f.walPath())
	if err != nil {
		return err
	}
	f.clearCache()
	w.pages, w.dbSize, w.walSize = nil, 0, info.Size()
	magic := binary.BigEndian.Uint32(data[0:])
	if magic != walMagicBE && magic != walMagicLE {
		return nil
	}
	be := magic == walMagicBE
	pageSize := int(binary.BigEndian.Uint32(data[8:]))
	if pageSize == 1 {
		pageSize = 65536
	}
	if pageSize != f.pageSize && f.pageSize != 0 {
		return nil
	}
	salt1, salt2 := binary.BigEndian.Uint32(data[16:]), binary.BigEndian.Uint32(data[20:])
	s1, s2 := walChecksum(data[:24], 0, 0, be)
	if s1 != binary.BigEndian.Uint32(data[24:]) || s2 != binary.BigEndian.Uint32(data[28:]) {
		return nil
	}
	pending := map[uint32]int64{}
	committed := map[uint32]int64{}
	for off := int64(walHdrSize); off+walFrameHdr+int64(pageSize) <= int64(len(data)); off += walFrameHdr + int64(pageSize) {
		h := data[off : off+walFrameHdr]
		if binary.BigEndian.Uint32(h[8:]) != salt1 || binary.BigEndian.Uint32(h[12:]) != salt2 {
			break
		}
		s1, s2 = walChecksum(h[:8], s1, s2, be)
		s1, s2 = walChecksum(data[off+walFrameHdr:off+walFrameHdr+int64(pageSize)], s1, s2, be)
		if s1 != binary.BigEndian.Uint32(h[16:]) || s2 != binary.BigEndian.Uint32(h[20:]) {
			break
		}
		pending[binary.BigEndian.Uint32(h[0:])] = off + walFrameHdr
		if size := binary.BigEndian.Uint32(h[4:]); size != 0 {
			for pg, o := range pending {
				committed[pg] = o
			}
			pending = map[uint32]int64{}
			w.dbSize = size
		}
	}
	if len(committed) > 0 {
		w.pages = committed
	}
	return nil
}

// walPage reads page n from the WAL if it has a committed copy.
func (f *dbFile) walPage(n uint32, buf []byte) bool {
	if f.walSt == nil || f.walSt.pages == nil {
		return false
	}
	off, ok := f.walSt.pages[n]
	if !ok {
		return false
	}
	wf, err := os.Open(f.walPath())
	if err != nil {
		return false
	}
	defer wf.Close()
	_, err = wf.ReadAt(buf, off)
	return err == nil || err == io.EOF
}

// walCheckpoint copies the WAL's committed pages into the database file
// and empties the WAL.
func (f *dbFile) walCheckpoint() error {
	w := f.walSt
	if w == nil || w.pages == nil {
		return nil
	}
	pages := make([]uint32, 0, len(w.pages))
	for n := range w.pages {
		pages = append(pages, n)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i] < pages[j] })
	buf := make([]byte, f.pageSize)
	for _, n := range pages {
		if n > w.dbSize {
			continue
		}
		if !f.walPage(n, buf) {
			return errorf("%s: can't read its WAL", f.path)
		}
		if _, err := f.f.WriteAt(buf, int64(n-1)*int64(f.pageSize)); err != nil {
			return err
		}
	}
	if err := f.f.Truncate(int64(w.dbSize) * int64(f.pageSize)); err != nil {
		return err
	}
	if !f.syncsOff() {
		if err := syncFile(f.f); err != nil {
			return err
		}
	}
	if err := os.Truncate(f.walPath(), 0); err != nil {
		return err
	}
	w.pages, w.dbSize, w.walSize = nil, 0, 0
	return nil
}

// walCommit saves a transaction the WAL way: frames for every changed
// page, the last marking the commit, made durable; then the pages go into
// the database file and the WAL is emptied.
func (f *dbFile) walCommit(pages []uint32) error {
	wf, err := os.OpenFile(f.walPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return errorf("can't write %s: %v", f.walPath(), err)
	}
	var salt [8]byte
	rand.Read(salt[:])
	hdr := make([]byte, walHdrSize)
	binary.BigEndian.PutUint32(hdr[0:], walMagicBE)
	binary.BigEndian.PutUint32(hdr[4:], 3007000)
	ps := uint32(f.pageSize)
	if f.pageSize == 65536 {
		ps = 1
	}
	binary.BigEndian.PutUint32(hdr[8:], ps)
	copy(hdr[16:24], salt[:])
	s1, s2 := walChecksum(hdr[:24], 0, 0, true)
	binary.BigEndian.PutUint32(hdr[24:], s1)
	binary.BigEndian.PutUint32(hdr[28:], s2)
	out := append([]byte{}, hdr...)
	for i, n := range pages {
		fh := make([]byte, walFrameHdr)
		binary.BigEndian.PutUint32(fh[0:], n)
		if i == len(pages)-1 {
			binary.BigEndian.PutUint32(fh[4:], f.pageCount)
		}
		copy(fh[8:16], salt[:])
		s1, s2 = walChecksum(fh[:8], s1, s2, true)
		s1, s2 = walChecksum(f.dirty[n], s1, s2, true)
		binary.BigEndian.PutUint32(fh[16:], s1)
		binary.BigEndian.PutUint32(fh[20:], s2)
		out = append(out, fh...)
		out = append(out, f.dirty[n]...)
	}
	if _, err := wf.Write(out); err != nil {
		wf.Close()
		os.Truncate(f.walPath(), 0)
		return errorf("can't write %s: %v", f.walPath(), err)
	}
	if !f.syncsOff() {
		if err := syncFile(wf); err != nil {
			wf.Close()
			return err
		}
		syncDir(filepath.Dir(f.path))
	}
	wf.Close()
	if crashAt == "wal" {
		panic(simulatedCrash{})
	}
	// Committed. Now into the database file.
	for _, n := range pages {
		if _, err := f.f.WriteAt(f.dirty[n], int64(n-1)*int64(f.pageSize)); err != nil {
			return f.writeFailed(err)
		}
	}
	if err := f.f.Truncate(int64(f.pageCount) * int64(f.pageSize)); err != nil {
		return f.writeFailed(err)
	}
	if !f.syncsOff() {
		if err := syncFile(f.f); err != nil {
			return f.writeFailed(err)
		}
	}
	if err := os.Truncate(f.walPath(), 0); err != nil {
		return err
	}
	if f.walSt != nil {
		f.walSt.pages, f.walSt.dbSize, f.walSt.walSize = nil, 0, 0
	}
	f.keepWritten()
	return nil
}
