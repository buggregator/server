package mcp

import (
	"context"
	"log/slog"
	"net"
	"os"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// StartListener accepts MCP connections on a Unix socket.
// Each connection becomes an independent MCP session.
// Blocks until ctx is cancelled.
func StartListener(ctx context.Context, socketPath string, server *sdkmcp.Server) error {
	// Remove stale socket file.
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	slog.Info("MCP listener started", "socket", socketPath)

	// Clean up on shutdown.
	go func() {
		<-ctx.Done()
		listener.Close()
		os.Remove(socketPath)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				slog.Error("MCP accept error", "err", err)
				continue
			}
		}

		go func() {
			transport := &sdkmcp.IOTransport{
				Reader: conn,
				Writer: conn,
			}
			session, err := server.Connect(ctx, transport, nil)
			if err != nil {
				slog.Error("MCP session error", "err", err)
				conn.Close()
				return
			}
			slog.Info("MCP client connected")
			if err := session.Wait(); err != nil {
				slog.Debug("MCP session ended", "err", err)
			}
			slog.Info("MCP client disconnected")
		}()
	}
}
