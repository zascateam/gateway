package rdp

type EventBroadcaster interface {
	BroadcastEvent(event interface{})
}
