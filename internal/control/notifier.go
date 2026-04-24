package control

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
	"gateway/internal/protocol"
	"gateway/pkg/logger"
)

type Notifier struct {
	mu    sync.Mutex
	conns []net.Conn
}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (n *Notifier) AddConn(conn net.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.conns = append(n.conns, conn)
	logger.Info("notifier: client connected", "addr", conn.RemoteAddr().String())
}

func (n *Notifier) RemoveConn(conn net.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i, c := range n.conns {
		if c == conn {
			n.conns = append(n.conns[:i], n.conns[i+1:]...)
			break
		}
	}
}

func (n *Notifier) Broadcast(event protocol.Event) {
	data, err := msgpack.Marshal(event)
	if err != nil {
		logger.Error("notifier marshal event error", "err", err)
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	var failed []net.Conn
	for _, conn := range n.conns {
		if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err != nil {
			failed = append(failed, conn)
			continue
		}
		if _, err := conn.Write(data); err != nil {
			failed = append(failed, conn)
		}
	}

	for _, conn := range failed {
		conn.Close()
		n.removeConnUnlocked(conn)
	}
}

func (n *Notifier) BroadcastEvent(event interface{}) {
	if e, ok := event.(protocol.Event); ok {
		n.Broadcast(e)
	}
}

func (n *Notifier) removeConnUnlocked(conn net.Conn) {
	for i, c := range n.conns {
		if c == conn {
			n.conns = append(n.conns[:i], n.conns[i+1:]...)
			break
		}
	}
}
