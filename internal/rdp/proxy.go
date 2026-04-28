package rdp

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"gateway/internal/config"
	"gateway/pkg/logger"
)

type Proxy struct {
	cfg    config.RDPConfig
	router *Router

	mu      sync.Mutex
	listener net.Listener
}

func NewProxy(cfg config.RDPConfig, router *Router) *Proxy {
	return &Proxy{
		cfg:    cfg,
		router: router,
	}
}

func (p *Proxy) Start() error {
	cert, err := tls.LoadX509KeyPair(p.cfg.TLSCert, p.cfg.TLSKey)
	if err != nil {
		return err
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	addr := fmt.Sprintf(":%d", p.cfg.Port)
	ln, err := tls.Listen("tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	p.listener = ln

	logger.Info("RDP proxy starting", "port", p.cfg.Port)

	go p.acceptLoop()
	return nil
}

func (p *Proxy) Close() error {
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}

func (p *Proxy) acceptLoop() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			p.mu.Lock()
			closed := p.listener == nil
			p.mu.Unlock()
			if closed {
				return
			}
			logger.Error("RDP proxy accept error", "err", err)
			continue
		}

		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			conn.Close()
			continue
		}

		if err := tlsConn.Handshake(); err != nil {
			logger.Error("RDP TLS handshake error", "err", err)
			conn.Close()
			continue
		}

		go p.router.HandleConnection(conn, tlsConn)
	}
}
