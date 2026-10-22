package system

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"math"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"
)

func TestMemoryByteTelemetryWireCompatibility(t *testing.T) {
	const gib = uint64(1 << 30)
	stats := Stats{
		Mem: 2, MemUsed: 1, MemPct: 50, MemBuffCache: .25,
		MaxMem: 1.5, Swap: .5, SwapUsed: .25,
		MemBytes:    [4]uint64{2 * gib, gib, gib / 4, 0},
		MaxMemBytes: ^uint64(0), SwapBytes: [2]uint64{gib / 2, gib / 4},
	}

	for name, marshal := range map[string]func(any) ([]byte, error){
		"json_v1": json.Marshal,
		"json_v2": func(value any) ([]byte, error) { return jsonv2.Marshal(value) },
	} {
		t.Run(name, func(t *testing.T) {
			wire, err := marshal(stats)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(wire, &fields))
			require.JSONEq(t, `[2147483648,1073741824,268435456,0]`, string(fields["mem"]))
			require.JSONEq(t, `[536870912,268435456]`, string(fields["swap"]))
			require.Contains(t, fields, "memm")
			require.Contains(t, fields, "m", "deprecated GB field remains available to old Hubs")
			var decoded Stats
			require.NoError(t, json.Unmarshal(wire, &decoded))
			require.Equal(t, stats.MemBytes, decoded.MemBytes)
			require.Equal(t, stats.MaxMemBytes, decoded.MaxMemBytes)
			require.Equal(t, stats.SwapBytes, decoded.SwapBytes)

			// An older JSON Hub ignores the new names and can still use the GB contract.
			var oldHub struct {
				Mem     float64 `json:"m"`
				MemUsed float64 `json:"mu"`
				Swap    float64 `json:"s"`
			}
			require.NoError(t, json.Unmarshal(wire, &oldHub))
			require.Equal(t, 2.0, oldHub.Mem)
			require.Equal(t, 1.0, oldHub.MemUsed)
			require.Equal(t, .5, oldHub.Swap)
		})
	}

	cborWire, err := cbor.Marshal(stats)
	require.NoError(t, err)
	var keys map[uint64]cbor.RawMessage
	require.NoError(t, cbor.Unmarshal(cborWire, &keys))
	require.Contains(t, keys, uint64(41), "memory byte arrays use the upstream key")
	require.Contains(t, keys, uint64(42), "swap byte arrays use the upstream key")
	var decoded Stats
	require.NoError(t, cbor.Unmarshal(cborWire, &decoded))
	require.Equal(t, stats.MemBytes, decoded.MemBytes)
	require.Equal(t, stats.SwapBytes, decoded.SwapBytes)

	// Old CBOR decoders ignore keys they do not know; legacy keys remain readable.
	var oldHub struct {
		Mem     float64 `cbor:"2,keyasint"`
		MemUsed float64 `cbor:"3,keyasint"`
		Swap    float64 `cbor:"7,keyasint"`
	}
	require.NoError(t, cbor.Unmarshal(cborWire, &oldHub))
	require.Equal(t, 2.0, oldHub.Mem)
	require.Equal(t, 1.0, oldHub.MemUsed)
	require.Equal(t, .5, oldHub.Swap)
}

func TestLegacyMemoryMigrationIsBoundedAndBytePrecise(t *testing.T) {
	require.Equal(t, uint64(1), LegacyGigabytesToBytes(1.0/(1<<30)))
	require.Equal(t, uint64(2<<30), LegacyGigabytesToBytes(2))
	require.Equal(t, uint64(0), LegacyGigabytesToBytes(-1))
	require.Equal(t, uint64(0), LegacyGigabytesToBytes(math.NaN()))
	require.Equal(t, uint64(0), LegacyGigabytesToBytes(math.Inf(1)))
	require.Equal(t, ^uint64(0), LegacyGigabytesToBytes(1e300))

	legacy := Stats{Mem: 0.125, MemUsed: 0.25, MemBuffCache: .5, MaxMem: .5, Swap: .25, SwapUsed: .5}
	legacy.MigrateDeprecatedMemoryFields()
	require.Equal(t, [4]uint64{1 << 27, 1 << 27, 1 << 27, 0}, legacy.MemBytes,
		"legacy used and cache values above capacity are bounded")
	require.Equal(t, [2]uint64{1 << 28, 1 << 28}, legacy.SwapBytes, "legacy used values are bounded by total")
	require.Equal(t, uint64(1<<29), legacy.MaxMemBytes)

	modern := Stats{
		Mem: 9, MemUsed: 9, Swap: 9,
		MemBytes: [4]uint64{64, 100, 120, 90}, SwapBytes: [2]uint64{0, 90},
	}
	modern.MigrateDeprecatedMemoryFields()
	require.Equal(t, [4]uint64{64, 64, 64, 64}, modern.MemBytes)
	require.Equal(t, [2]uint64{}, modern.SwapBytes, "zero capacity cannot carry positive usage")
}
