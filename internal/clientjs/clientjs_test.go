package clientjs

import (
	"strings"
	"testing"
)

func TestClientJSIsEmbedded(t *testing.T) {
	if ClientJS == "" {
		t.Fatal("expected ClientJS to be non-empty")
	}
}

func TestClientJSContent(t *testing.T) {
	if !strings.Contains(ClientJS, "WebSocket") {
		t.Error("expected client JS to contain WebSocket code")
	}
	if !strings.Contains(ClientJS, "/__bs/ws") {
		t.Error("expected client JS to reference WebSocket endpoint")
	}
	if !strings.Contains(ClientJS, "scroll") {
		t.Error("expected client JS to handle scroll events")
	}
	if !strings.Contains(ClientJS, "reload") {
		t.Error("expected client JS to handle reload events")
	}
	if !strings.Contains(ClientJS, "css") {
		t.Error("expected client JS to handle CSS events")
	}
}

func TestClientJSProtocolDetection(t *testing.T) {
	if !strings.Contains(ClientJS, "wss:") {
		t.Error("expected client JS to handle wss:// protocol")
	}
	if !strings.Contains(ClientJS, "ws:") {
		t.Error("expected client JS to handle ws:// protocol")
	}
}

func TestClientJSIIFE(t *testing.T) {
	if !strings.HasPrefix(strings.TrimSpace(ClientJS), "(function()") {
		t.Error("expected client JS to be wrapped in an IIFE")
	}
}
