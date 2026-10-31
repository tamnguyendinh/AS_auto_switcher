package connector

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var (
	// ErrCodexBinaryNotFound is returned when codex.exe cannot be located on the system.
	ErrCodexBinaryNotFound = errors.New("codex binary not found")
)

// FindCodexBinary locates the codex.exe executable on the host system.
// Priority order:
// 1. Explicit override paths provided as arguments (if existing and valid).
// 2. %LOCALAPPDATA%\OpenAI\Codex\bin\*\codex.exe (sorted by newest modification time).
// 3. System PATH lookup via LookPath("codex.exe") or LookPath("codex").
func FindCodexBinary(overridePaths ...string) (string, error) {
	for _, override := range overridePaths {
		trimmed := strings.TrimSpace(override)
		if trimmed != "" && isRegularExecutable(trimmed) {
			return filepath.Clean(trimmed), nil
		}
	}

	// 2. Search %LOCALAPPDATA%\OpenAI\Codex\bin\*\codex.exe
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		pattern := filepath.Join(localAppData, "OpenAI", "Codex", "bin", "*", "codex.exe")
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			var valid []string
			for _, match := range matches {
				if isRegularExecutable(match) {
					valid = append(valid, match)
				}
			}
			if len(valid) > 0 {
				// Sort descending by file modification time so the latest binary is preferred
				sort.Slice(valid, func(i, j int) bool {
					fi1, err1 := os.Stat(valid[i])
					fi2, err2 := os.Stat(valid[j])
					if err1 == nil && err2 == nil {
						return fi1.ModTime().After(fi2.ModTime())
					}
					return valid[i] > valid[j]
				})
				return filepath.Clean(valid[0]), nil
			}
		}
	}

	// 3. Search system PATH
	if p, err := exec.LookPath("codex.exe"); err == nil && isRegularExecutable(p) {
		return filepath.Clean(p), nil
	}
	if p, err := exec.LookPath("codex"); err == nil && isRegularExecutable(p) {
		return filepath.Clean(p), nil
	}

	return "", fmt.Errorf("%w: checked LOCALAPPDATA and PATH", ErrCodexBinaryNotFound)
}

func isRegularExecutable(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fi.IsDir()
}
