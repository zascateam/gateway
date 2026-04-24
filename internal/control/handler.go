package control

import (
	"gateway/internal/protocol"
	"gateway/internal/tunnel"
	"gateway/pkg/logger"
)

type DomainRouter interface {
	Bind(domain, token string)
	Unbind(domain string)
}

type Handler struct {
	pool   *tunnel.Pool
	router DomainRouter
}

func NewHandler(pool *tunnel.Pool, router DomainRouter) *Handler {
	return &Handler{
		pool:   pool,
		router: router,
	}
}

func (h *Handler) Handle(cmd protocol.Command) protocol.Response {
	switch cmd.Type {
	case protocol.CmdDomainBind:
		return h.handleDomainBind(cmd)
	case protocol.CmdDomainUnbind:
		return h.handleDomainUnbind(cmd)
	case protocol.CmdTunnelKick:
		return h.handleTunnelKick(cmd)
	case protocol.CmdTunnelStats:
		return h.handleTunnelStats(cmd)
	case protocol.CmdRemoteExec:
		return h.handleRemoteExec(cmd)
	default:
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "unknown command type",
		}
	}
}

func (h *Handler) handleDomainBind(cmd protocol.Command) protocol.Response {
	domain := cmd.Payload.Domain
	token := cmd.Payload.Token

	if domain == "" || token == "" {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "domain and token are required",
		}
	}

	conn := h.pool.Get(token)
	if conn == nil {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "tunnel not found",
		}
	}

	h.router.Bind(domain, token)
	logger.Info("domain bound", "domain", domain, "token", token)

	return protocol.Response{
		ReqID:   cmd.ReqID,
		Success: true,
	}
}

func (h *Handler) handleDomainUnbind(cmd protocol.Command) protocol.Response {
	domain := cmd.Payload.Domain

	if domain == "" {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "domain is required",
		}
	}

	h.router.Unbind(domain)
	logger.Info("domain unbound", "domain", domain)

	return protocol.Response{
		ReqID:   cmd.ReqID,
		Success: true,
	}
}

func (h *Handler) handleTunnelKick(cmd protocol.Command) protocol.Response {
	token := cmd.Payload.Token

	if token == "" {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "token is required",
		}
	}

	conn := h.pool.Get(token)
	if conn == nil {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "tunnel not found",
		}
	}

	conn.Close()
	h.pool.Remove(token)
	logger.Info("tunnel kicked", "token", token)

	return protocol.Response{
		ReqID:   cmd.ReqID,
		Success: true,
	}
}

func (h *Handler) handleTunnelStats(cmd protocol.Command) protocol.Response {
	token := cmd.Payload.Token

	if token == "" {
		stats := h.pool.Stats()
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: true,
			Data:    stats,
		}
	}

	conn := h.pool.Get(token)
	if conn == nil {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "tunnel not found",
		}
	}

	stat := conn.Stats()
	return protocol.Response{
		ReqID:   cmd.ReqID,
		Success: true,
		Data:    stat,
	}
}

func (h *Handler) handleRemoteExec(cmd protocol.Command) protocol.Response {
	token := cmd.Payload.Token

	if token == "" {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "token is required",
		}
	}

	conn := h.pool.Get(token)
	if conn == nil {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "tunnel not found",
		}
	}

	reqID := cmd.ReqID
	frame := tunnel.Frame{
		Channel: tunnel.ChannelRemoteExec,
		Payload: cmd.Payload.Script,
	}

	if err := conn.SendFrame(frame); err != nil {
		return protocol.Response{
			ReqID:   cmd.ReqID,
			Success: false,
			Error:   "failed to send remote_exec frame: " + err.Error(),
		}
	}

	conn.RegisterPendingExec(reqID)

	logger.Info("remote_exec sent", "token", token, "req_id", reqID)

	return protocol.Response{
		ReqID:   cmd.ReqID,
		Success: true,
	}
}
