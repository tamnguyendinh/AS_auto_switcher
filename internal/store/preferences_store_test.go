package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_PreferencesLoadDefault(t *testing.T) {
	tmpDir := t.TempDir()
	prefPath := filepath.Join(tmpDir, "non_existent.json")

	pref, err := LoadUIPreferences(prefPath)
	if err != nil {
		t.Fatalf("unexpected error loading non-existent preferences: %v", err)
	}
	if pref.Language != "vi" {
		t.Errorf("expected default language 'vi', got '%s'", pref.Language)
	}
	if pref.Theme != "dark" {
		t.Errorf("expected default theme 'dark', got '%s'", pref.Theme)
	}
	if pref.AlwaysOnTop == nil || !*pref.AlwaysOnTop {
		t.Errorf("expected default always_on_top to be true")
	}
}

func TestStore_PreferencesSaveAndLoadLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	prefPath := filepath.Join(tmpDir, "pref.json")

	top := false
	toSave := &UIPreferences{
		Theme:        "light",
		MaskAccounts: true,
		AutoSwitch:   true,
		AlwaysOnTop:  &top,
		Language:     "en",
	}

	if err := SaveUIPreferences(prefPath, toSave); err != nil {
		t.Fatalf("failed saving preferences: %v", err)
	}

	loaded, err := LoadUIPreferences(prefPath)
	if err != nil {
		t.Fatalf("failed loading preferences: %v", err)
	}
	if loaded.Language != "en" {
		t.Errorf("expected language 'en', got '%s'", loaded.Language)
	}
	if loaded.Theme != "light" {
		t.Errorf("expected theme 'light', got '%s'", loaded.Theme)
	}
	if !loaded.MaskAccounts || !loaded.AutoSwitch {
		t.Errorf("expected mask_accounts and auto_switch to be true")
	}
	if loaded.AlwaysOnTop == nil || *loaded.AlwaysOnTop != false {
		t.Errorf("expected always_on_top to be false")
	}
}

func TestStore_PreferencesLanguageInvalidFallback(t *testing.T) {
	tmpDir := t.TempDir()
	prefPath := filepath.Join(tmpDir, "pref_invalid.json")

	// Write raw JSON with invalid language
	rawJSON := `{"theme":"dark","language":"fr"}`
	if err := os.WriteFile(prefPath, []byte(rawJSON), 0644); err != nil {
		t.Fatalf("failed writing test file: %v", err)
	}

	loaded, err := LoadUIPreferences(prefPath)
	if err != nil {
		t.Fatalf("failed loading preferences: %v", err)
	}
	if loaded.Language != "vi" {
		t.Errorf("expected invalid language to fallback to 'vi', got '%s'", loaded.Language)
	}
}
