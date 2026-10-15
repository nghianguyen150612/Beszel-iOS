//go:build ios

package agent

import (
	"context"
	"github.com/shirou/gopsutil/v4/sensors"
)

// The macOS SMC/IOKit sensor collector is not an iOS adapter. Do not spawn
// timeout workers or present unmeasured temperatures on an unsupported source.
var getSensorTemps = func(context.Context) ([]sensors.TemperatureStat, error) { return nil, nil }
