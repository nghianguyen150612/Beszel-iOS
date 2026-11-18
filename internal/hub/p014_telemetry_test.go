//go:build testing

package hub_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/smart"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/entities/zfs"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	"github.com/henrygd/beszel/internal/testutil/telemetry"
	"github.com/lxzan/gws"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// This peer verifies the actual Hub signature and answers authenticated polls.
// It never calls a database write or a Hub CreateRecords testing helper.
type p014Agent struct {
	optional p014OptionalActions
	gws.BuiltinEventHandler
	t                  *testing.T
	token, fingerprint string
	key                ssh.PublicKey
	data               atomic.Pointer[system.CombinedData]
	polls              atomic.Int32
}

type p014OptionalActions struct {
	Smart    *smart.SmartDataResponse `json:"smart"`
	Zfs      *zfs.ZfsData             `json:"zfs"`
	Packages *system.PackageUpdates   `json:"packages"`
}

func (a *p014Agent) OnMessage(conn *gws.Conn, msg *gws.Message) {
	defer msg.Close()
	var req common.HubRequest[cbor.RawMessage]
	if err := cbor.Unmarshal(msg.Bytes(), &req); err != nil {
		a.t.Error(err)
		return
	}
	resp := common.AgentResponse{Id: req.Id}
	switch req.Action {
	case common.CheckFingerprint:
		var challenge common.FingerprintRequest
		if err := cbor.Unmarshal(req.Data, &challenge); err != nil {
			a.t.Error(err)
			return
		}
		if err := a.key.Verify([]byte(a.token), &ssh.Signature{Format: a.key.Type(), Blob: challenge.Signature}); err != nil {
			a.t.Error(err)
			return
		}
		resp.Fingerprint = &common.FingerprintResponse{Fingerprint: a.fingerprint}
	case common.GetData:
		resp.SystemData = a.data.Load()
		resp.Data, _ = cbor.Marshal(resp.SystemData)
		a.polls.Add(1)
	case common.GetSmartData:
		if a.optional.Smart != nil {
			resp.Data, _ = cbor.Marshal(a.optional.Smart)
		} else {
			resp.Error = "collector unavailable"
		}
	case common.GetZfsData:
		if a.optional.Zfs != nil {
			resp.Data, _ = cbor.Marshal(a.optional.Zfs)
		} else {
			resp.Error = "collector unavailable"
		}
	case common.GetPackageUpdates:
		if a.optional.Packages != nil {
			resp.Data, _ = cbor.Marshal(a.optional.Packages)
		} else {
			resp.Error = "collector unavailable"
		}
	case common.SyncNetworkMonitors:
		resp.Data, _ = cbor.Marshal(map[int]any{})
	default:
		resp.Error = "synthetic fixture does not implement this collector"
	}
	raw, err := cbor.Marshal(resp)
	if err == nil {
		err = conn.WriteMessage(gws.OpcodeBinary, raw)
	}
	if err != nil {
		a.t.Error(err)
	}
}

func TestP014AuthenticatedRemoteTelemetryAPI(t *testing.T) {
	t.Setenv("DISABLE_SSH", "true")
	t.Setenv("BESZEL_HUB_DISABLE_SSH", "true")
	t.Setenv("SHARE_ALL_SYSTEMS", "false")
	hub, handler := firstUserTestMux(t)
	defer hub.Cleanup()
	server := httptest.NewServer(handler)
	defer server.Close()
	signer, err := hub.GetSSHKey("")
	require.NoError(t, err)
	collection := func(name string) *core.Collection {
		c, e := hub.FindCachedCollectionByNameOrId(name)
		require.NoError(t, e)
		return c
	}
	get := func(path, token string, methods ...string) (int, []byte) {
		method := http.MethodGet
		if len(methods) > 0 {
			method = methods[0]
		}
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", token)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code, w.Body.Bytes()
	}
	for _, platform := range []string{"linux", "windows", "macos", "ios"} {
		t.Run(platform, func(t *testing.T) {
			f, err := telemetry.Load(platform)
			require.NoError(t, err)
			owner := core.NewRecord(collection("users"))
			owner.Id = f.OwnerID
			owner.Set("email", platform+"@p014.example")
			owner.SetPassword("p014-synthetic-password")
			owner.Set("role", "user")
			require.NoError(t, hub.Save(owner))
			token, err := owner.NewAuthToken()
			require.NoError(t, err)
			outsider, err := beszelTests.CreateUserWithRole(hub, platform+"-outsider@p014.example", "p014-synthetic-password", "user")
			require.NoError(t, err)
			otherToken, err := outsider.NewAuthToken()
			require.NoError(t, err)
			record := core.NewRecord(collection("systems"))
			record.Id = f.SystemID
			record.Load(map[string]any{"name": "p014-" + platform, "host": "127.0.0.1", "port": "45876", "users": []string{owner.Id}})
			require.NoError(t, hub.Save(record))
			agentToken := "p014-agent-token-" + platform
			fp := core.NewRecord(collection("fingerprints"))
			fp.Load(map[string]any{"system": record.Id, "token": agentToken})
			require.NoError(t, hub.SaveNoValidate(fp))
			peer := &p014Agent{t: t, token: agentToken, fingerprint: "p014-fingerprint-" + platform, key: signer.PublicKey()}
			actionsRaw, err := os.ReadFile(filepath.Join("testdata", "p014", platform+"-actions.json"))
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(actionsRaw, &peer.optional))
			peer.data.Store(&f.Data)
			dial := func() *gws.Conn {
				conn, _, err := gws.NewClient(peer, &gws.ClientOption{Addr: "ws" + strings.TrimPrefix(server.URL, "http") + "/api/beszel/agent-connect", RequestHeader: http.Header{"X-Token": {agentToken}, "X-Beszel": {"0.21.0"}}})
				require.NoError(t, err)
				go conn.ReadLoop()
				return conn
			}
			conn := dial()
			defer conn.NetConn().Close()
			path := "/api/collections/system_stats/records?filter=" + url.QueryEscape(fmt.Sprintf("system='%s'", record.Id))
			var history struct {
				Items []struct {
					ID     string          `json:"id"`
					System string          `json:"system"`
					Stats  json.RawMessage `json:"stats"`
				} `json:"items"`
			}
			require.Eventually(t, func() bool {
				code, body := get(path, token)
				if code != 200 {
					return false
				}
				return json.Unmarshal(body, &history) == nil && len(history.Items) > 0
			}, 10*time.Second, 10*time.Millisecond)
			require.Equal(t, int32(1), peer.polls.Load())
			require.Equal(t, record.Id, history.Items[0].System)
			var stats system.Stats
			require.NoError(t, json.Unmarshal(history.Items[0].Stats, &stats))
			expectedStats := f.Data.Stats
			expectedStats.MigrateDeprecatedMemoryFields()
			require.Equal(t, expectedStats, stats)
			var actual map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(history.Items[0].Stats, &actual))
			require.Equal(t, byte('['), actual["cpus"][0])
			code, body := get(path, otherToken)
			require.Equal(t, 200, code)
			var foreign struct {
				TotalItems int `json:"totalItems"`
			}
			require.NoError(t, json.Unmarshal(body, &foreign))
			require.Zero(t, foreign.TotalItems)
			code, _ = get("/api/collections/system_stats/records/"+history.Items[0].ID, otherToken)
			require.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, code)
			code, body = get("/api/collections/systems/records/"+record.Id, token)
			require.Equal(t, 200, code)
			var apiSystem struct {
				Info json.RawMessage `json:"info"`
			}
			require.NoError(t, json.Unmarshal(body, &apiSystem))
			snapshot := map[string]json.RawMessage{"info": apiSystem.Info, "stats": history.Items[0].Stats}
			list := func(collection string) []map[string]json.RawMessage {
				t.Helper()
				path := "/api/collections/" + collection + "/records?filter=" + url.QueryEscape(fmt.Sprintf("system='%s'", record.Id))
				code, body := get(path, token)
				require.Equal(t, 200, code)
				var result struct {
					Items []map[string]json.RawMessage `json:"items"`
				}
				require.NoError(t, json.Unmarshal(body, &result))
				_, other := get(path, otherToken)
				var denied struct {
					TotalItems int `json:"totalItems"`
				}
				require.NoError(t, json.Unmarshal(other, &denied))
				require.Zero(t, denied.TotalItems)
				return result.Items
			}
			for _, collection := range []string{"container_stats", "systemd_services"} {
				items := list(collection)
				expected := 0
				if collection == "container_stats" && len(f.Data.Containers) > 0 {
					expected = 1
				}
				if collection == "systemd_services" {
					expected = len(f.Data.SystemdServices)
				}
				require.Len(t, items, expected)
				if collection == "container_stats" && len(items) > 0 {
					snapshot["containers"] = items[0]["stats"]
				}
			}
			for _, action := range []struct {
				path, collection string
				supported        bool
			}{
				{"/api/beszel/smart/refresh", "smart_devices", peer.optional.Smart != nil},
				{"/api/beszel/zfs/refresh", "zfs_pools", peer.optional.Zfs != nil},
			} {
				path := action.path + "?system=" + record.Id
				code, _ := get(path, otherToken, http.MethodPost)
				require.Equal(t, 404, code)
				code, _ = get(path, token, http.MethodPost)
				if action.supported {
					require.Equal(t, 200, code)
					require.Len(t, list(action.collection), 1)
				} else {
					require.Equal(t, 500, code)
					require.Empty(t, list(action.collection))
				}
			}
			code, body = get("/api/beszel/package-updates?system="+record.Id, token)
			if peer.optional.Packages != nil {
				require.Equal(t, 200, code)
				var updates system.PackageUpdates
				require.NoError(t, json.Unmarshal(body, &updates))
				require.Equal(t, *peer.optional.Packages, updates)
				snapshot["packages"] = body
			} else {
				require.Equal(t, 500, code)
			}

			encoded, err := json.MarshalIndent(snapshot, "", "  ")
			require.NoError(t, err)
			golden := filepath.Join("testdata", "p014", platform+"-api.json")
			if os.Getenv("UPDATE_P014") == "1" {
				require.NoError(t, os.MkdirAll(filepath.Dir(golden), 0755))
				require.NoError(t, os.WriteFile(golden, append(encoded, '\n'), 0644))
			}
			want, err := os.ReadFile(golden)
			require.NoError(t, err)
			require.JSONEq(t, string(want), string(encoded))
			// Reauthenticate a new connection with a partial snapshot. It must not reuse optional fields.
			partial := &system.CombinedData{Stats: system.Stats{Cpu: 7}, Info: system.Info{Cpu: 7, AgentVersion: "0.21.0"}}
			peer.data.Store(partial)
			reconnect := dial()
			defer reconnect.NetConn().Close()
			require.Eventually(t, func() bool {
				_, body := get(path, token)
				return json.Unmarshal(body, &history) == nil && len(history.Items) == 2
			}, 10*time.Second, 10*time.Millisecond)
			for _, h := range history.Items {
				var s system.Stats
				require.NoError(t, json.Unmarshal(h.Stats, &s))
				if s.Cpu == 7 {
					require.Empty(t, s.GPUData)
					require.Empty(t, s.Temperatures)
					require.Empty(t, s.Battery)
					require.Empty(t, s.WiFi)
				}
			}
			stored, err := hub.FindRecordById("fingerprints", fp.Id)
			require.NoError(t, err)
			require.Equal(t, peer.fingerprint, stored.GetString("fingerprint"))
			require.Equal(t, agentToken, stored.GetString("token"))
			code, body = get("/api/collections/system_details/records/"+record.Id, token)
			require.Equal(t, 200, code)
			var details struct {
				Os   int    `json:"os"`
				Name string `json:"os_name"`
			}
			require.NoError(t, json.Unmarshal(body, &details))
			require.Equal(t, int(f.Data.Details.Os), details.Os)
			require.Equal(t, platform, details.Name)
			// Revoked membership must also revoke previously written history.
			current, err := hub.FindRecordById("systems", record.Id)
			require.NoError(t, err)
			current.Set("users", []string{outsider.Id})
			require.NoError(t, hub.SaveNoValidate(current))
			code, body = get(path, token)
			require.Equal(t, 200, code)
			require.NoError(t, json.Unmarshal(body, &foreign))
			require.Zero(t, foreign.TotalItems)

		})
	}
}
