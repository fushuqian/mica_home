//go:build !linux

package sidecar

import (
	"os/exec"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return nil
}

func killGroup(cmd *exec.Cmd, force bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}

// reapZombies 在非 Linux 平台无需处理。
func reapZombies() {}
