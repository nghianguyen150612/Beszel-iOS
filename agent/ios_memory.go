package agent

import (
	"errors"
	"github.com/shirou/gopsutil/v4/mem"
	"math"
)

// Preserve Darwin's free+inactive available semantics, widening page counters
// before arithmetic. No cache or swap value is inferred from inactive pages.
func iosMemoryFromPages(total, pageSize, free, active, inactive, wired uint64) (*mem.VirtualMemoryStat, error) {
	if total == 0 || pageSize == 0 {
		return nil, errors.New("invalid iOS memory total or page size")
	}
	pagesToBytes := func(pages uint64) uint64 {
		if pages > math.MaxUint64/pageSize {
			return total
		}
		return min(total, pages*pageSize)
	}
	freeBytes, inactiveBytes := pagesToBytes(free), pagesToBytes(inactive)
	available := freeBytes + min(inactiveBytes, total-freeBytes)
	used := total - available
	return &mem.VirtualMemoryStat{Total: total, Available: available, Used: used,
		UsedPercent: float64(used) / float64(total) * 100, Free: freeBytes,
		Active: pagesToBytes(active), Inactive: inactiveBytes, Wired: pagesToBytes(wired)}, nil
}
