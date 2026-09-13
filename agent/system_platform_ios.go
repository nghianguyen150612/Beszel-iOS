//go:build ios

package agent

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/henrygd/beszel/internal/entities/system"
	"golang.org/x/sys/unix"
	"howett.net/plist"
)

type iosSystemVersion struct {
	ProductVersion string `plist:"ProductVersion"`
}

func (a *Agent) adjustPlatformSystemDetails() {
	d := &a.systemDetails

	d.Os = system.Darwin
	d.Arch = runtime.GOARCH

	if hostname, err := unix.Sysctl("kern.hostname"); err == nil && hostname != "" {
		d.Hostname = hostname
	}

	if kernel, err := unix.Sysctl("kern.osrelease"); err == nil && kernel != "" {
		d.Kernel = kernel
	}

	if cores, err := unix.SysctlUint32("hw.physicalcpu"); err == nil {
		d.Cores = int(cores)
	}

	if threads, err := unix.SysctlUint32("hw.logicalcpu"); err == nil {
		d.Threads = int(threads)
	}

	machine, _ := unix.Sysctl("hw.machine")
	if machine != "" {
		d.CpuModel = iosCPUModel(machine)
	}

	d.OsName = "iOS"
	if data, err := os.ReadFile("/System/Library/CoreServices/SystemVersion.plist"); err == nil {
		var version iosSystemVersion
		if _, err := plist.Unmarshal(data, &version); err == nil && version.ProductVersion != "" {
			d.OsName = "iOS " + version.ProductVersion
		}
	}
}

func iosCPUModel(machine string) string {
	// iPad4,1 through iPad4,9 are A7-family iPads.
	if strings.HasPrefix(machine, "iPad4,") {
		return "Apple A7"
	}

	switch machine {
	case "iPhone6,1", "iPhone6,2":
		return "Apple A7"
	default:
		return fmt.Sprintf("Apple SoC (%s)", machine)
	}
}
