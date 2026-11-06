package smtp

import (
	"context"
	"errors"
	"net"
	"net/smtp"
	"testing"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	serverhttp "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
)

// loginAuth is a client side of AUTH LOGIN; net/smtp only ships PLAIN.
type loginAuth struct{ username, password string }

func (a loginAuth) Start(*smtp.ServerInfo) (string, []byte, error) { return "LOGIN", nil, nil }

func (a loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch string(fromServer) {
	case "Username:":
		return []byte(a.username), nil
	case "Password:":
		return []byte(a.password), nil
	}
	return nil, errors.New("unexpected challenge " + string(fromServer))
}

// startSMTP runs the real SMTP server on a spare port, storing events through
// the real EventService into an in-memory SQLite store.
func startSMTP(t *testing.T) (addr string, store *storage.SQLiteStore) {
	t.Helper()
	_, store, _ = setupAPITest(t)
	es := serverhttp.NewEventService(store, ws.NewHub(), module.NewRegistry(), nil, nil)

	srv := newSMTPServer("127.0.0.1:0", es, nil, nil)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { srv.server.Serve(l); close(done) }()
	t.Cleanup(func() { srv.server.Close(); <-done })
	return l.Addr().String(), store
}

// send delivers count messages over one connection, authenticating with auth
// when it is not nil.
func send(t *testing.T, addr string, auth smtp.Auth, count int) {
	t.Helper()
	c, err := smtp.Dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Hello("localhost"); err != nil {
		t.Fatal(err)
	}
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			t.Fatalf("AUTH: %v", err)
		}
	}
	for i := 0; i < count; i++ {
		if err := c.Mail("from@example.com"); err != nil {
			t.Fatal(err)
		}
		if err := c.Rcpt("to@example.com"); err != nil {
			t.Fatal(err)
		}
		w, err := c.Data()
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("From: from@example.com\r\nTo: to@example.com\r\nSubject: hi\r\n\r\nbody\r\n"))
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.Quit(); err != nil {
		t.Fatal(err)
	}
}

func storedProjects(t *testing.T, store *storage.SQLiteStore, want int) []string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		events, err := store.FindAll(context.Background(), event.FindOptions{Type: "smtp"})
		if err != nil {
			t.Fatal(err)
		}
		if len(events) >= want || time.Now().After(deadline) {
			projects := make([]string, len(events))
			for i, e := range events {
				projects[i] = e.Project
			}
			return projects
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestSMTPAuthSetsProject: the AUTH username selects the project (#343),
// for every message of the connection; without AUTH mail goes to default.
func TestSMTPAuthSetsProject(t *testing.T) {
	tests := []struct {
		name string
		auth smtp.Auth
		want string
	}{
		{"PLAIN", smtp.PlainAuth("", "plain-proj", "secret", "127.0.0.1"), "plain-proj"},
		{"PLAIN user@host", smtp.PlainAuth("", "proj@smtp", "secret", "127.0.0.1"), "proj"},
		{"LOGIN", loginAuth{"login-proj", "secret"}, "login-proj"},
		{"no AUTH", nil, "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, store := startSMTP(t)
			send(t, addr, tt.auth, 2)

			projects := storedProjects(t, store, 2)
			if len(projects) != 2 {
				t.Fatalf("stored %d events, want 2", len(projects))
			}
			for i, p := range projects {
				if p != tt.want {
					t.Errorf("event %d project = %q, want %q", i, p, tt.want)
				}
			}
		})
	}
}
