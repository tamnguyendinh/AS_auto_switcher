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

	"antigravity-switch/internal/engines/antigravity/model"
)

var (
	// ErrAccountNotFound is returned when an account JSON file does not exist.
	ErrAccountNotFound = errors.New("account not found in store")

	// ErrCorruptedStoreFile is returned when a store file cannot be decoded or is invalid.
	ErrCorruptedStoreFile = errors.New("corrupted account store file")
)

// StoredAccountEnvelope defines the JSON schema for accounts_store/<email>.json.
// Matches Python core/account_store.py schema exactly:
// { "email": "...", "status": "on", "saved_at": 1792056078.91, "token_data": { ... } }
type StoredAccountEnvelope struct {
	Email     string           `json:"email"`
	Tier      string           `json:"tier,omitempty"`
	Status    string           `json:"status,omitempty"` // "on" or "off", defaults to "on" if empty
	SavedAt   float64          `json:"saved_at"`
	TokenData *model.TokenData `json:"token_data"`
}

// AccountStore manages reading, writing, and listing individual account JSON files in accounts_store/.
type AccountStore struct {
	storeDir string
	mu       sync.RWMutex
}

// NewAccountStore initializes an AccountStore targeting the specified directory.
// If storeDir is empty, GetDefaultStoreDir() is used. The directory is created if missing.
func NewAccountStore(storeDir string) (*AccountStore, error) {
	if storeDir == "" {
		storeDir = GetDefaultStoreDir()
	}
	if storeDir == "" {
		return nil, errors.New("cannot determine account store directory; set ANTIGRAVITY_STORE_DIR")
	}

	absDir, err := filepath.Abs(storeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve account store directory: %w", err)
	}

	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create account store directory %s: %w", absDir, err)
	}

	return &AccountStore{
		storeDir: absDir,
	}, nil
}

// GetStoreDir returns the resolved absolute directory path of the store.
func (s *AccountStore) GetStoreDir() string {
	return s.storeDir
}

// SaveAccount atomically writes or updates the account JSON file for a given email.
// It uses an atomic temp file + fsync + rename pattern to guarantee that a sudden crash
// or power cut will never leave a half-written or corrupted file on disk.
// Invariant-Token-Update-Preservation: Before overwriting, it checks if the account already
// exists with status == "off", and strictly preserves "off" so refresh/capture does not reset it.
func (s *AccountStore) SaveAccount(email string, token *model.TokenData) error {
	if strings.TrimSpace(email) == "" {
		return ErrInvalidEmail
	}
	if token == nil || !token.IsValid() {
		return errors.New("cannot store invalid or empty token")
	}

	targetPath, err := SafeAccountPath(s.storeDir, email)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	normEmail := strings.ToLower(strings.TrimSpace(email))

	// Invariant-Token-Update-Preservation: preserve "off" status and tier if already present on disk
	status := "on"
	var existingTier string
	if existingData, readErr := os.ReadFile(targetPath); readErr == nil {
		var existingEnv StoredAccountEnvelope
		if jsonErr := json.Unmarshal(existingData, &existingEnv); jsonErr == nil {
			if strings.EqualFold(strings.TrimSpace(existingEnv.Status), "off") {
				status = "off"
			}
			existingTier = existingEnv.Tier
		}
	}

	envelope := StoredAccountEnvelope{
		Email:     normEmail,
		Tier:      existingTier,
		Status:    status,
		SavedAt:   float64(time.Now().UnixNano()) / 1e9,
		TokenData: token,
	}

	// Create temporary file in the same directory (crucial for atomic os.Rename across filesystem boundaries)
	tmpFile, err := os.CreateTemp(s.storeDir, "acc_tmp_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for atomic write: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup in case of write failure
	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(envelope); err != nil {
		return fmt.Errorf("failed to encode account envelope: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file to disk: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename to final target path
	if err := os.Rename(tmpPath, targetPath); err != nil {
		// On Windows, if target exists, os.Rename will overwrite in Go 1.5+
		// but if it fails, attempt explicit remove then rename
		_ = os.Remove(targetPath)
		if retryErr := os.Rename(tmpPath, targetPath); retryErr != nil {
			return fmt.Errorf("failed to rename temp file to target %s: %w", targetPath, retryErr)
		}
	}

	success = true
	return nil
}

// SetAccountStatus atomically updates only the status field ("on" or "off") of an existing account file.
func (s *AccountStore) SetAccountStatus(email string, status string) error {
	if strings.TrimSpace(email) == "" {
		return ErrInvalidEmail
	}
	normStatus := strings.ToLower(strings.TrimSpace(status))
	if normStatus != "off" {
		normStatus = "on"
	}
	targetPath, err := SafeAccountPath(s.storeDir, email)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("failed to read account file: %w", err)
	}

	var env StoredAccountEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return ErrCorruptedStoreFile
	}

	env.Status = normStatus

	// Ghi tệp an toàn bằng atomic temp file + os.Rename
	tmpFile, err := os.CreateTemp(s.storeDir, "acc_tmp_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for atomic write: %w", err)
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
	if err := encoder.Encode(env); err != nil {
		return fmt.Errorf("failed to encode account envelope: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file to disk: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(targetPath)
		if retryErr := os.Rename(tmpPath, targetPath); retryErr != nil {
			return fmt.Errorf("failed to rename temp file to target %s: %w", targetPath, retryErr)
		}
	}

	success = true
	return nil
}

// UpdateAccountTier atomically updates the tier field of an existing account file.
func (s *AccountStore) UpdateAccountTier(email string, tier string) error {
	if strings.TrimSpace(email) == "" {
		return ErrInvalidEmail
	}
	targetPath, err := SafeAccountPath(s.storeDir, email)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("failed to read account file: %w", err)
	}

	var env StoredAccountEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return ErrCorruptedStoreFile
	}

	env.Tier = tier

	tmpFile, err := os.CreateTemp(s.storeDir, "acc_tmp_*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for atomic write: %w", err)
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
	if err := encoder.Encode(env); err != nil {
		return fmt.Errorf("failed to encode account envelope: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file to disk: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(targetPath)
		if retryErr := os.Rename(tmpPath, targetPath); retryErr != nil {
			return fmt.Errorf("failed to rename temp file to target %s: %w", targetPath, retryErr)
		}
	}

	success = true
	return nil
}

// GetAccountStatus returns the current status ("on" or "off") of an account in the store.
// If the account does not exist, it returns ErrAccountNotFound.
// If the status field is missing or empty (legacy accounts), it returns "on", nil for backward compatibility.
func (s *AccountStore) GetAccountStatus(email string) (string, error) {
	if strings.TrimSpace(email) == "" {
		return "on", ErrInvalidEmail
	}
	targetPath, err := SafeAccountPath(s.storeDir, email)
	if err != nil {
		return "on", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "on", ErrAccountNotFound
		}
		return "on", fmt.Errorf("failed to read account file: %w", err)
	}

	var env StoredAccountEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return "on", ErrCorruptedStoreFile
	}

	normStatus := strings.ToLower(strings.TrimSpace(env.Status))
	if normStatus == "off" {
		return "off", nil
	}
	return "on", nil
}

// GetAccount reads and parses the TokenData for an email from its JSON file.
func (s *AccountStore) GetAccount(email string) (*model.TokenData, error) {
	if strings.TrimSpace(email) == "" {
		return nil, ErrInvalidEmail
	}

	targetPath, err := SafeAccountPath(s.storeDir, email)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to read account file: %w", err)
	}

	var envelope StoredAccountEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, ErrCorruptedStoreFile
	}

	if envelope.TokenData == nil || !envelope.TokenData.IsValid() {
		return nil, ErrCorruptedStoreFile
	}

	return envelope.TokenData, nil
}

// GetAccountEnvelope reads and parses the full StoredAccountEnvelope for an email from its JSON file.
func (s *AccountStore) GetAccountEnvelope(email string) (*StoredAccountEnvelope, error) {
	if strings.TrimSpace(email) == "" {
		return nil, ErrInvalidEmail
	}

	targetPath, err := SafeAccountPath(s.storeDir, email)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to read account file: %w", err)
	}

	var envelope StoredAccountEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, ErrCorruptedStoreFile
	}

	return &envelope, nil
}

// HasAccount checks whether a valid, parseable JSON account file exists for the email.
func (s *AccountStore) HasAccount(email string) bool {
	token, err := s.GetAccount(email)
	return err == nil && token != nil && token.IsValid()
}

// DeleteAccount removes the JSON file for an email.
func (s *AccountStore) DeleteAccount(email string) error {
	if strings.TrimSpace(email) == "" {
		return ErrInvalidEmail
	}

	targetPath, err := SafeAccountPath(s.storeDir, email)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(targetPath); err != nil {
		if os.IsNotExist(err) {
			return nil // idempotent deletion
		}
		return fmt.Errorf("failed to delete account file %s: %w", targetPath, err)
	}

	return nil
}

// ListAccounts returns a sorted list of all valid emails currently stored in accounts_store/.
func (s *AccountStore) ListAccounts() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.storeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read store directory: %w", err)
	}

	emailSet := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, "acc_tmp_") || strings.HasSuffix(name, ".tmp") {
			continue
		}

		filePath := filepath.Join(s.storeDir, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var envelope StoredAccountEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			continue
		}

		if envelope.Email != "" && envelope.TokenData != nil && envelope.TokenData.IsValid() {
			emailSet[strings.ToLower(envelope.Email)] = true
		}
	}

	result := make([]string, 0, len(emailSet))
	for em := range emailSet {
		result = append(result, em)
	}
	sort.Strings(result)

	return result, nil
}
