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

// GetDefaultStoreDir resolves one data location for credentials and their consumers.
// An explicit override is never discarded just because it does not exist yet.
func GetDefaultStoreDir() string {
	cwd, _ := os.Getwd()
	exe, _ := os.Executable()
	return resolveStoreDir(os.Getenv("ANTIGRAVITY_STORE_DIR"), cwd, exe)
}

// GetDefaultRootStoreDir resolves the root accounts_store/ directory.
// Returns empty string if ANTIGRAVITY_STORE_DIR is explicitly configured.
func GetDefaultRootStoreDir() string {
	if strings.TrimSpace(os.Getenv("ANTIGRAVITY_STORE_DIR")) != "" {
		return ""
	}
	cwd, _ := os.Getwd()
	exe, _ := os.Executable()
	return resolveRootStoreDir(cwd, exe)
}

func resolveRootStoreDir(cwd, exe string) string {
	absolute := func(path string) string {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return filepath.Clean(path)
	}
	// A repo's data lives outside its disposable build output, including on
	// first run when accounts_store has not been created yet.
	findRepo := func(start string) string {
		for dir, level := start, 0; dir != "" && level < 6; level++ {
			if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
				return filepath.Join(dir, "accounts_store")
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
		return ""
	}
	exeDir := ""
	if exe != "" {
		exeDir = filepath.Dir(exe)
		if path := findRepo(exeDir); path != "" {
			return absolute(path)
		}
	}
	if path := findRepo(cwd); path != "" {
		return absolute(path)
	}
	if exeDir != "" {
		// Preserve existing standalone distributions, including bin/../accounts_store.
		// Recognizes legacy root accounts_store first before selecting subfolder.
		for _, path := range []string{filepath.Join(filepath.Dir(exeDir), "accounts_store"), filepath.Join(exeDir, "accounts_store")} {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				return absolute(path)
			}
		}
		return absolute(filepath.Join(exeDir, "accounts_store"))
	}
	return ""
}

func resolveStoreDir(override, cwd, exe string) string {
	absolute := func(path string) string {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return filepath.Clean(path)
	}
	if override = strings.TrimSpace(override); override != "" {
		return absolute(override)
	}
	root := resolveRootStoreDir(cwd, exe)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "antigravity")
}

// ResolveDataRootDir resolves the directory for ancillary data (such as logs/).
// Note: Candidate accounts discovered from Chrome are now co-located under
// <storeDir>/discovered_accounts (i.e. accounts_store/antigravity/discovered_accounts).
// When storeDir points to .../accounts_store/antigravity, the data root is the parent of accounts_store.
// For legacy stores or explicit overrides, it returns filepath.Dir(storeDir).
func ResolveDataRootDir(storeDir string) string {
	if storeDir == "" {
		return ""
	}
	clean := filepath.Clean(storeDir)
	if strings.EqualFold(filepath.Base(clean), "antigravity") {
		parent := filepath.Dir(clean)
		if strings.EqualFold(filepath.Base(parent), "accounts_store") {
			return filepath.Dir(parent)
		}
	}
	if strings.EqualFold(filepath.Base(clean), "accounts_store") {
		return filepath.Dir(clean)
	}
	return filepath.Dir(clean)
}
