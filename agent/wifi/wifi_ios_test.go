//go:build ios

package wifi

import (
	"context"
	"sync"
	"testing"
)

// Also run on cloud hosts with an explicit file list: Go then compiles the
// production iOS adapter unchanged without selecting the host implementation.
// These test unavailable behavior, not simulated MobileWiFi authorization.
func TestP012IOSUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, ctx := range []context.Context{context.Background(), ctx} {
		for range 1000 {
			if snapshot := collect(ctx); snapshot != nil || Signals(snapshot) != nil {
				t.Fatal("unavailable provider emitted telemetry", snapshot)
			}
		}
	}
}

func TestP012IOSConcurrentUnavailable(t *testing.T) {
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 100 {
				if got := Collect(); got != nil {
					t.Error("unavailable provider emitted telemetry", got)
				}
			}
		})
	}
	wg.Wait()
}
