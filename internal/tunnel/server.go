package tunnel

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
	"gateway/internal/protocol"
	"gateway/pkg/logger"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type Server struct {
	port      int
	pool      *Pool
	heartbeat *Heartbeat
	broadcaster EventBroadcaster
}

func NewServer(port int, pool *Pool, heartbeat *Heartbeat, broadcaster EventBroadcaster) *Server {
	return &Server{
		port:        port,
		pool:        pool,
		heartbeat:   heartbeat,
		broadcaster: broadcaster,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/tunnel", s.handleTunnel)

	server := &http.Server{
		Addr:    ":9000",
		Handler: mux,
	}

	logger.Info("WSS tunnel server starting", "port", s.port)
	return server.ListenAndServe()
}

func (s *Server) handleTunnel(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WSS upgrade failed", "err", err)
		return
	}

	clientIP := r.RemoteAddr
	clientVer := r.URL.Query().Get("ver")
	publicKey := r.URL.Query().Get("pubkey")

	tc := NewTunnelConn(token, conn, clientIP, clientVer, []byte(publicKey))

	s.pool.Add(token, tc)
	s.heartbeat.Register(token, tc)

	s.broadcaster.Broadcast(protocol.Event{
		Type: protocol.EventTunnelOnline,
		Payload: protocol.EventPayload{
			Token:     token,
			ClientIP:  clientIP,
			ClientVer: clientVer,
			PublicKey: tc.PublicKey,
		},
	})

	logger.Info("tunnel connected", "token", token, "ip", clientIP)

	go tc.WritePump()
	go s.readPump(tc)
}

func (s *Server) readPump(tc *TunnelConn) {
	defer func() {
		tc.Close()
		s.pool.Remove(tc.Token)
		s.heartbeat.Unregister(tc.Token)

		s.broadcaster.Broadcast(protocol.Event{
			Type: protocol.EventTunnelOffline,
			Payload: protocol.EventPayload{
				Token: tc.Token,
			},
		})

		logger.Info("tunnel disconnected", "token", tc.Token)
	}()

	for {
		_, message, err := tc.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Error("tunnel read error", "token", tc.Token, "err", err)
			}
			return
		}

		tc.UpdateLastSeen()
		s.heartbeat.Reset(tc.Token)

		frame, err := parseFrame(message)
		if err != nil {
			logger.Error("tunnel parse frame error", "token", tc.Token, "err", err)
			continue
		}

		switch frame.Channel {
		case ChannelControl:
			logger.Debug("tunnel control frame", "token", tc.Token, "len", len(frame.Payload))
		case ChannelRemoteExec:
			s.handleRemoteExecResult(tc, frame)
		default:
			logger.Warn("tunnel unknown channel", "token", tc.Token, "channel", frame.Channel)
		}
	}
}

func (s *Server) handleRemoteExecResult(tc *TunnelConn, frame Frame) {
	var result struct {
		ReqID    string `msgpack:"req_id"`
		Stdout   []byte `msgpack:"stdout"`
		Stderr   []byte `msgpack:"stderr"`
		ExitCode int    `msgpack:"exit_code"`
	}

	if err := msgpack.Unmarshal(frame.Payload, &result); err != nil {
		logger.Error("remote_exec result unmarshal error", "token", tc.Token, "err", err)
		return
	}

	tc.NotifyExecResult(result.ReqID, protocol.EventPayload{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		ReqID:    result.ReqID,
	})

	s.broadcaster.Broadcast(protocol.Event{
		Type: protocol.EventRemoteExecResult,
		Payload: protocol.EventPayload{
			Token:    tc.Token,
			ReqID:    result.ReqID,
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
		},
	})

	logger.Info("remote_exec result received", "token", tc.Token, "req_id", result.ReqID, "exit_code", result.ExitCode)
}

func parseFrame(data []byte) (Frame, error) {
	if len(data) < 3 {
		return Frame{}, ErrFrameTooShort
	}

	length := uint16(data[0])<<8 | uint16(data[1])
	channel := data[2]

	var payload []byte
	if length > 0 && len(data) > 3 {
		payload = data[3:]
	}

	return Frame{
		Channel: channel,
		Payload: payload,
	}, nil
}
