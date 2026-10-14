package transport

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestP009SSHCloseCancelsSetup(t *testing.T) {
	tr, reached, closed := newSSHTestTransport(t, "handshake")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := tr.Connect(ctx); done <- err }()
	select {
	case <-reached:
	case <-ctx.Done():
		t.Fatal("handshake not reached")
	}
	tr.Close()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-ctx.Done():
		t.Fatal("transport Close did not cancel handshake")
	}
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal("setup connection not closed")
	}
	require.False(t, tr.IsConnected())
	// Close retires current setup; it does not permanently disable reconnection.
	client, err := tr.Connect(t.Context())
	require.NoError(t, err)
	require.Same(t, client, tr.GetClient())
}
