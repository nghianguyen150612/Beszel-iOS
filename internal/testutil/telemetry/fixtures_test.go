//go:build testing

package telemetry

import (
	"encoding/json"
	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRemoteFixtureSchema(t *testing.T) {
	ids, users := map[string]bool{}, map[string]bool{}
	for _, f := range All() {
		t.Run(f.Name, func(t *testing.T) {
			require.False(t, ids[f.SystemID])
			ids[f.SystemID] = true
			require.False(t, users[f.UserID])
			users[f.UserID] = true
			wire, err := cbor.Marshal(f.Data)
			require.NoError(t, err)
			require.Less(t, len(wire), 8192, "representative snapshots stay bounded")
			var decoded system.CombinedData
			require.NoError(t, cbor.Unmarshal(wire, &decoded))
			require.Equal(t, f.Data, decoded)
			b, err := json.Marshal(decoded)
			require.NoError(t, err)
			var public map[string]any
			require.NoError(t, json.Unmarshal(b, &public))
			stats := public["stats"].(map[string]any)
			require.Equal(t, []any{float64(0), float64(25), float64(50), float64(25)}, stats["cpus"])
			if f.Name == "ios" {
				for _, key := range []string{"g", "wf", "z", "t", "f", "dio", "diot"} {
					require.NotContains(t, stats, key)
				}
				require.Nil(t, decoded.Info.WiFi)
				require.Nil(t, decoded.SystemdServices)
				require.Equal(t, []uint16{0}, decoded.Info.PackageUpdates)
			} else {
				require.NotEmpty(t, decoded.Info.WiFi)
				require.Equal(t, f.Data.Details.Os, decoded.Details.Os)
			}
			if f.Name == "windows" || f.Name == "macos" {
				require.Nil(t, decoded.SystemdServices)
				require.Nil(t, decoded.Stats.ZfsPools)
			}
		})
	}
	first, second := All(), All()
	first[0].Data.Info.WiFi["different"] = system.WiFi{}
	require.Len(t, second[0].Data.Info.WiFi, 1, "fixtures must not share mutable maps")
}
