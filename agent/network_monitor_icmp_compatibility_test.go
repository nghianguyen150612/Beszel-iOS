//go:build testing

package agent

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/icmp"
)

func fixtureICMPFamily(v6 bool) *icmpFamily {
	if v6 {
		return &icmpFamily{rawNetwork: icmpV6.rawNetwork, dgramNetwork: icmpV6.dgramNetwork, listenAddr: icmpV6.listenAddr, echoType: icmpV6.echoType, replyType: icmpV6.replyType, proto: icmpV6.proto, isIPv6: true}
	}
	return &icmpFamily{rawNetwork: icmpV4.rawNetwork, dgramNetwork: icmpV4.dgramNetwork, listenAddr: icmpV4.listenAddr, echoType: icmpV4.echoType, replyType: icmpV4.replyType, proto: icmpV4.proto}
}

func TestP010ICMPCapabilityRecoveryAndIsolation(t *testing.T) {
	v4, v6 := fixtureICMPFamily(false), fixtureICMPFamily(true)
	ip := net.ParseIP("127.0.0.1")
	rawDenied := false
	listenCalls, execCalls := 0, 0
	listen := func(network, address string) (icmpPacketConn, error) { listenCalls++; return testICMPPacketConn{}, nil }
	native := func(ctx context.Context, network string, f *icmpFamily, dst net.Addr) (int64, error) {
		if rawDenied && network == f.rawNetwork {
			return -1, &os.SyscallError{Syscall: "socket", Err: syscall.EPERM}
		}
		return 123, nil
	}
	fallback := func(context.Context, string, bool) (int64, error) { execCalls++; return -1, exec.ErrNotFound }
	us, err := monitorICMPFamily(t.Context(), v4, ip, listen, native, fallback)
	require.NoError(t, err)
	require.Equal(t, int64(123), us)
	rawDenied = true
	us, err = monitorICMPFamily(t.Context(), v4, ip, listen, native, fallback)
	require.ErrorIs(t, err, syscall.EPERM)
	require.Equal(t, int64(-1), us)
	us, err = monitorICMPFamily(t.Context(), v4, ip, listen, native, fallback)
	require.NoError(t, err)
	require.Equal(t, icmpDatagram, v4.mode)
	require.Equal(t, icmpUntried, v6.mode)
	// Cooldown expiry permits recovery to raw sockets.
	v4.retryAt = time.Now().Add(-time.Second)
	rawDenied = false
	_, err = monitorICMPFamily(t.Context(), v4, ip, listen, native, fallback)
	require.NoError(t, err)
	require.Equal(t, icmpRaw, v4.mode)
	denied := func(string, string) (icmpPacketConn, error) { listenCalls++; return nil, syscall.EACCES }
	v4 = fixtureICMPFamily(false)
	_, err = monitorICMPFamily(t.Context(), v4, ip, denied, native, fallback)
	require.ErrorIs(t, err, exec.ErrNotFound)
	previous := listenCalls
	for range 10 {
		us, err = monitorICMPFamily(t.Context(), v4, ip, denied, native, fallback)
		require.Error(t, err)
		require.Equal(t, int64(-1), us)
	}
	require.Equal(t, previous, listenCalls)
	require.Equal(t, 1, execCalls)
	v4.retryAt = time.Now().Add(-time.Second)
	_, err = monitorICMPFamily(t.Context(), v4, ip, listen, native, fallback)
	require.NoError(t, err)
	require.Equal(t, icmpRaw, v4.mode)
}

func TestP010IOSPingValidation(t *testing.T) {
	for _, v6 := range []bool{false, true} {
		c := new(iosPingCapability)
		calls := 0
		name := "ping"
		target := "127.0.0.1"
		if v6 {
			name, target = "ping6", "::1"
		}
		lookup := func(got string) (string, error) { require.Equal(t, name, got); return "/fixture/" + name, nil }
		run := func(ctx context.Context, path string, args []string) (int64, error) {
			calls++
			require.Equal(t, []string{"-n", "-c", "1", target}, args)
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.LessOrEqual(t, time.Until(deadline), time.Second)
			return 37, nil
		}
		path, err := c.command(t.Context(), v6, lookup, run)
		require.NoError(t, err)
		require.Equal(t, "/fixture/"+name, path)
		_, err = c.command(t.Context(), v6, lookup, run)
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		c.at = time.Now().Add(-icmpCapabilityRetry)
		_, err = c.command(t.Context(), v6, lookup, func(context.Context, string, []string) (int64, error) {
			calls++
			return -1, errors.New("unsupported options")
		})
		require.ErrorIs(t, err, errors.ErrUnsupported)
		_, err = c.command(t.Context(), v6, lookup, run)
		require.Error(t, err)
		require.Equal(t, 2, calls)
		c.at = time.Now().Add(-icmpCapabilityRetry)
		_, err = c.command(t.Context(), v6, lookup, run)
		require.NoError(t, err)
		require.Equal(t, 3, calls)
	}
	c := new(iosPingCapability)
	_, err := c.command(t.Context(), true, func(string) (string, error) { return "", exec.ErrNotFound }, nil)
	require.ErrorIs(t, err, errors.ErrUnsupported)
}

func TestP010ICMPConcurrentWrapAndDelayedDuplicate(t *testing.T) {
	icmpSequence.Store(65534)
	t.Cleanup(func() { icmpSequence.Store(0) })
	for _, v6 := range []bool{false, true} {
		family := fixtureICMPFamily(v6)
		ip := net.ParseIP("127.0.0.1")
		if v6 {
			ip = net.ParseIP("::1")
		}
		dst := &net.IPAddr{IP: ip}
		var prior []byte
		for range 3 {
			c := &scriptedICMPConn{local: dst}
			c.onWrite = func(data []byte, _ net.Addr) {
				msg, err := icmp.ParseMessage(family.proto, data)
				require.NoError(t, err)
				msg.Type = family.replyType
				reply, err := msg.Marshal(nil)
				require.NoError(t, err)
				// Delayed/duplicate previous echo must fail the sequence/token checks.
				if prior != nil {
					c.replies = append(c.replies, icmpTestReply{prior, dst})
				}
				c.replies = append(c.replies, icmpTestReply{reply, dst}, icmpTestReply{reply, dst})
				prior = reply
			}
			us, err := monitorICMPPacket(t.Context(), c, family, dst)
			require.NoError(t, err)
			require.GreaterOrEqual(t, us, int64(0))
			require.Len(t, c.replies, 1)
		}
	}
	var wg sync.WaitGroup
	var failures atomic.Int32
	for range 32 {
		wg.Go(func() {
			f := fixtureICMPFamily(false)
			dst := &net.IPAddr{IP: net.ParseIP("127.0.0.1")}
			c := &scriptedICMPConn{local: dst}
			c.onWrite = func(data []byte, _ net.Addr) {
				msg, err := icmp.ParseMessage(f.proto, data)
				if err != nil {
					failures.Add(1)
					return
				}
				msg.Type = f.replyType
				reply, _ := msg.Marshal(nil)
				c.replies = []icmpTestReply{{reply, dst}}
			}
			if _, err := monitorICMPPacket(t.Context(), c, f, dst); err != nil {
				failures.Add(1)
			}
		})
	}
	wg.Wait()
	require.Zero(t, failures.Load())
}

func TestP010PingBoundedOutputAndInheritedPipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX subprocess fixtures")
	}
	for _, script := range []string{
		"#!/bin/sh\nhead -c 70000 /dev/zero\n",
		// A child retaining stdout must not hold cmd.Wait indefinitely.
		"#!/bin/sh\nsleep 5 &\nprintf 'time=1.234 ms\\n'\n",
	} {
		path := filepath.Join(t.TempDir(), "ping")
		require.NoError(t, os.WriteFile(path, []byte(script), 0755))
		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		start := time.Now()
		us, err := runPingCommand(ctx, path, nil)
		cancel()
		require.Error(t, err)
		require.Equal(t, int64(-1), us)
		require.Less(t, time.Since(start), time.Second)
	}
	var out limitedPingOutput
	_, err := out.Write(make([]byte, pingOutputLimit+1))
	require.ErrorIs(t, err, errPingOutputLimit)
	require.Zero(t, out.Len())
}

func TestP010NativeSocketFaultAndClosure(t *testing.T) {
	original := icmpNativeListen
	defer func() { icmpNativeListen = original }()
	for _, fault := range []error{syscall.EPERM, syscall.EACCES, syscall.EPROTONOSUPPORT} {
		icmpNativeListen = func(string, string) (net.PacketConn, error) { return nil, fault }
		us, err := monitorICMPNative(t.Context(), "ip4:icmp", &icmpV4, &net.IPAddr{IP: net.ParseIP("127.0.0.1")})
		require.ErrorIs(t, err, fault)
		require.Equal(t, int64(-1), us)
	}
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	icmpNativeListen = func(string, string) (net.PacketConn, error) { return conn, nil }
	us, err := monitorICMPNative(t.Context(), "udp4", &icmpV4, &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	require.Error(t, err)
	require.Equal(t, int64(-1), us)
}
