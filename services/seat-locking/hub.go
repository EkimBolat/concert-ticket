package main

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	// Demo project: allow any origin. Restrict this before shipping.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// hub fans out seat updates for a single event to every WebSocket client
// currently watching that event's seat map.
type hub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func newHub() *hub {
	return &hub{clients: make(map[*websocket.Conn]bool)}
}

func (h *hub) add(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	conn.Close()
}

func (h *hub) broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

// hubManager keeps one hub per event and lazily subscribes it to that
// event's Redis Pub/Sub channel the first time a client connects. This is
// what makes the seat map "real-time": a lock acquired by one user is
// broadcast to every other browser watching the same event within
// milliseconds, no polling involved.
type hubManager struct {
	mu   sync.Mutex
	rdb  *redis.Client
	hubs map[string]*hub
}

func newHubManager(rdb *redis.Client) *hubManager {
	return &hubManager{rdb: rdb, hubs: make(map[string]*hub)}
}

func (m *hubManager) hubFor(ctx context.Context, eventID string) *hub {
	m.mu.Lock()
	defer m.mu.Unlock()

	if h, ok := m.hubs[eventID]; ok {
		return h
	}

	h := newHub()
	m.hubs[eventID] = h
	go m.subscribe(ctx, eventID, h)
	return h
}

func (m *hubManager) subscribe(ctx context.Context, eventID string, h *hub) {
	sub := m.rdb.Subscribe(ctx, channelFor(eventID))
	defer sub.Close()

	for msg := range sub.Channel() {
		h.broadcast([]byte(msg.Payload))
	}
}

func handleWS(manager *hubManager, w http.ResponseWriter, r *http.Request, eventID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	h := manager.hubFor(r.Context(), eventID)
	h.add(conn)
	defer h.remove(conn)

	// This socket is server -> client only. Reading just detects
	// disconnects so we can clean the client out of the hub.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
