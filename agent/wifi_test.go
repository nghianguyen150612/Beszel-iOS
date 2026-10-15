package agent

import (
	"testing"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/require"
)

// Simulated acquisition results exercise the actual production polling method.
// They establish no native iOS association or permission behavior.
func TestP012WiFiPollingLifecycle(t *testing.T) {
	signal := -40.0
	a := &Agent{}
	calls := 0
	var next map[string]system.WiFi
	acquire := func() map[string]system.WiFi { calls++; return next }
	for _, step := range []struct {
		name     string
		snapshot map[string]system.WiFi
	}{
		{"connected", map[string]system.WiFi{"en0": {SSID: "home", Signal: &signal}}},
		{"disconnected", nil},
		{"reconnected", map[string]system.WiFi{"en0": {SSID: "new", Signal: &signal}}},
		{"disabled", map[string]system.WiFi{}},
		{"permission denied", nil}, {"temporarily unavailable", nil},
		{"recovery hidden SSID", map[string]system.WiFi{"en0": {Signal: &signal}}},
		{"renamed and multiple", map[string]system.WiFi{"en2": {SSID: "same", Signal: &signal}, "en3": {SSID: "same"}}},
		{"failure after success", nil}, {"repeated failure", nil},
		{"recovered SSID only", map[string]system.WiFi{"en0": {SSID: "网络 café"}}},
	} {
		t.Run(step.name, func(t *testing.T) {
			next = step.snapshot
			// Start with old output to verify missing signals clear it too.
			stats := system.Stats{WiFi: map[string]int8{"old": -50}}
			before := calls
			a.updateWiFiStats(defaultDataCacheTimeMs, &stats, acquire)
			require.Equal(t, before+1, calls)
			require.Equal(t, next, a.systemInfo.WiFi)
			require.NotContains(t, stats.WiFi, "old")
			require.Len(t, stats.WiFi, countFixtureSignals(step.snapshot))
			for id, reading := range step.snapshot {
				if reading.Signal != nil {
					require.Equal(t, int8(-40), stats.WiFi[id])
				}
			}
			// A realtime request must not acquire again, even if the source
			// has changed since the default poll. It is a reused snapshot.
			next = map[string]system.WiFi{"unexpected": {SSID: "not acquired"}}
			for range 5 {
				a.updateWiFiStats(1000, &stats, acquire)
				require.Equal(t, before+1, calls)
				require.Equal(t, step.snapshot, a.systemInfo.WiFi)
			}
		})
	}
}

func countFixtureSignals(snapshot map[string]system.WiFi) int {
	n := 0
	for _, v := range snapshot {
		if v.Signal != nil {
			n++
		}
	}
	return n
}
