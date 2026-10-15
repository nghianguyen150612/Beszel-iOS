//go:build ios

package zfs

import "errors"

// Presence of a utility does not establish an iOS ZFS kernel capability.
func checkZfsDevice() error { return errors.ErrUnsupported }
