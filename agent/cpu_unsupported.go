//go:build !linux

package agent

// containerCpuMetrics is Linux-only (cgroup accounting). Other platforms keep
// gopsutil host accounting (Mach on Darwin/iOS).
func containerCpuMetrics(uint16, bool) (CpuMetrics, bool) {
	return CpuMetrics{}, false
}

func (a *Agent) initializeCpu() {}

func (a *Agent) warnIfRootCgroup() {}
