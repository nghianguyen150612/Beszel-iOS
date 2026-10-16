//go:build testing

package alerts_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	"github.com/henrygd/beszel/internal/testutil/telemetry"
	"github.com/stretchr/testify/require"
)

func TestP014RemoteMetricAlerts(t *testing.T) {
	for _, fixture := range telemetry.All() {
		t.Run(fixture.Name, func(t *testing.T) {
			h, user := beszelTests.GetHubWithUser(t)
			defer h.Cleanup()
			records, err := beszelTests.CreateSystems(h, 1, user.Id, "paused")
			require.NoError(t, err)
			r := records[0]
			r.Set("updated", time.Now().UTC())
			// TestApp installs a fake mailer; no real notification destination is contacted.
			for _, metric := range []string{"CPU", "Memory", "Disk", "Bandwidth", "Battery", "GPU", "Temperature"} {
				t.Run(metric, func(t *testing.T) {
					rule, err := beszelTests.CreateRecord(h, "alerts", map[string]any{"name": metric, "system": r.Id, "user": user.Id, "value": 1, "min": 1})
					require.NoError(t, err)
					data := fixture.Data
					if metric == "Battery" {
						rule.Set("value", 80)
						require.NoError(t, h.Save(rule))
					}
					if metric == "Bandwidth" {
						rule.Set("value", 0.001)
						require.NoError(t, h.Save(rule))
					}
					synctest.Test(t, func(t *testing.T) { require.NoError(t, h.GetAlertManager().HandleSystemAlerts(r, &data)) })
					saved, err := h.FindRecordById("alerts", rule.Id)
					require.NoError(t, err)
					available := true
					switch metric {
					case "GPU":
						available = len(data.Stats.GPUData) > 0 && data.Info.GpuPct > 1
					case "Temperature":
						available = data.Info.DashboardTemp > 1
					case "Battery":
						available = data.Stats.Battery[0] > 0
					}
					require.Equal(t, available, saved.GetBool("triggered"))
					require.NoError(t, h.Delete(saved))
				})
			}
		})
	}
}

func TestP014MissingGPUDoesNotRecoverAlert(t *testing.T) {
	for _, minutes := range []int{1, 2} {
		t.Run(time.Duration(minutes).String(), func(t *testing.T) {
			h, user := beszelTests.GetHubWithUser(t)
			defer h.Cleanup()
			records, err := beszelTests.CreateSystems(h, 1, user.Id, "paused")
			require.NoError(t, err)
			r := records[0]
			r.Set("updated", time.Now().UTC())
			rule, err := beszelTests.CreateRecord(h, "alerts", map[string]any{"name": "GPU", "system": r.Id, "user": user.Id, "value": 50, "min": minutes, "triggered": true})
			require.NoError(t, err)
			evaluate := func(data system.CombinedData) {
				t.Helper()
				synctest.Test(t, func(t *testing.T) { require.NoError(t, h.GetAlertManager().HandleSystemAlerts(r, &data)) })
			}
			// Populate the averaging window, including optional absence. Current missing
			// acquisition must not be read as an idle GPU and recover an active alert.
			for i := range 4 {
				_, err := beszelTests.CreateRecord(h, "system_stats", map[string]any{"system": r.Id, "type": "1m", "created": time.Now().UTC().Add(-time.Duration(i) * time.Minute), "stats": system.Stats{}})
				require.NoError(t, err)
			}
			evaluate(system.CombinedData{})
			saved, err := h.FindRecordById("alerts", rule.Id)
			require.NoError(t, err)
			require.True(t, saved.GetBool("triggered"), "missing GPU is unknown, not measured recovery")
			if minutes == 1 {
				// A reported idle device is a legitimate zero and must recover.
				evaluate(system.CombinedData{Stats: system.Stats{GPUData: map[string]system.GPUData{"0": {Name: "remote GPU", Usage: 0}}}})
				saved, err = h.FindRecordById("alerts", rule.Id)
				require.NoError(t, err)
				require.False(t, saved.GetBool("triggered"))
			}
		})
	}
}
