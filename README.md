# gosync

A fast, lightweight BrowserSync clone written in Go. Watches files and synchronizes changes to browser tabs in real-time via WebSocket.

## Features

- **Static file server** — serve a directory of files
- **Reverse proxy** — proxy to an upstream dev server (Vite, React, etc.)
- **Live reload** — full page reload on HTML/JS file changes
- **CSS injection** — hot-swap stylesheets without a full reload
- **File watching** — recursive directory watching with debounce
- **WebSocket sync** — real-time events pushed to all connected browsers
- **Scroll sync** — synchronized scrolling across browser tabs
- **TLS support** — optional HTTPS/WSS with modern cipher suites

## Installation

```bash
go install github.com/gosync/cmd/gosync@latest
```

Or build from source:

```bash
git clone <repo-url>
cd gosync
CGO_ENABLED=0 go build -o gosync ./cmd/gosync
```

## Usage

### Serve static files

```bash
gosync --port 3001 --dir ./public --watch ./public
```

### Proxy to a dev server

```bash
gosync --port 3001 --proxy http://localhost:5173 --watch ./src
```

### Enable HTTPS

```bash
gosync --port 443 --tls-cert ./cert.pem --tls-key ./key.pem
```

### All options

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `3001` | Port to listen on |
| `--dir` | `.` | Static files directory |
| `--proxy` | `""` | Upstream proxy target URL |
| `--watch` | `.` | Comma-separated directories to watch |
| `--tls-cert` | `""` | TLS certificate file path |
| `--tls-key` | `""` | TLS private key file path |

## Architecture

```
                     ┌──────────────┐
                     │  File Watcher │
                     └──────┬───────┘
                            │ change event
                            ▼
          ┌──────────────┐  WebSocket   ┌──────────────┐
          │ Go HTTP      │◀────────────▶│ Browser Tabs │
          │ Server       │              └──────────────┘
          │ (proxy + UI) │
          └──────┬───────┘
                 │
                 ▼
          ┌──────────────┐
          │ Static files │
          │ or upstream  │
          │ dev server   │
          └──────────────┘
```

## Project structure

```
cmd/gosync/        — CLI entrypoint
internal/server/   — HTTP server setup
internal/proxy/    — Reverse proxy
internal/ws/       — WebSocket hub
internal/watch/    — File watcher
internal/inject/   — HTML injection middleware
internal/clientjs/ — Embedded client JavaScript
```

## How it works

1. **HTTP server** starts in either static file or reverse proxy mode
2. **Middleware** injects a `<script src="/__bs.js">` tag into HTML responses
3. **Client JS** connects to the server via WebSocket and listens for events
4. **File watcher** monitors directories for changes using fsnotify
5. On change: CSS files trigger stylesheet hot-swap; everything else triggers a full reload
6. **Scroll/form sync** broadcasts user interactions to all connected clients

## Security

- Same-origin WebSocket policy (no CSWSH)
- Proxy target restricted to http/https schemes (no SSRF)
- HTTP timeouts configured (Slowloris protection)
- WebSocket rate limiting (max 100 concurrent connections)
- Read/write deadlines and message size limits

## License

MIT
