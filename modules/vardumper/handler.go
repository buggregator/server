package vardumper

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

// tcpHandler handles VarDumper TCP connections on port 9912.
// Symfony VarDumper sends one base64-encoded serialized payload per connection,
// terminated by a newline.
type tcpHandler struct {
	php          *PHPProcess
	eventService EventStorer
}

func (h *tcpHandler) HandleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		h.processPayload(line)
	}

	if err := scanner.Err(); err != nil {
		slog.Debug("vardumper: connection error", "err", err)
	}
}

func (h *tcpHandler) processPayload(base64Payload string) {
	result, err := h.php.Parse(base64Payload)
	if err != nil {
		slog.Error("vardumper: parse error", "err", err)
		return
	}

	// Build event payload matching the PHP Buggregator format.
	payload := map[string]any{
		"payload": map[string]any{
			"type":  result.Type,
			"value": result.Value,
			"label": result.Label,
		},
		"context": json.RawMessage(result.Context),
	}

	if result.Language != nil {
		payload["payload"].(map[string]any)["language"] = *result.Language
	}

	b, _ := json.Marshal(payload)

	project := ""
	if result.Project != nil {
		project = *result.Project
	}

	inc := &event.Incoming{
		Type:    "var-dump",
		Payload: json.RawMessage(b),
		Project: project,
	}

	if err := h.eventService.HandleIncoming(context.Background(), inc); err != nil {
		slog.Error("vardumper: failed to store event", "err", err)
	}
}
