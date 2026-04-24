package tunnel

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gateway/internal/protocol"
)

var (
	ErrFrameTooShort  = errors.New("frame too short")
	ErrConnClosed     = errors.New("connection closed")
	ErrSendBufferFull = errors.New("send buffer full")
)

type TunnelConn struct {
	Token       string
	Conn        *websocket.Conn
	ClientIP    string
	ClientVer   string
	PublicKey   []byte
	ConnectedAt time.Time
	LastSeenAt  time.Time

	mu          sync.Mutex
	pendingExec map[string]chan protocol.EventPayload
	sendCh      chan []byte
	closed      chan struct{}
	closeOnce   sync.Once
}

func NewTunnelConn(token string, conn *websocket.Conn, clientIP, clientVer string, publicKey []byte) *TunnelConn {
	now := time.Now()
	return &TunnelConn{
		Token:       token,
		Conn:        conn,
		ClientIP:    clientIP,
		ClientVer:   clientVer,
		PublicKey:   publicKey,
		ConnectedAt: now,
		LastSeenAt:  now,
		pendingExec: make(map[string]chan protocol.EventPayload),
		sendCh:      make(chan []byte, 256),
		closed:      make(chan struct{}),
	}
}

func (tc *TunnelConn) Close() {
	tc.closeOnce.Do(func() {
		close(tc.closed)
		tc.Conn.Close()
	})
}

func (tc *TunnelConn) SendFrame(frame Frame) error {
	data, err := frame.Marshal()
	if err != nil {
		return err
	}

	select {
	case tc.sendCh <- data:
		return nil
	case <-tc.closed:
		return ErrConnClosed
	default:
		return ErrSendBufferFull
	}
}

func (tc *TunnelConn) WritePump() {
	for {
		select {
		case data := <-tc.sendCh:
			if err := tc.Conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				return
			}
		case <-tc.closed:
			return
		}
	}
}

func (tc *TunnelConn) UpdateLastSeen() {
	tc.mu.Lock()
	tc.LastSeenAt = time.Now()
	tc.mu.Unlock()
}

func (tc *TunnelConn) RegisterPendingExec(reqID string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.pendingExec[reqID] = make(chan protocol.EventPayload, 1)
}

func (tc *TunnelConn) NotifyExecResult(reqID string, payload protocol.EventPayload) {
	tc.mu.Lock()
	ch, ok := tc.pendingExec[reqID]
	if ok {
		delete(tc.pendingExec, reqID)
	}
	tc.mu.Unlock()

	if ok {
		ch <- payload
	}
}

func (tc *TunnelConn) Stats() map[string]interface{} {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	return map[string]interface{}{
		"token":        tc.Token,
		"status":       "online",
		"client_ip":    tc.ClientIP,
		"client_ver":   tc.ClientVer,
		"connected_at": tc.ConnectedAt.Unix(),
		"last_seen_at": tc.LastSeenAt.Unix(),
	}
}
