// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

// Command turtle runs a .turtle Turtle script, or shows documentation:
//
//	turtle                         the REPL (or runs piped-in code)
//	turtle script.turtle [args...]
//	turtle trace script.turtle [args...]
//	turtle debug script.turtle [args...]
//	turtle doc [library | function | file.turtle]
//	turtle test [file.turtle | folder ...]
//	turtle fmt [--check] [file.turtle | folder ...]
//	turtle lsp                     the language server, for editors
//	turtle version | help
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"Turtle/evaluator"
	"Turtle/lexer"
	"Turtle/lsp"
	"Turtle/parser"
	"Turtle/repl"
)

// bundleFS is a built program's packed files (see bundle.go), or nil.
var bundleFS fs.FS

// version is the release's tag, set when the release is built
// (-ldflags "-X main.version=v0.9.150"); "dev" for a local build.
var version = "dev"

const usage = `Turtle %s

  turtle                       the interactive prompt (REPL)
  turtle script.turtle [args]     run a program
  turtle trace script.turtle      run it, showing each line as it runs (on stderr)
  turtle debug script.turtle      run it a line at a time: step, breakpoints, look at values
  turtle test [file | folder]  run the test_ functions in test_*.turtle files
  turtle fmt [file | folder]   lay out .turtle files the standard way (--check: only list them)
  turtle build script.turtle      make one program file that runs without Turtle (-o name)
  turtle doc [topic]           the standard library's documentation
  turtle lsp                   the language server, for editors (VS Code, Neovim ...)
  turtle version               the version
  turtle help                  this

With input piped in (echo 'show 1 + 2 .' | turtle), turtle runs it.
Docs: https://github.com/NigeleM/Turtle/tree/main/docs
`

// command reports whether the first argument is name and no file of that
// name is here (a script called "test" still runs; a folder called test
// doesn't count).
func command(name string) bool {
	if len(os.Args) < 2 || os.Args[1] != name {
		return false
	}
	info, err := os.Stat(name)
	return err != nil || info.IsDir()
}

func main() {
	evaluator.Version = version
	adaptGC()
	// A program made by turtle build: run what's packed in it.
	if code, ok := runBundled(); ok {
		os.Exit(code)
	}
	switch {
	case command("version"):
		fmt.Println("turtle " + version)
		return
	case command("help"):
		fmt.Printf(usage, version)
		return
	case command("doc"):
		cwd, _ := os.Getwd()
		text, err := evaluator.Doc(strings.Join(os.Args[2:], " "), cwd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "turtle doc:", err)
			os.Exit(1)
		}
		fmt.Print(text)
		return
	case command("lsp"):
		// The language server, for editors: JSON-RPC on stdin and stdout.
		os.Exit(lsp.Serve(os.Stdin, os.Stdout, version))
	case command("trace"), command("debug"):
		mode := os.Args[1]
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: turtle %s script.turtle [args]\n", mode)
			os.Exit(2)
		}
		path := os.Args[2]
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "turtle:", err)
			os.Exit(1)
		}
		os.Exit(runWatched(string(data), filepath.Dir(path), filepath.Base(path), os.Args[3:], mode))
	case command("fmt"):
		os.Exit(fmtCommand(os.Args[2:], os.Stdout, os.Stderr))
	case command("build"):
		os.Exit(buildCommand(os.Args[2:], os.Stdout, os.Stderr))
	case command("test"):
		cwd, _ := os.Getwd()
		os.Exit(evaluator.TestCommand(os.Args[2:], cwd, os.Stdout))
	}
	if len(os.Args) < 2 {
		if repl.IsTerminal() {
			os.Exit(repl.Run(version))
		}
		// Piped in: run it as a program, as python does.
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "turtle:", err)
			os.Exit(1)
		}
		cwd, _ := os.Getwd()
		os.Exit(runProgram(string(data), cwd, "", nil))
	}
	path := os.Args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "turtle:", err)
		os.Exit(1)
	}
	os.Exit(runProgram(string(data), filepath.Dir(path), filepath.Base(path), os.Args[2:]))
}

// runProgram runs Turtle source and returns the exit code.
func runProgram(src, dir, script string, args []string) int {
	return runWatched(src, dir, script, args, "")
}

// runWatched is runProgram, with each line shown on stderr as it runs
// (mode "trace") or run a line at a time (mode "debug").
func runWatched(src, dir, script string, args []string, mode string) int {
	p := parser.New(lexer.New(src))
	p.ModuleDir = dir
	p.Modules = bundleFS // a built program's own files, or nil
	program := p.ParseProgram()
	if errs := p.ErrorList(); len(errs) > 0 {
		// The first error only: the rest are usually the parser tripping
		// over what follows it.
		where := script
		if where == "" {
			where = "input"
		}
		fmt.Fprintf(os.Stderr, "turtle: %s, %s\n", where, parser.Format(src, errs[0]))
		return 1
	}
	for _, w := range p.Warnings() {
		where := script
		if where == "" {
			where = "input"
		}
		fmt.Fprintf(os.Stderr, "turtle: note: %s, line %d: %s\n", where, w.Line, w.Msg)
	}
	it := evaluator.New(dir)
	it.Script = script
	it.Args = args // everything after the script path: system's args[]
	it.Bundle = bundleFS
	switch mode {
	case "trace":
		it.Trace = os.Stderr
		it.TraceSource("", src)
	case "debug":
		it.Debug(os.Stderr, src)
	}
	if err := it.Run(program); err != nil {
		if ex, ok := err.(evaluator.ExitRequest); ok {
			return ex.Code
		}
		fmt.Fprintln(os.Stderr, "turtle:", err)
		return 1
	}
	return 0
}
