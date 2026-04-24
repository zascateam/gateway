package control

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
	"gateway/internal/protocol"
	"gateway/pkg/logger"
)

type Server struct {
	socketPath string
	listener   net.Listener
	handler    *Handler
	notifier   *Notifier

	mu      sync.Mutex
	conns   map[net.Conn]struct{}
	closed  bool
}

func NewServer(socketPath string, handler *Handler, notifier *Notifier) *Server {
	return &Server{
		socketPath: socketPath,
		handler:    handler,
		notifier:   notifier,
		conns:      make(map[net.Conn]struct{}),
	}
}

func (s *Server) Start() error {
	osRemoveSocket(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	s.listener = ln

	logger.Info("control socket listening", "path", s.socketPath)

	go s.acceptLoop()
	return nil
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	if s.listener != nil {
		s.listener.Close()
	}
	for conn := range s.conns {
		conn.Close()
	}
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			logger.Error("control socket accept error", "err", err)
			continue
		}

		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		conn.Close()
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	for {
		var length uint32
		if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
			if err != io.EOF {
				logger.Error("control socket read length error", "err", err)
			}
			return
		}

		if length > 10*1024*1024 {
			logger.Error("control socket frame too large", "length", length)
			return
		}

		buf := make([]byte, length)
		if _, err := io.ReadFull(conn, buf); err != nil {
			logger.Error("control socket read payload error", "err", err)
			return
		}

		var cmd protocol.Command
		if err := msgpack.Unmarshal(buf, &cmd); err != nil {
			logger.Error("control socket unmarshal command error", "err", err)
			continue
		}

		resp := s.handler.Handle(cmd)

		respData, err := msgpack.Marshal(resp)
		if err != nil {
			logger.Error("control socket marshal response error", "err", err)
			continue
		}

		if err := binary.Write(conn, binary.BigEndian, uint32(len(respData))); err != nil {
			logger.Error("control socket write response length error", "err", err)
			return
		}
		if _, err := conn.Write(respData); err != nil {
			logger.Error("control socket write response error", "err", err)
			return
		}
	}
}

func osRemoveSocket(path string) {
	os.Remove(path)
}
