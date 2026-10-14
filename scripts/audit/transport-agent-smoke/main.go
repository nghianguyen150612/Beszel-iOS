// Command transport-agent-smoke runs a supplied Agent binary against the real
// Hub WS/SSH transports on disposable loopback endpoints. It never opens a Hub DB.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/blang/semver"
	"github.com/henrygd/beszel/internal/common"
	esystem "github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/hub/transport"
	"github.com/henrygd/beszel/internal/hub/ws"
	"github.com/lxzan/gws"
	"golang.org/x/crypto/ssh"
)

type result struct {
	Transport string   `json:"transport"`
	Version   string   `json:"agent_version"`
	Samples   []string `json:"sample_received_utc"`
	OS        uint8    `json:"os"`
	Uptime    uint64   `json:"uptime"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	binary := flag.String("agent", "", "absolute Agent executable path")
	version := flag.String("version", "0.21.0", "expected Agent version")
	flag.Parse()
	if !filepath.IsAbs(*binary) {
		return errors.New("--agent must be an absolute executable path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return err
	}
	var results []result
	for _, mode := range []string{"websocket", "ssh"} {
		r, err := exercise(ctx, *binary, *version, mode, signer)
		if err != nil {
			return fmt.Errorf("%s: %w", mode, err)
		}
		results = append(results, r)
	}
	return json.NewEncoder(os.Stdout).Encode(results)
}
func exercise(ctx context.Context, binary, version, mode string, signer ssh.Signer) (result, error) {
	r := result{Transport: mode, Version: version}
	dir, err := os.MkdirTemp("", "beszel-transport-smoke-")
	if err != nil {
		return r, err
	}
	defer os.RemoveAll(dir)
	env := []string{}
	for _, item := range os.Environ() {
		name, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(name, "BESZEL_AGENT_") {
			continue
		}
		switch name {
		case "KEY", "KEY_FILE", "HUB_URL", "HUB_HEADERS", "TOKEN", "TOKEN_FILE", "CA_CERT_FILE", "DATA_DIR", "DISABLE_SSH", "LISTEN", "PORT", "NETWORK":
			continue
		}
		env = append(env, item)
	}
	env = append(env, "BESZEL_AGENT_DATA_DIR="+dir, "BESZEL_AGENT_KEY="+strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))))
	var tr transport.Transport
	var server *httptest.Server
	ready := make(chan *ws.WsConn, 4)
	var host, port string
	if mode == "websocket" {
		up := gws.NewUpgrader(&ws.Handler{}, nil)
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Header.Get("X-Token") != "p009-disposable" || req.Header.Get("X-Beszel") != version {
				w.WriteHeader(401)
				return
			}
			conn, err := up.Upgrade(w, req)
			if err != nil {
				return
			}
			wc := ws.NewWsConnection(conn, semver.MustParse(version))
			conn.Session().Store("wsConn", wc)
			ready <- wc
			conn.ReadLoop()
		}))
		defer server.Close()
		env = append(env, "BESZEL_AGENT_HUB_URL="+server.URL, "BESZEL_AGENT_TOKEN=p009-disposable", "BESZEL_AGENT_DISABLE_SSH=true")
	} else {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return r, err
		}
		host, port, err = net.SplitHostPort(ln.Addr().String())
		ln.Close()
		if err != nil {
			return r, err
		}
		env = append(env, "BESZEL_AGENT_DISABLE_SSH=false", "BESZEL_AGENT_LISTEN="+net.JoinHostPort(host, port))
	}
	cmd := exec.CommandContext(ctx, binary)
	cmd.Env = env
	// Suppress child telemetry/config logs; report only sanitized failure metadata.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return r, err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	defer func() {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case <-exited:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-exited
		}
	}()
	if mode == "websocket" {
		select {
		case wc := <-ready:
			auth, cancel := context.WithTimeout(ctx, 5*time.Second)
			fp, err := wc.GetFingerprint(auth, "p009-disposable", signer, false)
			cancel()
			if err != nil {
				return r, err
			}
			if fp.Fingerprint == "" {
				return r, errors.New("missing authenticated fingerprint")
			}
			tr = transport.NewWebSocketTransport(wc)
		case err := <-exited:
			exited <- err
			return r, errors.New("agent exited before connection")
		case <-ctx.Done():
			return r, ctx.Err()
		}
	} else {
		// This disposable Agent generates its host key at process start. Pin the first
		// key for this isolated fixture; this does not establish production host trust.
		var mu sync.Mutex
		var pinned ssh.PublicKey
		config := &ssh.ClientConfig{User: "u", ClientVersion: "SSH-2.0-beszel_0.21.0", Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, Timeout: time.Second,
			HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
				mu.Lock()
				defer mu.Unlock()
				if pinned == nil {
					pinned = key
				}
				if string(pinned.Marshal()) != string(key.Marshal()) {
					return errors.New("fixture host key changed")
				}
				return nil
			}}
		st := transport.NewSSHTransport(transport.SSHTransportConfig{Host: host, Port: port, Config: config})
		tr = st
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			if client, err := connectSSH(ctx, st, host, port, config); err == nil {
				_, v, _ := strings.Cut(string(client.Conn.ServerVersion()), "_")
				r.Version = v
				break
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return r, ctx.Err()
			}
		}
	}
	defer tr.Close()
	for range 2 {
		var sample esystem.CombinedData
		request, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := tr.Request(request, common.GetData, common.DataRequestOptions{CacheTimeMs: 0, IncludeDetails: true}, &sample)
		cancel()
		if err != nil {
			return r, err
		}
		if sample.Info.Uptime == 0 {
			return r, errors.New("sample lacks host uptime")
		}
		r.Samples = append(r.Samples, time.Now().UTC().Format(time.RFC3339Nano))
		r.OS = sample.Info.Os
		r.Uptime = sample.Info.Uptime
	}
	if r.Version != version {
		return r, errors.New("unexpected Agent version")
	}
	return r, nil
}

// Adapt the historical transport's externally-owned setup API without changing
// its request/response implementation. New transports own Connect themselves.
func connectSSH(ctx context.Context, tr *transport.SSHTransport, host, port string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	if current, ok := any(tr).(interface {
		Connect(context.Context) (*ssh.Client, error)
	}); ok {
		return current.Connect(ctx)
	}
	legacy, ok := any(tr).(interface {
		SetClient(*ssh.Client)
		SetAgentVersion(semver.Version)
	})
	if !ok {
		return nil, errors.New("unknown historical SSH setup API")
	}
	dialer := net.Dialer{Timeout: time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	sc, ch, req, err := ssh.NewClientConn(conn, net.JoinHostPort(host, port), cfg)
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	client := ssh.NewClient(sc, ch, req)
	_, v, _ := strings.Cut(string(client.Conn.ServerVersion()), "_")
	version, err := semver.Parse(v)
	if err != nil {
		client.Close()
		return nil, err
	}
	legacy.SetAgentVersion(version)
	legacy.SetClient(client)
	return client, nil
}
