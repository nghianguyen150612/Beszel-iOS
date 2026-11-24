package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/lxzan/gws"
	"golang.org/x/crypto/ssh"
)

func embeddedTestConfig(t *testing.T) (EmbeddedConfig, ssh.Signer) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return EmbeddedConfig{HubURL: "https://localhost", Token: "fixture-token", HubKey: string(ssh.MarshalAuthorizedKey(signer.PublicKey())), DataDirectory: t.TempDir(), Collect: func(common.DataRequestOptions) (*system.CombinedData, error) {
		return &system.CombinedData{Stats: system.Stats{Mem: 8}}, nil
	}}, signer
}
func TestEmbeddedValidation(t *testing.T) {
	c, _ := embeddedTestConfig(t)
	for _, u := range []string{"", "file:///tmp", "http://localhost", "ws://localhost", "https://user:secret@host", "https://host?token=secret", "https://host#fragment"} {
		c.HubURL = u
		if _, _, err := validateEmbedded(c); err == nil {
			t.Errorf("accepted %q", u)
		}
	}
	c.HubURL = "wss://localhost/prefix"
	if _, _, err := validateEmbedded(c); err != nil {
		t.Fatal(err)
	}
	c.HubKey = ""
	if _, _, err := validateEmbedded(c); err == nil {
		t.Fatal("missing trust key accepted")
	}
}
func TestEmbeddedRepeatedConcurrentStopAndRecovery(t *testing.T) {
	c, _ := embeddedTestConfig(t)
	c.HubURL = "http://127.0.0.1:1"
	c.AllowInsecureDevelopment = true
	var r EmbeddedRuntime
	for i := 0; i < 10; i++ {
		if err := r.Start(c); err != nil {
			t.Fatal(err)
		}
		if err := r.Start(c); err == nil {
			t.Fatal("duplicate start")
		}
		var wg sync.WaitGroup
		for j := 0; j < 5; j++ {
			wg.Add(1)
			go func() { defer wg.Done(); _ = r.Status(); r.Stop() }()
		}
		wg.Wait()
		if r.Status().State != "disconnected" {
			t.Fatal(r.Status())
		}
	}
	c.Token = ""
	if r.Start(c) == nil {
		t.Fatal("invalid token")
	}
	c.Token = "new-token"
	if err := r.Start(c); err != nil {
		t.Fatal(err)
	}
	r.Stop()
}

type fixtureHandler struct {
	gws.BuiltinEventHandler
	signer  ssh.Signer
	token   string
	replies chan common.AgentResponse
}

func (h *fixtureHandler) OnOpen(c *gws.Conn) {
	sig, _ := h.signer.Sign(rand.Reader, []byte(h.token))
	data, _ := cbor.Marshal(common.FingerprintRequest{Signature: sig.Blob, NeedSysInfo: true})
	id := uint32(1)
	b, _ := cbor.Marshal(common.HubRequest[cbor.RawMessage]{Action: common.CheckFingerprint, Data: data, Id: &id})
	_ = c.WriteMessage(gws.OpcodeBinary, b)
}
func (h *fixtureHandler) OnMessage(c *gws.Conn, m *gws.Message) {
	defer m.Close()
	var r common.AgentResponse
	if cbor.Unmarshal(m.Bytes(), &r) != nil {
		return
	}
	h.replies <- r
	if r.Fingerprint != nil {
		id := uint32(2)
		b, _ := cbor.Marshal(common.HubRequest[common.DataRequestOptions]{Action: common.GetData, Id: &id})
		_ = c.WriteMessage(gws.OpcodeBinary, b)
	}
}
func TestEmbeddedProtocolAndStats(t *testing.T) {
	c, signer := embeddedTestConfig(t)
	h := &fixtureHandler{signer: signer, token: c.Token, replies: make(chan common.AgentResponse, 10)}
	up := gws.NewUpgrader(h, nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/beszel/agent-connect" || r.Header.Get("X-Token") != c.Token || r.Header.Get("X-Beszel") == "" {
			http.Error(w, "denied", 401)
			return
		}
		conn, err := up.Upgrade(w, r)
		if err == nil {
			conn.ReadLoop()
		}
	}))
	defer server.Close()
	c.HubURL = server.URL
	c.AllowInsecureDevelopment = true
	var runtime EmbeddedRuntime
	if err := runtime.Start(c); err != nil {
		t.Fatal(err)
	}
	defer runtime.Stop()
	for i := 0; i < 2; i++ {
		select {
		case reply := <-h.replies:
			if i == 0 && (reply.Fingerprint == nil || reply.Fingerprint.Fingerprint == "") {
				t.Fatal(reply)
			}
			if i == 1 && (reply.SystemData == nil || reply.SystemData.Stats.Mem != 8) {
				t.Fatal(reply)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("protocol timeout", runtime.Status())
		}
	}
	if runtime.Status().LastConnected.IsZero() {
		t.Fatal("not authenticated")
	}
}
func TestEmbeddedRejectedTokenAndTLS(t *testing.T) {
	for _, tls := range []bool{false, true} {
		t.Run(map[bool]string{false: "token", true: "tls"}[tls], func(t *testing.T) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "denied", 401) }))
			if tls {
				server.StartTLS()
			} else {
				server.Start()
			}
			defer server.Close()
			c, _ := embeddedTestConfig(t)
			c.HubURL = server.URL
			c.AllowInsecureDevelopment = true
			var r EmbeddedRuntime
			if err := r.Start(c); err != nil {
				t.Fatal(err)
			}
			defer r.Stop()
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				s := r.Status()
				if s.State == "network_error" && tls || s.State == "authentication_error" && !tls {
					return
				}
				time.Sleep(time.Millisecond * 10)
			}
			t.Fatal(r.Status())
		})
	}
}
func TestEmbeddedCancellationDuringHandshake(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	c, _ := embeddedTestConfig(t)
	c.HubURL = server.URL
	c.AllowInsecureDevelopment = true
	var r EmbeddedRuntime
	if err := r.Start(c); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("dial timeout")
	}
	done := make(chan struct{})
	go func() { r.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation blocked")
	}
}
func TestEmbeddedSSHGuard(t *testing.T) {
	a := &Agent{embeddedCollect: func(common.DataRequestOptions) (*system.CombinedData, error) { return nil, errors.New("unavailable") }}
	if a.StartServer(ServerOptions{Network: "tcp", Addr: "127.0.0.1:0"}) == nil {
		t.Fatal("SSH enabled")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (contextDialer{ctx}).Dial("tcp", "127.0.0.1:1"); err == nil {
		t.Fatal("dial after cancel")
	}
}

func TestEmbeddedReconnectIdentityAndUnavailableStats(t *testing.T) {
	c, signer := embeddedTestConfig(t)
	c.Collect = func(common.DataRequestOptions) (*system.CombinedData, error) { return nil, errors.New("not available") }
	h := &fixtureHandler{signer: signer, token: c.Token, replies: make(chan common.AgentResponse, 20)}
	up := gws.NewUpgrader(h, nil)
	sockets := make(chan *gws.Conn, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r)
		if err == nil {
			sockets <- conn
			conn.ReadLoop()
		}
	}))
	defer server.Close()
	c.HubURL = server.URL
	c.AllowInsecureDevelopment = true
	c.ReconnectInterval = time.Millisecond * 10
	var r EmbeddedRuntime
	if err := r.Start(c); err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	var fingerprint string
	for attempt := 0; attempt < 3; attempt++ {
		for i := 0; i < 2; i++ {
			select {
			case reply := <-h.replies:
				if i == 0 {
					if reply.Fingerprint == nil {
						t.Fatal(reply)
					}
					if fingerprint != "" && fingerprint != reply.Fingerprint.Fingerprint {
						t.Fatal("identity changed")
					}
					fingerprint = reply.Fingerprint.Fingerprint
				}
				if i == 1 && (reply.Error == "" || reply.SystemData != nil) {
					t.Fatal("fabricated stats", reply)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("reconnect timeout", r.Status())
			}
		}
		conn := <-sockets
		if attempt == 0 {
			_ = conn.NetConn().Close()
		} // Hub disconnect and automatic reconnect
		if attempt == 1 {
			r.Stop()
			if err := r.Start(c); err != nil {
				t.Fatal(err)
			}
		} // persisted app identity
	}
	if !r.Status().LastMetrics.IsZero() {
		t.Fatal("failed stats marked successful")
	}
}

func TestEmbeddedConcurrentStartStop(t *testing.T) {
	c, _ := embeddedTestConfig(t)
	c.HubURL = "http://127.0.0.1:1"
	c.AllowInsecureDevelopment = true
	var r EmbeddedRuntime
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = r.Start(c); _ = r.Status(); r.Stop() }()
	}
	wg.Wait()
	r.Stop()
	if r.Status().State != "disconnected" {
		t.Fatal(r.Status())
	}
}
