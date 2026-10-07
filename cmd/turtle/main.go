// Command turtle runs a .trt Turtle script, or shows documentation:
//
//	turtle                         the REPL (or runs piped-in code)
//	turtle script.trt [args...]
//	turtle trace script.trt [args...]
//	turtle doc [library | function | file.trt]
//	turtle test [file.trt | folder ...]
//	turtle lsp                     the language server, for editors
//	turtle version | help
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"Turtle/evaluator"
	"Turtle/lexer"
	"Turtle/lsp"
	"Turtle/parser"
	"Turtle/repl"
)

// version is the release's tag, set when the release is built
// (-ldflags "-X main.version=v0.9.150"); "dev" for a local build.
var version = "dev"

const usage = `Turtle %s

  turtle                       the interactive prompt (REPL)
  turtle script.trt [args]     run a program
  turtle trace script.trt      run it, showing each line as it runs (on stderr)
  turtle test [file | folder]  run the test_ functions in test_*.trt files
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
	case command("trace"):
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: turtle trace script.trt [args]")
			os.Exit(2)
		}
		path := os.Args[2]
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "turtle:", err)
			os.Exit(1)
		}
		os.Exit(runTraced(string(data), filepath.Dir(path), filepath.Base(path), os.Args[3:], os.Stderr))
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
	return runTraced(src, dir, script, args, nil)
}

// runTraced is runProgram, writing each line to trace as it runs (when
// trace isn't nil).
func runTraced(src, dir, script string, args []string, trace io.Writer) int {
	p := parser.New(lexer.New(src))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "turtle: parse error:", e)
		}
		return 1
	}
	it := evaluator.New(dir)
	it.Script = script
	it.Args = args // everything after the script path: system's args[]
	if trace != nil {
		it.Trace = trace
		it.TraceSource("", src)
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
