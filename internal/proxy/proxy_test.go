package proxy

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNewValidHTTPTarget(t *testing.T) {
	h, err := New("http://localhost:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestNewValidHTTPSTarget(t *testing.T) {
	h, err := New("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestNewRejectsInvalidScheme(t *testing.T) {
	targets := []string{
		"file:///etc/passwd",
		"gopher://internal:25",
		"dict://internal:6379",
		"ftp://files.example.com",
		"ldap://ldap.example.com",
	}

	for _, target := range targets {
		h, err := New(target)
		if err == nil {
			t.Errorf("expected error for scheme %q, got handler %v", target, h)
		}
	}
}

func TestNewRejectsEmptyHost(t *testing.T) {
	targets := []string{
		"http:///path",
		"https:///path",
	}

	for _, target := range targets {
		h, err := New(target)
		if err == nil {
			t.Errorf("expected error for empty host in %q, got handler %v", target, h)
		}
	}
}

func TestNewRejectsInvalidURL(t *testing.T) {
	targets := []string{
		":invalid",
		"",
		"not a url",
	}

	for _, target := range targets {
		h, err := New(target)
		if err == nil {
			t.Errorf("expected error for invalid URL %q, got handler %v", target, h)
		}
	}
}

func TestProxyForwardsHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Error("expected X-Forwarded-For header")
		}
		if r.Header.Get("X-Forwarded-Proto") == "" {
			t.Error("expected X-Forwarded-Proto header")
		}
		if r.Header.Get("X-Forwarded-Host") == "" {
			t.Error("expected X-Forwarded-Host header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy, err := New(upstream.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	req.Host = "example.com"
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestProxyErrorHandler(t *testing.T) {
	proxy, err := New("http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}

func TestDirectorModifiesRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		if xff == "" {
			t.Error("expected X-Forwarded-For to be set")
		}
		xproto := r.Header.Get("X-Forwarded-Proto")
		if xproto != "http" {
			t.Errorf("expected X-Forwarded-Proto=http, got %q", xproto)
		}
		xhost := r.Header.Get("X-Forwarded-Host")
		if xhost != "test.example.com" {
			t.Errorf("expected X-Forwarded-Host=test.example.com, got %q", xhost)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(upstream.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:8080"
	req.Host = "test.example.com"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.2:8080"
	req2.Host = "test.example.com"
	req2.Header.Set("X-Forwarded-For", "1.2.3.4")
	w2 := httptest.NewRecorder()
	p.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w2.Code)
	}
}

func TestProxyTypeImplementsHandler(t *testing.T) {
	p, err := New("http://localhost:3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handlerType := reflect.TypeOf((*http.Handler)(nil)).Elem()
	if !reflect.TypeOf(p).Implements(handlerType) {
		t.Error("proxy should implement http.Handler")
	}
}

func TestNewWithOptionsBasic(t *testing.T) {
	h, err := NewWithOptions("http://localhost:4000", Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestNewWithOptionsTimeout(t *testing.T) {
	h, err := NewWithOptions("http://localhost:4000", Options{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestNewWithOptionsRejectsInvalidScheme(t *testing.T) {
	_, err := NewWithOptions("file:///etc/passwd", Options{Timeout: 5 * time.Second})
	if err == nil {
		t.Fatal("expected error for invalid scheme")
	}
}

func TestNewWithOptionsRejectsEmptyHost(t *testing.T) {
	_, err := NewWithOptions("http:///path", Options{Timeout: 5 * time.Second})
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestNewWithOptionsRejectsInvalidURL(t *testing.T) {
	_, err := NewWithOptions(":", Options{Timeout: 5 * time.Second})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestNewWithOptionsForwardsHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Error("expected X-Forwarded-For header")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:8080"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNewWithOptionsErrorHandler(t *testing.T) {
	p, err := NewWithOptions("http://127.0.0.1:1", Options{Timeout: time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", resp.StatusCode)
	}
}
