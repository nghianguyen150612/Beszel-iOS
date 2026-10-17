package system

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestP014ByteMetricJSONDecoding(t *testing.T) {
	for name, decode := range map[string]func([]byte, any) error{
		"v1": json.Unmarshal, "v2": func(b []byte, v any) error { return jsonv2.Unmarshal(b, v) },
	} {
		t.Run(name, func(t *testing.T) {
			for _, raw := range []string{`{"cpus":[0,25,100],"bat":[0,1]}`, `{"cpus":"ABlk","bat":"AAE="}`} {
				var got Stats
				require.NoError(t, decode([]byte(raw), &got))
				require.Equal(t, Uint8Slice{0, 25, 100}, got.CpuCoresUsage)
				require.Equal(t, Battery{0, 1}, got.Battery)
			}
			for _, raw := range []string{`{"cpus":[256]}`, `{"cpus":[-1]}`, `{"bat":[1,2,3]}`, `{"bat":"AA=="}`} {
				var got Stats
				require.Error(t, decode([]byte(raw), &got), raw)
			}
		})
	}
}
