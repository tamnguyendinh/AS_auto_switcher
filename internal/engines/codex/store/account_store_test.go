package store

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"as-auto-switcher/internal/engines/codex/model"
)

func makeTestCodexJWT(email, name string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := map[string]interface{}{
		"email":          email,
		"name":           name,
		"email_verified": true,
		"exp":            time.Now().Add(1 * time.Hour).Unix(),
		"iat":            time.Now().Unix(),
		"iss":            "https://auth.openai.com",
		"sub":            "user-12345",
	}
	claimsBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)
	sig := base64.RawURLEncoding.EncodeToString([]byte("signature_bytes"))
	return fmt.Sprintf("%s.%s.%s", header, payload, sig)
}

func makeSampleAuthData(email, accountID string, extraKeyVal ...string) json.RawMessage {
	idToken := makeTestCodexJWT(email, "Test User")
	payload := map[string]interface{}{
		"OPENAI_API_KEY": nil,
		"auth_mode":      "chatgpt",
		"last_refresh":   time.Now().UTC().Format(time.RFC3339),
		"tokens": map[string]interface{}{
			"access_token":  "at_test_" + accountID,
			"refresh_token": "rt_test_" + accountID,
			"id_token":      idToken,
			"account_id":    accountID,
		},
	}
	for i := 0; i < len(extraKeyVal); i += 2 {
		if i+1 < len(extraKeyVal) {
			payload[extraKeyVal[i]] = extraKeyVal[i+1]
		}
	}
	data, _ := json.Marshal(payload)
	return data
}

func TestCodexAccountStore_RoundTrip_SaveGetDeleteHas(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	email := "Codex.User+Test@openai.com"
	authData := makeSampleAuthData(email, "acc-001", "unknown_custom_field", "preserve_me_100%")

	envelope := &model.CodexAccountEnvelope{
		Email:      email,
		Status:     "on",
		PlanType:   "pro",
		AuthData:   authData,
		Generation: "gen_1",
	}

	// 1. Initial save
	if err := store.SaveAccount(email, envelope); err != nil {
		t.Fatalf("SaveAccount failed: %v", err)
	}

	// 2. HasAccount check
	if !store.HasAccount(email) {
		t.Fatalf("expected HasAccount(%s) to be true", email)
	}
	if !store.HasAccount("codex.user+test@openai.com") {
		t.Fatalf("expected case-insensitive HasAccount to be true")
	}

	// 3. GetAccount check
	got, err := store.GetAccount(email)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}

	if got.Email != "codex.user+test@openai.com" {
		t.Errorf("expected normalized email, got %s", got.Email)
	}
	if got.Status != "on" {
		t.Errorf("expected status 'on', got %s", got.Status)
	}
	if got.PlanType != "pro" {
		t.Errorf("expected plan_type 'pro', got %s", got.PlanType)
	}
	if got.Revision != 1 {
		t.Errorf("expected revision 1, got %d", got.Revision)
	}
	if got.Generation != "gen_1" {
		t.Errorf("expected generation 'gen_1', got %s", got.Generation)
	}

	// 4. Verify round-trip preservation of unknown fields in AuthData
	var authMap map[string]interface{}
	if err := json.Unmarshal(got.AuthData, &authMap); err != nil {
		t.Fatalf("unmarshal auth_data: %v", err)
	}
	if authMap["unknown_custom_field"] != "preserve_me_100%" {
		t.Errorf("unknown field lost in roundtrip; got %v", authMap["unknown_custom_field"])
	}

	// 5. DeleteAccount
	if err := store.DeleteAccount(email); err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	if store.HasAccount(email) {
		t.Errorf("expected account to be deleted")
	}

	// 6. GetAccount after delete should return ErrAccountNotFound
	_, err = store.GetAccount(email)
	if !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound, got %v", err)
	}

	// 7. Delete non-existent account
	if err := store.DeleteAccount("nonexistent@example.com"); !errors.Is(err, ErrAccountNotFound) {
		t.Errorf("expected ErrAccountNotFound deleting non-existent, got %v", err)
	}
}

func TestCodexAccountStore_ListAccounts_StableSort_And_IgnoreSubdirs(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	// Create subdirectories that must be ignored
	runtimeDir := filepath.Join(tempDir, ".runtime", "oauth", "attempt-1")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "candidate.json"), []byte(`{}`), 0600); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	// Create fake leftover temp files that must be ignored
	if err := os.WriteFile(filepath.Join(tempDir, "codex_tmp_12345.json"), []byte(`{}`), 0600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "acc_tmp_999.json"), []byte(`{}`), 0600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "junk.tmp"), []byte(`{}`), 0600); err != nil {
		t.Fatalf("write junk: %v", err)
	}

	// Save accounts with distinct CreatedAt
	t1 := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	accB := &model.CodexAccountEnvelope{
		Email:     "b@openai.com",
		Status:    "on",
		PlanType:  "plus",
		CreatedAt: t2,
		AuthData:  makeSampleAuthData("b@openai.com", "acc-b"),
	}
	accA := &model.CodexAccountEnvelope{
		Email:     "a@openai.com",
		Status:    "on",
		PlanType:  "pro",
		CreatedAt: t1,
		AuthData:  makeSampleAuthData("a@openai.com", "acc-a"),
	}
	accC := &model.CodexAccountEnvelope{
		Email:     "c@openai.com",
		Status:    "off",
		PlanType:  "team",
		CreatedAt: t3,
		AuthData:  makeSampleAuthData("c@openai.com", "acc-c"),
	}

	_ = store.SaveAccount("b@openai.com", accB)
	_ = store.SaveAccount("a@openai.com", accA)
	_ = store.SaveAccount("c@openai.com", accC)

	list, err := store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts failed: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 accounts, got %d", len(list))
	}

	if list[0].Email != "a@openai.com" || list[1].Email != "b@openai.com" || list[2].Email != "c@openai.com" {
		t.Errorf("list not sorted stably by CreatedAt: %s, %s, %s", list[0].Email, list[1].Email, list[2].Email)
	}
}

func TestCodexAccountStore_AtomicReplace_NoTmpArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	for i := 0; i < 20; i++ {
		email := fmt.Sprintf("user%d@openai.com", i)
		env := &model.CodexAccountEnvelope{
			Email:    email,
			Status:   "on",
			PlanType: "pro",
			AuthData: makeSampleAuthData(email, fmt.Sprintf("acc-%d", i)),
		}
		if err := store.SaveAccount(email, env); err != nil {
			t.Fatalf("SaveAccount #%d failed: %v", i, err)
		}
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "codex_tmp_") || strings.HasSuffix(name, ".tmp") {
			t.Errorf("found leftover atomic temp file: %s", name)
		}
	}
}

func TestCodexAccountStore_StatusPreservation_And_SetStatus(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	email := "preserve_status@openai.com"
	env := &model.CodexAccountEnvelope{
		Email:    email,
		Status:   "off",
		PlanType: "plus",
		AuthData: makeSampleAuthData(email, "acc-pres"),
	}

	// 1. Initial save with status "off"
	if err := store.SaveAccount(email, env); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	st, err := store.GetAccountStatus(email)
	if err != nil || st != "off" {
		t.Fatalf("expected status 'off', got %s, err: %v", st, err)
	}

	// 2. Reauth / refresh update with empty status must preserve "off"
	currentForUpdate, _ := store.GetAccount(email)
	updateEnv := &model.CodexAccountEnvelope{Revision: currentForUpdate.Revision,
		Email:    email,
		Status:   "", // empty status should preserve existing "off"
		PlanType: "plus",
		AuthData: makeSampleAuthData(email, "acc-pres", "updated", "true"),
	}
	if err := store.SaveAccount(email, updateEnv); err != nil {
		t.Fatalf("reauth save failed: %v", err)
	}

	st, err = store.GetAccountStatus(email)
	if err != nil || st != "off" {
		t.Fatalf("expected status 'off' to be preserved, got %s", st)
	}

	// 3. SetAccountStatus to "on"
	if err := store.SetAccountStatus(email, "on"); err != nil {
		t.Fatalf("SetAccountStatus('on') failed: %v", err)
	}
	st, err = store.GetAccountStatus(email)
	if err != nil || st != "on" {
		t.Fatalf("expected status 'on', got %s", st)
	}

	// 4. SetAccountStatus to "off"
	if err := store.SetAccountStatus(email, "off"); err != nil {
		t.Fatalf("SetAccountStatus('off') failed: %v", err)
	}
	st, err = store.GetAccountStatus(email)
	if err != nil || st != "off" {
		t.Fatalf("expected status 'off', got %s", st)
	}

	// 5. Check revision incremented
	got, _ := store.GetAccount(email)
	if got.Revision < 3 {
		t.Errorf("expected revision >= 3 after updates, got %d", got.Revision)
	}
}

func TestCodexAccountStore_CAS_RevisionConflict(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	email := "cas_test@openai.com"
	env := &model.CodexAccountEnvelope{
		Email:    email,
		Status:   "on",
		PlanType: "pro",
		AuthData: makeSampleAuthData(email, "acc-cas"),
	}
	if err := store.SaveAccount(email, env); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	current, err := store.GetAccount(email)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if current.Revision != 1 {
		t.Fatalf("expected initial revision 1, got %d", current.Revision)
	}

	// Attempt save with stale revision (e.g. revision 99)
	staleEnv := current.Clone()
	staleEnv.Revision = 99
	err = store.SaveAccount(email, staleEnv)
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("expected ErrRevisionConflict, got %v", err)
	}

	// Save with correct revision (1) must succeed and bump to 2
	staleEnv.Revision = 1
	if err := store.SaveAccount(email, staleEnv); err != nil {
		t.Fatalf("save with correct revision failed: %v", err)
	}

	updated, _ := store.GetAccount(email)
	if updated.Revision != 2 {
		t.Errorf("expected bumped revision 2, got %d", updated.Revision)
	}
}

func TestCodexAccountStore_IdentityConflict(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	email := "identity_conflict@openai.com"
	env := &model.CodexAccountEnvelope{
		Email:    email,
		Status:   "on",
		PlanType: "pro",
		AuthData: makeSampleAuthData(email, "account_id_workspace_A"),
	}
	if err := store.SaveAccount(email, env); err != nil {
		t.Fatalf("save account A: %v", err)
	}

	// Try saving same email but different account_id
	conflictEnv := &model.CodexAccountEnvelope{Revision: 1,
		Email:    email,
		Status:   "on",
		PlanType: "pro",
		AuthData: makeSampleAuthData(email, "account_id_workspace_B"),
	}
	err = store.SaveAccount(email, conflictEnv)
	if !errors.Is(err, ErrIdentityConflict) {
		t.Fatalf("expected ErrIdentityConflict when account_id changes, got %v", err)
	}
}

func TestCodexAccountStore_FilenameSanitization_And_ReservedNames(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"user@openai.com", "user@openai.com.json"},
		{"User.Name+Tag@Domain.COM", "user.name+tag@domain.com.json"},
		{"CON@openai.com", "con@openai.com.json"},
		{"con", "_con.json"},
		{"prn", "_prn.json"},
		{"aux", "_aux.json"},
		{"nul", "_nul.json"},
		{"com1", "_com1.json"},
		{"lpt9", "_lpt9.json"},
		{"../../evil@hack.com", "evil@hack.com.json"},
		{"user...dots@domain.com", "user.dots@domain.com.json"},
		{"   spaced@domain.com   ", "spaced@domain.com.json"},
	}

	for _, c := range cases {
		got := SanitizeFilename(c.input)
		if got != c.expected {
			t.Errorf("SanitizeFilename(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}

func TestCodexAccountStore_PathTraversalGuards(t *testing.T) {
	tempDir := t.TempDir()

	traversals := []string{
		"../outside@domain.com",
		"../../etc/passwd",
		"..\\..\\windows\\system32",
		"",
		"   ",
	}

	for _, badEmail := range traversals {
		_, err := SafeAccountPath(tempDir, badEmail)
		if err == nil {
			t.Errorf("expected error for traversal/invalid email %q, got nil", badEmail)
		}
	}
}

func TestCodexAccountStore_MergeActiveCredential(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	email := "active_merge@openai.com"

	// 1. Merge when account does not exist (Auto-Bootstrap)
	newEnv := &model.CodexAccountEnvelope{
		Email:      email,
		PlanType:   "unknown",
		Generation: "gen_active_1",
		AuthData:   makeSampleAuthData(email, "acc-merge-1"),
	}
	if err := store.MergeActiveCredential(email, newEnv); err != nil {
		t.Fatalf("MergeActiveCredential (new) failed: %v", err)
	}

	got, err := store.GetAccount(email)
	if err != nil {
		t.Fatalf("GetAccount failed: %v", err)
	}
	if got.Status != "on" {
		t.Errorf("expected default 'on', got %s", got.Status)
	}
	if got.Revision != 1 {
		t.Errorf("expected revision 1, got %d", got.Revision)
	}

	// Update to status "off"
	_ = store.SetAccountStatus(email, "off")

	// 2. Merge existing active credential: must preserve "off" and order
	updatedActive := &model.CodexAccountEnvelope{
		Email:      email,
		PlanType:   "unknown",
		Generation: "gen_active_1",
		AuthData:   makeSampleAuthData(email, "acc-merge-1", "refreshed_by_desktop", "true"),
	}
	if err := store.MergeActiveCredential(email, updatedActive); err != nil {
		t.Fatalf("MergeActiveCredential (existing) failed: %v", err)
	}

	got2, _ := store.GetAccount(email)
	if got2.Status != "off" {
		t.Errorf("expected status 'off' to be preserved, got %s", got2.Status)
	}
	if got2.Revision < 3 {
		t.Errorf("expected revision >= 3, got %d", got2.Revision)
	}

	// 3. Reauth generation protection:
	// If store has a pending reauth generation ("reauth_xxx"), merge from older active generation must be blocked!
	pendingReauth := got2.Clone()
	pendingReauth.Generation = "reauth_attempt_999"
	_ = store.SaveAccount(email, pendingReauth)

	staleActive := &model.CodexAccountEnvelope{
		Email:      email,
		Generation: "gen_active_old",
		AuthData:   makeSampleAuthData(email, "acc-merge-1"),
	}
	err = store.MergeActiveCredential(email, staleActive)
	if !errors.Is(err, ErrReauthGenerationProtected) {
		t.Fatalf("expected ErrReauthGenerationProtected, got %v", err)
	}
}

func TestCodexAccountStore_ConcurrentAccess_RaceDetector(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewAccountStore(tempDir)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	const workerCount = 20
	const iterations = 10
	var wg sync.WaitGroup

	// Populate initial accounts
	for i := 0; i < 5; i++ {
		email := fmt.Sprintf("concurrent%d@openai.com", i)
		env := &model.CodexAccountEnvelope{
			Email:    email,
			Status:   "on",
			PlanType: "pro",
			AuthData: makeSampleAuthData(email, fmt.Sprintf("acc-%d", i)),
		}
		_ = store.SaveAccount(email, env)
	}

	wg.Add(workerCount)
	for w := 0; w < workerCount; w++ {
		workerID := w
		go func() {
			defer wg.Done()
			for it := 0; it < iterations; it++ {
				email := fmt.Sprintf("concurrent%d@openai.com", (workerID+it)%5)

				switch (workerID + it) % 4 {
				case 0:
					_, _ = store.GetAccount(email)
				case 1:
					_, _ = store.ListAccounts()
				case 2:
					_ = store.SetAccountStatus(email, "off")
					_ = store.SetAccountStatus(email, "on")
				case 3:
					env := &model.CodexAccountEnvelope{
						Email:    email,
						Status:   "on",
						PlanType: "pro",
						AuthData: makeSampleAuthData(email, fmt.Sprintf("acc-%d", (workerID+it)%5)),
					}
					_ = store.SaveAccount(email, env)
				}
			}
		}()
	}

	wg.Wait()

	list, err := store.ListAccounts()
	if err != nil {
		t.Fatalf("ListAccounts after concurrency failed: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("expected 5 accounts, got %d", len(list))
	}
}

func TestCodexModel_JWT_ParseAndExtractEmail(t *testing.T) {
	// 1. Valid JWT
	jwt := makeTestCodexJWT("User.Valid@example.com", "Valid User")
	email, err := model.ExtractEmailFromJWT(jwt)
	if err != nil {
		t.Fatalf("ExtractEmailFromJWT failed: %v", err)
	}
	if email != "user.valid@example.com" {
		t.Errorf("expected normalized email, got %s", email)
	}

	claims, err := model.ParseJWTClaims(jwt)
	if err != nil {
		t.Fatalf("ParseJWTClaims failed: %v", err)
	}
	if claims.Name != "Valid User" {
		t.Errorf("expected name 'Valid User', got %s", claims.Name)
	}
	if !claims.EmailVerified {
		t.Errorf("expected email_verified = true")
	}

	// 2. Malformed JWT
	_, err = model.ExtractEmailFromJWT("not.a.valid.jwt")
	if !errors.Is(err, model.ErrInvalidJWT) {
		t.Errorf("expected ErrInvalidJWT for invalid base64, got %v", err)
	}

	_, err = model.ExtractEmailFromJWT("truncated")
	if !errors.Is(err, model.ErrInvalidJWT) {
		t.Errorf("expected ErrInvalidJWT for truncated string, got %v", err)
	}

	// 3. Missing email claim
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256"}`))
	noEmailPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"name":"No Email"}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	noEmailJWT := fmt.Sprintf("%s.%s.%s", header, noEmailPayload, sig)

	_, err = model.ExtractEmailFromJWT(noEmailJWT)
	if !errors.Is(err, model.ErrEmailNotFoundInJWT) {
		t.Errorf("expected ErrEmailNotFoundInJWT, got %v", err)
	}
}
