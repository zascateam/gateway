package protocol

type CommandType string

const (
	CmdDomainBind   CommandType = "domain_bind"
	CmdDomainUnbind CommandType = "domain_unbind"
	CmdTunnelKick   CommandType = "tunnel_kick"
	CmdTunnelStats  CommandType = "tunnel_stats"
	CmdRemoteExec   CommandType = "remote_exec"
)

type EventType string

const (
	EventTunnelOnline    EventType = "tunnel_online"
	EventTunnelOffline   EventType = "tunnel_offline"
	EventRDPConnect      EventType = "rdp_connect"
	EventRDPDisconnect   EventType = "rdp_disconnect"
	EventRemoteExecResult EventType = "remote_exec_result"
)

type Command struct {
	Type     CommandType     `msgpack:"type"`
	ReqID    string          `msgpack:"req_id"`
	Payload  CommandPayload  `msgpack:"payload"`
}

type CommandPayload struct {
	Token       string `msgpack:"token,omitempty"`
	Domain      string `msgpack:"domain,omitempty"`
	Script      []byte `msgpack:"script,omitempty"`
	EncryptedKey []byte `msgpack:"encrypted_key,omitempty"`
	Signature   []byte `msgpack:"signature,omitempty"`
	PubKeyID    string `msgpack:"pub_key_id,omitempty"`
}

type Event struct {
	Type    EventType     `msgpack:"type"`
	Payload EventPayload  `msgpack:"payload"`
}

type EventPayload struct {
	Token       string `msgpack:"token,omitempty"`
	Domain      string `msgpack:"domain,omitempty"`
	ClientIP    string `msgpack:"client_ip,omitempty"`
	ClientVer   string `msgpack:"client_ver,omitempty"`
	PublicKey   []byte `msgpack:"public_key,omitempty"`
	Stdout      []byte `msgpack:"stdout,omitempty"`
	Stderr      []byte `msgpack:"stderr,omitempty"`
	ExitCode    int    `msgpack:"exit_code,omitempty"`
	ReqID       string `msgpack:"req_id,omitempty"`
	Duration    int64  `msgpack:"duration,omitempty"`
	BytesIn     int64  `msgpack:"bytes_in,omitempty"`
	BytesOut    int64  `msgpack:"bytes_out,omitempty"`
}

type Response struct {
	ReqID   string      `msgpack:"req_id"`
	Success bool        `msgpack:"success"`
	Error   string      `msgpack:"error,omitempty"`
	Data    interface{} `msgpack:"data,omitempty"`
}

type TunnelStatsData struct {
	Token        string `msgpack:"token"`
	Status       string `msgpack:"status"`
	ClientIP     string `msgpack:"client_ip"`
	ClientVer    string `msgpack:"client_ver"`
	ConnectedAt  int64  `msgpack:"connected_at"`
	LastSeenAt   int64  `msgpack:"last_seen_at"`
	RDPSessions  int    `msgpack:"rdp_sessions"`
}
