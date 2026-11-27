//go:build !ios || !stockios

package agent

import "github.com/shirou/gopsutil/v4/cpu"

// init initializes the CPU monitoring by storing the initial CPU times
// for the default 60-second cache interval.
func init() {
	if times, err := cpu.Times(false); err == nil && len(times) > 0 {
		lastCpuTimes[60000] = times[0]
	}
	if perCoreTimes, err := cpu.Times(true); err == nil && len(perCoreTimes) > 0 {
		lastPerCoreCpuTimes[60000] = perCoreTimes
	}
}
