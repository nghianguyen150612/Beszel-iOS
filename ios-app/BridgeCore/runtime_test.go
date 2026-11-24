package bridgecore

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

func TestNativeMetricsAvailabilityAndUnits(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
	}{
		{`{"physicalMemory":8589934592,"batteryPercent":0,"batteryState":"unplugged","volumeCapacity":100,"availableToApp":0}`, true},
		{`{"physicalMemory":8589934592,"batteryState":"unknown"}`, true},
		{`{"physicalMemory":18446744073709551616,"batteryState":"unknown"}`, false},
		{`{"physicalMemory":-1,"batteryState":"unknown"}`, false},
		{`{"batteryPercent":101,"batteryState":"charging"}`, false},
		{`{"batteryPercent":-1,"batteryState":"unknown"}`, false},
		{`{"batteryState":"made-up"}`, false},
		{`{"batteryState":"unknown","volumeCapacity":10,"availableToApp":11}`, false},
	}
	for _, tt := range tests {
		if err := UpdateMetrics(tt.input); (err == nil) != tt.ok {
			t.Errorf("input %s: %v", tt.input, err)
		}
	}
	if err := UpdateMetrics(`{"physicalMemory":8589934592,"batteryPercent":0,"batteryState":"unknown"}`); err != nil {
		t.Fatal(err)
	}
	var status struct {
		Metrics Snapshot `json:"metrics"`
	}
	if err := json.Unmarshal([]byte(StatusJSON()), &status); err != nil {
		t.Fatal(err)
	}
	if status.Metrics.PhysicalMemory != 8*1024*1024*1024 || status.Metrics.BatteryPercent == nil || *status.Metrics.BatteryPercent != 0 || status.Metrics.VolumeCapacity != nil {
		t.Fatal(status)
	}
	if strings.Contains(StatusJSON(), "processMemory") {
		t.Fatal("process memory mislabeled")
	}
	if err := UpdateMetrics(`{"batteryState":"unknown"}`); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal([]byte(StatusJSON()), &status)
	if status.Metrics.BatteryPercent != nil {
		t.Fatal("unknown became zero")
	}
}
func TestInvalidBridgeConfigurationRecovery(t *testing.T) {
	Stop()
	for _, input := range []string{"{", `{}`, `{"hubURL":"http://localhost","token":"SECRET"}`} {
		if Start(input) == nil {
			t.Fatal("invalid config accepted")
		}
	}
	if strings.Contains(StatusJSON(), "SECRET") {
		t.Fatal("credential leaked")
	}
	Stop()
	Stop()
}

func TestConcurrentNativeProbeSnapshots(t *testing.T) {
	const available = `{"batteryState":"unplugged","probes":{"batteryPercent":{"available":true,"scope":"device battery","unit":"percent","sampledAt":"2026-11-24T08:00:00Z","value":0}}}`
	const unavailable = `{"batteryState":"unknown","probes":{"batteryPercent":{"available":false,"scope":"device battery","unit":"percent","sampledAt":"2026-11-24T08:00:01Z","reason":"unknown"}}}`

	for _, invalid := range []string{
		strings.Replace(available, `"value":0`, `"value":null`, 1),
		strings.Replace(unavailable, `"reason":"unknown"`, `"reason":"unknown","value":0`, 1),
		strings.Replace(unavailable, `"sampledAt":"2026-11-24T08:00:01Z",`, "", 1),
	} {
		if err := UpdateMetrics(invalid); err == nil {
			t.Errorf("accepted inconsistent probe: %s", invalid)
		}
	}

	if err := UpdateMetrics(available); err != nil {
		t.Fatal(err)
	}
	var updates sync.WaitGroup
	for _, payload := range []string{available, unavailable} {
		updates.Go(func() {
			for range 100 {
				if err := UpdateMetrics(payload); err != nil {
					t.Error(err)
				}
			}
		})
	}
	for range 100 {
		var status struct {
			Metrics Snapshot `json:"metrics"`
		}
		if err := json.Unmarshal([]byte(StatusJSON()), &status); err != nil {
			t.Fatal(err)
		}
		reading := status.Metrics.Probes["batteryPercent"]
		if reading.Available {
			if string(reading.Value) != "0" || status.Metrics.BatteryState != "unplugged" {
				t.Error("inconsistent available snapshot", status)
			}
		} else if reading.Reason != "unknown" || status.Metrics.BatteryState != "unknown" {
			t.Error("inconsistent unavailable snapshot", status)
		}
	}
	updates.Wait()
}
