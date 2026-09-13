//go:build ios

package battery

import (
	"os/exec"

	"howett.net/plist"
)

type iosBattery struct {
	CurrentCapacity     int  `plist:"CurrentCapacity"`
	MaxCapacity         int  `plist:"MaxCapacity"`
	AppleRawMaxCapacity int  `plist:"AppleRawMaxCapacity"`
	ExternalConnected   bool `plist:"ExternalConnected"`
	IsCharging          bool `plist:"IsCharging"`
	FullyCharged        bool `plist:"FullyCharged"`
	BatteryInstalled    bool `plist:"BatteryInstalled"`
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

	var batteries []iosBattery
	if _, err := plist.Unmarshal(out, &batteries); err != nil {
		return nil, err
	}
	return batteries, nil
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
