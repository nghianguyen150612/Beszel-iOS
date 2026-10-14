//go:build ios

package agent

import (
	"golang.org/x/sys/unix"
	"os"
	"runtime"
)

func (a *Agent) adjustPlatformSystemDetails() {
	applyIOSSystemDetails(&a.systemDetails, iosMetadataSource{
		sysctl: unix.Sysctl, count: unix.SysctlUint32, readFile: os.ReadFile, arch: runtime.GOARCH,
	})
}
