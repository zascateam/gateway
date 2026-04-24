package rdp

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"gateway/internal/protocol"
	"gateway/internal/tunnel"
	"gateway/pkg/logger"
)

type Router struct {
	mu         sync.RWMutex
	routes     map[string]string
	sessions   map[string]*RDPSession
	pool       *tunnel.Pool
	broadcaster EventBroadcaster
}

func NewRouter(pool *tunnel.Pool, broadcaster EventBroadcaster) *Router {
	return &Router{
		routes:      make(map[string]string),
		sessions:    make(map[string]*RDPSession),
		pool:        pool,
		broadcaster: broadcaster,
	}
}

func (r *Router) Bind(domain, token string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[domain] = token
	logger.Info("RDP route bound", "domain", domain, "token", token)
}

func (r *Router) Unbind(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.routes, domain)
	logger.Info("RDP route unbound", "domain", domain)
}

func (r *Router) Lookup(domain string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	token, ok := r.routes[domain]
	return token, ok
}

func (r *Router) HandleConnection(clientConn net.Conn, tlsConn *tls.Conn) {
	domain := extractSNI(tlsConn)
	if domain == "" {
		clientConn.Close()
		return
	}

	token, ok := r.Lookup(domain)
	if !ok {
		logger.Warn("RDP domain not found", "domain", domain)
		clientConn.Close()
		return
	}

	tc := r.pool.Get(token)
	if tc == nil {
		logger.Warn("tunnel not found for RDP domain", "domain", domain, "token", token)
		clientConn.Close()
		return
	}

	session := &RDPSession{
		ID:          generateSessionID(),
		Domain:      domain,
		Token:       token,
		ClientIP:    clientConn.RemoteAddr().String(),
		StartedAt:   0,
		BytesIn:     0,
		BytesOut:    0,
		clientConn:  tlsConn,
		tunnelConn:  tc,
		broadcaster: r.broadcaster,
	}

	r.mu.Lock()
	r.sessions[session.ID] = session
	r.mu.Unlock()

	r.broadcaster.BroadcastEvent(protocol.Event{
		Type: protocol.EventRDPConnect,
		Payload: protocol.EventPayload{
			Token:    token,
			Domain:   domain,
			ClientIP: session.ClientIP,
		},
	})

	go session.proxy()
}

func (r *Router) RemoveSession(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
}

type RDPSession struct {
	ID          string
	Domain      string
	Token       string
	ClientIP    string
	StartedAt   int64
	BytesIn     int64
	BytesOut    int64

	clientConn  *tls.Conn
	tunnelConn  *tunnel.TunnelConn
	broadcaster EventBroadcaster
}

func (s *RDPSession) proxy() {
	defer func() {
		s.clientConn.Close()
		s.broadcaster.BroadcastEvent(protocol.Event{
			Type: protocol.EventRDPDisconnect,
			Payload: protocol.EventPayload{
				Token:    s.Token,
				Domain:   s.Domain,
				ClientIP: s.ClientIP,
				Duration: s.StartedAt,
				BytesIn:  atomic.LoadInt64(&s.BytesIn),
				BytesOut: atomic.LoadInt64(&s.BytesOut),
			},
		})
	}()

	buf := make([]byte, 32*1024)
	for {
		n, err := s.clientConn.Read(buf)
		if err != nil {
			if err != io.EOF {
				logger.Error("RDP client read error", "session", s.ID, "err", err)
			}
			return
		}

		atomic.AddInt64(&s.BytesIn, int64(n))

		frame := tunnel.Frame{
			Channel: tunnel.ChannelRDP,
			Payload: buf[:n],
		}

		if err := s.tunnelConn.SendFrame(frame); err != nil {
			logger.Error("RDP tunnel send error", "session", s.ID, "err", err)
			return
		}
	}
}

func (s *RDPSession) WriteBack(data []byte) (int, error) {
	n, err := s.clientConn.Write(data)
	if err != nil {
		return n, err
	}
	atomic.AddInt64(&s.BytesOut, int64(n))
	return n, nil
}
