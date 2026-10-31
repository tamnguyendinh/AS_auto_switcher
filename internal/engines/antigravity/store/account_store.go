package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"as-auto-switcher/internal/core/credentialio"
	"as-auto-switcher/internal/engines/antigravity/model"
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
	dataRoot string
	mu       sync.RWMutex
}

// NewAccountStore initializes an AccountStore targeting the specified directory.
// If storeDir is empty, GetDefaultStoreDir() is used. The directory is created if missing.
func NewAccountStore(storeDir string) (*AccountStore, error) {
	explicit := strings.TrimSpace(storeDir) != "" || strings.TrimSpace(os.Getenv("ANTIGRAVITY_STORE_DIR")) != ""
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

	dataRoot := filepath.Dir(absDir)
	if !explicit {
		dataRoot = ResolveDataRootDir(absDir)
	}
	return &AccountStore{storeDir: absDir, dataRoot: dataRoot}, nil
}

// GetStoreDir returns the resolved absolute directory path of the store.
func (s *AccountStore) GetStoreDir() string {
	return s.storeDir
}

// GetDataRootDir returns the resolved data root directory for ancillary assets.
func (s *AccountStore) GetDataRootDir() string {
	if s.dataRoot != "" {
		return s.dataRoot
	}
	return filepath.Dir(s.storeDir)
}

// GetDiscoveredStoreDir returns the directory path for candidate accounts discovered from Chrome profiles.
// It is strictly located inside the engine's store directory: <storeDir>/discovered_accounts.
func (s *AccountStore) GetDiscoveredStoreDir() string {
	return filepath.Join(s.storeDir, "discovered_accounts")
}

// SaveAccount atomically writes or updates the account JSON file for a given email.
// It uses an atomic temp file + fsync + rename pattern to guarantee that a sudden crash
// or power cut will never leave a half-written or corrupted file on disk.
// Invariant-Token-Update-Preservation: Before overwriting, it checks if the account already
// exists with status == "off", and strictly preserves "off" so refresh/capture does not reset it.
func (s *AccountStore) SaveAccount(email string, token *model.TokenData) error {
	return s.saveAccount(email, token, false)
}

// SaveAccountIfChanged checks disk under the write lock; capture never rewrites
// unchanged credentials or destroys an unreadable existing record.
func (s *AccountStore) SaveAccountIfChanged(email string, token *model.TokenData) error {
	return s.saveAccount(email, token, true)
}

func (s *AccountStore) saveAccount(email string, token *model.TokenData, onlyChanged bool) error {
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
	existingData, readErr := os.ReadFile(targetPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read existing credential: %w", readErr)
	}
	if readErr == nil {
		var existingEnv StoredAccountEnvelope
		if err := json.Unmarshal(existingData, &existingEnv); err != nil {
			if onlyChanged {
				return ErrCorruptedStoreFile
			}
		} else {
			if strings.EqualFold(strings.TrimSpace(existingEnv.Status), "off") {
				status = "off"
			}
			existingTier = existingEnv.Tier
			if onlyChanged {
				if existingEnv.TokenData == nil || !existingEnv.TokenData.IsValid() {
					return ErrCorruptedStoreFile
				}
				before, err := json.Marshal(existingEnv.TokenData)
				if err != nil {
					return err
				}
				after, err := json.Marshal(token)
				if err != nil {
					return err
				}
				var left, right interface{}
				if err = json.Unmarshal(before, &left); err != nil {
					return err
				}
				if err = json.Unmarshal(after, &right); err != nil {
					return err
				}
				before, _ = json.Marshal(left)
				after, _ = json.Marshal(right)
				if bytes.Equal(before, after) {
					return nil
				}
			}
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

	// A failed replacement must leave the original credential intact.
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("replace credential: %w", err)
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
	accounts, err := s.listAccounts(false)
	return accounts, err
}

// ListAccountsWithErrors exposes partial reads to the session presenter.
func (s *AccountStore) ListAccountsWithErrors() ([]string, error) { return s.listAccounts(true) }

func (s *AccountStore) listAccounts(reportPartial bool) ([]string, error) {
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
	var failures []error
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
			failures = append(failures, err)
			continue
		}

		var envelope StoredAccountEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			failures = append(failures, ErrCorruptedStoreFile)
			continue
		}

		if envelope.Email != "" && envelope.TokenData != nil && envelope.TokenData.IsValid() {
			emailSet[strings.ToLower(envelope.Email)] = true
		} else {
			failures = append(failures, ErrCorruptedStoreFile)
		}
	}

	result := make([]string, 0, len(emailSet))
	for em := range emailSet {
		result = append(result, em)
	}
	sort.Strings(result)

	if reportPartial {
		return result, errors.Join(failures...)
	}
	return result, nil
}

// ValidateLegacyAccountFile validates that a candidate legacy account file:
// 1. Is contained directly within rootStoreDir (no traversal or reparse escape).
// 2. Is a regular file (not a symlink, directory, or special device).
// 3. Contains a valid Antigravity account JSON envelope with non-empty email and valid token_data.
func ValidateLegacyAccountFile(rootStoreDir, filePath string) error {
	cleanRoot := filepath.Clean(rootStoreDir)
	cleanFile := filepath.Clean(filePath)

	rel, err := filepath.Rel(cleanRoot, cleanFile)
	if err != nil {
		return fmt.Errorf("failed to evaluate relative path: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes root store directory: %s", filePath)
	}
	if filepath.Dir(rel) != "." {
		return fmt.Errorf("file is not directly within root store directory: %s", filePath)
	}

	info, err := os.Lstat(cleanFile)
	if err != nil {
		return fmt.Errorf("failed to lstat file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("file is not a regular file (mode: %s)", info.Mode().String())
	}

	data, err := os.ReadFile(cleanFile)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var env StoredAccountEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("corrupt or invalid json: %w", err)
	}

	if strings.TrimSpace(env.Email) == "" {
		return fmt.Errorf("empty email in envelope")
	}

	if env.TokenData == nil || !env.TokenData.IsValid() {
		return fmt.Errorf("token_data is nil or invalid for %s", env.Email)
	}

	return nil
}

// MigrateLegacyRootStore scans rootStoreDir and atomically moves valid Antigravity
// account files into rootStoreDir/antigravity/, and ensures rootStoreDir/codex/ exists.
// It accepts only an already resolved legacy root directory.
func MigrateLegacyRootStore(rootStoreDir string) error {
	if strings.TrimSpace(rootStoreDir) == "" {
		return fmt.Errorf("legacy store root is not resolved")
	}

	if err := credentialio.CheckPath(rootStoreDir); err != nil {
		return err
	}
	antiDir := filepath.Join(rootStoreDir, "antigravity")
	codexDir := filepath.Join(rootStoreDir, "codex")

	if err := credentialio.CheckPath(antiDir); err != nil {
		return err
	}
	if err := credentialio.CheckPath(codexDir); err != nil {
		return err
	}
	if err := os.MkdirAll(antiDir, 0755); err != nil {
		return fmt.Errorf("failed to create antigravity store dir: %w", err)
	}
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return fmt.Errorf("failed to create codex store dir: %w", err)
	}

	entries, err := os.ReadDir(rootStoreDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type migrationMove struct {
		src string
		dst string
	}
	var moves []migrationMove

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Do not touch subdirectories (antigravity/, codex/, etc.)
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)
		if strings.HasPrefix(lowerName, "acc_tmp_") || strings.HasSuffix(lowerName, ".tmp") {
			continue // Skip scratch temporary files
		}
		if strings.HasSuffix(lowerName, ".json") {
			src := filepath.Join(rootStoreDir, name)
			dst := filepath.Join(antiDir, name)

			if err := ValidateLegacyAccountFile(rootStoreDir, src); err != nil {
				return fmt.Errorf("migration preflight %s: %w", name, err)
			}
			if _, err := os.Stat(dst); err == nil {
				return fmt.Errorf("migration destination already exists: %s", name)
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("migration stat %s: %w", name, err)
			}
			moves = append(moves, migrationMove{src: src, dst: dst})
		}
	}

	// Atomically move files after preflight passes for all candidates
	for _, move := range moves {
		if err := credentialio.CheckPath(move.src); err != nil {
			return err
		}
		if err := credentialio.CheckPath(move.dst); err != nil {
			return err
		}
		if _, err := os.Stat(move.dst); !os.IsNotExist(err) {
			return fmt.Errorf("migration destination changed")
		}
		if err := os.Rename(move.src, move.dst); err != nil {
			return fmt.Errorf("partial migration; retry required: %w", err)
		}
	}

	return nil
}

// MigrateLegacyDiscoveredStore scans the legacy root-level discovered_accounts directory
// (located next to accounts_store, i.e. filepath.Dir(rootStoreDir)/discovered_accounts)
// and safely migrates all candidate account records into rootStoreDir/antigravity/discovered_accounts/.
// It preserves existing 'added' flags and retains the newer timestamp if conflicts exist.
// After all JSON files are migrated, the empty legacy directory is removed.
func MigrateLegacyDiscoveredStore(rootStoreDir string) error {
	if strings.TrimSpace(rootStoreDir) == "" {
		return nil
	}

	legacyRoot := filepath.Dir(rootStoreDir)
	legacyDir := filepath.Join(legacyRoot, "discovered_accounts")
	targetDir := filepath.Join(rootStoreDir, "antigravity", "discovered_accounts")

	fi, err := os.Stat(legacyDir)
	if os.IsNotExist(err) {
		return nil // Không có thư mục cũ -> hoàn thành an toàn (idempotent)
	}
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return nil
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed creating target discovered accounts dir: %w", err)
	}

	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		return fmt.Errorf("failed reading legacy discovered dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)
		if strings.HasPrefix(lowerName, ".discovered-") || strings.HasSuffix(lowerName, ".tmp") {
			continue // Bỏ qua tệp nháp tạm
		}
		if !strings.HasSuffix(lowerName, ".json") {
			continue
		}

		srcPath := filepath.Join(legacyDir, name)
		dstPath := filepath.Join(targetDir, name)

		srcBytes, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}
		var srcEnv DiscoveredAccountEnvelope
		if err := json.Unmarshal(srcBytes, &srcEnv); err != nil {
			continue
		}

		// Kiểm tra nếu tệp đích đã tồn tại -> Hợp nhất dữ liệu để bảo tồn cờ Added & timestamp
		if dstBytes, err := os.ReadFile(dstPath); err == nil {
			var dstEnv DiscoveredAccountEnvelope
			if err := json.Unmarshal(dstBytes, &dstEnv); err == nil {
				if srcEnv.Added {
					dstEnv.Added = true
				}
				if srcEnv.ScannedAt > dstEnv.ScannedAt {
					dstEnv.ScannedAt = srcEnv.ScannedAt
				}
				mergedBytes, marshalErr := json.MarshalIndent(dstEnv, "", "  ")
				if marshalErr == nil {
					if writeErr := os.WriteFile(dstPath, mergedBytes, 0644); writeErr == nil {
						_ = os.Remove(srcPath)
						continue
					}
				}
			}
		}

		// Nếu tệp đích chưa có -> Di chuyển nguyên tử
		if err := os.Rename(srcPath, dstPath); err != nil {
			// Fallback copy nếu khác volume ổ đĩa
			if writeErr := os.WriteFile(dstPath, srcBytes, 0644); writeErr == nil {
				_ = os.Remove(srcPath)
			}
		}
	}

	// Xóa thư mục cũ nếu đã trống hoàn toàn
	remaining, err := os.ReadDir(legacyDir)
	if err == nil && len(remaining) == 0 {
		_ = os.Remove(legacyDir)
	}

	return nil
}
