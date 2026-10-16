//go:build testing

package systems

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver"
	esystem "github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/hub/ws"
	"github.com/lxzan/gws"
	"github.com/stretchr/testify/require"
)

type p014DelayedPeer struct {
	sequenceDataClient
	requested chan struct{}
}

func (p *p014DelayedPeer) OnMessage(conn *gws.Conn, message *gws.Message) {
	close(p.requested)
	p.sequenceDataClient.OnMessage(conn, message)
}

func TestP014CancelledPollDoesNotPersistLateResponse(t *testing.T) {
	sys, app := newTestSystemWithHub(t)
	sys.ctx, sys.cancel = context.WithCancel(context.Background())
	defer sys.cancel()
	sys.Status = up
	connections := make(chan *ws.WsConn, 1)
	upgrader := gws.NewUpgrader(&monitorSyncServer{}, nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r)
		if err != nil {
			t.Error(err)
			return
		}
		wc := ws.NewWsConnection(conn, semver.MustParse("0.21.0"))
		conn.Session().Store("wsConn", wc)
		connections <- wc
		conn.ReadLoop()
	}))
	defer server.Close()
	peer := &p014DelayedPeer{sequenceDataClient: sequenceDataClient{responses: make(chan esystem.CombinedData, 1)}, requested: make(chan struct{})}
	conn, _, err := gws.NewClient(peer, &gws.ClientOption{Addr: "ws" + strings.TrimPrefix(server.URL, "http")})
	require.NoError(t, err)
	defer conn.NetConn().Close()
	go conn.ReadLoop()
	select {
	case sys.WsConn = <-connections:
	case <-time.After(5 * time.Second):
		t.Fatal("fixture did not connect")
	}
	done := make(chan error, 1)
	go func() { done <- sys.update() }()
	select {
	case <-peer.requested:
	case <-time.After(5 * time.Second):
		t.Fatal("Hub did not request telemetry")
	}
	// The old system is removed/replaced while this request is in flight.
	sys.cancel()
	peer.responses <- esystem.CombinedData{Stats: esystem.Stats{Cpu: 99}}
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("late response did not finish the poll")
	}
	count, err := app.CountRecords("system_stats")
	require.NoError(t, err)
	require.Zero(t, count)
}
