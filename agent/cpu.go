package agent

import (
	"errors"
	"math"
	"runtime"
	"sync"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/shirou/gopsutil/v4/cpu"
)

var cpuTimes = cpu.Times
var cpuTimesMu sync.Mutex
var errCpuSample = errors.New("CPU sample unavailable: baseline, reset or invalid counters")

var lastCpuTimes = make(map[uint16]cpu.TimesStat)
var lastPerCoreCpuTimes = make(map[uint16][]cpu.TimesStat)

// init initializes the CPU monitoring by storing the initial CPU times
// for the default 60-second cache interval.
func init() {
	if times, err := cpuTimes(false); err == nil && len(times) > 0 {
		lastCpuTimes[60000] = times[0]
	}
	if perCoreTimes, err := cpuTimes(true); err == nil && len(perCoreTimes) > 0 {
		lastPerCoreCpuTimes[60000] = perCoreTimes
	}
}

// CpuMetrics contains detailed CPU usage breakdown
type CpuMetrics struct {
	Total  float64
	User   float64
	System float64
	Iowait float64
	Steal  float64
	Idle   float64
	// fromCgroup is set when Total comes from cgroup accounting rather than
	// /proc/stat, so per-core /proc/stat usage would not match it.
	fromCgroup bool
}

// getCpuMetrics calculates detailed CPU usage metrics using cached previous measurements.
// It returns percentages for total, user, system, iowait, and steal time.
func getCpuMetrics(cacheTimeMs uint16, forceUseCgroup bool) (CpuMetrics, error) {
	// Inside LXC, lxcfs serves /proc/stat with the host cores' counters, not
	// the guest's own usage. Prefer cgroup accounting there or when requested. (#2332)
	if metrics, ok := containerCpuMetrics(cacheTimeMs, forceUseCgroup); ok {
		metrics.fromCgroup = true
		return metrics, nil
	}
	cpuTimesMu.Lock()
	defer cpuTimesMu.Unlock()
	times, err := cpuTimes(false)
	if err != nil || len(times) == 0 {
		delete(lastCpuTimes, cacheTimeMs)
		if cacheTimeMs != 60000 {
			delete(lastCpuTimes, 60000)
		}
		if err == nil {
			err = errCpuSample
		}
		return CpuMetrics{}, err
	}
	return cpuMetricsSample(cacheTimeMs, times[0])
}

// cpuMetricsFromTimes calculates host CPU usage from the latest counters.
func cpuMetricsFromTimes(cacheTimeMs uint16, current cpu.TimesStat) CpuMetrics {
	metrics, _ := cpuMetricsSample(cacheTimeMs, current)
	return metrics
}

// Caller holds cpuTimesMu for runtime collection; pure fixture callers are serial.
func cpuMetricsSample(cacheTimeMs uint16, current cpu.TimesStat) (CpuMetrics, error) {
	previous, ok := lastCpuTimes[cacheTimeMs]
	if !ok {
		previous, ok = lastCpuTimes[60000]
	}
	lastCpuTimes[cacheTimeMs] = current
	if !validCpuTimes(current) {
		delete(lastCpuTimes, cacheTimeMs)
		return CpuMetrics{}, errCpuSample
	}
	t1, t2 := previous, current
	t1All, _ := getAllBusy(t1)
	t2All, _ := getAllBusy(t2)
	totalDelta := t2All - t1All
	if !ok || !validCpuDelta(t1, t2) || totalDelta <= 0 || math.IsInf(totalDelta, 0) {
		return CpuMetrics{}, errCpuSample
	}

	metrics := CpuMetrics{
		Total:  calculateBusy(t1, t2),
		User:   clampPercent((t2.User - t1.User) / totalDelta * 100),
		System: clampPercent((t2.System - t1.System) / totalDelta * 100),
		Iowait: clampPercent((t2.Iowait - t1.Iowait) / totalDelta * 100),
		Steal:  clampPercent((t2.Steal - t1.Steal) / totalDelta * 100),
		Idle:   clampPercent((t2.Idle - t1.Idle) / totalDelta * 100),
	}

	return metrics, nil
}

// clampPercent ensures the percentage is between 0 and 100
func clampPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Min(100, math.Max(0, value))
}

// getPerCoreCpuUsage calculates per-core CPU busy usage as integer percentages (0-100).
// It uses cached previous measurements for the provided cache interval.
func getPerCoreCpuUsage(cacheTimeMs uint16) (system.Uint8Slice, error) {
	cpuTimesMu.Lock()
	defer cpuTimesMu.Unlock()
	perCoreTimes, err := cpuTimes(true)
	if err != nil || len(perCoreTimes) == 0 {
		delete(lastPerCoreCpuTimes, cacheTimeMs)
		if cacheTimeMs != 60000 {
			delete(lastPerCoreCpuTimes, 60000)
		}
		if err == nil {
			err = errCpuSample
		}
		return nil, err
	}

	// Initialize cache if needed
	if _, ok := lastPerCoreCpuTimes[cacheTimeMs]; !ok {
		lastPerCoreCpuTimes[cacheTimeMs] = lastPerCoreCpuTimes[60000]
	}

	lastTimes := lastPerCoreCpuTimes[cacheTimeMs]

	lastPerCoreCpuTimes[cacheTimeMs] = perCoreTimes
	if len(lastTimes) != len(perCoreTimes) {
		return nil, errCpuSample
	}
	length := len(perCoreTimes)
	for i := range perCoreTimes {
		a, _ := getAllBusy(lastTimes[i])
		b, _ := getAllBusy(perCoreTimes[i])
		if !validCpuDelta(lastTimes[i], perCoreTimes[i]) || b <= a || math.IsInf(b-a, 0) {
			return nil, errCpuSample
		}
	}

	usage := make([]uint8, length)
	for i := 0; i < length; i++ {
		t1 := lastTimes[i]
		t2 := perCoreTimes[i]
		usage[i] = uint8(math.Round(calculateBusy(t1, t2)))
	}

	lastPerCoreCpuTimes[cacheTimeMs] = perCoreTimes
	return usage, nil
}

// calculateBusy calculates the CPU busy percentage between two time points.
// It computes the ratio of busy time to total time elapsed between t1 and t2,
// returning a percentage clamped between 0 and 100.
func calculateBusy(t1, t2 cpu.TimesStat) float64 {
	t1All, t1Busy := getAllBusy(t1)
	t2All, t2Busy := getAllBusy(t2)

	if !validCpuDelta(t1, t2) || t2All <= t1All || t2Busy <= t1Busy {
		return 0
	}
	return clampPercent((t2Busy - t1Busy) / (t2All - t1All) * 100)
}

// getAllBusy calculates the total CPU time and busy CPU time from CPU times statistics.
// On Linux, it excludes guest and guest_nice time from the total to match kernel behavior.
// Returns total CPU time and busy CPU time (total minus idle and I/O wait time).
func getAllBusy(t cpu.TimesStat) (float64, float64) {
	tot := t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq +
		t.Softirq + t.Steal + t.Guest + t.GuestNice
	if runtime.GOOS == "linux" {
		tot -= t.Guest     // Linux 2.6.24+
		tot -= t.GuestNice // Linux 3.2.0+
	}

	busy := tot - t.Idle - t.Iowait

	return tot, busy
}

func cpuCounters(t cpu.TimesStat) [10]float64 {
	return [10]float64{t.User, t.System, t.Idle, t.Nice, t.Iowait, t.Irq, t.Softirq, t.Steal, t.Guest, t.GuestNice}
}
func validCpuTimes(t cpu.TimesStat) bool {
	for _, v := range cpuCounters(t) {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
func validCpuDelta(a, b cpu.TimesStat) bool {
	if !validCpuTimes(a) || !validCpuTimes(b) {
		return false
	}
	before, after := cpuCounters(a), cpuCounters(b)
	for i := range before {
		if after[i] < before[i] {
			return false
		}
	}
	return true
}
