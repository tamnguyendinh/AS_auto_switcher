package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestDiscoveredStoreRepeatReloadAndPartialRead(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "discovered_accounts")
	s, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.SaveDiscovered(" USER@example.com "); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	raw, err := os.ReadFile(filepath.Join(dir, "user@example.com.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if len(record) != 2 || record["email"] != "user@example.com" || record["scanned_at"].(float64) <= 0 {
		t.Fatalf("unexpected catalog schema: %v", record)
	}
	reopened, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		emails, err := reopened.ListDiscovered()
		if err != nil || !reflect.DeepEqual(emails, []string{"user@example.com"}) {
			t.Fatalf("%v %v", emails, err)
		}
	}
	after, _ := os.ReadFile(filepath.Join(dir, "user@example.com.json"))
	if string(raw) != string(after) {
		t.Fatal("reading changed scanned_at")
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	emails, err := s.ListDiscovered()
	if err == nil || len(emails) != 1 {
		t.Fatalf("partial read lost valid record or hid error: %v %v", emails, err)
	}
}

func TestDiscoveredStoreUnavailableAndFailedReplacement(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "catalog")
	if err := os.WriteFile(dir, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewDiscoveredStore(dir)
	if s == nil || err == nil {
		t.Fatal("expected usable path handle with visible I/O error")
	}
	if err := s.SaveDiscovered("user@example.com"); err == nil {
		t.Fatal("write should fail")
	}
	if _, err := s.ListDiscovered(); err == nil {
		t.Fatal("read should fail")
	}
	dir = filepath.Join(root, "working")
	s, err = NewDiscoveredStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "user@example.com.json")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "keep")
	os.WriteFile(marker, []byte("preserve"), 0600)
	if err := s.SaveDiscovered("user@example.com"); err == nil {
		t.Fatal("replacement should fail")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("failed replacement destroyed target")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temporary file leaked: %v", entries)
	}
}

func TestResolveStoreLocationsAndInvalidOverride(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	unrelated := filepath.Join(root, "cwd")
	for _, p := range []string{repo, unrelated} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module fixture"), 0600)
	repoStore := filepath.Join(repo, "accounts_store")
	if got := resolveStoreDir("", unrelated, filepath.Join(repo, "bin", "app.exe")); got != repoStore {
		t.Fatalf("repo executable: %s", got)
	}
	if got := resolveStoreDir("", filepath.Join(repo, "internal", "store"), filepath.Join(root, "go-build", "app.exe")); got != repoStore {
		t.Fatalf("go run: %s", got)
	}
	portable := filepath.Join(root, "portable")
	if got := resolveStoreDir("", unrelated, filepath.Join(portable, "app.exe")); got != filepath.Join(portable, "accounts_store") {
		t.Fatalf("portable: %s", got)
	}
	os.MkdirAll(filepath.Join(portable, "accounts_store"), 0755)
	if got := resolveStoreDir("", unrelated, filepath.Join(portable, "bin", "app.exe")); got != filepath.Join(portable, "accounts_store") {
		t.Fatalf("legacy portable: %s", got)
	}
	override := filepath.Join(root, "custom", "vault")
	t.Setenv("ANTIGRAVITY_STORE_DIR", override)
	s, err := NewAccountStore("")
	if err != nil || s.GetStoreDir() != override {
		t.Fatalf("new override: %v %v", s, err)
	}
	occupied := filepath.Join(root, "occupied")
	os.WriteFile(occupied, []byte("keep"), 0600)
	t.Setenv("ANTIGRAVITY_STORE_DIR", occupied)
	if _, err := NewAccountStore(""); err == nil {
		t.Fatal("invalid override silently accepted/fell back")
	}
	if got := resolveStoreDir("", "", ""); got != "" {
		t.Fatalf("unexpected cwd fallback: %s", got)
	}
}

func TestDiscoveredStore_BackwardCompatibility_LegacyJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "legacy_store")
	s, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Write a legacy JSON file on disk without the "added" field
	legacyJSON := []byte(`{"email":"legacy@example.com","scanned_at":1710000000.5}`)
	legacyFile := filepath.Join(dir, "legacy@example.com.json")
	if err := os.WriteFile(legacyFile, legacyJSON, 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	// Read records via ListDiscoveredRecords
	records, err := s.ListDiscoveredRecords()
	if err != nil {
		t.Fatalf("unexpected error reading legacy record: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Email != "legacy@example.com" {
		t.Errorf("expected email legacy@example.com, got %s", records[0].Email)
	}
	if records[0].ScannedAt != 1710000000.5 {
		t.Errorf("expected scanned_at 1710000000.5, got %f", records[0].ScannedAt)
	}
	if records[0].Added != false {
		t.Errorf("expected Added == false for legacy json without added field, got %v", records[0].Added)
	}

	// Direct unmarshal check
	var env DiscoveredAccountEnvelope
	if err := json.Unmarshal(legacyJSON, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Added != false {
		t.Errorf("direct unmarshal expected Added == false, got %v", env.Added)
	}
}

func TestDiscoveredStore_SaveDiscovered_PreservesAddedFlag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "preserve_store")
	s, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	email := "user.persist@example.com"

	// 1. First scan discovers email (added should be false)
	if err := s.SaveDiscovered(email); err != nil {
		t.Fatalf("SaveDiscovered failed: %v", err)
	}

	records, err := s.ListDiscoveredRecords()
	if err != nil || len(records) != 1 {
		t.Fatalf("initial scan records: %v, err: %v", records, err)
	}
	if records[0].Added != false {
		t.Fatalf("initial scan should have Added == false, got %v", records[0].Added)
	}
	firstScannedAt := records[0].ScannedAt

	// 2. User marks account as added
	if err := s.SetDiscoveredAdded(email, true); err != nil {
		t.Fatalf("SetDiscoveredAdded true failed: %v", err)
	}

	records, err = s.ListDiscoveredRecords()
	if err != nil || len(records) != 1 {
		t.Fatalf("after set added: %v, err: %v", records, err)
	}
	if records[0].Added != true {
		t.Fatalf("expected Added == true after SetDiscoveredAdded, got %v", records[0].Added)
	}

	// Wait slightly to ensure timestamp would differ
	time.Sleep(20 * time.Millisecond)

	// 3. Chrome re-scans the same account
	if err := s.SaveDiscovered(email); err != nil {
		t.Fatalf("re-scan SaveDiscovered failed: %v", err)
	}

	// 4. Verify Added == true is strictly PRESERVED
	records, err = s.ListDiscoveredRecords()
	if err != nil || len(records) != 1 {
		t.Fatalf("after re-scan records: %v, err: %v", records, err)
	}
	if records[0].Added != true {
		t.Fatalf("Invariant Violation: re-scanning Chrome reset Added flag to false! Expected true, got %v", records[0].Added)
	}
	if records[0].ScannedAt <= firstScannedAt {
		t.Errorf("expected ScannedAt to update on new scan: got %f <= %f", records[0].ScannedAt, firstScannedAt)
	}
}

func TestDiscoveredStore_SetDiscoveredAdded_ToggleAndPersistence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "toggle_store")
	s, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	email := "toggle@example.com"

	// 1. Calling SetDiscoveredAdded directly on non-existent record should create it
	if err := s.SetDiscoveredAdded(email, true); err != nil {
		t.Fatalf("SetDiscoveredAdded failed for new email: %v", err)
	}

	records, err := s.ListDiscoveredRecords()
	if err != nil || len(records) != 1 {
		t.Fatalf("expected 1 record, got %v (err: %v)", records, err)
	}
	if records[0].Email != email || records[0].Added != true || records[0].ScannedAt <= 0 {
		t.Fatalf("unexpected record after direct add: %+v", records[0])
	}
	originalScannedAt := records[0].ScannedAt

	// Check raw json on disk has "added": true
	raw, err := os.ReadFile(filepath.Join(dir, "toggle@example.com.json"))
	if err != nil {
		t.Fatalf("read raw file error: %v", err)
	}
	var rawMap map[string]any
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		t.Fatalf("unmarshal raw error: %v", err)
	}
	if rawMap["added"] != true {
		t.Errorf("raw json missing 'added: true': %v", rawMap)
	}

	// 2. Toggle to false
	if err := s.SetDiscoveredAdded(email, false); err != nil {
		t.Fatalf("SetDiscoveredAdded false failed: %v", err)
	}

	records, err = s.ListDiscoveredRecords()
	if err != nil || len(records) != 1 {
		t.Fatalf("expected 1 record after toggle false, got %v (err: %v)", records, err)
	}
	if records[0].Added != false {
		t.Fatalf("expected Added == false, got %v", records[0].Added)
	}
	// ScannedAt should be preserved from before
	if records[0].ScannedAt != originalScannedAt {
		t.Errorf("ScannedAt was not preserved: got %f, want %f", records[0].ScannedAt, originalScannedAt)
	}

	// Check raw json: when added == false, omitempty should omit added or leave it false
	rawAfter, err := os.ReadFile(filepath.Join(dir, "toggle@example.com.json"))
	if err != nil {
		t.Fatalf("read raw after toggle error: %v", err)
	}
	var rawAfterMap map[string]any
	if err := json.Unmarshal(rawAfter, &rawAfterMap); err != nil {
		t.Fatalf("unmarshal raw error: %v", err)
	}
	if val, exists := rawAfterMap["added"]; exists && val.(bool) != false {
		t.Errorf("raw json should omit added or have false: %v", rawAfterMap)
	}

	// 3. Reopen store from disk to verify cross-process/reload persistence
	reopened, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	reopenedRecords, err := reopened.ListDiscoveredRecords()
	if err != nil || len(reopenedRecords) != 1 {
		t.Fatalf("reopened records: %v (err: %v)", reopenedRecords, err)
	}
	if reopenedRecords[0].Email != email || reopenedRecords[0].Added != false {
		t.Fatalf("reopened record mismatch: %+v", reopenedRecords[0])
	}
}

func TestDiscoveredStore_ListDiscoveredRecords_SortingAndFiltering(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sort_filter_store")
	s, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Add accounts out of order
	if err := s.SaveDiscovered("charlie@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDiscovered("alice@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDiscovered("bob@example.com"); err != nil {
		t.Fatal(err)
	}

	// Mark bob as added
	if err := s.SetDiscoveredAdded("bob@example.com", true); err != nil {
		t.Fatal(err)
	}

	// Add invalid artifacts in directory to test filtering and partial error reporting
	// 1. Subdirectory
	if err := os.Mkdir(filepath.Join(dir, "sub_directory"), 0755); err != nil {
		t.Fatal(err)
	}
	// 2. Non-json file
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}
	// 3. Corrupt json
	if err := os.WriteFile(filepath.Join(dir, "corrupt@example.com.json"), []byte("{broken json"), 0644); err != nil {
		t.Fatal(err)
	}
	// 4. Filename mismatch
	if err := os.WriteFile(filepath.Join(dir, "mismatch@example.com.json"), []byte(`{"email":"other@example.com","scanned_at":100}`), 0644); err != nil {
		t.Fatal(err)
	}
	// 5. Zero scanned_at
	if err := os.WriteFile(filepath.Join(dir, "zero@example.com.json"), []byte(`{"email":"zero@example.com","scanned_at":0}`), 0644); err != nil {
		t.Fatal(err)
	}

	records, err := s.ListDiscoveredRecords()
	// Should return partial failure error for the corrupt/mismatched/zero files
	if err == nil {
		t.Errorf("expected partial read errors for invalid files, got nil")
	}

	// Should strictly return 3 valid records sorted alphabetically by Email (A-Z)
	if len(records) != 3 {
		t.Fatalf("expected 3 valid records, got %d: %+v", len(records), records)
	}

	expectedEmails := []string{"alice@example.com", "bob@example.com", "charlie@example.com"}
	expectedAdded := []bool{false, true, false}

	for i := range records {
		if records[i].Email != expectedEmails[i] {
			t.Errorf("index %d: expected email %s, got %s", i, expectedEmails[i], records[i].Email)
		}
		if records[i].Added != expectedAdded[i] {
			t.Errorf("index %d (%s): expected Added == %v, got %v", i, records[i].Email, expectedAdded[i], records[i].Added)
		}
	}

	// Non-existent store directory returns empty slice, nil error
	emptyStore, err := NewDiscoveredStore(filepath.Join(t.TempDir(), "non_existent"))
	if err != nil {
		t.Fatal(err)
	}
	// remove the directory to simulate IsNotExist
	os.RemoveAll(emptyStore.dir)
	emptyRecords, emptyErr := emptyStore.ListDiscoveredRecords()
	if emptyErr != nil {
		t.Errorf("expected nil error on not exist, got %v", emptyErr)
	}
	if len(emptyRecords) != 0 {
		t.Errorf("expected 0 records, got %d", len(emptyRecords))
	}
}

func TestDiscoveredStore_SetDiscoveredAdded_ErrorAndEdgeCases(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "error_store")
	s, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Empty email
	if err := s.SetDiscoveredAdded("", true); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail for empty email, got %v", err)
	}
	// Whitespace email
	if err := s.SetDiscoveredAdded("   ", true); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail for whitespace email, got %v", err)
	}

	// Case & whitespace normalization
	if err := s.SetDiscoveredAdded("  Normalize@Example.COM  ", true); err != nil {
		t.Fatalf("failed with whitespace/casing: %v", err)
	}
	records, err := s.ListDiscoveredRecords()
	if err != nil || len(records) != 1 {
		t.Fatalf("expected 1 record, got %v (err: %v)", records, err)
	}
	if records[0].Email != "normalize@example.com" {
		t.Errorf("expected normalized email 'normalize@example.com', got '%s'", records[0].Email)
	}

	// Store in an occupied file location (I/O error)
	occupiedPath := filepath.Join(t.TempDir(), "occupied_file")
	if err := os.WriteFile(occupiedPath, []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}
	badStore, _ := NewDiscoveredStore(occupiedPath)
	if err := badStore.SetDiscoveredAdded("test@example.com", true); err == nil {
		t.Errorf("expected error when setting added in occupied directory path, got nil")
	}
}
