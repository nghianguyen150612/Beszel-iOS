//go:build testing

package alerts_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stretchr/testify/require"
)

func TestP014SparseOptionalAlertHistory(t *testing.T) {
	for _, name := range []string{"Temperature", "Disk"} {
		t.Run(name, func(t *testing.T) {
			hub, user := beszelTests.GetHubWithUser(t)
			defer hub.Cleanup()
			systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "paused")
			require.NoError(t, err)
			record := systems[0]
			alert, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{"name": name, "system": record.Id, "user": user.Id, "value": 50, "min": 2, "triggered": true})
			require.NoError(t, err)
			require.True(t, alert.GetBool("triggered"))
			now := time.Now().UTC()
			reading := func(value float64) system.Stats {
				if name == "Temperature" {
					return system.Stats{Temperatures: map[string]float64{"sensor0": value}}
				}
				return system.Stats{ExtraFs: map[string]*system.FsStats{"disk0": {DiskTotal: 100, DiskUsed: value}}}
			}
			for _, sample := range []struct {
				age   time.Duration
				stats system.Stats
			}{
				{3 * time.Minute, system.Stats{}}, {90 * time.Second, reading(90)}, {60 * time.Second, system.Stats{}}, {0, reading(30)},
			} {
				history, err := beszelTests.CreateRecord(hub, "system_stats", map[string]any{"system": record.Id, "type": "1m", "stats": sample.stats, "created": now.Add(-sample.age).Format(types.DefaultDateLayout)})
				require.NoError(t, err)
				_, err = hub.DB().NewQuery("UPDATE system_stats SET created={:created} WHERE id={:id}").Bind(dbx.Params{"created": now.Add(-sample.age).Format(types.DefaultDateLayout), "id": history.Id}).Execute()
				require.NoError(t, err)
				history, err = hub.FindRecordById("system_stats", history.Id)
				require.NoError(t, err)
				require.WithinDuration(t, now.Add(-sample.age), history.GetDateTime("created").Time(), time.Millisecond)
			}
			data := &system.CombinedData{Stats: reading(30)}
			if name == "Temperature" {
				data.Info.DashboardTemp = 30
			}
			record.Set("updated", now)
			synctest.Test(t, func(t *testing.T) { require.NoError(t, hub.GetAlertManager().HandleSystemAlerts(record, data)) })
			saved, err := hub.FindRecordById("alerts", alert.Id)
			require.NoError(t, err)
			require.True(t, saved.GetBool("triggered"), "available readings average 60; missing devices must not dilute to 40 and recover")
		})
	}
}
