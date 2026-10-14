//go:build testing

package agent

import (
	jsonv1 "encoding/json"
	jsonv2 "encoding/json/v2"
	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestP008PayloadPresence(t *testing.T) {
	for _, tt := range []struct {
		name       string
		stats      system.Stats
		hasBattery bool
	}{
		{"absent", system.Stats{}, false},
		{"legitimate empty battery", system.Stats{Battery: system.Battery{0, 4}, Batteries: map[string]uint8{"Primary": 0}}, true},
		{"present metrics", system.Stats{Cpu: 15, Mem: 1, MemUsed: .8, MemPct: 80, DiskTotal: 16, DiskUsed: 8, DiskPct: 50, Battery: system.Battery{42, 3}, Batteries: map[string]uint8{"Primary": 42}, NetworkInterfaces: map[string][4]uint64{"en0": {0, 1000, 0, 2000}}, CpuCoresUsage: system.Uint8Slice{15, 15}}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, marshal := range []func(any) ([]byte, error){jsonv1.Marshal, func(v any) ([]byte, error) { return jsonv2.Marshal(v) }} {
				raw, err := marshal(tt.stats)
				require.NoError(t, err)
				var fields map[string]any
				require.NoError(t, jsonv1.Unmarshal(raw, &fields))
				assert.Contains(t, fields, "cpu")
				assert.Contains(t, fields, "m")
				assert.Contains(t, fields, "d")
				_, has := fields["bat"]
				assert.Equal(t, tt.hasBattery, has)
				for _, key := range []string{"s", "su", "dr", "dw", "dio", "dios", "diot", "wf"} {
					assert.NotContains(t, fields, key)
				}
				if tt.hasBattery {
					assert.Contains(t, fields, "bats")
				}
			}
			raw, err := cbor.Marshal(tt.stats)
			require.NoError(t, err)
			var fields map[uint64]any
			require.NoError(t, cbor.Unmarshal(raw, &fields))
			_, has := fields[29]
			assert.Equal(t, tt.hasBattery, has)
			for _, key := range []uint64{7, 8, 12, 13, 32, 35, 38, 40} {
				assert.NotContains(t, fields, key)
			}
			var decoded system.Stats
			require.NoError(t, cbor.Unmarshal(raw, &decoded))
			assert.Equal(t, tt.stats.Battery, decoded.Battery)
			assert.Equal(t, tt.stats.NetworkInterfaces, decoded.NetworkInterfaces)
		})
	}
}
