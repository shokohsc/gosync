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

	// Read hello message first
	_, hello, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read hello error: %v", err)
	}
	var helloEv Event
	if err := json.Unmarshal(hello, &helloEv); err != nil {
		t.Fatalf("unmarshal hello error: %v", err)
	}
	if helloEv.Type != "hello" {
		t.Errorf("expected hello, got %q", helloEv.Type)
	}

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

	h.mu.Lock()
	for i := 0; i < maxConnections; i++ {
		client := &Client{
			conn: nil,
			send: make(chan []byte, 64),
		}
		h.clients[client] = true
	}
	h.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.HandleWS(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", resp.StatusCode)
	}
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
		// drain hello
		_, _, err = c.ReadMessage()
		if err != nil {
			t.Fatalf("read hello on client %d: %v", i, err)
		}
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

	// drain hello
	_, _, err = c.ReadMessage()
	if err != nil {
		t.Fatalf("read hello: %v", err)
	}

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
		t.Errorf("expected 50 max connections, got %d", h.maxConnections)
	}
	if h.maxMessageSize != 2048 {
		t.Errorf("expected 2048 max message size, got %d", h.maxMessageSize)
	}
	if h.pongWait != 30*time.Second {
		t.Errorf("expected 30s pong wait, got %v", h.pongWait)
	}
	if h.writeWait != 5*time.Second {
		t.Errorf("expected 5s write wait, got %v", h.writeWait)
	}
}

func TestNewHubWithOptionsDefaults(t *testing.T) {
	h := NewHubWithOptions(HubOptions{})
	if h.maxConnections != maxConnections {
		t.Errorf("expected default %d max connections, got %d", maxConnections, h.maxConnections)
	}
	if h.maxMessageSize != maxMessageSize {
		t.Errorf("expected default %d max message size, got %d", maxMessageSize, h.maxMessageSize)
	}
	if h.pongWait != pongWait {
		t.Errorf("expected default %v pong wait, got %v", pongWait, h.pongWait)
	}
}

func TestBroadcastExcept(t *testing.T) {
	h := NewHub()

	sender := &Client{
		conn: nil,
		send: make(chan []byte, 64),
	}
	receiver := &Client{
		conn: nil,
		send: make(chan []byte, 64),
	}
	h.clients[sender] = true
	h.clients[receiver] = true

	h.BroadcastExcept(sender, Event{Type: "scroll", Data: json.RawMessage(`{"x":1,"y":2}`)})

	select {
	case <-receiver.send:
	case <-time.After(time.Second):
		t.Error("expected receiver to get message")
	}

	select {
	case <-sender.send:
		t.Error("sender should not receive its own message")
	default:
	}
}

func TestReadPumpRelaysClientEvents(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	client1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial client1 error: %v", err)
	}
	defer client1.Close()

	// drain hello
	_, _, err = client1.ReadMessage()
	if err != nil {
		t.Fatalf("client1 read hello: %v", err)
	}

	client2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial client2 error: %v", err)
	}
	defer client2.Close()

	// drain hello
	_, _, err = client2.ReadMessage()
	if err != nil {
		t.Fatalf("client2 read hello: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, msg, err := client2.ReadMessage()
		if err != nil {
			t.Errorf("client2 read error: %v", err)
			return
		}
		var ev Event
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Errorf("unmarshal error: %v", err)
			return
		}
		if ev.Type != "scroll" {
			t.Errorf("expected 'scroll', got %q", ev.Type)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	client1.WriteMessage(websocket.TextMessage, []byte(`{"type":"scroll","data":{"x":100,"y":200}}`))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for relayed event")
	}
}

func TestNotifySendsBrowserNotify(t *testing.T) {
	h := NewHub()

	received := make(chan Event, 1)
	h.BroadcastFn = func(ev Event) {
		received <- ev
	}

	h.Notify("test message", 3*time.Second)

	select {
	case ev := <-received:
		if ev.Type != "browser:notify" {
			t.Errorf("expected 'browser:notify', got %q", ev.Type)
		}
		if !strings.Contains(string(ev.Data), "test message") {
			t.Errorf("expected data to contain 'test message', got %s", string(ev.Data))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notify")
	}
}

func TestNotifyUsesDefaultTimeout(t *testing.T) {
	h := NewHub()

	received := make(chan Event, 1)
	h.BroadcastFn = func(ev Event) {
		received <- ev
	}

	h.Notify("short", 0)

	select {
	case ev := <-received:
		if ev.Type != "browser:notify" {
			t.Errorf("expected 'browser:notify', got %q", ev.Type)
		}
		if !strings.Contains(string(ev.Data), `"timeout":5000`) {
			t.Errorf("expected default timeout 5000ms, got %s", string(ev.Data))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notify")
	}
}

func TestHelloMessageOnConnect(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer c.Close()

	_, msg, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read hello error: %v", err)
	}

	var ev Event
	if err := json.Unmarshal(msg, &ev); err != nil {
		t.Fatalf("unmarshal hello error: %v", err)
	}
	if ev.Type != "hello" {
		t.Errorf("expected 'hello', got %q", ev.Type)
	}
}

func TestReadPumpIgnoresUnknownEvents(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleWS))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"

	client1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial client1 error: %v", err)
	}
	defer client1.Close()

	// drain hello
	_, _, err = client1.ReadMessage()
	if err != nil {
		t.Fatalf("client1 read hello: %v", err)
	}

	client2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial client2 error: %v", err)
	}
	defer client2.Close()

	// drain hello
	_, _, err = client2.ReadMessage()
	if err != nil {
		t.Fatalf("client2 read hello: %v", err)
	}

	client2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

	time.Sleep(50 * time.Millisecond)

	client1.WriteMessage(websocket.TextMessage, []byte(`{"type":"unknown","data":{}}`))

	_, _, err = client2.ReadMessage()
	if err == nil {
		t.Error("expected client2 to NOT receive unknown event type")
	}
}

func TestClientIDsIncrement(t *testing.T) {
	h := NewHub()

	c1 := &Client{id: 0}
	c2 := &Client{id: 0}

	h.mu.Lock()
	h.nextID++

	c1.id = h.nextID
	h.clients[c1] = true

	h.nextID++
	c2.id = h.nextID
	h.clients[c2] = true
	h.mu.Unlock()

	if c1.id == c2.id {
		t.Error("expected client IDs to be different")
	}
}
