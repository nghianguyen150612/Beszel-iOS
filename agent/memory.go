package agent

import (
	"runtime"

	"github.com/henrygd/beszel/agent/utils"
	"github.com/henrygd/beszel/agent/zfs"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/shirou/gopsutil/v4/mem"
)

// memoryMetrics holds byte counters. For cgroups, Used excludes file cache,
// while retaining shared memory/tmpfs. BuffCache is the subtracted file cache.
type memoryMetrics struct {
	Total, Used, BuffCache uint64
}

var memoryVirtualMemory = mem.VirtualMemory

func (a *Agent) updateMemoryStats(stats *system.Stats) {
	v, err := memoryVirtualMemory()
	var hostTotal uint64
	if err == nil && v != nil {
		hostTotal = v.Total
		// Swap fields are populated only when supplied by the platform source.
		// iOS acquisition leaves unsupported swap fields absent.
		stats.Swap = utils.BytesToGigabytes(v.SwapTotal)
		stats.SwapUsed = utils.BytesToGigabytes(saturatingSub(v.SwapTotal, v.SwapFree))
	}

	metrics, fromCgroup := containerMemoryMetrics(hostTotal, a.forceUseCgroup)
	if !fromCgroup {
		if err != nil || v == nil {
			return
		}
		used, cacheBuff := platformMemoryUsage(v, a.memCalc == "htop", runtime.GOOS)
		metrics = memoryMetrics{Total: v.Total, Used: used, BuffCache: cacheBuff}
		// Host ARC must not be subtracted from container-scoped accounting.
		if a.zfs {
			if arcSize, _ := zfs.ARCSize(); arcSize > 0 && arcSize < metrics.Used {
				metrics.Used -= arcSize
				stats.MemZfsArc = utils.BytesToGigabytes(arcSize)
			}
		}
	}

	stats.Mem = utils.BytesToGigabytes(metrics.Total)
	stats.MemUsed = utils.BytesToGigabytes(metrics.Used)
	stats.MemBuffCache = utils.BytesToGigabytes(metrics.BuffCache)
	if metrics.Total > 0 {
		stats.MemPct = utils.TwoDecimals(float64(metrics.Used) / float64(metrics.Total) * 100)
	}
	if a.systemDetails.MemoryTotal != metrics.Total {
		a.updateSystemDetails(func(details *system.Details) {
			details.MemoryTotal = metrics.Total
		})
	}
}

// iOS has no equivalent cache/buffer measurement in this acquisition path.
// MEM_CALC=htop is a Linux formula and must not reclassify iOS inactive pages.
func platformMemoryUsage(v *mem.VirtualMemoryStat, htop bool, platform string) (uint64, uint64) {
	if platform == "ios" {
		return min(v.Used, v.Total), 0
	}
	used, cache, _ := calculateHostMemoryUsage(v, htop)
	return used, cache
}
