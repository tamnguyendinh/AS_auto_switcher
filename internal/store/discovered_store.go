package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DiscoveredAccountEnvelope is an identity record, never a credential.
type DiscoveredAccountEnvelope struct {
	Email     string  `json:"email"`
	ScannedAt float64 `json:"scanned_at"`
}

type DiscoveredStore struct {
	dir string
	mu  sync.RWMutex
}

// NewDiscoveredStore keeps the selected path even when it is temporarily
// unavailable. Callers may display live scan results while reporting that error.
func NewDiscoveredStore(dir string) (*DiscoveredStore, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("discovered store directory is empty")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	s := &DiscoveredStore{dir: abs}
	return s, os.MkdirAll(abs, 0755)
}

func (s *DiscoveredStore) SaveDiscovered(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	path, err := SafeAccountPath(s.dir, email)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, ".discovered-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	entry := DiscoveredAccountEnvelope{Email: email, ScannedAt: float64(time.Now().UnixNano()) / 1e9}
	if err := json.NewEncoder(tmp).Encode(entry); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Never delete the previous record if replacement fails (e.g. a locked file).
	return os.Rename(tmp.Name(), path)
}

// ListDiscovered returns all readable records, alongside any partial-read error.
// Reading never updates scanned_at and a missing scan never prunes this catalog.
func (s *DiscoveredStore) ListDiscovered() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var failures []error
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		var record DiscoveredAccountEnvelope
		if err == nil {
			err = json.Unmarshal(data, &record)
		}
		email := strings.ToLower(strings.TrimSpace(record.Email))
		if err != nil || email == "" || SanitizeFilename(email) != entry.Name() || record.ScannedAt <= 0 {
			failures = append(failures, fmt.Errorf("không đọc được bản ghi danh bạ %s", entry.Name()))
			continue
		}
		seen[email] = true
	}
	emails := make([]string, 0, len(seen))
	for email := range seen {
		emails = append(emails, email)
	}
	sort.Strings(emails)
	return emails, errors.Join(failures...)
}
