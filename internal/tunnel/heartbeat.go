package tunnel

import (
	"sync"
	"time"

	"gateway/pkg/logger"
)

type Heartbeat struct {
	interval time.Duration
	timeout  time.Duration
	pool     *Pool

	mu     sync.Mutex
	timers map[string]*time.Timer
}

func NewHeartbeat(intervalSec, timeoutSec int, pool *Pool) *Heartbeat {
	return &Heartbeat{
		interval: time.Duration(intervalSec) * time.Second,
		timeout:  time.Duration(timeoutSec) * time.Second,
		pool:     pool,
		timers:   make(map[string]*time.Timer),
	}
}

func (h *Heartbeat) Register(token string, conn *TunnelConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	timer := time.AfterFunc(h.timeout, func() {
		h.handleTimeout(token, conn)
	})
	h.timers[token] = timer
}

func (h *Heartbeat) Unregister(token string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if timer, ok := h.timers[token]; ok {
		timer.Stop()
		delete(h.timers, token)
	}
}

func (h *Heartbeat) Reset(token string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if timer, ok := h.timers[token]; ok {
		timer.Reset(h.timeout)
	}
}

func (h *Heartbeat) handleTimeout(token string, conn *TunnelConn) {
	logger.Warn("tunnel heartbeat timeout", "token", token)

	conn.Close()
	h.pool.Remove(token)

	h.mu.Lock()
	delete(h.timers, token)
	h.mu.Unlock()
}
