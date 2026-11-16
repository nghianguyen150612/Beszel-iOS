//go:build testing

package hub

import (
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"os"
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
			{PublicAppInfo{AUTO_LOGIN: value}, "AUTO_LOGIN"},
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

func TestPublicAppInfoAutomaticLoginConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		value  string
		expect bool
	}{
		{name: "disabled by default"},
		{name: "automatic email", key: "AUTO_LOGIN", value: "owner@example.test", expect: true},
		{name: "prefixed automatic email", key: "BESZEL_HUB_AUTO_LOGIN", value: "owner@example.test", expect: true},
		{name: "trusted header", key: "TRUSTED_AUTH_HEADER", value: "X-Forwarded-User", expect: true},
		{name: "prefixed trusted header", key: "BESZEL_HUB_TRUSTED_AUTH_HEADER", value: "X-Forwarded-User", expect: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unsetEnvForAutomaticLoginTest(t, "BESZEL_HUB_AUTO_LOGIN")
			unsetEnvForAutomaticLoginTest(t, "BESZEL_HUB_TRUSTED_AUTH_HEADER")
			t.Setenv("AUTO_LOGIN", "")
			t.Setenv("TRUSTED_AUTH_HEADER", "")
			if tc.key != "" {
				t.Setenv(tc.key, tc.value)
			}
			info := getPublicAppInfo(&Hub{appURL: "http://localhost/"})
			require.Equal(t, tc.expect, info.AUTO_LOGIN)
		})
	}
}

func unsetEnvForAutomaticLoginTest(t *testing.T, key string) {
	t.Helper()
	old, existed := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
