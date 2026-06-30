# gosync

A fast, lightweight BrowserSync clone written in Go. Uses a reverse proxy and WebSocket to synchronize browser tabs in real-time.

## Features

- **Static file server** — serve a directory of files directly
- **Reverse proxy** — proxy to an upstream dev server (Vite, React, etc.) with BrowserSync-compatible features
- **Proxy — changeOrigin** — rewrite the `Host` header to match the upstream target
- **Proxy — autoRewrite** — rewrite `Location` headers in 3xx redirects so they point to the proxy host
- **Proxy — cookieDomainRewrite** — strip `Domain` from `Set-Cookie` headers to prevent cookie scope mismatch
- **Proxy — rewriteLinks** — replace the target host with the proxy host inside HTML response bodies
- **Proxy — custom headers** — inject custom request headers to upstream targets
- **Proxy — insecure TLS** — skip upstream TLS certificate verification (opt-in)
- **Proxy — configurable timeout** — set a response header timeout for upstream requests
- **CSS injection** — hot-swap stylesheets without a full reload
- **WebSocket sync** — real-time events pushed to all connected browsers
- **Scroll sync** — synchronized scrolling across browser tabs
- **TLS support** — optional HTTPS/WSS with modern cipher suites
- **Config file** — YAML-based configuration (`.gosync.yaml`)
- **Environment variable overrides** — all options configurable via env vars
- **Configurable WebSocket hub** — tune rate limits, message sizes, timeouts
- **Docker support** — multi-arch (amd64/arm64) scratch-based container images

## Dependencies

- [gorilla/websocket](https://github.com/gorilla/websocket) — WebSocket hub
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) — YAML config parsing

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
gosync --port 3001 --dir ./public
```

### Proxy to a dev server

```bash
gosync --port 3001 --proxy http://localhost:5173
```

### Enable HTTPS

```bash
gosync --port 443 --tls-cert ./cert.pem --tls-key ./key.pem
```

### CLI flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `3001` | Port to listen on |
| `--dir` | `.` | Static files directory |
| `--proxy` | `""` | Upstream proxy target URL |
| `--tls-cert` | `""` | TLS certificate file path |
| `--tls-key` | `""` | TLS private key file path |

## Configuration

gosync supports four configuration sources, listed in priority order (highest first):

1. **CLI flags** — command-line arguments (see above)
2. **Environment variables** — override config file and defaults
3. **Config file** — `.gosync.yaml` (path overridable via `GOSYNC_CONFIG`)
4. **Built-in defaults** — sensible defaults for all options

### Config file

By default, gosync reads `.gosync.yaml` from the current directory. Use the
`GOSYNC_CONFIG` environment variable to specify a different path.

```yaml
port: "3001"
dir: "."
proxy: "http://localhost:5173"

# Proxy features (BrowserSync-compatible)
proxy_timeout_seconds: 30
proxy_change_origin: true
proxy_auto_rewrite: true
proxy_strip_cookies: true
proxy_rewrite_links: true
proxy_insecure: false

# TLS
tls_cert: ""
tls_key: ""

hub_options:
  rate_limit_conns: 100
  max_msg_size_bytes: 4096
  pong_wait_seconds: 60
  ping_pong_interval_seconds: 54
  write_wait_seconds: 10
```

### Environment variables

All config file options can be overridden with environment variables.
Individual hub option env vars take precedence over `GOSYNC_HUB_OPTIONS`.

| Env variable | Description |
|---|---|
| `GOSYNC_CONFIG` | Path to YAML config file (default: `.gosync.yaml`) |
| `GOSYNC_PORT` | Port to listen on |
| `GOSYNC_DIR` | Static files directory |
| `GOSYNC_PROXY` | Upstream proxy target URL |
| `GOSYNC_TLS_CERT` | TLS certificate file path |
| `GOSYNC_TLS_KEY` | TLS private key file path |
| `GOSYNC_PROXY_TIMEOUT_SECONDS` | Proxy response header timeout |
| `GOSYNC_PROXY_CHANGE_ORIGIN` | Rewrite Host header to upstream target |
| `GOSYNC_PROXY_AUTO_REWRITE` | Rewrite Location headers in redirects |
| `GOSYNC_PROXY_STRIP_COOKIES` | Strip Domain from Set-Cookie headers |
| `GOSYNC_PROXY_REWRITE_LINKS` | Rewrite target host in HTML bodies |
| `GOSYNC_PROXY_INSECURE` | Skip upstream TLS verification |
| `GOSYNC_HUB_OPTIONS` | JSON object overriding hub options (see below) |
| `GOSYNC_RATE_LIMIT_CONNS` | Max concurrent WebSocket connections |
| `GOSYNC_MAX_MSG_SIZE_BYTES` | Max WebSocket message size in bytes |
| `GOSYNC_PING_PONG_INTERVAL_SECONDS` | Ping/pong interval |
| `GOSYNC_PONG_WAIT_SECONDS` | Pong response wait time |
| `GOSYNC_WRITE_WAIT_SECONDS` | Write deadline |

### HubOptions

The WebSocket hub parameters are fully configurable to suit your workload:

| Option | Default | Description |
|---|---|---|
| `rate_limit_conns` | `100` | Maximum concurrent WebSocket connections |
| `max_msg_size_bytes` | `4096` | Maximum message size in bytes |
| `ping_pong_interval_seconds` | `54` | Interval between pings |
| `pong_wait_seconds` | `60` | Time to wait for pong before disconnect |
| `write_wait_seconds` | `10` | Write deadline for outgoing messages |

Set these via the config file `hub_options:` section, `GOSYNC_HUB_OPTIONS`
JSON env var, or individual env vars (`GOSYNC_RATE_LIMIT_CONNS`, etc.).

Example with environment variables:

```bash
GOSYNC_HUB_OPTIONS='{"RateLimitConns": 200, "MaxMsgSizeBytes": 8192}' gosync
```

Or override individual values (takes precedence over `GOSYNC_HUB_OPTIONS`):

```bash
GOSYNC_RATE_LIMIT_CONNS=200 GOSYNC_MAX_MSG_SIZE_BYTES=8192 gosync
```

## Docker

### Quick start

```bash
docker run --rm -p 3001:3001 -v ./myapp:/app ghcr.io/gosync/gosync:latest \
  --dir /app
```

### Multi-arch images

Published images support both `linux/amd64` and `linux/arm64` platforms.

### Building locally

```bash
docker build -t gosync .
```

Images are scratch-based for minimal size. The published image on
`ghcr.io` is automatically built and tagged via GitHub Actions on every
push to `main` (tagged `latest`) and on semver tags (`v1.2.3`).

## Architecture

```
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
internal/proxy/    — Reverse proxy with BrowserSync features
internal/ws/       — WebSocket hub
internal/inject/   — HTML injection middleware
internal/clientjs/ — Embedded client JavaScript
internal/config/   — Configuration loading (YAML, env vars, defaults)
```

## How it works

1. **HTTP server** starts in either static file or reverse proxy mode
2. **Middleware** injects a `<script src="/__bs.js">` tag into HTML responses
3. **Client JS** connects to the server via WebSocket and listens for events
4. **Proxy features** rewrite requests and responses for seamless upstream integration: `Host` header rewriting, redirect `Location` rewriting, cookie domain stripping, and HTML link rewriting
5. **Scroll/form sync** broadcasts user interactions to all connected clients

## Security

- Same-origin WebSocket policy (no CSWSH)
- Proxy target restricted to http/https schemes (no SSRF)
- HTTP timeouts configured (Slowloris protection)
- WebSocket rate limiting (max 100 concurrent connections, configurable)
- Read/write deadlines and message size limits (configurable)
- TLS minimum v1.2, AES-GCM / ChaCha20-Poly1305 only
- Configurable proxy features: insecure TLS verification is opt-in only (`proxy_insecure: true`)

## License

MIT
