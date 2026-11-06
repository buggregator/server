package ws

import "testing"

func TestHandleConnect_ServerChannels(t *testing.T) {
	h := NewHub()
	h.SetProjectProvider(func() []string { return []string{"default"} })
	h.AddServerChannel("xdebug")
	reply := h.handleConnect(1)
	for _, ch := range []string{"events.project.default", "xdebug"} {
		if _, ok := reply.Connect.Subs[ch]; !ok {
			t.Errorf("missing server-side subscription %q: %v", ch, reply.Connect.Subs)
		}
	}
}
