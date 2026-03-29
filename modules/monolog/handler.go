package monolog

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net"

	"github.com/buggregator/go-buggregator/internal/event"
)

type tcpHandler struct {
	eventService EventStorer
}

func (h *tcpHandler) HandleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal(line, &payload); err != nil {
			slog.Debug("monolog: invalid JSON", "err", err)
			continue
		}

		// Extract project from context.
		project := ""
		if ctx, ok := payload["context"].(map[string]any); ok {
			if p, ok := ctx["project"].(string); ok {
				project = p
			}
		}

		b, _ := json.Marshal(payload)

		inc := &event.Incoming{
			Type:    "monolog",
			Payload: json.RawMessage(b),
			Project: project,
		}

		if err := h.eventService.HandleIncoming(context.Background(), inc); err != nil {
			slog.Error("monolog: failed to store event", "err", err)
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Debug("monolog: connection error", "err", err)
	}
}
