//go:build testing

package agent

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/henrygd/beszel/agent/deltatracker"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP008CPUFixtures(t *testing.T) {
	saved := lastCpuTimes
	t.Cleanup(func() { lastCpuTimes = saved })
	baseline := cpu.TimesStat{User: 10, System: 5, Idle: 85}
	for _, tt := range []struct {
		name            string
		before, current cpu.TimesStat
		want            float64
		bad             bool
	}{
		{"steady", baseline, cpu.TimesStat{User: 20, System: 10, Idle: 170}, 15, false},
		{"idle", baseline, cpu.TimesStat{User: 10, System: 5, Idle: 185}, 0, false},
		{"saturated", baseline, cpu.TimesStat{User: 110, System: 5, Idle: 85}, 100, false},
		{"zero elapsed", baseline, baseline, 0, true},
		{"complete reset", baseline, cpu.TimesStat{}, 0, true},
		{"one counter reset while total grows", baseline, cpu.TimesStat{User: 1, System: 5, Idle: 200}, 0, true},
		{"negative", baseline, cpu.TimesStat{User: -1, Idle: 200}, 0, true},
		{"nan", baseline, cpu.TimesStat{User: math.NaN(), Idle: 200}, 0, true},
		{"inf", baseline, cpu.TimesStat{User: math.Inf(1), Idle: 200}, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lastCpuTimes = map[uint16]cpu.TimesStat{1000: tt.before}
			metrics, err := cpuMetricsSample(1000, tt.current)
			if tt.bad {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.InDelta(t, tt.want, metrics.Total, 1e-9)
			for _, v := range []float64{metrics.Total, metrics.User, metrics.System, metrics.Idle, metrics.Iowait, metrics.Steal} {
				assert.False(t, math.IsNaN(v))
				assert.False(t, math.IsInf(v, 0))
				assert.GreaterOrEqual(t, v, 0.0)
				assert.LessOrEqual(t, v, 100.0)
			}
		})
	}
	t.Run("first sample and recovery", func(t *testing.T) {
		lastCpuTimes = make(map[uint16]cpu.TimesStat)
		_, err := cpuMetricsSample(1000, baseline)
		require.Error(t, err)
		got, err := cpuMetricsSample(1000, cpu.TimesStat{User: 20, System: 10, Idle: 170})
		require.NoError(t, err)
		assert.Equal(t, 15.0, got.Total)
	})
}
func TestP008CPUAcquisitionRecoveryAndCores(t *testing.T) {
	savedTimes, savedTotals, savedCores := cpuTimes, lastCpuTimes, lastPerCoreCpuTimes
	t.Cleanup(func() { cpuTimes, lastCpuTimes, lastPerCoreCpuTimes = savedTimes, savedTotals, savedCores })
	lastCpuTimes = map[uint16]cpu.TimesStat{}
	lastPerCoreCpuTimes = map[uint16][]cpu.TimesStat{}
	fail, empty := false, false
	sample := cpu.TimesStat{User: 10, System: 5, Idle: 85}
	count := 2
	cpuTimes = func(perCore bool) ([]cpu.TimesStat, error) {
		if fail {
			return nil, errors.New("source failed")
		}
		if empty {
			return nil, nil
		}
		if perCore {
			out := make([]cpu.TimesStat, count)
			for i := range out {
				out[i] = sample
			}
			return out, nil
		}
		return []cpu.TimesStat{sample}, nil
	}
	_, err := getCpuMetrics(60000, false)
	require.Error(t, err)
	_, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	sample = cpu.TimesStat{User: 20, System: 10, Idle: 170}
	metrics, err := getCpuMetrics(60000, false)
	require.NoError(t, err)
	assert.Equal(t, 15.0, metrics.Total)
	cores, err := getPerCoreCpuUsage(60000)
	require.NoError(t, err)
	assert.Equal(t, system.Uint8Slice{15, 15}, cores)
	fail = true
	_, err = getCpuMetrics(60000, false)
	require.Error(t, err)
	_, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	fail = false
	sample = cpu.TimesStat{User: 30, System: 15, Idle: 255}
	_, err = getCpuMetrics(60000, false)
	require.Error(t, err)
	_, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	sample = cpu.TimesStat{User: 40, System: 20, Idle: 340}
	_, err = getCpuMetrics(60000, false)
	require.NoError(t, err)
	_, err = getPerCoreCpuUsage(60000)
	require.NoError(t, err)
	count = 3
	cores, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	assert.Nil(t, cores)
	// Unchanged, malformed and reset per-core counters remain unavailable.
	_, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	sample.User = math.NaN()
	cores, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	assert.Nil(t, cores)
	sample = cpu.TimesStat{User: 10, System: 5, Idle: 85}
	_, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	sample = cpu.TimesStat{User: 20, System: 10, Idle: 170}
	_, err = getPerCoreCpuUsage(60000)
	require.NoError(t, err)
	assert.False(t, validCpuDelta(cpu.TimesStat{CPU: "cpu0"}, cpu.TimesStat{CPU: "cpu1", Idle: 100}))
	empty = true
	_, err = getCpuMetrics(60000, false)
	require.Error(t, err)
	_, err = getPerCoreCpuUsage(60000)
	require.Error(t, err)
	empty = false
	// Runtime acquisition guards the shared baselines, including multiple Agents.
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _, _ = getPerCoreCpuUsage(1000) })
	}
	wg.Wait()
}
func TestP008IOSMemoryFixtures(t *testing.T) {
	for _, tt := range []struct {
		name                                       string
		total, page, free, active, inactive, wired uint64
		bad                                        bool
	}{
		{"normal", 1 << 30, 4096, 1000, 100000, 10000, 50000, false},
		{"low memory", 1 << 30, 4096, 0, 200000, 1, 50000, false},
		{"available exceeds total", 1 << 30, 4096, 400000, 0, 400000, 0, false},
		{"zero total", 0, 4096, 0, 0, 0, 0, true},
		{"zero page size", 1 << 30, 0, 0, 0, 0, 0, true},
		{"large uint64", math.MaxUint64, 4096, math.MaxUint64, math.MaxUint64, math.MaxUint64, math.MaxUint64, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v, err := iosMemoryFromPages(tt.total, tt.page, tt.free, tt.active, tt.inactive, tt.wired)
			if tt.bad {
				require.Error(t, err)
				assert.Nil(t, v)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, v.Total, v.Used+v.Available)
			assert.LessOrEqual(t, v.Used, v.Total)
			assert.LessOrEqual(t, v.Available, v.Total)
			assert.LessOrEqual(t, v.UsedPercent, 100.0)
			assert.Zero(t, v.Cached)
			assert.Zero(t, v.Buffers)
			assert.Zero(t, v.SwapTotal)
			assert.Zero(t, v.SwapFree)
			for _, htop := range []bool{false, true} {
				used, cache := platformMemoryUsage(v, htop, "ios")
				assert.Equal(t, v.Used, used)
				assert.Zero(t, cache)
			}
		})
	}
}
func TestP008MemoryPollingFailureRecovery(t *testing.T) {
	saved := memoryVirtualMemory
	t.Cleanup(func() { memoryVirtualMemory = saved })
	a := &Agent{}
	fail := false
	memoryVirtualMemory = func() (*mem.VirtualMemoryStat, error) {
		if fail {
			return nil, errors.New("unavailable")
		}
		return &mem.VirtualMemoryStat{Total: 1000, Available: 200, Used: 800, Free: 100}, nil
	}
	for _, f := range []bool{false, false, true, false} {
		fail = f
		var stats system.Stats
		a.updateMemoryStats(&stats)
		if fail {
			assert.Zero(t, stats.Mem)
			assert.Zero(t, stats.MemPct)
		} else {
			assert.Equal(t, 80.0, stats.MemPct)
		}
		assert.Zero(t, stats.Swap)
		assert.Zero(t, stats.SwapUsed)
	}
}
func TestP008IOSDiskDiscovery(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		parts                  []disk.PartitionStat
		setting, extras, mount string
		count                  int
	}{
		{"root only", []disk.PartitionStat{{Device: "/dev/disk0s1", Mountpoint: "/", Opts: []string{"ro"}}}, "", "", "/", 1},
		{"related mounts not automatic", []disk.PartitionStat{{Device: "/dev/disk0s1", Mountpoint: "/"}, {Device: "/dev/disk0s2", Mountpoint: "/private/var"}}, "", "", "/", 1},
		{"missing device", []disk.PartitionStat{{Mountpoint: "/"}}, "", "", "/", 1},
		{"missing partitions", nil, "", "", "/", 1},
		{"nonstandard configured mount", []disk.PartitionStat{{Device: "/dev/disk9", Mountpoint: "/odd name"}}, "/odd name", "", "/odd name", 1},
		{"duplicate same device", []disk.PartitionStat{{Device: "/dev/disk0", Mountpoint: "/"}, {Device: "/dev/disk0", Mountpoint: "/related"}}, "", "/related", "/", 1},
		{"multiple explicit partitions", []disk.PartitionStat{{Device: "/dev/disk0", Mountpoint: "/"}, {Device: "/dev/disk1", Mountpoint: "/other"}}, "", "/other", "/", 2},
		{"inaccessible extra", nil, "", "/missing", "/", 1},
		{"zero capacity extra", nil, "", "/zero", "/", 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{fsStats: map[string]*system.FsStats{}}
			d := diskDiscovery{agent: a, partitions: tt.parts, ctx: fsRegistrationContext{filesystem: tt.setting}, usageFn: func(path string) (*disk.UsageStat, error) {
				if path == "/zero" {
					return &disk.UsageStat{}, nil
				}
				return nil, errors.New("inaccessible")
			}}
			d.addIOSFilesystems(tt.extras)
			require.Len(t, a.fsStats, tt.count)
			roots := 0
			for _, fs := range a.fsStats {
				if fs.Root {
					roots++
					assert.Equal(t, tt.mount, fs.Mountpoint)
				}
			}
			assert.Equal(t, 1, roots)
			assert.Empty(t, a.fsNames, "usage identity must not invent an I/O device")
		})
	}
	_, err := unavailableIOSDiskIO()
	require.ErrorIs(t, err, errIOSDiskIO)
}
func TestP008DiskUsageRecovery(t *testing.T) {
	saved := diskUsage
	t.Cleanup(func() { diskUsage = saved })
	a := &Agent{fsStats: map[string]*system.FsStats{"root": {Root: true, Mountpoint: "/"}}}
	fail := false
	diskUsage = func(string) (*disk.UsageStat, error) {
		if fail {
			return nil, errors.New("root inaccessible")
		}
		return &disk.UsageStat{Total: 1000, Used: 2000, UsedPercent: math.Inf(1)}, nil
	}
	for _, f := range []bool{false, true, false} {
		fail = f
		var stats system.Stats
		a.updateDiskUsage(&stats)
		assert.LessOrEqual(t, stats.DiskUsed, stats.DiskTotal)
		assert.False(t, math.IsInf(stats.DiskPct, 0))
		if f {
			assert.Zero(t, stats.DiskTotal)
		}
	}
	diskUsage = func(string) (*disk.UsageStat, error) { return &disk.UsageStat{}, nil }
	var stats system.Stats
	a.updateDiskUsage(&stats)
	assert.Zero(t, stats.DiskPct)
}
func TestP008DiskIOResetDisappearanceAndRecovery(t *testing.T) {
	oldCounters, oldNow := diskIOCounters, diskNow
	t.Cleanup(func() { diskIOCounters, diskNow = oldCounters, oldNow })
	a := &Agent{fsStats: map[string]*system.FsStats{"disk0": {Root: true, Mountpoint: "/"}}, fsNames: []string{"disk0"}, diskPrev: map[uint16]map[string]prevDisk{}}
	now := time.Unix(100, 0)
	diskNow = func() time.Time { return now }
	current := disk.IOCountersStat{Name: "disk0", ReadBytes: 1000, WriteBytes: 2000, ReadCount: 100, WriteCount: 100}
	fail, missing := false, false
	diskIOCounters = func(...string) (map[string]disk.IOCountersStat, error) {
		if fail {
			return nil, errors.New("source error")
		}
		if missing {
			return map[string]disk.IOCountersStat{}, nil
		}
		return map[string]disk.IOCountersStat{"disk0": current}, nil
	}
	poll := func() system.Stats { var s system.Stats; a.updateDiskIo(1000, &s); return s }
	assert.Zero(t, poll().DiskIO)
	now = now.Add(time.Second)
	current.ReadBytes += 1000
	current.WriteBytes += 2000
	assert.Equal(t, [2]uint64{1000, 2000}, poll().DiskIO)
	now = now.Add(time.Second)
	current.ReadBytes = 5
	current.WriteBytes = 8
	current.ReadCount = 0
	current.WriteCount = 0
	assert.Equal(t, [2]uint64{}, poll().DiskIO)
	now = now.Add(time.Second)
	current.ReadBytes += 1000
	current.WriteBytes += 2000
	assert.Equal(t, [2]uint64{1000, 2000}, poll().DiskIO)
	missing = true
	now = now.Add(time.Second)
	assert.Zero(t, poll().DiskIO)
	assert.Zero(t, a.fsStats["disk0"].DiskReadBytes)
	missing = false
	now = now.Add(time.Second)
	assert.Zero(t, poll().DiskIO)
	fail = true
	now = now.Add(time.Second)
	assert.Zero(t, poll().DiskIO)
	fail = false
	now = now.Add(time.Second)
	assert.Zero(t, poll().DiskIO)
	now = now.Add(time.Second)
	current.ReadBytes += 1000
	assert.Equal(t, uint64(1000), poll().DiskIO[0])
	now = now.Add(-time.Hour)
	assert.Zero(t, poll().DiskIO)
}
func TestP008NetworkDiscoveryPollingRecovery(t *testing.T) {
	savedIO, savedInterfaces := networkIOCounters, networkInterfaces
	t.Cleanup(func() { networkIOCounters, networkInterfaces = savedIO, savedInterfaces })
	networkInterfaces = func() (psnet.InterfaceStatList, error) {
		return []psnet.InterfaceStat{
			{Name: "en0", Flags: []string{"up"}, Addrs: []psnet.InterfaceAddr{{Addr: "192.0.2.1/24"}}},
			{Name: "inactive", Flags: []string{"down"}},
		}, nil
	}
	a := &Agent{netInterfaces: map[string]bool{}, netIoStats: map[uint16]system.NetIoStats{}, netInterfaceDeltaTrackers: map[uint16]*deltatracker.DeltaTracker[string, uint64]{}}
	counters := []psnet.IOCountersStat{{Name: "en0"}, {Name: "inactive"}, {Name: "lo0", BytesRecv: 1}, {Name: "awdl0", BytesSent: 1}}
	fail := false
	networkIOCounters = func(bool) ([]psnet.IOCountersStat, error) {
		if fail {
			return nil, errors.New("source error")
		}
		return counters, nil
	}
	poll := func() system.Stats { var s system.Stats; a.updateNetworkStats(1000, &s); return s }
	stats := poll()
	assert.Contains(t, stats.NetworkInterfaces, "en0")
	assert.NotContains(t, stats.NetworkInterfaces, "inactive")
	assert.NotContains(t, stats.NetworkInterfaces, "lo0")
	assert.Contains(t, stats.NetworkInterfaces, "awdl0", "preserve existing active-interface policy; NICS can exclude it")
	assert.Zero(t, stats.Bandwidth)
	counters = append(counters, psnet.IOCountersStat{Name: "utun8", BytesRecv: 1000}, psnet.IOCountersStat{Name: "pdp_ip0", BytesSent: 1000})
	stats = poll()
	assert.Contains(t, stats.NetworkInterfaces, "utun8")
	assert.Contains(t, stats.NetworkInterfaces, "pdp_ip0")
	assert.Zero(t, stats.NetworkInterfaces["utun8"][1])
	counters = []psnet.IOCountersStat{{Name: "en0", BytesSent: 2000}}
	stats = poll()
	assert.NotContains(t, stats.NetworkInterfaces, "utun8")
	fail = true
	stats = poll()
	assert.Empty(t, stats.NetworkInterfaces)
	assert.Zero(t, stats.Bandwidth)
	fail = false
	counters = append(counters, psnet.IOCountersStat{Name: "utun8", BytesRecv: 3000})
	stats = poll()
	assert.Zero(t, stats.Bandwidth)
	assert.Contains(t, stats.NetworkInterfaces, "utun8")
	a.nicConfig = newNicConfig("lo0,inactive")
	a.netInterfaces = map[string]bool{}
	counters = []psnet.IOCountersStat{{Name: "lo0"}, {Name: "inactive"}, {Name: "en0", BytesRecv: 1}}
	stats = poll()
	assert.Contains(t, stats.NetworkInterfaces, "lo0")
	assert.Contains(t, stats.NetworkInterfaces, "inactive")
	assert.NotContains(t, stats.NetworkInterfaces, "en0")
	a.nicConfig = newNicConfig("-utun*")
	a.netInterfaces = map[string]bool{}
	counters = []psnet.IOCountersStat{{Name: "utun7", BytesSent: 1}, {Name: "en0", BytesRecv: 1}}
	stats = poll()
	assert.NotContains(t, stats.NetworkInterfaces, "utun7")
	assert.Contains(t, stats.NetworkInterfaces, "en0")
}
func TestP008NetworkRatesIntervalsAndAnomalies(t *testing.T) {
	a := &Agent{netInterfaces: map[string]bool{"en0": false, "utun0": false}, netInterfaceDeltaTrackers: map[uint16]*deltatracker.DeltaTracker[string, uint64]{}, netIoStats: map[uint16]system.NetIoStats{}}
	poll := func(interval uint16, ms uint64, counters []psnet.IOCountersStat) system.Stats {
		var s system.Stats
		a.ensureNetworkInterfacesMap(&s)
		up, down := a.sumAndTrackPerNicDeltas(interval, ms, counters, &s)
		a.applyNetworkTotals(interval, counters, &s, system.NetIoStats{}, up, down)
		return s
	}
	initial := []psnet.IOCountersStat{{Name: "en0", BytesSent: 1000}, {Name: "utun0", BytesRecv: 1000}}
	assert.Zero(t, poll(1000, 1000, initial).Bandwidth)
	assert.Zero(t, poll(60000, 60000, initial).Bandwidth)
	second := []psnet.IOCountersStat{{Name: "en0", BytesSent: 3000}, {Name: "utun0", BytesRecv: 5000}}
	assert.Equal(t, [2]uint64{2000, 4000}, poll(1000, 1000, second).Bandwidth)
	assert.Equal(t, [2]uint64{1000, 2000}, poll(60000, 2000, second).Bandwidth)
	assert.Zero(t, poll(1000, 1000, second).Bandwidth, "idle")
	assert.Zero(t, poll(1000, 0, second).Bandwidth, "zero elapsed")
	reset := []psnet.IOCountersStat{{Name: "en0", BytesSent: 2000}, {Name: "utun0", BytesRecv: 2000}}
	assert.Zero(t, poll(1000, 1000, reset).Bandwidth)
	huge := []psnet.IOCountersStat{{Name: "en0", BytesSent: math.MaxUint64}}
	stats := poll(1000, 1, huge)
	assert.Zero(t, stats.Bandwidth)
	for _, v := range stats.NetworkInterfaces {
		assert.Zero(t, v[0])
		assert.Zero(t, v[1])
	}
	a.netIoStats[1000] = system.NetIoStats{Time: time.Now().Add(time.Hour)}
	_, ms := a.loadAndTickNetBaseline(1000)
	assert.Zero(t, ms)
}
func TestP008CounterRateOverflow(t *testing.T) {
	for _, tt := range []struct{ current, previous, ms, want uint64 }{
		{3000, 1000, 1000, 2000}, {1000, 3000, 1000, 0}, {1000, 0, 0, 0},
		{math.MaxUint64, 0, 1000, math.MaxUint64}, {math.MaxUint64, 0, 1, math.MaxUint64},
		{math.MaxUint64, math.MaxUint64 - 1000, 2000, 500},
	} {
		assert.Equal(t, tt.want, rateFromCounters(tt.current, tt.previous, tt.ms))
	}
}
