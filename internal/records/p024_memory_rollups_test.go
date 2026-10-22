//go:build testing

package records_test

import (
	"testing"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/records"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stretchr/testify/require"
)

func TestP024MemoryCompatibilityAcrossSystemRollups(t *testing.T) {
	hub, err := beszelTests.NewTestHub(t.TempDir())
	require.NoError(t, err)
	defer hub.Cleanup()

	user, err := beszelTests.CreateUser(hub, "p024-memory-rollups@example.com", "p024-synthetic-password")
	require.NoError(t, err)
	sys, err := beszelTests.CreateRecord(hub, "systems", map[string]any{
		"users": []string{user.Id}, "name": "p024-memory-rollups",
		"host": "127.0.0.1", "port": "45876", "status": "up",
	})
	require.NoError(t, err)
	collection, err := hub.FindCachedCollectionByNameOrId("system_stats")
	require.NoError(t, err)

	const (
		mib      = uint64(1 << 20)
		memTotal = 128 * mib
		memUsed  = 64 * mib
		memCache = 32 * mib
		maxUsed  = 96 * mib
		swapSize = 64 * mib
	)
	newStats := func(legacy bool) system.Stats {
		if legacy {
			return system.Stats{
				Cpu: 10, Mem: 0.125, MemUsed: 0.0625, MemBuffCache: 0.03125,
				MaxMem: 0.09375, Swap: 0.0625, SwapUsed: 0,
			}
		}
		return system.Stats{
			Cpu:         10,
			MemBytes:    [4]uint64{memTotal, memUsed, memCache, 0},
			MaxMemBytes: maxUsed,
			SwapBytes:   [2]uint64{swapSize, 0},
		}
	}
	add := func(recordType string, created time.Time, stats system.Stats) {
		row := core.NewRecord(collection)
		row.Load(map[string]any{
			"system": sys.Id, "type": recordType, "stats": stats, "created": created,
		})
		require.NoError(t, hub.SaveNoValidate(row))
		_, err := hub.DB().NewQuery("UPDATE system_stats SET created={:created} WHERE id={:id}").
			Bind(dbx.Params{
				"created": created.UTC().Format(types.DefaultDateLayout),
				"id":      row.Id,
			}).Execute()
		require.NoError(t, err)
	}

	now := time.Now().UTC()
	// Alternating legacy GB and exact byte records represent identical low
	// memory values. The source format must not change through any rollup tier.
	for i := range 9 {
		add("1m", now.Add(-time.Duration(i+1)*time.Minute), newStats(i%2 == 0))
	}
	add("10m", now.Add(-15*time.Minute), newStats(true))
	for i := range 5 {
		add("20m", now.Add(-time.Duration(30+15*i)*time.Minute), newStats(i%2 == 0))
	}
	for i := range 3 {
		add("120m", now.Add(-time.Duration(150+60*i)*time.Minute), newStats(i%2 == 0))
	}

	records.NewRecordManager(hub).CreateLongerRecords()

	expectedRows := map[string]int{"10m": 2, "20m": 6, "120m": 4, "480m": 1}
	for _, recordType := range []string{"10m", "20m", "120m", "480m"} {
		t.Run(recordType, func(t *testing.T) {
			rows, err := hub.FindAllRecords("system_stats", dbx.HashExp{
				"system": sys.Id, "type": recordType,
			})
			require.NoError(t, err)
			require.Len(t, rows, expectedRows[recordType])
			var generated []*core.Record
			for _, row := range rows {
				if row.GetDateTime("created").Time().After(now.Add(-5 * time.Minute)) {
					generated = append(generated, row)
				}
			}
			require.Len(t, generated, 1)
			for _, row := range generated {
				var got system.Stats
				require.NoError(t, row.UnmarshalJSONField("stats", &got))
				require.Equal(t, [4]uint64{memTotal, memUsed, memCache, 0}, got.MemBytes)
				require.Equal(t, maxUsed, got.MaxMemBytes)
				require.Equal(t, [2]uint64{swapSize, 0}, got.SwapBytes)
				require.Equal(t, 10.0, got.Cpu)
				// The deprecated GB fields remain populated for older integrations.
				require.Equal(t, 0.13, got.Mem)
				require.Equal(t, 0.06, got.MemUsed)
				require.Equal(t, 0.06, got.Swap)
			}
		})
	}

	// Reopening the disposable Hub must leave persisted byte fields intact.
	require.NoError(t, hub.TestApp.ResetBootstrapState())
	require.NoError(t, hub.TestApp.Bootstrap())
	rows, err := hub.FindAllRecords("system_stats", dbx.HashExp{
		"system": sys.Id, "type": "480m",
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	var reopened system.Stats
	require.NoError(t, rows[0].UnmarshalJSONField("stats", &reopened))
	require.Equal(t, [4]uint64{memTotal, memUsed, memCache, 0}, reopened.MemBytes)
	require.Equal(t, [2]uint64{swapSize, 0}, reopened.SwapBytes)
}
