package ioshelper

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/agent/wifi"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/require"
)

func fixtureDevice() device {
	return device{Interface: "en0", Powered: true, Station: true, Association: "associated", CurrentNetwork: true, SSIDBase64: base64.StdEncoding.EncodeToString([]byte("网络 café")), RSSI: json.RawMessage(`{"RSSI_CTL_AGR":-40}`)}
}
func fixtureReport(devices ...device) []byte {
	b, err := json.Marshal(snapshot{Version: 1, Source: "mobilewifi-ios12", Status: "ok", Devices: devices})
	if err != nil {
		panic(err)
	}
	return b
}
func fixtureExists(name string) bool { return name == "en0" || name == "en1" }

func TestProtocolAssociationAndMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*device)
		count  int
		signal bool
		ssid   string
	}{
		{"associated", func(*device) {}, 1, true, "网络 café"},
		{"disconnected stale network", func(d *device) { d.Association = "disconnected" }, 0, false, ""},
		{"unknown with network", func(d *device) { d.Association = "unknown" }, 0, false, ""},
		{"disabled", func(d *device) { d.Powered = false }, 0, false, ""},
		{"not station", func(d *device) { d.Station = false }, 0, false, ""},
		{"no current network", func(d *device) { d.CurrentNetwork = false }, 0, false, ""},
		{"missing interface", func(d *device) { d.Interface = "" }, 0, false, ""},
		{"unknown OS interface", func(d *device) { d.Interface = "en9" }, 0, false, ""},
		{"invalid interface", func(d *device) { d.Interface = "en0\x00other" }, 0, false, ""},
		{"missing RSSI", func(d *device) { d.RSSI = nil }, 1, false, "网络 café"},
		{"wrong dictionary type", func(d *device) { d.RSSI = json.RawMessage(`[-40]`) }, 1, false, "网络 café"},
		{"hidden SSID", func(d *device) { d.SSIDBase64 = "" }, 1, true, ""},
		{"invalid SSID bytes", func(d *device) { d.SSIDBase64 = base64.StdEncoding.EncodeToString([]byte{0xff, 0xfe}) }, 1, true, ""},
		{"invalid base64", func(d *device) { d.SSIDBase64 = "!" }, 1, true, ""},
		{"SSID too long", func(d *device) { d.SSIDBase64 = base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 33))) }, 1, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := fixtureDevice()
			tc.change(&d)
			got, err := parseSnapshot(fixtureReport(d), fixtureExists)
			require.NoError(t, err)
			require.Len(t, got, tc.count)
			if tc.count == 1 {
				require.Equal(t, tc.ssid, got["en0"].SSID)
				require.Equal(t, tc.signal, got["en0"].Signal != nil)
			}
		})
	}
}
func TestProtocolRawRSSI(t *testing.T) {
	for _, raw := range []string{"-40", "-85", "-1", "-128", "-150", "0", "1", "-151", "-999", "-2147483648", "18446744073709551615", "-4e1", "-40.0", `"-40"`, "null", "true"} {
		t.Run(raw, func(t *testing.T) {
			d := fixtureDevice()
			d.RSSI = json.RawMessage(`{"RSSI_CTL_AGR":` + raw + `}`)
			got, err := parseSnapshot(fixtureReport(d), fixtureExists)
			require.NoError(t, err)
			valid := raw == "-40" || raw == "-85" || raw == "-1" || raw == "-128" || raw == "-150"
			require.Equal(t, valid, got["en0"].Signal != nil)
			if valid {
				require.Equal(t, raw, fmt.Sprint(int(*got["en0"].Signal)))
			}
		})
	}
	for _, raw := range []string{`{}`, `{"ScaledRSSI":0.75}`, `null`, `"wrong"`} {
		d := fixtureDevice()
		d.RSSI = json.RawMessage(raw)
		got, err := parseSnapshot(fixtureReport(d), fixtureExists)
		require.NoError(t, err)
		require.Nil(t, got["en0"].Signal)
	}
}
func TestProtocolRejectMalformedAndAmbiguous(t *testing.T) {
	valid := string(fixtureReport(fixtureDevice()))
	for _, raw := range []string{"", "null", "[]", "{", valid + valid,
		strings.Replace(valid, `"version":1`, `"version":2`, 1),
		strings.Replace(valid, `"version":1`, `"version":1,"version":1`, 1),
		strings.Replace(valid, `"interface":"en0"`, `"interface":"en0","interface":"en1"`, 1),
		strings.Replace(valid, `"RSSI_CTL_AGR":-40`, `"RSSI_CTL_AGR":-40,"RSSI_CTL_AGR":0`, 1),
		strings.Replace(valid, `"status":"ok"`, `"status":"unknown"`, 1),
		strings.Replace(valid, `"source":"mobilewifi-ios12"`, `"source":"quality-percent"`, 1),
		strings.Replace(valid, `"version":1`, `"unexpected":true,"version":1`, 1),
		strings.Repeat(" ", MaxOutputBytes) + valid,
		strings.Replace(valid, "en0", "bad\xff", 1),
	} {
		got, err := parseSnapshot([]byte(raw), fixtureExists)
		require.ErrorIs(t, err, ErrProtocol)
		require.Nil(t, got)
	}
	d := fixtureDevice()
	_, err := parseSnapshot(fixtureReport(d, d), fixtureExists)
	require.ErrorIs(t, err, ErrProtocol)
	devices := make([]device, MaxDevices+1)
	_, err = parseSnapshot(fixtureReport(devices...), fixtureExists)
	require.ErrorIs(t, err, ErrProtocol)
}
func TestProtocolLifecycleAndWire(t *testing.T) {
	first := fixtureDevice()
	other := fixtureDevice()
	other.Interface = "en1"
	other.SSIDBase64 = base64.StdEncoding.EncodeToString([]byte("other"))
	disconnected := first
	disconnected.Association = "disconnected"
	for _, ds := range [][]device{{first}, {disconnected}, {first}, {other}, {first, other}, {}, nil, {first}} {
		got, err := parseSnapshot(fixtureReport(ds...), fixtureExists)
		require.NoError(t, err)
		want := 0
		for _, d := range ds {
			if d.Association == "associated" {
				want++
			}
		}
		require.Len(t, got, want)
		original := system.CombinedData{Info: system.Info{WiFi: got}, Stats: system.Stats{WiFi: wifi.Signals(got)}}
		for _, codec := range []struct {
			marshal   func(any) ([]byte, error)
			unmarshal func([]byte, any) error
		}{{json.Marshal, json.Unmarshal}, {cbor.Marshal, cbor.Unmarshal}} {
			b, err := codec.marshal(original)
			require.NoError(t, err)
			var decoded system.CombinedData
			require.NoError(t, codec.unmarshal(b, &decoded))
			require.Equal(t, original.Info.WiFi, decoded.Info.WiFi)
			require.Equal(t, original.Stats.WiFi, decoded.Stats.WiFi)
		}
	}
	got, err := parseSnapshot([]byte(`{"version":1,"source":"mobilewifi-ios12","status":"unavailable","reason":"activation-blocked","devices":[]}`), fixtureExists)
	require.NoError(t, err)
	require.Nil(t, got)
	_, err = parseSnapshot([]byte(`{"version":1,"source":"mobilewifi-ios12","status":"unavailable","devices":[{}]}`), fixtureExists)
	require.ErrorIs(t, err, ErrProtocol)
}

func TestProtocolRejectCBORAndResourceAbuse(t *testing.T) {
	data, err := cbor.Marshal(snapshot{Version: 1, Source: "mobilewifi-ios12", Status: "ok"})
	require.NoError(t, err)
	_, err = parseSnapshot(data, fixtureExists)
	require.ErrorIs(t, err, ErrProtocol)
	for _, raw := range []string{strings.Repeat("[", 9) + "0" + strings.Repeat("]", 9), "[" + strings.Repeat("0,", 1100) + "0]"} {
		require.False(t, uniqueJSON([]byte(raw)))
	}
}
