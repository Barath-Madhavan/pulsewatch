package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

// pingPeriod is a var, not a const, purely so tests can shrink it instead
// of waiting out a real 54 seconds to exercise writePump's keep-alive
// ticker. Production behavior is unchanged: this is still computed once
// from pongWait at package init, same value a const would have held.
var pingPeriod = (pongWait * 9) / 10

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// The dashboard is a separate frontend origin (e.g. Vercel), not a
		// same-origin page, so we don't gate on Origin here.
		return true
	},
}

// Client represents one connected dashboard, with its own outbound buffer
// and read/write pump goroutines.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// ServeWS upgrades the HTTP request to a WebSocket connection and registers
// it with hub. If snapshot is non-nil, its messages are queued for this
// client before any live broadcast, so a newly connected dashboard sees
// current state immediately instead of waiting for the next tick per
// monitor.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, snapshot func() [][]byte) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket: upgrade failed: %v", err)
		return
	}

	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 32)}
	hub.register <- client

	if snapshot != nil {
		for _, msg := range snapshot() {
			select {
			case client.send <- msg:
			default:
			}
		}
	}

	go client.writePump()
	go client.readPump()
}

// readPump drains incoming frames (dashboards don't send data, but reading
// is required to process pong control frames and detect disconnects) and
// unregisters the client when the connection closes.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket: read error: %v", err)
			}
			break
		}
	}
}

// writePump serializes all writes to the connection: broadcast messages and
// periodic pings, both funneled through the send channel / ticker here so
// only this goroutine ever calls conn.Write*.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
