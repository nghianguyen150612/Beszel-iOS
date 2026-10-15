//go:build ios

package wifi

import (
	"context"

	"github.com/henrygd/beszel/internal/entities/system"
)

// collect deliberately reports unavailable until the private iOS 12 ABI,
// association freshness, authorization and bounded-call contract are verified.
// See docs/ios-wifi-provider-feasibility.md. Do not load MobileWiFi or infer dBm
// from a scaled value. This adapter has no native objects, workers or cache.
func collect(context.Context) map[string]system.WiFi { return nil }
