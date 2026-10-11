// Copyright 2017-2026 Nigele McCoy
// Licensed under the Apache License, Version 2.0;
// see LICENSE and NOTICE.

//go:build windows

package evaluator

import (
	"os/exec"
	"syscall"
)

// shellCommand runs command through cmd, for sys and runall. cmd reads
// its own command line instead of the quoted arguments Go writes (echo
// "hi" would reach it as echo \"hi\"), so the line is given as is: /s
// with the whole command in quotes keeps every quote inside it.
func shellCommand(command string) *exec.Cmd {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /s /c "` + command + `"`}
	return cmd
}
