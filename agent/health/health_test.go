//go:build testing

package health

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	// Override both marker paths to use temporary files for this test.
	tmpDir := t.TempDir()
	restorePaths := SetPathsForTesting(filepath.Join(tmpDir, "beszel_health_test"), filepath.Join(tmpDir, "beszel_connected_test"))
	defer restorePaths()

	t.Run("check with no health file", func(t *testing.T) {
		err := Check()
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err), "expected a file-not-exist error, but got: %v", err)
	})

	t.Run("update and check", func(t *testing.T) {
		err := Update()
		require.NoError(t, err, "Update() failed")

		err = Check()
		assert.NoError(t, err, "Check() failed immediately after Update()")
	})

	t.Run("authenticated connection marker lifecycle", func(t *testing.T) {
		require.Error(t, CheckConnection(), "connection check must fail before authentication")
		require.NoError(t, UpdateConnection())
		require.NoError(t, CheckConnection())
		info, err := os.Stat(connectionFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		stale := time.Now().Add(-92 * time.Second)
		require.NoError(t, os.Chtimes(connectionFile, stale, stale))
		require.Error(t, CheckConnection(), "a stale connection marker must fail")
		require.NoError(t, UpdateConnection())
		require.NoError(t, CleanUpConnection())
		require.Error(t, CheckConnection(), "disconnect removes the connection marker")
	})

	t.Run("connection marker rejects symlinks", func(t *testing.T) {
		target := filepath.Join(tmpDir, "marker-target")
		require.NoError(t, os.WriteFile(target, []byte("x"), 0o600))
		require.NoError(t, os.Symlink(target, connectionFile))
		require.Error(t, CheckConnection(), "symlink marker must fail closed")
		require.NoError(t, UpdateConnection(), "update replaces the symlink itself with a new private marker")
		require.NoError(t, CheckConnection())
		contents, err := os.ReadFile(target)
		require.NoError(t, err)
		assert.Equal(t, []byte("x"), contents, "marker update must not touch the symlink target")
	})

	// This test uses synctest to simulate time passing.
	t.Run("check with simulated time", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			// Update the file to set the initial timestamp.
			require.NoError(t, Update(), "Update() failed inside synctest")

			// Set the mtime to the current fake time to align the file's timestamp with the simulated clock.
			now := time.Now()
			require.NoError(t, os.Chtimes(healthFile, now, now), "Chtimes failed")

			// Wait a duration less than the threshold.
			time.Sleep(89 * time.Second)
			synctest.Wait()

			// The check should still pass.
			assert.NoError(t, Check(), "Check() failed after 89s")

			// Wait for the total duration to exceed the threshold.
			time.Sleep(5 * time.Second)
			synctest.Wait()

			// The check should now fail as unhealthy.
			err := Check()
			require.Error(t, err, "Check() should have failed after 91s")
			assert.Equal(t, "unhealthy", err.Error(), "Check() returned wrong error")
		})
	})
}
