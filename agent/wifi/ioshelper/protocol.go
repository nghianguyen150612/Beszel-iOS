// Package ioshelper contains the disabled iOS diagnostic helper boundary.
// It is not wired into Agent polling. Parsing a helper claim does not establish
// its native ABI, association freshness, dBm provenance or authorization.
package ioshelper

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"unicode/utf8"

	"github.com/henrygd/beszel/internal/entities/system"
)

const (
	ProtocolVersion = 1
	MaxOutputBytes  = 16 * 1024
	MaxDevices      = 16
)

var ErrProtocol = errors.New("invalid Wi-Fi helper protocol")

type snapshot struct {
	Version int      `json:"version"`
	Source  string   `json:"source"`
	Status  string   `json:"status"`
	Reason  string   `json:"reason,omitempty"`
	Devices []device `json:"devices"`
}

type device struct {
	Interface      string          `json:"interface"`
	Powered        bool            `json:"powered"`
	Station        bool            `json:"station"`
	Association    string          `json:"association"`
	CurrentNetwork bool            `json:"current_network"`
	SSIDBase64     string          `json:"ssid_b64,omitempty"`
	RSSI           json.RawMessage `json:"rssi,omitempty"`
}

func interfaceExists(name string) bool {
	_, err := net.InterfaceByName(name)
	return err == nil
}

// parseSnapshot validates the narrow versioned helper protocol, not the truth
// of the helper's association claim. Activation must independently prove that
// the native producer can authoritatively emit "associated" and raw dBm.
func parseSnapshot(data []byte, exists func(string) bool) (map[string]system.WiFi, error) {
	if len(data) == 0 || len(data) > MaxOutputBytes || !utf8.Valid(data) || !uniqueJSON(data) {
		return nil, ErrProtocol
	}
	var report snapshot
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&report) != nil || report.Version != ProtocolVersion || report.Source != "mobilewifi-ios12" || len(report.Reason) > 64 || len(report.Devices) > MaxDevices {
		return nil, ErrProtocol
	}
	if report.Status == "unavailable" {
		if len(report.Devices) != 0 {
			return nil, ErrProtocol
		}
		return nil, nil
	}
	if report.Status != "ok" || report.Reason != "" {
		return nil, ErrProtocol
	}
	var readings map[string]system.WiFi
	seen := make(map[string]bool, len(report.Devices))
	for _, d := range report.Devices {
		if d.Interface != "" && seen[d.Interface] {
			return nil, ErrProtocol
		}
		seen[d.Interface] = true
		// A non-NULL current-network object alone is explicitly insufficient.
		if !d.Powered || !d.Station || d.Association != "associated" || !d.CurrentNetwork || !validInterface(d.Interface) || !exists(d.Interface) {
			continue
		}
		reading := system.WiFi{}
		if len(d.SSIDBase64) <= 44 {
			if ssid, err := base64.StdEncoding.Strict().DecodeString(d.SSIDBase64); err == nil && len(ssid) <= 32 && utf8.Valid(ssid) {
				reading.SSID = string(ssid)
			}
		}
		reading.Signal = rawRSSI(d.RSSI)
		if readings == nil {
			readings = make(map[string]system.WiFi)
		}
		readings[d.Interface] = reading
	}
	return readings, nil
}

func validInterface(name string) bool {
	if len(name) == 0 || len(name) > 15 {
		return false
	}
	for _, b := range []byte(name) {
		if !(b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-' || b == '.' || b == ':') {
			return false
		}
	}
	return true
}

func rawRSSI(data json.RawMessage) *float64 {
	var property map[string]json.RawMessage
	if json.Unmarshal(data, &property) != nil {
		return nil
	}
	// Only an integer raw component is accepted. Scaled keys, floats,
	// exponents, strings, overflow and native error sentinels are not dBm.
	value, err := strconv.ParseInt(string(bytes.TrimSpace(property["RSSI_CTL_AGR"])), 10, 64)
	if err != nil || value < -150 || value > -1 {
		return nil
	}
	signal := float64(value)
	return &signal
}

// Reject duplicate keys, trailing values and excessive nesting before typed
// decoding, so conflicting association/identity fields cannot be last-wins.
func uniqueJSON(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	remaining := 1024
	var value func(int) bool
	value = func(depth int) bool {
		remaining--
		if depth > 8 || remaining < 0 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delim, compound := token.(json.Delim)
		if !compound {
			return true
		}
		switch delim {
		case '{':
			keys := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return false
				}
				name, ok := key.(string)
				if !ok || keys[name] {
					return false
				}
				keys[name] = true
				if !value(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim('}')
		case '[':
			for d.More() {
				if !value(depth + 1) {
					return false
				}
			}
			end, err := d.Token()
			return err == nil && end == json.Delim(']')
		default:
			return false
		}
	}
	if !value(0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}
