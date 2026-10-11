// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build !windows

package evaluator

import "os/exec"

// shellCommand runs command through the shell, for sys and runall.
func shellCommand(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
