package migrations_test

import (
	"path/filepath"
	"testing"

	"ganigeorgiev.com/pocketbase/dbx"
	"ganigeorgiev.com/pocketbase/pocketbase/core"
	pbtests "ganigeorgiev.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestSQLiteReadersKeepSnapshotAcrossConcurrentCommit(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	for _, tc := range []struct {
		name   string
		writer dbx.Builder
	}{
		{"data.db", app.NonconcurrentDB()},
		{"auxiliary.db", app.AuxNonconcurrentDB()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := core.DefaultDBConnect(filepath.Join(app.DataDir(), tc.name))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, reader.Close()) })

			var mode string
			require.NoError(t, reader.NewQuery("PRAGMA journal_mode").Row(&mode))
			require.Equal(t, "wal", mode)
			_, err = tc.writer.NewQuery("CREATE TABLE p030_wal_probe (id INTEGER PRIMARY KEY)").Execute()
			require.NoError(t, err)

			tx, err := reader.Begin()
			require.NoError(t, err)
			defer tx.Rollback()

			var count int
			const countRows = "SELECT COUNT(*) FROM p030_wal_probe"
			require.NoError(t, tx.NewQuery(countRows).Row(&count))
			require.Zero(t, count)

			_, err = tc.writer.NewQuery("INSERT INTO p030_wal_probe VALUES (1)").Execute()
			require.NoError(t, err, "a concurrent reader must not block a WAL write")
			require.NoError(t, tx.NewQuery(countRows).Row(&count))
			require.Zero(t, count, "the active read transaction must keep its original snapshot")
			require.NoError(t, tx.Commit())

			require.NoError(t, reader.NewQuery(countRows).Row(&count))
			require.Equal(t, 1, count)
		})
	}
}
