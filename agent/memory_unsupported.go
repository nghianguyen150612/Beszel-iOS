//go:build !linux

package agent

import "runtime"

// Keep the established iOS behavior: its memory totals are not host-accounted.
func canSubtractZfsArc() bool { return runtime.GOOS != "ios" }

func containerMemoryMetrics(uint64, bool) (memoryMetrics, bool) {
	return memoryMetrics{}, false
}
