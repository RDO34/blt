package store

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ResolveDataDir returns the directory used for BLT data.
// Order: BLT_DATA_DIR env override, then OS-specific default.
func ResolveDataDir() (string, error) {
	if custom := os.Getenv("BLT_DATA_DIR"); custom != "" {
		return custom, nil
	}

	switch runtime.GOOS {
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "blt"), nil
		}
		return "", errors.New("APPDATA not set")
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, "Library", "Application Support", "blt"), nil
		}
		return "", errors.New("home directory not found")
	default: // linux and others
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, ".local", "share", "blt"), nil
		}
		return "", errors.New("home directory not found")
	}
}
