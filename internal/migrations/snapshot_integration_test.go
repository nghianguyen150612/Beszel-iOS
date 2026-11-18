package migrations_test

import (
	"testing"

	_ "github.com/henrygd/beszel/internal/migrations"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

// This is a disposable current-schema migration test, not a 0.19 restore test.
func TestSnapshotMigrationsRoundTrip(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	monitorCollection, err := app.FindCollectionByNameOrId("network_monitors")
	require.NoError(t, err)
	require.IsType(t, &core.BoolField{}, monitorCollection.Fields.GetByName("skipTlsVerify"))
	poolCollection, err := app.FindCollectionByNameOrId("zfs_pools")
	require.NoError(t, err)
	require.IsType(t, &core.TextField{}, poolCollection.Fields.GetByName("status"))
	monitor := core.NewRecord(monitorCollection)
	monitor.Set("target", "https://synthetic.example")
	require.False(t, monitor.GetBool("skipTlsVerify"), "migration default must verify TLS")
	monitor.Set("skipTlsVerify", true)
	require.NoError(t, app.SaveNoValidate(monitor))
	pool := core.NewRecord(poolCollection)
	pool.Set("name", "synthetic-pool")
	pool.Set("status", "synthetic advisory")
	require.NoError(t, app.SaveNoValidate(pool))
	// Select the two P006 migrations without changing the global registry.
	// initial-settings.go has no Down function and must retain its ledger entry.
	selected := core.MigrationsList{}
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == "1790924727_zfs_pool_status.go" || migration.File == "1790964708_network_monitor_skip_tls_verify.go" {
			selected.Add(migration)
		}
	}
	require.Len(t, selected.Items(), 2)
	runner := core.NewMigrationsRunner(app, selected)
	applied, err := runner.Up()
	require.NoError(t, err)
	require.Empty(t, applied, "fresh database already applied registered migrations")
	// Inspect the last three ledger entries; only the selected two are reverted.
	reverted, err := runner.Down(3)
	require.NoError(t, err)
	require.Equal(t, []string{"1790964708_network_monitor_skip_tls_verify.go", "1790924727_zfs_pool_status.go"}, reverted)
	monitorCollection, err = app.FindCollectionByNameOrId("network_monitors")
	require.NoError(t, err)
	require.Nil(t, monitorCollection.Fields.GetByName("skipTlsVerify"))
	poolCollection, err = app.FindCollectionByNameOrId("zfs_pools")
	require.NoError(t, err)
	require.Nil(t, poolCollection.Fields.GetByName("status"))
	applied, err = runner.Up()
	require.NoError(t, err)
	require.Equal(t, []string{"1790924727_zfs_pool_status.go", "1790964708_network_monitor_skip_tls_verify.go"}, applied)
	monitor, err = app.FindRecordById("network_monitors", monitor.Id)
	require.NoError(t, err)
	require.Equal(t, "https://synthetic.example", monitor.GetString("target"))
	require.False(t, monitor.GetBool("skipTlsVerify"), "down/up loses the new field; it is not a full restore")
	pool, err = app.FindRecordById("zfs_pools", pool.Id)
	require.NoError(t, err)
	require.Equal(t, "synthetic-pool", pool.GetString("name"))
	require.Empty(t, pool.GetString("status"))
	applied, err = runner.Up()
	require.NoError(t, err)
	require.Empty(t, applied, "second application must be idempotent")
	require.NoError(t, app.RunAllMigrations(), "full ledger remains applied")
}

func TestSnapshotGenericDownIsNotRestore(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	runner := core.NewMigrationsRunner(app, core.AppMigrations)
	reverted, err := runner.Down(3)
	require.NoError(t, err)
	require.Equal(t, "initial-settings.go", reverted[0])
	_, err = runner.Up()
	// initial-settings has no Down action; replaying it collides with the existing
	// superuser. Generic down/up is unsafe recovery; a full backup is mandatory.
	require.ErrorContains(t, err, "initial-settings.go")
	require.ErrorContains(t, err, "unique")
}
