// Package storage provides a persistent data directory abstraction for ONCE apps.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns the effective storage root directory.
//
// Priority:
//  1. STORAGE_DIR environment variable (explicit override)
//  2. /storage if it already exists (ONCE/Docker mounted volume)
//  3. ./data as a local dev fallback
func Dir() string {
	if d := os.Getenv("STORAGE_DIR"); d != "" {
		return d
	}
	if fi, err := os.Stat("/storage"); err == nil && fi.IsDir() {
		return "/storage"
	}
	return "data"
}

// OpenDir ensures the app's named subdirectory exists under the storage root
// and returns its path. Use this to keep app data isolated when running multiple
// apps on the same host.
func OpenDir(name string) (string, error) {
	root := Dir()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create storage dir %q: %w", path, err)
	}
	return path, nil
}
