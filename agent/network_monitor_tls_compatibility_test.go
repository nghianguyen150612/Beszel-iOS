//go:build testing

package agent

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/stretchr/testify/require"
)

func p010Certificate(t *testing.T, serial int64, host string, expires time.Time) (tls.Certificate, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: "P010 fixture"}, NotBefore: time.Now().Add(-24 * time.Hour), NotAfter: expires, KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	leaf, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, leaf
}

func TestP010TLSVerificationAndMetadataIsolation(t *testing.T) {
	for _, tc := range []struct {
		name, host       string
		expires          time.Time
		trusted, success bool
	}{
		{"trusted", "127.0.0.1", time.Now().Add(time.Hour), true, true},
		{"unknown CA", "127.0.0.1", time.Now().Add(time.Hour), false, false},
		{"hostname mismatch", "wrong.invalid", time.Now().Add(time.Hour), true, false},
		{"expired", "127.0.0.1", time.Now().Add(-time.Hour), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert, leaf := p010Certificate(t, 1, tc.host, tc.expires)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
			server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
			server.StartTLS()
			defer server.Close()
			roots := x509.NewCertPool()
			if tc.trusted {
				roots.AddCert(leaf)
			}
			secureTransport := http.DefaultTransport.(*http.Transport).Clone()
			secureTransport.TLSClientConfig = &tls.Config{RootCAs: roots}
			defer secureTransport.CloseIdleConnections()
			insecureTransport := secureTransport.Clone()
			insecureTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			defer insecureTransport.CloseIdleConnections()
			probe := networkMonitorProbe(&http.Client{Transport: secureTransport}, &http.Client{Transport: insecureTransport})
			// Verify policy does not leak between monitor IDs, including updates.
			for _, skip := range []bool{false, true, false} {
				cfg := monitor.Config{ID: "secure", Protocol: "http", Target: server.URL, SkipTLSVerify: skip}
				if skip {
					cfg.ID = "opt-out"
				}
				us, err := probe(t.Context(), cfg)
				if skip || tc.success {
					require.NoError(t, err)
					require.GreaterOrEqual(t, us, int64(0))
				} else {
					require.Error(t, err)
					require.Equal(t, int64(-1), us)
				}
			}
			info, err := checkCert(t.Context(), server.URL)
			require.NoError(t, err)
			require.Equal(t, leaf.NotAfter.UnixMilli(), info.Expires)
			require.False(t, secureTransport.TLSClientConfig.InsecureSkipVerify)
		})
	}
}

func TestP010CertificateRotationCancellationAndRecovery(t *testing.T) {
	cert, leaf := p010Certificate(t, 1, "127.0.0.1", time.Now().Add(time.Hour))
	var current atomic.Pointer[tls.Certificate]
	current.Store(&cert)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	server.TLS = &tls.Config{GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
		return &tls.Config{Certificates: []tls.Certificate{*current.Load()}}, nil
	}}
	server.StartTLS()
	defer server.Close()
	task := newMonitorTask(monitor.Config{ID: "cert", Protocol: "http", Target: server.URL})
	defer task.cancel()
	task.refreshCert(checkCert)
	require.Equal(t, leaf.NotAfter.UnixMilli(), task.takeUnsentCert().Expires)
	require.Nil(t, task.takeUnsentCert())
	rotated, leaf2 := p010Certificate(t, 2, "127.0.0.1", time.Now().Add(48*time.Hour))
	current.Store(&rotated)
	task.nextCertCheck = time.Time{}
	task.refreshCert(checkCert)
	require.Equal(t, leaf2.NotAfter.UnixMilli(), task.takeUnsentCert().Expires)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := checkCert(ctx, server.URL)
	require.Error(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	ctx, cancel = context.WithTimeout(t.Context(), 30*time.Millisecond)
	waitCtx := ctx
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err == nil {
			defer conn.Close()
			<-waitCtx.Done()
		}
	}()
	_, err = checkCert(ctx, "https://"+listener.Addr().String())
	cancel()
	require.Error(t, err)
	<-done
	// A failed inspection keeps known metadata, does not mark it newly unsent,
	// then a later due successful inspection recovers.
	task.nextCertCheck = time.Time{}
	task.refreshCert(func(context.Context, string) (monitor.CertInfo, error) {
		return monitor.CertInfo{}, context.DeadlineExceeded
	})
	require.Nil(t, task.takeUnsentCert())
	require.Equal(t, leaf2.NotAfter.UnixMilli(), task.certInfo().Expires)
	task.nextCertCheck = time.Time{}
	task.refreshCert(checkCert)
	require.NotNil(t, task.takeUnsentCert())
	replacement := newMonitorTaskFromExisting(monitor.Config{ID: "cert", Protocol: "tcp", Target: server.URL}, task)
	defer replacement.cancel()
	require.Nil(t, replacement.certInfo())
}
