// notes is a lightweight synchronized note-taking app.
package main

import (
	"log/slog"
	"os"

	"once-stack/pkg/notes"
	"once-stack/pkg/server"
	"once-stack/pkg/storage"
)

var appName = "notes"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dir, err := storage.OpenDir(appName)
	if err != nil {
		logger.Error("failed to open storage", "err", err)
		os.Exit(1)
	}
	logger.Info("storage ready", "dir", dir)

	store, err := notes.NewStore(dir)
	if err != nil {
		logger.Error("failed to create store", "err", err)
		os.Exit(1)
	}
	logger.Info("store ready", "dir", dir)

	app := notes.NewApp(store)
	mux := app.Routes()

	srv := server.New(mux, "")
	if err := server.Run(srv); err != nil {
		logger.Error("server exited", "err", err)
		os.Exit(1)
	}
}
