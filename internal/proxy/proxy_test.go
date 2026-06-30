package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
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
	_, err := NewWithOptions("file:///etc/passwd", Options{})
	if err == nil {
		t.Fatal("expected error for invalid scheme")
	}
}

func TestNewWithOptionsChangeOrigin(t *testing.T) {
	var addr string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != addr {
			t.Errorf("expected Host=%q (target), got %q", addr, r.Host)
		}
		if r.Header.Get("X-Forwarded-Host") != "original-host.com" {
			t.Errorf("expected X-Forwarded-Host=original-host.com, got %q", r.Header.Get("X-Forwarded-Host"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	addr = upstream.Listener.Addr().String()
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{ChangeOrigin: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "original-host.com"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNewWithOptionsAutoRewrite(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", r.Header.Get("X-Forwarded-Host"))
		w.WriteHeader(http.StatusFound)
	}))
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{AutoRewrite: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "proxy-host.com:3001"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}

	loc := resp.Header.Get("Location")
	if strings.Contains(loc, upstream.URL) {
		t.Errorf("expected Location to not contain upstream URL %q, got %q", upstream.URL, loc)
	}
	if !strings.Contains(loc, "proxy-host.com:3001") {
		t.Errorf("expected Location to contain proxy host, got %q", loc)
	}
}

func TestNewWithOptionsStripCookiesDomain(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=abc; Domain=.upstream.com; Path=/")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{StripCookiesDomain: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	cookie := resp.Header.Get("Set-Cookie")
	if strings.Contains(cookie, "Domain=") || strings.Contains(cookie, "domain=") {
		t.Errorf("expected cookie to have no Domain, got %q", cookie)
	}
	if !strings.Contains(cookie, "session=abc") {
		t.Errorf("expected cookie to retain session=abc, got %q", cookie)
	}
}

func TestNewWithOptionsRewriteLinks(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<a href="http://` + r.Header.Get("X-Forwarded-Host") + `/page">link</a>`))
	}))
	defer upstream.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "myproxy:3001"

	p, err := NewWithOptions(upstream.URL, Options{RewriteLinks: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if strings.Contains(string(body), upstream.URL) {
		t.Errorf("expected body to not contain upstream URL, got %s", string(body))
	}
	if !strings.Contains(string(body), "//myproxy:3001") {
		t.Errorf("expected body to contain proxy host, got %s", string(body))
	}
}

func TestNewWithOptionsRequestHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "value1" {
			t.Errorf("expected X-Custom=value1, got %q", r.Header.Get("X-Custom"))
		}
		if r.Header.Get("X-Debug") != "true" {
			t.Errorf("expected X-Debug=true, got %q", r.Header.Get("X-Debug"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{
		RequestHeaders: map[string]string{
			"X-Custom": "value1",
			"X-Debug":  "true",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNewWithOptionsXForwardedHostPreserved(t *testing.T) {
	var addr string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Host") != "original-host.com" {
			t.Errorf("expected X-Forwarded-Host=original-host.com, got %q", r.Header.Get("X-Forwarded-Host"))
		}
		if r.Host != addr {
			t.Errorf("expected Host=%q (target), got %q", addr, r.Host)
		}
		w.WriteHeader(http.StatusOK)
	}))
	addr = upstream.Listener.Addr().String()
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{ChangeOrigin: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "original-host.com"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNewWithOptionsInsecureSkipVerify(t *testing.T) {
	p, err := NewWithOptions("https://localhost:4443", Options{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil handler")
	}

	proxyImpl, ok := p.(*Proxy)
	if !ok {
		t.Fatal("expected *Proxy type")
	}
	if proxyImpl.opts.InsecureSkipVerify != true {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestStripCookieDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"session=abc; Domain=.example.com; Path=/", "session=abc; Path=/"},
		{"session=abc; domain=.example.com; Path=/", "session=abc; Path=/"},
		{"session=abc; Path=/", "session=abc; Path=/"},
		{"session=abc; Domain=.example.com", "session=abc"},
	}

	for _, tt := range tests {
		result := stripCookieDomain(tt.input)
		if result != tt.expected {
			t.Errorf("stripCookieDomain(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRewriteLinksReplacesTarget(t *testing.T) {
	p := &Proxy{target: &url.URL{Host: "oldhost:8080"}}

	body := p.rewriteLinks(
		[]byte(`<a href="http://oldhost:8080/page">link</a>`),
		&http.Request{Host: "newhost:3001"},
	)
	result := string(body)
	if !strings.Contains(result, "//newhost:3001") {
		t.Errorf("expected body to contain new host, got %s", result)
	}
	if strings.Contains(result, "//oldhost:8080") {
		t.Errorf("expected body to not contain old host, got %s", result)
	}
}

func TestRewriteLinksSkipsSameHost(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body><a href="/relative">link</a></body></html>`))
	}))
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{RewriteLinks: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proxyImpl, ok := p.(*Proxy)
	if !ok {
		t.Fatal("expected *Proxy type")
	}

	body := proxyImpl.rewriteLinks(
		[]byte(`<a href="http://samehost:3001/page">link</a>`),
		&http.Request{Host: "samehost:3001"},
	)
	result := string(body)
	if !strings.Contains(result, "//samehost:3001") {
		t.Errorf("expected body to retain same host, got %s", result)
	}
}

func TestIsHTML(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")
	if !isHTML(resp) {
		t.Error("expected true for text/html")
	}

	resp.Header.Set("Content-Type", "application/json")
	if isHTML(resp) {
		t.Error("expected false for application/json")
	}
}

func TestIsRedirect(t *testing.T) {
	for _, code := range []int{301, 302, 307, 308} {
		if !isRedirect(code) {
			t.Errorf("expected true for %d", code)
		}
	}
	if isRedirect(200) {
		t.Error("expected false for 200")
	}
	if isRedirect(404) {
		t.Error("expected false for 404")
	}
}

func TestIsCompressed(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Content-Encoding", "gzip")
	if !isCompressed(resp) {
		t.Error("expected true for gzip")
	}

	resp.Header.Set("Content-Encoding", "")
	if isCompressed(resp) {
		t.Error("expected false for empty")
	}
}

func TestRewriteLinksNoTargetHost(t *testing.T) {
	p := &Proxy{target: &url.URL{Host: ""}}
	body := p.rewriteLinks([]byte("content"), &http.Request{Host: "host"})
	if string(body) != "content" {
		t.Errorf("expected unchanged body, got %s", string(body))
	}
}

func TestRewriteLinksNoReqHost(t *testing.T) {
	p := &Proxy{target: &url.URL{Host: "target.com"}}
	body := p.rewriteLinks([]byte("content"), &http.Request{Host: ""})
	if string(body) != "content" {
		t.Errorf("expected unchanged body, got %s", string(body))
	}
}

func TestNewWithOptionsRewriteLinksSkipsCompressed(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := NewWithOptions(upstream.URL, Options{RewriteLinks: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
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
