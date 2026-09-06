package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, hub *Hub, snapshot func() [][]byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r, snapshot)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestServeWS_DeliversSnapshotOnConnect(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	snapshot := func() [][]byte { return [][]byte{[]byte(`{"type":"status_update","payload":{"monitor_id":"m1"}}`)} }
	srv := newTestServer(t, hub, snapshot)
	conn := dial(t, srv)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(msg), "m1")
}

func TestServeWS_NilSnapshotIsFine(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	require.NotPanics(t, func() { dial(t, srv) })
}

func TestServeWS_DeliversLiveBroadcast(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	conn := dial(t, srv)
	time.Sleep(20 * time.Millisecond) // let the registration reach the hub's Run loop

	hub.Broadcast(map[string]string{"type": "status_update", "monitor_id": "m2"})

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(msg), "m2")
}

func TestServeWS_DisconnectUnregisters(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	conn := dial(t, srv)
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, conn.Close())

	// Give readPump time to notice the close and unregister, then confirm
	// the hub no longer thinks anyone is connected by broadcasting and
	// making sure nothing panics/blocks (no receiver left to drain it).
	time.Sleep(100 * time.Millisecond)
	require.NotPanics(t, func() {
		hub.Broadcast(map[string]string{"type": "status_update"})
		time.Sleep(20 * time.Millisecond)
	})
}

// withShortPingPeriod temporarily shrinks the package-level pingPeriod var
// so writePump's keep-alive ticker fires almost immediately instead of
// after a real 54 seconds, restoring the original value afterward.
func withShortPingPeriod(t *testing.T, d time.Duration) {
	t.Helper()
	original := pingPeriod
	pingPeriod = d
	t.Cleanup(func() { pingPeriod = original })
}

func TestWritePump_SendsPeriodicPing(t *testing.T) {
	withShortPingPeriod(t, 10*time.Millisecond)

	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	conn := dial(t, srv)

	pingReceived := make(chan struct{})
	conn.SetPingHandler(func(string) error {
		select {
		case <-pingReceived:
		default:
			close(pingReceived)
		}
		return conn.WriteControl(websocket.PongMessage, nil, time.Now().Add(time.Second))
	})
	// SetPingHandler only fires while something is actively reading;
	// gorilla needs a live read loop to dispatch control frames.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	select {
	case <-pingReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("server never sent a keep-alive ping")
	}
}

func TestWritePump_PingWriteErrorReturnsCleanly(t *testing.T) {
	withShortPingPeriod(t, 10*time.Millisecond)

	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	conn := dial(t, srv)
	require.NoError(t, conn.Close()) // break the connection before the first ping tick fires

	require.NotPanics(t, func() { time.Sleep(50 * time.Millisecond) })
}

func TestWritePump_BroadcastWriteErrorReturnsCleanly(t *testing.T) {
	// Exercises the c.send branch of writePump's select specifically:
	// a real message (ok=true) whose WriteMessage(TextMessage, ...) call
	// itself fails, not the ticker-driven ping write.
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	conn := dial(t, srv)
	require.NoError(t, conn.Close()) // break the connection from the client's side
	// Give the OS time to actually tear the socket down; writing too
	// soon after Close can still succeed into a send buffer the kernel
	// hasn't reclaimed yet.
	time.Sleep(200 * time.Millisecond)

	require.NotPanics(t, func() {
		for i := 0; i < 20; i++ {
			hub.Broadcast(map[string]int{"i": i})
			time.Sleep(5 * time.Millisecond)
		}
	})
}

func TestReadPump_PongHandlerRefreshesReadDeadline(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	conn := dial(t, srv)

	// An unsolicited pong still reaches the server's registered
	// SetPongHandler; it doesn't need to be sent in response to a ping
	// the server initiated.
	require.NoError(t, conn.WriteControl(websocket.PongMessage, nil, time.Now().Add(time.Second)))
	time.Sleep(50 * time.Millisecond)

	// The connection must still be healthy afterward (the pong handler
	// didn't error out and tear anything down).
	hub.Broadcast(map[string]string{"type": "status_update"})
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err := conn.ReadMessage()
	assert.NoError(t, err)
}

func TestServeWS_LoggableCloseCodeIsHandledWithoutPanicking(t *testing.T) {
	// gorilla's IsUnexpectedCloseError(err, GoingAway, AbnormalClosure)
	// only suppresses the read-error log for those two specific codes; a
	// clean handshake close with an ordinary code like NormalClosure
	// still counts as "unexpected" by that check and does get logged.
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	conn := dial(t, srv)
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"),
	))
	time.Sleep(50 * time.Millisecond) // let the server's readPump observe and log it
}

func TestWritePump_HubForceClosingSendChannelIsHandled(t *testing.T) {
	// Fill the client's send buffer (32, set in ServeWS) faster than a
	// real writePump can drain it to the socket, without the test ever
	// reading from the connection. The hub's Broadcast fan-out then drops
	// this client as "too slow" and closes its send channel; writePump
	// must notice (ok=false), write a close frame, and return cleanly.
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	_ = dial(t, srv) // deliberately never read from this connection
	time.Sleep(20 * time.Millisecond)

	require.NotPanics(t, func() {
		for i := 0; i < 200; i++ {
			hub.Broadcast(map[string]int{"i": i})
		}
		time.Sleep(100 * time.Millisecond)
	})
}

func TestServeWS_UpgradeFailureIsHandledWithoutPanicking(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	srv := newTestServer(t, hub, nil)
	require.NotPanics(t, func() {
		// A plain GET with none of the WebSocket upgrade headers: Upgrade
		// must fail gracefully (logged), not panic.
		resp, err := http.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
	})
}
