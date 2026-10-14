// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

package evaluator

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// Show's output is buffered, and must still read as the program wrote it:
// in order with a child process's output, kept when the program fails,
// complete when large, and not lost when scheduled code writes from other
// goroutines.

func TestBufferedShowKeepsOrderWithSys(t *testing.T) {
	src := "show \"first\" .\nsys echo second\nshow \"third\" .\n"
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	want := "first\nsecond\nthird\n"
	if strings.ReplaceAll(got, "\r\n", "\n") != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBufferedShowKeepsOutputBeforeAnError(t *testing.T) {
	got, err := run(t, "show \"before\" .\nshow 1 / 0 .\nshow \"after\" .\n", "")
	if err == nil || !strings.Contains(err.Error(), "division by zero") {
		t.Fatalf("want a division error, got %v", err)
	}
	if got != "before\n" {
		t.Errorf("output before the error: got %q, want %q", got, "before\n")
	}
}

func TestBufferedShowLargeOutputIsComplete(t *testing.T) {
	const n = 50000
	src := fmt.Sprintf("[loop][i = 0; i < %d; i++]\n    show i .\n[loop][end]\n", n)
	got, err := run(t, src, "")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != n || lines[0] != "0" || lines[n-1] != fmt.Sprint(n-1) {
		t.Errorf("got %d lines (first %q, last %q), want %d numbered lines", len(lines), lines[0], lines[len(lines)-1], n)
	}
}

func TestStderrLineFlushesStdoutFirst(t *testing.T) {
	out := captureStdout(t, func() {
		fmt.Fprint(stdoutBuf, "pending")
		stderrLine("to stderr")
		if n := stdoutBuf.w.Buffered(); n != 0 {
			t.Errorf("%d bytes still buffered when stderr was written", n)
		}
	})
	if out != "pending" {
		t.Errorf("stdout: got %q, want %q", out, "pending")
	}
}

// Scheduled code runs show from goroutines: no output may be lost or torn.
func TestBufferedShowFromGoroutines(t *testing.T) {
	const workers, each = 8, 2000
	var wg sync.WaitGroup
	out := captureStdout(t, func() {
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for i := 0; i < each; i++ {
					fmt.Fprintln(stdoutBuf, "line")
					runtime.Gosched()
				}
			}(w)
		}
		wg.Wait()
		stdoutBuf.Flush()
	})
	if n := strings.Count(out, "line\n"); n != workers*each {
		t.Errorf("got %d lines, want %d", n, workers*each)
	}
}
