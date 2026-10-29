//go:build linux

package sidecar

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func sysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
}

func killGroup(cmd *exec.Cmd, force bool) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	_ = syscall.Kill(-cmd.Process.Pid, sig)
}

// reapZombies 回收被容器 init 收养的僵尸子进程（watch 崩溃后遗留的 mihomo）。
func reapZombies() {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}
	self := strconv.Itoa(os.Getpid())
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + e.Name() + "/stat")
		if err != nil {
			continue
		}
		end := bytes.LastIndexByte(data, ')')
		if end < 0 || end+2 >= len(data) {
			continue
		}
		fields := strings.Fields(string(data[end+2:]))
		if len(fields) < 2 || fields[0] != "Z" || fields[1] != self {
			continue
		}
		var ws syscall.WaitStatus
		_, _ = syscall.Wait4(pid, &ws, syscall.WNOHANG, nil)
	}
}
