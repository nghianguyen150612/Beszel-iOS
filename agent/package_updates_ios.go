//go:build ios

package agent

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const procursusProbeTimeout = 3 * time.Second

func newIOSPackageUpdatesManager() *packageUpdatesManager {
	interval, enabled := packageUpdatesInterval()
	if !enabled {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), procursusProbeTimeout)
	defer cancel()
	detection := detectProcursusAPT(ctx, procursusOSProbe{}, []string{"/usr/bin", "/var/jb/usr/bin"})
	if detection.status != procursusSupported {
		slog.Debug("iOS package updates unavailable", "reason", detection.status.String())
		return nil
	}
	slog.Debug("Package updates", "manager", "procursus-apt", "interval", interval, "layout", detection.layout.kind)
	return &packageUpdatesManager{
		name:     "apt",
		check:    newProcursusAPTCheck(detection.layout),
		interval: interval,
	}
}

type procursusOSProbe struct{}

func (procursusOSProbe) stat(path string) (os.FileInfo, error)         { return os.Stat(path) }
func (procursusOSProbe) readDir(path string) ([]os.DirEntry, error)    { return os.ReadDir(path) }
func (procursusOSProbe) open(path string) (procursusReadCloser, error) { return os.Open(path) }
func (procursusOSProbe) evalSymlinks(path string) (string, error)      { return filepath.EvalSymlinks(path) }
func (procursusOSProbe) lookPath(name string) (string, error)          { return exec.LookPath(name) }
func (procursusOSProbe) run(ctx context.Context, executable string, args ...string) (string, error) {
	if !procursusCommandAllowed(executable, args) {
		return "", errProcursusCommandNotAllowed
	}
	return runProcursusProbeCommand(ctx, executable, args...)
}
