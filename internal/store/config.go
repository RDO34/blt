package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rdo34/blt/internal/model"
)

type Preferences struct {
	Period      model.Period       `json:"period"`
	TextFilter  string             `json:"text_filter"`
	Types       []model.BulletType `json:"types"`
	Tags        []string           `json:"tags"`
	LastDate    string             `json:"last_date"` // YYYY-MM-DD
	CenterWidth int                `json:"center_width,omitempty"`
}

func prefsPath() (string, error) {
	dir, err := ResolveDataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "prefs.json"), nil
}

func LoadPreferences() (Preferences, error) {
	var p Preferences
	path, err := prefsPath()
	if err != nil {
		return p, err
	}
	f, err := os.Open(path)
	if err != nil {
		return p, err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	err = dec.Decode(&p)
	return p, err
}

func SavePreferences(p Preferences) error {
	path, err := prefsPath()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "prefs-*.tmp")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&p); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
