//go:build testing

package agent

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/dns/dnsmessage"
)

func TestP010DNSIPv6NXDOMAINTimeoutRecovery(t *testing.T) {
	conn, err := net.ListenPacket("udp6", "[::1]:0")
	require.NoError(t, err)
	defer conn.Close()
	var mode atomic.Int32
	go func() {
		buf := make([]byte, 1500)
		for {
			n, peer, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			if mode.Load() == 1 {
				continue
			}
			var msg dnsmessage.Message
			if msg.Unpack(buf[:n]) != nil {
				continue
			}
			msg.Header.Response = true
			msg.Header.RecursionAvailable = true
			if mode.Load() == 2 {
				msg.Header.RCode = dnsmessage.RCodeNameError
			} else {
				for _, q := range msg.Questions {
					if q.Type == dnsmessage.TypeAAAA {
						msg.Answers = append(msg.Answers, dnsmessage.Resource{Header: dnsmessage.ResourceHeader{Name: q.Name, Type: q.Type, Class: q.Class}, Body: &dnsmessage.AAAAResource{AAAA: [16]byte(net.ParseIP("2001:db8::10").To16())}})
					}
				}
			}
			packet, err := msg.Pack()
			if err == nil {
				_, _ = conn.WriteTo(packet, peer)
			}
		}
	}()
	for _, tc := range []struct {
		mode int32
		fail bool
	}{{0, false}, {2, true}, {1, true}, {0, false}} {
		mode.Store(tc.mode)
		ctx, cancel := context.WithTimeout(t.Context(), 80*time.Millisecond)
		us, err := monitorDNS(ctx, "p010-resolver.invalid.", conn.LocalAddr().String())
		cancel()
		if tc.fail {
			require.Error(t, err)
			require.Equal(t, int64(-1), us)
		} else {
			require.NoError(t, err)
			require.GreaterOrEqual(t, us, int64(0))
		}
	}
	// Default resolver succeeds for this same target, while explicit failed
	// selection must still fail rather than silently using that default.
	original := net.DefaultResolver
	net.DefaultResolver = tcpMonitorTestResolver([]string{"192.0.2.10"})
	defer func() { net.DefaultResolver = original }()
	_, err = monitorDNS(t.Context(), "p010-resolver.invalid.", "")
	require.NoError(t, err)
	mode.Store(1)
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	us, err := monitorDNS(ctx, "p010-resolver.invalid.", conn.LocalAddr().String())
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
}

func TestP010DNSLiteralCancellationAndInvalidServer(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	us, err := monitorDNS(ctx, "127.0.0.1", "")
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int64(-1), us)
	us, err = monitorDNS(t.Context(), "127.0.0.1", "127.0.0.1:0")
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
}
