package transport

import (
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/require"
)

// Numeric-key fixtures match the existing wire contract, independent of the
// current Go entity tags. Older agents may omit Wi-Fi entirely.
func TestP012WiFiOldAgentWireRecovery(t *testing.T) {
	var dest system.CombinedData
	for _, iface := range []string{"wlan0", "en0", "Wi-Fi", "", "en2"} {
		info, stats := map[uint]any{}, map[uint]any{}
		if iface != "" {
			info[26] = map[string]any{iface: map[uint]any{0: "网络 café", 1: -85}}
			stats[40] = map[string]int8{iface: -85}
		}
		wire, err := cbor.Marshal(map[uint]any{0: stats, 1: info})
		require.NoError(t, err)
		require.NoError(t, UnmarshalResponse(common.AgentResponse{Data: wire}, common.GetData, &dest))
		if iface == "" {
			require.Empty(t, dest.Info.WiFi)
			require.Empty(t, dest.Stats.WiFi)
		} else {
			require.Len(t, dest.Info.WiFi, 1)
			require.Equal(t, "网络 café", dest.Info.WiFi[iface].SSID)
			require.NotNil(t, dest.Info.WiFi[iface].Signal)
			require.Equal(t, -85.0, *dest.Info.WiFi[iface].Signal)
			require.Equal(t, map[string]int8{iface: -85}, dest.Stats.WiFi)
		}
	}
}
