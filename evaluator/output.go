// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"bufio"
	"io"
	"os"
	"sync"
)

// stdoutBuf is the program's standard output. Writes gather in a buffer
// and go to os.Stdout in large pieces (a program printing a line at a
// time made a system call per line). It is flushed when a run ends, and
// before anything else writes to the terminal, reads input, or starts a
// program, so the output keeps its order. Scheduled code writes from
// other goroutines, so it's locked.
var stdoutBuf = &outBuffer{w: bufio.NewWriterSize(liveStdout{}, 64<<10)}

// liveStdout is os.Stdout as it is when written, so a buffer made before a
// run redirects its stdout (tests do this) still writes to the right place.
type liveStdout struct{}

func (liveStdout) Write(p []byte) (int, error) { return os.Stdout.Write(p) }

type outBuffer struct {
	mu sync.Mutex
	w  *bufio.Writer
}

func (b *outBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.w.Write(p)
}

// Flush writes out what the buffer holds.
func (b *outBuffer) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.w.Flush()
}

// stderrLine writes s to stderr after the program's output so far.
func stderrLine(s string) {
	stdoutBuf.Flush()
	os.Stderr.WriteString(s + "\n")
}

// StderrWriter is stderr for text written as the program runs (turtle
// trace): each write comes after the program's output so far.
var StderrWriter io.Writer = stderrWriter{}

type stderrWriter struct{}

func (stderrWriter) Write(p []byte) (int, error) {
	stdoutBuf.Flush()
	return os.Stderr.Write(p)
}
