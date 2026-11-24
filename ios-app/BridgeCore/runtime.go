package bridgecore

import (
	"encoding/json"
	"errors"
	"runtime"
	"sync"
	"time"

	"github.com/henrygd/beszel/agent"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
)

type Configuration struct {
	HubURL        string `json:"hubURL"`
	Token         string `json:"token"`
	HubKey        string `json:"hubKey"`
	DataDirectory string `json:"dataDirectory"`
	Model         string `json:"model"`
	OSVersion     string `json:"osVersion"`
	Name          string `json:"name"`
}

// Snapshot preserves availability independently from a valid zero measurement.
// Volume values describe only the application container's accessible volume.
type Snapshot struct {
	PhysicalMemory uint64                  `json:"physicalMemory,omitempty"`
	BatteryPercent *float64                `json:"batteryPercent"`
	BatteryState   string                  `json:"batteryState"`
	VolumeCapacity *uint64                 `json:"volumeCapacity"`
	AvailableToApp *uint64                 `json:"availableToApp"`
	SampledAt      time.Time               `json:"sampledAt"`
	Probes         map[string]ProbeReading `json:"probes,omitempty"`
}

// ProbeReading is local diagnostics only. A raw JSON value preserves large
// integer counters, strings and tick arrays without conversion to float64.
type ProbeReading struct {
	Available bool            `json:"available"`
	Scope     string          `json:"scope"`
	Unit      string          `json:"unit"`
	SampledAt time.Time       `json:"sampledAt"`
	Value     json.RawMessage `json:"value,omitempty"`
	Reason    string          `json:"reason,omitempty"`
}

var native struct {
	runtime agent.EmbeddedRuntime
	sync.Mutex
	snapshot Snapshot
}

func Start(configuration string) error {
	var c Configuration
	if json.Unmarshal([]byte(configuration), &c) != nil {
		return errors.New("invalid configuration JSON")
	}
	return native.runtime.Start(agent.EmbeddedConfig{HubURL: c.HubURL, Token: c.Token, HubKey: c.HubKey, DataDirectory: c.DataDirectory,
		Details: system.Details{Hostname: c.Name, CpuModel: c.Model, Os: system.Darwin, OsName: "iOS " + c.OSVersion, Arch: runtime.GOARCH},
		Collect: func(common.DataRequestOptions) (*system.CombinedData, error) {
			// The current Hub cannot represent a partial stock system snapshot.
			// Preserve the native sample locally until an approved protocol solution exists.
			return nil, errors.New("required whole-system metrics unavailable")
		},
	})
}
func Stop() { native.runtime.Stop() }
func UpdateMetrics(payload string) error {
	var s Snapshot
	if json.Unmarshal([]byte(payload), &s) != nil {
		return errors.New("invalid native metrics JSON")
	}
	if s.BatteryPercent != nil && (*s.BatteryPercent < 0 || *s.BatteryPercent > 100) {
		return errors.New("battery outside percent range")
	}
	switch s.BatteryState {
	case "unknown", "charging", "full", "unplugged":
	default:
		return errors.New("invalid battery state")
	}
	if s.VolumeCapacity != nil && s.AvailableToApp != nil && *s.AvailableToApp > *s.VolumeCapacity {
		return errors.New("invalid volume capacity")
	}
	for _, reading := range s.Probes {
		hasValue := len(reading.Value) != 0 && string(reading.Value) != "null"
		if reading.Scope == "" || reading.Unit == "" || reading.SampledAt.IsZero() ||
			(reading.Available && (!hasValue || reading.Reason != "")) ||
			(!reading.Available && (hasValue || reading.Reason == "")) {
			return errors.New("invalid native probe reading")
		}
	}
	native.Lock()
	native.snapshot = s
	native.Unlock()
	return nil
}
func StatusJSON() string {
	native.Lock()
	s := native.snapshot
	native.Unlock()
	b, _ := json.Marshal(struct {
		agent.RuntimeStatus
		GoVersion  string   `json:"goVersion"`
		Metrics    Snapshot `json:"metrics"`
		Limitation string   `json:"limitation"`
	}{native.runtime.Status(), runtime.Version(), s, "Hub stats blocked: current Hub requires a complete system snapshot"})
	return string(b)
}
