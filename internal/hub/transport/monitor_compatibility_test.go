package transport

import (
	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestP010MonitorRecoveryClearsOmittedFields(t *testing.T) {
	stale := monitor.SyncResponse{Result: monitor.Result{PacketLoss: 100, PacketLoss1h: 100, Cert: &monitor.CertInfo{Expires: 123}, TotalCount: 1}}
	fresh := monitor.SyncResponse{Result: monitor.Result{AvgResponse: 25, TotalCount: 1, SuccessCount: 1, ResponseSum: 25}}
	encoded, err := cbor.Marshal(fresh)
	require.NoError(t, err)
	require.NoError(t, UnmarshalResponse(common.AgentResponse{Data: encoded}, common.SyncNetworkMonitors, &stale))
	require.Equal(t, fresh, stale)
	previous := stale
	require.Error(t, UnmarshalResponse(common.AgentResponse{Data: []byte{0xa1}}, common.SyncNetworkMonitors, &stale))
	require.Equal(t, previous, stale)
	require.Error(t, unmarshalFreshCBOR(encoded, (*monitor.SyncResponse)(nil)))
	require.Error(t, unmarshalFreshCBOR(encoded, monitor.SyncResponse{}))
}

func TestP010SnapshotMapDropsAbsentRemoteData(t *testing.T) {
	dest := map[string]int{"old": 42}
	data, err := cbor.Marshal(map[string]int{"new": 37})
	require.NoError(t, err)
	require.NoError(t, unmarshalFreshCBOR(data, &dest))
	require.Equal(t, map[string]int{"new": 37}, dest)
}
