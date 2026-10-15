//go:build linux

package ioshelper

import (
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Test-only adoption avoids relying on a container PID 1 to reap a deliberately
// orphaned pipe-holder. This is not a production process-management requirement.
func fixtureReaper() error { return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0) }
func reapFixtureDescendant(pid int) error {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var status syscall.WaitStatus
		found, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
		if found == pid || err == syscall.ECHILD {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return syscall.ETIMEDOUT
}
