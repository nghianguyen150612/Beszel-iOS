package system

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"
)

func TestP014GPUPowerPresenceCodecs(t *testing.T) {
	for name, codec := range map[string]struct {
		encode func(any) ([]byte, error)
		decode func([]byte, any) error
	}{
		"json_v1": {json.Marshal, json.Unmarshal},
		"json_v2": {func(v any) ([]byte, error) { return jsonv2.Marshal(v) }, func(b []byte, v any) error { return jsonv2.Unmarshal(b, v) }},
		"cbor":    {cbor.Marshal, cbor.Unmarshal},
	} {
		t.Run(name, func(t *testing.T) {
			var missing GPUData
			null, err := codec.encode(nil)
			require.NoError(t, err)
			require.Error(t, codec.decode(null, &missing), "a null device must not become an idle GPU")

			for _, present := range []bool{false, true} {
				source := GPUData{Name: "synthetic", Usage: 0}
				if present {
					source.SetPower(0)
					source.SetPackagePower(0)
				}
				raw, err := codec.encode(source)
				require.NoError(t, err)
				var decoded GPUData
				require.NoError(t, codec.decode(raw, &decoded))
				require.Equal(t, present, decoded.HasPower())
				require.Equal(t, present, decoded.HasPackagePower())
				require.Equal(t, source, decoded)
				public, err := jsonv2.Marshal(decoded)
				require.NoError(t, err)
				var fields map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(public, &fields))
				if present {
					require.Equal(t, "0", string(fields["p"]))
					require.Equal(t, "0", string(fields["pp"]))
				} else {
					require.NotContains(t, fields, "p")
					require.NotContains(t, fields, "pp")
				}
			}
		})
	}
	var null GPUData
	require.NoError(t, json.Unmarshal([]byte(`{"n":"A","u":0,"p":null,"pp":null}`), &null))
	require.False(t, null.HasPower())
	require.False(t, null.HasPackagePower())
}
