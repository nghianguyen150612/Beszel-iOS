//go:build testing

package hub

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/henrygd/beszel/agent"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// This uses the real Hub registration handler and a disposable PocketBase DB.
// It proves authentication and stats requests, not acceptance of stock metrics.
func TestStockEmbeddedActualHub(t *testing.T) {
	hub, app, err := createTestHub(t)
	require.NoError(t, err)
	defer cleanupTestHub(hub, app)
	signer, err := hub.GetSSHKey("")
	require.NoError(t, err)
	user, err := createTestUser(app)
	require.NoError(t, err)
	rec, err := createTestRecord(app, "systems", map[string]any{"name": "stock-test", "host": "localhost", "port": "45876", "status": "pending", "users": []string{user.Id}})
	require.NoError(t, err)
	fp, err := createTestRecord(app, "fingerprints", map[string]any{"system": rec.Id, "token": "disposable-stock-token"})
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/beszel/agent-connect" {
			http.NotFound(w, r)
			return
		}
		request := agentConnectRequest{hub: hub, req: r, res: w}
		_ = request.agentConnect()
	}))
	defer server.Close()
	requested := make(chan struct{}, 1)
	var runtime agent.EmbeddedRuntime
	err = runtime.Start(agent.EmbeddedConfig{HubURL: server.URL, Token: "disposable-stock-token", HubKey: string(ssh.MarshalAuthorizedKey(signer.PublicKey())), DataDirectory: t.TempDir(), AllowInsecureDevelopment: true,
		Collect: func(common.DataRequestOptions) (*system.CombinedData, error) {
			select {
			case requested <- struct{}{}:
			default:
			}
			return nil, errors.New("stock system-wide fields unavailable")
		},
	})
	require.NoError(t, err)
	defer runtime.Stop()
	select {
	case <-requested:
	case <-time.After(10 * time.Second):
		t.Fatal("real Hub never requested metrics", runtime.Status())
	}
	require.False(t, runtime.Status().LastConnected.IsZero())
	saved, err := app.FindRecordById("fingerprints", fp.Id)
	require.NoError(t, err)
	require.NotEmpty(t, saved.GetString("fingerprint"))
	require.True(t, runtime.Status().LastMetrics.IsZero(), "blocked stats cannot be marked accepted")
}
