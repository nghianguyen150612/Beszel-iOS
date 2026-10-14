//go:build testing

package ws

import (
	"bytes"
	"context"
	"fmt"
	"github.com/blang/semver"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/lxzan/gws"
	"github.com/stretchr/testify/require"
)

func p009Message(t *testing.T, v any) *gws.Message {
	t.Helper()
	b, err := cbor.Marshal(v)
	require.NoError(t, err)
	return &gws.Message{Opcode: gws.OpcodeBinary, Data: bytes.NewBuffer(b)}
}
func p009Pending(rm *RequestManager, id RequestID) *PendingRequest {
	ctx, cancel := context.WithCancel(context.Background())
	req := &PendingRequest{ID: id, Context: ctx, Cancel: cancel, ResponseCh: make(chan *gws.Message, 1), CreatedAt: time.Now()}
	rm.pendingReqs[id] = req
	return req
}
func TestP009DuplicateResponseDoesNotBlock(t *testing.T) {
	rm := NewRequestManager(nil)
	defer rm.Close()
	req := p009Pending(rm, 1)
	defer req.Cancel()
	id := uint32(1)
	// Fill the slot to model a second response racing the first delivery.
	req.ResponseCh <- p009Message(t, common.AgentResponse{Id: &id})
	done := make(chan struct{})
	go func() { rm.handleResponse(p009Message(t, common.AgentResponse{Id: &id})); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		req.Cancel()
		<-done
		t.Fatal("duplicate response blocked callback")
	}
	(<-req.ResponseCh).Close()
	require.Zero(t, rm.GetPendingCount())
}
func TestP009AmbiguousLegacyResponse(t *testing.T) {
	rm := NewRequestManager(nil)
	defer rm.Close()
	a, b := p009Pending(rm, 1), p009Pending(rm, 2)
	rm.routeLegacyResponse(p009Message(t, map[string]string{"sample": "unassociated"}))
	require.Empty(t, a.ResponseCh, "no-ID payload cannot be associated with parallel requests")
	require.Empty(t, b.ResponseCh)
}
func TestP009ResponseCancellationRace(t *testing.T) {
	for range 200 {
		rm := NewRequestManager(nil)
		req := p009Pending(rm, 1)
		go rm.cleanupRequest(req)
		id := uint32(1)
		msg := p009Message(t, common.AgentResponse{Id: &id})
		var wg sync.WaitGroup
		wg.Go(func() { rm.handleResponse(msg) })
		wg.Go(req.Cancel)
		wg.Wait()
		rm.Close()
		require.Eventually(t, func() bool { return len(req.ResponseCh) == 0 }, time.Second, time.Millisecond, "abandoned response must be released")
	}
}

type p009RequestAgent struct {
	gws.BuiltinEventHandler
	requests chan common.HubRequest[cbor.RawMessage]
}

func (a *p009RequestAgent) OnMessage(_ *gws.Conn, msg *gws.Message) {
	defer msg.Close()
	var req common.HubRequest[cbor.RawMessage]
	if cbor.Unmarshal(msg.Bytes(), &req) == nil {
		a.requests <- req
	}
}
func p009RequestPair(t *testing.T, version string) (*WsConn, *gws.Conn, <-chan common.HubRequest[cbor.RawMessage]) {
	t.Helper()
	ready := make(chan *WsConn, 1)
	up := gws.NewUpgrader(&Handler{}, nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r)
		if err != nil {
			return
		}
		wc := NewWsConnection(conn, semver.MustParse(version))
		conn.Session().Store("wsConn", wc)
		ready <- wc
		conn.ReadLoop()
	}))
	a := &p009RequestAgent{requests: make(chan common.HubRequest[cbor.RawMessage], 128)}
	conn, _, err := gws.NewClient(a, &gws.ClientOption{Addr: "ws" + strings.TrimPrefix(server.URL, "http")})
	require.NoError(t, err)
	go conn.ReadLoop()
	t.Cleanup(func() { conn.NetConn().Close(); server.Close() })
	select {
	case wc := <-ready:
		return wc, conn, a.requests
	case <-time.After(time.Second):
		t.Fatal("not connected")
		return nil, nil, nil
	}
}
func p009Reply(t *testing.T, conn *gws.Conn, id *uint32, value any) {
	t.Helper()
	data, err := cbor.Marshal(value)
	require.NoError(t, err)
	msg, err := cbor.Marshal(common.AgentResponse{Id: id, Data: data})
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(gws.OpcodeBinary, msg))
}
func TestP009ConcurrentOutOfOrderResponses(t *testing.T) {
	wc, conn, requests := p009RequestPair(t, "0.21.0")
	pending := make([]*PendingRequest, 64)
	var wg sync.WaitGroup
	for i := range pending {
		wg.Go(func() {
			req, err := wc.SendRequest(t.Context(), common.GetData, i)
			if err != nil {
				t.Error(err)
				return
			}
			pending[i] = req
		})
	}
	wg.Wait()
	received := make([]common.HubRequest[cbor.RawMessage], 64)
	seen := map[uint32]bool{}
	for i := range received {
		select {
		case r := <-requests:
			require.NotNil(t, r.Id)
			require.False(t, seen[*r.Id])
			seen[*r.Id] = true
			received[i] = r
		case <-time.After(time.Second):
			t.Fatal("request lost")
		}
	}
	for i := len(received) - 1; i >= 0; i-- {
		p009Reply(t, conn, received[i].Id, *received[i].Id)
	}
	for _, r := range pending {
		select {
		case msg := <-r.ResponseCh:
			var resp common.AgentResponse
			require.NoError(t, cbor.Unmarshal(msg.Bytes(), &resp))
			msg.Close()
			require.Equal(t, uint32(r.ID), *resp.Id)
			var payload uint32
			require.NoError(t, cbor.Unmarshal(resp.Data, &payload))
			require.Equal(t, uint32(r.ID), payload)
			r.Cancel()
		case <-time.After(time.Second):
			t.Fatal("response not correlated")
		}
	}
	require.Zero(t, wc.requestManager.GetPendingCount())
}
func TestP009TimeoutLateResponseAndShutdown(t *testing.T) {
	wc, conn, requests := p009RequestPair(t, "0.21.0")
	ctx, cancel := context.WithCancel(t.Context())
	first, err := wc.SendRequest(ctx, common.GetData, nil)
	require.NoError(t, err)
	wireFirst := <-requests
	cancel()
	<-first.Context.Done()
	require.Eventually(t, func() bool { return wc.requestManager.GetPendingCount() == 0 }, time.Second, time.Millisecond)
	second, err := wc.SendRequest(t.Context(), common.GetData, nil)
	require.NoError(t, err)
	wireSecond := <-requests
	p009Reply(t, conn, wireFirst.Id, "late")
	p009Reply(t, conn, wireSecond.Id, "current")
	select {
	case msg := <-second.ResponseCh:
		var resp common.AgentResponse
		require.NoError(t, cbor.Unmarshal(msg.Bytes(), &resp))
		msg.Close()
		second.Cancel()
		require.Equal(t, wireSecond.Id, resp.Id)
	case <-time.After(time.Second):
		t.Fatal("current response lost")
	}
	abandoned, err := wc.SendRequest(t.Context(), common.GetData, nil)
	require.NoError(t, err)
	<-requests
	require.NoError(t, conn.NetConn().Close())
	select {
	case <-abandoned.Context.Done():
	case <-time.After(time.Second):
		t.Fatal("close did not cancel pending request")
	}
	require.Zero(t, wc.requestManager.GetPendingCount())
	_, err = wc.SendRequest(t.Context(), common.GetData, nil)
	require.ErrorIs(t, err, gws.ErrConnClosed)
}
func TestP009LegacySerializationAndCancellation(t *testing.T) {
	for _, abandon := range []bool{false, true} {
		t.Run(fmt.Sprint(abandon), func(t *testing.T) {
			wc, conn, requests := p009RequestPair(t, "0.12.0")
			first, err := wc.SendRequest(t.Context(), common.GetData, "first")
			require.NoError(t, err)
			<-requests
			entered := make(chan struct{})
			done := make(chan *PendingRequest, 1)
			errs := make(chan error, 1)
			go func() {
				close(entered)
				r, e := wc.SendRequest(t.Context(), common.GetData, "second")
				done <- r
				errs <- e
			}()
			<-entered
			require.Equal(t, 1, wc.requestManager.GetPendingCount())
			require.Empty(t, requests, "parallel legacy request must not reach the wire")
			if !abandon {
				body, err := cbor.Marshal(map[uint8]any{0: "legacy-first"})
				require.NoError(t, err)
				require.NoError(t, conn.WriteMessage(gws.OpcodeBinary, body))
				select {
				case msg := <-first.ResponseCh:
					require.Equal(t, body, msg.Bytes())
					msg.Close()
				case <-time.After(time.Second):
					t.Fatal("legacy response not delivered")
				}
			}
			first.Cancel()
			select {
			case second := <-done:
				if abandon {
					require.Nil(t, second)
					require.Error(t, <-errs)
				} else {
					require.NoError(t, <-errs)
					require.NotNil(t, second)
					<-requests
					second.Cancel()
				}
			case <-time.After(time.Second):
				t.Fatal("legacy queue stuck")
			}
		})
	}
}
func TestP009RequestIDWrapDoesNotReplacePending(t *testing.T) {
	wc, _, requests := p009RequestPair(t, "0.21.0")
	first, err := wc.SendRequest(t.Context(), common.GetData, nil)
	require.NoError(t, err)
	defer first.Cancel()
	<-requests
	wc.requestManager.nextID.Store(^uint32(0))
	second, err := wc.SendRequest(t.Context(), common.GetData, nil)
	require.NoError(t, err)
	defer second.Cancel()
	<-requests
	require.NotEqual(t, first.ID, second.ID)
	require.Equal(t, 2, wc.requestManager.GetPendingCount())
}
