//go:build testing

package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/henrygd/beszel/internal/hub/transport"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Exercise real health probes through the production Hub transports and Agent
// request registry, reusing P009's authenticated disposable loopback fixtures.
func TestP010MonitorProtocolOverWSAndSSH(t *testing.T) {
	h := newP009Hub(t)
	h.a.monitorManager.Stop()
	h.a.monitorManager = newMonitorManager(nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(503)
		} else {
			w.WriteHeader(204)
		}
	}))
	defer server.Close()
	tlsServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer tlsServer.Close()
	exercise := func(tr transport.Transport) {
		t.Helper()
		var resp monitor.SyncResponse
		cfg := monitor.Config{ID: "p010-stable", Protocol: "http", Target: server.URL, Interval: 3600}
		for _, req := range []monitor.SyncRequest{
			{Action: monitor.SyncActionReplace, Configs: []monitor.Config{cfg}},
			{Action: monitor.SyncActionReplace, Configs: []monitor.Config{cfg}},
			{Action: monitor.SyncActionUpsert, Config: cfg, RunNow: true},
		} {
			require.NoError(t, tr.Request(t.Context(), common.SyncNetworkMonitors, req, &resp))
		}
		require.Equal(t, int64(1), resp.Result.TotalCount)
		require.Equal(t, int64(1), resp.Result.SuccessCount)
		require.Zero(t, resp.Result.PacketLoss)
		require.Positive(t, resp.Result.LastProbeAt)
		cfg.Target = server.URL + "/fail"
		require.NoError(t, tr.Request(t.Context(), common.SyncNetworkMonitors, monitor.SyncRequest{Action: monitor.SyncActionUpsert, Config: cfg, RunNow: true}, &resp))
		require.Equal(t, 100.0, resp.Result.PacketLoss)
		require.Zero(t, resp.Result.SuccessCount)
		require.Zero(t, resp.Result.ResponseSum)
		for _, skip := range []bool{false, true, false} {
			cfg.Target = tlsServer.URL
			cfg.SkipTLSVerify = skip
			require.NoError(t, tr.Request(t.Context(), common.SyncNetworkMonitors, monitor.SyncRequest{Action: monitor.SyncActionUpsert, Config: cfg, RunNow: true}, &resp))
			require.NotNil(t, resp.Result.Cert)
			if skip {
				require.Zero(t, resp.Result.PacketLoss)
			} else {
				require.Equal(t, 100.0, resp.Result.PacketLoss)
			}
			encoded, err := json.Marshal(resp)
			require.NoError(t, err)
			var decoded monitor.SyncResponse
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Equal(t, resp, decoded)
		}
		require.NoError(t, tr.Request(t.Context(), common.SyncNetworkMonitors, monitor.SyncRequest{Action: monitor.SyncActionDelete, Config: cfg}, &resp))
		require.Empty(t, h.a.monitorManager.GetResults(1000))
		require.NoError(t, tr.Request(t.Context(), common.SyncNetworkMonitors, monitor.SyncRequest{Action: monitor.SyncActionReplace}, &resp))
	}
	wc, _ := h.connect(t)
	h.authenticate(t, wc)
	exercise(transport.NewWebSocketTransport(wc))
	h.c.Close()
	t.Setenv("BESZEL_AGENT_DISABLE_SSH", "false")
	serverSSH, ln, err := h.a.prepareSSHServer(ServerOptions{Network: "tcp", Addr: "127.0.0.1:0", Keys: h.a.keys})
	require.NoError(t, err)
	_, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	hostSigner, err := ssh.NewSignerFromKey(private)
	require.NoError(t, err)
	serverSSH.AddHostKey(hostSigner)
	served := make(chan struct{})
	go func() { _ = h.a.serveSSHServer(serverSSH, ln); close(served) }()
	t.Cleanup(func() {
		h.a.StopServer()
		select {
		case <-served:
		case <-time.After(time.Second):
			t.Error("SSH shutdown stalled")
		}
	})
	host, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	tr := transport.NewSSHTransport(transport.SSHTransportConfig{Host: host, Port: port, Config: &ssh.ClientConfig{User: "fixture", ClientVersion: "SSH-2.0-beszel_0.21.0", Auth: []ssh.AuthMethod{ssh.PublicKeys(h.signer)}, HostKeyCallback: ssh.FixedHostKey(hostSigner.PublicKey()), Timeout: time.Second}})
	defer tr.Close()
	exercise(tr)
}

func TestP010MonitorCBORFieldCompatibility(t *testing.T) {
	cfg := monitor.Config{ID: "stable", Target: "https://fixture", Protocol: "http", Interval: 60, SkipTLSVerify: true}
	encoded, err := cbor.Marshal(cfg)
	require.NoError(t, err)
	var fields map[int]any
	require.NoError(t, cbor.Unmarshal(encoded, &fields))
	require.Equal(t, true, fields[6])
	require.Equal(t, "stable", fields[0])
	// An old config has no TLS key: secure default; future fields are ignored.
	delete(fields, 6)
	fields[99] = "future"
	encoded, err = cbor.Marshal(fields)
	require.NoError(t, err)
	var decoded monitor.Config
	require.NoError(t, cbor.Unmarshal(encoded, &decoded))
	require.False(t, decoded.SkipTLSVerify)
	require.Equal(t, cfg.ID, decoded.ID)
	result := monitor.Result{PacketLoss: 100, PacketLoss1h: 100, LastProbeAt: 123, SampleCount: 1, TotalCount: 1, Cert: &monitor.CertInfo{Expires: 456, Issuer: "fixture"}}
	encoded, err = cbor.Marshal(result)
	require.NoError(t, err)
	require.NoError(t, cbor.Unmarshal(encoded, &fields))
	for _, key := range []int{8, 10, 11, 12} {
		require.Contains(t, fields, key)
	}
	var roundtrip monitor.Result
	require.NoError(t, cbor.Unmarshal(encoded, &roundtrip))
	require.Equal(t, result, roundtrip)
}
