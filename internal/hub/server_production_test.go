//go:build !development

package hub

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProductionServerDisablesHTTP2(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	disableVulnerableHTTP2Server(server.Config)
	server.StartTLS()
	t.Cleanup(server.Close)

	conn, err := tls.Dial("tcp", server.Listener.Addr().String(), &tls.Config{
		InsecureSkipVerify: true, // httptest uses a self-signed certificate.
		NextProtos:         []string{"h2", "http/1.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if got := conn.ConnectionState().NegotiatedProtocol; got != "http/1.1" {
		t.Fatalf("HTTPS negotiated protocol = %q, want http/1.1", got)
	}
}
