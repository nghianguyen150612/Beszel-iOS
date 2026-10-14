//go:build testing

package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"github.com/blang/semver"
	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/common"
	esystem "github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/hub/transport"
	hubws "github.com/henrygd/beszel/internal/hub/ws"
	"golang.org/x/crypto/ssh"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/lxzan/gws"
	"github.com/stretchr/testify/require"
)

func TestP009WSSPreservesTLS(t *testing.T) {
	u, _ := url.Parse("wss://localhost:1234")
	c := &WebSocketClient{hubURL: u}
	require.Equal(t, "wss://localhost:1234/api/beszel/agent-connect", c.getOptions().Addr)
}
func TestP009CloseCallbackCannotBlock(t *testing.T) {
	a := createTestAgent(t)
	cm := a.connectionManager
	cm.eventChan = make(chan ConnectionEvent, 1)
	cm.eventChan <- WebSocketConnect
	conn := &gws.Conn{}
	c := &WebSocketClient{agent: a, Conn: conn}
	done := make(chan struct{})
	go func() { c.OnClose(conn, nil); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		<-cm.eventChan
		<-done
		t.Fatal("close callback blocked on unconsumed event")
	}
}

// p009Hub uses production Hub WebSocket routing and the actual Agent handlers.
// Faults are injected only at the loopback endpoint/socket boundary.
type p009Hub struct {
	a           *Agent
	c           *WebSocketClient
	connections chan *hubws.WsConn
	sockets     chan *gws.Conn
	signer      ssh.Signer
	server      *httptest.Server
	reject      atomic.Bool
	pong        chan struct{}
}

func newP009Hub(t *testing.T) *p009Hub {
	t.Helper()
	h := &p009Hub{a: createTestAgent(t), connections: make(chan *hubws.WsConn, 64), sockets: make(chan *gws.Conn, 64)}
	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	h.signer, err = ssh.NewSignerFromKey(key)
	require.NoError(t, err)
	h.a.keys = []ssh.PublicKey{h.signer.PublicKey()}
	h.pong = make(chan struct{}, 64)
	upgrader := gws.NewUpgrader(&p009HubHandler{pong: h.pong}, nil)
	h.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.reject.Load() {
			http.Error(w, "injected rejection", http.StatusServiceUnavailable)
			return
		}
		if r.Header.Get("X-Token") != "p009-synthetic" || r.Header.Get("X-Beszel") != beszel.Version {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			return
		}
		wc := hubws.NewWsConnection(conn, semver.MustParse(beszel.Version))
		conn.Session().Store("wsConn", wc)
		h.connections <- wc
		h.sockets <- conn
		conn.ReadLoop()
	}))
	t.Setenv("BESZEL_AGENT_HUB_URL", h.server.URL)
	t.Setenv("BESZEL_AGENT_TOKEN", "p009-synthetic")
	t.Setenv("BESZEL_AGENT_CA_CERT_FILE", "")
	t.Setenv("BESZEL_AGENT_DISABLE_SSH", "true")
	h.c, err = newWebSocketClient(h.a)
	require.NoError(t, err)
	h.a.connectionManager.wsClient = h.c
	t.Cleanup(func() { h.c.Close(); h.a.monitorManager.Stop(); h.server.Close() })
	return h
}
func (h *p009Hub) connect(t *testing.T) (*hubws.WsConn, *gws.Conn) {
	t.Helper()
	require.NoError(t, h.c.ConnectContext(t.Context()))
	select {
	case wc := <-h.connections:
		raw := <-h.sockets
		require.NoError(t, raw.WritePing(nil))
		select {
		case <-h.pong:
		case <-time.After(time.Second):
			t.Fatal("read loop did not start")
		}
		return wc, raw
	case <-time.After(3 * time.Second):
		t.Fatal("upgrade not observed")
		return nil, nil
	}
}
func (h *p009Hub) authenticate(t *testing.T, wc *hubws.WsConn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	fp, err := wc.GetFingerprint(ctx, "p009-synthetic", h.signer, false)
	require.NoError(t, err)
	require.Equal(t, h.c.fingerprint, fp.Fingerprint)
	require.Eventually(t, h.c.isVerified, time.Second, time.Millisecond)
	h.a.connectionManager.handleWebSocketChange()
	require.Equal(t, WebSocketConnected, h.a.connectionManager.getState())
}
func TestP009WebSocketAuthAndFaultRecovery(t *testing.T) {
	h := newP009Hub(t)
	wc, raw := h.connect(t)
	require.False(t, h.c.isVerified())
	require.Equal(t, Disconnected, h.a.connectionManager.getState())
	// No protected action is processed before the signature challenge.
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	var updates esystem.PackageUpdates
	require.ErrorIs(t, transport.NewWebSocketTransport(wc).Request(ctx, common.GetPackageUpdates, nil, &updates), context.DeadlineExceeded)
	cancel()
	// A signature over the wrong token closes this unauthenticated connection.
	ctx, cancel = context.WithTimeout(t.Context(), time.Second)
	_, err := wc.GetFingerprint(ctx, "wrong-token", h.signer, false)
	require.Error(t, err)
	cancel()
	require.Eventually(t, func() bool { return h.c.getConn() == nil }, time.Second, time.Millisecond)
	require.False(t, h.c.isVerified())
	wc, raw = h.connect(t)
	h.authenticate(t, wc)
	old := h.c.getConn()
	// Text and malformed binary traffic cannot dispatch a protected action.
	require.NoError(t, raw.WriteMessage(gws.OpcodeText, []byte("ignored")))
	require.NoError(t, raw.WriteMessage(gws.OpcodeBinary, []byte{0xff}))
	require.NoError(t, transport.NewWebSocketTransport(wc).Request(t.Context(), common.GetPackageUpdates, nil, &updates))
	h.c.Close()
	wc, _ = h.connect(t)
	h.authenticate(t, wc)
	h.c.OnClose(old, errors.New("late close"))
	require.True(t, h.c.isVerified())
	require.ErrorContains(t, h.c.handleHubRequest(&common.HubRequest[cbor.RawMessage]{Action: common.GetPackageUpdates}, nil, old), "hub not verified", "stale connection cannot borrow replacement authentication")
	sig, err := h.signer.Sign(rand.Reader, []byte(h.c.token))
	require.NoError(t, err)
	challenge, err := cbor.Marshal(common.FingerprintRequest{Signature: sig.Blob})
	require.NoError(t, err)
	require.ErrorIs(t, h.c.handleAuthChallenge(&common.HubRequest[cbor.RawMessage]{Data: challenge}, nil, old), gws.ErrConnClosed)
	require.True(t, h.c.isVerified())
	require.NoError(t, transport.NewWebSocketTransport(wc).Request(t.Context(), common.GetPackageUpdates, nil, &updates))
}
func TestP009WebSocketRepeatedRecoveryAndFreshData(t *testing.T) {
	h := newP009Hub(t)
	for i := range 12 {
		h.reject.Store(true)
		for range 3 {
			require.Error(t, h.c.ConnectContext(t.Context()))
		}
		require.Nil(t, h.c.getConn())
		h.reject.Store(false)
		wc, raw := h.connect(t)
		h.authenticate(t, wc)
		// Counter stored through the actual cache proves each recovery consumes a
		// fresh Agent response rather than a retained Hub result.
		h.a.cache.Set(&esystem.CombinedData{Stats: esystem.Stats{Cpu: float64(i + 1)}}, 60000)
		var data esystem.CombinedData
		require.NoError(t, transport.NewWebSocketTransport(wc).Request(t.Context(), common.GetData, common.DataRequestOptions{CacheTimeMs: 60000}, &data))
		require.Equal(t, float64(i+1), data.Stats.Cpu)
		require.NoError(t, raw.NetConn().Close())
		require.Eventually(t, func() bool { return h.c.getConn() == nil }, time.Second, time.Millisecond)
	}
	require.False(t, h.a.connectionManager.isConnectingNow())
}
func TestP009ChallengeSilenceExpires(t *testing.T) {
	h := newP009Hub(t)
	_, _ = h.connect(t)
	conn := h.c.getConn()
	require.NotNil(t, conn)
	require.NoError(t, conn.SetDeadline(time.Now().Add(30*time.Millisecond)))
	require.Eventually(t, func() bool { return h.c.getConn() == nil }, time.Second, time.Millisecond)
	require.False(t, h.c.isVerified())
	wc, _ := h.connect(t)
	h.authenticate(t, wc)
}
func TestP009ShutdownDuringWebSocketUpgrade(t *testing.T) {
	a := createTestAgent(t)
	accepted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(accepted); <-r.Context().Done() }))
	defer server.Close()
	t.Setenv("BESZEL_AGENT_HUB_URL", server.URL)
	t.Setenv("BESZEL_AGENT_TOKEN", "p009-synthetic")
	t.Setenv("BESZEL_AGENT_DISABLE_SSH", "true")
	c, err := newWebSocketClient(a)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- c.Connect() }()
	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("upgrade not reached")
	}
	c.Close()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("Close failed to cancel upgrade")
	}
	require.Nil(t, c.getConn())
}
func TestP009DialFaults(t *testing.T) {
	for _, kind := range []string{"dns", "connect-timeout", "refused"} {
		t.Run(kind, func(t *testing.T) {
			h := newP009Hub(t)
			options := h.c.getOptions()
			options.NewDialer = func() (gws.Dialer, error) { return p009FaultDialer{kind: kind}, nil }
			require.Error(t, h.c.ConnectContext(t.Context()))
			require.Nil(t, h.c.getConn())
			require.False(t, h.c.isVerified())
		})
	}
}

type p009FaultDialer struct{ kind string }

func (d p009FaultDialer) Dial(n, a string) (net.Conn, error) {
	return d.DialContext(context.Background(), n, a)
}
func (d p009FaultDialer) DialContext(ctx context.Context, n, a string) (net.Conn, error) {
	switch d.kind {
	case "dns":
		return nil, &net.OpError{Op: "dial", Net: n, Err: &net.DNSError{Name: "synthetic.invalid", IsNotFound: true}}
	case "connect-timeout":
		return nil, &net.OpError{Op: "dial", Net: n, Err: context.DeadlineExceeded}
	default:
		return nil, &net.OpError{Op: "dial", Net: n, Err: syscall.ECONNREFUSED}
	}
}

type p009HubHandler struct {
	hubws.Handler
	pong chan struct{}
}

func (h *p009HubHandler) OnPong(_ *gws.Conn, _ []byte) {
	select {
	case h.pong <- struct{}{}:
	default:
	}
}

func TestP009AuthenticationWaitsForFingerprintSend(t *testing.T) {
	h := newP009Hub(t)
	// Leave the read loop stopped so the close callback cannot mask the state
	// transition after a failed fingerprint write.
	conn, _, err := gws.NewClient(&gws.BuiltinEventHandler{}, h.c.getOptions())
	require.NoError(t, err)
	h.c.connMu.Lock()
	h.c.Conn = conn
	h.c.connMu.Unlock()
	require.NoError(t, conn.NetConn().Close())
	sig, err := h.signer.Sign(rand.Reader, []byte(h.c.token))
	require.NoError(t, err)
	data, err := cbor.Marshal(common.FingerprintRequest{Signature: sig.Blob})
	require.NoError(t, err)
	require.Error(t, h.c.handleAuthChallenge(&common.HubRequest[cbor.RawMessage]{Data: data}, nil, conn))
	require.False(t, h.c.isVerified(), "a failed fingerprint send must not announce polling readiness")
}

func TestP009QueuedConnectCancellation(t *testing.T) {
	h := newP009Hub(t)
	entered := make(chan struct{})
	blocker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
	defer blocker.Close()
	u, _ := url.Parse(blocker.URL)
	h.c.hubURL = u
	first := make(chan error, 1)
	go func() { first <- h.c.ConnectContext(t.Context()) }()
	<-entered
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	second := make(chan error, 1)
	go func() { second <- h.c.ConnectContext(ctx) }()
	select {
	case err := <-second:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(50 * time.Millisecond):
		h.c.Close()
		<-first
		<-second
		t.Fatal("queued connection ignored cancellation")
	}
	h.c.Close()
	<-first
}
