package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultQuotaCachePath is the canonical disk cache file for quota telemetry.
	DefaultQuotaCachePath = "accounts_store/.quota_cache.json"
)

var (
	// ErrNilQuotaData is returned when attempting to cache nil quota data.
	ErrNilQuotaData = errors.New("quota data cannot be nil")
)

// QuotaCacheSnapshot represents the on-disk serialization envelope for quota telemetry.
type QuotaCacheSnapshot struct {
	SavedAt      time.Time                    `json:"saved_at"`
	AccountCount int                          `json:"account_count"`
	Matrix       map[string]*AccountQuotaData `json:"matrix"`
}

// QuotaCache manages in-memory and on-disk cached quota data across all accounts.
type QuotaCache struct {
	filePath string
	mu       sync.RWMutex
	matrix   map[string]*AccountQuotaData
}

// NewQuotaCache initializes a new two-tier QuotaCache targeting filePath.
// If filePath is empty, DefaultQuotaCachePath is used.
func NewQuotaCache(filePath string) *QuotaCache {
	if strings.TrimSpace(filePath) == "" {
		filePath = DefaultQuotaCachePath
	}
	return &QuotaCache{
		filePath: filePath,
		matrix:   make(map[string]*AccountQuotaData),
	}
}

// FilePath returns the configured cache file path.
func (c *QuotaCache) FilePath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.filePath
}

// SetFilePath updates the target cache file path.
func (c *QuotaCache) SetFilePath(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.TrimSpace(path) != "" {
		c.filePath = path
	}
}

// Get retrieves cached quota data for the specified account email.
// Returns a deep copy to prevent data races from concurrent caller mutation.
func (c *QuotaCache) Get(email string) (*AccountQuotaData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, exists := c.matrix[strings.ToLower(strings.TrimSpace(email))]
	if !exists || data == nil {
		return nil, false
	}

	clone := *data
	return &clone, true
}

// GetAll returns a copy of the entire in-memory quota matrix.
func (c *QuotaCache) GetAll() map[string]*AccountQuotaData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*AccountQuotaData, len(c.matrix))
	for k, v := range c.matrix {
		if v != nil {
			clone := *v
			result[k] = &clone
		}
	}
	return result
}

// Update saves or replaces quota data for an account in memory.
func (c *QuotaCache) Update(data *AccountQuotaData) {
	if data == nil || strings.TrimSpace(data.Email) == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	clone := *data
	c.matrix[strings.ToLower(strings.TrimSpace(data.Email))] = &clone
}

// UpdateBatch updates multiple account quotas in memory under a single write lock.
func (c *QuotaCache) UpdateBatch(items []*AccountQuotaData) {
	if len(items) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, data := range items {
		if data != nil && strings.TrimSpace(data.Email) != "" {
			clone := *data
			c.matrix[strings.ToLower(strings.TrimSpace(data.Email))] = &clone
		}
	}
}

// SaveToDisk atomically writes the current in-memory quota matrix to disk.
// If customPath is provided, it writes to that path instead of c.filePath.
func (c *QuotaCache) SaveToDisk(customPath ...string) error {
	targetPath := c.filePath
	if len(customPath) > 0 && strings.TrimSpace(customPath[0]) != "" {
		targetPath = customPath[0]
	}

	// Capture memory snapshot under read lock
	c.mu.RLock()
	snapshot := QuotaCacheSnapshot{
		SavedAt:      time.Now().UTC(),
		AccountCount: len(c.matrix),
		Matrix:       make(map[string]*AccountQuotaData, len(c.matrix)),
	}
	for k, v := range c.matrix {
		if v != nil {
			clone := *v
			snapshot.Matrix[k] = &clone
		}
	}
	c.mu.RUnlock()

	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", targetDir, err)
	}

	// Atomic write using temp file + flush + rename
	tmpFile, err := os.CreateTemp(targetDir, ".quota_cache_tmp_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary cache file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		return fmt.Errorf("failed to encode quota snapshot: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to flush temp cache file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp cache file: %w", err)
	}

	// Atomic rename to final path (with remove fallback on Windows)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(targetPath)
		if retryErr := os.Rename(tmpPath, targetPath); retryErr != nil {
			return fmt.Errorf("failed to replace quota cache file %s: %w", targetPath, retryErr)
		}
	}

	success = true
	return nil
}

// LoadFromDisk reads the quota matrix snapshot from disk into memory.
// If the file does not exist, it returns nil without error (cold start).
func (c *QuotaCache) LoadFromDisk(customPath ...string) error {
	targetPath := c.filePath
	if len(customPath) > 0 && strings.TrimSpace(customPath[0]) != "" {
		targetPath = customPath[0]
	}

	dataBytes, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Missing cache is not fatal
		}
		return fmt.Errorf("failed to read quota cache from %s: %w", targetPath, err)
	}

	if len(strings.TrimSpace(string(dataBytes))) == 0 {
		return nil
	}

	// First try decoding with QuotaCacheSnapshot envelope
	var snapshot QuotaCacheSnapshot
	if err := json.Unmarshal(dataBytes, &snapshot); err == nil && snapshot.Matrix != nil {
		c.mu.Lock()
		for k, v := range snapshot.Matrix {
			if v != nil {
				c.matrix[strings.ToLower(k)] = v
			}
		}
		c.mu.Unlock()
		return nil
	}

	// Fallback to direct flat map: map[string]*AccountQuotaData
	var flatMap map[string]*AccountQuotaData
	if err := json.Unmarshal(dataBytes, &flatMap); err != nil {
		return fmt.Errorf("failed to unmarshal quota cache json: %w", err)
	}

	c.mu.Lock()
	for k, v := range flatMap {
		if v != nil {
			c.matrix[strings.ToLower(k)] = v
		}
	}
	c.mu.Unlock()

	return nil
}
