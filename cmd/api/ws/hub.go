package ws

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

// Event represents a message broadcast to subscribers.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var wsClients = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "ws_clients",
	Help: "Number of connected WebSocket clients",
})

func init() { prometheus.MustRegister(wsClients) }

// PublishEvent sends an event to the Redis "events" channel.
func PublishEvent(ctx context.Context, rdb *redis.Client, ev Event) {
	if rdb == nil {
		return
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	_ = rdb.Publish(ctx, "events", b).Err()
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	rdb        *redis.Client
	register   chan *Client
	unregister chan *Client
	clients    map[*Client]bool
	broadcast  chan Event
}

// NewHub constructs a Hub. rdb may be nil to disable cross-process broadcasting.
func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rdb:        rdb,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 16),
	}
}

// Run starts the hub loop, optionally subscribing to Redis events.
func (h *Hub) Run(ctx context.Context) {
	var ch <-chan *redis.Message
	if h.rdb != nil {
		sub := h.rdb.Subscribe(ctx, "events")
		ch = sub.Channel()
		go func() {
			<-ctx.Done()
			_ = sub.Close()
		}()
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if ok && msg != nil {
				var ev Event
				if err := json.Unmarshal([]byte(msg.Payload), &ev); err == nil {
					h.Broadcast(ev)
				}
			}
		case c := <-h.register:
			h.clients[c] = true
			wsClients.Inc()
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				wsClients.Dec()
			}
		case ev := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- ev:
				default:
					delete(h.clients, c)
					close(c.send)
					wsClients.Dec()
				}
			}
		}
	}
}

// Broadcast enqueues an event for all clients.
func (h *Hub) Broadcast(ev Event) { h.broadcast <- ev }

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) { h.register <- c }

// Client represents a WebSocket connection.
type Client struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan Event
	isAdmin bool
}

// NewClient constructs a client.
func NewClient(h *Hub, conn *websocket.Conn, isAdmin bool) *Client {
	return &Client{hub: h, conn: conn, send: make(chan Event, 8), isAdmin: isAdmin}
}

// ReadPump reads messages from the WebSocket to detect disconnects.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

// WritePump writes events to the WebSocket connection.
func (c *Client) WritePump(ctx context.Context) {
	defer func() { _ = c.conn.Close() }()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-c.send:
			if !ok {
				return
			}
			if ev.Type == "queue_changed" && !c.isAdmin {
				continue
			}
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		}
	}
}

// Websocket upgrader with permissive CORS (expected to be protected by middleware).
var Upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
