package rdp

import (
	"crypto/rand"
	"encoding/hex"
)

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
