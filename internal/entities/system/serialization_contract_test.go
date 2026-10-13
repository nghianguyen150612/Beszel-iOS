package system

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/entities/container"
	"github.com/henrygd/beszel/internal/entities/zfs"
	"github.com/stretchr/testify/require"
)

func TestSerializationPresenceContracts(t *testing.T) {
	for name, marshal := range map[string]func(any) ([]byte, error){
		"json_v1": json.Marshal,
		"json_v2": func(v any) ([]byte, error) { return jsonv2.Marshal(v) },
	} {
		t.Run(name, func(t *testing.T) {
			object := func(v any) map[string]json.RawMessage {
				data, err := marshal(v)
				require.NoError(t, err)
				var result map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(data, &result))
				return result
			}
			stats := object(Stats{Temperatures: map[string]float64{}, CpuBreakdown: []float64{}})
			require.Equal(t, "0", string(stats["cpu"]), "required zero remains explicit")
			for _, key := range []string{"cpum", "mz", "bat", "b", "t", "cpub", "g", "z"} {
				require.NotContains(t, stats, key, "optional zero/empty field %s", key)
			}
			zero := 0.0
			require.Empty(t, object(WiFi{}))
			require.Equal(t, "0", string(object(WiFi{Signal: &zero})["r"]), "non-nil zero RSSI pointer is present")
			require.NotContains(t, object(ZfsPool{}), "raw")
			require.Equal(t, "true", string(object(ZfsPool{Raw: true})["raw"]))
			updates := object(PackageUpdates{Packages: []PackageUpdate{}})
			require.Equal(t, "[]", string(updates["packages"]), "successful empty inventory stays explicit")
			require.NotContains(t, updates, "securityKnown")
			require.NotContains(t, updates, "checkedAt")
			require.NotContains(t, object(PackageUpdate{}), "security")
			require.Equal(t, "true", string(object(PackageUpdate{Security: true})["security"]))
			combined := object(CombinedData{Containers: []*container.Stats{}})
			require.Equal(t, "[]", string(combined["container"]))
			require.NotContains(t, combined, "systemdUpdated")
			require.NotContains(t, object(zfs.PoolDetail{}), "scrub")
			require.JSONEq(t, `{"state":"NONE"}`, string(object(zfs.PoolDetail{Scrub: &zfs.Scrub{State: "NONE"}})["scrub"]))
		})
	}
}

func TestSerializationCBORKeysAndOptionalZero(t *testing.T) {
	zero := 0.0
	for _, signal := range []*float64{nil, &zero} {
		data, err := cbor.Marshal(WiFi{Signal: signal})
		require.NoError(t, err)
		var keys map[uint64]any
		require.NoError(t, cbor.Unmarshal(data, &keys))
		_, present := keys[1]
		require.Equal(t, signal != nil, present)
		var decoded WiFi
		require.NoError(t, cbor.Unmarshal(data, &decoded))
		require.Equal(t, signal, decoded.Signal)
	}
	data, err := cbor.Marshal(Stats{Battery: Battery{0, 1}, Bandwidth: [2]uint64{0, 10}})
	require.NoError(t, err)
	var keys map[uint64]any
	require.NoError(t, cbor.Unmarshal(data, &keys))
	require.Contains(t, keys, uint64(0), "required CPU tag is unchanged")
	require.Contains(t, keys, uint64(26), "bandwidth tag is unchanged")
	require.Contains(t, keys, uint64(29), "zero percent with known charge state is present")
	require.NotContains(t, keys, uint64(22), "absent GPU remains absent")
	// Details carries Go durations and is transported as CBOR, not stored JSON v2.
	combined := CombinedData{Details: &Details{MemoryTotal: 1024}}
	data, err = cbor.Marshal(combined)
	require.NoError(t, err)
	var decoded CombinedData
	require.NoError(t, cbor.Unmarshal(data, &decoded))
	require.Equal(t, combined.Details, decoded.Details)
}
