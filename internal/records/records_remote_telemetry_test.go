//go:build testing

package records_test

import (
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/records"
	"github.com/henrygd/beszel/internal/testutil/telemetry"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestP014OptionalRollupDoesNotInventZero(t *testing.T) {
	for _, f := range telemetry.All() {
		t.Run(f.Name, func(t *testing.T) {
			original := f.Data.Stats
			avg := records.AverageSystemStatsSlice([]system.Stats{original, {}})
			require.Equal(t, 12.5, avg.Cpu, "required CPU zero is measured")
			require.Equal(t, original.CpuCoresUsage, avg.CpuCoresUsage, "missing optional per-core readings are not idle cores")
			require.Equal(t, original.CpuBreakdown, avg.CpuBreakdown)
			require.Equal(t, original.Temperatures, avg.Temperatures)
			require.Equal(t, original.Fans, avg.Fans)
			require.Equal(t, original.GPUData, avg.GPUData)
			for name, fs := range original.ExtraFs {
				require.Equal(t, fs.DiskTotal, avg.ExtraFs[name].DiskTotal)
				require.Equal(t, fs.DiskUsed, avg.ExtraFs[name].DiskUsed)
				require.Equal(t, fs.DiskReadBytes, avg.ExtraFs[name].DiskReadBytes)
				require.Equal(t, fs.DiskWriteBytes, avg.ExtraFs[name].DiskWriteBytes)
			}
			require.Equal(t, original.NetworkInterfaces, avg.NetworkInterfaces)
			require.Equal(t, original.WiFi, avg.WiFi)
			require.Equal(t, original.Battery, avg.Battery)
		})
	}
}

func TestP014RollupPerDevicePresenceAndZero(t *testing.T) {
	avg := records.AverageSystemStatsSlice([]system.Stats{
		{Temperatures: map[string]float64{"a": 40, "b": 60}, Fans: map[string]uint16{"a": 1200, "b": 2000}, GPUData: map[string]system.GPUData{"a": {Name: "a", Usage: 40, PowerPkg: 4}, "b": {Name: "b", Usage: 60, PowerPkg: 6}}},
		{Temperatures: map[string]float64{"a": 0}, Fans: map[string]uint16{"a": 0}, GPUData: map[string]system.GPUData{"a": {Name: "a", Usage: 0, PowerPkg: 0}}},
	})
	require.Equal(t, map[string]float64{"a": 20, "b": 60}, avg.Temperatures)
	require.Equal(t, map[string]uint16{"a": 600, "b": 2000}, avg.Fans)
	require.Equal(t, 20.0, avg.GPUData["a"].Usage)
	require.Equal(t, 60.0, avg.GPUData["b"].Usage)
	require.Equal(t, 2.0, avg.GPUData["a"].PowerPkg)
	require.Equal(t, 6.0, avg.GPUData["b"].PowerPkg)
}
