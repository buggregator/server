package app

import (
	"net"
	"strings"
	"testing"
)

func TestListenHTTP_BusyPortFails(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { busy.Close() })
	addr := busy.Addr().String()

	ln, err := listenHTTP(addr)
	if err == nil {
		ln.Close()
		t.Fatalf("listenHTTP(%s) on a busy port: want an error", addr)
	}
	for _, want := range []string{addr, "-http-addr", "HTTP_ADDR"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

func TestListenHTTP_FreePortBinds(t *testing.T) {
	ln, err := listenHTTP("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if _, err := net.Listen("tcp", ln.Addr().String()); err == nil {
		t.Fatal("port is not held after listenHTTP returns")
	}
}
