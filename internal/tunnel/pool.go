package tunnel

import (
	"sync"
)

type Pool struct {
	mu    sync.RWMutex
	conns map[string]*TunnelConn
}

func NewPool() *Pool {
	return &Pool{
		conns: make(map[string]*TunnelConn),
	}
}

func (p *Pool) Add(token string, conn *TunnelConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if old, exists := p.conns[token]; exists {
		old.Close()
	}
	p.conns[token] = conn
}

func (p *Pool) Get(token string) *TunnelConn {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.conns[token]
}

func (p *Pool) Remove(token string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.conns, token)
}

func (p *Pool) Stats() []map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var stats []map[string]interface{}
	for _, conn := range p.conns {
		stats = append(stats, conn.Stats())
	}
	return stats
}

func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.conns)
}

func (p *Pool) ForEach(fn func(token string, conn *TunnelConn)) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for token, conn := range p.conns {
		fn(token, conn)
	}
}
