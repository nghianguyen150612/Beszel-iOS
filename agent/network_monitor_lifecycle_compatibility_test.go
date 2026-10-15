//go:build testing

package agent

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/stretchr/testify/require"
)

func TestP010MonitorHistoryIdentity(t *testing.T) {
	original := monitor.Config{ID: "stable", Protocol: "http", Target: "https://fixture", Interval: 30}
	task := newMonitorTask(original)
	defer task.cancel()
	task.history.record(monitorSample{responseUs: 42, timestamp: time.Now()})
	for _, field := range []string{"interval", "target", "protocol", "tls"} {
		cfg := original
		switch field {
		case "interval":
			cfg.Interval = 60
		case "target":
			cfg.Target += "/new"
		case "protocol":
			cfg.Protocol = "icmp"
		case "tls":
			cfg.SkipTLSVerify = true
		}
		replacement := newMonitorTaskFromExisting(cfg, task)
		defer replacement.cancel()
		result, ok := replacement.history.result(time.Minute, time.Now())
		if field == "interval" {
			require.True(t, ok)
			require.Equal(t, int64(42), result.AvgResponse)
		} else {
			require.False(t, ok, field)
		}
	}
	for _, protocol := range []string{"tcp", "dns"} {
		a := monitor.Config{Protocol: protocol, Target: "fixture", Port: 80, Server: "127.0.0.1"}
		b := a
		if protocol == "tcp" {
			b.Port = 443
		} else {
			b.Server = "::1"
		}
		require.False(t, sameMonitorObservation(a, b))
	}
}

func TestP010DeleteDuringCertificateWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		certStarted, release := make(chan struct{}), make(chan struct{})
		pm := newMonitorManagerWithProbe(func(context.Context, monitor.Config) (int64, error) { return 42, nil })
		defer pm.Stop()
		pm.certCheck = func(context.Context, string) (monitor.CertInfo, error) {
			close(certStarted)
			<-release
			return monitor.CertInfo{Expires: 123}, nil
		}
		cfg := monitor.Config{ID: "deleted", Protocol: "http", Target: "https://fixture", Interval: 3600}
		result := make(chan *monitor.Result, 1)
		go func() { r, _ := pm.UpsertMonitor(cfg, true); result <- r }()
		<-certStarted
		synctest.Wait()
		// Health probe is already complete, but RunNow is waiting for certificate I/O.
		pm.DeleteMonitor(cfg.ID)
		close(release)
		synctest.Wait()
		require.Nil(t, <-result, "deleted monitor must not publish while waiting for certificate")
		require.Empty(t, pm.GetResults(1000))
	})
}

func TestP010CancellationPublicationAndSync(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		for _, action := range []string{"delete", "replace", "stop"} {
			release := make(chan struct{})
			pm := newMonitorManagerWithProbe(func(context.Context, monitor.Config) (int64, error) { <-release; return 42, nil })
			cfg := monitor.Config{ID: "stable", Interval: 3600}
			_, err := pm.UpsertMonitor(cfg, false)
			require.NoError(t, err)
			task := pm.monitors[cfg.ID]
			result := make(chan *monitor.Result, 1)
			go func() { r, _ := pm.UpsertMonitor(cfg, true); result <- r }()
			synctest.Wait()
			switch action {
			case "delete":
				pm.DeleteMonitor(cfg.ID)
			case "stop":
				pm.Stop()
			case "replace":
				cfg.Interval--
				pm.SyncMonitors([]monitor.Config{cfg})
			}
			close(release)
			synctest.Wait()
			require.Nil(t, <-result)
			require.Empty(t, task.history.samples)
			if action == "replace" {
				replacement := pm.monitors[cfg.ID]
				require.Empty(t, replacement.history.samples)
				for range 20 {
					pm.SyncMonitors([]monitor.Config{cfg, cfg})
					require.Same(t, replacement, pm.monitors[cfg.ID])
				}
			}
			pm.SyncMonitors(nil)
			require.Empty(t, pm.monitors)
			pm.Stop()
			synctest.Wait()
		}
	})
}

func TestP010OutOfOrderHistory(t *testing.T) {
	h := newMonitorHistory()
	now := time.Now()
	h.record(monitorSample{responseUs: 30, timestamp: now})
	h.record(monitorSample{responseUs: -1, timestamp: now.Add(-time.Second)})
	r, ok := h.result(time.Minute, now)
	require.True(t, ok)
	require.Equal(t, now.UnixMilli(), r.LastProbeAt)
	require.Equal(t, 50.0, r.PacketLoss)
	require.Equal(t, int64(30), r.AvgResponse)
	// Older sample collides with the current ring slot; newest bucket survives.
	h.record(monitorSample{responseUs: 999, timestamp: now.Add(-61 * time.Minute)})
	r, ok = h.result(time.Hour, now)
	require.True(t, ok)
	require.Equal(t, int64(30), r.AvgResponse1h)
	require.Equal(t, int64(3), r.SampleCount)
	require.Len(t, h.samples, 2)
}
