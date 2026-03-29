package tcp

import (
	"context"
	"log/slog"
	"net"
	"sync"

	"github.com/buggregator/go-buggregator/internal/metrics"
)

// ConnectionHandler handles a single TCP connection.
type ConnectionHandler interface {
	HandleConnection(conn net.Conn)
}

// Starter is an alternative to ConnectionHandler for servers that
// manage their own listener (e.g., go-smtp).
type Starter interface {
	Start(ctx context.Context) error
	Stop() error
}

// ServerConfig describes a TCP server a module needs.
// Provide either Handler (for generic TCP) or Starter (for self-managed servers).
type ServerConfig struct {
	Name    string
	Address string
	Handler ConnectionHandler // For generic TCP accept loop.
	Starter Starter           // For self-managed servers (SMTP, etc.).
}

// Manager starts and stops TCP servers.
type Manager struct {
	servers  []ServerConfig
	starters []Starter
	wg       sync.WaitGroup
	metrics  *metrics.Collector
}

func NewManager(servers []ServerConfig, m *metrics.Collector) *Manager {
	return &Manager{servers: servers, metrics: m}
}

// Start begins listening on all configured TCP servers.
func (m *Manager) Start(ctx context.Context) error {
	for _, sc := range m.servers {
		if sc.Starter != nil {
			if err := sc.Starter.Start(ctx); err != nil {
				return err
			}
			m.starters = append(m.starters, sc.Starter)
			slog.Info("server started", "name", sc.Name, "addr", sc.Address)
			continue
		}

		ln, err := net.Listen("tcp", sc.Address)
		if err != nil {
			return err
		}
		slog.Info("TCP server started", "name", sc.Name, "addr", sc.Address)

		m.wg.Add(1)
		go m.acceptLoop(ctx, ln, sc)
	}
	return nil
}

func (m *Manager) acceptLoop(ctx context.Context, ln net.Listener, sc ServerConfig) {
	defer m.wg.Done()
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				slog.Error("TCP accept error", "name", sc.Name, "err", err)
				continue
			}
		}
		if m.metrics != nil {
			m.metrics.TCPConnectionsActive.WithLabelValues(sc.Name).Inc()
		}
		go func(c net.Conn) {
			defer func() {
				if m.metrics != nil {
					m.metrics.TCPConnectionsActive.WithLabelValues(sc.Name).Dec()
				}
			}()
			sc.Handler.HandleConnection(c)
		}(conn)
	}
}

// Wait blocks until all accept loops exit and stops starters.
func (m *Manager) Wait() {
	for _, s := range m.starters {
		_ = s.Stop()
	}
	m.wg.Wait()
}
