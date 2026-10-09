//go:build !windows

package evaluator

import "os/exec"

// shellCommand runs command through the shell, for sys and runall.
func shellCommand(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
