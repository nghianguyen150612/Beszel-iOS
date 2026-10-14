package battery

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func iosFixture(properties string) []byte {
	return []byte("+-o AppleARMPMUCharger <class AppleARMPMUCharger>\n{\n" + properties + "\n}\n")
}
func TestIOSBatteryHistoricalFixtures(t *testing.T) {
	for _, name := range []string{"recursive.ioreg", "legacy.ioreg"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/ios/" + name)
			require.NoError(t, err)
			bats, err := parseIOSBatteryOutput(data)
			require.NoError(t, err)
			require.Len(t, bats, 1)
			assert.Equal(t, uint8(42), bats[0].Percent)
			assert.Equal(t, stateCharging, bats[0].State)
			assert.Equal(t, "Primary", bats[0].Name)
			assert.Equal(t, uint64(3989), bats[0].FullChargeCapacity)
		})
	}
}
func TestIOSBatteryParserEdges(t *testing.T) {
	base := `"CurrentCapacity" = 42
"MaxCapacity" = 100`
	for _, tt := range []struct {
		name, props    string
		percent, state uint8
		bad            bool
	}{
		{"unknown charging", base, 42, stateUnknown, false},
		{"missing current", `"MaxCapacity" = 100`, 0, 0, true},
		{"missing maximum", `"CurrentCapacity" = 42`, 0, 0, true},
		{"zero maximum", `"CurrentCapacity" = 42
"MaxCapacity" = 0`, 0, 0, true},
		{"negative current", strings.Replace(base, "42", "-1", 1), 0, 0, true},
		{"negative maximum", strings.Replace(base, "100", "-1", 1), 0, 0, true},
		{"malformed", strings.Replace(base, "42", "abc", 1), 0, 0, true},
		{"uint64 overflow", strings.Replace(base, "42", "18446744073709551616", 1), 0, 0, true},
		{"large capacity", `"CurrentCapacity" = 18446744073709551614
"MaxCapacity" = 18446744073709551615`, 99, stateUnknown, false},
		{"exceeds maximum", strings.Replace(base, "42", "101", 1), 100, stateUnknown, false},
		{"not installed", base + "\n\"BatteryInstalled\" = No", 0, 0, true},
		{"invalid installed", base + "\n\"BatteryInstalled\" = maybe", 0, 0, true},
		{"invalid external", base + "\n\"ExternalConnected\" = maybe", 42, stateUnknown, false},
		{"disconnected", base + "\n\"ExternalConnected\" = No", 42, stateDischarging, false},
		{"charging", base + "\n\"ExternalConnected\" = Yes\n\"IsCharging\" = Yes", 42, stateCharging, false},
		{"idle", base + "\n\"ExternalConnected\" = true\n\"IsCharging\" = 0\n\"FullyCharged\" = false", 42, stateIdle, false},
		{"missing full", base + "\n\"ExternalConnected\" = Yes\n\"IsCharging\" = No", 42, stateUnknown, false},
		{"invalid charging", base + "\n\"ExternalConnected\" = Yes\n\"IsCharging\" = maybe", 42, stateUnknown, false},
		{"full", base + "\n\"FullyCharged\" = Yes", 42, stateFull, false},
		{"100 idle", strings.Replace(base, "42", "100", 1) + "\n\"ExternalConnected\" = Yes\n\"IsCharging\" = No", 100, stateFull, false},
		{"empty connected", strings.Replace(base, "42", "0", 1) + "\n\"ExternalConnected\" = Yes\n\"IsCharging\" = No", 0, stateEmpty, false},
		{"empty discharging", strings.Replace(base, "42", "0", 1) + "\n\"ExternalConnected\" = No", 0, stateDischarging, false},
		{"duplicate", base + "\n\"CurrentCapacity\" = 1", 0, 0, true},
		{"nested multiline", base + "\n\"BatteryData\" = {\n\"CurrentCapacity\" = 1\n\"MaxCapacity\" = 2\n}", 42, stateUnknown, false},
		{"nested inline", `"BatteryData" = {"CurrentCapacity"=1,"MaxCapacity"=2}` + "\n" + base, 42, stateUnknown, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bats, err := parseIOSBatteryOutput(iosFixture(tt.props))
			if tt.bad {
				require.Error(t, err)
				require.Empty(t, bats)
				return
			}
			require.NoError(t, err)
			require.Len(t, bats, 1)
			assert.Equal(t, tt.percent, bats[0].Percent)
			assert.Equal(t, tt.state, bats[0].State)
		})
	}
	for _, tt := range []struct {
		name string
		out  []byte
	}{
		{"truncated", iosFixture(base)[:len(iosFixture(base))-2]},
		{"multiple blocks", append(iosFixture(base), iosFixture(base)...)},
		{"empty", nil},
		{"oversized", []byte(strings.Repeat("x", iosBatteryOutputLimit+1))},
		{"unexpected registry", iosFixture(`"Other" = 42`)},
		{"wrong registry class", []byte(strings.ReplaceAll(string(iosFixture(base)), "AppleARMPMUCharger", "OtherCharger"))},
	} {
		t.Run(tt.name, func(t *testing.T) { _, err := parseIOSBatteryOutput(tt.out); require.Error(t, err) })
	}
}
func TestIOSBatteryCacheFailureRecovery(t *testing.T) {
	var collector iosBatteryCollector
	now := time.Unix(100, 0)
	calls := 0
	fail := false
	run := func(ctx context.Context) ([]byte, error) {
		calls++
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		assert.LessOrEqual(t, time.Until(deadline), iosBatteryTimeout)
		if fail {
			return nil, errors.New("fixture failure")
		}
		return iosFixture("\"CurrentCapacity\" = 42\n\"MaxCapacity\" = 100"), nil
	}
	first, err := collector.collect(now, run)
	require.NoError(t, err)
	first[0].Percent = 0
	cached, err := collector.collect(now.Add(time.Second), run)
	require.NoError(t, err)
	assert.Equal(t, uint8(42), cached[0].Percent)
	assert.Equal(t, 1, calls)
	fail = true
	bats, err := collector.collect(now.Add(time.Minute), run)
	require.Error(t, err)
	assert.Empty(t, bats)
	_, err = collector.collect(now.Add(61*time.Second), run)
	require.Error(t, err)
	assert.Equal(t, 2, calls)
	fail = false
	bats, err = collector.collect(now.Add(2*time.Minute), run)
	require.NoError(t, err)
	assert.Len(t, bats, 1)
	// Parallel realtime requests reuse one acquisition.
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _, _ = collector.collect(now.Add(121*time.Second), run) })
	}
	wg.Wait()
	assert.Equal(t, 3, calls)
}
func TestIOSBatteryCommandHelper(t *testing.T) {
	if os.Getenv("BESZEL_BATTERY_TEST_HELPER") != "1" {
		return
	}
	switch os.Getenv("BESZEL_BATTERY_TEST_ACTION") {
	case "hang":
		time.Sleep(time.Minute)
	case "fail":
		os.Exit(3)
	case "oversized":
		os.Stdout.Write([]byte(strings.Repeat("x", iosBatteryOutputLimit+1)))
	case "success":
		os.Stdout.Write(iosFixture("\"CurrentCapacity\" = 42\n\"MaxCapacity\" = 100"))
	}
	os.Exit(0)
}
func TestIOSBatterySubprocessDeadlineAndRecovery(t *testing.T) {
	for _, action := range []string{"hang", "fail", "oversized", "success"} {
		t.Run(action, func(t *testing.T) {
			budget := iosBatteryTimeout
			if action == "hang" {
				budget = 50 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(context.Background(), budget)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestIOSBatteryCommandHelper$")
			cmd.Env = append(os.Environ(), "BESZEL_BATTERY_TEST_HELPER=1", "BESZEL_BATTERY_TEST_ACTION="+action)
			start := time.Now()
			out, err := runIOSBatteryCommand(cmd)
			if action == "success" {
				require.NoError(t, err)
				_, err = parseIOSBatteryOutput(out)
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Empty(t, out)
			}
			if action == "hang" {
				assert.Less(t, time.Since(start), time.Second)
				assert.NotNil(t, cmd.ProcessState, "subprocess must be waited/reaped")
			}
		})
	}
}
