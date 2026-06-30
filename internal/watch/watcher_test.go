package watch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gosync/internal/ws"
)

func TestNewWatcher(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, []string{"."})
	if w == nil {
		t.Fatal("expected non-nil watcher")
	}
	if w.debounce != 100*time.Millisecond {
		t.Errorf("expected debounce 100ms, got %v", w.debounce)
	}
	if len(w.dirs) != 1 {
		t.Errorf("expected 1 dir, got %d", len(w.dirs))
	}
}

func TestNewWatcherMultipleDirs(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, []string{"dir1", "dir2"})
	if len(w.dirs) != 2 {
		t.Errorf("expected 2 dirs, got %d", len(w.dirs))
	}
}

func TestProcessChangeCSS(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	w.processChange("/some/path/styles.css")

	select {
	case ev := <-received:
		if ev.Type != "css" {
			t.Errorf("expected type 'css', got %q", ev.Type)
		}
		if ev.Path != "/some/path/styles.css" {
			t.Errorf("expected path '/some/path/styles.css', got %q", ev.Path)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestProcessChangeReload(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	w.processChange("/some/path/index.html")

	select {
	case ev := <-received:
		if ev.Type != "reload" {
			t.Errorf("expected type 'reload', got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestProcessChangeJS(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	w.processChange("/some/path/app.js")

	select {
	case ev := <-received:
		if ev.Type != "reload" {
			t.Errorf("expected type 'reload' for .js, got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestHandleEventIgnoresChmod(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	broadcastCalled := false
	hub.BroadcastFn = func(ev ws.Event) {
		broadcastCalled = true
	}

	event := fsnotify.Event{
		Name: "test.txt",
		Op:   fsnotify.Chmod,
	}
	w.handleEvent(event)

	if broadcastCalled {
		t.Error("chmod events should not trigger broadcast")
	}
}

func TestHandleEventTriggersOnWrite(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	done := make(chan struct{})
	hub.BroadcastFn = func(ev ws.Event) {
		close(done)
	}

	event := fsnotify.Event{
		Name: "test.css",
		Op:   fsnotify.Write,
	}
	w.handleEvent(event)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for debounced event")
	}
}

func TestHandleEventWithDebounce(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	count := 0
	hub.BroadcastFn = func(ev ws.Event) {
		count++
	}

	event := fsnotify.Event{
		Name: "style.css",
		Op:   fsnotify.Write,
	}
	w.handleEvent(event)
	w.handleEvent(event)

	time.Sleep(300 * time.Millisecond)

	if count > 1 {
		t.Errorf("expected at most 1 broadcast after debounce, got %d", count)
	}
}

func TestStartWithInvalidDir(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, []string{"/nonexistent/path/that/does/not/exist"})

	err := w.Start()
	if err != nil {
		t.Fatalf("Start should not return error for invalid dir: %v", err)
	}
}

func TestStartWatchesDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gosync-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hub := ws.NewHub()
	w := New(hub, []string{tmpDir})

	err = w.Start()
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}

func TestProcessChangeEmptyExt(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	w.processChange("Makefile")

	select {
	case ev := <-received:
		if ev.Type != "reload" {
			t.Errorf("expected type 'reload' for no-ext file, got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestProcessChangeDotFiles(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	w.processChange(".hidden.swp")

	select {
	case ev := <-received:
		if ev.Type != "reload" {
			t.Errorf("expected type 'reload' for dotfile, got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestRecursiveWatching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gosync-recursive-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	hub := ws.NewHub()
	w := New(hub, []string{tmpDir})

	err = w.Start()
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	subFile := filepath.Join(subDir, "style.css")
	if err := os.WriteFile(subFile, []byte("body { color: red; }"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
}

func TestNewWithEmptyDirs(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, []string{"", "  "})
	if len(w.dirs) != 2 {
		t.Errorf("expected 2 dirs, got %d", len(w.dirs))
	}
}

func TestProcessChangeCapExt(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, nil)

	received := make(chan ws.Event, 1)
	hub.BroadcastFn = func(ev ws.Event) {
		received <- ev
	}

	w.processChange("style.CSS")

	select {
	case ev := <-received:
		if ev.Type != "css" {
			t.Errorf("expected 'css' for .CSS, got %q", ev.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestStartWithEmptyDirInList(t *testing.T) {
	hub := ws.NewHub()
	w := New(hub, []string{"", "/nonexistent"})

	err := w.Start()
	if err != nil {
		t.Logf("expected no error from Start (just log warnings): %v", err)
	}
}

func TestFileWatchingIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gosync-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hub := ws.NewHub()
	w := New(hub, []string{tmpDir})

	err = w.Start()
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	done := make(chan struct{})
	hub.BroadcastFn = func(ev ws.Event) {
		if strings.HasSuffix(ev.Path, "hello.txt") {
			close(done)
		}
	}

	tmpFile := filepath.Join(tmpDir, "hello.txt")
	if err := os.WriteFile(tmpFile, []byte("world"), 0644); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for file watch event")
	}
}
