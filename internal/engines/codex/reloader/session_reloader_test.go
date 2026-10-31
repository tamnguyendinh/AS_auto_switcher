package reloader

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"as-auto-switcher/internal/core/credentialio"
	"as-auto-switcher/internal/engines/codex/connector"
	"as-auto-switcher/internal/engines/codex/model"
	"as-auto-switcher/internal/engines/codex/store"
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

func makeSampleAuthData(email, accountID string, extraKeyVal ...interface{}) json.RawMessage {
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
			key, ok := extraKeyVal[i].(string)
			if ok {
				payload[key] = extraKeyVal[i+1]
			}
		}
	}
	data, _ := json.Marshal(payload)
	return data
}

// This adapter is an isolated transaction fixture, never evidence of Desktop capability.
type observedDesktop struct {
	state         DesktopState
	stops, starts int
	stopFn        func(int) error
	startFn       func() (int, error)
	verifyFn      func(int, string, string) (bool, error)
	quietFn       func(string) error
}

func (d *observedDesktop) Probe(context.Context) (DesktopState, error) { return d.state, nil }
func (d *observedDesktop) IsRunning(context.Context) (bool, int, error) {
	return d.state.Running, d.state.PID, nil
}
func (d *observedDesktop) Stop(_ context.Context, pid int) error {
	d.stops++
	if d.stopFn != nil {
		if err := d.stopFn(pid); err != nil {
			return err
		}
	}
	d.state.Running = false
	return nil
}
func (d *observedDesktop) Start(context.Context) (int, error) {
	d.starts++
	if d.startFn != nil {
		return d.startFn()
	}
	d.state.Running = true
	return 2000, nil
}
func (d *observedDesktop) VerifyIdentity(ctx context.Context, pid int, email string) (bool, error) {
	return d.VerifyAccount(ctx, pid, email, d.state.AccountID)
}
func (d *observedDesktop) VerifyAccount(_ context.Context, pid int, email, id string) (bool, error) {
	if d.verifyFn != nil {
		return d.verifyFn(pid, email, id)
	}
	return true, nil
}
func (d *observedDesktop) Quiescent(_ context.Context, path string) error {
	if d.quietFn != nil {
		return d.quietFn(path)
	}
	if d.state.Running {
		return ErrAuthHomeBusy
	}
	return nil
}
type mockTokenRefresher struct {
	shouldPurgeFn func(authData json.RawMessage, now time.Time) (bool, error)
	refreshFn     func(ctx context.Context, authData json.RawMessage) (json.RawMessage, error)
	refreshCalls  int
}

func (m *mockTokenRefresher) ShouldPurge(authData json.RawMessage, now time.Time) (bool, error) {
	if m.shouldPurgeFn != nil {
		return m.shouldPurgeFn(authData, now)
	}
	return false, nil
}

func (m *mockTokenRefresher) RefreshToken(ctx context.Context, authData json.RawMessage) (json.RawMessage, error) {
	m.refreshCalls++
	if m.refreshFn != nil {
		return m.refreshFn(ctx, authData)
	}
	return authData, nil
}

func setupTransaction(t *testing.T) (*sessionReloader, store.AccountStore, *observedDesktop, []byte) {
	t.Helper()
	root := t.TempDir()
	st, err := store.NewAccountStore(filepath.Join(root, "store"))
	if err != nil {
		t.Fatal(err)
	}
	a := makeSampleAuthData("a@example.com", "id-a")
	b := makeSampleAuthData("b@example.com", "id-b")
	for _, e := range []*model.CodexAccountEnvelope{{Email: "a@example.com", Status: "off", AuthData: a, Generation: "generation-a"}, {Email: "b@example.com", Status: "off", AuthData: b, Generation: "generation-b"}} {
		if err := st.SaveAccount(e.Email, e); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "active", "auth.json")
	if err := credentialio.Write(path, a); err != nil {
		t.Fatal(err)
	}
	dt := &observedDesktop{state: DesktopState{Running: true, PID: 1000, ProcessKey: "fixture:1000:creation", AuthPath: path, Backend: "file", LauncherVerified: true, BusyKnown: true, WritersSafe: true, Email: "a@example.com", AccountID: "id-a", Ready: true}}
	r := NewSessionReloader(SessionReloaderConfig{
		Store:            st,
		ActiveAuthPath:   path,
		Desktop:          dt,
		SettlingDuration: -1,
		RetryDelay:       1 * time.Millisecond,
	}).(*sessionReloader)
	if err := r.writeBinding(Binding{Email: "a@example.com", AccountID: "id-a", Generation: "generation-a", AuthPath: path}); err != nil {
		t.Fatal(err)
	}
	return r, st, dt, a
}
func assertAuth(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || hashBytes(got) != hashBytes(want) {
		t.Fatalf("credential changed unexpectedly (read error %v)", err)
	}
}
func assertNoJournal(t *testing.T, r *sessionReloader) {
	t.Helper()
	files, err := os.ReadDir(r.journalDir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatal("journal not cleaned")
	}
}
func TestSessionReloader_DefaultCapabilityHasNoSideEffects(t *testing.T) {
	r, st, dt, a := setupTransaction(t)
	r.desktop = NewDefaultWindowsDesktopLifecycle(r.activeAuthPath)
	if err := r.SwitchSession("b@example.com"); !errors.Is(err, ErrUnsupportedDesktopVerification) {
		t.Fatal(err)
	}
	assertAuth(t, r.activeAuthPath, a)
	assertNoJournal(t, r)
	if email, err := r.GetActiveSessionEmail(); email != "" || !errors.Is(err, ErrUnsupportedDesktopVerification) {
		t.Fatal("file treated as verified Desktop")
	}
	if dt.stops != 0 || dt.starts != 0 {
		t.Fatal("process side effects")
	}
	if !st.HasAccount("a@example.com") {
		t.Fatal("store changed")
	}
}
func TestSessionReloader_PreflightRejectsUnknownAndBusy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		alter func(*DesktopState)
		want  error
	}{
		{"busy", func(s *DesktopState) { s.Busy = true }, ErrDesktopBusy},
		{"busy-unknown", func(s *DesktopState) { s.BusyKnown = false }, ErrDesktopBusy},
		{"outside-writer", func(s *DesktopState) { s.WritersSafe = false }, ErrAuthHomeBusy},
		{"keyring", func(s *DesktopState) { s.Backend = "keyring" }, ErrUnsupportedAuthBackend},
		{"unbound-process", func(s *DesktopState) { s.ProcessKey = "" }, ErrUnsupportedDesktopVerification},
		{"launcher-missing", func(s *DesktopState) { s.LauncherVerified = false }, ErrDesktopStartFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, dt, a := setupTransaction(t)
			tc.alter(&dt.state)
			if err := r.SwitchSession("b@example.com"); !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
			assertAuth(t, r.activeAuthPath, a)
			if dt.stops+dt.starts != 0 {
				t.Fatal("preflight mutated process")
			}
		})
	}
}
func TestSessionReloader_SwitchCommitsOnlyVerifiedTarget(t *testing.T) {
	r, st, dt, _ := setupTransaction(t)
	dt.verifyFn = func(pid int, email, id string) (bool, error) {
		if pid != 2000 || email != "b@example.com" || id != "id-b" {
			t.Fatal("wrong process/identity verified")
		}
		return true, nil
	}
	if err := r.SwitchSession("b@example.com"); err != nil {
		t.Fatal(err)
	}
	b, _ := st.GetAccount("b@example.com")
	assertAuth(t, r.activeAuthPath, b.AuthData)
	binding, err := r.readBinding()
	if err != nil || binding.Generation != b.Generation {
		t.Fatal("binding missing")
	}
	if b.Status != "off" || dt.starts != 1 || dt.stops != 1 {
		t.Fatal("switch preference/process sequence")
	}
	assertNoJournal(t, r)
}
func TestSessionReloader_NoOpRequiresVerifiedGenerationAndRunning(t *testing.T) {
	r, st, dt, _ := setupTransaction(t)
	if err := r.SwitchSession("a@example.com"); err != nil || dt.stops+dt.starts != 0 {
		t.Fatal("verified same generation not no-op")
	}
	newA, _ := st.GetAccount("a@example.com")
	newA.Generation = "reauth-new"
	newA.AuthData = makeSampleAuthData(newA.Email, "id-a", "reauth", "new")
	if err := st.SaveAccount(newA.Email, newA); err != nil {
		t.Fatal(err)
	}
	if err := r.SwitchSession(newA.Email); err != nil {
		t.Fatal(err)
	}
	if dt.starts != 1 || dt.stops != 1 {
		t.Fatal("reauth generation skipped restart")
	}
	dt.state.Running = false
	if err := r.SwitchSession(newA.Email); err != nil {
		t.Fatal(err)
	}
	if dt.starts != 2 {
		t.Fatal("closed Desktop not opened")
	}
}
func TestSessionReloader_FinalCapturePreservesRefreshAndPendingReauth(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprint(pending), func(t *testing.T) {
			r, st, dt, _ := setupTransaction(t)
			fresh := makeSampleAuthData("a@example.com", "id-a", "runtime_refresh", "new")
			if pending {
				a, _ := st.GetAccount("a@example.com")
				a.Generation = "reauth-new"
				a.AuthData = makeSampleAuthData(a.Email, "id-a", "reauth", "new")
				if err := st.SaveAccount(a.Email, a); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := st.GetAccount("a@example.com")
			dt.stopFn = func(int) error { return credentialio.Write(r.activeAuthPath, fresh) }
			if err := r.SwitchSession("b@example.com"); err != nil {
				t.Fatal(err)
			}
			saved, _ := st.GetAccount("a@example.com")
			want := fresh
			if pending {
				want = before.AuthData
			}
			if model.CredentialGeneration(saved.AuthData) != model.CredentialGeneration(want) {
				t.Fatal("latest/pending credential lost")
			}
			if saved.Status != "off" {
				t.Fatal("capture reset OFF")
			}
		})
	}
}
func TestSessionReloader_BackupFailureNeverInstallsTarget(t *testing.T) {
	r, _, dt, a := setupTransaction(t)
	// Make backup location invalid after preflight but before final capture.
	dt.stopFn = func(int) error {
		if err := os.Remove(r.backupDir); err != nil {
			return err
		}
		return os.WriteFile(r.backupDir, []byte("blocked"), 0600)
	}
	err := r.SwitchSession("b@example.com")
	if !errors.Is(err, ErrSwitchFailedRolledBack) && !errors.Is(err, ErrRecoveryRequired) {
		t.Fatal(err)
	}
	assertAuth(t, r.activeAuthPath, a)
}
func TestSessionReloader_JournalFailureBlocksInstall(t *testing.T) {
	r, _, dt, a := setupTransaction(t)
	dt.stopFn = func(int) error {
		entries, err := os.ReadDir(r.journalDir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			p := filepath.Join(r.journalDir, e.Name())
			if err := os.Remove(p); err != nil {
				return err
			}
			if err := os.Mkdir(p, 0700); err != nil {
				return err
			}
		}
		return nil
	}
	if err := r.SwitchSession("b@example.com"); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatal(err)
	}
	assertAuth(t, r.activeAuthPath, a)
	if err := r.SwitchSession("b@example.com"); !errors.Is(err, ErrRecoveryRequired) {
		t.Fatal("recovery did not block mutation")
	}
}
func TestSessionReloader_RollbackStopsWriterBeforeRestore(t *testing.T) {
	for _, stopFails := range []bool{false, true} {
		t.Run(fmt.Sprint(stopFails), func(t *testing.T) {
			r, st, dt, a := setupTransaction(t)
			dt.verifyFn = func(_ int, email, id string) (bool, error) { return email == "a@example.com", nil }
			dt.stopFn = func(pid int) error {
				if pid == 2000 && stopFails {
					return errors.New("writer alive")
				}
				return nil
			}
			err := r.SwitchSession("b@example.com")
			if stopFails {
				if !errors.Is(err, ErrRecoveryRequired) {
					t.Fatal(err)
				}
				b, _ := st.GetAccount("b@example.com")
				assertAuth(t, r.activeAuthPath, b.AuthData)
			} else {
				if !errors.Is(err, ErrSwitchFailedRolledBack) {
					t.Fatal(err)
				}
				assertAuth(t, r.activeAuthPath, a)
				assertNoJournal(t, r)
			}
		})
	}
}
func TestSessionReloader_RollbackPreservesBRefresh(t *testing.T) {
	r, st, dt, a := setupTransaction(t)
	refreshed := makeSampleAuthData("b@example.com", "id-b", "refresh", "latest")
	dt.verifyFn = func(_ int, email, id string) (bool, error) {
		if email == "b@example.com" {
			if err := credentialio.Write(r.activeAuthPath, refreshed); err != nil {
				t.Fatal(err)
			}
			return false, nil
		}
		return true, nil
	}
	if err := r.SwitchSession("b@example.com"); !errors.Is(err, ErrSwitchFailedRolledBack) {
		t.Fatal(err)
	}
	assertAuth(t, r.activeAuthPath, a)
	b, _ := st.GetAccount("b@example.com")
	if model.CredentialGeneration(b.AuthData) != model.CredentialGeneration(refreshed) {
		t.Fatal("B refresh lost")
	}
}
func TestSessionReloader_RestoreAbsenceOnStartFailure(t *testing.T) {
	r, _, dt, _ := setupTransaction(t)
	dt.state.Running = false
	_ = os.Remove(r.activeAuthPath)
	dt.startFn = func() (int, error) { return 0, ErrDesktopStartFailed }
	if err := r.SwitchSession("b@example.com"); !errors.Is(err, ErrSwitchFailedRolledBack) {
		t.Fatal(err)
	}
	if _, err := os.Stat(r.activeAuthPath); !os.IsNotExist(err) {
		t.Fatal("initial absence not restored")
	}
	assertNoJournal(t, r)
}
func TestSessionReloader_RecoveryRejectsExternalChangeAndCorruptJournal(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprint(corrupt), func(t *testing.T) {
			r, st, dt, a := setupTransaction(t)
			dt.state.Running = false
			if err := credentialio.PrivateDir(r.journalDir); err != nil {
				t.Fatal(err)
			}
			b, _ := st.GetAccount("b@example.com")
			j := &JournalEntry{OperationID: "fixture", Stage: StageInstallTarget, AuthPath: r.activeAuthPath, PriorHadAuth: true, PriorFingerprint: hashBytes(a), TargetFingerprint: hashBytes(b.AuthData), TargetEmail: b.Email, TargetAccountID: "id-b"}
			if err := r.saveJournal(j); err != nil {
				t.Fatal(err)
			}
			if corrupt {
				if err := credentialio.Write(filepath.Join(r.journalDir, "fixture.json"), []byte("broken")); err != nil {
					t.Fatal(err)
				}
			}
			external := makeSampleAuthData("external@example.com", "id-x")
			if err := credentialio.Write(r.activeAuthPath, external); err != nil {
				t.Fatal(err)
			}
			if err := r.RecoverPendingJournals(); !errors.Is(err, ErrRecoveryRequired) {
				t.Fatal(err)
			}
			assertAuth(t, r.activeAuthPath, external)
			if dt.starts+dt.stops != 0 {
				t.Fatal("unknown process mutated")
			}
		})
	}
}
func TestSessionReloader_RecoveryCommitsVerifiedInstalledTarget(t *testing.T) {
	r, st, dt, a := setupTransaction(t)
	b, _ := st.GetAccount("b@example.com")
	_ = credentialio.PrivateDir(r.journalDir)
	j := &JournalEntry{OperationID: "fixture", Stage: StageInstallTarget, AuthPath: r.activeAuthPath, PriorHadAuth: true, PriorFingerprint: hashBytes(a), TargetFingerprint: hashBytes(b.AuthData), TargetEmail: b.Email, TargetAccountID: "id-b", TargetGeneration: b.Generation}
	if err := r.saveJournal(j); err != nil {
		t.Fatal(err)
	}
	if err := credentialio.Write(r.activeAuthPath, b.AuthData); err != nil {
		t.Fatal(err)
	}
	dt.state.Email = b.Email
	dt.state.AccountID = "id-b"
	if err := r.RecoverPendingJournals(); err != nil {
		t.Fatal(err)
	}
	assertAuth(t, r.activeAuthPath, b.AuthData)
	assertNoJournal(t, r)
}

func TestSessionReloader_FailSafeProtection(t *testing.T) {
	r, _, dt, a := setupTransaction(t)
	mock := &mockTokenRefresher{
		shouldPurgeFn: func(_ json.RawMessage, _ time.Time) (bool, error) {
			return true, nil
		},
		refreshFn: func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
			return nil, fmt.Errorf("%w: invalid_grant (HTTP 401)", connector.ErrAuthTerminal)
		},
	}
	r.refresher = mock

	err := r.SwitchSession("b@example.com")
	if err == nil || !errors.Is(err, connector.ErrAuthTerminal) {
		t.Fatalf("expected ErrAuthTerminal error, got %v", err)
	}
	if dt.stops != 0 || dt.starts != 0 {
		t.Fatalf("app A was mutated: stops=%d, starts=%d", dt.stops, dt.starts)
	}
	assertAuth(t, r.activeAuthPath, a)
	assertNoJournal(t, r)
	if mock.refreshCalls != 1 {
		t.Fatalf("expected exactly 1 call without retries, got %d", mock.refreshCalls)
	}
}

func TestSessionReloader_FastExecutionWithZeroSettlingDuration(t *testing.T) {
	r, st, dt, _ := setupTransaction(t)
	if r.settlingDuration != 0 {
		t.Fatalf("expected settling duration 0, got %v", r.settlingDuration)
	}
	start := time.Now()
	if err := r.SwitchSession("b@example.com"); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected fast execution (< 500ms), took %v", elapsed)
	}
	b, _ := st.GetAccount("b@example.com")
	assertAuth(t, r.activeAuthPath, b.AuthData)
	if dt.starts != 1 || dt.stops != 1 {
		t.Fatal("switch failed to run desktop")
	}
}

func TestSessionReloader_SettlingWindowTiming(t *testing.T) {
	r, _, _, _ := setupTransaction(t)
	r.settlingDuration = 80 * time.Millisecond
	start := time.Now()
	if err := r.SwitchSession("b@example.com"); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed < 80*time.Millisecond {
		t.Fatalf("expected settling delay >= 80ms, got %v", elapsed)
	}
}

func TestSessionReloader_SaveAccountRefreshedTarget(t *testing.T) {
	r, st, _, _ := setupTransaction(t)

	// Set b in store with Generation: "" and Revision: 1
	targetPath, err := store.SafeAccountPath(st.GetStoreDir(), "b@example.com")
	if err != nil {
		t.Fatal(err)
	}
	bOld := makeSampleAuthData("b@example.com", "id-b")
	envB := &model.CodexAccountEnvelope{
		Email:      "b@example.com",
		AuthData:   bOld,
		Generation: "",
		Revision:   1,
		Status:     "off",
		SavedAt:    time.Now(),
	}
	rawB, err := json.Marshal(envB)
	if err != nil {
		t.Fatal(err)
	}
	if err := credentialio.Write(targetPath, rawB); err != nil {
		t.Fatal(err)
	}

	// Verify before switch that b has empty generation and revision 1
	bBefore, err := st.GetAccount("b@example.com")
	if err != nil || bBefore.Generation != "" {
		t.Fatalf("expected empty generation before, got %v", bBefore.Generation)
	}
	if bBefore.Revision != 1 {
		t.Fatalf("expected revision 1, got %d", bBefore.Revision)
	}

	refreshedData := makeSampleAuthData("b@example.com", "id-b", "refreshed", "true")
	mock := &mockTokenRefresher{
		shouldPurgeFn: func(_ json.RawMessage, _ time.Time) (bool, error) {
			return true, nil
		},
		refreshFn: func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
			return refreshedData, nil
		},
	}
	r.refresher = mock

	if err := r.SwitchSession("b@example.com"); err != nil {
		t.Fatalf("switch failed: %v", err)
	}

	bAfter, err := st.GetAccount("b@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if bAfter.Revision != 2 {
		t.Fatalf("expected revision incremented to 2, got %d", bAfter.Revision)
	}
	if bAfter.Generation == "" {
		t.Fatal("expected generation to be auto-generated, got empty")
	}
	expectedGen := model.CredentialGeneration(refreshedData)
	if bAfter.Generation != expectedGen {
		t.Fatalf("expected generation %s, got %s", expectedGen, bAfter.Generation)
	}
	assertAuth(t, r.activeAuthPath, bAfter.AuthData)
	assertNoJournal(t, r)
}

func TestSessionReloader_DefaultRefresherFallback(t *testing.T) {
	root := t.TempDir()
	st, err := store.NewAccountStore(filepath.Join(root, "store"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "active", "auth.json")

	reloader := NewSessionReloader(SessionReloaderConfig{
		Store:          st,
		ActiveAuthPath: path,
	})
	r, ok := reloader.(*sessionReloader)
	if !ok {
		t.Fatalf("expected *sessionReloader, got %T", reloader)
	}
	if r.refresher == nil {
		t.Fatal("expected default refresher, got nil")
	}
	if r.settlingDuration != 8*time.Second {
		t.Fatalf("expected default settling duration 8s, got %v", r.settlingDuration)
	}
	if r.retryDelay != 10*time.Second {
		t.Fatalf("expected default retry delay 10s, got %v", r.retryDelay)
	}
	if _, ok := r.desktop.(*WindowsChatGPTLifecycle); !ok {
		t.Fatalf("expected *WindowsChatGPTLifecycle, got %T", r.desktop)
	}
}

func TestSessionReloader_ReconcileEmptyGeneration(t *testing.T) {
	r, st, _, a := setupTransaction(t)

	// Overwrite a@example.com in store with empty generation
	targetPath, err := store.SafeAccountPath(st.GetStoreDir(), "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	envA := &model.CodexAccountEnvelope{
		Email:      "a@example.com",
		AuthData:   a,
		Generation: "",
		Revision:   1,
		Status:     "on",
		SavedAt:    time.Now(),
	}
	rawA, err := json.Marshal(envA)
	if err != nil {
		t.Fatal(err)
	}
	if err := credentialio.Write(targetPath, rawA); err != nil {
		t.Fatal(err)
	}

	// Verify before Reconcile that generation is empty
	before, err := st.GetAccount("a@example.com")
	if err != nil || before.Generation != "" {
		t.Fatalf("expected empty generation before, got %v", before.Generation)
	}

	// Reconcile should repair the empty generation without throwing ErrReauthGenerationProtected
	if err := r.Reconcile(); err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	after, err := st.GetAccount("a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if after.Generation == "" {
		t.Fatal("expected non-empty generation after Reconcile")
	}
	expectedGen := model.CredentialGeneration(a)
	if after.Generation != expectedGen {
		t.Fatalf("expected generation %s, got %s", expectedGen, after.Generation)
	}

	b, err := r.readBinding()
	if err != nil {
		t.Fatal(err)
	}
	if b.Generation != expectedGen {
		t.Fatalf("expected binding generation %s, got %s", expectedGen, b.Generation)
	}
}

func TestSessionReloader_NetworkFailureRetry3Times10Seconds(t *testing.T) {
	r, _, dt, a := setupTransaction(t)
	r.retryDelay = 1 * time.Millisecond

	var statusMessages []string
	r.statusReporter = func(msg string) {
		statusMessages = append(statusMessages, msg)
	}

	mock := &mockTokenRefresher{
		shouldPurgeFn: func(_ json.RawMessage, _ time.Time) (bool, error) {
			return true, nil
		},
		refreshFn: func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
			return nil, fmt.Errorf("%w: connection timeout", connector.ErrNetworkLoss)
		},
	}
	r.refresher = mock

	err := r.SwitchSession("b@example.com")
	if err == nil {
		t.Fatal("expected network failure error, got nil")
	}
	if !strings.Contains(err.Error(), "không thể đổi tài khoản vì mất mạng, vui lòng thử lại sau") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if !errors.Is(err, connector.ErrNetworkLoss) {
		t.Fatalf("expected wrapped ErrNetworkLoss, got %v", err)
	}
	if mock.refreshCalls != 3 {
		t.Fatalf("expected exactly 3 refresh attempts, got %d", mock.refreshCalls)
	}
	if len(statusMessages) != 3 {
		t.Fatalf("expected 3 status messages, got %d: %v", len(statusMessages), statusMessages)
	}
	wantMsg0 := "⚠️ Mất kết nối mạng. Đang thử làm mới token lần 1/3 sau 10s..."
	wantMsg1 := "⚠️ Mất kết nối mạng. Đang thử làm mới token lần 2/3 sau 10s..."
	wantMsg2 := "❌ Không thể đổi tài khoản vì mất mạng. Vui lòng thử lại sau."
	if statusMessages[0] != wantMsg0 {
		t.Errorf("msg[0] = %q, want %q", statusMessages[0], wantMsg0)
	}
	if statusMessages[1] != wantMsg1 {
		t.Errorf("msg[1] = %q, want %q", statusMessages[1], wantMsg1)
	}
	if statusMessages[2] != wantMsg2 {
		t.Errorf("msg[2] = %q, want %q", statusMessages[2], wantMsg2)
	}

	if dt.stops != 0 || dt.starts != 0 {
		t.Fatalf("app A was modified on network failure: stops=%d, starts=%d", dt.stops, dt.starts)
	}
	assertAuth(t, r.activeAuthPath, a)
	assertNoJournal(t, r)
}
