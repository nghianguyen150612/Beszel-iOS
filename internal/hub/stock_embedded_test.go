//go:build testing

package hub

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/agent"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/hub/transport"
	"github.com/pocketbase/dbx"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// This uses the real Hub registration handler and a disposable PocketBase DB.
// The success control deliberately uses synthetic data and a separate system.
func TestStockEmbeddedActualHub(t *testing.T) {
	hub, app, err := createTestHub(t)
	require.NoError(t, err)
	defer cleanupTestHub(hub, app)
	signer, err := hub.GetSSHKey("")
	require.NoError(t, err)
	user, err := createTestUser(app)
	require.NoError(t, err)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	host, port, err := net.SplitHostPort(address)
	require.NoError(t, err)
	makeSystem := func(t *testing.T, token string) (string, string) {
		t.Helper()
		rec, err := createTestRecord(app, "systems", map[string]any{"name": token, "host": host, "port": port, "status": "pending", "users": []string{user.Id}})
		require.NoError(t, err)
		fp, err := createTestRecord(app, "fingerprints", map[string]any{"system": rec.Id, "token": token})
		require.NoError(t, err)
		return rec.Id, fp.Id
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/beszel/agent-connect" {
			http.NotFound(w, r)
			return
		}
		request := agentConnectRequest{hub: hub, req: r, res: w}
		_ = request.agentConnect()
	}))
	defer server.Close()
	t.Run("unavailable remains an error and creates no records", func(t *testing.T) {
		const token = "disposable-stock-token"
		id, fpID := makeSystem(t, token)
		requested := make(chan struct{}, 2)
		var runtime agent.EmbeddedRuntime
		cfg := agent.EmbeddedConfig{
			HubURL: server.URL, Token: "invalid-token", HubKey: string(ssh.MarshalAuthorizedKey(signer.PublicKey())),
			DataDirectory: t.TempDir(), AllowInsecureDevelopment: true, ReconnectInterval: 500 * time.Millisecond,
			Collect: func(common.DataRequestOptions) (*system.CombinedData, error) {
				select {
				case requested <- struct{}{}:
				default:
				}
				return nil, errors.New("stock system-wide fields unavailable")
			},
		}
		require.NoError(t, runtime.Start(cfg))
		defer runtime.Stop()
		require.Eventually(t, func() bool { return runtime.Status().State == "authentication_error" }, 5*time.Second, 20*time.Millisecond)
		require.Empty(t, requested)
		runtime.Stop()

		cfg.Token = token
		require.NoError(t, runtime.Start(cfg))
		awaitRequest := func() {
			t.Helper()
			select {
			case <-requested:
			case <-time.After(10 * time.Second):
				t.Fatal("real Hub never requested metrics", runtime.Status())
			}
		}
		awaitRequest()
		first := runtime.Status().LastConnected
		require.False(t, first.IsZero())
		saved, err := app.FindRecordById("fingerprints", fpID)
		require.NoError(t, err)
		fingerprint := saved.GetString("fingerprint")
		require.NotEmpty(t, fingerprint)
		awaitRequest() // The current Hub closes the socket after a GetData error.
		require.True(t, runtime.Status().LastConnected.After(first))
		require.Eventually(t, func() bool {
			record, err := app.FindRecordById("systems", id)
			return err == nil && record.GetString("status") == "down"
		}, 5*time.Second, 20*time.Millisecond)
		saved, err = app.FindRecordById("fingerprints", fpID)
		require.NoError(t, err)
		require.Equal(t, fingerprint, saved.GetString("fingerprint"))
		require.True(t, runtime.Status().LastMetrics.IsZero())
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if conn != nil {
			conn.Close()
		}
		require.Error(t, err, "embedded Agent must never expose the configured SSH port")
		runtime.Stop()
		require.Equal(t, "disconnected", runtime.Status().State)
		count, err := app.CountRecords("system_stats", dbx.HashExp{"system": id})
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("synthetic partial success stores misleading zeros", func(t *testing.T) {
		const token = "disposable-partial-token"
		id, _ := makeSystem(t, token)
		var runtime agent.EmbeddedRuntime
		require.NoError(t, runtime.Start(agent.EmbeddedConfig{
			HubURL: server.URL, Token: token, HubKey: string(ssh.MarshalAuthorizedKey(signer.PublicKey())),
			DataDirectory: t.TempDir(), AllowInsecureDevelopment: true,
			Collect: func(common.DataRequestOptions) (*system.CombinedData, error) {
				battery := system.Battery{0, 4} // A measured zero with a nonzero charge state.
				return &system.CombinedData{
					Stats: system.Stats{Mem: 4, Battery: battery},
					Info:  system.Info{AgentVersion: "test", Battery: battery},
				}, nil
			},
		}))
		defer runtime.Stop()

		var stats map[string]json.RawMessage
		require.Eventually(t, func() bool {
			records, err := app.FindRecordsByFilter("system_stats", "system={:id} && type='1m'", "-created", 1, 0, dbx.Params{"id": id})
			return err == nil && len(records) == 1 && json.Unmarshal([]byte(records[0].GetString("stats")), &stats) == nil
		}, 10*time.Second, 20*time.Millisecond)
		for _, key := range []string{"cpu", "mu", "mp", "mb", "d", "du", "dp"} {
			require.JSONEq(t, "0", string(stats[key]), key)
		}
		require.JSONEq(t, "4", string(stats["m"]))
		require.JSONEq(t, "[0,4]", string(stats["bat"]))
		record, err := app.FindRecordById("systems", id)
		require.NoError(t, err)
		var info map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(record.GetString("info")), &info))
		for _, key := range []string{"cpu", "mp", "dp", "bb", "u"} {
			require.JSONEq(t, "0", string(info[key]), key)
		}
		require.JSONEq(t, "[0,4]", string(info["bat"]))
	})
}

func TestStockEmbeddedActualHubCBORPresence(t *testing.T) {
	// Candidate extension fields alone cannot change how a legacy Hub decodes
	// the original data structure. The flags here are synthetic, not allocated keys.
	for _, cpu := range []map[int]any{{}, {0: nil}} {
		wire, err := cbor.Marshal(map[int]any{0: cpu, 1: map[int]any{6: nil}, 99: []string{"battery"}})
		require.NoError(t, err)
		response := common.AgentResponse{Data: wire}
		var fresh system.CombinedData
		require.NoError(t, transport.UnmarshalResponse(response, common.GetData, &fresh))
		require.Zero(t, fresh.Stats.Cpu)
		require.Zero(t, fresh.Info.Cpu)
		reused := system.CombinedData{Stats: system.Stats{Cpu: 71}, Info: system.Info{Cpu: 71}}
		require.NoError(t, transport.UnmarshalResponse(response, common.GetData, &reused))
		require.Equal(t, 71.0, reused.Stats.Cpu)
		require.Equal(t, 71.0, reused.Info.Cpu)
	}

	wire, err := cbor.Marshal(map[int]any{0: map[int]any{0: 0}, 1: map[int]any{6: 0}})
	require.NoError(t, err)
	reused := system.CombinedData{Stats: system.Stats{Cpu: 71}, Info: system.Info{Cpu: 71}}
	require.NoError(t, transport.UnmarshalResponse(common.AgentResponse{Data: wire}, common.GetData, &reused))
	require.Zero(t, reused.Stats.Cpu)
	require.Zero(t, reused.Info.Cpu)
}
