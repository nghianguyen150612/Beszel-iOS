package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"github.com/blang/semver"
	"golang.org/x/crypto/ssh"
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

func TestP009SSHRejectsWrongHostKey(t *testing.T) {
	tr, _, _ := newSSHTestTransport(t, "")
	expected := tr.config.HostKeyCallback
	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	wrong, err := ssh.NewSignerFromKey(key)
	require.NoError(t, err)
	tr.config.HostKeyCallback = ssh.FixedHostKey(wrong.PublicKey())
	_, err = tr.Connect(t.Context())
	require.ErrorContains(t, err, "host key mismatch")
	require.False(t, tr.IsConnected())
	tr.config.HostKeyCallback = expected
	var version semver.Version
	tr.onConnect = func(v semver.Version) { version = v }
	_, err = tr.Connect(t.Context())
	require.NoError(t, err)
	require.Equal(t, "0.20.0", version.String())
}
