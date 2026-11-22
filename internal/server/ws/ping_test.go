package ws

import (
	"testing"
	"time"
)

// A connected client gets a ping ({}) every pingInterval, and its pong ({})
// gets no reply: a reply would read as a ping and start a ping-pong loop.
func TestHub_PingsConnectedClient(t *testing.T) {
	h := NewHub()
	h.pingInterval = 100 * time.Millisecond
	c := dialHub(t, h)
	waitClients(t, h, 1)

	c.send(`{"id":1,"connect":{}}`)
	if m := c.next(); m["id"] != float64(1) || m["connect"] == nil {
		t.Fatalf("connect reply = %v", m)
	}

	for i := 0; i < 3; i++ {
		m := c.next()
		if m == nil || len(m) != 0 {
			t.Fatalf("ping %d = %v, want {}", i, m)
		}
		c.send(`{}`) // pong
	}

	// Pongs are not answered: within one interval only pings arrive, and
	// no more than the interval allows.
	deadline := time.Now().Add(350 * time.Millisecond)
	frames := 0
	for time.Now().Before(deadline) {
		m := c.next()
		if m == nil {
			break
		}
		if len(m) != 0 {
			t.Fatalf("unexpected frame %v", m)
		}
		frames++
		c.send(`{}`)
	}
	if frames > 5 {
		t.Errorf("%d frames in 350 ms: pongs are answered", frames)
	}
}

// No ping before connect: the client does not expect one yet.
func TestHub_NoPingBeforeConnect(t *testing.T) {
	h := NewHub()
	h.pingInterval = 50 * time.Millisecond
	c := dialHub(t, h)
	waitClients(t, h, 1)
	if m := c.next(); m != nil {
		t.Fatalf("frame before connect: %v", m)
	}
}

// A ping command with an id (old clients) is still answered with its id.
func TestHub_PingWithIDAnswered(t *testing.T) {
	h := NewHub()
	c := dialHub(t, h)
	waitClients(t, h, 1)
	c.send(`{"id":1,"connect":{}}`)
	c.next()
	c.send(`{"id":7}`)
	if m := c.next(); m["id"] != float64(7) {
		t.Fatalf("reply = %v, want id 7", m)
	}
}
