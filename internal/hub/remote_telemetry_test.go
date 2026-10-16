//go:build testing

package hub_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/henrygd/beszel/internal/testutil/telemetry"
	"github.com/lxzan/gws"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// This endpoint is a source-derived protocol peer, not an executed remote Agent.
// It verifies the real Hub signature and answers the real Hub request manager.
type telemetryAgent struct {
	gws.BuiltinEventHandler
	fixture telemetry.Fixture
	key     ssh.PublicKey
	token   string
	errors  chan error
	syncs   chan monitor.SyncRequest
}

func (a *telemetryAgent) OnMessage(conn *gws.Conn, message *gws.Message) {
	defer message.Close()
	var req common.HubRequest[cbor.RawMessage]
	if err := cbor.Unmarshal(message.Bytes(), &req); err != nil {
		a.errors <- err
		return
	}
	resp := common.AgentResponse{Id: req.Id}
	switch req.Action {
	case common.CheckFingerprint:
		var challenge common.FingerprintRequest
		if err := cbor.Unmarshal(req.Data, &challenge); err != nil {
			a.errors <- err
			return
		}
		if err := a.key.Verify([]byte(a.token), &ssh.Signature{Format: a.key.Type(), Blob: challenge.Signature}); err != nil {
			a.errors <- err
			return
		}
		resp.Fingerprint = &common.FingerprintResponse{Fingerprint: "synthetic-" + a.fixture.Name, Hostname: a.fixture.Data.Details.Hostname}
	case common.GetData:
		resp.Data, _ = cbor.Marshal(a.fixture.Data)
	case common.SyncNetworkMonitors:
		var sync monitor.SyncRequest
		if err := cbor.Unmarshal(req.Data, &sync); err != nil {
			a.errors <- err
			return
		}
		a.syncs <- sync
		resp.Data, _ = cbor.Marshal(monitor.SyncResponse{})
	case common.GetPackageUpdates:
		resp.Data, _ = cbor.Marshal(a.fixture.Packages)
	default:
		resp.Error = "fixture action unavailable"
	}
	wire, err := cbor.Marshal(resp)
	if err == nil {
		err = conn.WriteMessage(gws.OpcodeBinary, wire)
	}
	if err != nil {
		select {
		case a.errors <- err:
		default:
		}
	}
}

func telemetryAPI(t *testing.T, mux http.Handler, token, path string, status int) map[string]any {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", token)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	require.Equal(t, status, res.Code, "%s: %s", path, res.Body.String())
	var data map[string]any
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &data))
	return data
}

func TestP014AuthenticatedRemoteTelemetryAPI(t *testing.T) {
	t.Setenv("BESZEL_HUB_SHARE_ALL_SYSTEMS", "false")
	t.Setenv("SHARE_ALL_SYSTEMS", "false")
	t.Setenv("BESZEL_HUB_DISABLE_SSH", "true")
	snapshots := map[string]any{}
	for _, fixture := range telemetry.All() {
		t.Run(fixture.Name, func(t *testing.T) {
			h, mux := firstUserTestMux(t)
			defer h.Cleanup()
			save := func(collection, id string, fields map[string]any) *core.Record {
				t.Helper()
				col, err := h.FindCollectionByNameOrId(collection)
				require.NoError(t, err)
				r := core.NewRecord(col)
				r.Id = id
				r.Load(fields)
				require.NoError(t, h.Save(r))
				return r
			}
			owner := save("users", fixture.UserID, map[string]any{"email": fixture.Name + "@example.invalid", "password": "synthetic-password-123", "role": "user"})
			stranger := save("users", "strangeruser001", map[string]any{"email": "stranger@example.invalid", "password": "synthetic-password-123", "role": "user"})
			token, err := owner.NewAuthToken()
			require.NoError(t, err)
			otherToken, err := stranger.NewAuthToken()
			require.NoError(t, err)
			// Paused prevents acquisition while all configuration and subscriptions are prepared.
			record := save("systems", fixture.SystemID, map[string]any{"name": "synthetic-" + fixture.Name, "host": "127.0.0.1", "port": "1", "users": []string{owner.Id}, "status": "paused"})
			record.Set("status", "paused")
			require.NoError(t, h.Save(record))
			save("fingerprints", "fingerprint0001", map[string]any{"system": record.Id, "token": "synthetic-token-" + fixture.Name, "fingerprint": "synthetic-" + fixture.Name})
			results := make(map[string]monitor.Result)
			for _, config := range fixture.Configs {
				r := save("network_monitors", config.ID, map[string]any{"system": record.Id, "target": config.Target, "protocol": config.Protocol, "port": config.Port, "interval": config.Interval, "enabled": true})
				results[r.Id] = fixture.Data.Monitors[config.ID]
			}
			fixture.Data.Monitors = results

			alert := save("alerts", "cpualert0000001", map[string]any{"system": record.Id, "user": owner.Id, "name": "CPU", "value": 20, "min": 1})
			record.Set("status", "pending")
			require.NoError(t, h.Save(record))
			signer, err := h.GetSSHKey("")
			require.NoError(t, err)
			server := httptest.NewServer(mux)
			defer server.Close()
			peer := &telemetryAgent{fixture: fixture, key: signer.PublicKey(), token: "synthetic-token-" + fixture.Name, errors: make(chan error, 16), syncs: make(chan monitor.SyncRequest, 16)}
			conn, _, err := gws.NewClient(peer, &gws.ClientOption{Addr: "ws" + strings.TrimPrefix(server.URL, "http") + "/api/beszel/agent-connect", RequestHeader: http.Header{"X-Token": {peer.token}, "X-Beszel": {"0.21.0"}}})
			require.NoError(t, err)
			done := make(chan struct{})
			go func() { defer close(done); conn.ReadLoop() }()
			defer func() { _ = conn.NetConn().Close(); <-done }()
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				rows, err := h.FindRecordsByFilter("system_stats", "system={:id}", "", 0, 0, map[string]any{"id": record.Id})
				assert.NoError(c, err)
				assert.Len(c, rows, 1)
			}, 5*time.Second, 10*time.Millisecond)
			sys, err := h.GetSystemManager().GetSystem(record.Id)
			require.NoError(t, err)
			sys.StopUpdater()
			h.GetSystemManager().WaitForUpdatersForTesting()
			select {
			case err := <-peer.errors:
				t.Fatal(err)
			default:
			}
			select {
			case sync := <-peer.syncs:
				require.Equal(t, monitor.SyncActionReplace, sync.Action)
				require.Len(t, sync.Configs, 5)
			case <-time.After(time.Second):
				t.Fatal("authenticated connection did not replay monitor config")
			}
			current := telemetryAPI(t, mux, token, "/api/collections/systems/records/"+record.Id, 200)
			require.Equal(t, "up", current["status"])
			require.Equal(t, float64(25), current["info"].(map[string]any)["cpu"])
			telemetryAPI(t, mux, otherToken, "/api/collections/systems/records/"+record.Id, 404)
			details := telemetryAPI(t, mux, token, "/api/collections/system_details/records/"+record.Id, 200)
			require.Equal(t, fixture.Data.Details.OsName, details["os_name"])
			require.EqualValues(t, fixture.Data.Details.Os, details["os"])
			collections := map[string]any{}
			for _, name := range []string{"system_stats", "container_stats", "containers", "systemd_services", "network_monitors", "network_monitor_stats"} {
				path := "/api/collections/" + name + "/records?sort=id&perPage=100&filter=" + url.QueryEscape("system='"+record.Id+"'")
				list := telemetryAPI(t, mux, token, path, 200)
				denied := telemetryAPI(t, mux, otherToken, path, 200)
				require.Empty(t, denied["items"])
				items := list["items"].([]any)
				if name == "system_stats" {
					require.Len(t, items, 1)
					require.Equal(t, float64(25), items[0].(map[string]any)["stats"].(map[string]any)["cpu"])
				}
				if name == "network_monitor_stats" {
					require.Len(t, items, 5)
					for _, item := range items {
						r := item.(map[string]any)
						require.Equal(t, float64(4), r["total_count"])
						require.Equal(t, float64(3), r["success_count"])
						require.Equal(t, float64(4500), r["res_sum"])
					}
				}
				for _, item := range items {
					r := item.(map[string]any)
					for _, key := range []string{"id", "created", "updated", "collectionId", "collectionName"} {
						delete(r, key)
					}
				}
				collections[name] = items
			}
			// Reuse exactly the served JSON in frontend contract tests; strip only record metadata.
			for _, key := range []string{"created", "updated", "collectionId", "collectionName"} {
				delete(current, key)
				delete(details, key)
			}
			snapshots[fixture.Name] = map[string]any{"system": current, "details": details, "collections": collections}
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				r, err := h.FindRecordById("alerts", alert.Id)
				assert.NoError(c, err)
				assert.True(c, r.GetBool("triggered"))
			}, 2*time.Second, 10*time.Millisecond)
			// A second snapshot exercises the same persistence path and duplicate-monitor policy.
			_, err = sys.CreateRecords(&fixture.Data)
			require.NoError(t, err)
			n, err := h.CountRecords("network_monitor_stats")
			require.NoError(t, err)
			require.EqualValues(t, 5, n)
			n, err = h.CountRecords("system_stats")
			require.NoError(t, err)
			require.EqualValues(t, 2, n, "system snapshots use Hub receipt time, without Agent sample deduplication")
			require.NoError(t, h.Delete(record))
			n, err = h.CountRecords("system_stats")
			require.NoError(t, err)
			require.Zero(t, n)
			n, err = h.CountRecords("network_monitor_stats")
			require.NoError(t, err)
			require.Zero(t, n)
		})
	}
	b, err := json.MarshalIndent(snapshots, "", "  ")
	require.NoError(t, err)
	path := filepath.Join("..", "site", "src", "lib", "testdata", "remote-telemetry.json")
	if os.Getenv("UPDATE_REMOTE_TELEMETRY_FIXTURES") == "1" {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		require.NoError(t, os.WriteFile(path, append(b, '\n'), 0644))
	}
	golden, err := os.ReadFile(path)
	require.NoError(t, err)
	require.JSONEq(t, string(golden), string(b), "regenerate explicitly after reviewing actual API changes")
}
