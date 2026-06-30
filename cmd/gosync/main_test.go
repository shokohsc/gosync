package main

import (
	"testing"
	"time"

	"github.com/gosync/internal/config"
)

func TestRunWithInvalidProxy(t *testing.T) {
	cfg := &config.Config{
		Port:  "4599",
		Dir:   ".",
		Proxy: "invalid://bad-scheme",
	}
	cfg.ApplyDefaults()

	err := run(cfg)
	if err == nil {
		t.Fatal("expected error for invalid proxy scheme")
	}
}

func TestRunTLSWithMissingKey(t *testing.T) {
	cfg := &config.Config{
		Port:    "4596",
		Dir:     ".",
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

func TestRunWithProxyTimeout(t *testing.T) {
	timeout := 30
	cfg := &config.Config{
		Port:             "4594",
		Dir:              ".",
		Proxy:            "ftp://bad",
		ProxyTimeoutSecs: &timeout,
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
		Port: "4593",
		Dir:  ".",
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
