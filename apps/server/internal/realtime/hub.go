package realtime

import (
	"context"
	"sync"

	"github.com/coder/websocket"
)

type Hub interface {
	Register(ctx context.Context, gameSessionID string, client *Client) error
	Unregister(gameSessionID string, client *Client)
	BroadcastToGame(ctx context.Context, gameSessionID string, event Event)
	Close(ctx context.Context) error
}

type MemoryHub struct {
	mu     sync.RWMutex
	rooms  map[string]*room
	closed bool
}

func NewMemoryHub() *MemoryHub {
	return &MemoryHub{rooms: map[string]*room{}}
}

func (h *MemoryHub) Register(_ context.Context, gameSessionID string, client *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return context.Canceled
	}
	r := h.rooms[gameSessionID]
	if r == nil {
		r = newRoom()
		h.rooms[gameSessionID] = r
	}
	r.clients[client] = struct{}{}
	return nil
}

func (h *MemoryHub) Unregister(gameSessionID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r := h.rooms[gameSessionID]
	if r == nil {
		return
	}
	delete(r.clients, client)
	closeSend(client)
	if len(r.clients) == 0 {
		delete(h.rooms, gameSessionID)
	}
}

func (h *MemoryHub) BroadcastToGame(_ context.Context, gameSessionID string, event Event) {
	h.mu.RLock()
	r := h.rooms[gameSessionID]
	var clients []*Client
	if r != nil {
		for c := range r.clients {
			clients = append(clients, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range clients {
		select {
		case c.Send <- event:
		default:
			h.Unregister(gameSessionID, c)
		}
	}
}

func (h *MemoryHub) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for id, r := range h.rooms {
		for c := range r.clients {
			closeSend(c)
			if c.Conn != nil {
				_ = c.Conn.Close(websocket.StatusGoingAway, "server shutting down")
			}
		}
		delete(h.rooms, id)
	}
	return ctx.Err()
}

func closeSend(c *Client) {
	defer func() { _ = recover() }()
	close(c.Send)
}
