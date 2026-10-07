//go:build !darwin && !linux && !windows

package repl

import "errors"

// Other systems: no line editing; the REPL reads plain lines.

type termState struct{}

func isTerminal(fd int) bool             { return false }
func makeRaw(fd int) (*termState, error) { return nil, errors.New("no terminal support") }
func restore(fd int, s *termState)       {}
func termWidth(fd int) int               { return 80 }
func enableColors(fd int) bool           { return false }
