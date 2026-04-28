// Package storage provides a persistent data directory abstraction for ONCE apps.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns the ONCE storage directory (default /storage).
func Dir() string {
	if d := os.Getenv("STORAGE_DIR"); d != "" {
		return d
	}
	return "/storage"
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
