package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func TestGetTodayQuotaLogPath(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	expectedFileName := fmt.Sprintf("%s_quota_log.json", today)

	// 1. Default baseDir
	defaultPath := GetTodayQuotaLogPath()
	expectedDefault := filepath.Join(DefaultQuotaLogDir, expectedFileName)
	if defaultPath != expectedDefault {
		t.Errorf("expected default path %q, got %q", expectedDefault, defaultPath)
	}

	// 2. Custom baseDir
	tempDir := t.TempDir()
	customPath := GetTodayQuotaLogPath(tempDir)
	expectedCustom := filepath.Join(tempDir, expectedFileName)
	if customPath != expectedCustom {
		t.Errorf("expected custom path %q, got %q", expectedCustom, customPath)
	}

	// 3. Empty / whitespace baseDir fallback
	fallbackPath := GetTodayQuotaLogPath("   ")
	if fallbackPath != expectedDefault {
		t.Errorf("expected whitespace fallback %q, got %q", expectedDefault, fallbackPath)
	}

	// 4. Verify ISO date regex format
	pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}_quota_log\.json$`)
	if !pattern.MatchString(filepath.Base(defaultPath)) {
		t.Errorf("file name %q does not match ISO date pattern", filepath.Base(defaultPath))
	}
}

// TestDailyQuotaLog_LoadTodayOrFallback verifies:
// 1. Loading previous day's log when today's file does not exist yet.
// 2. Saving writes to today's canonical daily file.
func TestDailyQuotaLog_LoadTodayOrFallback(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Prepare yesterday's mock quota log
	yesterdayFile := filepath.Join(tempDir, "2026-10-20_quota_log.json")
	yesterdaySnapshot := QuotaCacheSnapshot{
		SavedAt:      time.Now().Add(-24 * time.Hour).UTC(),
		AccountCount: 1,
		Matrix: map[string]*AccountQuotaData{
			"yesterday@example.com": {
				Email:  "yesterday@example.com",
				Status: StatusPristine,
				Claude: ModelQuotaBucket{
					ModelID:           "claude-3-5",
					RemainingFraction: 0.75,
					FiveHourFraction:  0.8,
					WeeklyFraction:    0.7,
				},
				LastUpdated: time.Now().Add(-24 * time.Hour).UTC(),
			},
		},
	}
	rawJSON, err := json.MarshalIndent(yesterdaySnapshot, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal yesterday snapshot: %v", err)
	}
	if err := os.WriteFile(yesterdayFile, rawJSON, 0644); err != nil {
		t.Fatalf("failed to write yesterday file: %v", err)
	}

	// 2. Today's file must NOT exist before load
	todayPath := GetTodayQuotaLogPath(tempDir)
	_ = os.Remove(todayPath)

	// 3. Initialize cache targeting today's path and load from disk
	cache := NewQuotaCache(todayPath)
	if err := cache.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	// 4. Verify data from yesterday was loaded into memory
	loaded, ok := cache.Get("yesterday@example.com")
	if !ok || loaded == nil {
		t.Fatalf("expected data loaded from yesterday's fallback file")
	}
	if loaded.Claude.RemainingFraction != 0.75 {
		t.Errorf("expected Claude remaining fraction 0.75, got %f", loaded.Claude.RemainingFraction)
	}

	// 5. SaveToDisk must generate today's file
	if err := cache.SaveToDisk(); err != nil {
		t.Fatalf("SaveToDisk failed: %v", err)
	}

	if _, err := os.Stat(todayPath); os.IsNotExist(err) {
		t.Fatalf("expected today's file to be created at %q, but it does not exist", todayPath)
	}

	// 6. Reload from fresh cache targeting today's path
	freshCache := NewQuotaCache(todayPath)
	if err := freshCache.LoadFromDisk(); err != nil {
		t.Fatalf("freshCache.LoadFromDisk failed: %v", err)
	}
	reloaded, ok := freshCache.Get("yesterday@example.com")
	if !ok || reloaded == nil {
		t.Fatalf("expected data persisted and reloaded from today's file")
	}
	if reloaded.Claude.RemainingFraction != 0.75 {
		t.Errorf("expected Claude remaining fraction 0.75, got %f", reloaded.Claude.RemainingFraction)
	}
}

func TestDailyQuotaLog_TodayExists(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Create yesterday's file
	yesterdayFile := filepath.Join(tempDir, "2026-10-20_quota_log.json")
	yesterdaySnapshot := QuotaCacheSnapshot{
		SavedAt:      time.Now().Add(-24 * time.Hour).UTC(),
		AccountCount: 1,
		Matrix: map[string]*AccountQuotaData{
			"user@example.com": {
				Email:  "user@example.com",
				Status: StatusPristine,
				Claude: ModelQuotaBucket{RemainingFraction: 0.2},
			},
		},
	}
	yesterdayBytes, _ := json.Marshal(yesterdaySnapshot)
	_ = os.WriteFile(yesterdayFile, yesterdayBytes, 0644)

	// 2. Create today's file with updated data
	todayPath := GetTodayQuotaLogPath(tempDir)
	todaySnapshot := QuotaCacheSnapshot{
		SavedAt:      time.Now().UTC(),
		AccountCount: 1,
		Matrix: map[string]*AccountQuotaData{
			"user@example.com": {
				Email:  "user@example.com",
				Status: StatusPlentiful,
				Claude: ModelQuotaBucket{RemainingFraction: 0.95},
			},
		},
	}
	todayBytes, _ := json.Marshal(todaySnapshot)
	_ = os.WriteFile(todayPath, todayBytes, 0644)

	// 3. Load should read today's file, NOT yesterday
	cache := NewQuotaCache(todayPath)
	if err := cache.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	loaded, ok := cache.Get("user@example.com")
	if !ok || loaded == nil {
		t.Fatalf("expected user data loaded")
	}
	if loaded.Claude.RemainingFraction != 0.95 {
		t.Errorf("expected today's fraction 0.95, got %f", loaded.Claude.RemainingFraction)
	}
	if loaded.Status != StatusPlentiful {
		t.Errorf("expected today's status %s, got %s", StatusPlentiful, loaded.Status)
	}
}

func TestDailyQuotaLog_MultiplePreviousDays(t *testing.T) {
	tempDir := t.TempDir()

	// Create 3 old daily files
	dates := []struct {
		filename string
		fraction float64
	}{
		{"2026-10-10_quota_log.json", 0.10},
		{"2026-10-15_quota_log.json", 0.50},
		{"2026-10-18_quota_log.json", 0.85},
	}

	for _, d := range dates {
		snap := QuotaCacheSnapshot{
			SavedAt:      time.Now().UTC(),
			AccountCount: 1,
			Matrix: map[string]*AccountQuotaData{
				"test@example.com": {
					Email:  "test@example.com",
					Claude: ModelQuotaBucket{RemainingFraction: d.fraction},
				},
			},
		}
		bytes, _ := json.Marshal(snap)
		_ = os.WriteFile(filepath.Join(tempDir, d.filename), bytes, 0644)
	}

	// Also write unrelated files that should be ignored
	_ = os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("notes"), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "other.json"), []byte("{}"), 0644)

	todayPath := GetTodayQuotaLogPath(tempDir)
	cache := NewQuotaCache(todayPath)
	if err := cache.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	loaded, ok := cache.Get("test@example.com")
	if !ok || loaded == nil {
		t.Fatalf("expected data loaded from newest fallback")
	}
	// Should pick 2026-10-18 (most recent before today)
	if loaded.Claude.RemainingFraction != 0.85 {
		t.Errorf("expected fraction from newest day (0.85), got %f", loaded.Claude.RemainingFraction)
	}
}

func TestDailyQuotaLog_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	todayPath := GetTodayQuotaLogPath(tempDir)

	cache := NewQuotaCache(todayPath)
	if err := cache.LoadFromDisk(); err != nil {
		t.Fatalf("expected nil error on empty dir, got %v", err)
	}

	if len(cache.GetAll()) != 0 {
		t.Errorf("expected empty cache matrix, got %d items", len(cache.GetAll()))
	}
}

func TestDailyQuotaLog_SaveCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested", "quota_log")

	todayPath := GetTodayQuotaLogPath(nestedDir)
	cache := NewQuotaCache(todayPath)
	cache.Update(&AccountQuotaData{
		Email:  "nested@example.com",
		Status: StatusPristine,
	})

	if err := cache.SaveToDisk(); err != nil {
		t.Fatalf("SaveToDisk failed to create nested directory: %v", err)
	}

	if _, err := os.Stat(todayPath); os.IsNotExist(err) {
		t.Fatalf("expected file created at %q", todayPath)
	}
}

func TestDailyQuotaLog_DirectoryPathTarget(t *testing.T) {
	tempDir := t.TempDir()

	// Put a fallback file in tempDir
	fallbackFile := filepath.Join(tempDir, "2026-10-19_quota_log.json")
	snap := QuotaCacheSnapshot{
		SavedAt:      time.Now().UTC(),
		AccountCount: 1,
		Matrix: map[string]*AccountQuotaData{
			"dirtest@example.com": {
				Email:  "dirtest@example.com",
				Status: StatusPristine,
				Claude: ModelQuotaBucket{RemainingFraction: 0.65},
			},
		},
	}
	bytes, _ := json.Marshal(snap)
	_ = os.WriteFile(fallbackFile, bytes, 0644)

	// Initialize cache targeting the directory itself
	cache := NewQuotaCache(tempDir)
	if err := cache.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk failed: %v", err)
	}

	loaded, ok := cache.Get("dirtest@example.com")
	if !ok || loaded == nil {
		t.Fatalf("expected data loaded via directory target")
	}
	if loaded.Claude.RemainingFraction != 0.65 {
		t.Errorf("expected fraction 0.65, got %f", loaded.Claude.RemainingFraction)
	}

	// Save should write today's file inside tempDir
	if err := cache.SaveToDisk(); err != nil {
		t.Fatalf("SaveToDisk failed: %v", err)
	}

	todayPath := GetTodayQuotaLogPath(tempDir)
	if _, err := os.Stat(todayPath); os.IsNotExist(err) {
		t.Fatalf("expected today's file created at %q", todayPath)
	}
}
