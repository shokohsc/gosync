package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	maxConnections = 100
)

type HubOptions struct {
	MaxConnections      int
	MaxMessageSizeBytes int
	PongWait            time.Duration
	PingInterval        time.Duration
	WriteWait           time.Duration
}

type Event struct {
	Type string          `json:"type"`
	Path string          `json:"path,omitempty"`
	URL  string          `json:"url,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type Client struct {
	id   uint64
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu              sync.RWMutex
	clients         map[*Client]bool
	upgrader        websocket.Upgrader
	maxConnections  int
	maxMessageSize  int64
	pongWait        time.Duration
	pingPeriod      time.Duration
	writeWait       time.Duration
	nextID          uint64

	BroadcastFn func(event Event)

	HelloData json.RawMessage
}

func defaultHubOptions() HubOptions {
	return HubOptions{
		MaxConnections:      maxConnections,
		MaxMessageSizeBytes: maxMessageSize,
		PongWait:            pongWait,
		WriteWait:           writeWait,
	}
}

func NewHub() *Hub {
	return NewHubWithOptions(HubOptions{})
}

func NewHubWithOptions(opts HubOptions) *Hub {
	def := defaultHubOptions()

	if opts.MaxConnections <= 0 {
		opts.MaxConnections = def.MaxConnections
	}
	if opts.MaxMessageSizeBytes <= 0 {
		opts.MaxMessageSizeBytes = def.MaxMessageSizeBytes
	}
	if opts.PongWait <= 0 {
		opts.PongWait = def.PongWait
	}
	if opts.WriteWait <= 0 {
		opts.WriteWait = def.WriteWait
	}
	if opts.PingInterval <= 0 {
		opts.PingInterval = (opts.PongWait * 9) / 10
	}

	return &Hub{
		clients: make(map[*Client]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin:     nil,
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		maxConnections: opts.MaxConnections,
		maxMessageSize: int64(opts.MaxMessageSizeBytes),
		pongWait:       opts.PongWait,
		pingPeriod:     opts.PingInterval,
		writeWait:      opts.WriteWait,
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if len(h.clients) >= h.maxConnections {
		h.mu.Unlock()
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}
	h.mu.Unlock()

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(h.maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(h.pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(h.pongWait))
		return nil
	})

	h.mu.Lock()
	h.nextID++
	clientID := h.nextID
	client := &Client{
		id:   clientID,
		conn: conn,
		send: make(chan []byte, 64),
	}
	h.clients[client] = true
	h.mu.Unlock()

	go h.writePump(client)
	go h.readPump(client)

	helloData := h.HelloData
	if helloData == nil {
		helloData = json.RawMessage(`{}`)
	}
	hello := Event{Type: "hello", Data: helloData}
	data, _ := json.Marshal(hello)
	select {
	case client.send <- data:
	default:
	}
}

func (h *Hub) Broadcast(event Event) {
	if h.BroadcastFn != nil {
		h.BroadcastFn(event)
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("json marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			go h.closeClient(client)
		}
	}
}

func (h *Hub) BroadcastExcept(sender *Client, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("json marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client == sender {
			continue
		}
		select {
		case client.send <- data:
		default:
			go h.closeClient(client)
		}
	}
}

func (h *Hub) Notify(message string, timeout time.Duration) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	timeoutMS := int(timeout / time.Millisecond)
	data, _ := json.Marshal(map[string]interface{}{
		"message": message,
		"timeout": timeoutMS,
	})
	h.Broadcast(Event{Type: "browser:notify", Data: data})
}

func (h *Hub) writePump(client *Client) {
	ticker := time.NewTicker(h.pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
		h.closeClient(client)
	}()

	for {
		select {
		case msg, ok := <-client.send:
			if !ok {
				return
			}
			client.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) readPump(client *Client) {
	defer func() {
		client.conn.Close()
		h.closeClient(client)
	}()

	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		var ev Event
		if err := json.Unmarshal(message, &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "scroll", "click", "input:text", "input:toggles", "form:submit", "form:reset":
			h.BroadcastExcept(client, ev)
		}
	}
}

func (h *Hub) closeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}
