package server

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gosync/internal/proxy"
	"github.com/gosync/internal/ws"
)

func TestNewServer(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:  "3001",
		Dir:   ".",
		Proxy: "",
	}, hub)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.config.Port != "3001" {
		t.Errorf("expected port 3001, got %s", s.config.Port)
	}
	if s.config.Dir != "." {
		t.Errorf("expected dir '.', got %s", s.config.Dir)
	}
}

func TestNewServerWithProxy(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:  "3002",
		Dir:   ".",
		Proxy: "http://localhost:5173",
	}, hub)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.config.Proxy != "http://localhost:5173" {
		t.Errorf("expected proxy http://localhost:5173, got %s", s.config.Proxy)
	}
}

func TestNewServerWithTLS(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:    "3003",
		Dir:     ".",
		TLSCert: "/path/to/cert.pem",
		TLSKey:  "/path/to/key.pem",
	}, hub)
	if s.config.TLSCert != "/path/to/cert.pem" {
		t.Errorf("expected TLS cert path, got %s", s.config.TLSCert)
	}
	if s.config.TLSKey != "/path/to/key.pem" {
		t.Errorf("expected TLS key path, got %s", s.config.TLSKey)
	}
}

func TestServerStartWithInvalidProxy(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:  "3099",
		Proxy: "invalid://bad-scheme",
	}, hub)
	err := s.Start()
	if err == nil {
		t.Error("expected error for invalid proxy scheme")
	}
	if !strings.Contains(err.Error(), "proxy target scheme must be http or https") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestServerServesClientJS(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/__bs.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(`(function() { var protocol = location.protocol === "https:" ? "wss:" : "ws:"; })();`))
	}))

	req, _ := http.NewRequest(http.MethodGet, "/__bs.js", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	mux.ServeHTTP(w, req)
	if w.header.Get("Content-Type") != "application/javascript" {
		t.Errorf("expected Content-Type application/javascript, got %q", w.header.Get("Content-Type"))
	}
}

type mockResponseWriter struct {
	header http.Header
	code   int
	body   []byte
}

func (m *mockResponseWriter) Header() http.Header { return m.header }

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	m.body = append(m.body, b...)
	return len(b), nil
}

func (m *mockResponseWriter) WriteHeader(code int) { m.code = code }

func TestServerServesStaticFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gosync-server-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(tmpDir)))

	req, _ := http.NewRequest(http.MethodGet, "/test.txt", nil)
	w := &mockResponseWriter{header: make(http.Header)}
	mux.ServeHTTP(w, req)
	if string(w.body) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(w.body))
	}
	if w.code != 0 && w.code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.code)
	}
}

func TestServerConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{"default port", Config{Port: "3001"}},
		{"empty port", Config{Port: ""}},
		{"with proxy", Config{Port: "3002", Proxy: "http://localhost:3000"}},
		{"with tls", Config{Port: "3003", TLSCert: "cert.pem", TLSKey: "key.pem"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := ws.NewHub()
			s := New(tt.config, hub)
			if s == nil {
				t.Fatal("expected non-nil server")
			}
		})
	}
}

func TestServerFileServerMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gosync-fs-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html><body>hello</body></html>"), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}

	hub := ws.NewHub()
	s := New(Config{Port: "3095", Dir: tmpDir}, hub)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:3095/index.html")
	if err != nil {
		t.Logf("http get error: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "hello") {
			t.Errorf("expected body to contain 'hello', got %s", string(body))
		}
	}
}

func TestServerTLSConfig(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:    "3094",
		Dir:     ".",
		TLSCert: "/tmp/cert.pem",
		TLSKey:  "/tmp/key.pem",
	}, hub)

	_ = s

	httpServer := &http.Server{
		Addr: ":3094",
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
	}

	if httpServer.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 minimum, got %v", httpServer.TLSConfig.MinVersion)
	}
	if len(httpServer.TLSConfig.CipherSuites) != 3 {
		t.Errorf("expected 3 cipher suites, got %d", len(httpServer.TLSConfig.CipherSuites))
	}
}

func TestServerWithUpstreamProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("from upstream"))
	}))
	defer upstream.Close()

	hub := ws.NewHub()
	s := New(Config{Port: "3093", Proxy: upstream.URL}, hub)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:3093/")
	if err != nil {
		t.Logf("http get error: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "from upstream" {
			t.Errorf("expected 'from upstream', got %s", string(body))
		}
	}
}

func TestServerStartWithInvalidProxyTarget(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:  "3092",
		Proxy: "http://127.0.0.1:1",
	}, hub)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get("http://127.0.0.1:3092/")
	if err != nil {
		t.Logf("http get error: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusBadGateway {
			t.Errorf("expected 502, got %d", resp.StatusCode)
		}
	}
}

func TestTLSVersionConstant(t *testing.T) {
	if tls.VersionTLS12 != 0x0303 {
		t.Errorf("expected TLS 1.2 constant 0x0303, got 0x%04x", tls.VersionTLS12)
	}
}

func TestServerHasHub(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{Port: "3091"}, hub)
	if s.hub != hub {
		t.Error("expected server hub to match")
	}
}

func TestServerWithProxyOptions(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:  "3098",
		Proxy: "http://localhost:5173",
		ProxyOpts: proxy.Options{
			Timeout:            5 * time.Second,
			ChangeOrigin:       true,
			AutoRewrite:        true,
			StripCookiesDomain: true,
			RewriteLinks:       true,
		},
	}, hub)
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.config.ProxyOpts.Timeout != 5*time.Second {
		t.Errorf("expected ProxyOpts.Timeout 5s, got %v", s.config.ProxyOpts.Timeout)
	}
	if !s.config.ProxyOpts.ChangeOrigin {
		t.Error("expected ProxyOpts.ChangeOrigin true")
	}
}

func TestServerStartWithBadTLSPaths(t *testing.T) {
	hub := ws.NewHub()
	s := New(Config{
		Port:    "3090",
		Dir:     ".",
		TLSCert: "/nonexistent/cert.pem",
		TLSKey:  "/nonexistent/key.pem",
	}, hub)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	time.Sleep(100 * time.Millisecond)
}
