// Package server provides common HTTP server wiring for ONCE apps.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"once-stack/pkg/health"
)

// DefaultPort is the ONCE-required HTTP port.
const DefaultPort = "80"

// New creates a minimal *http.Server with the ONCE /up health check already mounted.
// Pass additional handlers to register on the default mux.
func New(mux *http.ServeMux, port string) *http.Server {
	if port == "" {
		port = DefaultPort
		if p := os.Getenv("PORT"); p != "" {
			port = p
		}
	}

	if mux == nil {
		mux = http.NewServeMux()
	}

	// ONCE health check
	mux.Handle("GET /up", health.Handler())
	mux.Handle("GET /health", health.Readiness())

	return &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// Run starts the server and blocks until a shutdown signal is received.
func Run(srv *http.Server) error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("server listening on %s (pid %d)\n", srv.Addr, os.Getpid())
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
