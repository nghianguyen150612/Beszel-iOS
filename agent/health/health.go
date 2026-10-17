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
var connectionFile = getConnectionFilePath()

func getHealthFilePath() string {
	filename := "beszel_health"
	if runtime.GOOS == "linux" {
		fullPath := filepath.Join("/dev/shm", filename)
		if err := updateHealthFile(fullPath); err == nil {
			return fullPath
		}
	}
	return filepath.Join(os.TempDir(), filename)
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
	return checkFile(healthFile, "unhealthy")
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
	return updateHealthFile(healthFile)
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

// CleanUp removes the health file
func CleanUp() error {
	return os.Remove(healthFile)
}

// CleanUpConnection removes the marker after the Agent loses its Hub connection.
func CleanUpConnection() error {
	err := os.Remove(connectionFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
