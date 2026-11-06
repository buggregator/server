package module

// Broadcaster publishes an event on a WebSocket channel: the ws.Hub.
type Broadcaster interface {
	Broadcast(channel, eventName string, payload any)
}

// Broadcasting is implemented by a module that sends notices of its own to
// the browser, beside the events it stores through the EventService - for
// example that it stored rows that create no feed card. It is optional:
// WireBroadcaster hands every module that implements it the hub at startup.
type Broadcasting interface {
	SetBroadcaster(b Broadcaster)
}

// WireBroadcaster gives the broadcaster to every registered module that
// implements Broadcasting.
func (r *Registry) WireBroadcaster(b Broadcaster) {
	for _, m := range r.modules {
		if bm, ok := m.(Broadcasting); ok {
			bm.SetBroadcaster(b)
		}
	}
}
