package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"nhooyr.io/websocket"
)

// Hub manages WebSocket connections and broadcasts messages
// using the Centrifugo v5 JSON protocol.
type Hub struct {
	mu              sync.RWMutex
	clients         map[*client]struct{}
	rpcHandler      RPCHandler
	projectProvider ProjectProvider
}

// RPCHandler processes Centrifugo RPC calls (e.g., "delete:api/event/{id}").
type RPCHandler interface {
	HandleRPC(method, uri string, data json.RawMessage) (json.RawMessage, error)
}

// ProjectProvider returns the list of project keys for server-side subscriptions.
type ProjectProvider func() []string

type client struct {
	conn *websocket.Conn
	ctx  context.Context
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*client]struct{}),
	}
}

// SetRPCHandler sets the handler for Centrifugo RPC calls.
func (h *Hub) SetRPCHandler(handler RPCHandler) {
	h.rpcHandler = handler
}

// SetProjectProvider sets the function that returns project keys for auto-subscription.
func (h *Hub) SetProjectProvider(p ProjectProvider) {
	h.projectProvider = p
}

// Run keeps the hub alive until context is cancelled.
func (h *Hub) Run(ctx context.Context) {
	<-ctx.Done()
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		c.conn.Close(websocket.StatusGoingAway, "server shutting down")
	}
}

// HandleUpgrade is the HTTP handler for /connection/websocket.
func (h *Hub) HandleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("websocket accept error", "err", err)
		return
	}

	c := &client{conn: conn, ctx: r.Context()}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
	}()

	// Read loop — handle Centrifugo protocol commands.
	for {
		_, data, err := conn.Read(c.ctx)
		if err != nil {
			break
		}
		h.handleClientMessage(c, data)
	}
}

// handleClientMessage processes incoming Centrifugo JSON protocol messages.
// Format: newline-delimited JSON commands.
func (h *Hub) handleClientMessage(c *client, data []byte) {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var cmd centrifugoCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			slog.Debug("ws: invalid command", "err", err)
			continue
		}

		reply := h.processCommand(cmd)
		replyBytes, _ := json.Marshal(reply)
		c.conn.Write(c.ctx, websocket.MessageText, replyBytes)
	}
}

// processCommand handles a single Centrifugo protocol command.
func (h *Hub) processCommand(cmd centrifugoCommand) centrifugoReply {
	if cmd.Connect != nil {
		return h.handleConnect(cmd.ID)
	}
	if cmd.Subscribe != nil {
		return h.handleSubscribe(cmd.ID, cmd.Subscribe)
	}
	if cmd.RPC != nil {
		return h.handleRPC(cmd.ID, cmd.RPC)
	}
	// Ping (empty command with just id) — respond with empty reply.
	return centrifugoReply{ID: cmd.ID}
}

func (h *Hub) handleConnect(id uint32) centrifugoReply {
	// Build server-side subscriptions for all project channels.
	subs := make(map[string]serverSideSubV)
	if h.projectProvider != nil {
		for _, key := range h.projectProvider() {
			subs["events.project."+key] = serverSideSubV{}
		}
	}

	return centrifugoReply{
		ID: id,
		Connect: &connectResult{
			Client:  "go-buggregator",
			Version: "0.0.1",
			Ping:    25,
			Pong:    true,
			Subs:    subs,
		},
	}
}

func (h *Hub) handleSubscribe(id uint32, sub *subscribeRequest) centrifugoReply {
	return centrifugoReply{
		ID:        id,
		Subscribe: &subscribeResult{},
	}
}

func (h *Hub) handleRPC(id uint32, rpc *rpcRequest) centrifugoReply {
	if h.rpcHandler == nil {
		return centrifugoReply{
			ID:    id,
			Error: &centrifugoError{Code: 100, Message: "RPC not supported"},
		}
	}

	// Parse method format: "delete:api/event/{id}"
	parts := strings.SplitN(rpc.Method, ":", 2)
	if len(parts) != 2 {
		return centrifugoReply{
			ID:    id,
			Error: &centrifugoError{Code: 102, Message: "invalid RPC method format"},
		}
	}

	httpMethod := strings.ToUpper(parts[0])
	uri := parts[1]

	result, err := h.rpcHandler.HandleRPC(httpMethod, uri, rpc.Data)
	if err != nil {
		return centrifugoReply{
			ID:    id,
			Error: &centrifugoError{Code: 100, Message: err.Error()},
		}
	}

	return centrifugoReply{
		ID:  id,
		RPC: &rpcResult{Data: result},
	}
}

// Broadcast sends a publication push to all connected clients.
func (h *Hub) Broadcast(channel, eventName string, payload any) {
	pubData := map[string]any{
		"event": eventName,
		"data":  payload,
	}
	pubDataBytes, err := json.Marshal(pubData)
	if err != nil {
		slog.Error("broadcast marshal error", "err", err)
		return
	}

	push := centrifugoPush{
		Push: &pushData{
			Channel: channel,
			Pub: &publication{
				Data: json.RawMessage(pubDataBytes),
			},
		},
	}

	msg, _ := json.Marshal(push)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		if err := c.conn.Write(c.ctx, websocket.MessageText, msg); err != nil {
			slog.Debug("websocket write error", "err", err)
		}
	}
}

// --- Centrifugo v5 JSON protocol types ---

// Client → Server command.
type centrifugoCommand struct {
	ID        uint32            `json:"id"`
	Connect   *connectRequest   `json:"connect,omitempty"`
	Subscribe *subscribeRequest `json:"subscribe,omitempty"`
	RPC       *rpcRequest       `json:"rpc,omitempty"`
}

type connectRequest struct {
	Token   string         `json:"token,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Name    string         `json:"name,omitempty"`
	Version string         `json:"version,omitempty"`
}

type subscribeRequest struct {
	Channel string `json:"channel"`
}

type rpcRequest struct {
	Method string          `json:"method"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// Server → Client reply (response to a command with id).
type centrifugoReply struct {
	ID        uint32           `json:"id,omitempty"`
	Connect   *connectResult   `json:"connect,omitempty"`
	Subscribe *subscribeResult `json:"subscribe,omitempty"`
	RPC       *rpcResult       `json:"rpc,omitempty"`
	Error     *centrifugoError `json:"error,omitempty"`
}

type connectResult struct {
	Client  string                    `json:"client"`
	Version string                    `json:"version"`
	Ping    int                       `json:"ping,omitempty"`
	Pong    bool                      `json:"pong,omitempty"`
	Subs    map[string]serverSideSubV `json:"subs,omitempty"`
}

type serverSideSubV struct {
	Recoverable bool `json:"recoverable,omitempty"`
}

type subscribeResult struct{}

type rpcResult struct {
	Data json.RawMessage `json:"data"`
}

type centrifugoError struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

// Server → Client push (no id, unsolicited).
type centrifugoPush struct {
	Push *pushData `json:"push"`
}

type pushData struct {
	Channel string       `json:"channel"`
	Pub     *publication `json:"pub,omitempty"`
}

type publication struct {
	Data json.RawMessage `json:"data"`
}
