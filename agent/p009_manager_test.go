//go:build testing

package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/henrygd/beszel/internal/hub/transport"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestP009ManagerShutdownDuringSetup(t *testing.T) {
	a := createTestAgent(t)
	accepted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(accepted); <-r.Context().Done() }))
	defer server.Close()
	t.Setenv("BESZEL_AGENT_HUB_URL", server.URL)
	t.Setenv("BESZEL_AGENT_TOKEN", "synthetic")
	t.Setenv("BESZEL_AGENT_DISABLE_SSH", "true")
	cm := a.connectionManager
	done := make(chan error, 1)
	go func() { done <- cm.Start(ServerOptions{}) }()
	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("setup did not start")
	}
	cm.cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("manager shutdown blocked on setup")
	}
	require.False(t, cm.isConnectingNow())
	require.Nil(t, cm.wsClient.getConn())
	cm.launchConnect()
	require.False(t, cm.isConnectingNow(), "shutdown forbids further attempts")
}
func TestP009ManagerRetriesAsynchronousFailure(t *testing.T) {
	h := newP009Hub(t)
	cm := h.a.connectionManager
	done := make(chan error, 1)
	go func() { done <- cm.Start(ServerOptions{Keys: h.a.keys}) }()
	t.Cleanup(func() {
		cm.cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("manager did not stop")
		}
	})
	// TCP and upgrade succeed, but the first endpoint vanishes before auth.
	select {
	case <-h.connections:
		raw := <-h.sockets
		require.NoError(t, raw.NetConn().Close())
	case <-time.After(2 * time.Second):
		t.Fatal("first attempt missing")
	}
	// The production retry interval is ten seconds, without a shortened test clock.
	select {
	case wc := <-h.connections:
		<-h.sockets
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := wc.GetFingerprint(ctx, "p009-synthetic", h.signer, false)
		require.NoError(t, err)
	case <-time.After(13 * time.Second):
		t.Fatal("async handshake failure stopped retries")
	}
	require.Eventually(t, func() bool { return cm.getState() == WebSocketConnected }, time.Second, time.Millisecond)
	require.False(t, cm.isConnectingNow())
}
func TestP009ConcurrentConnectClose(t *testing.T) {
	h := newP009Hub(t)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { _ = h.c.ConnectContext(t.Context()) })
		wg.Go(h.c.Close)
	}
	wg.Wait()
	h.c.Close()
	require.Nil(t, h.c.getConn())
	require.False(t, h.c.isVerified())
	for len(h.connections) > 0 {
		<-h.connections
	}
	for len(h.sockets) > 0 {
		<-h.sockets
	}
	wc, _ := h.connect(t)
	h.authenticate(t, wc)
}
func TestP009MonitorTransportConvergence(t *testing.T) {
	h := newP009Hub(t)
	t.Setenv("BESZEL_AGENT_DISABLE_SSH", "false")
	// An inert injected probe avoids any network-monitor feature validation.
	h.a.monitorManager.Stop()
	h.a.monitorManager = newMonitorManagerWithProbe(func(context.Context, monitor.Config) (int64, error) { return 0, nil })
	cfg := monitor.Config{ID: "p009-stable", Protocol: "noop", Target: "synthetic", Interval: 60}
	syncConfig := func(tr transport.Transport, configs []monitor.Config) {
		t.Helper()
		var resp monitor.SyncResponse
		require.NoError(t, tr.Request(t.Context(), common.SyncNetworkMonitors, monitor.SyncRequest{Action: monitor.SyncActionReplace, Configs: configs}, &resp))
	}
	// Start the real Agent SSH server, pin its generated test host key, and use
	// the actual Hub SSHTransport. No insecure test callback is needed.
	server, ln, err := h.a.prepareSSHServer(ServerOptions{Network: "tcp", Addr: "127.0.0.1:0", Keys: h.a.keys})
	require.NoError(t, err)

	// Gliderlabs initializes the host signer inside ensureHandlers; obtain a
	// dedicated ephemeral signer first so both sides use a known test identity.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	hostSigner, err := ssh.NewSignerFromKey(priv)
	require.NoError(t, err)
	// Host keys must be installed before the first connection is accepted.
	server.AddHostKey(hostSigner)
	served := make(chan struct{})
	go func() { _ = h.a.serveSSHServer(server, ln); close(served) }()
	t.Cleanup(func() {
		h.a.StopServer()
		select {
		case <-served:
		case <-time.After(2 * time.Second):
			t.Error("SSH listener cleanup did not finish")
		}
	})
	host, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	tr := transport.NewSSHTransport(transport.SSHTransportConfig{Host: host, Port: port, Config: &ssh.ClientConfig{User: "u", ClientVersion: "SSH-2.0-beszel_0.21.0", Auth: []ssh.AuthMethod{ssh.PublicKeys(h.signer)}, HostKeyCallback: ssh.FixedHostKey(hostSigner.PublicKey()), Timeout: time.Second}})
	defer tr.Close()
	syncConfig(tr, []monitor.Config{cfg})
	h.a.monitorManager.mu.RLock()
	task := h.a.monitorManager.monitors[cfg.ID]
	h.a.monitorManager.mu.RUnlock()
	require.NotNil(t, task)
	syncConfig(tr, []monitor.Config{cfg})
	h.a.monitorManager.mu.RLock()
	duplicateTask := h.a.monitorManager.monitors[cfg.ID]
	h.a.monitorManager.mu.RUnlock()
	require.Same(t, task, duplicateTask)
	tr.Close()
	syncConfig(tr, []monitor.Config{cfg})
	wc, _ := h.connect(t)
	h.authenticate(t, wc)
	wsTr := transport.NewWebSocketTransport(wc)
	cfg.Target = "updated"
	syncConfig(wsTr, []monitor.Config{cfg})
	h.a.monitorManager.mu.RLock()
	count := len(h.a.monitorManager.monitors)
	updatedTask := h.a.monitorManager.monitors[cfg.ID]
	h.a.monitorManager.mu.RUnlock()
	require.Equal(t, 1, count)
	require.NotNil(t, updatedTask)
	require.Equal(t, "updated", updatedTask.config.Target)
	syncConfig(wsTr, nil)
	h.a.monitorManager.mu.RLock()
	count = len(h.a.monitorManager.monitors)
	h.a.monitorManager.mu.RUnlock()
	require.Zero(t, count)
	h.c.Close()
	wc, _ = h.connect(t)
	h.authenticate(t, wc)
	syncConfig(transport.NewWebSocketTransport(wc), nil)
}
