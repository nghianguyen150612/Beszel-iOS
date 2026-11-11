package system

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"
)

// Generated with P027 c239ce0d and cbor v2.9.6, not with the new encoder.
// Fixed struct fields and single-entry maps make this byte contract stable.
func TestP028CBORHistoricalWire(t *testing.T) {
	encoded, err := os.ReadFile("testdata/p027-cbor.hex")
	require.NoError(t, err)
	wire, err := hex.DecodeString(strings.TrimSpace(string(encoded)))
	require.NoError(t, err)
	expected := Stats{Cpu: 12.5, Battery: Battery{59, 5}, CpuCoresUsage: Uint8Slice{0, 100}, MemBytes: [4]uint64{1073741824, 536870912, 1024, 0}, SwapBytes: [2]uint64{2147483648, 4096}, Fans: map[string]uint16{"cpu": 1200}, WiFi: map[string]int8{"en0": -55}}
	var decoded Stats
	require.NoError(t, cbor.Unmarshal(wire, &decoded))
	require.Equal(t, expected, decoded)
	current, err := cbor.Marshal(expected)
	require.NoError(t, err)
	require.Equal(t, wire, current, "P027 agent/Hub telemetry bytes must remain compatible")
}
