# Justfile for once-stack
# Install `just`: https://github.com/casey/just

set dotenv-load

default:
    @just --list

# ── Development ──────────────────────────────────────────────────────────────

# Run an app locally (usage: just run notes)
run app:
    go run ./cmd/{{app}}

# Run an app with a custom storage dir
run-with-storage app dir:
    STORAGE_DIR={{dir}} just run {{app}}

# ── Testing / Quality ────────────────────────────────────────────────────────

# Run all tests
test:
    go test ./...

# Format all code
fmt:
    go fmt ./...

# Run go vet
vet:
    go vet ./...

# Tidy modules
tidy:
    go mod tidy

# Build shared Tailwind CSS assets
# Run this whenever pkg/ui/styles/once.css or templates using Tailwind classes change.
css-build:
    mkdir -p pkg/ui/static
    bun run css:build

# Verify generated CSS is up to date
css-check:
    #!/usr/bin/env bash
    just css-build
    git diff --exit-code pkg/ui/static/once.css

# Run the full quality pipeline
quality: css-check fmt vet test

# ── Docker ───────────────────────────────────────────────────────────────────

# Build a single app image (usage: just build notes)
build app:
    docker build \
        --build-arg APP={{app}} \
        -t once-stack/{{app}}:latest \
        -f build/Dockerfile \
        .

# Build all app images
build-all:
    just build notes
    just build files
    just build blog

# ── Cleanup ──────────────────────────────────────────────────────────────────

clean:
    #!/usr/bin/env bash
    docker images --filter 'reference=once-stack/*' --format '{{{{.ID}}' | xargs -r docker rmi -f 2>/dev/null || true
