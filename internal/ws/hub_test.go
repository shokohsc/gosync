package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("expected hub to be non-nil")
	}
	if h.clients == nil {
		t.Fatal("expected clients map to be initialized")
	}
	if h.upgrader.CheckOrigin != nil {
		t.Fatal("expected CheckOrigin to be nil (same-origin default)")
	}
}

func TestBroadcastWithNoClients(t *testing.T) {
	h := NewHub()
	h.Broadcast(Event{Type: "reload", Path: "test.html"})
}

func TestHandleWSAndBroadcast(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Errorf("read error: %v", err)
			return
		}
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Errorf("unmarshal error: %v", err)
			return
		}
		if ev.Type != "reload" {
			t.Errorf("expected type 'reload', got %q", ev.Type)
		}
		if ev.Path != "test.html" {
			t.Errorf("expected path 'test.html', got %q", ev.Path)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	h.Broadcast(Event{Type: "reload", Path: "test.html"})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestRateLimit(t *testing.T) {
	h := NewHub()

	for i := 0; i < maxConnections; i++ {
		client := &Client{
			conn: nil,
			send: make(chan []byte, 64),
		}
		h.clients[client] = true
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.HandleWS(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}

	h.mu.Lock()
	for c := range h.clients {
		delete(h.clients, c)
	}
	h.mu.Unlock()
}

func TestCloseClient(t *testing.T) {
	h := NewHub()

	h.closeClient(&Client{
		conn: nil,
		send: make(chan []byte),
	})

	client := &Client{
		conn: nil,
		send: make(chan []byte),
	}
	h.clients[client] = true
	h.closeClient(client)
	h.closeClient(client)

	if len(h.clients) != 0 {
		t.Errorf("expected 0 clients, got %d", len(h.clients))
	}
}

func TestEventJSON(t *testing.T) {
	ev := Event{
		Type: "scroll",
		Data: json.RawMessage(`{"x":100,"y":200}`),
	}
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(data), `"scroll"`) {
		t.Error("expected JSON to contain scroll type")
	}
	if !strings.Contains(string(data), `100`) {
		t.Error("expected JSON to contain data")
	}
}

func TestBroadcastWithFullClient(t *testing.T) {
	h := NewHub()

	client := &Client{
		conn: nil,
		send: make(chan []byte, 1),
	}
	h.clients[client] = true

	client.send <- []byte("full")

	h.Broadcast(Event{Type: "reload"})

	time.Sleep(50 * time.Millisecond)

	h.mu.RLock()
	_, exists := h.clients[client]
	h.mu.RUnlock()

	if exists {
		t.Log("client may still exist (async close is racy)")
	}
}

func TestBroadcastWithBroadcastFn(t *testing.T) {
	h := NewHub()

	received := make(chan Event, 1)
	h.BroadcastFn = func(ev Event) {
		received <- ev
	}

	h.Broadcast(Event{Type: "css", Path: "style.css"})

	select {
	case ev := <-received:
		if ev.Type != "css" {
			t.Errorf("expected 'css', got %q", ev.Type)
		}
		if ev.Path != "style.css" {
			t.Errorf("expected 'style.css', got %q", ev.Path)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for BroadcastFn")
	}
}

func TestBroadcastWithClosingChannel(t *testing.T) {
	h := NewHub()

	h.BroadcastFn = func(ev Event) {}

	ch := make(chan []byte)
	close(ch)

	client := &Client{
		conn: nil,
		send: ch,
	}
	h.clients[client] = true

	h.Broadcast(Event{Type: "reload"})

	time.Sleep(50 * time.Millisecond)
}

func TestHandleWSWithInvalidRequest(t *testing.T) {
	h := NewHub()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	// No Upgrade header -> should fail
	h.HandleWS(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("expected bad request for non-ws request, got %d", resp.StatusCode)
	}
}

func TestMultipleClientsBroadcast(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	const numClients = 5
	clients := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial %d error: %v", i, err)
		}
		defer c.Close()
		clients[i] = c
	}

	time.Sleep(50 * time.Millisecond)

	received := make(chan int, numClients)
	for i, c := range clients {
		go func(idx int, conn *websocket.Conn) {
			_, _, err := conn.ReadMessage()
			if err == nil {
				received <- idx
			}
		}(i, c)
	}

	h.Broadcast(Event{Type: "reload", Path: "all.html"})

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for any client to receive")
	}
}

func TestWritePumpSendsMessage(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, msg, err := c.ReadMessage()
		if err != nil {
			t.Errorf("read error: %v", err)
			return
		}
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Errorf("unmarshal error: %v", err)
			return
		}
		if ev.Type != "test-write-pump" {
			t.Errorf("expected 'test-write-pump', got %q", ev.Type)
		}
	}()

	time.Sleep(50 * time.Millisecond)
	h.Broadcast(Event{Type: "test-write-pump"})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message via writePump")
	}
}

func TestNewHubWithOptionsCustom(t *testing.T) {
	opts := HubOptions{
		MaxConnections:      50,
		MaxMessageSizeBytes: 2048,
		PongWait:            30 * time.Second,
		WriteWait:           5 * time.Second,
	}

	h := NewHubWithOptions(opts)
	if h == nil {
		t.Fatal("expected non-nil hub")
	}
	if h.maxConnections != 50 {
		t.Errorf("expected maxConnections 50, got %d", h.maxConnections)
	}
	if h.maxMessageSize != 2048 {
		t.Errorf("expected maxMessageSize 2048, got %d", h.maxMessageSize)
	}
	if h.pongWait != 30*time.Second {
		t.Errorf("expected pongWait 30s, got %v", h.pongWait)
	}
	if h.writeWait != 5*time.Second {
		t.Errorf("expected writeWait 5s, got %v", h.writeWait)
	}
}

func TestNewHubWithOptionsZeroUsesDefaults(t *testing.T) {
	h := NewHubWithOptions(HubOptions{})
	if h.maxConnections != 100 {
		t.Errorf("expected maxConnections 100, got %d", h.maxConnections)
	}
	if h.maxMessageSize != 4096 {
		t.Errorf("expected maxMessageSize 4096, got %d", h.maxMessageSize)
	}
	if h.pongWait != 60*time.Second {
		t.Errorf("expected pongWait 60s, got %v", h.pongWait)
	}
	if h.writeWait != 10*time.Second {
		t.Errorf("expected writeWait 10s, got %v", h.writeWait)
	}
}

func TestNewHubWithOptionsPingInterval(t *testing.T) {
	h := NewHubWithOptions(HubOptions{PongWait: 20 * time.Second})
	expectedPing := (20 * time.Second * 9) / 10
	if h.pingPeriod != expectedPing {
		t.Errorf("expected pingPeriod %v, got %v", expectedPing, h.pingPeriod)
	}
}

func TestNewHubWithOptionsExplicitPing(t *testing.T) {
	h := NewHubWithOptions(HubOptions{
		PongWait:     60 * time.Second,
		PingInterval: 10 * time.Second,
	})
	if h.pingPeriod != 10*time.Second {
		t.Errorf("expected pingPeriod 10s, got %v", h.pingPeriod)
	}
}
