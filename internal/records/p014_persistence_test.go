//go:build testing

package records_test

import (
	"testing"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/records"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	"github.com/henrygd/beszel/internal/testutil/telemetry"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestP014PersistedOptionalRollupAndRestart(t *testing.T) {
	hub, err := beszelTests.NewTestHub(t.TempDir())
	require.NoError(t, err)
	defer hub.Cleanup()
	user, err := beszelTests.CreateUser(hub, "p014-rollup@example.com", "p014-synthetic-password")
	require.NoError(t, err)
	sys, err := beszelTests.CreateRecord(hub, "systems", map[string]any{"users": []string{user.Id}, "name": "p014-rollup", "host": "127.0.0.1", "port": "45876", "status": "up"})
	require.NoError(t, err)
	collection, err := hub.FindCachedCollectionByNameOrId("system_stats")
	require.NoError(t, err)
	for i := range 9 {
		stats := system.Stats{Cpu: 10}
		if i < 2 {
			f, err := telemetry.Load("linux")
			require.NoError(t, err)
			stats = f.Data.Stats
			stats.Cpu = 10
			if i == 1 {
				gpu := stats.GPUData["gpu0"]
				gpu.Usage = 60
				stats.GPUData["gpu0"] = gpu
			}
		}
		row := core.NewRecord(collection)
		row.Load(map[string]any{"system": sys.Id, "type": "1m", "stats": stats, "created": time.Now().UTC().Add(-time.Duration(i+1) * time.Minute)})
		require.NoError(t, hub.SaveNoValidate(row))
	}
	records.NewRecordManager(hub).CreateLongerRecords()
	rows, err := hub.FindAllRecords("system_stats", dbx.HashExp{"system": sys.Id, "type": "10m"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	id := rows[0].Id
	verify := func() {
		row, err := hub.FindRecordById("system_stats", id)
		require.NoError(t, err)
		var got system.Stats
		require.NoError(t, row.UnmarshalJSONField("stats", &got))
		require.Equal(t, 10.0, got.Cpu)
		require.Equal(t, 42.5, got.Temperatures["sensor1"])
		require.Equal(t, 30.0, got.GPUData["gpu0"].Usage)
		require.Equal(t, 7.5, got.GPUData["gpu0"].PowerPkg)
		require.Contains(t, got.Temperatures, "sensor0")
		require.Zero(t, got.Temperatures["sensor0"])
	}
	verify()
	// Close and reopen the same disposable database. No live-file copy is used.
	require.NoError(t, hub.TestApp.ResetBootstrapState())
	require.NoError(t, hub.TestApp.Bootstrap())
	verify()
	// A disconnected system must not gain synthetic system history from rollups.
	sys.Set("status", "down")
	require.NoError(t, hub.SaveNoValidate(sys))
	before, err := hub.CountRecords("system_stats")
	require.NoError(t, err)
	records.NewRecordManager(hub).CreateLongerRecords()
	after, err := hub.CountRecords("system_stats")
	require.NoError(t, err)
	require.Equal(t, before, after)
}
