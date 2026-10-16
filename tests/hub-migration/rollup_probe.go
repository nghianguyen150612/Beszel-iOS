// Command rollup_probe exercises v0.21 migrations immediately before the real
// record rollup and retention managers on a stopped, disposable Hub database.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/henrygd/beszel/internal/entities/container"
	"github.com/henrygd/beszel/internal/entities/system"
	_ "github.com/henrygd/beszel/internal/migrations"
	"github.com/henrygd/beszel/internal/records"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func fail(err error) {
	fmt.Fprintln(os.Stderr, "P013 rollup probe:", err)
	os.Exit(1)
}

func migrationCount(app core.App) (int64, error) {
	var count int64
	err := app.DB().NewQuery("SELECT COUNT(*) FROM _migrations").Row(&count)
	return count, err
}

func deleteTestSystem(app core.App, systemID string) error {
	for _, collection := range []string{
		"network_monitor_stats", "network_monitors", "container_stats", "system_stats",
	} {
		records, err := app.FindAllRecords(collection, dbx.HashExp{"system": systemID})
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				return err
			}
		}
	}
	record, err := app.FindFirstRecordByData("systems", "id", systemID)
	if err != nil {
		return err
	}
	return app.Delete(record)
}

func create(app core.App, collection string, data map[string]any) (*core.Record, error) {
	coll, err := app.FindCachedCollectionByNameOrId(collection)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(coll)
	for key, value := range data {
		if key == "created" {
			record.SetRaw(key, value)
		} else {
			record.Set(key, value)
		}
	}
	if err := app.SaveNoValidate(record); err != nil {
		return nil, err
	}
	return record, nil
}

type evidence struct {
	MigrationLedgerCountBefore      int64           `json:"migration_ledger_count_before"`
	MigrationLedgerCountAfter       int64           `json:"migration_ledger_count_after"`
	MigrationsUnchanged             bool            `json:"migrations_unchanged"`
	RollupImmediatelyAfterMigration rollupEvidence  `json:"rollup_immediately_after_migration"`
	RetentionCleanup                cleanupEvidence `json:"retention_cleanup"`
}

type rollupEvidence struct {
	SystemStats10mCount      int64   `json:"system_stats_10m_count"`
	SystemStats10mCPU        float64 `json:"system_stats_10m_cpu"`
	ContainerStats10mCount   int64   `json:"container_stats_10m_count"`
	ContainerStats10mCPU     float64 `json:"container_stats_10m_cpu"`
	ContainerStats10mMem     float64 `json:"container_stats_10m_mem"`
	MonitorStats10mCount     int64   `json:"monitor_stats_10m_count"`
	MonitorStatsTotalCount   int64   `json:"monitor_stats_total_count"`
	MonitorStatsSuccessCount int64   `json:"monitor_stats_success_count"`
	MonitorStatsResponseSum  int64   `json:"monitor_stats_response_sum"`
	MonitorStatsResponseMin  float64 `json:"monitor_stats_response_min"`
	MonitorStatsResponseMax  float64 `json:"monitor_stats_response_max"`
}

type cleanupEvidence struct {
	ExpiredSystemSamplesDeleted    bool `json:"expired_system_samples_deleted"`
	ExpiredContainerSamplesDeleted bool `json:"expired_container_samples_deleted"`
	ExpiredMonitorSamplesDeleted   bool `json:"expired_monitor_samples_deleted"`
	RepeatedCleanupStable          bool `json:"repeated_cleanup_stable"`
	TestRowsRemoved                bool `json:"test_rows_removed"`
}

func run(dataDir string) (evidence, error) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir, HideStartBanner: true}).App
	if err := app.Bootstrap(); err != nil {
		return evidence{}, fmt.Errorf("bootstrap migrated database: %w", err)
	}
	defer func() {
		if err := app.ClearBootstrap(); err != nil {
			fmt.Fprintln(os.Stderr, "P013 rollup probe close:", err)
		}
	}()

	before, err := migrationCount(app)
	if err != nil {
		return evidence{}, fmt.Errorf("read migration ledger: %w", err)
	}
	if err := app.RunAllMigrations(); err != nil {
		return evidence{}, fmt.Errorf("rerun migrations immediately before rollup: %w", err)
	}
	after, err := migrationCount(app)
	if err != nil {
		return evidence{}, fmt.Errorf("read migration ledger after startup: %w", err)
	}
	if before != after {
		return evidence{}, fmt.Errorf("migration ledger changed before rollup: %d to %d", before, after)
	}

	systemRecord, err := create(app, "systems", map[string]any{
		"name": "p013-rollup-probe", "host": "127.0.0.1", "port": "45876", "status": "up",
	})
	if err != nil {
		return evidence{}, fmt.Errorf("create isolated rollup system: %w", err)
	}
	systemID := systemRecord.Id
	cleaned := false
	defer func() {
		if !cleaned {
			if err := deleteTestSystem(app, systemID); err != nil {
				fmt.Fprintln(os.Stderr, "P013 rollup probe cleanup:", err)
			}
		}
	}()

	monitorRecord, err := create(app, "network_monitors", map[string]any{
		"system": systemID, "target": "http://p013-rollup.synthetic.invalid", "protocol": "http",
		"port": 80, "interval": 60, "enabled": true,
	})
	if err != nil {
		return evidence{}, fmt.Errorf("create rollup monitor: %w", err)
	}
	monitorID := monitorRecord.Id
	now := time.Now().UTC()
	for i := 1; i <= 9; i++ {
		createdAt := now.Add(-time.Duration(i) * time.Minute)
		if _, err := create(app, "system_stats", map[string]any{
			"system": systemID, "type": "1m", "created": createdAt.Format(types.DefaultDateLayout),
			"stats": system.Stats{Cpu: float64(i), Mem: float64(i * 100)},
		}); err != nil {
			return evidence{}, fmt.Errorf("seed system rollup sample %d: %w", i, err)
		}
		if _, err := create(app, "container_stats", map[string]any{
			"system": systemID, "type": "1m", "created": createdAt.Format(types.DefaultDateLayout),
			"stats": []container.Stats{{Name: "p013-rollup", Cpu: float64(i), Mem: float64(i * 100)}},
		}); err != nil {
			return evidence{}, fmt.Errorf("seed container rollup sample %d: %w", i, err)
		}
		if _, err := create(app, "network_monitor_stats", map[string]any{
			"system": systemID, "monitor": monitorID, "type": "1m", "created": createdAt.UnixMilli(),
			"res_min": 12, "res_max": 24, "total_count": 4, "success_count": 3, "res_sum": 54,
		}); err != nil {
			return evidence{}, fmt.Errorf("seed monitor rollup sample %d: %w", i, err)
		}
	}
	manager := records.NewRecordManager(app)
	manager.CreateLongerRecords()
	rollup := rollupEvidence{}
	for collection, field := range map[string]string{
		"system_stats": "system", "container_stats": "system", "network_monitor_stats": "monitor",
	} {
		filter := dbx.HashExp{field: systemID, "type": "10m"}
		if collection == "network_monitor_stats" {
			filter[field] = monitorID
		}
		count, err := app.CountRecords(collection, filter)
		if err != nil {
			return evidence{}, fmt.Errorf("count %s 10m rollup: %w", collection, err)
		}
		switch collection {
		case "system_stats":
			rollup.SystemStats10mCount = count
			records, err := app.FindAllRecords(collection, filter)
			if err != nil || len(records) != 1 {
				return evidence{}, fmt.Errorf("read system 10m rollup: count=%d err=%v", len(records), err)
			}
			var result system.Stats
			if err := records[0].UnmarshalJSONField("stats", &result); err != nil {
				return evidence{}, fmt.Errorf("decode system 10m rollup: %w", err)
			}
			rollup.SystemStats10mCPU = result.Cpu
			if result.Cpu != 5 {
				return evidence{}, fmt.Errorf("system 10m CPU average = %v, want 5", result.Cpu)
			}
		case "container_stats":
			rollup.ContainerStats10mCount = count
			records, err := app.FindAllRecords(collection, filter)
			if err != nil || len(records) != 1 {
				return evidence{}, fmt.Errorf("read container 10m rollup: count=%d err=%v", len(records), err)
			}
			var result []container.Stats
			if err := records[0].UnmarshalJSONField("stats", &result); err != nil || len(result) != 1 {
				return evidence{}, fmt.Errorf("decode container 10m rollup: count=%d err=%v", len(result), err)
			}
			rollup.ContainerStats10mCPU = result[0].Cpu
			rollup.ContainerStats10mMem = result[0].Mem
			if result[0].Cpu != 5 || result[0].Mem != 500 {
				return evidence{}, fmt.Errorf("container 10m rollup = cpu %v/mem %v, want 5/500", result[0].Cpu, result[0].Mem)
			}
		case "network_monitor_stats":
			rollup.MonitorStats10mCount = count
			records, err := app.FindAllRecords(collection, filter)
			if err != nil || len(records) != 1 {
				return evidence{}, fmt.Errorf("read monitor 10m rollup: count=%d err=%v", len(records), err)
			}
			rollup.MonitorStatsTotalCount = int64(records[0].GetInt("total_count"))
			rollup.MonitorStatsSuccessCount = int64(records[0].GetInt("success_count"))
			rollup.MonitorStatsResponseSum = int64(records[0].GetInt("res_sum"))
			rollup.MonitorStatsResponseMin = records[0].GetFloat("res_min")
			rollup.MonitorStatsResponseMax = records[0].GetFloat("res_max")
			if rollup.MonitorStatsTotalCount != 36 || rollup.MonitorStatsSuccessCount != 27 || rollup.MonitorStatsResponseSum != 486 || rollup.MonitorStatsResponseMin != 12 || rollup.MonitorStatsResponseMax != 24 {
				return evidence{}, fmt.Errorf("monitor 10m rollup invariant failed: %+v", rollup)
			}
		}
		if count != 1 {
			return evidence{}, fmt.Errorf("%s 10m rollup count = %d, want 1", collection, count)
		}
	}

	// Add expired rows for all three metric families, run the real scheduled job
	// twice, and verify that deleting aged synthetic rows is stable/idempotent.
	expiredAt := now.Add(-2 * time.Hour)
	ids := map[string]string{}
	for collection, fields := range map[string]map[string]any{
		"system_stats":          {"system": systemID, "type": "1m", "created": expiredAt.Format(types.DefaultDateLayout), "stats": system.Stats{Cpu: 99}},
		"container_stats":       {"system": systemID, "type": "1m", "created": expiredAt.Format(types.DefaultDateLayout), "stats": []container.Stats{{Name: "p013-expired", Cpu: 99}}},
		"network_monitor_stats": {"system": systemID, "monitor": monitorID, "type": "1m", "created": expiredAt.UnixMilli(), "total_count": 1, "success_count": 1, "res_sum": 99, "res_min": 99, "res_max": 99},
	} {
		record, err := create(app, collection, fields)
		if err != nil {
			return evidence{}, fmt.Errorf("seed expired %s sample: %w", collection, err)
		}
		ids[collection] = record.Id
	}
	manager.DeleteOldRecords()
	afterFirstCleanup, err := app.CountRecords("system_stats", dbx.HashExp{"system": systemID, "type": "10m"})
	if err != nil || afterFirstCleanup != 1 {
		return evidence{}, fmt.Errorf("first cleanup changed valid rollup: count=%d err=%v", afterFirstCleanup, err)
	}
	for collection, id := range ids {
		if _, err := app.FindRecordById(collection, id); err == nil {
			return evidence{}, fmt.Errorf("first cleanup retained expired %s row", collection)
		}
	}
	countsAfterFirst := make(map[string]int64)
	for _, collection := range []string{"system_stats", "container_stats", "network_monitor_stats"} {
		countsAfterFirst[collection], err = app.CountRecords(collection, dbx.HashExp{"system": systemID})
		if err != nil {
			return evidence{}, err
		}
	}
	manager.DeleteOldRecords()
	cleanup := cleanupEvidence{
		ExpiredSystemSamplesDeleted: true, ExpiredContainerSamplesDeleted: true,
		ExpiredMonitorSamplesDeleted: true, RepeatedCleanupStable: true,
	}
	for collection, firstCount := range countsAfterFirst {
		count, err := app.CountRecords(collection, dbx.HashExp{"system": systemID})
		if err != nil || count != firstCount {
			return evidence{}, fmt.Errorf("repeated cleanup changed %s count from %d to %d (err=%v)", collection, firstCount, count, err)
		}
	}
	if err := deleteTestSystem(app, systemID); err != nil {
		return evidence{}, fmt.Errorf("remove synthetic rollup system: %w", err)
	}
	cleaned = true
	cleanup.TestRowsRemoved = true
	return evidence{
		MigrationLedgerCountBefore:      before,
		MigrationLedgerCountAfter:       after,
		MigrationsUnchanged:             before == after,
		RollupImmediatelyAfterMigration: rollup,
		RetentionCleanup:                cleanup,
	}, nil
}

func main() {
	dataDir := flag.String("data-dir", "", "stopped disposable v0.21 Hub data directory")
	flag.Parse()
	if *dataDir == "" {
		fail(errors.New("--data-dir is required"))
	}
	result, err := run(*dataDir)
	if err != nil {
		fail(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		fail(err)
	}
	fmt.Println(string(encoded))
}
