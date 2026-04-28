# ONCE Stack

A Go monorepo of simple, self-hostable web applications packaged as [ONCE](https://github.com/basecamp/once)-compatible Docker images.

The goal is to provide an easily deployable stack of daily-use apps that run anywhere Docker runs — a VPS, Raspberry Pi, homelab, or laptop.

## Apps

| App | Path | Description |
|-----|------|-------------|
| **Notes** | [`cmd/notes`](./cmd/notes) | Lightweight synchronized notes across devices |
| **Files** | [`cmd/files`](./cmd/files) | Simple file drop & retrieval (like a minimal Syncthing) |
| **Blog** | [`cmd/blog`](./cmd/blog) | Markdown-based blog renderer |

## Quick Start

### Prerequisites

- [Go](https://golang.org/dl/) 1.24+
- [Docker](https://docs.docker.com/get-docker/)
- `make` (optional, but recommended)

### Local Development

```bash
# Run an app locally (binds :8080, stores data in ./data/<app>)
just run notes

# Override storage location or port
STORAGE_DIR=/tmp/notes-storage PORT=3000 just run notes
```

### Build Docker Images

```bash
# Build an app image
docker build --build-arg APP=notes -t once-stack/notes:latest -f build/Dockerfile .

# Build all app images
just build-all
```

### Run with ONCE

```bash
# Install once (if you haven't already)
curl https://get.once.com | sh

# In the ONCE TUI, choose "Custom image" and enter:
#   once-stack/notes:latest
```

---

## Project Structure

This repo follows Go monorepo best practices:

```
once-stack/
├── cmd/                    # Application entrypoints (one per app)
│   ├── notes/
│   │   ├── main.go
│   │   └── Dockerfile      # (optional per-app override)
│   ├── files/
│   │   └── main.go
│   └── blog/
│       └── main.go
├── pkg/                    # Shared libraries
│   ├── server/             # HTTP server helpers, middleware
│   ├── health/             # ONCE-compatible /up health checks
│   ├── storage/            # Persistent storage abstractions
│   └── web/                # Templates, static assets, handlers
├── build/
│   ├── Dockerfile            # Standard ONCE-compatible Dockerfile
│   └── docker-entrypoint.sh  # Container entrypoint
├── go.mod                  # Root module (single-module monorepo)
├── go.sum
└── README.md
```

### Why a Single Go Module?

We use a **single root `go.mod`** rather than a multi-module workspace (`go.work`) because:

- All apps are small, tightly related, and share common packages
- No need for independent versioning or release cycles
- Simpler dependency management (`go get` / `go mod tidy` at root)
- Easier CI/CD with a single build pipeline
- Cross-package refactoring is painless

If an app later needs to be extracted or versioned independently, it can be promoted to its own module with minimal effort.

---

## ONCE Compatibility

Every app in this repo conforms to the [ONCE spec](https://github.com/basecamp/once) for self-hostable Docker images.

### Requirements

| Requirement | How we satisfy it |
|-------------|-----------------|
| **Docker container** | Standard multi-stage Dockerfile in `build/Dockerfile` |
| **HTTP on port 80** | Apps bind `:80` in Docker / `:8080` locally (override with `PORT`) |
| **Health check at `/up`** | [`pkg/health`](./pkg/health) provides a standard handler; every app mounts it |
| **Persistent data in `/storage`** | [`pkg/storage`](./pkg/storage) uses `/storage` in Docker, `./data` locally |
| **Non-root user** | Dockerfile creates and runs as `appuser` (UID 1000) |

### Optional Enhancements

| Hook | Support |
|------|---------|
| `/hooks/pre-backup` | Apps using SQLite can add a hook to create a consistent snapshot |
| `/hooks/post-restore` | Cleanup hook after restoring from backup |

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `80` in Docker / `8080` locally |
| `STORAGE_DIR` | Persistent storage path | `/storage` in Docker / `./data` locally |
| `SECRET_KEY_BASE` | Cryptographic secret (injected by ONCE) | auto-generated fallback |
| `DISABLE_SSL` | Set to `true` when running without TLS | `false` |

---

## Standard Dockerfile

All apps use the same [`build/Dockerfile`](./build/Dockerfile) via a `--build-arg APP=<name>` parameter. This keeps builds consistent, cache-friendly, and easy to maintain.

```dockerfile
# syntax=docker/dockerfile:1

ARG APP

# ── Builder stage ───────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder
ARG APP
WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.appName=${APP}" \
    -o /bin/app \
    ./cmd/${APP}

# ── Runtime stage ───────────────────────────────────────────────────────────
FROM alpine:3.21
ARG APP

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D -s /bin/sh appuser

RUN mkdir -p /storage && chown -R appuser:appgroup /storage

COPY --from=builder /bin/app /usr/local/bin/app

USER appuser:appgroup

EXPOSE 80

ENTRYPOINT ["/usr/local/bin/app"]
```

### Build Commands

```bash
# Notes
docker build --build-arg APP=notes -t once-stack/notes:latest -f build/Dockerfile .

# Files
docker build --build-arg APP=files -t once-stack/files:latest -f build/Dockerfile .

# Blog
docker build --build-arg APP=blog -t once-stack/blog:latest -f build/Dockerfile .
```

---

## Adding a New App

1. **Create the app directory:**
   ```bash
   mkdir cmd/myapp
   touch cmd/myapp/main.go
   ```

2. **Write `main.go`:** Use `pkg/server` and `pkg/health` for the boilerplate.

3. **Mount `/up`:** Ensure the health endpoint is registered.

4. **Use `/storage`:** Call `storage.OpenDir()` for persistent data.

5. **Build & run:**
   ```bash
   just build myapp
   just run myapp
   ```

---

## Development Guidelines

- **Single module:** Keep `go.mod` at the root. Import shared packages with `once-stack/pkg/<name>`.
- **No `vendor/`:** Rely on Go modules and the module proxy.
- **Static binaries:** All Docker builds use `CGO_ENABLED=0` for portability.
- **Minimal images:** Alpine-based runtime stage, stripped binaries (`-ldflags="-w -s"`).
- **Graceful shutdown:** HTTP servers listen for `SIGTERM` / `SIGINT` and drain connections.
- **Structured logging:** Use `log/slog` (stdlib) with JSON or text output.

---

## License

MIT
