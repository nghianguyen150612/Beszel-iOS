//go:build !linux && !darwin && !ios

package agent

import (
	"os"
	"os/exec"
)

func configureProcursusProcess(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return os.ErrProcessDone
		}
		return cmd.Process.Kill()
	}
}
