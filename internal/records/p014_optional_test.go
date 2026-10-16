//go:build testing

package records_test

import (
	"encoding/json"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/records"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestP014SparseDeviceRollups(t *testing.T) {
	input := []system.Stats{
		{Temperatures: map[string]float64{"a": 40, "zero": 0}, Fans: map[string]uint16{"a": 1000}, ExtraFs: map[string]*system.FsStats{"a": {DiskTotal: 100, DiskUsed: 20}}, GPUData: map[string]system.GPUData{"a": {Name: "GPU A", Usage: 60, Power: 12.5, PowerPkg: 22.75, Engines: map[string]float64{"render": 40}}}, CpuCoresUsage: system.Uint8Slice{80, 40}, CpuBreakdown: []float64{20, 10}, NetworkInterfaces: map[string][4]uint64{"a": {100, 200, 500, 900}}},
		{Temperatures: map[string]float64{"b": 80}, Fans: map[string]uint16{"b": 2000}, GPUData: map[string]system.GPUData{"b": {Name: "GPU B", Usage: 90, PowerPkg: 15}}, CpuCoresUsage: system.Uint8Slice{0}},
		{},
	}
	got := records.AverageSystemStatsSlice(input)
	assert.Equal(t, 40.0, got.Temperatures["a"])
	assert.Equal(t, 0.0, got.Temperatures["zero"])
	assert.Equal(t, uint16(1000), got.Fans["a"])
	assert.Equal(t, 100.0, got.ExtraFs["a"].DiskTotal)
	assert.Equal(t, 60.0, got.GPUData["a"].Usage)
	assert.Equal(t, 22.75, got.GPUData["a"].PowerPkg)
	assert.Equal(t, 40.0, got.GPUData["a"].Engines["render"])
	assert.Equal(t, system.Uint8Slice{40, 40}, got.CpuCoresUsage)
	assert.Equal(t, []float64{20, 10}, got.CpuBreakdown)
	assert.Equal(t, [4]uint64{100, 200, 500, 900}, got.NetworkInterfaces["a"])
	assert.Equal(t, uint8(80), input[0].CpuCoresUsage[0])
}

func TestP014NullOptionalFilesystem(t *testing.T) {
	require.NotPanics(t, func() {
		got := records.AverageSystemStatsSlice([]system.Stats{{ExtraFs: map[string]*system.FsStats{"bad": nil}}, {}})
		require.Empty(t, got.ExtraFs)
	})
}

func TestP014OptionalPowerPresence(t *testing.T) {
	var input []system.Stats
	require.NoError(t, json.Unmarshal([]byte(`[
 {"g":{"a":{"n":"A","u":0,"p":8,"pp":20}}},
 {"g":{"a":{"n":"A","u":0}}},
 {"g":{"a":{"n":"A","u":0,"p":0,"pp":0}}}
 ]`), &input))
	got := records.AverageSystemStatsSlice(input)
	require.Equal(t, 4.0, got.GPUData["a"].Power, "only the available 8W and measured 0W are observations")
	require.Equal(t, 10.0, got.GPUData["a"].PowerPkg)
}
