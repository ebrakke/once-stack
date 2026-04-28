# syntax=docker/dockerfile:1

# ------------------------------------------------------------------------------
# Standard ONCE-compatible Dockerfile for once-stack apps.
#
# Build with:
#   docker build --build-arg APP=notes -t once-stack/notes -f build/Dockerfile .
# ------------------------------------------------------------------------------

ARG APP

# ── Builder stage ─────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder
ARG APP

WORKDIR /build

RUN apk add --no-cache git ca-certificates

# Download deps first (layer cache friendly)
COPY go.mod ./
COPY go.sum* ./
RUN go mod download

# Copy entire source tree
COPY . .

# Build a static, stripped binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.appName=${APP}" \
    -o /bin/app \
    ./cmd/${APP}

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.21
ARG APP

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user (ONCE-friendly)
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -D -s /bin/sh appuser

# ONCE mounts persistent storage here
RUN mkdir -p /storage && chown -R appuser:appgroup /storage

COPY --from=builder /bin/app /usr/local/bin/app

USER appuser:appgroup

# ONCE requires port 80
EXPOSE 80

ENTRYPOINT ["/usr/local/bin/app"]
