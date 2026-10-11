// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build ignore

// perfcheck: does a change make Turtle slower or hungrier? It builds
// turtle from a base commit and from the working folder, runs every
// program in bench/ with each, taking turns so both see the same machine,
// and fails if the new one is slower or uses more memory by more than
// the allowance.
//
//	go run scripts/perfcheck.go                 base: origin/main
//	go run scripts/perfcheck.go -base v0.9.192  base: any commit
//
// Times are each program's own (its ms= line, best of its runs); memory
// is the peak the system reports for the process. macOS and Linux.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var (
	base     = flag.String("base", "origin/main", "the commit to compare with")
	rounds   = flag.Int("rounds", 3, "runs of each program with each turtle")
	slower   = flag.Float64("slower", 15, "allowed slowdown, percent")
	bigger   = flag.Float64("bigger", 15, "allowed memory growth, percent")
	minMS    = flag.Float64("min-ms", 5, "differences under this many ms are noise")
	minMB    = flag.Float64("min-mb", 8, "differences under this many MB are noise")
	msLine   = regexp.MustCompile(`ms=([0-9.]+)`)
	skipList = map[string]bool{"run.turtle": true, "hello.turtle": true, "linear.turtle": true} // no single ms= line
)

type result struct{ ms, mb float64 }

func main() {
	flag.Parse()
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	check(err, "not in a git repository")
	dir := strings.TrimSpace(string(root))
	work, err := os.MkdirTemp("", "perfcheck")
	check(err, "temp folder")
	defer os.RemoveAll(work)

	baseSrc := filepath.Join(work, "base")
	check(os.MkdirAll(baseSrc, 0o755), "temp folder")
	archive := exec.Command("sh", "-c", fmt.Sprintf("git archive %q | tar -x -C %q", *base, baseSrc))
	archive.Dir = dir
	if out, err := archive.CombinedOutput(); err != nil {
		fmt.Printf("perfcheck: can't read %s (%s); nothing to compare with\n", *base, strings.TrimSpace(string(out)))
		return
	}
	baseBin := filepath.Join(work, "turtle-base")
	newBin := filepath.Join(work, "turtle-new")
	build(baseSrc, baseBin)
	build(dir, newBin)

	programs, _ := filepath.Glob(filepath.Join(dir, "bench", "*.turtle"))
	sort.Strings(programs)
	fmt.Printf("%-12s %18s %18s %10s %10s\n", "program", "base ms / MB", "new ms / MB", "time", "memory")
	failed := false
	for _, prog := range programs {
		name := filepath.Base(prog)
		if skipList[name] {
			continue
		}
		b, n, ok := measure(baseBin, newBin, prog, name, *rounds, result{ms: -1}, result{ms: -1})
		if !ok { // a program the base can't run yet: nothing to compare
			n, _ = run(newBin, prog)
			fmt.Printf("%-12s %18s   %9.1f / %4.0f\n", strings.TrimSuffix(name, ".turtle"), "(new)", n.ms, n.mb)
			continue
		}
		// Shared machines have noisy moments: a program that looks slower
		// or bigger runs again, more times, before it counts.
		if worse(b, n) {
			b, n, _ = measure(baseBin, newBin, prog, name, 3**rounds, b, n)
		}
		dt := pct(b.ms, n.ms)
		dm := pct(b.mb, n.mb)
		mark := ""
		if slowerBy(b, n) {
			mark += "  SLOWER"
			failed = true
		}
		if biggerBy(b, n) {
			mark += "  BIGGER"
			failed = true
		}
		fmt.Printf("%-12s %9.1f / %4.0f   %9.1f / %4.0f   %+8.1f%%  %+8.1f%%%s\n",
			strings.TrimSuffix(name, ".turtle"), b.ms, b.mb, n.ms, n.mb, dt, dm, mark)
	}
	if failed {
		fmt.Printf("\nperfcheck: slower or bigger than %s by more than %.0f%% / %.0f%%\n", *base, *slower, *bigger)
		os.Exit(1)
	}
	fmt.Printf("\nperfcheck: no slower or bigger than %s\n", *base)
}

// measure runs prog rounds times with each turtle, taking turns, and
// gives the best of each, starting from b and n; false if the base can't
// run it.
func measure(baseBin, newBin, prog, name string, rounds int, b, n result) (result, result, bool) {
	for r := 0; r < rounds; r++ {
		res, ok := run(baseBin, prog)
		if !ok {
			return b, n, false
		}
		b = better(b, res)
		if res, ok = run(newBin, prog); !ok {
			fmt.Printf("perfcheck: %s fails with the new turtle\n", name)
			os.Exit(1)
		}
		n = better(n, res)
	}
	return b, n, true
}

func slowerBy(b, n result) bool { return n.ms-b.ms > *minMS && pct(b.ms, n.ms) > *slower }
func biggerBy(b, n result) bool { return n.mb-b.mb > *minMB && pct(b.mb, n.mb) > *bigger }
func worse(b, n result) bool    { return slowerBy(b, n) || biggerBy(b, n) }

func build(src, out string) {
	cmd := exec.Command("go", "build", "-o", out, "./cmd/turtle")
	cmd.Dir = src
	if b, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("perfcheck: building in %s failed:\n%s", src, b)
		os.Exit(1)
	}
}

// run runs one program: its own ms, and the process's peak memory; false
// if it failed or printed no ms= line.
func run(bin, prog string) (result, bool) {
	cmd := exec.Command(bin, filepath.Base(prog))
	cmd.Dir = filepath.Dir(prog)
	out, err := cmd.Output()
	m := msLine.FindSubmatch(out)
	if err != nil || m == nil {
		return result{}, false
	}
	ms, _ := strconv.ParseFloat(string(m[1]), 64)
	mb := 0.0
	if ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
		mb = float64(ru.Maxrss) / 1024 // kilobytes on Linux
		if runtime.GOOS == "darwin" {
			mb /= 1024 // bytes on macOS
		}
	}
	return result{ms, mb}, true
}

// better keeps the fastest time and the smallest memory seen; a is
// result{ms: -1} before the first run.
func better(a, b result) result {
	if a.ms < 0 {
		return b
	}
	return result{min(a.ms, b.ms), min(a.mb, b.mb)}
}

func pct(from, to float64) float64 {
	if from == 0 {
		return 0
	}
	return (to - from) / from * 100
}

func check(err error, what string) {
	if err != nil {
		fmt.Printf("perfcheck: %s: %v\n", what, err)
		os.Exit(1)
	}
}
