package websocket

import (
	"context"
	"encoding/json"
	"log"
)

// Hub owns the set of connected clients and fans broadcast messages out to
// all of them. Only Run's goroutine ever touches the clients map, so no
// locking is needed around it.
type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for c := range h.clients {
				close(c.send)
				delete(h.clients, c)
			}
			return

		case c := <-h.register:
			h.clients[c] = struct{}{}

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					// Client isn't draining its send buffer fast enough; drop it
					// rather than block the whole hub on one slow connection.
					close(c.send)
					delete(h.clients, c)
				}
			}
		}
	}
}

// Broadcast marshals v to JSON and fans it out to every connected client.
func (h *Hub) Broadcast(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("websocket: failed to marshal broadcast message: %v", err)
		return
	}
	select {
	case h.broadcast <- data:
	default:
		log.Printf("websocket: broadcast channel full, dropping message")
	}
}
