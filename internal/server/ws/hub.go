package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/buggregator/go-buggregator/internal/metrics"
	"nhooyr.io/websocket"
)

// Hub manages WebSocket connections and broadcasts messages
// using the Centrifugo v5 JSON protocol.
type Hub struct {
	mu              sync.RWMutex
	clients         map[*client]struct{}
	rpcHandler      RPCHandler
	projectProvider ProjectProvider
	metrics         *metrics.Collector
	serverChannels  []string // extra channels every client is subscribed to

	// validToken checks connect.token when API auth is on; nil = auth off.
	validToken TokenValidator
	// connectTimeout closes a socket that has not passed connect in time
	// (auth on only).
	connectTimeout time.Duration
}

// DefaultConnectTimeout is how long, with auth on, a socket may stay open
// without a connect that passed.
const DefaultConnectTimeout = 10 * time.Second

// TokenValidator reports whether an API auth token is valid and when it
// expires (zero time: never).
type TokenValidator func(token string) (expires time.Time, ok bool)

// RPCHandler processes Centrifugo RPC calls (e.g., "delete:api/event/{id}").
type RPCHandler interface {
	HandleRPC(method, uri string, data json.RawMessage) (json.RawMessage, error)
}

// ProjectProvider returns the list of project keys for server-side subscriptions.
type ProjectProvider func() []string

type client struct {
	conn *websocket.Conn
	ctx  context.Context

	// mu guards the server-side subscriptions below and orders the connect
	// reply before any push that depends on it.
	mu        sync.Mutex
	connected bool            // the connect reply was sent
	subs      map[string]bool // server-side channels the client knows
	expiry    *time.Timer     // closes the socket when its token expires
}

func NewHub() *Hub {
	return &Hub{
		clients:        make(map[*client]struct{}),
		connectTimeout: DefaultConnectTimeout,
	}
}

// SetRPCHandler sets the handler for Centrifugo RPC calls.
func (h *Hub) SetRPCHandler(handler RPCHandler) {
	h.rpcHandler = handler
}

// SetTokenValidator turns on auth for the socket: a connect needs a token v
// accepts, only connected clients get commands answered and pushes, a socket
// without a passed connect is closed after the connect timeout, and a socket
// is closed when its token expires. Call it before serving.
func (h *Hub) SetTokenValidator(v TokenValidator) {
	h.validToken = v
}

// SetProjectProvider sets the function that returns project keys for auto-subscription.
func (h *Hub) SetProjectProvider(p ProjectProvider) {
	h.projectProvider = p
}

// AddServerChannel subscribes every client to channel on connect
// (server-side subscription), like the project channels.
func (h *Hub) AddServerChannel(channel string) {
	h.serverChannels = append(h.serverChannels, channel)
}

// SetMetrics sets the metrics collector for the hub.
func (h *Hub) SetMetrics(m *metrics.Collector) {
	h.metrics = m
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
	if h.validToken != nil {
		deadline := time.AfterFunc(h.connectTimeout, func() {
			if !c.isConnected() {
				c.conn.Close(websocket.StatusPolicyViolation, "connect timeout")
			}
		})
		defer deadline.Stop()
		defer func() {
			c.mu.Lock()
			if c.expiry != nil {
				c.expiry.Stop()
			}
			c.mu.Unlock()
		}()
	}
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	if h.metrics != nil {
		h.metrics.WSConnectionsActive.Inc()
	}

	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		if h.metrics != nil {
			h.metrics.WSConnectionsActive.Dec()
		}
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

		if cmd.Connect != nil {
			if !h.connectClient(c, cmd.ID, cmd.Connect.Token) {
				return
			}
			continue
		}
		if h.validToken != nil && !c.isConnected() {
			// Auth on: nothing is answered before a connect with a valid token.
			continue
		}
		reply := h.processCommand(cmd)
		replyBytes, _ := json.Marshal(reply)
		c.conn.Write(c.ctx, websocket.MessageText, replyBytes)
	}
}

// connectClient answers a connect command and remembers the server-side
// channels it gave, so Broadcast can subscribe the client to channels that
// appear later (a project created after the page opened). With auth on, a
// connect without a valid token gets error 109 and the socket is closed with
// 1008; it returns false then. centrifuge-js reconnects on that close code
// and, on 109, asks its getToken for a fresh token first. A token that
// expires later closes the socket then with 3500, on which centrifuge-js does
// not reconnect: the page connects again after a new login.
func (h *Hub) connectClient(c *client, id uint32, token string) bool {
	var expires time.Time
	if h.validToken != nil {
		var ok bool
		if token != "" {
			expires, ok = h.validToken(token)
		}
		if !ok {
			return h.refuseConnect(c, id)
		}
	}

	reply := h.handleConnect(id)
	replyBytes, _ := json.Marshal(reply)

	c.mu.Lock()
	defer c.mu.Unlock()
	c.subs = make(map[string]bool, len(reply.Connect.Subs))
	for ch := range reply.Connect.Subs {
		c.subs[ch] = true
	}
	c.conn.Write(c.ctx, websocket.MessageText, replyBytes)
	c.connected = true
	if !expires.IsZero() {
		if c.expiry != nil {
			c.expiry.Stop()
		}
		c.expiry = time.AfterFunc(time.Until(expires), func() {
			c.conn.Close(websocket.StatusCode(3500), "token expired")
		})
	}
	return true
}

// refuseConnect answers a connect with error 109 and closes the socket.
func (h *Hub) refuseConnect(c *client, id uint32) bool {
	reply, _ := json.Marshal(centrifugoReply{
		ID:    id,
		Error: &centrifugoError{Code: 109, Message: "unauthorized"},
	})
	c.mu.Lock()
	c.conn.Write(c.ctx, websocket.MessageText, reply)
	c.mu.Unlock()
	c.conn.Close(websocket.StatusPolicyViolation, "unauthorized")
	return false
}

func (c *client) isConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
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
	for _, ch := range h.serverChannels {
		subs[ch] = serverSideSubV{}
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
	// Server-side subscribe push: centrifuge-js drops publications of a
	// channel it was never subscribed to, and server-side channels are given
	// only at connect.
	subMsg, _ := json.Marshal(centrifugoPush{Push: &pushData{Channel: channel, Subscribe: &subscribePush{}}})

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		c.mu.Lock()
		if h.validToken != nil && !c.connected {
			// Auth on: no data for a socket that has not passed connect.
			c.mu.Unlock()
			continue
		}
		if c.connected && !c.subs[channel] {
			c.subs[channel] = true
			if err := c.conn.Write(c.ctx, websocket.MessageText, subMsg); err != nil {
				slog.Debug("websocket write error", "err", err)
			}
		}
		if err := c.conn.Write(c.ctx, websocket.MessageText, msg); err != nil {
			slog.Debug("websocket write error", "err", err)
		}
		c.mu.Unlock()
	}

	if h.metrics != nil {
		h.metrics.WSMessagesSentTotal.WithLabelValues(channel).Inc()
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
	Channel   string         `json:"channel"`
	Pub       *publication   `json:"pub,omitempty"`
	Subscribe *subscribePush `json:"subscribe,omitempty"`
}

// subscribePush tells the client it is now subscribed to the push's channel
// (a server-side subscription made after connect).
type subscribePush struct {
	Recoverable bool `json:"recoverable,omitempty"`
}

type publication struct {
	Data json.RawMessage `json:"data"`
}
