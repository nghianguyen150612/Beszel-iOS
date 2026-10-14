//go:build testing

package hub

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"testing"

	"github.com/henrygd/beszel/internal/alerts"
	"github.com/stretchr/testify/require"
)

func TestPublicAndAlertJSONZeroContracts(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, test := range []struct {
			value any
			key   string
		}{
			{PublicAppInfo{OAUTH_DISABLE_POPUP: value}, "OAUTH_DISABLE_POPUP"},
			{alerts.SystemAlertZfsPool{Raw: value}, "raw"},
		} {
			v1, err := json.Marshal(test.value)
			require.NoError(t, err)
			v2, err := jsonv2.Marshal(test.value)
			require.NoError(t, err)
			require.JSONEq(t, string(v1), string(v2))
			var object map[string]any
			require.NoError(t, json.Unmarshal(v2, &object))
			if value {
				require.Equal(t, true, object[test.key])
			} else {
				require.NotContains(t, object, test.key)
			}
		}
	}
}
