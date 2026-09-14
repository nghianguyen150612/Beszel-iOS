//go:build ios

package battery

import (
	"os/exec"
	"strconv"
	"strings"
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
		"-l",
	).Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")

	current, okCurrent := ioregInt(lines, "CurrentCapacity")
	maxCapacity, okMax := ioregInt(lines, "MaxCapacity")

	if !okCurrent || !okMax || maxCapacity <= 0 {
		return nil, errNoBatteries
	}

	rawMax, _ := ioregInt(lines, "AppleRawMaxCapacity")

	installed, okInstalled := ioregBool(lines, "BatteryInstalled")
	if !okInstalled {
		installed = true
	}

	external, _ := ioregBool(lines, "ExternalConnected")
	charging, _ := ioregBool(lines, "IsCharging")
	full, _ := ioregBool(lines, "FullyCharged")

	return []iosBattery{
		{
			CurrentCapacity:     current,
			MaxCapacity:         maxCapacity,
			AppleRawMaxCapacity: rawMax,
			ExternalConnected:   external,
			IsCharging:          charging,
			FullyCharged:        full,
			BatteryInstalled:    installed,
		},
	}, nil
}

func ioregValue(lines []string, key string) (string, bool) {
	prefix := `"` + key + `" = `

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Legacy ioreg output prefixes registry properties with "|".
		if strings.HasPrefix(line, "|") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "|"))
		}

		// Only match a top-level property line. This intentionally avoids
		// matching keys embedded inside the large BatteryData dictionary.
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		return value, true
	}

	return "", false
}

func ioregInt(lines []string, key string) (int, bool) {
	value, ok := ioregValue(lines, key)
	if !ok {
		return 0, false
	}

	value = strings.Trim(value, `"`)
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}

	return n, true
}

func ioregBool(lines []string, key string) (bool, bool) {
	value, ok := ioregValue(lines, key)
	if !ok {
		return false, false
	}

	switch strings.ToLower(strings.Trim(value, `"`)) {
	case "yes", "true", "1":
		return true, true
	case "no", "false", "0":
		return false, true
	default:
		return false, false
	}
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

		b := Battery{
			Name:    "Primary",
			Percent: uint8(percent),
			State:   state,
			System:  true,
		}

		if bat.AppleRawMaxCapacity > 0 {
			b.FullChargeCapacity = uint64(bat.AppleRawMaxCapacity)
			b.HasFullChargeCapacity = true
		}

		result = append(result, b)
	}

	if len(result) == 0 {
		return nil, errNoBatteries
	}

	return normalizeBatteries(result), nil
}
