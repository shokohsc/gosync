package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type Options struct {
	Timeout time.Duration
}

func New(target string) (http.Handler, error) {
	return NewWithOptions(target, Options{})
}

func NewWithOptions(target string, opts Options) (http.Handler, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy target %q: %w", target, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("proxy target scheme must be http or https, got %q", u.Scheme)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("proxy target must have a valid host")
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	if opts.Timeout > 0 {
		proxy.Transport = &http.Transport{
			ResponseHeaderTimeout: opts.Timeout,
		}
	}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		if clientIP := req.Header.Get("X-Forwarded-For"); clientIP != "" {
			req.Header.Set("X-Forwarded-For", clientIP+", "+req.RemoteAddr)
		} else {
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		}

		if req.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}

		if req.Host != "" {
			req.Header.Set("X-Forwarded-Host", req.Host)
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return proxy, nil
}
