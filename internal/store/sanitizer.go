package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// regexUnsafeChars matches any character outside lowercase alphanumerics, dots, underscores, @, +, and -.
	regexUnsafeChars = regexp.MustCompile(`[^a-z0-9_.@+-]`)

	// regexMultiDots matches consecutive dots to prevent traversal tricks.
	regexMultiDots = regexp.MustCompile(`\.{2,}`)

	// windowsReservedNames matches classic DOS device names that cannot be used as filenames on Windows.
	windowsReservedNames = map[string]bool{
		"con": true, "prn": true, "aux": true, "nul": true,
		"com1": true, "com2": true, "com3": true, "com4": true,
		"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
		"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
		"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
	}

	// ErrPathTraversal is returned when a path traversal attempt is detected.
	ErrPathTraversal = errors.New("potential path traversal detected")

	// ErrInvalidEmail is returned when an email string is empty or invalid.
	ErrInvalidEmail = errors.New("invalid or empty email")
)

// SanitizeFilename normalizes an email into a safe Windows filename ending with .json.
// Matches Python core/account_store.py sanitize_filename behavior while adding Windows reserved name guards.
func SanitizeFilename(email string) string {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return "unknown_account.json"
	}

	norm := strings.ToLower(trimmed)

	// Replace characters outside safe whitelist with underscore
	safe := regexUnsafeChars.ReplaceAllString(norm, "_")

	// Collapse consecutive dots
	safe = regexMultiDots.ReplaceAllString(safe, ".")

	// Strip leading/trailing dots or spaces (Windows filename limitation)
	safe = strings.Trim(safe, ". ")

	// Extract base name to eliminate any directory separators (/ or \)
	// Replace forward slash with backslash for filepath.Base on Windows
	safe = strings.ReplaceAll(safe, "/", string(filepath.Separator))
	safe = strings.ReplaceAll(safe, "\\", string(filepath.Separator))
	safe = filepath.Base(safe)

	if safe == "" || safe == "." || safe == string(filepath.Separator) {
		safe = "unknown_account"
	}

	// Check against Windows reserved device names
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
// strictly verifying that the target path remains inside storeDir.
func SafeAccountPath(storeDir, email string) (string, error) {
	if strings.TrimSpace(email) == "" {
		return "", ErrInvalidEmail
	}

	absStoreDir, err := filepath.Abs(storeDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve store directory: %w", err)
	}

	filename := SanitizeFilename(email)
	targetPath := filepath.Join(absStoreDir, filename)
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Traversal Guard: Ensure absTargetPath has absStoreDir as its parent prefix
	rel, err := filepath.Rel(absStoreDir, absTargetPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", ErrPathTraversal
	}

	return absTargetPath, nil
}

// GetDefaultStoreDir returns the canonical "accounts_store" directory across all execution environments.
// It resolves the repository root containing "accounts_store" and "go.mod" by:
// 1. Checking ANTIGRAVITY_STORE_DIR environment variable if explicitly specified.
// 2. Checking if current working directory (os.Getwd()) is the repository root (contains accounts_store and go.mod).
// 3. Inspecting the executable path (os.Executable()) and walking up its parents (e.g. bin/..).
// 4. Walking up parent directories from CWD (up to 5 levels) to support testing from subdirectories (e.g. internal/store).
// 5. Falling back to the default workspace path or local accounts_store directory.
// Always returns a clean, absolute path.
func GetDefaultStoreDir() string {
	// 0. Explicit environment variable override
	if envDir := strings.TrimSpace(os.Getenv("ANTIGRAVITY_STORE_DIR")); envDir != "" {
		if stat, err := os.Stat(envDir); err == nil && stat.IsDir() {
			if abs, err := filepath.Abs(envDir); err == nil {
				return abs
			}
			return envDir
		}
	}

	isDir := func(p string) bool {
		stat, err := os.Stat(p)
		return err == nil && stat.IsDir()
	}

	isFile := func(p string) bool {
		stat, err := os.Stat(p)
		return err == nil && !stat.IsDir()
	}

	// isProjectRootWithStore verifies that dir contains both accounts_store (directory) and go.mod (file)
	isProjectRootWithStore := func(dir string) bool {
		storePath := filepath.Join(dir, "accounts_store")
		goModPath := filepath.Join(dir, "go.mod")
		return isDir(storePath) && isFile(goModPath)
	}

	// findRootWithStore walks up parent directories (up to maxLevels) to find a directory containing go.mod and accounts_store
	findRootWithStore := func(startDir string, maxLevels int) string {
		curr := startDir
		for i := 0; i < maxLevels; i++ {
			if isProjectRootWithStore(curr) {
				return filepath.Join(curr, "accounts_store")
			}
			parent := filepath.Dir(curr)
			if parent == curr || parent == "" {
				break
			}
			curr = parent
		}
		return ""
	}

	// 1. Check if current working directory is the project root
	cwd, cwdErr := os.Getwd()
	if cwdErr == nil {
		if isProjectRootWithStore(cwd) {
			storePath := filepath.Join(cwd, "accounts_store")
			if abs, err := filepath.Abs(storePath); err == nil {
				return abs
			}
			return storePath
		}
	}

	// 2. Check binary executable location (os.Executable())
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)

		// 2a. Walk up from executable directory up to 5 levels (e.g. bin/.. -> repo root)
		if found := findRootWithStore(exeDir, 5); found != "" {
			if abs, err := filepath.Abs(found); err == nil {
				return abs
			}
			return found
		}

		// 2b. Standalone distribution support (without go.mod): check exeDir/../accounts_store or exeDir/accounts_store
		exeParent := filepath.Dir(exeDir)
		parentStore := filepath.Join(exeParent, "accounts_store")
		if isDir(parentStore) {
			if abs, err := filepath.Abs(parentStore); err == nil {
				return abs
			}
			return parentStore
		}
		dirStore := filepath.Join(exeDir, "accounts_store")
		if isDir(dirStore) {
			if abs, err := filepath.Abs(dirStore); err == nil {
				return abs
			}
			return dirStore
		}
	}

	// 3. Walk up from current working directory up to 5 levels (e.g. internal/store -> repo root during go test)
	if cwdErr == nil {
		if found := findRootWithStore(cwd, 5); found != "" {
			if abs, err := filepath.Abs(found); err == nil {
				return abs
			}
			return found
		}

		// Check if cwd itself has accounts_store directory
		cwdStore := filepath.Join(cwd, "accounts_store")
		if isDir(cwdStore) {
			if abs, err := filepath.Abs(cwdStore); err == nil {
				return abs
			}
			return cwdStore
		}
	}

	// 4. Well-known fallback for this workspace
	defaultWorkspaceStore := `G:\antigravity_auto_switch\accounts_store`
	if isDir(defaultWorkspaceStore) {
		return defaultWorkspaceStore
	}

	// 5. Final fallback
	if cwdErr == nil {
		storePath := filepath.Join(cwd, "accounts_store")
		if abs, err := filepath.Abs(storePath); err == nil {
			return abs
		}
		return storePath
	}
	return "accounts_store"
}
