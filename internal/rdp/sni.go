package rdp

import (
	"crypto/tls"
)

func extractSNI(conn *tls.Conn) string {
	state := conn.ConnectionState()
	if len(state.ServerName) > 0 {
		return state.ServerName
	}
	return ""
}
