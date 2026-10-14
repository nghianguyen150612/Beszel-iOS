package battery

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"math/bits"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const iosBatteryTimeout = 2 * time.Second
const iosBatteryCacheInterval = 60 * time.Second
const iosBatteryOutputLimit = 1 << 20

// A fixed command, no shell, bounded output and deadline. Output is never logged.
func runIOSIoreg(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/sbin/ioreg", "-r", "-c", "AppleARMPMUCharger", "-l")
	return runIOSBatteryCommand(cmd)
}

type limitedBatteryOutput struct{ buffer bytes.Buffer }

func (b *limitedBatteryOutput) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > iosBatteryOutputLimit {
		return 0, errors.New("battery output exceeds limit")
	}
	return b.buffer.Write(p)
}
func runIOSBatteryCommand(cmd *exec.Cmd) ([]byte, error) {
	var out limitedBatteryOutput
	cmd.Stdout = &out
	// Bound waits on inherited pipes as well as the direct process. CommandContext
	// kills and Wait reaps ioreg on deadline; no shell or persistent goroutine.
	cmd.WaitDelay = 250 * time.Millisecond
	if err := cmd.Run(); err != nil {
		return nil, errors.New("battery command failed or timed out")
	}
	return out.buffer.Bytes(), nil
}

type iosBatteryCollector struct {
	sync.Mutex
	at        time.Time
	batteries []Battery
	err       error
}

func (c *iosBatteryCollector) collect(now time.Time, run func(context.Context) ([]byte, error)) ([]Battery, error) {
	c.Lock()
	defer c.Unlock()
	if !c.at.IsZero() && now.Sub(c.at) >= 0 && now.Sub(c.at) < iosBatteryCacheInterval {
		return append([]Battery(nil), c.batteries...), c.err
	}
	ctx, cancel := context.WithTimeout(context.Background(), iosBatteryTimeout)
	defer cancel()
	out, err := run(ctx)
	if err == nil {
		c.batteries, err = parseIOSBatteryOutput(out)
	} else {
		c.batteries = nil
	}
	c.at, c.err = now, err
	return append([]Battery(nil), c.batteries...), err
}

// Parse only direct properties of a complete registry block. Nested BatteryData
// and child registry entries cannot supply or overwrite the charger's values.
func parseIOSBatteryOutput(out []byte) ([]Battery, error) {
	if len(out) > iosBatteryOutputLimit {
		return nil, errNoBatteries
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 4096), iosBatteryOutputLimit)
	depth := 0
	props := map[string]string{}
	var candidates []Battery
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimLeft(scanner.Text(), " |"))
		if depth == 1 && strings.HasPrefix(line, `"`) {
			key, value, found := strings.Cut(line, " = ")
			key = strings.Trim(key, `"`)
			if found && iosBatteryProperty(key) {
				if _, duplicate := props[key]; duplicate {
					return nil, errNoBatteries
				}
				props[key] = strings.TrimSpace(value)
			}
		}
		// Braces inside quoted strings do not alter dictionary/registry depth.
		quoted, escaped := false, false
		for _, ch := range line {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && quoted {
				escaped = true
				continue
			}
			if ch == '"' {
				quoted = !quoted
				continue
			}
			if quoted {
				continue
			}
			if ch == '{' {
				depth++
			}
			if ch == '}' {
				depth--
				if depth < 0 {
					return nil, errNoBatteries
				}
				if depth == 0 {
					if _, ok := props["CurrentCapacity"]; ok {
						b, err := iosBatteryFromProperties(props)
						if err != nil {
							return nil, err
						}
						candidates = append(candidates, b)
					}
					props = map[string]string{}
				}
			}
		}
	}
	if scanner.Err() != nil || depth != 0 || len(candidates) != 1 {
		return nil, errNoBatteries
	}
	return candidates, nil
}
func iosBatteryProperty(key string) bool {
	switch key {
	case "CurrentCapacity", "MaxCapacity", "AppleRawMaxCapacity", "ExternalConnected", "IsCharging", "FullyCharged", "BatteryInstalled":
		return true
	}
	return false
}
func iosBatteryInt(props map[string]string, key string) (uint64, bool) {
	v, ok := props[key]
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.Trim(v, `"`), 10, 64)
	return n, err == nil
}
func iosBatteryBool(props map[string]string, key string) (bool, bool) {
	v, ok := props[key]
	if !ok {
		return false, false
	}
	switch strings.ToLower(strings.Trim(v, `"`)) {
	case "yes", "true", "1":
		return true, true
	case "no", "false", "0":
		return false, true
	}
	return false, false
}
func iosBatteryFromProperties(props map[string]string) (Battery, error) {
	current, ok := iosBatteryInt(props, "CurrentCapacity")
	if !ok {
		return Battery{}, errNoBatteries
	}
	maximum, ok := iosBatteryInt(props, "MaxCapacity")
	if !ok || maximum == 0 {
		return Battery{}, errNoBatteries
	}
	installed, known := iosBatteryBool(props, "BatteryInstalled")
	if _, present := props["BatteryInstalled"]; present && (!known || !installed) {
		return Battery{}, errNoBatteries
	}
	// Match the historic truncating percentage without uint64 multiplication overflow.
	percent := uint8(100)
	if current < maximum {
		hi, lo := bits.Mul64(current, 100)
		n, _ := bits.Div64(hi, lo, maximum)
		percent = uint8(n)
	}
	external, externalKnown := iosBatteryBool(props, "ExternalConnected")
	charging, chargingKnown := iosBatteryBool(props, "IsCharging")
	full, fullKnown := iosBatteryBool(props, "FullyCharged")
	state := stateUnknown
	switch {
	case externalKnown && !external:
		state = stateDischarging
	case externalKnown && external && chargingKnown && charging:
		state = stateCharging
	case fullKnown && full:
		state = stateFull
	case externalKnown && external && chargingKnown && !charging && percent == 0:
		state = stateEmpty
	case externalKnown && external && chargingKnown && !charging && percent == 100:
		state = stateFull
	case externalKnown && external && chargingKnown && !charging && fullKnown:
		state = stateIdle
	}
	b := Battery{Name: "Primary", Percent: percent, State: state, System: true}
	if raw, ok := iosBatteryInt(props, "AppleRawMaxCapacity"); ok && raw > 0 {
		b.FullChargeCapacity = raw
		b.HasFullChargeCapacity = true
	}
	return b, nil
}
