package agent

import (
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/henrygd/beszel/agent/deltatracker"
	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/internal/entities/system"
	psutilNet "github.com/shirou/gopsutil/v4/net"
)

var networkIOCounters = psutilNet.IOCounters
var networkInterfaces = psutilNet.Interfaces

// NicConfig controls inclusion/exclusion of network interfaces via the NICS env var
//
// Behavior mirrors SensorConfig's matching logic:
// - Leading '-' means blacklist mode; otherwise whitelist mode
// - Supports '*' wildcards using path.Match
// - In whitelist mode with an empty list, no NICs are selected
// - In blacklist mode with an empty list, all NICs are selected
type NicConfig struct {
	nics         map[string]struct{}
	isBlacklist  bool
	hasWildcards bool
}

func newNicConfig(nicsEnvVal string) *NicConfig {
	cfg := &NicConfig{
		nics: make(map[string]struct{}),
	}
	if strings.HasPrefix(nicsEnvVal, "-") {
		cfg.isBlacklist = true
		nicsEnvVal = nicsEnvVal[1:]
	}
	for nic := range strings.SplitSeq(nicsEnvVal, ",") {
		nic = strings.TrimSpace(nic)
		if nic != "" {
			cfg.nics[nic] = struct{}{}
			if strings.Contains(nic, "*") {
				cfg.hasWildcards = true
			}
		}
	}
	return cfg
}

// isValidNic determines if a NIC should be included based on NicConfig rules
func isValidNic(nicName string, cfg *NicConfig) bool {
	// Empty list behavior differs by mode: blacklist: allow all; whitelist: allow none
	if len(cfg.nics) == 0 {
		return cfg.isBlacklist
	}

	// Exact match: return true if whitelist, false if blacklist
	if _, exactMatch := cfg.nics[nicName]; exactMatch {
		return !cfg.isBlacklist
	}

	// If no wildcards, return true if blacklist, false if whitelist
	if !cfg.hasWildcards {
		return cfg.isBlacklist
	}

	// Check for wildcard patterns
	for pattern := range cfg.nics {
		if !strings.Contains(pattern, "*") {
			continue
		}
		if match, _ := path.Match(pattern, nicName); match {
			return !cfg.isBlacklist
		}
	}

	return cfg.isBlacklist
}

func (a *Agent) updateNetworkStats(cacheTimeMs uint16, systemStats *system.Stats) {
	// network stats
	a.ensureNetInterfacesInitialized()

	a.ensureNetworkInterfacesMap(systemStats)

	if netIO, err := networkIOCounters(true); err == nil {
		a.reconcileNetworkInterfaces(netIO)
		nis, msElapsed := a.loadAndTickNetBaseline(cacheTimeMs)
		bytesSentPerSecond, bytesRecvPerSecond := a.sumAndTrackPerNicDeltas(cacheTimeMs, msElapsed, netIO, systemStats)
		a.applyNetworkTotals(cacheTimeMs, netIO, systemStats, nis, bytesSentPerSecond, bytesRecvPerSecond)
	} else {
		// Failure breaks continuity; recovery establishes fresh per-interval baselines.
		a.netIoStats = make(map[uint16]system.NetIoStats)
		a.netInterfaceDeltaTrackers = make(map[uint16]*deltatracker.DeltaTracker[string, uint64])
	}
}

func (a *Agent) initializeNetIoStats() {
	// reset valid network interfaces
	a.netInterfaces = make(map[string]bool, 0)

	// parse NICS env var for whitelist / blacklist
	nicsEnvVal, nicsEnvExists := utils.GetEnv("NICS")
	var nicCfg *NicConfig
	if nicsEnvExists {
		nicCfg = newNicConfig(nicsEnvVal)
	}

	a.nicConfig = nicCfg
	if netIO, err := networkIOCounters(true); err == nil {
		a.reconcileNetworkInterfaces(netIO)
	}

	// Reset per-cache-time trackers and baselines so they will reinitialize on next use
	a.netInterfaceDeltaTrackers = make(map[uint16]*deltatracker.DeltaTracker[string, uint64])
	a.netIoStats = make(map[uint16]system.NetIoStats)
}

// ensureNetInterfacesInitialized parses policy and initializes state once.
func (a *Agent) ensureNetInterfacesInitialized() {
	if a.netInterfaces == nil {
		a.initializeNetIoStats()
	}
}

// ensureNetworkInterfacesMap ensures systemStats.NetworkInterfaces map exists
func (a *Agent) ensureNetworkInterfacesMap(systemStats *system.Stats) {
	if systemStats.NetworkInterfaces == nil {
		systemStats.NetworkInterfaces = make(map[string][4]uint64, 0)
	}
}

// loadAndTickNetBaseline returns the NetIoStats baseline and milliseconds elapsed, updating time
func (a *Agent) loadAndTickNetBaseline(cacheTimeMs uint16) (netIoStat system.NetIoStats, msElapsed uint64) {
	netIoStat = a.netIoStats[cacheTimeMs]
	if netIoStat.Time.IsZero() {
		netIoStat.Time = time.Now()
		msElapsed = 0
	} else {
		elapsed := time.Since(netIoStat.Time).Milliseconds()
		if elapsed > 0 {
			msElapsed = uint64(elapsed)
		}
		netIoStat.Time = time.Now()
	}
	return netIoStat, msElapsed
}

// sumAndTrackPerNicDeltas records per-NIC up/down deltas into systemStats and returns their summed rates.
// Summing per-NIC deltas (rather than diffing summed counters) keeps the total from underflowing
// when a tracked interface disappears or its counters reset.
func (a *Agent) sumAndTrackPerNicDeltas(cacheTimeMs uint16, msElapsed uint64, netIO []psutilNet.IOCountersStat, systemStats *system.Stats) (bytesSentPerSecond, bytesRecvPerSecond uint64) {
	tracker := a.netInterfaceDeltaTrackers[cacheTimeMs]
	if tracker == nil {
		tracker = deltatracker.NewDeltaTracker[string, uint64]()
		a.netInterfaceDeltaTrackers[cacheTimeMs] = tracker
	}
	tracker.Cycle()

	for _, v := range netIO {
		useMacCounters, exists := a.netInterfaces[v.Name]
		if !exists {
			continue
		}
		if useMacCounters {
			correctNvethernetCounters(&v)
		}
		var upDelta, downDelta uint64
		upKey, downKey := fmt.Sprintf("%sup", v.Name), fmt.Sprintf("%sdown", v.Name)
		tracker.Set(upKey, v.BytesSent)
		tracker.Set(downKey, v.BytesRecv)
		if msElapsed > 0 {
			if prev, ok := tracker.Previous(upKey); ok {
				upDelta = rateFromCounters(v.BytesSent, prev, msElapsed)
			}
			if prev, ok := tracker.Previous(downKey); ok {
				downDelta = rateFromCounters(v.BytesRecv, prev, msElapsed)
			}
		}

		systemStats.NetworkInterfaces[v.Name] = [4]uint64{upDelta, downDelta, v.BytesSent, v.BytesRecv}
		bytesSentPerSecond = saturatingAdd(bytesSentPerSecond, upDelta)
		bytesRecvPerSecond = saturatingAdd(bytesRecvPerSecond, downDelta)
	}

	return bytesSentPerSecond, bytesRecvPerSecond
}

// applyNetworkTotals validates and writes computed network stats, or resets on anomaly
func (a *Agent) applyNetworkTotals(
	cacheTimeMs uint16,
	netIO []psutilNet.IOCountersStat,
	systemStats *system.Stats,
	nis system.NetIoStats,
	bytesSentPerSecond, bytesRecvPerSecond uint64,
) {
	if bytesSentPerSecond > 10_000_000_000 || bytesRecvPerSecond > 10_000_000_000 {
		slog.Warn("Invalid net stats. Resetting.", "sent", bytesSentPerSecond, "recv", bytesRecvPerSecond)
		for _, v := range netIO {
			if _, exists := a.netInterfaces[v.Name]; !exists {
				continue
			}
			slog.Info(v.Name, "recv", v.BytesRecv, "sent", v.BytesSent)
		}
		a.initializeNetIoStats()
		delete(a.netIoStats, cacheTimeMs)
		delete(a.netInterfaceDeltaTrackers, cacheTimeMs)
		systemStats.Bandwidth[0], systemStats.Bandwidth[1] = 0, 0
		for name, values := range systemStats.NetworkInterfaces {
			values[0], values[1] = 0, 0
			systemStats.NetworkInterfaces[name] = values
		}
		return
	}

	systemStats.Bandwidth[0], systemStats.Bandwidth[1] = bytesSentPerSecond, bytesRecvPerSecond
	a.netIoStats[cacheTimeMs] = nis
}

// skipNetworkInterface returns true if the network interface should be ignored.
func skipNetworkInterface(v psutilNet.IOCountersStat, nicCfg *NicConfig) bool {
	if nicCfg != nil {
		if !isValidNic(v.Name, nicCfg) {
			return true
		}
		// In whitelist mode, we honor explicit inclusion without auto-filtering.
		if !nicCfg.isBlacklist {
			return false
		}
		// In blacklist mode, still apply the auto-filter below.
	}

	switch {
	case strings.HasPrefix(v.Name, "lo"),
		strings.HasPrefix(v.Name, "docker"),
		strings.HasPrefix(v.Name, "br-"),
		strings.HasPrefix(v.Name, "veth"),
		strings.HasPrefix(v.Name, "bond"),
		strings.HasPrefix(v.Name, "cali"),
		v.Name == "awdl0",
		(v.BytesRecv == 0 && v.BytesSent == 0):
		return true
	default:
		return false
	}
}

// Refresh identities from the already collected counters; driver lookup is only
// paid for new interfaces. Zero-counter interfaces require UP and an address,
// or explicit NICS inclusion. Idle tracked interfaces retain their baselines.
func (a *Agent) reconcileNetworkInterfaces(counters []psutilNet.IOCountersStat) {
	present := make(map[string]bool, len(counters))
	var identities []psutilNet.InterfaceStat
	loaded := false
	for _, v := range counters {
		present[v.Name] = true
		if _, ok := a.netInterfaces[v.Name]; ok {
			continue
		}
		if skipNetworkInterface(v, a.nicConfig) {
			probe := v
			probe.BytesSent, probe.BytesRecv = 1, 1
			if v.BytesSent != 0 || v.BytesRecv != 0 || skipNetworkInterface(probe, a.nicConfig) {
				continue
			}
			if !loaded {
				identities, _ = networkInterfaces()
				loaded = true
			}
			active := false
			for _, iface := range identities {
				if iface.Name == v.Name && len(iface.Addrs) > 0 {
					for _, flag := range iface.Flags {
						if flag == "up" {
							active = true
						}
					}
				}
			}
			if !active {
				continue
			}
		}
		a.netInterfaces[v.Name] = isNvidiaEthernet(v.Name)
	}
	for name := range a.netInterfaces {
		if !present[name] {
			delete(a.netInterfaces, name)
		}
	}
}
func saturatingAdd(a, b uint64) uint64 {
	const maximum = ^uint64(0)
	if b > maximum-a {
		return maximum
	}
	return a + b
}
