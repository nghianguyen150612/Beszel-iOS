//go:build testing

package systems

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/hub/transport"
	"github.com/henrygd/beszel/internal/hub/ws"
	"github.com/henrygd/beszel/internal/testutil/telemetry"
	"github.com/lxzan/gws"
	"github.com/stretchr/testify/require"
)

type remoteTelemetryPeer struct {
	gws.BuiltinEventHandler
	version  semver.Version
	payloads chan []byte
}

func (p *remoteTelemetryPeer) OnMessage(conn *gws.Conn, message *gws.Message) {
	defer message.Close()
	var req common.HubRequest[cbor.RawMessage]
	if cbor.Unmarshal(message.Bytes(), &req) != nil {
		return
	}
	payload := <-p.payloads
	if p.version.GTE(beszel.MinVersionAgentResponse) {
		payload, _ = cbor.Marshal(common.AgentResponse{Id: req.Id, Data: payload})
	}
	_ = conn.WriteMessage(gws.OpcodeBinary, payload)
}

func connectRemoteTelemetryPeer(t *testing.T, version string) (*ws.WsConn, chan []byte) {
	t.Helper()
	connections := make(chan *ws.WsConn, 1)
	v := semver.MustParse(version)
	upgrader := gws.NewUpgrader(&monitorSyncServer{}, nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		wc := ws.NewWsConnection(conn, v)
		conn.Session().Store("wsConn", wc)
		connections <- wc
		conn.ReadLoop()
	}))
	t.Cleanup(server.Close)
	peer := &remoteTelemetryPeer{version: v, payloads: make(chan []byte, 8)}
	conn, _, err := gws.NewClient(peer, &gws.ClientOption{Addr: "ws" + strings.TrimPrefix(server.URL, "http")})
	require.NoError(t, err)
	done := make(chan struct{})
	go func() { defer close(done); conn.ReadLoop() }()
	t.Cleanup(func() { _ = conn.NetConn().Close(); <-done })
	select {
	case wc := <-connections:
		t.Cleanup(func() { wc.Close(nil) })
		return wc, peer.payloads
	case <-time.After(3 * time.Second):
		t.Fatal("remote fixture did not connect")
		return nil, nil
	}
}

func TestP014RemoteTelemetryPolling(t *testing.T) {
	for _, fixture := range telemetry.All() {
		for _, version := range []string{"0.12.0", "0.19.0", "0.21.0"} {
			t.Run(fixture.Name+"/"+version, func(t *testing.T) {
				sys, app := newTestSystemWithHub(t)
				sys.ctx = context.Background()
				wc, payloads := connectRemoteTelemetryPeer(t, version)
				sys.WsConn = wc
				sys.setAgentVersion(semver.MustParse(version))
				full := fixture.Data
				full.Info.AgentVersion = version
				if version == "0.12.0" || version == "0.19.0" {
					full.Monitors = nil
				} // feature guard: monitors require 0.20+
				wire, err := cbor.Marshal(full)
				require.NoError(t, err)
				payloads <- wire
				require.NoError(t, sys.update(), "actual scheduled-poll path must accept the remote schema")
				current, err := app.FindRecordById("systems", sys.Id)
				require.NoError(t, err)
				require.Equal(t, "up", current.GetString("status"))
				var info system.Info
				require.NoError(t, current.UnmarshalJSONField("info", &info))
				require.Equal(t, full.Info.Cpu, info.Cpu)
				require.Equal(t, full.Info.WiFi, info.WiFi)
				details, err := app.FindRecordById("system_details", sys.Id)
				require.NoError(t, err)
				require.Equal(t, fixture.Data.Details.OsName, details.GetString("os_name"))
				require.EqualValues(t, fixture.Data.Details.Os, details.GetInt("os"))
				rows, err := app.FindAllRecords("system_stats")
				require.NoError(t, err)
				require.Len(t, rows, 1)
				var stored system.Stats
				require.NoError(t, rows[0].UnmarshalJSONField("stats", &stored))
				require.Equal(t, full.Stats, stored)
				require.Equal(t, sys.Id, rows[0].GetString("system"))
				require.False(t, rows[0].GetDateTime("created").IsZero())

				// Fresh unavailable snapshot must replace every optional map without changing old history.
				minimal := system.CombinedData{Info: system.Info{AgentVersion: version, Uptime: 86410}, Stats: system.Stats{}}
				wire, err = cbor.Marshal(minimal)
				require.NoError(t, err)
				payloads <- wire
				require.NoError(t, sys.update())
				current, err = app.FindRecordById("systems", sys.Id)
				require.NoError(t, err)
				var public map[string]any
				require.NoError(t, current.UnmarshalJSONField("info", &public))
				require.Equal(t, float64(0), public["cpu"], "valid zero is a reading")
				for _, key := range []string{"g", "wf", "pu", "sv", "bat"} {
					require.NotContains(t, public, key)
				}
				firstRecordID := rows[0].Id
				rows, err = app.FindAllRecords("system_stats")
				require.NoError(t, err)
				require.Len(t, rows, 2)
				old, err := app.FindRecordById("system_stats", firstRecordID)
				require.NoError(t, err)
				var original system.Stats
				require.NoError(t, old.UnmarshalJSONField("stats", &original))
				require.Equal(t, full.Stats, original) // receipt time histories preserve the first sample
				require.Empty(t, sys.data.Stats.GPUData)
				require.Empty(t, sys.data.Stats.WiFi)
				require.Empty(t, sys.data.Stats.ZfsPools)

				// Malformed CBOR cannot become a new record or alter the previous snapshot.
				before, err := json.Marshal(sys.data)
				require.NoError(t, err)
				payloads <- []byte{0xff}
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				var invalid system.CombinedData
				err = transport.NewWebSocketTransport(wc).Request(ctx, common.GetData, common.DataRequestOptions{}, &invalid)
				cancel()
				require.Error(t, err)
				after, err := json.Marshal(sys.data)
				require.NoError(t, err)
				require.Equal(t, string(before), string(after))
				count, err := app.CountRecords("system_stats")
				require.NoError(t, err)
				require.EqualValues(t, 2, count)

				// Replace the carrying connection, then accept a fresh remote snapshot.
				wc.Close(nil)
				replacement, queue := connectRemoteTelemetryPeer(t, version)
				sys.WsConn = replacement
				queue <- wire
				require.NoError(t, sys.update())
				count, err = app.CountRecords("system_stats")
				require.NoError(t, err)
				require.EqualValues(t, 3, count)
			})
		}
	}
}
