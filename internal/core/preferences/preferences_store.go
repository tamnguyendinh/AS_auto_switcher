package preferences

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultUIPreferencesPath is the canonical disk path for UI settings.
	DefaultUIPreferencesPath = "logs/ui_preferences.json"
)

// UIPreferences stores persistent user interface settings.
type UIPreferences struct {
	Theme        string `json:"theme"`
	MaskAccounts bool   `json:"mask_accounts"`
	AutoSwitch   bool   `json:"auto_switch"`
	AlwaysOnTop  *bool  `json:"always_on_top,omitempty"`
	Language     string `json:"language,omitempty"` // "vi" hoặc "en"
}

// LoadUIPreferences loads UI settings from disk. Returns default preferences if file is missing.
func LoadUIPreferences(path string) (*UIPreferences, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultUIPreferencesPath
	}

	dataBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			defaultTop := true
			return &UIPreferences{Theme: "dark", AlwaysOnTop: &defaultTop, Language: "vi"}, nil
		}
		return nil, fmt.Errorf("failed reading ui preferences: %w", err)
	}

	if len(strings.TrimSpace(string(dataBytes))) == 0 {
		defaultTop := true
		return &UIPreferences{Theme: "dark", AlwaysOnTop: &defaultTop, Language: "vi"}, nil
	}

	var pref UIPreferences
	if err := json.Unmarshal(dataBytes, &pref); err != nil {
		defaultTop := true
		return &UIPreferences{Theme: "dark", AlwaysOnTop: &defaultTop, Language: "vi"}, nil
	}

	if pref.Theme != "light" && pref.Theme != "warm" && pref.Theme != "dark" {
		pref.Theme = "dark"
	}

	if pref.AlwaysOnTop == nil {
		defaultTop := true
		pref.AlwaysOnTop = &defaultTop
	}

	if pref.Language != "en" && pref.Language != "vi" {
		pref.Language = "vi"
	}

	return &pref, nil
}

// SaveUIPreferences atomically writes UI preferences to disk.
func SaveUIPreferences(path string, pref *UIPreferences) error {
	if pref == nil {
		defaultTop := true
		pref = &UIPreferences{Theme: "dark", AlwaysOnTop: &defaultTop, Language: "vi"}
	}
	if strings.TrimSpace(path) == "" {
		path = DefaultUIPreferencesPath
	}

	targetDir := filepath.Dir(path)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed creating directory for ui preferences %s: %w", targetDir, err)
	}

	tmpFile, err := os.CreateTemp(targetDir, ".ui_preferences_tmp_*.json")
	if err != nil {
		return fmt.Errorf("failed creating temporary preferences file: %w", err)
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
	if err := encoder.Encode(pref); err != nil {
		return fmt.Errorf("failed encoding ui preferences: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed flushing temporary preferences file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed closing temporary preferences file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(path)
		if retryErr := os.Rename(tmpPath, path); retryErr != nil {
			return fmt.Errorf("failed replacing preferences file %s: %w", path, retryErr)
		}
	}

	success = true
	return nil
}
