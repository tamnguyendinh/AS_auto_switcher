package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"as-auto-switcher/internal/core/credentialio"
	"as-auto-switcher/internal/engines/codex/model"
)

var (
	// ErrAccountNotFound is returned when an account file does not exist.
	ErrAccountNotFound = errors.New("account not found in codex store")

	// ErrCorruptedStoreFile is returned when an account file is corrupted or cannot be unmarshaled.
	ErrCorruptedStoreFile = errors.New("corrupted codex account store file")

	// ErrRevisionConflict is returned when a CAS revision check fails.
	ErrRevisionConflict = errors.New("cas revision conflict in codex account store")

	// ErrIdentityConflict is returned when account_id does not match the existing account identity.
	ErrIdentityConflict = errors.New("identity conflict: account_id mismatch for email")

	// ErrFilenameCollision is returned when two distinct emails resolve to the same sanitized filename.
	ErrFilenameCollision = errors.New("filename collision detected in codex account store")

	// ErrPathTraversal is returned when a potential path traversal attempt is detected.
	ErrPathTraversal = errors.New("potential path traversal detected")

	// ErrInvalidEmail is returned when an email is empty or invalid.
	ErrInvalidEmail = errors.New("invalid or empty email")

	// ErrReauthGenerationProtected is returned when trying to overwrite a pending reauth generation with older active credentials.
	ErrReauthGenerationProtected = errors.New("cannot overwrite pending reauth credential generation")

	regexUnsafeChars = regexp.MustCompile(`[^a-z0-9_.@+-]`)
	regexMultiDots   = regexp.MustCompile(`\.{2,}`)

	windowsReservedNames = map[string]bool{
		"con": true, "prn": true, "aux": true, "nul": true,
		"com1": true, "com2": true, "com3": true, "com4": true,
		"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
		"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
		"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
	}
)

// AccountStore defines the contract for persistent Codex account storage.
type AccountStore interface {
	SaveAccount(email string, envelope *model.CodexAccountEnvelope) error
	GetAccount(email string) (*model.CodexAccountEnvelope, error)
	ListAccounts() ([]*model.CodexAccountEnvelope, error)
	DeleteAccount(email string) error
	HasAccount(email string) bool
	GetAccountStatus(email string) (string, error)
	SetAccountStatus(email string, status string) error
	MergeActiveCredential(email string, envelope *model.CodexAccountEnvelope) error
	GetStoreDir() string
}

type accountStore struct {
	storeDir    string
	mu          sync.RWMutex
	coordinator model.Coordinator
}

// NewAccountStore initializes a Codex AccountStore targeting storeDir.
// If storeDir is empty, GetDefaultCodexStoreDir() is used.
func NewAccountStore(storeDir string) (AccountStore, error) {
	if storeDir == "" {
		storeDir = GetDefaultCodexStoreDir()
	}
	if storeDir == "" {
		return nil, errors.New("cannot determine codex store directory; set CODEX_STORE_DIR")
	}

	absDir, err := filepath.Abs(storeDir)
	if err != nil {
		return nil, fmt.Errorf("resolve codex store directory: %w", err)
	}

	if err := credentialio.PrivateDir(absDir); err != nil {
		return nil, fmt.Errorf("create codex store directory %s: %w", absDir, err)
	}

	return &accountStore{
		storeDir: absDir,
	}, nil
}

// GetStoreDir returns the resolved directory path of the store.
func (s *accountStore) GetStoreDir() string {
	return s.storeDir
}

// SanitizeFilename normalizes an email into a safe Windows filename ending with .json.
func SanitizeFilename(email string) string {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return "unknown_account.json"
	}

	norm := strings.ToLower(trimmed)

	// Extract base name to eliminate any directory traversal components
	norm = strings.ReplaceAll(norm, "/", string(filepath.Separator))
	norm = strings.ReplaceAll(norm, "\\", string(filepath.Separator))
	norm = filepath.Base(norm)

	safe := regexUnsafeChars.ReplaceAllString(norm, "_")
	safe = regexMultiDots.ReplaceAllString(safe, ".")
	safe = strings.Trim(safe, ". ")

	if safe == "" || safe == "." || safe == string(filepath.Separator) {
		safe = "unknown_account"
	}

	baseWithoutExt := strings.TrimSuffix(safe, ".json")
	if windowsReservedNames[baseWithoutExt] {
		safe = "_" + safe
	}

	if !strings.HasSuffix(safe, ".json") {
		safe = safe + ".json"
	}

	return safe
}

// SafeAccountPath constructs and validates the full absolute path for an account file,
// ensuring the path remains strictly inside storeDir and checking for email collision.
func SafeAccountPath(storeDir, email string) (string, error) {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" || len(normEmail) > 240 || !strings.Contains(normEmail, "@") {
		return "", ErrInvalidEmail
	}

	// Traversal Guard: Reject directory traversal or path separators in raw email input
	if strings.Contains(normEmail, "..") || strings.ContainsAny(normEmail, "/\\") {
		return "", ErrPathTraversal
	}

	absStoreDir, err := filepath.Abs(storeDir)
	if err != nil {
		return "", fmt.Errorf("resolve store dir: %w", err)
	}

	filename := SanitizeFilename(normEmail)
	targetPath := filepath.Join(absStoreDir, filename)
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}

	rel, err := filepath.Rel(absStoreDir, absTargetPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) || filepath.Dir(rel) != "." {
		return "", ErrPathTraversal
	}

	if err := credentialio.CheckPath(absTargetPath); err != nil {
		return "", err
	}
	return absTargetPath, nil
}

// SaveAccount atomically writes or updates a Codex account envelope on disk.
func (s *accountStore) SaveAccount(email string, envelope *model.CodexAccountEnvelope) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return ErrInvalidEmail
	}
	if envelope == nil {
		return errors.New("cannot save nil account envelope")
	}
	if len(envelope.AuthData) == 0 {
		return errors.New("auth_data cannot be empty")
	}

	identityEmail, identityID, err := model.ValidateAuth(envelope.AuthData)
	if err != nil {
		return err
	}
	if identityEmail != normEmail || !strings.EqualFold(envelope.Email, normEmail) {
		return ErrIdentityConflict
	}

	targetPath, err := SafeAccountPath(s.storeDir, normEmail)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clone envelope to avoid mutating caller object concurrently
	envToSave := envelope.Clone()
	envToSave.Email = normEmail

	// Check if account already exists
	existingBytes, readErr := credentialio.Read(targetPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read existing account file: %w", readErr)
	}

	now := time.Now()

	if readErr == nil {
		var existing model.CodexAccountEnvelope
		if err := json.Unmarshal(existingBytes, &existing); err != nil {
			return fmt.Errorf("unmarshal existing account file: %w", err)
		}

		// Collision check: verify stored email matches target email
		if !strings.EqualFold(strings.TrimSpace(existing.Email), normEmail) {
			return fmt.Errorf("%w: requested %s but file contains %s", ErrFilenameCollision, normEmail, existing.Email)
		}

		// CAS check: if caller provided a revision, verify it matches current revision
		if envToSave.Revision != existing.Revision {
			return fmt.Errorf("%w: expected revision %d, got %d", ErrRevisionConflict, existing.Revision, envToSave.Revision)
		}

		existingEmail, existingID, err := model.ValidateAuth(existing.AuthData)
		if err != nil || existingEmail != normEmail || existingID != identityID {
			return ErrIdentityConflict
		}
		envToSave.CreatedAt = existing.CreatedAt
		envToSave.Status = existing.Status

		// Preserve plan_type if caller gave empty and existing has one
		if strings.TrimSpace(envToSave.PlanType) == "" && strings.TrimSpace(existing.PlanType) != "" {
			envToSave.PlanType = existing.PlanType
		}

		envToSave.Revision = existing.Revision + 1
	} else {
		// New account; a missing expected record is a conflict, not an insert.
		if envToSave.Revision != 0 {
			return ErrRevisionConflict
		}
		if envToSave.CreatedAt.IsZero() {
			envToSave.CreatedAt = now
		}
		if strings.TrimSpace(envToSave.Status) == "" {
			envToSave.Status = "on"
		}
		envToSave.Revision = 1
	}

	normStatus := strings.ToLower(strings.TrimSpace(envToSave.Status))
	if normStatus != "off" {
		normStatus = "on"
	}
	envToSave.Status = normStatus
	envToSave.SavedAt = now
	if envToSave.Generation == "" {
		envToSave.Generation = model.CredentialGeneration(envToSave.AuthData)
	}

	return s.atomicWriteEnvelope(targetPath, envToSave)
}

// GetAccount retrieves a Codex account envelope by email.
func (s *accountStore) GetAccount(email string) (*model.CodexAccountEnvelope, error) {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return nil, ErrInvalidEmail
	}

	targetPath, err := SafeAccountPath(s.storeDir, normEmail)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := credentialio.Read(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("read account file: %w", err)
	}

	var env model.CodexAccountEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, ErrCorruptedStoreFile
	}

	// Verify identity inside file
	if !strings.EqualFold(strings.TrimSpace(env.Email), normEmail) {
		return nil, fmt.Errorf("%w: requested %s but file contains %s", ErrFilenameCollision, normEmail, env.Email)
	}

	if _, _, err := model.ValidateAuth(env.AuthData); err != nil {
		return nil, ErrCorruptedStoreFile
	}
	return &env, nil
}

// HasAccount checks if an account file exists and is valid.
func (s *accountStore) HasAccount(email string) bool {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return false
	}

	targetPath, err := SafeAccountPath(s.storeDir, normEmail)
	if err != nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	info, err := os.Stat(targetPath)
	return err == nil && !info.IsDir()
}

// ListAccounts lists all valid accounts in the store, sorted stably by CreatedAt then Email.
func (s *accountStore) ListAccounts() ([]*model.CodexAccountEnvelope, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.storeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.CodexAccountEnvelope{}, nil
		}
		return nil, fmt.Errorf("read codex store dir: %w", err)
	}

	accounts := make([]*model.CodexAccountEnvelope, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories such as .runtime/
		}
		name := entry.Name()
		if strings.HasPrefix(name, "codex_tmp_") || strings.HasPrefix(name, "acc_tmp_") || strings.HasSuffix(name, ".tmp") {
			continue // Skip temporary files
		}
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}

		filePath := filepath.Join(s.storeDir, name)
		data, err := credentialio.Read(filePath)
		if err != nil {
			return nil, err
		}

		var env model.CodexAccountEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			return nil, ErrCorruptedStoreFile
		}
		expectedPath, pathErr := SafeAccountPath(s.storeDir, env.Email)
		identityEmail, _, authErr := model.ValidateAuth(env.AuthData)
		if pathErr != nil || !strings.EqualFold(expectedPath, filePath) || authErr != nil || identityEmail != env.Email {
			return nil, ErrCorruptedStoreFile
		}

		accounts = append(accounts, &env)
	}

	// Stable sort: CreatedAt ascending, then Email ascending
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].CreatedAt.Equal(accounts[j].CreatedAt) {
			return accounts[i].Email < accounts[j].Email
		}
		return accounts[i].CreatedAt.Before(accounts[j].CreatedAt)
	})

	return accounts, nil
}

// DeleteAccount removes the single account file corresponding to email.
func (s *accountStore) DeleteAccount(email string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return ErrInvalidEmail
	}

	targetPath, err := SafeAccountPath(s.storeDir, normEmail)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify file exists
	if _, err := os.Stat(targetPath); err != nil {
		if os.IsNotExist(err) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("stat account file: %w", err)
	}

	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("remove account file: %w", err)
	}

	return nil
}

// GetAccountStatus returns the current status ("on" or "off") of the account.
func (s *accountStore) GetAccountStatus(email string) (string, error) {
	env, err := s.GetAccount(email)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(env.Status, "off") {
		return "off", nil
	}
	return "on", nil
}

// SetAccountStatus atomically updates the status field ("on" or "off") of an existing account.
func (s *accountStore) SetAccountStatus(email string, status string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return ErrInvalidEmail
	}

	normStatus := strings.ToLower(strings.TrimSpace(status))
	if normStatus != "off" {
		normStatus = "on"
	}

	targetPath, err := SafeAccountPath(s.storeDir, normEmail)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := credentialio.Read(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrAccountNotFound
		}
		return fmt.Errorf("read account file: %w", err)
	}

	var env model.CodexAccountEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return ErrCorruptedStoreFile
	}

	env.Status = normStatus
	env.Revision++
	env.SavedAt = time.Now()

	return s.atomicWriteEnvelope(targetPath, &env)
}

// MergeActiveCredential merges an active session credential into the store.
// Preserves existing "off" preference, insertion order, and protects unactivated reauth generations.
func (s *accountStore) MergeActiveCredential(email string, envelope *model.CodexAccountEnvelope) error {
	if envelope == nil {
		return errors.New("missing credential")
	}
	existing, err := s.GetAccount(email)
	if err != nil && !errors.Is(err, ErrAccountNotFound) {
		return err
	}
	incoming := envelope.Clone()
	if existing != nil {
		oldEmail, oldID, err := model.ValidateAuth(existing.AuthData)
		newEmail, newID, newErr := model.ValidateAuth(incoming.AuthData)
		if err != nil || newErr != nil || oldEmail != newEmail || oldID != newID {
			return ErrIdentityConflict
		}
		if existing.Generation == "" || incoming.Generation != existing.Generation {
			return ErrReauthGenerationProtected
		}
		incoming.Revision = existing.Revision
		incoming.Status = existing.Status
		incoming.CreatedAt = existing.CreatedAt
		if incoming.PlanType == "" || incoming.PlanType == "unknown" {
			incoming.PlanType = existing.PlanType
		}
	}
	return s.SaveAccount(email, incoming)
}

// Coordinator supplies the same mutation gate to every service using this store.
func Coordinator(st AccountStore) *model.Coordinator {
	if s, ok := st.(*accountStore); ok {
		return &s.coordinator
	}
	return &model.Coordinator{}
}

// atomicWriteEnvelope executes the atomic temp file + sync + rename pattern.
func (s *accountStore) atomicWriteEnvelope(targetPath string, envelope *model.CodexAccountEnvelope) error {
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	return credentialio.Write(targetPath, data)
}

// GetDefaultCodexStoreDir resolves the standard accounts_store/codex/ directory.
func GetDefaultCodexStoreDir() string {
	if override := os.Getenv("CODEX_STORE_DIR"); override != "" {
		return override
	}

	// 1. Search for go.mod to locate repo root
	cwd, err := os.Getwd()
	if err == nil {
		curr := cwd
		for i := 0; i < 6; i++ {
			if _, err := os.Stat(filepath.Join(curr, "go.mod")); err == nil {
				return filepath.Join(curr, "accounts_store", "codex")
			}
			parent := filepath.Dir(curr)
			if parent == curr {
				break
			}
			curr = parent
		}
	}

	// 2. Standalone distribution: check executable directory
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "..", "accounts_store", "codex")
		if _, err := os.Stat(filepath.Join(exeDir, "..", "accounts_store")); err == nil {
			return candidate
		}
	}

	// 3. Fallback to cwd/accounts_store/codex
	return filepath.Join(cwd, "accounts_store", "codex")
}
