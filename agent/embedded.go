package agent

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/lxzan/gws"
	"golang.org/x/crypto/ssh"
)

// EmbeddedConfig is explicit, instance-local configuration. It never reads or
// writes process environment and always disables SSH and privileged collectors.
// Collect must return an error rather than fabricate unavailable required stats.
type EmbeddedConfig struct {
	HubURL                   string
	Token                    string
	HubKey                   string // authorized_keys format; required to verify the Hub challenge
	DataDirectory            string
	Details                  system.Details
	Collect                  func(common.DataRequestOptions) (*system.CombinedData, error)
	ReconnectInterval        time.Duration
	AllowInsecureDevelopment bool
}

type RuntimeStatus struct {
	State         string    `json:"state"`
	Error         string    `json:"error,omitempty"`
	LastConnected time.Time `json:"lastConnected"`
	LastMetrics   time.Time `json:"lastMetrics"`
}

// EmbeddedRuntime serializes start/stop and joins every connection read loop.
// Stop may block on cleanup: callers must use a worker queue, never the UI thread.
type EmbeddedRuntime struct {
	mu       sync.Mutex
	statusMu sync.Mutex
	status   RuntimeStatus
	cancel   context.CancelFunc
	done     chan struct{}
}

func (r *EmbeddedRuntime) Status() RuntimeStatus {
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	s := r.status
	if s.State == "" {
		s.State = "disconnected"
	}
	return s
}
func (r *EmbeddedRuntime) update(state, detail string) {
	r.statusMu.Lock()
	defer r.statusMu.Unlock()
	if state == "network_error" && r.status.State == "authentication_error" {
		return
	}
	r.status.State = state
	r.status.Error = detail
	if state == "connected" {
		r.status.LastConnected = time.Now().UTC()
	}
}

func validateEmbedded(c EmbeddedConfig) (*url.URL, ssh.PublicKey, error) {
	u, err := url.Parse(c.HubURL)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, nil, errors.New("invalid Hub URL")
	}
	switch u.Scheme {
	case "https", "wss":
	case "http", "ws":
		if !c.AllowInsecureDevelopment {
			return nil, nil, errors.New("insecure Hub URL requires explicit development configuration")
		}
	default:
		return nil, nil, errors.New("unsupported Hub URL scheme")
	}
	if strings.TrimSpace(c.Token) == "" || strings.ContainsAny(c.Token, "\r\n") {
		return nil, nil, errors.New("invalid registration token")
	}
	key, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(c.HubKey))
	if err != nil || len(strings.TrimSpace(string(rest))) != 0 {
		return nil, nil, errors.New("invalid Hub public key")
	}
	if !filepath.IsAbs(c.DataDirectory) {
		return nil, nil, errors.New("absolute app data directory required")
	}
	if c.Collect == nil {
		return nil, nil, errors.New("metrics backend required")
	}
	return u, key, nil
}

func (r *EmbeddedRuntime) Start(c EmbeddedConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return errors.New("agent already started; stop before replacing configuration")
	}
	u, key, err := validateEmbedded(c)
	if err != nil {
		return err
	}
	dir, err := GetDataDir(c.DataDirectory)
	if err != nil {
		return err
	}
	// App identity is a random, persisted installation identifier. Do not probe
	// protected host identifiers or derive identity from the mutable device name.
	fp, err := readFingerprint(dir)
	if err != nil {
		b := make([]byte, 24)
		if _, err = rand.Read(b); err != nil {
			return err
		}
		fp = hex.EncodeToString(b)
		if err = SaveFingerprint(dir, fp); err != nil {
			return err
		}
	}
	a := &Agent{dataDir: dir, keys: []ssh.PublicKey{key}, systemDetails: c.Details,
		systemInfo: system.Info{AgentVersion: beszel.Version}, embeddedCollect: c.Collect}
	a.connectionManager = newConnectionManager(a)
	a.handlerRegistry = NewHandlerRegistry()
	client := &WebSocketClient{agent: a, hubURL: u, token: c.Token, fingerprint: fp}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	r.update("connecting", "")
	go func() { defer close(r.done); r.run(ctx, client, c.ReconnectInterval) }()
	return nil
}

func (r *EmbeddedRuntime) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
		<-r.done
		r.cancel = nil
		r.done = nil
	}
	r.update("disconnected", "")
}

// run uses the same WebSocket client, Hub verification and handler registry as
// the executable Agent. A single owner performs all dial/reconnect operations.
func (r *EmbeddedRuntime) run(ctx context.Context, client *WebSocketClient, retry time.Duration) {
	if retry <= 0 {
		retry = 10 * time.Second
	}
	client.embeddedStatus = func(state, detail string) { r.update(state, detail) }
	client.metricsUpdated = func() { r.statusMu.Lock(); r.status.LastMetrics = time.Now().UTC(); r.statusMu.Unlock() }
	for ctx.Err() == nil {
		r.update("connecting", "")
		attemptCtx, cancel := context.WithCancel(ctx)
		opts := client.getOptions()
		opts.NewDialer = func() (gws.Dialer, error) { return contextDialer{attemptCtx}, nil }
		client.hubVerified = false
		conn, resp, err := gws.NewClient(client, opts)
		if err != nil {
			cancel()
			if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
				r.update("authentication_error", "Hub rejected registration")
			} else {
				r.update("network_error", "Hub connection failed (check network, URL and TLS trust)")
			}
		} else {
			// After the handshake, the run loop owns cancellation so intentional
			// Stop can write a close frame before releasing the socket.
			raw := conn.NetConn()
			if secure, ok := raw.(*tls.Conn); ok {
				raw = secure.NetConn()
			}
			if cancellable, ok := raw.(*cancelConn); ok {
				cancellable.stop()
			}
			client.Conn = conn
			finished := make(chan struct{})
			go func() { defer close(finished); conn.ReadLoop() }()
			select {
			case <-ctx.Done():
			case <-finished:
			}
			// Bound the graceful close, then forcibly release the socket and join the reader.
			_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
			_ = conn.WriteClose(1000, nil)
			cancel()
			_ = conn.NetConn().Close()
			<-finished
			client.Conn = nil
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(retry):
		}
	}
}

type contextDialer struct{ ctx context.Context }

func (d contextDialer) Dial(network, addr string) (net.Conn, error) {
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(d.ctx, network, addr)
	if err != nil {
		return nil, err
	}
	// Cancellation interrupts TLS/HTTP handshake; run takes ownership afterward.
	stop := context.AfterFunc(d.ctx, func() { _ = conn.Close() })
	return &cancelConn{Conn: conn, stop: stop}, nil
}

type cancelConn struct {
	net.Conn
	stop func() bool
}

func (c *cancelConn) Close() error { c.stop(); return c.Conn.Close() }
