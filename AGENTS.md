# gosync — AGENTS.md

## Build

```bash
export PATH="/tmp/gosync-go/go/bin:$PATH"
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./cmd/gosync
```

- **CGO must be disabled** — system `gcc` has a broken `-m64` flag
- Go binary is at `/tmp/gosync-go/go/bin/go`, not system PATH
- Always set `GOTOOLCHAIN=local` to avoid Go toolchain auto-download failures

## Test

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go test -cover -timeout 60s ./...
```

- **cmd/gosync tests**: calling `run()` with a valid config blocks forever on `ListenAndServe`. Only test error-returning paths directly. For blocking paths, use a goroutine + select with timeout.
- **ws tests**: use `hub.BroadcastFn` to intercept broadcasts (Broadcast is a method, not a field — cannot be replaced directly).
- **Port ranges**: server tests use `309x`, cmd tests use `459x` — do not overlap (packages run in parallel).
- **Debounce tests**: `TestHandleEventWithDebounce` takes ~300ms. Other watcher tests may have sub-second sleeps for fsnotify to propagate.

## Architecture

- `cmd/gosync/main.go` entrypoint. `run()` does setup; `main()` calls `run()` and `log.Fatalf` on error.
- `internal/server/` wires hub, middleware, proxy/file-server into an `http.Server` with timeouts.
- `internal/inject/` wraps `http.ResponseWriter` to buffer HTML, injects `<script src="/__bs.js">` before `</body>`.
- `internal/ws/` gorilla/websocket hub. `HandleWS` upgrades, runs read/write pumps with pings/pongs. Rate limited to 100 conns.
- `internal/watch/` fsnotify recursive watcher. 100ms debounce. `.css` → `"css"` event, everything else → `"reload"`.
- `internal/proxy/` `httputil.ReverseProxy`. Rejects non-http(s) schemes (SSRF protection). Sets `X-Forwarded-*` headers.
- `internal/clientjs/` Go embed of `client.js` — served at `/__bs.js`.

## Key conventions

- Three external deps: `gorilla/websocket`, `fsnotify/fsnotify`, and `gopkg.in/yaml.v3`.
- HTML injection middleware strips `Content-Length` header (body size changes after injection).
- No Makefile, no CI, no pre-commit hooks, no codegen, no database, no migrations.
- All tests are pure unit tests — no external services needed.

## Config priority (highest to lowest)

1. CLI flags
2. Environment variables
3. Config file (`.gosync.yaml`, path overridable via `GOSYNC_CONFIG`)
4. Built-in defaults

## Key env vars

- `GOSYNC_CONFIG` — path to YAML config file (default: `.gosync.yaml`)
- `GOSYNC_PORT`, `GOSYNC_DIR`, `GOSYNC_PROXY`, `GOSYNC_WATCH`, `GOSYNC_TLS_CERT`, `GOSYNC_TLS_KEY`
- `GOSYNC_PROXY_TIMEOUT_SECONDS` — proxy response header timeout
- `GOSYNC_HUB_OPTIONS` — JSON object for hub options override
- `GOSYNC_RATE_LIMIT_CONNS`, `GOSYNC_MAX_MSG_SIZE_BYTES`, `GOSYNC_PING_PONG_INTERVAL_SECONDS`, `GOSYNC_PONG_WAIT_SECONDS`, `GOSYNC_WRITE_WAIT_SECONDS` — individual hub option overrides (take precedence over `GOSYNC_HUB_OPTIONS`)

## Security hardening (already applied)

- WebSocket `CheckOrigin: nil` — same-origin default (was `return true`).
- Proxy scheme validation — only `http`/`https` allowed.
- WS rate limit (100 max), message size limit (4KB), read/write deadlines, ping/pong.
- HTTP server timeouts: Read 15s, Write 15s, Idle 60s.
- TLS: minimum v1.2, AES-GCM / ChaCha20-Poly1305 only.
