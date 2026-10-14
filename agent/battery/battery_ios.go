//go:build ios

package battery

import "time"

var iosBatteries iosBatteryCollector

func HasReadableBattery() bool {
	batteries, _ := GetBatteryStats()
	return len(batteries) > 0
}

func GetBatteryStats() ([]Battery, error) {
	return iosBatteries.collect(time.Now(), runIOSIoreg)
}
