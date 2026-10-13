//go:build !darwin || ios

package agent

// startPowermetricsCollector is a no-op outside macOS; the real implementation is in gpu_darwin.go.
func (gm *GPUManager) startPowermetricsCollector() {}

// startMacmonCollector is a no-op outside macOS; the real implementation is in gpu_darwin.go.
func (gm *GPUManager) startMacmonCollector() {}
