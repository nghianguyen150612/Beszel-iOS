// Package bridgecore exposes small, platform-neutral pieces of the existing
// Beszel codebase to the stock-iOS native bridge.
package bridgecore

import (
	"runtime"

	"github.com/henrygd/beszel"
)

// Version is the actual Beszel version compiled into the native library.
func Version() string { return beszel.Version }

// Platform identifies the Go target. It is diagnostic metadata, not a system metric.
func Platform() string { return runtime.GOOS + "/" + runtime.GOARCH }
