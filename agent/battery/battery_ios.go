//go:build ios

package battery

import (
	"os/exec"
	"strconv"
	"strings"

	"howett.net/plist"
)

type iosBattery struct {
	CurrentCapacity     int
	MaxCapacity         int
	AppleRawMaxCapacity int
	ExternalConnected   bool
	IsCharging          bool
	FullyCharged        bool
	BatteryInstalled    bool
}

func readIOSBatteries() ([]iosBattery, error) {
	out, err := exec.Command(
		"/usr/sbin/ioreg",
		"-r",
		"-c",
		"AppleARMPMUCharger",
		"-a",
	).Output()
	if err != nil {
		return nil, err
	}

	// ioreg -a doesn't guarantee that the matching registry properties
	// are the top-level plist object. Decode generically and walk the tree
	// looking for a dictionary containing battery capacity properties.
	var root any
	if _, err := plist.Unmarshal(out, &root); err != nil {
		return nil, err
	}

	var batteries []iosBattery
	findIOSBatteries(root, &batteries)

	if len(batteries) == 0 {
		return nil, errNoBatteries
	}

	return batteries, nil
}

func findIOSBatteries(v any, result *[]iosBattery) {
	switch value := v.(type) {
	case map[string]any:
		if bat, ok := decodeIOSBattery(value); ok {
			*result = append(*result, bat)
			return
		}

		for _, child := range value {
			findIOSBatteries(child, result)
		}

	case []any:
		for _, child := range value {
			findIOSBatteries(child, result)
		}
	}
}

func decodeIOSBattery(values map[string]any) (iosBattery, bool) {
	current, hasCurrent := plistInt(values["CurrentCapacity"])
	maxCapacity, hasMax := plistInt(values["MaxCapacity"])

	// These two properties identify the AppleARMPMUCharger battery record.
	if !hasCurrent || !hasMax || maxCapacity <= 0 {
		return iosBattery{}, false
	}

	rawMax, _ := plistInt(values["AppleRawMaxCapacity"])

	installed, hasInstalled := plistBool(values["BatteryInstalled"])
	if !hasInstalled {
		// Some iOS versions omit BatteryInstalled from the serialized
		// AppleARMPMUCharger plist even though capacity data is present.
		installed = true
	}

	external, _ := plistBool(values["ExternalConnected"])
	charging, _ := plistBool(values["IsCharging"])
	full, _ := plistBool(values["FullyCharged"])

	return iosBattery{
		CurrentCapacity:     current,
		MaxCapacity:         maxCapacity,
		AppleRawMaxCapacity: rawMax,
		ExternalConnected:   external,
		IsCharging:          charging,
		FullyCharged:        full,
		BatteryInstalled:    installed,
	}, true
}

func plistInt(v any) (int, bool) {
	switch value := v.(type) {
	case int:
		return value, true
	case int8:
		return int(value), true
	case int16:
		return int(value), true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case uint:
		return int(value), true
	case uint8:
		return int(value), true
	case uint16:
		return int(value), true
	case uint32:
		return int(value), true
	case uint64:
		return int(value), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(value))
		return n, err == nil
	default:
		return 0, false
	}
}

func plistBool(v any) (bool, bool) {
	if value, ok := v.(bool); ok {
		return value, true
	}

	if n, ok := plistInt(v); ok {
		return n != 0, true
	}

	if value, ok := v.(string); ok {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "yes", "true", "on":
			return true, true
		case "no", "false", "off":
			return false, true
		}
	}

	return false, false
}

func HasReadableBattery() bool {
	batteries, _ := GetBatteryStats()
	return len(batteries) > 0
}

func GetBatteryStats() ([]Battery, error) {
	batteries, err := readIOSBatteries()
	if err != nil {
		return nil, err
	}

	result := make([]Battery, 0, len(batteries))

	for _, bat := range batteries {
		if !bat.BatteryInstalled || bat.MaxCapacity <= 0 {
			continue
		}

		percent := bat.CurrentCapacity * 100 / bat.MaxCapacity
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}

		state := stateUnknown
		switch {
		case !bat.ExternalConnected:
			state = stateDischarging
		case bat.IsCharging:
			state = stateCharging
		case percent == 0:
			state = stateEmpty
		case bat.FullyCharged || percent >= 100:
			state = stateFull
		default:
			state = stateIdle
		}

		battery := Battery{
			Name:    "Primary",
			Percent: uint8(percent),
			State:   state,
			System:  true,
		}

		if bat.AppleRawMaxCapacity > 0 {
			battery.FullChargeCapacity = uint64(bat.AppleRawMaxCapacity)
			battery.HasFullChargeCapacity = true
		}

		result = append(result, battery)
	}

	if len(result) == 0 {
		return nil, errNoBatteries
	}

	return normalizeBatteries(result), nil
}
