# AGENTS.md — Working with the ONCE Stack

This file provides context and conventions for AI agents (and future developers) contributing to the `once-stack` Go monorepo.

---

## Project Purpose

A collection of simple, self-hostable web applications packaged as [ONCE](https://github.com/basecamp/once)-compatible Docker images. Each app is a small, independent Go binary. The goal is easy deployment on any machine that runs Docker — VPS, Raspberry Pi, laptop, etc.

---

## Repository Layout

```
once-stack/
├── cmd/              # One directory per application
│   ├── notes/
│   ├── files/
│   └── blog/
├── pkg/              # Shared libraries used by all apps
│   ├── health/       # ONCE /up health check handler
│   ├── server/       # Graceful-shutdown HTTP server wiring
│   └── storage/      # Persistent data directory abstraction
├── build/
│   └── Dockerfile    # Single standard Dockerfile for ALL apps
├── justfile          # Development commands (replaces Make)
├── go.mod            # Single root Go module
└── README.md         # User-facing documentation
```

**Key rule:** `cmd/` contains app entrypoints only. `pkg/` contains reusable code. Do not duplicate logic across `cmd/` directories.

---

## ONCE Compatibility

Every app must satisfy these requirements to work with the ONCE installer.

| Requirement | How we do it |
|-------------|--------------|
| Serves HTTP on **port 80** | `pkg/server` binds `:80` in Docker (when `/storage` exists), `:8080` locally; override with `PORT` env var |
| **Health check** at `/up` returning success | `pkg/server.New()` automatically mounts `pkg/health.Handler()` at `GET /up` |
| Keeps persistent data in **`/storage`** | `pkg/storage.OpenDir("appname")` returns `/storage/appname` in Docker, `./data/appname` locally; override with `STORAGE_DIR` |
| Packaged as a **Docker container** | `build/Dockerfile` — multi-stage, static binary, Alpine runtime |
| **Non-root user** | Dockerfile creates `appuser` (UID 1000) |

### Docker Build

All apps share the same Dockerfile. Build with `--build-arg APP=<name>`:

```bash
just build notes
# or directly:
docker build --build-arg APP=notes -t once-stack/notes:latest -f build/Dockerfile .
```

The builder stage compiles a static Go binary (`CGO_ENABLED=0`). The runtime stage is Alpine Linux with `ca-certificates` and `tzdata`.

---

## Go Patterns & Conventions

### Single Module Monorepo

One root `go.mod`. Import shared packages with:

```go
import "once-stack/pkg/server"
```

Do not create sub-modules or `go.work` files unless a package genuinely needs independent versioning.

### App Skeleton

Every `cmd/<app>/main.go` should follow this minimal structure:

```go
package main

import (
    "log/slog"
    "net/http"
    "os"

    "once-stack/pkg/server"
    "once-stack/pkg/storage"
)

var appName = "<app>"

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

    dir, err := storage.OpenDir(appName)
    if err != nil {
        logger.Error("storage failed", "err", err)
        os.Exit(1)
    }
    logger.Info("storage ready", "dir", dir)

    mux := http.NewServeMux()
    mux.Handle("GET /", handleRoot())
    // DO NOT register /up here — pkg/server does it automatically

    srv := server.New(mux, "")
    if err := server.Run(srv); err != nil {
        logger.Error("server exited", "err", err)
        os.Exit(1)
    }
}
```

**Do not** manually register `GET /up` on your own `*http.ServeMux`. `pkg/server.New()` already does this. Registering it twice will panic at startup.

### Logging

Use the standard library `log/slog`. Prefer structured logging (`logger.Info`, `logger.Error`) over `fmt.Println`.

### Storage

Always call `storage.OpenDir(appName)` before doing any filesystem I/O. This ensures the directory exists under `/storage/<app>` in Docker or `./data/<app>` locally (override with `STORAGE_DIR`).

### Graceful Shutdown

Use `server.Run(srv)` from `pkg/server`. It traps `SIGINT`/`SIGTERM`, drains active connections with a 15-second timeout, and shuts down cleanly.

---

## Justfile Commands

Install `just` from https://github.com/casey/just. Common tasks:

| Command | Action |
|---------|--------|
| `just run notes` | Run an app locally with `go run` |
| `just run-with-storage notes /tmp/data` | Run with a custom storage dir |
| `just build notes` | Build the Docker image for one app |
| `just build-all` | Build all app images |
| `just test` | Run `go test ./...` |
| `just quality` | Run `fmt`, `vet`, and `test` in sequence |
| `just tidy` | Run `go mod tidy` |
| `just clean` | Remove locally built Docker images |

---

## Adding a New App

1. `mkdir cmd/<name>`
2. Write `cmd/<name>/main.go` following the skeleton above.
3. Ensure `storage.OpenDir("<name>")` is called.
4. Do not add a separate Dockerfile unless the app has unique system dependencies.
5. Build: `just build <name>`

---

## Testing

- Unit tests live alongside the code they test (`foo.go` + `foo_test.go`).
- Prefer table-driven tests.
- Avoid importing `cmd/` packages in tests — test `pkg/` directly.
- Use `t.TempDir()` for filesystem tests instead of hardcoded paths.

---

## Common Pitfalls

1. **Double-registering `/up`** — `pkg/server.New()` mounts it. Don't do it again in your app.
2. **Forgetting `storage.OpenDir()`** — Always open/create your app's storage directory before use.
3. **CGO in Docker builds** — The Dockerfile uses `CGO_ENABLED=0`. Do not add C dependencies without updating the build.
4. **Root module imports** — Use `once-stack/pkg/...`, not relative paths like `../../pkg/...`.
