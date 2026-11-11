package smtp

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/server/tcp"
)

type nopStorer struct{}

func (nopStorer) HandleIncoming(context.Context, *event.Incoming) error { return nil }

// A busy SMTP port fails startup, as for the other TCP servers, instead of
// leaving the server up without SMTP.
func TestSMTPStart_BusyPortFailsStartup(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	addr := busy.Addr().String()

	mod := New(addr, nil, nil)
	mod.SetEventService(nopStorer{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = tcp.NewManager(mod.TCPServers(), nil).Start(ctx)
	if err == nil {
		t.Fatal("startup on a busy SMTP port: no error")
	}
	if !strings.Contains(err.Error(), "smtp") || !strings.Contains(err.Error(), addr) {
		t.Errorf("error = %q, want it to name smtp and %s", err, addr)
	}
}

// A free port is bound when Start returns: a client connects at once.
func TestSMTPStart_BindsBeforeReturn(t *testing.T) {
	free, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := free.Addr().String()
	free.Close()

	srv := newSMTPServer(addr, nil, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial right after Start: %v", err)
	}
	conn.Close()
}
