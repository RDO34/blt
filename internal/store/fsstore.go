package store

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rdo34/blt/internal/model"
)

// FSStore implements Store using per-day JSONL files under a data root.
type FSStore struct {
	root string
}

// NewFSStore creates a store rooted at dir, creating it if needed.
func NewFSStore(dir string) (*FSStore, error) {
	if dir == "" {
		return nil, errors.New("empty dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FSStore{root: dir}, nil
}

// NewDefaultFSStore resolves the default data dir and returns a store.
func NewDefaultFSStore() (*FSStore, error) {
	dir, err := ResolveDataDir()
	if err != nil {
		return nil, err
	}
	return NewFSStore(dir)
}

func (s *FSStore) dayPath(date time.Time) string {
	y, m, d := date.Date()
	return filepath.Join(s.root, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", m), fmt.Sprintf("%02d.jsonl", d))
}

func (s *FSStore) ensureDayDir(date time.Time) error {
	y, m, _ := date.Date()
	return os.MkdirAll(filepath.Join(s.root, fmt.Sprintf("%04d", y), fmt.Sprintf("%02d", m)), 0o755)
}

// LoadDay reads all bullets for the given date from the JSONL file.
func (s *FSStore) LoadDay(date time.Time) ([]model.Bullet, error) {
	path := s.dayPath(date)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Bullet{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []model.Bullet
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			var b model.Bullet
			if jerr := json.Unmarshal(bytesTrimRightNewline(line), &b); jerr == nil {
				out = append(out, b)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// SaveDay atomically writes all bullets for the given day.
func (s *FSStore) SaveDay(date time.Time, items []model.Bullet) error {
	if err := s.ensureDayDir(date); err != nil {
		return err
	}
	path := s.dayPath(date)
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "day-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	bw := bufio.NewWriter(tmp)
	enc := json.NewEncoder(bw)
	for _, it := range items {
		if it.ID == "" {
			it.ID = generateID()
		}
		if it.CreatedAt.IsZero() {
			it.CreatedAt = time.Now()
		}
		if err := enc.Encode(&it); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return err
		}
	}
	if err := bw.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// Append adds a single bullet to the day file as one JSON line.
func (s *FSStore) Append(date time.Time, b model.Bullet) error {
	if err := s.ensureDayDir(date); err != nil {
		return err
	}
	if b.ID == "" {
		b.ID = generateID()
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	path := s.dayPath(date)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(&b)
}

// Update replaces a bullet with matching ID for that day; no-op if not found.
func (s *FSStore) Update(date time.Time, b model.Bullet) error {
	items, err := s.LoadDay(date)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == b.ID {
			items[i] = b
			break
		}
	}
	return s.SaveDay(date, items)
}

// Delete removes a bullet by ID for that day; no-op if not found.
func (s *FSStore) Delete(date time.Time, id string) error {
	items, err := s.LoadDay(date)
	if err != nil {
		return err
	}
	filtered := items[:0]
	for _, it := range items {
		if it.ID != id {
			filtered = append(filtered, it)
		}
	}
	return s.SaveDay(date, filtered)
}

func generateID() string {
	// 8 random bytes hex-encoded with time prefix for rough ordering.
	var rb [8]byte
	_, _ = rand.Read(rb[:])
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(rb[:]))
}

func bytesTrimRightNewline(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	if b[len(b)-1] == '\n' || b[len(b)-1] == '\r' {
		return b[:len(b)-1]
	}
	return b
}

var _ Store = (*FSStore)(nil)
