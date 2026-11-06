package module

import "testing"

type nopBroadcaster struct{}

func (nopBroadcaster) Broadcast(string, string, any) {}

type broadcastingModule struct {
	BaseModule
	got Broadcaster
}

func (m *broadcastingModule) Name() string                 { return "broadcasting" }
func (m *broadcastingModule) Type() string                 { return "broadcasting" }
func (m *broadcastingModule) SetBroadcaster(b Broadcaster) { m.got = b }

type plainModule struct{ BaseModule }

func (plainModule) Name() string { return "plain" }
func (plainModule) Type() string { return "plain" }

func TestRegistry_WireBroadcaster(t *testing.T) {
	r := NewRegistry()
	bm := &broadcastingModule{}
	r.Register(plainModule{})
	r.Register(bm)

	r.WireBroadcaster(nopBroadcaster{})

	if bm.got == nil {
		t.Error("a module implementing Broadcasting did not get the broadcaster")
	}
}
