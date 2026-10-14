//go:build testing

package agent

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestP010TCPBudgetAndCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	client, peer := net.Pipe()
	defer peer.Close()
	calls := 0
	elapsed, err := monitorTCPWith(ctx, "fixture", 443,
		func(context.Context, string) ([]string, error) { return []string{"192.0.2.1", "::1"}, nil },
		func(ctx context.Context, addr string, budget time.Duration) (net.Conn, error) {
			calls++
			if calls == 1 {
				require.Equal(t, "192.0.2.1:443", addr)
				require.Less(t, budget, 100*time.Millisecond)
				timer := time.NewTimer(budget)
				defer timer.Stop()
				select {
				case <-timer.C:
					return nil, context.DeadlineExceeded
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			require.Equal(t, "[::1]:443", addr)
			return client, nil
		})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Greater(t, elapsed, int64(1000))
	_, err = peer.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
}

func TestP010TCPNegativeAndIPv6(t *testing.T) {
	for _, target := range []string{"127.0.0.1", "::1"} {
		listener, err := net.Listen("tcp", net.JoinHostPort(target, "0"))
		require.NoError(t, err)
		t.Cleanup(func() { listener.Close() })
		done := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err == nil {
				defer conn.Close()
				_, err = conn.Read(make([]byte, 1))
			}
			done <- err
		}()
		us, err := monitorTCP(t.Context(), target, uint16(listener.Addr().(*net.TCPAddr).Port))
		require.NoError(t, err)
		require.GreaterOrEqual(t, us, int64(0))
		require.ErrorIs(t, <-done, io.EOF)
	}
	for _, tc := range []struct {
		target string
		port   uint16
	}{{"127.0.0.1", 0}, {"", 80}, {"127.0.0.1", 1}} {
		us, err := monitorTCP(t.Context(), tc.target, tc.port)
		require.Error(t, err)
		require.Equal(t, int64(-1), us)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	us, err := monitorTCP(ctx, "localhost", 80)
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
	us, err = monitorTCPWith(t.Context(), "empty", 80, func(context.Context, string) ([]string, error) { return nil, nil }, nil)
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
}

func TestP010DNSServerConfiguration(t *testing.T) {
	for _, tc := range []struct{ in, want string }{{"::1", "[::1]:53"}, {"[::1]", "[::1]:53"}, {"[::1]:5353", "[::1]:5353"}, {"127.0.0.1:5353", "127.0.0.1:5353"}, {"dns.test", "dns.test:53"}} {
		got, err := dnsServerAddress(tc.in)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
	for _, server := range []string{"127.0.0.1:0", "127.0.0.1:65536", "localhost:dns", ":53", "host:", "broken:ipv6:server"} {
		_, err := dnsResolverForServer(server).Dial(t.Context(), "udp", "127.0.0.1:53")
		require.Error(t, err)
	}
	// An explicit failing resolver cannot succeed using the machine's resolver.
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	us, err := monitorDNS(ctx, "p010.invalid.", "127.0.0.1:1")
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
	server := udpDNSTestServer(t, nil)
	us, err = monitorDNS(t.Context(), "p010.invalid.", server)
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
	server = udpDNSTestServer(t, []string{"192.0.2.10"})
	us, err = monitorDNS(t.Context(), "p010.invalid.", server)
	require.NoError(t, err)
	require.GreaterOrEqual(t, us, int64(0))
	ctx, cancel = context.WithCancel(t.Context())
	cancel()
	us, err = monitorDNS(ctx, "p010.invalid.", server)
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
}

func TestP010HTTPFailuresAndRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/success", http.StatusFound)
		case "/slow":
			<-r.Context().Done()
		case "/closed":
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
		case "/error":
			w.WriteHeader(503)
		default:
			time.Sleep(2 * time.Millisecond)
			w.WriteHeader(204)
		}
	}))
	defer server.Close()
	us, err := monitorHTTP(t.Context(), server.Client(), server.URL+"/redirect")
	require.NoError(t, err)
	require.GreaterOrEqual(t, us, int64(2000))
	for _, url := range []string{":bad", server.URL + "/error", server.URL + "/closed"} {
		us, err := monitorHTTP(t.Context(), server.Client(), url)
		require.Error(t, err)
		require.Equal(t, int64(-1), us)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	us, err = monitorHTTP(ctx, server.Client(), server.URL+"/slow")
	require.True(t, errors.Is(err, context.DeadlineExceeded))
	require.Equal(t, int64(-1), us)
	ctx, cancel = context.WithCancel(t.Context())
	cancel()
	us, err = monitorHTTP(ctx, server.Client(), server.URL)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int64(-1), us)
}
