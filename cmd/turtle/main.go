// Command turtle runs a .t Turtle script.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"Turtle/evaluator"
	"Turtle/lexer"
	"Turtle/parser"
)

func main() {
	path, err := resolveScriptPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "turtle:", err)
		os.Exit(1)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "turtle:", err)
		os.Exit(1)
	}

	p := parser.New(lexer.New(string(data)))
	program := p.ParseProgram()
	if errs := p.Errors(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "turtle: parse error:", e)
		}
		os.Exit(1)
	}

	it := evaluator.New(filepath.Dir(path))
	if len(os.Args) > 2 {
		it.Args = os.Args[2:] // everything after the script path: system's args[]
	}
	if err := it.Run(program); err != nil {
		fmt.Fprintln(os.Stderr, "turtle:", err)
		os.Exit(1)
	}
}

func resolveScriptPath() (string, error) {
	if len(os.Args) >= 2 {
		return os.Args[1], nil
	}
	return findLatestScript()
}

// findLatestScript replicates the legacy "no file given -> run the most
// recently modified .t/.T file in the current directory" behavior.
func findLatestScript() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(cwd)
	if err != nil {
		return "", err
	}
	var best string
	var bestTime int64 = -1
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".t" && ext != ".T" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Unix() > bestTime {
			bestTime = info.ModTime().Unix()
			best = filepath.Join(cwd, e.Name())
		}
	}
	if best == "" {
		return "", fmt.Errorf("no .t file found in %s and none given on the command line", cwd)
	}
	return best, nil
}
