// Package health tracks Agent process liveness separately from authenticated
// Hub connectivity. The connection marker is removed when every Hub transport
// disconnects and expires if the connection manager stops refreshing it.
package health

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// healthFile is the path to the health file
var healthFile = getHealthFilePath()
var healthFiles = getHealthFilePaths("/dev/shm")
var connectionFile = getConnectionFilePath()

func getHealthFilePath() string {
	return getHealthFilePaths("/dev/shm")[0]
}

// getHealthFilePaths lists the liveness marker locations in write preference order.
// Resolving paths is read-only so `agent health` cannot refresh its own marker.
func getHealthFilePaths(shmDir string) []string {
	const filename = "beszel_health"
	if dir := os.Getenv("BESZEL_HEALTH_DIR"); dir != "" {
		return []string{filepath.Join(dir, filename)}
	}
	tmpPath := filepath.Join(os.TempDir(), filename)
	if runtime.GOOS == "linux" {
		return []string{filepath.Join(shmDir, filename), tmpPath}
	}
	return []string{tmpPath}
}

func getConnectionFilePath() string {
	if runtime.GOOS == "ios" || runtime.GOOS == "darwin" {
		// LaunchDaemons and the root CLI must share the same iOS temp path even
		// when their inherited TMPDIR environments differ.
		return filepath.Join("/tmp", "beszel_connected")
	}
	return filepath.Join(os.TempDir(), "beszel_connected")
}

func updateHealthFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	return file.Close()
}

// Check checks whether the Agent event loop has refreshed its liveness marker recently.
func Check() error {
	var newest time.Time
	var lastError error
	for _, path := range healthFiles {
		info, err := os.Stat(path)
		if err != nil {
			lastError = err
			continue
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	if newest.IsZero() {
		return lastError
	}
	if time.Since(newest) > 91*time.Second {
		log.Println("over 90 seconds since last connection")
		return errors.New("unhealthy")
	}
	return nil
}

// CheckConnection reports whether the connection manager has a recent
// authenticated WebSocket or SSH connection to the Hub.
func CheckConnection() error {
	info, err := os.Lstat(connectionFile)
	if err != nil {
		return err
	}
	if !connectionMarkerTrusted(info) {
		return errors.New("untrusted Hub connection marker")
	}
	return checkFile(connectionFile, "not connected to the Hub")
}

func checkFile(path, staleMessage string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}
	if time.Since(fileInfo.ModTime()) > 91*time.Second {
		log.Println("over 90 seconds since last connection")
		return errors.New(staleMessage)
	}
	return nil
}

// Update updates the modification time of the health file
func Update() error {
	var err error
	for _, path := range healthFiles {
		if err = updateHealthFile(path); err == nil {
			return nil
		}
	}
	return err
}

// UpdateConnection records a successful authenticated Hub connection.
func UpdateConnection() error {
	if err := os.Remove(connectionFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := os.OpenFile(connectionFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}

// CleanUp removes the health file; another agent may already have removed it.
func CleanUp() error {
	for _, path := range healthFiles {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// CleanUpConnection removes the marker after the Agent loses its Hub connection.
func CleanUpConnection() error {
	err := os.Remove(connectionFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
