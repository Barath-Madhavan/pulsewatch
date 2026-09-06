package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient builds a *Client sufficient for whitebox Hub tests: no real
// network connection, just the send channel Hub actually touches.
func newTestClient(bufSize int) *Client {
	return &Client{send: make(chan []byte, bufSize)}
}

func TestHub_RegisterAndBroadcast(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(4)
	h.register <- c
	time.Sleep(10 * time.Millisecond) // let Run's select loop process the registration

	h.Broadcast(map[string]string{"type": "status_update"})

	select {
	case msg := <-c.send:
		assert.Contains(t, string(msg), "status_update")
	case <-time.After(time.Second):
		t.Fatal("registered client never received the broadcast")
	}
}

func TestHub_BroadcastReachesMultipleClients(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c1, c2 := newTestClient(4), newTestClient(4)
	h.register <- c1
	h.register <- c2
	time.Sleep(10 * time.Millisecond)

	h.Broadcast(map[string]string{"type": "ping"})

	for _, c := range []*Client{c1, c2} {
		select {
		case <-c.send:
		case <-time.After(time.Second):
			t.Fatal("a registered client didn't receive the broadcast")
		}
	}
}

func TestHub_Unregister(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	c := newTestClient(4)
	h.register <- c
	time.Sleep(10 * time.Millisecond)
	h.unregister <- c
	time.Sleep(10 * time.Millisecond)

	_, stillOpen := <-c.send
	assert.False(t, stillOpen, "unregistering must close the client's send channel")
}

func TestHub_UnregisterUnknownClientIsANoOp(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	// Never registered; the unregister branch's "if _, ok := ...; ok"
	// guard must skip closing anything that isn't tracked.
	c := newTestClient(1)
	require.NotPanics(t, func() {
		h.unregister <- c
		time.Sleep(10 * time.Millisecond)
	})
}

func TestHub_SlowClientIsDroppedNotBlocked(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	slow := newTestClient(0) // unbuffered and nobody ever reads it: always "full"
	fast := newTestClient(4)
	h.register <- slow
	h.register <- fast
	time.Sleep(10 * time.Millisecond)

	// If the hub blocked on the slow client instead of dropping it, this
	// broadcast (and the whole Run loop) would stall and fast would never
	// receive anything either.
	h.Broadcast(map[string]string{"type": "ping"})

	select {
	case <-fast.send:
	case <-time.After(time.Second):
		t.Fatal("a slow client blocked the broadcast to everyone else")
	}

	_, stillOpen := <-slow.send
	assert.False(t, stillOpen, "the slow client must be dropped (its channel closed)")
}

func TestHub_BroadcastMarshalErrorIsHandledGracefully(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go h.Run(ctx)

	require.NotPanics(t, func() {
		h.Broadcast(func() {}) // funcs can't be marshaled to JSON
		time.Sleep(10 * time.Millisecond)
	})
}

func TestHub_BroadcastChannelFullDropsWithoutBlocking(t *testing.T) {
	h := NewHub()
	// Run is deliberately never started: nothing drains h.broadcast, so
	// its buffer (256) fills up and the next Broadcast must hit the
	// default: drop branch instead of blocking forever.
	for i := 0; i < 256; i++ {
		h.Broadcast(map[string]int{"i": i})
	}

	done := make(chan struct{})
	go func() {
		h.Broadcast(map[string]string{"type": "one-too-many"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Broadcast blocked instead of dropping when the channel was full")
	}
}

func TestHub_ContextCancelClosesAllClients(t *testing.T) {
	h := NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)

	c := newTestClient(4)
	h.register <- c
	time.Sleep(10 * time.Millisecond)

	cancel()
	time.Sleep(10 * time.Millisecond)

	_, stillOpen := <-c.send
	assert.False(t, stillOpen, "cancelling the hub's context must close every client's send channel")
}
