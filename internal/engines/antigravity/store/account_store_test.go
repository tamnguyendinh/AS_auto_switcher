package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"antigravity-switch/internal/engines/antigravity/model"
)

func TestAccountStoreRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create AccountStore: %v", err)
	}

	email := "Test.User@Gmail.COM"
	normEmail := "test.user@gmail.com"

	// 1. Initial state: not stored
	if store.HasAccount(email) {
		t.Errorf("expected HasAccount to return false before save")
	}
	_, err = store.GetAccount(email)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got: %v", err)
	}

	// 2. Save account
	now := time.Now().Truncate(time.Second)
	token := &model.TokenData{
		AccessToken:  "ya29.test_token_p6a",
		RefreshToken: "1//refresh_token_p6a",
		IDToken:      "id_token_jwt_tiếng_việt",
		TokenType:    "Bearer",
		Expiry:       now.Add(2 * time.Hour),
		AuthMethod:   "consumer",
	}

	if err := store.SaveAccount(email, token); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// 3. HasAccount and GetAccount
	if !store.HasAccount(email) {
		t.Errorf("expected HasAccount to return true after save")
	}
	if !store.HasAccount(normEmail) {
		t.Errorf("expected HasAccount to match normalized lowercase email")
	}

	loadedToken, err := store.GetAccount(email)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if loadedToken.AccessToken != token.AccessToken {
		t.Errorf("AccessToken mismatch: expected %s, got %s", token.AccessToken, loadedToken.AccessToken)
	}
	if loadedToken.RefreshToken != token.RefreshToken {
		t.Errorf("RefreshToken mismatch: expected %s, got %s", token.RefreshToken, loadedToken.RefreshToken)
	}
	if loadedToken.IDToken != token.IDToken {
		t.Errorf("IDToken mismatch: expected %s, got %s", token.IDToken, loadedToken.IDToken)
	}

	// 4. Delete account
	if err := store.DeleteAccount(email); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	if store.HasAccount(email) {
		t.Errorf("expected HasAccount to return false after delete")
	}

	// 5. Delete again should be idempotent
	if err := store.DeleteAccount(email); err != nil {
		t.Fatalf("expected idempotent DeleteAccount to succeed, got: %v", err)
	}
}

func TestAccountStoreListAccounts(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create AccountStore: %v", err)
	}

	// Empty list
	list, err := store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got: %v", list)
	}

	// Add 3 accounts
	testEmails := []string{
		"zebra@example.com",
		"alpha@example.com",
		"beta@example.com",
	}

	for _, em := range testEmails {
		token := &model.TokenData{AccessToken: "token_" + em, RefreshToken: "ref_" + em}
		if err := store.SaveAccount(em, token); err != nil {
			t.Fatalf("SaveAccount failed for %s: %v", em, err)
		}
	}

	list, err = store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 accounts, got: %d", len(list))
	}

	// Verify sorted order: alpha, beta, zebra
	expectedOrder := []string{"alpha@example.com", "beta@example.com", "zebra@example.com"}
	for i, expected := range expectedOrder {
		if list[i] != expected {
			t.Errorf("at index %d: expected %s, got %s", i, expected, list[i])
		}
	}
}

func TestAccountStoreCorruptedJSONHandling(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create AccountStore: %v", err)
	}

	// 1. Truncated syntax JSON file
	badFile1 := filepath.Join(tempDir, "corrupt1@example.com.json")
	_ = os.WriteFile(badFile1, []byte(`{"email": "corrupt1@example.com", "token_data": {"access_token": "abc`), 0644)

	// 2. Empty file
	badFile2 := filepath.Join(tempDir, "empty@example.com.json")
	_ = os.WriteFile(badFile2, []byte(``), 0644)

	// 3. Binary garbage non-JSON file
	badFile3 := filepath.Join(tempDir, "binary@example.com.json")
	_ = os.WriteFile(badFile3, []byte("\x00\xff\xfe\x01\x02\x03GARBAGE"), 0644)

	// 4. Missing token_data field
	badFile4 := filepath.Join(tempDir, "notoken@example.com.json")
	_ = os.WriteFile(badFile4, []byte(`{"email": "notoken@example.com", "saved_at": 12345.67}`), 0644)

	// 5. Valid account alongside
	validToken := &model.TokenData{AccessToken: "valid_token"}
	if err := store.SaveAccount("valid@example.com", validToken); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// Verify HasAccount returns false for corrupt files
	if store.HasAccount("corrupt1@example.com") {
		t.Errorf("expected HasAccount false for truncated JSON")
	}
	if store.HasAccount("empty@example.com") {
		t.Errorf("expected HasAccount false for empty file")
	}
	if store.HasAccount("binary@example.com") {
		t.Errorf("expected HasAccount false for binary garbage")
	}
	if store.HasAccount("notoken@example.com") {
		t.Errorf("expected HasAccount false for missing token_data")
	}

	// Verify GetAccount returns error safely
	_, err = store.GetAccount("corrupt1@example.com")
	if err == nil {
		t.Errorf("expected error from GetAccount on corrupted JSON")
	}

	// Verify ListAccounts ignores all corrupted files and returns only valid account
	list, err := store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}
	if len(list) != 1 || list[0] != "valid@example.com" {
		t.Fatalf("expected exactly [valid@example.com], got: %v", list)
	}
}

func TestSanitizeFilenameEdgeCases(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"User.Test@Gmail.COM", "user.test@gmail.com.json"},
		{"  developer@domain.com.  ", "developer@domain.com.json"},
		{"dev+test@domain.com", "dev+test@domain.com.json"},
		{"user#special!name@domain.com", "user_special_name@domain.com.json"},
		{"user..name@domain.com", "user.name@domain.com.json"},
		{"../../evil@domain.com", "_._evil@domain.com.json"},
		{"..\\..\\evil@domain.com", "_._evil@domain.com.json"},
		{"", "unknown_account.json"},
		{"   ", "unknown_account.json"},
		{"con", "_con.json"},
		{"aux", "_aux.json"},
		{"nul", "_nul.json"},
		{"com1", "_com1.json"},
	}

	for _, tc := range testCases {
		got := SanitizeFilename(tc.input)
		if got != tc.expected {
			t.Errorf("SanitizeFilename(%q): expected %q, got %q", tc.input, tc.expected, got)
		}
	}
}

func TestSafeAccountPathTraversalGuards(t *testing.T) {
	tempDir := t.TempDir()

	// Normal email
	path, err := SafeAccountPath(tempDir, "user@domain.com")
	if err != nil {
		t.Fatalf("expected valid path, got: %v", err)
	}
	if filepath.Dir(path) != tempDir {
		t.Errorf("expected path inside tempDir %s, got: %s", tempDir, path)
	}

	// Empty email
	_, err = SafeAccountPath(tempDir, "")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail, got: %v", err)
	}
}

func TestAccountStoreConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create AccountStore: %v", err)
	}

	const goroutines = 20
	const iterations = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			email := fmt.Sprintf("concurrent_user_%d@example.com", id)
			for i := 0; i < iterations; i++ {
				token := &model.TokenData{
					AccessToken:  fmt.Sprintf("access_token_%d_%d", id, i),
					RefreshToken: fmt.Sprintf("refresh_token_%d_%d", id, i),
				}
				_ = store.SaveAccount(email, token)
				_ = store.HasAccount(email)
				_, _ = store.GetAccount(email)
				_, _ = store.ListAccounts()
			}
		}(g)
	}

	wg.Wait()

	list, err := store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts failed after concurrency: %v", err)
	}
	if len(list) != goroutines {
		t.Fatalf("expected %d accounts, got %d", goroutines, len(list))
	}
}

func TestModelTokenDataDualSerialization(t *testing.T) {
	// 1. Antigravity real nested format
	nestedJSON := []byte(`{
		"token": {
			"access_token": "ya29.nested_access",
			"token_type": "Bearer",
			"refresh_token": "1//nested_refresh",
			"expiry": "2026-10-15T16:56:40.5499029+07:00"
		},
		"auth_method": "consumer",
		"id_token": "jwt_id_token_nested"
	}`)

	var token1 model.TokenData
	if err := json.Unmarshal(nestedJSON, &token1); err != nil {
		t.Fatalf("failed to unmarshal nested token: %v", err)
	}

	if token1.AccessToken != "ya29.nested_access" {
		t.Errorf("expected AccessToken 'ya29.nested_access', got %q", token1.AccessToken)
	}
	if token1.RefreshToken != "1//nested_refresh" {
		t.Errorf("expected RefreshToken '1//nested_refresh', got %q", token1.RefreshToken)
	}
	if token1.IDToken != "jwt_id_token_nested" {
		t.Errorf("expected IDToken 'jwt_id_token_nested', got %q", token1.IDToken)
	}
	if token1.AuthMethod != "consumer" {
		t.Errorf("expected AuthMethod 'consumer', got %q", token1.AuthMethod)
	}
	if token1.Expiry.IsZero() {
		t.Errorf("expected non-zero parsed Expiry")
	}

	// 2. Flat format
	flatJSON := []byte(`{
		"access_token": "flat_access",
		"refresh_token": "flat_refresh",
		"token_type": "Bearer",
		"id_token": "jwt_flat",
		"expiry": "2026-10-15T16:56:40Z"
	}`)

	var token2 model.TokenData
	if err := json.Unmarshal(flatJSON, &token2); err != nil {
		t.Fatalf("failed to unmarshal flat token: %v", err)
	}

	if token2.AccessToken != "flat_access" {
		t.Errorf("expected AccessToken 'flat_access', got %q", token2.AccessToken)
	}
	if token2.RefreshToken != "flat_refresh" {
		t.Errorf("expected RefreshToken 'flat_refresh', got %q", token2.RefreshToken)
	}

	// 3. Marshal programmatic token produces both top-level and nested structure
	token3 := &model.TokenData{
		AccessToken:  "prog_access",
		RefreshToken: "prog_refresh",
		IDToken:      "prog_id",
	}
	marshaled, err := json.Marshal(token3)
	if err != nil {
		t.Fatalf("failed to marshal token3: %v", err)
	}

	var verifyMap map[string]interface{}
	if err := json.Unmarshal(marshaled, &verifyMap); err != nil {
		t.Fatalf("failed to re-parse marshaled json: %v", err)
	}
	if verifyMap["access_token"] != "prog_access" {
		t.Errorf("expected top-level access_token, got: %v", verifyMap["access_token"])
	}
	innerToken, ok := verifyMap["token"].(map[string]interface{})
	if !ok || innerToken["access_token"] != "prog_access" {
		t.Errorf("expected nested token.access_token, got: %v", innerToken)
	}
}

func TestGetDefaultStoreDir_CurrentProject(t *testing.T) {
	// Ensure no env var pollution
	t.Setenv("ANTIGRAVITY_STORE_DIR", "")

	storeDir := GetDefaultStoreDir()
	if !filepath.IsAbs(storeDir) {
		t.Errorf("expected GetDefaultStoreDir to return absolute path, got: %s", storeDir)
	}

	stat, err := os.Stat(storeDir)
	if err != nil {
		t.Fatalf("failed to stat GetDefaultStoreDir result %s: %v", storeDir, err)
	}
	if !stat.IsDir() {
		t.Errorf("expected storeDir to be directory, got non-dir: %s", storeDir)
	}
	if filepath.Base(storeDir) != "accounts_store" {
		t.Errorf("expected directory name to be accounts_store, got: %s", filepath.Base(storeDir))
	}
}

func TestGetDefaultStoreDir_SimulatedSubdirBin(t *testing.T) {
	t.Setenv("ANTIGRAVITY_STORE_DIR", "")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// Determine expected root accounts_store
	baseStoreDir := GetDefaultStoreDir()

	// Simulate changing working directory to bin/
	binDir := filepath.Join(filepath.Dir(baseStoreDir), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin directory: %v", err)
	}

	if err := os.Chdir(binDir); err != nil {
		t.Fatalf("failed to chdir to binDir %s: %v", binDir, err)
	}

	// Verify that from bin/, GetDefaultStoreDir() still resolves to root accounts_store
	resolvedFromBin := GetDefaultStoreDir()
	if resolvedFromBin != baseStoreDir {
		t.Errorf("expected resolution from bin/ to return %s, got %s", baseStoreDir, resolvedFromBin)
	}
}

func TestGetDefaultStoreDir_MockNestedHierarchy(t *testing.T) {
	t.Setenv("ANTIGRAVITY_STORE_DIR", "")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// Create mock workspace hierarchy in temp dir
	mockRoot := t.TempDir()
	mockMod := filepath.Join(mockRoot, "go.mod")
	if err := os.WriteFile(mockMod, []byte("module mock-antigravity\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatalf("failed to create mock go.mod: %v", err)
	}
	mockStore := filepath.Join(mockRoot, "accounts_store")
	if err := os.MkdirAll(mockStore, 0755); err != nil {
		t.Fatalf("failed to create mock accounts_store: %v", err)
	}
	mockBin := filepath.Join(mockRoot, "bin")
	if err := os.MkdirAll(mockBin, 0755); err != nil {
		t.Fatalf("failed to create mock bin: %v", err)
	}
	mockDeepSubdir := filepath.Join(mockRoot, "cmd", "nested", "deep")
	if err := os.MkdirAll(mockDeepSubdir, 0755); err != nil {
		t.Fatalf("failed to create mock deep subdir: %v", err)
	}

	// 1. Test when CWD is mockRoot/bin
	if err := os.Chdir(mockBin); err != nil {
		t.Fatalf("failed to chdir to mockBin: %v", err)
	}
	resBin := GetDefaultStoreDir()
	expectedStore, err := filepath.Abs(mockStore)
	if err != nil {
		t.Fatalf("failed to get abs of mockStore: %v", err)
	}
	if resBin != expectedStore {
		t.Errorf("expected resolution from mockBin to return %s, got %s", expectedStore, resBin)
	}

	// 2. Test when CWD is deep subdirectory
	if err := os.Chdir(mockDeepSubdir); err != nil {
		t.Fatalf("failed to chdir to mockDeepSubdir: %v", err)
	}
	resDeep := GetDefaultStoreDir()
	if resDeep != expectedStore {
		t.Errorf("expected resolution from mockDeepSubdir to return %s, got %s", expectedStore, resDeep)
	}
}

func TestGetDefaultStoreDir_EnvVarOverride(t *testing.T) {
	customTemp := t.TempDir()
	t.Setenv("ANTIGRAVITY_STORE_DIR", customTemp)

	resolved := GetDefaultStoreDir()
	expected, err := filepath.Abs(customTemp)
	if err != nil {
		t.Fatalf("failed to get abs of customTemp: %v", err)
	}
	if resolved != expected {
		t.Errorf("expected env var override %s, got %s", expected, resolved)
	}
}

func TestAccountStore_Status_Persistence_And_Preservation(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create AccountStore: %v", err)
	}

	email := "status.test@example.com"
	token := &model.TokenData{
		AccessToken:  "access_token_1",
		RefreshToken: "refresh_token_1",
	}

	// 1. Save new account -> should default to "on"
	if err := store.SaveAccount(email, token); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	status, err := store.GetAccountStatus(email)
	if err != nil {
		t.Fatalf("GetAccountStatus failed: %v", err)
	}
	if status != "on" {
		t.Errorf("expected default status 'on', got: %q", status)
	}

	// 2. Set status to "off"
	if err := store.SetAccountStatus(email, "off"); err != nil {
		t.Fatalf("SetAccountStatus('off') failed: %v", err)
	}

	status, err = store.GetAccountStatus(email)
	if err != nil {
		t.Fatalf("GetAccountStatus after off failed: %v", err)
	}
	if status != "off" {
		t.Errorf("expected status 'off', got: %q", status)
	}

	// 3. Invariant-Token-Update-Preservation:
	// Overwrite token using SaveAccount (simulating token refresh or credential capture)
	newToken := &model.TokenData{
		AccessToken:  "access_token_refreshed",
		RefreshToken: "refresh_token_refreshed",
	}
	if err := store.SaveAccount(email, newToken); err != nil {
		t.Fatalf("SaveAccount overwrite failed: %v", err)
	}

	// Status must strictly remain "off"
	status, err = store.GetAccountStatus(email)
	if err != nil {
		t.Fatalf("GetAccountStatus after token overwrite failed: %v", err)
	}
	if status != "off" {
		t.Errorf("Invariant-Token-Update-Preservation violated: expected 'off', got: %q", status)
	}

	// Verify token was indeed updated
	loadedToken, err := store.GetAccount(email)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if loadedToken.AccessToken != "access_token_refreshed" {
		t.Errorf("expected updated AccessToken, got: %q", loadedToken.AccessToken)
	}

	// 4. Set status back to "on"
	if err := store.SetAccountStatus(email, "on"); err != nil {
		t.Fatalf("SetAccountStatus('on') failed: %v", err)
	}
	status, err = store.GetAccountStatus(email)
	if err != nil {
		t.Fatalf("GetAccountStatus after on failed: %v", err)
	}
	if status != "on" {
		t.Errorf("expected status 'on', got: %q", status)
	}

	// 5. Invariant-Backward-Compatibility-Default-On:
	// Write a raw legacy JSON file without any status field
	legacyEmail := "legacy.user@example.com"
	legacyFile := filepath.Join(tempDir, "legacy.user@example.com.json")
	legacyJSON := `{
		"email": "legacy.user@example.com",
		"saved_at": 1700000000.0,
		"token_data": {
			"access_token": "legacy_access",
			"refresh_token": "legacy_refresh"
		}
	}`
	if err := os.WriteFile(legacyFile, []byte(legacyJSON), 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	legacyStatus, err := store.GetAccountStatus(legacyEmail)
	if err != nil {
		t.Fatalf("GetAccountStatus on legacy file failed: %v", err)
	}
	if legacyStatus != "on" {
		t.Errorf("expected legacy file to default to 'on', got: %q", legacyStatus)
	}

	// SaveAccount on legacy file without status should preserve "on"
	if err := store.SaveAccount(legacyEmail, token); err != nil {
		t.Fatalf("SaveAccount on legacy account failed: %v", err)
	}
	legacyStatus, err = store.GetAccountStatus(legacyEmail)
	if err != nil {
		t.Fatalf("GetAccountStatus after save on legacy file failed: %v", err)
	}
	if legacyStatus != "on" {
		t.Errorf("expected legacy file after save to remain 'on', got: %q", legacyStatus)
	}
}

func TestAccountStore_Status_ErrorAndEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create AccountStore: %v", err)
	}

	// 1. Non-existent account
	_, err = store.GetAccountStatus("nonexistent@example.com")
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound for non-existent account, got: %v", err)
	}

	err = store.SetAccountStatus("nonexistent@example.com", "off")
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound when setting status on non-existent account, got: %v", err)
	}

	// 2. Empty or invalid email
	_, err = store.GetAccountStatus("")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail for empty email, got: %v", err)
	}

	err = store.SetAccountStatus("", "off")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("expected ErrInvalidEmail for empty email on SetAccountStatus, got: %v", err)
	}

	// 3. Normalization of status strings
	email := "norm@example.com"
	token := &model.TokenData{AccessToken: "tok"}
	if err := store.SaveAccount(email, token); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// Case and whitespace insensitive " OFF " -> "off"
	if err := store.SetAccountStatus(email, "  OFF  "); err != nil {
		t.Fatalf("SetAccountStatus failed: %v", err)
	}
	st, _ := store.GetAccountStatus(email)
	if st != "off" {
		t.Errorf("expected 'off' after normalization, got: %q", st)
	}

	// Any non-"off" value (e.g. "random", "disabled", "true") -> "on"
	if err := store.SetAccountStatus(email, "something_else"); err != nil {
		t.Fatalf("SetAccountStatus failed: %v", err)
	}
	st, _ = store.GetAccountStatus(email)
	if st != "on" {
		t.Errorf("expected 'on' for arbitrary non-off string, got: %q", st)
	}

	// 4. Corrupted file
	corruptEmail := "corrupt@example.com"
	corruptFile := filepath.Join(tempDir, "corrupt@example.com.json")
	if err := os.WriteFile(corruptFile, []byte("{ bad json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	_, err = store.GetAccountStatus(corruptEmail)
	if !errors.Is(err, ErrCorruptedStoreFile) {
		t.Errorf("expected ErrCorruptedStoreFile for corrupt json, got: %v", err)
	}

	err = store.SetAccountStatus(corruptEmail, "off")
	if !errors.Is(err, ErrCorruptedStoreFile) {
		t.Errorf("expected ErrCorruptedStoreFile for SetAccountStatus on corrupt json, got: %v", err)
	}
}
