//go:build linux || darwin || ios || freebsd

package ioshelper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var fixtureBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "beszel-wifi-helper-fixture-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fixtureBinary = filepath.Join(dir, "fixture")
	command := exec.Command("go", "build", "-o", fixtureBinary, "./testdata/fakehelper")
	if out, err := command.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, string(out), err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	if err := fixtureReaper(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}
func helperPath(t *testing.T, mode string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), mode)
	require.NoError(t, os.Link(fixtureBinary, path))
	return path
}
func fixtureClient(t *testing.T, mode string) *Client {
	t.Helper()
	c, err := NewClient(helperPath(t, mode))
	require.NoError(t, err)
	c.exists = fixtureExists
	c.timeout = 100 * time.Millisecond
	c.waitDelay = 30 * time.Millisecond
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, c.Close(ctx))
	})
	return c
}
func waitPID(t *testing.T, path string) int {
	t.Helper()
	var pid int
	require.Eventually(t, func() bool {
		b, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(string(b))
		return err == nil && pid > 0
	}, time.Second, 5*time.Millisecond)
	return pid
}
func requireReaped(t *testing.T, pid int) {
	t.Helper()
	require.Eventually(t, func() bool { return syscall.Kill(pid, 0) == syscall.ESRCH }, time.Second, 5*time.Millisecond, "direct child must be reaped, not just killed")
}
func TestClientResultsAndFailures(t *testing.T) {
	for _, mode := range []string{"valid", "empty", "invalid", "oversize", "nonzero", "crash", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			c := fixtureClient(t, mode)
			start := time.Now()
			got, err := c.Read(context.Background())
			require.Less(t, time.Since(start), time.Second)
			if mode == "valid" {
				require.NoError(t, err)
				require.Equal(t, -40.0, *got["en0"].Signal)
			} else if mode == "unavailable" {
				require.NoError(t, err)
				require.Nil(t, got)
			} else {
				require.Error(t, err)
				require.Nil(t, got)
			}
			if mode == "oversize" {
				require.ErrorIs(t, err, ErrOutputLimit)
			}
			requireReaped(t, waitPID(t, c.path+".pid"))
		})
	}
}
func TestClientMissingPermissionAndRecovery(t *testing.T) {
	_, err := NewClient("relative-helper")
	require.ErrorIs(t, err, ErrExecutable)
	path := filepath.Join(t.TempDir(), "valid")
	c, err := NewClient(path)
	require.NoError(t, err)
	c.exists = fixtureExists
	got, err := c.Read(context.Background())
	require.ErrorIs(t, err, os.ErrNotExist)
	require.Nil(t, got)
	require.NoError(t, os.Link(fixtureBinary, path))
	got, err = c.Read(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got["en0"].Signal)
	// A regular file without execute permission is rejected before exec.
	denied := filepath.Join(t.TempDir(), "denied")
	require.NoError(t, os.WriteFile(denied, []byte("not executable"), 0600))
	c, _ = NewClient(denied)
	got, err = c.Read(context.Background())
	require.ErrorIs(t, err, ErrExecutable)
	require.Nil(t, got)
	if os.Geteuid() != 0 {
		dir := t.TempDir()
		protected := filepath.Join(dir, "valid")
		require.NoError(t, os.Link(fixtureBinary, protected))
		require.NoError(t, os.Chmod(dir, 0))
		defer os.Chmod(dir, 0700)
		c, _ = NewClient(protected)
		got, err = c.Read(context.Background())
		require.ErrorIs(t, err, os.ErrPermission)
		require.Nil(t, got)
	}
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(fixtureBinary, link))
	c, _ = NewClient(link)
	_, err = c.Read(context.Background())
	require.ErrorIs(t, err, ErrExecutable)
}
func TestClientTimeoutCancellationAndRepeatedCleanup(t *testing.T) {
	before := runtime.NumGoroutine()
	c := fixtureClient(t, "timeout")
	for range 12 {
		start := time.Now()
		got, err := c.Read(context.Background())
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Nil(t, got)
		require.Less(t, time.Since(start), time.Second)
		requireReaped(t, waitPID(t, c.path+".pid"))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Read(ctx)
	require.ErrorIs(t, err, context.Canceled)
	ctx, cancel = context.WithCancel(context.Background())
	c.timeout = time.Second
	done := make(chan error, 1)
	go func() { _, err := c.Read(ctx); done <- err }()
	// Ensure this acquisition, rather than the previous timeout's PID, started.
	var pid int
	require.Eventually(t, func() bool {
		b, _ := os.ReadFile(c.path + ".pid")
		pid, _ = strconv.Atoi(string(b))
		return pid > 0 && syscall.Kill(pid, 0) == nil
	}, time.Second, 5*time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	requireReaped(t, pid)
	require.Eventually(t, func() bool { return runtime.NumGoroutine() <= before+2 }, time.Second, 10*time.Millisecond, "no goroutine abandoned per timeout")
}
func TestClientInheritedPipeCleanup(t *testing.T) {
	c := fixtureClient(t, "pipes")
	start := time.Now()
	got, err := c.Read(context.Background())
	require.Error(t, err)
	require.Nil(t, got)
	require.Less(t, time.Since(start), time.Second)
	requireReaped(t, waitPID(t, c.path+".pid"))
	descendant := waitPID(t, c.path+".desc.pid")
	// The Linux test process adopts deliberately orphaned fixture descendants
	// solely to reap them; production never makes itself a global subreaper.
	require.NoError(t, reapFixtureDescendant(descendant))
	requireReaped(t, descendant)
}
func TestClientSingleAcquisitionAndShutdown(t *testing.T) {
	c := fixtureClient(t, "timeout")
	c.timeout = time.Second
	first := make(chan error, 1)
	go func() { _, err := c.Read(context.Background()); first <- err }()
	pid := waitPID(t, c.path+".pid")
	var wg sync.WaitGroup
	for range 64 {
		wg.Go(func() {
			got, err := c.Read(context.Background())
			if !errors.Is(err, ErrBusy) || got != nil {
				t.Errorf("concurrent acquisition: %v %v", got, err)
			}
		})
	}
	wg.Wait()
	starts, err := os.ReadFile(c.path + ".starts")
	require.NoError(t, err)
	require.Len(t, strings.Fields(string(starts)), 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, c.Close(ctx))
	require.ErrorIs(t, <-first, context.Canceled)
	requireReaped(t, pid)
	_, err = c.Read(context.Background())
	require.ErrorIs(t, err, ErrClosed)
	require.NoError(t, c.Close(ctx))
}
func TestClientNoStaleResultAndRestrictedEnvironment(t *testing.T) {
	t.Setenv("SECRET_SENTINEL", "must not reach helper")
	t.Setenv("DYLD_INSERT_LIBRARIES", "must not reach helper")
	c := fixtureClient(t, "control")
	c.timeout = time.Second
	var firstSignal *float64
	for _, mode := range []string{"valid", "nonzero", "empty", "unavailable", "valid"} {
		require.NoError(t, os.WriteFile(c.path+".mode", []byte(mode), 0600))
		got, err := c.Read(context.Background())
		if mode == "valid" {
			require.NoError(t, err)
			require.Equal(t, -40.0, *got["en0"].Signal)
			if firstSignal == nil {
				firstSignal = got["en0"].Signal
			} else {
				require.NotSame(t, firstSignal, got["en0"].Signal)
			}
		} else {
			require.Nil(t, got)
		}
		if mode == "nonzero" || mode == "empty" {
			require.Error(t, err)
		}
	}
	require.Equal(t, -40.0, *firstSignal)
}

func TestNativeDiagnosticUnavailable(t *testing.T) {
	path := os.Getenv("BESZEL_WIFI_PROBE_TEST_PATH")
	if path == "" {
		t.Skip("separate native diagnostic/CF fixtures run on hosted macOS, not this host")
	}
	c, err := NewClient(path)
	require.NoError(t, err)
	got, err := c.Read(context.Background())
	require.NoError(t, err)
	require.Nil(t, got)
	require.NoError(t, c.Close(context.Background()))
}
