package mcp

import (
	"context"
	"log/slog"
	"net/http"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// StartHTTP runs the MCP server over HTTP (Streamable HTTP / SSE).
// This enables remote MCP access for AI assistants over the network.
// If authToken is non-empty, requests must include "Authorization: Bearer <token>".
func StartHTTP(ctx context.Context, addr, authToken string, server *sdkmcp.Server) error {
	handler := sdkmcp.NewStreamableHTTPHandler(
		func(r *http.Request) *sdkmcp.Server {
			return server
		},
		nil,
	)

	var h http.Handler = handler
	if authToken != "" {
		h = authMiddleware(authToken, handler)
	}

	mux := http.NewServeMux()
	mux.Handle("/", h)

	srv := &http.Server{Addr: addr, Handler: mux}

	slog.Info("MCP HTTP server started", "addr", addr, "auth", authToken != "")

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
