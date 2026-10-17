//go:build unix

package health

import (
	"os"
	"syscall"
)

func connectionMarkerTrusted(info os.FileInfo) bool {
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && uint64(stat.Uid) == uint64(os.Geteuid())
}
