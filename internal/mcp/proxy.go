package mcp

import (
	"fmt"
	"io"
	"net"
	"os"
)

// RunProxy connects stdin/stdout to a running Buggregator's MCP Unix socket.
// This is the "buggregator mcp" subcommand — a thin bridge for MCP clients.
func RunProxy(socketPath string) error {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("cannot connect to buggregator at %s: %w (is the main process running?)", socketPath, err)
	}
	defer conn.Close()

	// Bridge: stdin → socket, socket → stdout.
	done := make(chan error, 2)

	go func() {
		_, err := io.Copy(conn, os.Stdin)
		done <- err
	}()

	go func() {
		_, err := io.Copy(os.Stdout, conn)
		done <- err
	}()

	// Wait for either direction to finish.
	return <-done
}
