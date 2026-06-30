package main

import (
	"strings"
	"testing"
	"time"

	"github.com/gosync/internal/config"
)

func TestRunWithInvalidProxy(t *testing.T) {
	cfg := &config.Config{
		Port:  "4599",
		Dir:   ".",
		Proxy: "invalid://bad-scheme",
		Watch: []string{"."},
	}
	cfg.ApplyDefaults()

	err := run(cfg)
	if err == nil {
		t.Fatal("expected error for invalid proxy scheme")
	}
	if !strings.Contains(err.Error(), "proxy target scheme must be http or https") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunTLSWithMissingKey(t *testing.T) {
	cfg := &config.Config{
		Port:    "4596",
		Dir:     ".",
		Watch:   []string{"."},
		TLSCert: "cert.pem",
	}
	cfg.ApplyDefaults()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(cfg)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Logf("error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRunInvalidWatchDirInGoroutine(t *testing.T) {
	cfg := &config.Config{
		Port:  "4597",
		Dir:   ".",
		Watch: []string{"/nonexistent/path"},
	}
	cfg.ApplyDefaults()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(cfg)
	}()

	select {
	case err := <-errCh:
		t.Logf("error: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRunDedupWatchDirsInGoroutine(t *testing.T) {
	cfg := &config.Config{
		Port:  "4595",
		Dir:   ".",
		Watch: []string{"dir1", "dir2", "dir1"},
	}
	cfg.ApplyDefaults()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(cfg)
	}()

	select {
	case err := <-errCh:
		t.Logf("error: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestRunWithProxyTimeout(t *testing.T) {
	timeout := 30
	cfg := &config.Config{
		Port:             "4594",
		Dir:              ".",
		Proxy:            "ftp://bad",
		ProxyTimeoutSecs: &timeout,
		Watch:            []string{"."},
	}
	cfg.ApplyDefaults()

	err := run(cfg)
	if err == nil {
		t.Fatal("expected error for invalid proxy scheme")
	}
}

func TestRunWithCustomHubOptions(t *testing.T) {
	rateLimit := 50
	msgSize := 2048
	pongWait := 30
	writeWait := 5
	pingInterval := 25

	cfg := &config.Config{
		Port:  "4593",
		Dir:   ".",
		Watch: []string{"."},
		HubOpts: config.HubOptions{
			RateLimitConns:       &rateLimit,
			MaxMsgSizeBytes:      &msgSize,
			PongWaitSecs:         &pongWait,
			WriteWaitSecs:        &writeWait,
			PingPongIntervalSecs: &pingInterval,
		},
	}
	cfg.ApplyDefaults()

	errCh := make(chan error, 1)
	go func() {
		errCh <- run(cfg)
	}()

	select {
	case err := <-errCh:
		t.Logf("error: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
}
