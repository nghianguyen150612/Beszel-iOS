package ws

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/common"
	"github.com/lxzan/gws"
)

// RequestID uniquely identifies a request
type RequestID uint32

// PendingRequest tracks an in-flight request
type PendingRequest struct {
	ID         RequestID
	ResponseCh chan *gws.Message
	Context    context.Context
	Cancel     context.CancelFunc
	CreatedAt  time.Time
}

// RequestManager handles concurrent requests to an agent
type RequestManager struct {
	sync.RWMutex
	conn        *gws.Conn
	pendingReqs map[RequestID]*PendingRequest
	nextID      atomic.Uint32
	closed      bool
	legacy      bool // No-ID peers permit exactly one in-flight operation.
	legacySlot  chan struct{}
}

// NewRequestManager creates a new request manager for a WebSocket connection
func NewRequestManager(conn *gws.Conn) *RequestManager {
	rm := &RequestManager{
		conn:        conn,
		pendingReqs: make(map[RequestID]*PendingRequest),
		legacySlot:  make(chan struct{}, 1),
	}
	return rm
}

// SendRequest sends a request and returns a channel for the response
func (rm *RequestManager) SendRequest(ctx context.Context, action common.WebSocketAction, data any) (*PendingRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if rm.legacy {
		select {
		case rm.legacySlot <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	release := func() {
		if rm.legacy {
			<-rm.legacySlot
		}
	}
	var reqID RequestID

	// Respect any caller-provided deadline. If none is set, apply a reasonable default
	// so pending requests don't live forever if the agent never responds.
	reqCtx := ctx
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		reqCtx, cancel = context.WithCancel(ctx)
	} else {
		reqCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
	}

	req := &PendingRequest{
		ID:         reqID,
		ResponseCh: make(chan *gws.Message, 1),
		Context:    reqCtx,
		Cancel:     cancel,
		CreatedAt:  time.Now(),
	}

	rm.Lock()
	if rm.closed {
		rm.Unlock()
		cancel()
		release()
		return nil, gws.ErrConnClosed
	}
	for {
		reqID = RequestID(rm.nextID.Add(1))
		if _, exists := rm.pendingReqs[reqID]; !exists {
			break
		}
	}
	req.ID = reqID
	rm.pendingReqs[reqID] = req
	rm.Unlock()
	// Watch before sending, so cancellation also bounds a blocked socket write.
	go func() { rm.cleanupRequest(req); release() }()

	hubReq := common.HubRequest[any]{
		Id:     (*uint32)(&reqID),
		Action: action,
		Data:   data,
	}

	// Send the request
	// Cancellation during a blocked send must release the shared socket; stop
	// this callback before waiting for a response, where modern cancellation is
	// request-local and IDs make late replies safe.
	sendClosed := make(chan struct{})
	stopSend := context.AfterFunc(reqCtx, func() {
		if rm.conn != nil {
			_ = rm.conn.NetConn().Close()
		}
		close(sendClosed)
	})
	sendErr := rm.sendMessage(hubReq)
	if !stopSend() {
		<-sendClosed
	}
	if err := sendErr; err != nil {
		rm.cancelRequest(reqID)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return req, nil
}

// sendMessage encodes and sends a message over WebSocket
func (rm *RequestManager) sendMessage(data any) error {
	if rm.conn == nil {
		return gws.ErrConnClosed
	}

	bytes, err := cbor.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	return rm.conn.WriteMessage(gws.OpcodeBinary, bytes)
}

// handleResponse processes a single response message
func (rm *RequestManager) handleResponse(message *gws.Message) {
	if cbor.Valid(message.Data.Bytes()) != nil {
		message.Close()
		return
	}
	if rm.legacy {
		rm.routeLegacyResponse(message)
		return
	}
	var response common.AgentResponse
	if cbor.Unmarshal(message.Data.Bytes(), &response) != nil || response.Id == nil {
		message.Close()
		return
	}
	rm.Lock()
	req := rm.pendingReqs[RequestID(*response.Id)]
	if req != nil {
		delete(rm.pendingReqs, req.ID)
	}
	rm.Unlock()
	rm.deliver(req, message)
}

// No-ID messages are only meaningful on an explicitly serialized legacy peer.
func (rm *RequestManager) routeLegacyResponse(message *gws.Message) {
	rm.Lock()
	var req *PendingRequest
	if rm.legacy && len(rm.pendingReqs) == 1 {
		for _, r := range rm.pendingReqs {
			req = r
			delete(rm.pendingReqs, r.ID)
		}
	}
	rm.Unlock()
	rm.deliver(req, message)
}

func (rm *RequestManager) deliver(req *PendingRequest, message *gws.Message) {
	if req == nil {
		message.Close()
		return
	}
	// Keep cancellation cleanup and delivery ordered: cancellation cannot drain
	// the slot before a concurrent delivery enqueues an abandoned pooled message.
	rm.Lock()
	defer rm.Unlock()
	if req.Context.Err() != nil {
		message.Close()
		return
	}
	select {
	case req.ResponseCh <- message:
	default:
		message.Close()
	}
}

// cleanupRequest releases even a delivered response if its consumer cancels.
func (rm *RequestManager) cleanupRequest(req *PendingRequest) {
	<-req.Context.Done()
	rm.Lock()
	pending := rm.pendingReqs[req.ID] == req
	if pending {
		delete(rm.pendingReqs, req.ID)
	}
	select {
	case msg := <-req.ResponseCh:
		msg.Close()
	default:
	}
	rm.Unlock()
	// A late no-ID response cannot be distinguished from the next request.
	// Retire that connection on abandonment; modern ID peers remain reusable.
	if pending && rm.legacy {
		rm.Close()
		if rm.conn != nil {
			_ = rm.conn.NetConn().Close()
		}
	}
}

// cancelRequest removes a request and cancels its context
func (rm *RequestManager) cancelRequest(reqID RequestID) {
	rm.Lock()
	defer rm.Unlock()

	if req, exists := rm.pendingReqs[reqID]; exists {
		req.Cancel()
		delete(rm.pendingReqs, reqID)
	}
}

// deleteRequest removes a request from the pending map without cancelling its context.
func (rm *RequestManager) deleteRequest(reqID RequestID) {
	rm.Lock()
	defer rm.Unlock()
	delete(rm.pendingReqs, reqID)
}

// Close shuts down the request manager
func (rm *RequestManager) Close() {
	rm.Lock()
	defer rm.Unlock()

	rm.closed = true
	// Cancel all pending requests
	for _, req := range rm.pendingReqs {
		req.Cancel()
	}
	rm.pendingReqs = make(map[RequestID]*PendingRequest)
}
