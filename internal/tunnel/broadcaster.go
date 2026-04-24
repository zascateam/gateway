package tunnel

import "gateway/internal/protocol"

type EventBroadcaster interface {
	Broadcast(event protocol.Event)
}
