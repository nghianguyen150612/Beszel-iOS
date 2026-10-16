//go:build testing

// Package telemetry supplies deterministic, source-derived synthetic Agent
// snapshots. These are protocol fixtures, not captures or native OS execution.
package telemetry

import (
	"github.com/henrygd/beszel/internal/entities/container"
	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/entities/systemd"
)

type Fixture struct {
	Name, SystemID, UserID, Interface string
	Configs                           []monitor.Config
	Data                              system.CombinedData
	Packages                          system.PackageUpdates
}

// All returns independent snapshots on every call. Source provenance is listed
// in docs/ios-remote-agent-telemetry-matrix.md. Units follow the shared entities:
// GiB memory/filesystems, MiB GPU memory, bytes/s I/O, microsecond probe latency.
func All() []Fixture {
	fixtures := []Fixture{
		{Name: "linux", SystemID: "fixturelinux001", UserID: "ownerlinux00001", Interface: "wlan0"},
		{Name: "windows", SystemID: "fixturewindows1", UserID: "ownerwindows001", Interface: "Wi-Fi"},
		{Name: "macos", SystemID: "fixturemacos001", UserID: "ownermacos00001", Interface: "en0"},
		{Name: "ios", SystemID: "fixtureios00001", UserID: "ownerios0000001"},
	}
	for i := range fixtures {
		f := &fixtures[i]
		f.Data = system.CombinedData{
			Info: system.Info{AgentVersion: "0.21.0", Uptime: 86400 + uint64(i), Threads: 4, Cpu: 25, MemPct: 50, DiskPct: 40, BandwidthBytes: 3072},
			Stats: system.Stats{Cpu: 25, Mem: 8, MemUsed: 4, MemPct: 50, DiskTotal: 100, DiskUsed: 40, DiskPct: 40,
				CpuBreakdown: []float64{20, 5, 0, 0, 75}, CpuCoresUsage: system.Uint8Slice{0, 25, 50, 25},
				Bandwidth: [2]uint64{1024, 2048}, NetworkInterfaces: map[string][4]uint64{"eth0": {1024, 2048, 1048576, 2097152}},
				ExtraFs: map[string]*system.FsStats{"data": {DiskTotal: 200, DiskUsed: 50}}},
			Details:  &system.Details{Hostname: "synthetic-" + f.Name, Kernel: "fixture-kernel", Cores: 4, Threads: 4, CpuModel: "fixture CPU", Arch: "amd64", MemoryTotal: 8 * 1024 * 1024 * 1024},
			Monitors: map[string]monitor.Result{},
		}
		for j, protocol := range []string{"icmp", "tcp", "dns", "http", "https"} {
			target := "127.0.0.1"
			wireProtocol := protocol
			if protocol == "http" || protocol == "https" {
				target = protocol + "://127.0.0.1"
				wireProtocol = "http"
			}
			id := f.Name + string(rune('a'+j)) + "monitor"
			f.Configs = append(f.Configs, monitor.Config{ID: id, Target: target, Protocol: wireProtocol, Port: 80, Interval: 60})
			f.Data.Monitors[id] = monitor.Result{AvgResponse: 1500, MinResponse: 1000, MaxResponse: 2000,
				AvgResponse1h: 1500, MinResponse1h: 1000, MaxResponse1h: 2000, PacketLoss: 25, PacketLoss1h: 25,
				LastProbeAt: 1792108800000, SampleCount: 4, TotalCount: 4, SuccessCount: 3, ResponseSum: 4500}
		}
		if f.Name != "ios" {
			signal := -55.0 - float64(i)
			f.Data.Info.WiFi = map[string]system.WiFi{f.Interface: {SSID: "synthetic network", Signal: &signal}}
			f.Data.Stats.WiFi = map[string]int8{f.Interface: int8(signal)}
			f.Data.Stats.DiskIO = [2]uint64{4096, 8192}
			f.Data.Stats.DiskIOTotal = [2]uint64{1048576, 2097152}
			f.Data.Stats.ExtraFs["data"].DiskReadBytes = 1024
			f.Data.Stats.ExtraFs["data"].DiskWriteBytes = 2048
		}
	}
	linux, windows, macos, ios := &fixtures[0], &fixtures[1], &fixtures[2], &fixtures[3]
	linux.Data.Details.OsName = "Debian GNU/Linux (synthetic)"
	linux.Data.Info.LoadAvg = [3]float64{1, 0.5, 0.25}
	linux.Data.Stats.LoadAvg = linux.Data.Info.LoadAvg
	linux.Data.Info.Services = []uint16{1, 0}
	linux.Data.Info.SystemdLogs = true
	linux.Data.SystemdServicesUpdated = true
	linux.Data.SystemdServices = []*systemd.Service{{Name: "fixture.service", State: systemd.StatusActive, Sub: systemd.SubStateRunning, Cpu: 2, Mem: 1048576}}
	linux.Data.Containers = []*container.Stats{{Id: "fixturecontainer", Name: "fixture", Image: "fixture:v1", Status: "running", Cpu: 2, Mem: 16, Bandwidth: [2]uint64{100, 200}}}
	linux.Data.Stats.ZfsPools = map[string]*system.ZfsPool{"tank": {Total: 1024, Used: 256, ReadBytes: 1024, WriteBytes: 2048, Health: "ONLINE"}}
	// Btrfs is reported through the same filesystem/pool schema, not a separate key.
	linux.Data.Stats.ExtraFs["btrfs-data"] = &system.FsStats{DiskTotal: 500, DiskUsed: 100, DiskReadBytes: 512, DiskWriteBytes: 1024}
	linux.Data.Stats.Fans = map[string]uint16{"hwmon_fan1": 1200}
	linux.Data.Stats.GPUData = map[string]system.GPUData{"0": {Name: "NVIDIA fixture", Usage: 42, MemoryUsed: 512, MemoryTotal: 4096}, "1": {Name: "AMD fixture", Usage: 0, MemoryTotal: 2048}}
	linux.Data.Info.GpuPct = 42
	linux.Data.Stats.Temperatures = map[string]float64{"NVIDIA fixture": 55, "coretemp": 42}
	linux.Data.Info.DashboardTemp = 55
	linux.Data.Info.PackageUpdates = []uint16{2, 1}
	linux.Packages = system.PackageUpdates{Manager: "apt", CheckedAt: 1792108800, SecurityKnown: true, Packages: []system.PackageUpdate{{Name: "fixture-lib", Current: "1.0-1", Available: "1.0-2", Security: true}, {Name: "fixture-app", Current: "2.0", Available: "2.1"}}}
	windows.Data.Details.Os = system.Windows
	windows.Data.Details.OsName = "Windows 11 (synthetic)"
	windows.Data.Details.Kernel = "10.0.26100"
	windows.Data.Stats.NetworkInterfaces = map[string][4]uint64{"Ethernet": {1024, 2048, 1048576, 2097152}}
	// NVML is supported on Windows; this fixture does not require LHM assets.
	windows.Data.Stats.GPUData = map[string]system.GPUData{"0": {Name: "NVIDIA Windows fixture", Usage: 0, MemoryUsed: 128, MemoryTotal: 4096}}
	macos.Data.Details.Os = system.Darwin
	macos.Data.Details.OsName = "macOS (synthetic)"
	macos.Data.Details.Arch = "arm64"
	macos.Data.Details.CpuModel = "Apple Silicon fixture"
	macos.Data.Stats.NetworkInterfaces = map[string][4]uint64{"en0": {1024, 2048, 1048576, 2097152}}
	macos.Data.Info.LoadAvg = [3]float64{0.5, 0.25, 0.1}
	macos.Data.Stats.LoadAvg = macos.Data.Info.LoadAvg
	macos.Data.Stats.GPUData = map[string]system.GPUData{"0": {Name: "Apple GPU fixture", Usage: 12, Power: 2}}
	macos.Data.Info.GpuPct = 12
	macos.Data.Stats.Temperatures = map[string]float64{"Apple GPU fixture": 45}
	macos.Data.Info.DashboardTemp = 45
	for _, f := range []*Fixture{macos, ios} {
		f.Data.Info.Battery = system.Battery{75, 1}
		f.Data.Stats.Battery = f.Data.Info.Battery
		f.Data.Stats.Batteries = map[string]uint8{"battery": 75}
	}
	ios.Data.Details.Os = system.Darwin // existing iOS metadata contract, with explicit os_name
	ios.Data.Details.OsName = "iOS 12.5.7 (synthetic)"
	ios.Data.Details.Arch = "arm64"
	ios.Data.Details.CpuModel = "Apple A7"
	ios.Data.Stats.NetworkInterfaces = map[string][4]uint64{"en0": {1024, 2048, 1048576, 2097152}}
	ios.Data.Info.PackageUpdates = []uint16{0} // verified zero, security unknown
	ios.Packages = system.PackageUpdates{Manager: "apt", CheckedAt: 1792108800, Packages: []system.PackageUpdate{}}
	return fixtures
}
