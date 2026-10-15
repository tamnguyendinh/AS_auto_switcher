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

// GetDefaultStoreDir returns the standard "accounts_store" directory relative to executable or working dir.
func GetDefaultStoreDir() string {
	// First check current working directory
	cwd, err := os.Getwd()
	if err == nil {
		dir := filepath.Join(cwd, "accounts_store")
		return dir
	}
	return "accounts_store"
}
