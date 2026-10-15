package wifi

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/entities/system"
)

func TestSkipFromEnv(t *testing.T) {
	for _, tc := range []struct {
		key, value string
		want       bool
	}{
		{"SKIP_WIFI", "true", true},
		{"BESZEL_AGENT_SKIP_WIFI", "true", true},
		{"SKIP_WIFI", "false", false},
		{"SKIP_WIFI", "", false},
	} {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if got := skipFromEnv(); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCollectSkipped(t *testing.T) {
	original := skip
	t.Cleanup(func() { skip = original })
	skip = func() bool { return true }
	if got := Collect(); got != nil {
		t.Fatalf("collected while skipped: %v", got)
	}
}

func TestSSIDWireSafety(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"home", "home"}, {"网络 café", "网络 café"}, {"", ""},
		{"raw\xffssid", ""}, {"truncated\xe2\x82", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			ssid := validSSID(tc.input)
			if ssid != tc.want {
				t.Fatalf("got %q, want %q", ssid, tc.want)
			}
			signal := -50.0
			payload := map[string]system.WiFi{"wlan0": {SSID: ssid, Signal: &signal}}
			wire, err := cbor.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]system.WiFi
			if err := cbor.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded["wlan0"].Signal == nil || *decoded["wlan0"].Signal != signal || decoded["wlan0"].SSID != tc.want {
				t.Fatal(decoded)
			}
		})
	}
}

func TestSignals(t *testing.T) {
	strong, weak, rounded := -40.0, -150.0, -52.6
	got := Signals(map[string]system.WiFi{
		"wlan0": {SSID: "home", Signal: &strong},
		"wlan1": {Signal: &weak},
		"wlan2": {Signal: &rounded},
		"wlan3": {SSID: "no rssi"},
	})
	want := map[string]int8{"wlan0": -40, "wlan1": -128, "wlan2": -53}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for id, signal := range want {
		if got[id] != signal {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if Signals(map[string]system.WiFi{"wlan0": {}}) != nil || Signals(nil) != nil {
		t.Fatal("expected nil without available readings")
	}
}

func TestP012SignalsValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input float64
		want  int8
		valid bool
	}{
		{"typical", -40, -40, true}, {"weak", -85, -85, true},
		{"maximum", -1, -1, true}, {"minimum", -150, -128, true},
		{"history minimum", -128, -128, true}, {"below history", -129, -128, true},
		{"zero", 0, 0, false}, {"positive", 50, 0, false},
		{"wrapping positive", 216, 0, false}, {"too low", -151, 0, false},
		{"sentinel", -999, 0, false}, {"min int32", math.MinInt32, 0, false},
		{"overflow", math.MaxFloat64, 0, false}, {"negative overflow", -math.MaxFloat64, 0, false},
		{"NaN", math.NaN(), 0, false}, {"infinity", math.Inf(1), 0, false},
		{"negative infinity", math.Inf(-1), 0, false},
		{"normalized signal", 0.75, 0, false}, {"scaled negative", -0.75, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Signals(map[string]system.WiFi{"en0": {Signal: &tc.input}})
			if !tc.valid {
				if got != nil {
					t.Fatalf("invalid signal became history: %v", got)
				}
				return
			}
			if len(got) != 1 || got["en0"] != tc.want {
				t.Fatalf("got %v, want %d", got, tc.want)
			}
		})
	}
}

func TestP012WiFiWireRoundTrip(t *testing.T) {
	signal := -85.0
	for name, snapshot := range map[string]map[string]system.WiFi{
		"absent":               nil,
		"SSID without RSSI":    {"en0": {SSID: "home"}},
		"RSSI without SSID":    {"en0": {Signal: &signal}},
		"multiple same SSID":   {"wlan0": {SSID: "same", Signal: &signal}, "Wi-Fi": {SSID: "same"}},
		"invalid SSID omitted": {"en0": {SSID: validSSID("raw\xffssid"), Signal: &signal}},
		"non ASCII":            {"en0": {SSID: "网络 café", Signal: &signal}},
	} {
		t.Run(name, func(t *testing.T) {
			original := system.CombinedData{Info: system.Info{WiFi: snapshot}, Stats: system.Stats{WiFi: Signals(snapshot)}}
			for _, codec := range []struct {
				marshal   func(any) ([]byte, error)
				unmarshal func([]byte, any) error
			}{{json.Marshal, json.Unmarshal}, {cbor.Marshal, cbor.Unmarshal}} {
				wire, err := codec.marshal(original)
				if err != nil {
					t.Fatal(err)
				}
				var got system.CombinedData
				if err := codec.unmarshal(wire, &got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(original.Info.WiFi, got.Info.WiFi) || !reflect.DeepEqual(original.Stats.WiFi, got.Stats.WiFi) {
					t.Fatalf("roundtrip: %#v", got)
				}
			}
		})
	}
}
