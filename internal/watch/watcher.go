package watch

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gosync/internal/ws"
)

type Watcher struct {
	hub      *ws.Hub
	dirs     []string
	debounce time.Duration
	mu       sync.Mutex
	timers   map[string]*time.Timer
}

func New(hub *ws.Hub, dirs []string) *Watcher {
	return &Watcher{
		hub:      hub,
		dirs:     dirs,
		debounce: 100 * time.Millisecond,
		timers:   make(map[string]*time.Timer),
	}
}

func (w *Watcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	for _, dir := range w.dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		absDir, err := filepath.Abs(dir)
		if err != nil {
			log.Printf("invalid watch dir %s: %v", dir, err)
			continue
		}
		if err := watcher.Add(absDir); err != nil {
			log.Printf("failed to watch %s: %v", absDir, err)
			continue
		}
		log.Printf("watching: %s", absDir)

		// Watch subdirectories recursively
		filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // skip errors
			}
			if info.IsDir() {
				if err := watcher.Add(path); err != nil {
					log.Printf("failed to watch %s: %v", path, err)
				}
			}
			return nil
		})
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				w.handleEvent(event)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)
			}
		}
	}()

	return nil
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) == 0 {
		return
	}

	w.mu.Lock()

	key := event.Name
	if t, ok := w.timers[key]; ok {
		t.Stop()
	}

	w.timers[key] = time.AfterFunc(w.debounce, func() {
		w.mu.Lock()
		// Check if our timer is still the registered one for this key.
		// If another event came in and replaced it, ours is stale - skip.
		if _, exists := w.timers[key]; exists {
			delete(w.timers, key)
			w.mu.Unlock()
			w.processChange(event.Name)
		} else {
			w.mu.Unlock()
		}
	})
	w.mu.Unlock()
}

func (w *Watcher) processChange(path string) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".css":
		w.hub.Broadcast(ws.Event{
			Type: "css",
			Path: path,
		})
		log.Printf("css update: %s", path)
	default:
		w.hub.Broadcast(ws.Event{
			Type: "reload",
			Path: path,
		})
		log.Printf("reload: %s", path)
	}
}
