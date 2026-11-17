//go:build testing

package health

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestHealthDirectoryOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BESZEL_HEALTH_DIR", dir)
	assert.Equal(t, filepath.Join(dir, "beszel_health"), getHealthFilePath())
}

func TestGetHealthFilePathsDoesNotWrite(t *testing.T) {
	shmDir := t.TempDir()
	t.Setenv("BESZEL_HEALTH_DIR", "")
	paths := getHealthFilePaths(shmDir)
	if runtime.GOOS == "linux" {
		assert.Equal(t, filepath.Join(shmDir, "beszel_health"), paths[0])
	}
	assert.NoFileExists(t, filepath.Join(shmDir, "beszel_health"))
}

func TestCheckDoesNotTouchHealthFile(t *testing.T) {
	healthFile := filepath.Join(t.TempDir(), "beszel_health")
	restorePaths := SetPathsForTesting(healthFile, filepath.Join(t.TempDir(), "beszel_connected"))
	t.Cleanup(restorePaths)
	require.NoError(t, Update())
	stale := time.Now().Add(-2 * time.Minute)
	require.NoError(t, os.Chtimes(healthFile, stale, stale))

	err := Check()
	require.Error(t, err)
	assert.Equal(t, "unhealthy", err.Error())
	info, err := os.Stat(healthFile)
	require.NoError(t, err)
	assert.True(t, info.ModTime().Equal(stale), "Check changed the health marker's mtime")
}

func TestCleanUpIsIdempotent(t *testing.T) {
	restorePaths := SetPathsForTesting(filepath.Join(t.TempDir(), "health"), filepath.Join(t.TempDir(), "connected"))
	t.Cleanup(restorePaths)
	require.NoError(t, Update())
	require.NoError(t, CleanUp())
	require.NoError(t, CleanUp())
}

func TestHealthFileFallback(t *testing.T) {
	dir := t.TempDir()
	unwritable := filepath.Join(dir, "missing-directory", "beszel_health")
	fallback := filepath.Join(dir, "beszel_health")
	restorePaths := SetHealthPathsForTesting([]string{unwritable, fallback}, filepath.Join(dir, "connected"))
	t.Cleanup(restorePaths)

	require.NoError(t, Update())
	assert.NoFileExists(t, unwritable)
	assert.FileExists(t, fallback)
	assert.NoError(t, Check())
}

func TestCheckUsesNewestHealthMarker(t *testing.T) {
	dir := t.TempDir()
	stalePath := filepath.Join(dir, "shm_beszel_health")
	freshPath := filepath.Join(dir, "tmp_beszel_health")
	restorePaths := SetHealthPathsForTesting([]string{stalePath, freshPath}, filepath.Join(dir, "connected"))
	t.Cleanup(restorePaths)

	require.NoError(t, os.WriteFile(stalePath, nil, 0o600))
	stale := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(stalePath, stale, stale))
	require.NoError(t, os.WriteFile(freshPath, nil, 0o600))
	assert.NoError(t, Check())
}
