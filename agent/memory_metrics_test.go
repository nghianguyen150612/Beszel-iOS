//go:build testing

package agent

import (
	"errors"
	"testing"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/stretchr/testify/require"
)

func TestPlatformSwapUsage(t *testing.T) {
	v := &mem.VirtualMemoryStat{SwapTotal: 2 << 20, SwapFree: 1 << 20}
	if swap, ok := platformSwapUsage(v, "ios"); ok {
		t.Fatalf("iOS swap must be unavailable, got %v", swap)
	}
	swap, ok := platformSwapUsage(v, "linux")
	require.True(t, ok)
	require.Equal(t, [2]uint64{2 << 20, 1 << 20}, swap)

	// An invalid free counter must not wrap or exceed total.
	v.SwapFree = 3 << 20
	swap, ok = platformSwapUsage(v, "linux")
	require.True(t, ok)
	require.Equal(t, [2]uint64{2 << 20, 0}, swap)
}

func TestNormalizeMemoryMetrics(t *testing.T) {
	require.Equal(t, memoryMetrics{}, normalizeMemoryMetrics(memoryMetrics{Used: 1, BuffCache: 2}))
	require.Equal(t, memoryMetrics{Total: 100, Used: 100, BuffCache: 100},
		normalizeMemoryMetrics(memoryMetrics{Total: 100, Used: 101, BuffCache: 200}))
	require.Equal(t, memoryMetrics{Total: 8 << 20, Used: 3 << 20, BuffCache: 0},
		normalizeMemoryMetrics(memoryMetrics{Total: 8 << 20, Used: 3 << 20}))
}

func TestUpdateMemoryStatsClearsUnavailableSample(t *testing.T) {
	original := memoryVirtualMemory
	t.Cleanup(func() { memoryVirtualMemory = original })
	memoryVirtualMemory = func() (*mem.VirtualMemoryStat, error) {
		return nil, errors.New("temporary unavailable")
	}
	stats := system.Stats{
		Mem: 4, MemUsed: 3, MemPct: 75, MemBuffCache: 1, MemZfsArc: .5,
		Swap: 2, SwapUsed: 1,
		MemBytes:  [4]uint64{4 << 30, 3 << 30, 1 << 30, 1 << 29},
		SwapBytes: [2]uint64{2 << 30, 1 << 30},
	}
	(&Agent{}).updateMemoryStats(&stats)
	require.Zero(t, stats.Mem)
	require.Zero(t, stats.MemUsed)
	require.Zero(t, stats.MemPct)
	require.Zero(t, stats.MemBuffCache)
	require.Zero(t, stats.MemZfsArc)
	require.Zero(t, stats.Swap)
	require.Zero(t, stats.SwapUsed)
	require.Equal(t, [4]uint64{}, stats.MemBytes)
	require.Equal(t, [2]uint64{}, stats.SwapBytes)
}
