//go:build !linux && !darwin && !ios && !freebsd && !openbsd && !netbsd && !dragonfly

package ioshelper

import (
	"errors"
	"os/exec"
)

func configureProcess(*exec.Cmd) error { return errors.ErrUnsupported }
func cleanupProcess(*exec.Cmd) error   { return nil }
