package protocol

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gosync/internal/ws"
)

type Handler struct {
	hub *ws.Hub
}

func NewHandler(hub *ws.Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.handleGET(w, r)
		return
	}
	if r.Method == http.MethodPost {
		h.handlePOST(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) handleGET(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	args := r.URL.Query()["args"]

	switch method {
	case "reload":
		if len(args) > 0 {
			for _, arg := range args {
				h.hub.Broadcast(ws.Event{Type: "file:reload", Path: arg})
			}
		} else {
			h.hub.Broadcast(ws.Event{Type: "browser:reload"})
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

	case "notify":
		message := "Reloading..."
		if len(args) > 0 {
			message = args[0]
		}
		h.hub.Notify(message, 0)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

	case "exit":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))

	default:
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

type protocolPayload struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

func (h *Handler) handlePOST(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var payload protocolPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	switch payload.Type {
	case "file:reload", "browser:reload", "browser:notify", "browser:location", "options:set":
		h.hub.Broadcast(ws.Event{
			Type: payload.Type,
			Data: payload.Data,
		})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	default:
		http.Error(w, "Unknown event type", http.StatusBadRequest)
	}
}
