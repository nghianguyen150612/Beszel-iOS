package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/require"
)

func TestPlatformFixtureSchemas(t *testing.T) {
	ids := map[string]bool{}
	for _, platform := range []string{"linux", "windows", "macos", "ios"} {
		t.Run(platform, func(t *testing.T) {
			f, err := Load(platform)
			require.NoError(t, err)
			require.False(t, ids[f.SystemID])
			ids[f.SystemID] = true
			require.False(t, ids[f.OwnerID])
			ids[f.OwnerID] = true
			require.Equal(t, platform, f.Platform)
			require.Equal(t, "p014-"+platform, f.Data.Details.Hostname)
			for name, codec := range map[string]struct {
				encode func(any) ([]byte, error)
				decode func([]byte, any) error
			}{
				"json_v1": {json.Marshal, json.Unmarshal}, "cbor": {cbor.Marshal, cbor.Unmarshal},
			} {
				t.Run(name, func(t *testing.T) {
					raw, err := codec.encode(f.Data)
					require.NoError(t, err)
					var got system.CombinedData
					require.NoError(t, codec.decode(raw, &got))
					require.Equal(t, f.Data.Stats, got.Stats)
					require.Equal(t, f.Data.Info, got.Info)
					require.Equal(t, f.Data.Details, got.Details)
				})
			}
			if platform == "ios" {
				require.Empty(t, f.Data.Stats.GPUData)
				require.Empty(t, f.Data.Stats.Temperatures)
				require.Empty(t, f.Data.Stats.WiFi)
				require.Empty(t, f.Data.SystemdServices)
			} else {
				require.Zero(t, f.Data.Stats.GPUData["gpu0"].Usage)
				require.Contains(t, f.Data.Stats.Temperatures, "sensor0")
				require.Equal(t, system.Battery{0, 1}, f.Data.Stats.Battery)
			}
		})
	}
}
