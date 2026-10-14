//go:build testing

package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lxzan/gws"
	"github.com/stretchr/testify/require"
)

type p009CA struct {
	cert *x509.Certificate
	key  ed25519.PrivateKey
	pem  []byte
}

func p009NewCA(t *testing.T) *p009CA {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	cert := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "P009 disposable CA"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, pub, key)
	require.NoError(t, err)
	cert, err = x509.ParseCertificate(der)
	require.NoError(t, err)
	return &p009CA{cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}
func (ca *p009CA) leaf(t *testing.T, kind string) tls.Certificate {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	cert := &x509.Certificate{SerialNumber: big.NewInt(2), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, DNSNames: []string{"localhost"}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if kind == "expired" {
		cert.NotBefore = time.Now().Add(-2 * time.Hour)
		cert.NotAfter = time.Now().Add(-time.Hour)
	}
	if kind == "hostname" {
		cert.IPAddresses = nil
		cert.DNSNames = []string{"different.invalid"}
	}
	if kind == "purpose" {
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, ca.cert, pub, ca.key)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}
func TestP009TLSCertificateMatrix(t *testing.T) {
	ca, other := p009NewCA(t), p009NewCA(t)
	for _, kind := range []string{"valid-ip", "valid-dns", "expired", "hostname", "purpose", "unknown", "multiple-roots", "wss"} {
		t.Run(kind, func(t *testing.T) {
			leaf := ca.leaf(t, kind)
			up := gws.NewUpgrader(&gws.BuiltinEventHandler{}, nil)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c, e := up.Upgrade(w, r)
				if e == nil {
					c.ReadLoop()
				}
			}))
			server.TLS = &tls.Config{Certificates: []tls.Certificate{leaf}}
			server.StartTLS()
			defer server.Close()
			roots := ca.pem
			if kind == "unknown" {
				roots = other.pem
			}
			if kind == "multiple-roots" {
				roots = append(append([]byte{}, other.pem...), ca.pem...)
			}
			file := filepath.Join(t.TempDir(), "roots.pem")
			require.NoError(t, os.WriteFile(file, roots, 0600))
			t.Setenv("BESZEL_AGENT_CA_CERT_FILE", file)
			cfg, err := getTLSConfig()
			require.NoError(t, err)
			require.False(t, cfg.InsecureSkipVerify)
			addr := "wss" + strings.TrimPrefix(server.URL, "https")
			if kind == "valid-dns" {
				u, _ := url.Parse(addr)
				u.Host = net.JoinHostPort("localhost", u.Port())
				addr = u.String()
			}
			// Production scheme conversion and TLS options, not a replacement dial stack.
			u, _ := url.Parse(addr)
			c := &WebSocketClient{hubURL: u, tlsConfig: cfg}
			conn, _, err := gws.NewClient(&gws.BuiltinEventHandler{}, c.getOptions())
			if kind == "expired" || kind == "hostname" || kind == "purpose" || kind == "unknown" {
				require.Error(t, err)
				require.Nil(t, conn)
			} else {
				require.NoError(t, err)
				require.NoError(t, conn.NetConn().Close())
			}
		})
	}
}
func TestP009TLSRotationAndTrustRecovery(t *testing.T) {
	oldCA, newCA := p009NewCA(t), p009NewCA(t)
	oldCert, newCert := oldCA.leaf(t, "valid"), newCA.leaf(t, "valid")
	var current atomic.Pointer[tls.Certificate]
	current.Store(&oldCert)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	server.TLS = &tls.Config{GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return current.Load(), nil }}
	server.StartTLS()
	defer server.Close()
	file := filepath.Join(t.TempDir(), "roots.pem")
	t.Setenv("BESZEL_AGENT_CA_CERT_FILE", file)
	check := func(roots []byte, wantOK bool) {
		require.NoError(t, os.WriteFile(file, roots, 0600))
		cfg, err := getTLSConfig()
		require.NoError(t, err)
		// SNI invokes the rotation callback instead of httptest's fallback cert.
		cfg.ServerName = "localhost"
		conn, err := tls.Dial("tcp", strings.TrimPrefix(server.URL, "https://"), cfg)
		if wantOK {
			require.NoError(t, err)
			conn.Close()
		} else {
			require.Error(t, err)
		}
	}
	check(oldCA.pem, true)
	current.Store(&newCert)
	check(oldCA.pem, false)
	check(append(append([]byte{}, oldCA.pem...), newCA.pem...), true)
}
func TestP009MissingSystemTrustFailsClosed(t *testing.T) {
	ca := p009NewCA(t)
	for _, load := range []func() (*x509.CertPool, error){func() (*x509.CertPool, error) { return nil, errors.New("unavailable fixture") }, func() (*x509.CertPool, error) { return nil, nil }} {
		cfg, err := tlsConfigWithRoots(ca.pem, "synthetic.pem", load)
		require.Nil(t, cfg)
		require.Error(t, err)
	}
}
func TestP009HeaderControlsAndBounds(t *testing.T) {
	for _, raw := range []string{"X-Synthetic: secret\v", "\fX-Synthetic: secret", "X-Synthetic: secret\x00", "X-Synthetic: " + strings.Repeat("x", 65537)} {
		headers, err := parseHubHeaders(raw)
		require.Nil(t, headers)
		require.ErrorIs(t, err, errInvalidHubHeaders)
		require.NotContains(t, err.Error(), "secret")
	}
	for _, raw := range []string{"x-token: evil, X-TOKEN: other\r\nx-beszel: 9.0.0", "X-Token: evil\nX-Beszel: 9.0.0", "x-empty:, x-multi: one, X-MULTI: two, Authorization: a:b"} {
		t.Setenv("BESZEL_AGENT_HUB_HEADERS", raw)
		h, err := getHubHeaders()
		require.NoError(t, err)
		u, _ := url.Parse("https://localhost")
		c := &WebSocketClient{hubURL: u, token: "actual-synthetic", hubHeaders: h}
		require.Equal(t, []string{"actual-synthetic"}, c.getOptions().RequestHeader.Values("X-Token"))
		require.Equal(t, 1, len(c.getOptions().RequestHeader.Values("X-Beszel")))
	}
}
func TestP009EndpointSchemes(t *testing.T) {
	a := createTestAgent(t)
	for _, scheme := range []string{"ftp", "file", "ssh", "wss", "ws", "https", "http"} {
		t.Run(scheme, func(t *testing.T) {
			t.Setenv("BESZEL_AGENT_HUB_URL", scheme+"://localhost:12345")
			t.Setenv("BESZEL_AGENT_TOKEN", "synthetic")
			_, err := newWebSocketClient(a)
			if scheme == "ftp" || scheme == "file" || scheme == "ssh" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
