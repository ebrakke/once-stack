// Package health provides ONCE-compatible health check handlers.
package health

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
)

// Response is the JSON shape returned by the /up endpoint.
type Response struct {
	Status string `json:"status"`
}

// Handler returns an http.Handler that responds 200 OK at /up.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Response{Status: "ok"})
	})
}

// Readiness returns a handler that reports whether the app is ready.
// It does a lightweight liveness check; swap in real checks as needed.
func Readiness() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		info, ok := debug.ReadBuildInfo()
		version := "unknown"
		if ok {
			version = info.Main.Version
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"version": version,
		})
	})
}
