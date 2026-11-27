package common

import (
	"github.com/fxamacker/cbor/v2"
	"testing"
)

func TestDecodePartialData(t *testing.T) {
	u := func(n uint8) *uint8 { return &n }
	for _, tc := range []struct {
		name    string
		value   any
		version uint8
		valid   bool
	}{
		{"zero", PartialData{u(1), []PartialMeasurement{{u(1), u(0), u(4)}}}, 1, true},
		{"eighty", PartialData{u(1), []PartialMeasurement{{u(1), u(80), u(3)}}}, 1, true},
		{"unavailable", PartialData{u(1), []PartialMeasurement{}}, 1, true},
		{"unknown state", PartialData{u(1), []PartialMeasurement{{u(1), u(0), u(0)}}}, 1, false},
		{"invalid percent", PartialData{u(1), []PartialMeasurement{{u(1), u(101), u(4)}}}, 1, false},
		{"duplicate metric", PartialData{u(1), []PartialMeasurement{{u(1), u(0), u(4)}, {u(1), u(80), u(4)}}}, 1, false},
		{"unknown metric", PartialData{u(1), []PartialMeasurement{{u(2), u(0), u(4)}}}, 1, false},
		{"unknown version", PartialData{u(2), []PartialMeasurement{}}, 2, false},
		{"missing list", PartialData{u(1), nil}, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := cbor.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			_, err = DecodePartialData(b, tc.version)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
	// Duplicate version keys must be rejected rather than overwritten.
	if _, err := DecodePartialData([]byte{0xa3, 0x18, 0x64, 0x01, 0x18, 0x64, 0x01, 0x18, 0x65, 0x80}, 1); err == nil {
		t.Fatal("accepted duplicate key")
	}
}

func TestDecodePartialDataRejectsMalformedWire(t *testing.T) {
	cases := map[string][]byte{
		"duplicate measurement key": {0xa2, 0x18, 0x64, 0x01, 0x18, 0x65, 0x81, 0xa4, 0x00, 0x01, 0x01, 0x00, 0x01, 0x50, 0x02, 0x04},
		"wrong measurement type":    {0xa2, 0x18, 0x64, 0x01, 0x18, 0x65, 0x81, 0xa3, 0x00, 0x01, 0x01, 0x61, 0x30, 0x02, 0x04},
		"conflicting legacy field":  {0xa3, 0x18, 0x64, 0x01, 0x18, 0x65, 0x80, 0x00, 0xa0},
		"trailing item":             {0xa2, 0x18, 0x64, 0x01, 0x18, 0x65, 0x80, 0x00},
		"null version":              {0xa2, 0x18, 0x64, 0xf6, 0x18, 0x65, 0x80},
	}
	for name, wire := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodePartialData(wire, PartialDataVersion1); err == nil {
				t.Fatal("accepted malformed partial payload")
			}
		})
	}
	if _, err := DecodePartialData(make([]byte, 1025), PartialDataVersion1); err == nil {
		t.Fatal("accepted oversized payload")
	}
}
