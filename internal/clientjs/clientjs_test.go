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

func TestClientJSWebSocket(t *testing.T) {
	if !strings.Contains(ClientJS, "WebSocket") {
		t.Error("expected client JS to contain WebSocket code")
	}
	if !strings.Contains(ClientJS, "/__bs/ws") {
		t.Error("expected client JS to reference WebSocket endpoint")
	}
	if !strings.Contains(ClientJS, "wss:") {
		t.Error("expected client JS to handle wss:// protocol")
	}
	if !strings.Contains(ClientJS, "ws:") {
		t.Error("expected client JS to handle ws:// protocol")
	}
}

func TestClientJSRellEvents(t *testing.T) {
	if !strings.Contains(ClientJS, "browser:reload") {
		t.Error("expected client JS to handle browser:reload")
	}
	if !strings.Contains(ClientJS, "location.reload()") {
		t.Error("expected client JS to handle reload")
	}
}

func TestClientJSCSSEvents(t *testing.T) {
	if !strings.Contains(ClientJS, "css") {
		t.Error("expected client JS to handle CSS events")
	}
	if !strings.Contains(ClientJS, "link[rel=stylesheet]") {
		t.Error("expected client JS to handle stylesheet reloading")
	}
}

func TestClientJSScrollEvents(t *testing.T) {
	if !strings.Contains(ClientJS, "scroll") {
		t.Error("expected client JS to handle scroll events")
	}
	if !strings.Contains(ClientJS, "window.scrollTo") {
		t.Error("expected client JS to scroll via window.scrollTo")
	}
}

func TestClientJSClickSync(t *testing.T) {
	if !strings.Contains(ClientJS, "click") {
		t.Error("expected client JS to handle click events")
	}
	if !strings.Contains(ClientJS, "simulateClick") {
		t.Error("expected client JS to simulate clicks")
	}
	if !strings.Contains(ClientJS, "getElementIndex") {
		t.Error("expected client JS to compute element index")
	}
}

func TestClientJSFormSync(t *testing.T) {
	if !strings.Contains(ClientJS, "input:text") {
		t.Error("expected client JS to handle input:text events")
	}
	if !strings.Contains(ClientJS, "input:toggles") {
		t.Error("expected client JS to handle input:toggles events")
	}
	if !strings.Contains(ClientJS, "form:submit") {
		t.Error("expected client JS to handle form:submit events")
	}
	if !strings.Contains(ClientJS, "form:reset") {
		t.Error("expected client JS to handle form:reset events")
	}
}

func TestClientJSNotification(t *testing.T) {
	if !strings.Contains(ClientJS, "browser:notify") {
		t.Error("expected client JS to handle browser:notify events")
	}
	if !strings.Contains(ClientJS, "__bs_notification") {
		t.Error("expected client JS to have notification UI")
	}
}

func TestClientJSLocationSync(t *testing.T) {
	if !strings.Contains(ClientJS, "browser:location") {
		t.Error("expected client JS to handle browser:location events")
	}
	if !strings.Contains(ClientJS, "location.href") {
		t.Error("expected client JS to navigate via location.href")
	}
}

func TestClientJSHelloEvent(t *testing.T) {
	if !strings.Contains(ClientJS, "hello") {
		t.Error("expected client JS to handle hello event")
	}
	if !strings.Contains(ClientJS, "options") {
		t.Error("expected client JS to store server options")
	}
}

func TestClientJSReconnect(t *testing.T) {
	if !strings.Contains(ClientJS, "scheduleReconnect") {
		t.Error("expected client JS to have reconnection logic")
	}
	if !strings.Contains(ClientJS, "Math.pow") {
		t.Error("expected client JS to have exponential backoff")
	}
}

func TestClientJSIIFE(t *testing.T) {
	if !strings.HasPrefix(strings.TrimSpace(ClientJS), "(function()") {
		t.Error("expected client JS to be wrapped in an IIFE")
	}
}

func TestClientJSSendFunction(t *testing.T) {
	if !strings.Contains(ClientJS, "JSON.stringify(event)") {
		t.Error("expected client JS to send events via socket")
	}
	if !strings.Contains(ClientJS, "socket.send") {
		t.Error("expected client JS to call socket.send")
	}
}

func TestClientJSGhostModeListeners(t *testing.T) {
	if !strings.Contains(ClientJS, "addEventListener('input'") {
		t.Error("expected client JS to listen for input events")
	}
	if !strings.Contains(ClientJS, "addEventListener('change'") {
		t.Error("expected client JS to listen for change events")
	}
	if !strings.Contains(ClientJS, "addEventListener('submit'") {
		t.Error("expected client JS to listen for submit events")
	}
	if !strings.Contains(ClientJS, "addEventListener('reset'") {
		t.Error("expected client JS to listen for reset events")
	}
}

func TestClientJSSetInputValue(t *testing.T) {
	if !strings.Contains(ClientJS, "setInputValue") {
		t.Error("expected client JS to have setInputValue function")
	}
	if !strings.Contains(ClientJS, "HTMLInputElement.prototype") {
		t.Error("expected client JS to use native value setter")
	}
}

func TestClientJSSetToggleValue(t *testing.T) {
	if !strings.Contains(ClientJS, "setToggleValue") {
		t.Error("expected client JS to have setToggleValue function")
	}
	if !strings.Contains(ClientJS, "el.checked") {
		t.Error("expected client JS to set checked property")
	}
}

func TestClientJSDispatchFormEvent(t *testing.T) {
	if !strings.Contains(ClientJS, "dispatchEvent") {
		t.Error("expected client JS to dispatch DOM events")
	}
}

func TestClientJSExponentialBackoff(t *testing.T) {
	if !strings.Contains(ClientJS, "Math.min") {
		t.Error("expected client JS to cap reconnect delay")
	}
	if !strings.Contains(ClientJS, "30000") {
		t.Error("expected client JS to have max 30s reconnect delay")
	}
}
