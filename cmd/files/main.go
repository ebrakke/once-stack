// files is a simple file drop & retrieval service.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"once-stack/pkg/server"
	"once-stack/pkg/storage"
)

var appName = "files"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dir, err := storage.OpenDir(appName)
	if err != nil {
		logger.Error("failed to open storage", "err", err)
		os.Exit(1)
	}
	logger.Info("storage ready", "dir", dir)

	mux := http.NewServeMux()
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Files app — coming soon")
	}))

	srv := server.New(mux, "")
	if err := server.Run(srv); err != nil {
		logger.Error("server exited", "err", err)
		os.Exit(1)
	}
}
