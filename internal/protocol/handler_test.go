package protocol

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gosync/internal/ws"
)

func TestGETReload(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodGet, "/__browser_sync__?method=reload", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case ev := <-received:
		if ev.Type != "browser:reload" {
			t.Errorf("expected 'browser:reload', got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}

	if !strings.Contains(string(body), "ok") {
		t.Errorf("expected ok response, got %s", string(body))
	}
}

func TestGETReloadWithArgs(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 2)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodGet, "/__browser_sync__?method=reload&args=core.css&args=index.html", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	events := make([]ws.Event, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case ev := <-received:
			events = append(events, ev)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event %d", i)
		}
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != "file:reload" || events[0].Path != "core.css" {
		t.Errorf("expected file:reload core.css, got type=%q path=%q", events[0].Type, events[0].Path)
	}
	if events[1].Type != "file:reload" || events[1].Path != "index.html" {
		t.Errorf("expected file:reload index.html, got type=%q path=%q", events[1].Type, events[1].Path)
	}
}

func TestGETNotify(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodGet, "/__browser_sync__?method=notify&args=Hello+world", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case ev := <-received:
		if ev.Type != "browser:notify" {
			t.Errorf("expected 'browser:notify', got %q", ev.Type)
		}
		if !strings.Contains(string(ev.Data), "Hello world") {
			t.Errorf("expected data to contain 'Hello world', got %s", string(ev.Data))
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notify")
	}
}

func TestGETExit(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodGet, "/__browser_sync__?method=exit", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("expected ok response, got %s", string(body))
	}
}

func TestGETUnknownMethod(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodGet, "/__browser_sync__?method=unknown", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPOSTReload(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	body := `{"type":"browser:reload"}`
	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case ev := <-received:
		if ev.Type != "browser:reload" {
			t.Errorf("expected 'browser:reload', got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestPOSTNotify(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	body := `{"type":"browser:notify","data":{"message":"test","timeout":3000}}`
	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case ev := <-received:
		if ev.Type != "browser:notify" {
			t.Errorf("expected 'browser:notify', got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notify")
	}
}

func TestPOSTLocation(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	body := `{"type":"browser:location","data":{"url":"/new-page"}}`
	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case ev := <-received:
		if ev.Type != "browser:location" {
			t.Errorf("expected 'browser:location', got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for location event")
	}
}

func TestPOSTUnknownEvent(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	body := `{"type":"unknown"}`
	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPOSTInvalidJSON(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	body := `not-json`
	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req := httptest.NewRequest(method, "/__browser_sync__", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 for %s, got %d", method, resp.StatusCode)
		}
	}
}

func TestPOSTEmptyBody(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestNewHandler(t *testing.T) {
	hub := ws.NewHub()
	h := NewHandler(hub)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.hub != hub {
		t.Error("expected hub to match")
	}
}

func TestGETReloadProducesValidJSON(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodGet, "/__browser_sync__?method=reload", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Errorf("expected valid JSON response, got error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", result["status"])
	}
}

func TestPOSTOptionsSet(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	body := `{"type":"options:set","data":{"ghostMode":false}}`
	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case ev := <-received:
		if ev.Type != "options:set" {
			t.Errorf("expected 'options:set', got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for options:set")
	}
}

func TestGETReloadNoMethodParam(t *testing.T) {
	hub := ws.NewHub()
	handler := NewHandler(hub)

	req := httptest.NewRequest(http.MethodGet, "/__browser_sync__", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing method, got %d", resp.StatusCode)
	}
}

func TestPOSTFileReload(t *testing.T) {
	hub := ws.NewHub()

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	handler := NewHandler(hub)

	body := `{"type":"file:reload","data":{"path":"style.css"}}`
	req := httptest.NewRequest(http.MethodPost, "/__browser_sync__", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	select {
	case ev := <-received:
		if ev.Type != "file:reload" {
			t.Errorf("expected 'file:reload', got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for file:reload")
	}
}
