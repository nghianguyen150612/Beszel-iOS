//go:build linux || darwin || ios || freebsd || openbsd || netbsd || dragonfly

package ioshelper

import (
	"os"
	"os/exec"
	"syscall"
)

func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	if err == syscall.ESRCH {
		return nil
	}
	return err
}
func configureProcess(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return killGroup(cmd)
	}
	return nil
}
func cleanupProcess(cmd *exec.Cmd) error { return killGroup(cmd) }
