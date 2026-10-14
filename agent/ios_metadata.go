package agent

import (
	"fmt"
	"strings"

	"github.com/henrygd/beszel/internal/entities/system"
	"howett.net/plist"
)

type iosMetadataSource struct {
	sysctl   func(string) (string, error)
	count    func(string) (uint32, error)
	readFile func(string) ([]byte, error)
	arch     string
}

func applyIOSSystemDetails(d *system.Details, source iosMetadataSource) {
	d.Os, d.OsName, d.Arch = system.Darwin, "iOS", source.arch
	// Discard inherited macOS/Linux metadata; unavailable optional values stay absent.
	d.Hostname, d.Kernel, d.CpuModel = "", "", ""
	d.Cores, d.Threads = 0, 0
	if value, err := source.sysctl("kern.hostname"); err == nil {
		d.Hostname = value
	}
	if value, err := source.sysctl("kern.osrelease"); err == nil {
		d.Kernel = value
	}
	if value, err := source.sysctl("hw.machine"); err == nil {
		d.CpuModel = iosCPUModel(value)
	}
	// Reject absurd firmware/count responses; zero means unavailable in the existing payload.
	if value, err := source.count("hw.physicalcpu"); err == nil && value <= 1<<20 {
		d.Cores = int(value)
	}
	if value, err := source.count("hw.logicalcpu"); err == nil && value <= 1<<20 {
		d.Threads = int(value)
	}
	if d.Threads > 0 && d.Cores > d.Threads {
		d.Cores = d.Threads
	}
	if data, err := source.readFile("/System/Library/CoreServices/SystemVersion.plist"); err == nil {
		if version := iosProductVersion(data); version != "" {
			d.OsName += " " + version
		}
	}
}
func iosProductVersion(data []byte) string {
	var version struct {
		ProductVersion string `plist:"ProductVersion"`
	}
	if _, err := plist.Unmarshal(data, &version); err != nil {
		return ""
	}
	return strings.TrimSpace(version.ProductVersion)
}
func iosCPUModel(machine string) string {
	switch machine {
	case "iPad4,1", "iPad4,2", "iPad4,3", "iPad4,4", "iPad4,5", "iPad4,6", "iPad4,7", "iPad4,8", "iPad4,9", "iPhone6,1", "iPhone6,2":
		return "Apple A7"
	case "":
		return ""
	default:
		return fmt.Sprintf("Apple SoC (%s)", machine)
	}
}
