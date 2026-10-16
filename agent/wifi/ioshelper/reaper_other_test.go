//go:build darwin || ios || freebsd

package ioshelper

import (
	"syscall"
	"time"
)

func fixtureReaper() error { return nil }
func reapFixtureDescendant(pid int) error {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) == syscall.ESRCH {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
	return syscall.ETIMEDOUT
}
