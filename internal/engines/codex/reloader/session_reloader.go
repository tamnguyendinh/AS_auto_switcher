package reloader

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"as-auto-switcher/internal/core/credentialio"
	"as-auto-switcher/internal/engines/codex/connector"
	"as-auto-switcher/internal/engines/codex/model"
	"as-auto-switcher/internal/engines/codex/store"
)

const (
	DefaultSettlingDuration = 8 * time.Second
	DefaultRetryDelay       = 10 * time.Second
)

var (
	ErrEmptyTargetEmail               = errors.New("target email is empty")
	ErrTargetAccountNotFound          = store.ErrAccountNotFound
	ErrTargetUnauthenticated          = errors.New("target credential is incomplete")
	ErrTargetAlreadyActive            = errors.New("target already active")
	ErrDesktopStopTimeout             = errors.New("Desktop did not stop")
	ErrDesktopStartFailed             = errors.New("Desktop did not start")
	ErrDesktopVerificationFailed      = errors.New("Desktop identity verification failed")
	ErrExternalAuthModified           = errors.New("external credential change detected")
	ErrSwitchFailedRolledBack         = errors.New("switch failed; prior session restored")
	ErrRecoveryRequired               = model.ErrRecoveryRequired
	ErrSwitchInProgress               = model.ErrMutationBusy
	ErrUnsupportedDesktopVerification = errors.New("UnsupportedDesktopVerification: cannot verify Codex Desktop identity and busy state")
	ErrUnsupportedAuthBackend         = errors.New("UnsupportedAuthBackend: Desktop file credential backend is not verified")
	ErrDesktopBusy                    = errors.New("Desktop busy or busy state unknown")
	ErrAuthHomeBusy                   = errors.New("AuthHomeBusy: credential writers are not quiescent")
)

type SwitchStage string

const (
	StagePreflight        SwitchStage = "Preflight"
	StagePrepared         SwitchStage = "Prepared"
	StageStopDesktop      SwitchStage = "StopDesktop"
	StageCaptureFinalA    SwitchStage = "CaptureFinalA"
	StageValidateTarget   SwitchStage = "ValidateTarget"
	StageInstallTarget    SwitchStage = "InstallTarget"
	StageStartDesktop     SwitchStage = "StartDesktop"
	StageVerifyDesktop    SwitchStage = "VerifyDesktop"
	StageCommit           SwitchStage = "Commit"
	StageRolledBack       SwitchStage = "RolledBack"
	StageRecoveryRequired SwitchStage = "RecoveryRequired"
)

// DesktopState must come from a process-bound observer, never from auth.json.
// ProcessKey binds executable/package, PID and creation time. WritersSafe means
// all writers were classified and none outside the selected Desktop share home.
type DesktopState struct {
	Running                      bool
	PID                          int
	ProcessKey                   string
	AuthPath                     string
	Backend                      string
	LauncherVerified             bool
	BusyKnown, Busy, WritersSafe bool
	Email, AccountID             string
	Ready                        bool
}
type DesktopLifecycle interface {
	IsRunning(context.Context) (bool, int, error)
	Stop(context.Context, int) error
	Start(context.Context) (int, error)
	VerifyIdentity(context.Context, int, string) (bool, error)
}

// VerifiedDesktop is required before any active-auth or process mutation.
// Stop must gracefully stop the bound tree, Start must preserve its lifetime
// beyond the launch context, and Quiescent must inspect all relevant writers.
type VerifiedDesktop interface {
	DesktopLifecycle
	Probe(context.Context) (DesktopState, error)
	Quiescent(context.Context, string) error
	VerifyAccount(context.Context, int, string, string) (bool, error)
}
type SessionReloader interface {
	SwitchSession(string) error
	GetActiveSessionEmail() (string, error)
	RecoverPendingJournals() error
}
type SessionReloaderConfig struct {
	Store            store.AccountStore
	ActiveAuthPath   string
	JournalDir       string
	BackupDir        string
	Desktop          DesktopLifecycle
	NowFunc          func() time.Time
	Coordinator      *model.Coordinator
	Refresher        connector.TokenRefresher
	SettlingDuration time.Duration
	StatusReporter   func(string)
	RetryDelay       time.Duration
}
type JournalEntry struct {
	OperationID                                       string
	Stage                                             SwitchStage
	AuthPath                                          string
	TargetEmail, TargetAccountID, TargetGeneration    string
	TargetRevision                                    uint64
	TargetFingerprint                                 string
	PriorActiveEmail, PriorAccountID, PriorGeneration string
	PriorHadAuth                                      bool
	PriorAuthBackup, PriorFingerprint                 string
	PriorAppRunning                                   bool
	PriorProcessKey                                   string
	CreatedAt, UpdatedAt                              time.Time
}
type Binding struct {
	Email, AccountID, Generation, AuthPath string
}
type sessionReloader struct {
	store            store.AccountStore
	desktop          DesktopLifecycle
	gate             *model.Coordinator
	activeAuthPath   string
	journalDir       string
	backupDir        string
	bindingPath      string
	nowFunc          func() time.Time
	refresher        connector.TokenRefresher
	settlingDuration time.Duration
	statusReporter   func(string)
	retryDelay       time.Duration
}

func NewSessionReloader(cfg SessionReloaderConfig) SessionReloader {
	if cfg.NowFunc == nil {
		cfg.NowFunc = time.Now
	}
	if cfg.Coordinator == nil {
		cfg.Coordinator = store.Coordinator(cfg.Store)
	}
	root := ""
	if cfg.Store != nil {
		root = cfg.Store.GetStoreDir()
	}
	if cfg.JournalDir == "" {
		cfg.JournalDir = filepath.Join(root, ".runtime", "journal")
	}
	if cfg.BackupDir == "" {
		cfg.BackupDir = filepath.Join(root, ".runtime", "backups")
	}
	if strings.TrimSpace(cfg.ActiveAuthPath) == "" {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			cfg.ActiveAuthPath = filepath.Join(home, ".codex", "auth.json")
		} else {
			cfg.ActiveAuthPath = filepath.Join(os.Getenv("USERPROFILE"), ".codex", "auth.json")
		}
	}
	if cfg.Desktop == nil {
		cfg.Desktop = NewWindowsChatGPTLifecycle(cfg.ActiveAuthPath)
	}
	if cfg.Refresher == nil {
		cfg.Refresher = connector.NewTokenRefresher()
	}
	settling := cfg.SettlingDuration
	if cfg.SettlingDuration < 0 {
		settling = 0
	} else if cfg.SettlingDuration == 0 {
		settling = DefaultSettlingDuration
	}

	retryDelay := cfg.RetryDelay
	if cfg.RetryDelay < 0 {
		retryDelay = 0
	} else if cfg.RetryDelay == 0 {
		retryDelay = DefaultRetryDelay
	}

	return &sessionReloader{
		store:            cfg.Store,
		desktop:          cfg.Desktop,
		gate:             cfg.Coordinator,
		activeAuthPath:   cfg.ActiveAuthPath,
		journalDir:       cfg.JournalDir,
		backupDir:        cfg.BackupDir,
		bindingPath:      filepath.Join(root, ".runtime", "binding.json"),
		nowFunc:          cfg.NowFunc,
		refresher:        cfg.Refresher,
		settlingDuration: settling,
		statusReporter:   cfg.StatusReporter,
		retryDelay:       retryDelay,
	}
}
func (r *sessionReloader) probe() (VerifiedDesktop, DesktopState, error) {
	dt, ok := r.desktop.(VerifiedDesktop)
	if !ok {
		return nil, DesktopState{}, ErrUnsupportedDesktopVerification
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	state, err := dt.Probe(ctx)
	if err != nil {
		return nil, state, err
	}
	if state.Backend != "file" || !filepath.IsAbs(state.AuthPath) {
		return nil, state, ErrUnsupportedAuthBackend
	}
	if r.activeAuthPath != "" && !strings.EqualFold(filepath.Clean(r.activeAuthPath), filepath.Clean(state.AuthPath)) {
		return nil, state, ErrUnsupportedAuthBackend
	}
	if err := credentialio.CheckPath(state.AuthPath); err != nil {
		return nil, state, err
	}
	if !state.WritersSafe {
		return nil, state, ErrAuthHomeBusy
	}
	if !state.LauncherVerified {
		return nil, state, ErrDesktopStartFailed
	}
	if state.Running && (state.PID <= 0 || state.ProcessKey == "" || !state.Ready || state.Email == "" || state.AccountID == "") {
		return nil, state, ErrUnsupportedDesktopVerification
	}
	if !state.BusyKnown || state.Busy {
		return nil, state, ErrDesktopBusy
	}
	return dt, state, nil
}
func (r *sessionReloader) GetActiveSessionEmail() (string, error) {
	_, s, err := r.probe()
	if err != nil {
		return "", err
	}
	if !s.Running {
		return "", nil
	}
	return strings.ToLower(s.Email), nil
}
func (r *sessionReloader) readBinding() (Binding, error) {
	var b Binding
	raw, err := credentialio.Read(r.bindingPath)
	if os.IsNotExist(err) {
		return b, nil
	}
	if err != nil {
		return b, err
	}
	if json.Unmarshal(raw, &b) != nil {
		return b, ErrRecoveryRequired
	}
	return b, nil
}
func (r *sessionReloader) writeBinding(b Binding) error {
	raw, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return credentialio.Write(r.bindingPath, raw)
}

// Reconcile imports only a verified file backend. Changed existing credentials
// require a matching persisted ownership generation; timestamps cannot decide.
func (r *sessionReloader) Reconcile() error {
	r.gate.Lock()
	defer r.gate.Unlock()
	if err := r.gate.Check(""); err != nil {
		return err
	}
	_, s, err := r.probe()
	if err != nil {
		return err
	}
	raw, err := credentialio.Read(s.AuthPath)
	if os.IsNotExist(err) && !s.Running {
		return nil
	}
	if err != nil {
		return err
	}
	email, id, err := model.ValidateAuth(raw)
	if err != nil {
		return err
	}
	if s.Running && (email != strings.ToLower(s.Email) || id != s.AccountID) {
		return ErrExternalAuthModified
	}
	b, err := r.readBinding()
	if err != nil {
		return err
	}
	existing, err := r.store.GetAccount(email)
	if err != nil && !errors.Is(err, store.ErrAccountNotFound) {
		return err
	}
	generation := model.CredentialGeneration(raw)
	if existing != nil {
		// TRAP FIX: Nếu tài khoản trong kho có Generation rỗng, MergeActiveCredential sẽ chặn với ErrReauthGenerationProtected.
		// Ta cập nhật trực tiếp qua SaveAccount để khởi tạo generation hợp lệ mà không làm mất Revision hay Status.
		if existing.Generation == "" {
			env := &model.CodexAccountEnvelope{
				Email:      email,
				AuthData:   raw,
				Generation: generation,
				Revision:   existing.Revision,
				Status:     existing.Status,
			}
			if err := r.store.SaveAccount(email, env); err != nil {
				return err
			}
			if s.Running {
				return r.writeBinding(Binding{Email: email, AccountID: id, Generation: generation, AuthPath: s.AuthPath})
			}
			return nil
		}

		if model.CredentialGeneration(existing.AuthData) == model.CredentialGeneration(raw) {
			if s.Running {
				return r.writeBinding(Binding{Email: email, AccountID: id, Generation: existing.Generation, AuthPath: s.AuthPath})
			}
			return nil
		}

		if b.Email != email || b.AccountID != id || b.AuthPath != s.AuthPath || b.Generation != existing.Generation {
			return ErrExternalAuthModified
		}
		generation = b.Generation
	}
	if err := r.store.MergeActiveCredential(email, &model.CodexAccountEnvelope{Email: email, AuthData: raw, Generation: generation}); err != nil {
		return err
	}
	if s.Running {
		return r.writeBinding(Binding{Email: email, AccountID: id, Generation: generation, AuthPath: s.AuthPath})
	}
	return nil
}

func (r *sessionReloader) SwitchSession(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ErrEmptyTargetEmail
	}
	r.gate.Lock()
	defer r.gate.Unlock()
	if err := r.gate.Check(""); err != nil {
		return err
	}
	if r.store == nil {
		return ErrTargetAccountNotFound
	}
	// Pending operational state blocks new work even before startup recovery.
	entries, err := os.ReadDir(r.journalDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) > 0 {
		r.gate.RecoveryRequired = true
		return ErrRecoveryRequired
	}
	target, err := r.store.GetAccount(email)
	if err != nil {
		return err
	}
	targetEmail, targetID, err := model.ValidateAuth(target.AuthData)
	if err != nil || targetEmail != email {
		return ErrTargetUnauthenticated
	}
	dt, state, err := r.probe()
	if err != nil {
		return err
	}
	b, err := r.readBinding()
	if err != nil {
		return err
	}

	// =========================================================================
	// BƯỚC 1.1: KIỂM TRA NO-OP TRƯỚC TIÊN (ZERO IMPACT KHI ĐÃ ĐÚNG TÀI KHOẢN)
	// =========================================================================
	if state.Running && strings.EqualFold(state.Email, email) && state.AccountID == targetID &&
		b.Email == email && b.AccountID == targetID && b.Generation == target.Generation && b.AuthPath == state.AuthPath {
		return nil
	}

	// =========================================================================
	// BƯỚC 1.2: TIỀN KIỂM TRA 55 PHÚT & RETRY MẤT MẠNG 3 LẦN X 10S
	// =========================================================================
	if r.refresher != nil {
		shouldPurge, _ := r.refresher.ShouldPurge(target.AuthData, r.nowFunc())
		if shouldPurge {
			var freshAuth json.RawMessage
			var refreshErr error
			const maxRetries = 3

			for attempt := 1; attempt <= maxRetries; attempt++ {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				freshAuth, refreshErr = r.refresher.RefreshToken(ctx, target.AuthData)
				cancel()
				if refreshErr == nil {
					break
				}

				// Nếu là lỗi tài khoản dứt khoát (HTTP 400/401 revoke/expired) -> Dừng ngay không retry, giữ nguyên App A
				if errors.Is(refreshErr, connector.ErrAuthTerminal) {
					return fmt.Errorf("tài khoản %s đã hết hạn hoặc bị thu hồi (vui lòng đăng nhập lại qua OAuth): %w", email, refreshErr)
				}

				// Nếu là lỗi mất mạng / timeout / server error -> Thử lại tối đa 3 lần, mỗi lần cách nhau retryDelay
				if attempt < maxRetries {
					if r.statusReporter != nil {
						r.statusReporter(fmt.Sprintf("⚠️ Mất kết nối mạng. Đang thử làm mới token lần %d/%d sau 10s...", attempt, maxRetries))
					}
					time.Sleep(r.retryDelay)
				}
			}

			if refreshErr != nil {
				// Sau 3 lần retry thất bại vì mất mạng: Dừng dứt khoát, giữ nguyên 100% App A
				if r.statusReporter != nil {
					r.statusReporter("❌ Không thể đổi tài khoản vì mất mạng. Vui lòng thử lại sau.")
				}
				return fmt.Errorf("không thể đổi tài khoản vì mất mạng, vui lòng thử lại sau: %w", refreshErr)
			}

			// Ghi đè token mới vào kho bằng SaveAccount (CAS safe, tự tăng revision, tự sinh generation)
			target.AuthData = freshAuth
			if saveErr := r.store.SaveAccount(email, target); saveErr != nil {
				return fmt.Errorf("không thể lưu token đã làm mới vào kho: %w", saveErr)
			}
		}
	}

	// =========================================================================
	// BƯỚC 1.3: ĐỌC LẠI SNAPSHOT TARGET MỚI NHẤT ĐỂ TẠO JOURNAL ENTRY CHUẨN XÁC
	// =========================================================================
	target, err = r.store.GetAccount(email)
	if err != nil {
		return err
	}
	targetEmail, targetID, err = model.ValidateAuth(target.AuthData)
	if err != nil || targetEmail != email {
		return ErrTargetUnauthenticated
	}

	if err := credentialio.PrivateDir(r.journalDir); err != nil {
		return err
	}
	if err := credentialio.PrivateDir(r.backupDir); err != nil {
		return err
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return err
	}
	j := &JournalEntry{
		OperationID:       hex.EncodeToString(id[:]),
		Stage:             StagePrepared,
		AuthPath:          state.AuthPath,
		TargetEmail:       email,
		TargetAccountID:   targetID,
		TargetGeneration:  target.Generation,
		TargetRevision:    target.Revision,
		TargetFingerprint: hashBytes(target.AuthData),
		PriorAppRunning:   state.Running,
		PriorProcessKey:   state.ProcessKey,
		PriorActiveEmail:  strings.ToLower(state.Email),
		PriorAccountID:    state.AccountID,
		CreatedAt:         r.nowFunc(),
	}
	if err := r.saveJournal(j); err != nil {
		return err
	}
	if state.Running {
		if err := r.stage(j, StageStopDesktop); err != nil {
			return r.recovery(j)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err = dt.Stop(ctx, state.PID)
		cancel()
		if err != nil {
			return r.recovery(j)
		}
	}
	if err := r.quiescent(dt, j.AuthPath); err != nil {
		return r.recovery(j)
	}
	if err := r.stage(j, StageCaptureFinalA); err != nil {
		return r.recovery(j)
	}
	raw, err := credentialio.Read(j.AuthPath)
	if err != nil && !os.IsNotExist(err) {
		return r.recovery(j)
	}
	if err == nil {
		oldEmail, oldID, e := model.ValidateAuth(raw)
		if e != nil || (state.Running && (oldEmail != strings.ToLower(state.Email) || oldID != state.AccountID)) {
			return r.recovery(j)
		}
		j.PriorHadAuth = true
		j.PriorActiveEmail = oldEmail
		j.PriorAccountID = oldID
		j.PriorFingerprint = hashBytes(raw)
		j.PriorAuthBackup = filepath.Join(r.backupDir, j.OperationID+"_a.json")
		if err := credentialio.Write(j.PriorAuthBackup, raw); err != nil {
			return r.abort(dt, j)
		}
		existing, e := r.store.GetAccount(oldEmail)
		if e != nil && !errors.Is(e, store.ErrAccountNotFound) {
			return r.abort(dt, j)
		}
		if existing == nil {
			j.PriorGeneration = model.CredentialGeneration(raw)
			if e := r.store.MergeActiveCredential(oldEmail, &model.CodexAccountEnvelope{Email: oldEmail, AuthData: raw, Generation: j.PriorGeneration}); e != nil {
				return r.abort(dt, j)
			}
		} else if model.CredentialGeneration(existing.AuthData) == model.CredentialGeneration(raw) {
			j.PriorGeneration = existing.Generation
		} else if b.Email == oldEmail && b.AccountID == oldID && b.AuthPath == j.AuthPath {
			j.PriorGeneration = b.Generation
			if b.Generation == existing.Generation {
				if e := r.store.MergeActiveCredential(oldEmail, &model.CodexAccountEnvelope{Email: oldEmail, AuthData: raw, Generation: b.Generation}); e != nil {
					return r.abort(dt, j)
				}
			} // newer store generation is preserved; final A remains in private backup.
		} else {
			return r.recovery(j)
		}
	}
	if err := r.saveJournal(j); err != nil {
		return r.recovery(j)
	}
	if err := r.stage(j, StageValidateTarget); err != nil {
		return r.recovery(j)
	}
	fresh, err := r.store.GetAccount(email)
	if err != nil || fresh.Revision != j.TargetRevision || hashBytes(fresh.AuthData) != j.TargetFingerprint {
		return r.abort(dt, j)
	}
	if err := r.quiescent(dt, j.AuthPath); err != nil {
		return r.recovery(j)
	}
	if !matchesFile(j.AuthPath, j.PriorHadAuth, j.PriorFingerprint) {
		return r.recovery(j)
	}
	if err := r.stage(j, StageInstallTarget); err != nil {
		return r.recovery(j)
	}
	if err := credentialio.Write(j.AuthPath, fresh.AuthData); err != nil {
		if !matchesFile(j.AuthPath, j.PriorHadAuth, j.PriorFingerprint) {
			return r.recovery(j)
		}
		return r.abort(dt, j)
	}
	if err := r.stage(j, StageStartDesktop); err != nil {
		return r.recovery(j)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	pid, startErr := dt.Start(ctx)
	cancel()
	if startErr != nil {
		return r.rollback(dt, j, pid)
	}
	if r.settlingDuration > 0 {
		time.Sleep(r.settlingDuration)
	}
	if err := r.stage(j, StageVerifyDesktop); err != nil {
		return r.recovery(j)
	}
	if !r.verify(dt, pid, email, targetID) {
		return r.rollback(dt, j, pid)
	}
	if err := r.writeBinding(Binding{Email: email, AccountID: targetID, Generation: target.Generation, AuthPath: j.AuthPath}); err != nil {
		return r.recovery(j)
	}
	if err := r.stage(j, StageCommit); err != nil {
		return r.recovery(j)
	}
	return r.cleanup(j)
}
func (r *sessionReloader) quiescent(dt VerifiedDesktop, path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return dt.Quiescent(ctx, path)
}
func (r *sessionReloader) verify(dt VerifiedDesktop, pid int, email, id string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ok, err := dt.VerifyAccount(ctx, pid, email, id)
	return err == nil && ok
}
func (r *sessionReloader) recovery(j *JournalEntry) error {
	r.gate.RecoveryRequired = true
	// Keep the last durable stage; never overwrite it with an uninformative stage.
	return ErrRecoveryRequired
}
func (r *sessionReloader) abort(dt VerifiedDesktop, j *JournalEntry) error {
	if j.PriorAppRunning {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		pid, err := dt.Start(ctx)
		cancel()
		if err != nil || !r.verify(dt, pid, j.PriorActiveEmail, j.PriorAccountID) {
			return r.recovery(j)
		}
	}
	if err := r.stage(j, StageRolledBack); err != nil {
		return r.recovery(j)
	}
	if err := r.cleanup(j); err != nil {
		return r.recovery(j)
	}
	return ErrSwitchFailedRolledBack
}
func (r *sessionReloader) rollback(dt VerifiedDesktop, j *JournalEntry, pid int) error {
	if pid > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := dt.Stop(ctx, pid)
		cancel()
		if err != nil {
			return r.recovery(j)
		}
	}
	if err := r.quiescent(dt, j.AuthPath); err != nil {
		return r.recovery(j)
	}
	// Preserve final B only when its lineage is unambiguous under a verified
	// transaction; unknown file changes are never overwritten.
	current, err := credentialio.Read(j.AuthPath)
	if err != nil {
		return r.recovery(j)
	}
	email, id, e := model.ValidateAuth(current)
	if e != nil || email != j.TargetEmail || id != j.TargetAccountID {
		return r.recovery(j)
	}
	currentHash := hashBytes(current)
	if currentHash != j.TargetFingerprint {
		if pid <= 0 {
			return r.recovery(j)
		}
		if err := credentialio.Write(filepath.Join(r.backupDir, j.OperationID+"_b.json"), current); err != nil {
			return r.recovery(j)
		}
		if err := r.store.MergeActiveCredential(email, &model.CodexAccountEnvelope{Email: email, AuthData: current, Generation: j.TargetGeneration}); err != nil {
			return r.recovery(j)
		}
	}
	if !matchesFile(j.AuthPath, true, currentHash) {
		return r.recovery(j)
	}
	if j.PriorHadAuth {
		if j.PriorAuthBackup == "" {
			return r.recovery(j)
		}
		raw, err := credentialio.Read(j.PriorAuthBackup)
		if err != nil || hashBytes(raw) != j.PriorFingerprint {
			return r.recovery(j)
		}
		if err := credentialio.Write(j.AuthPath, raw); err != nil {
			return r.recovery(j)
		}
	} else if err := os.Remove(j.AuthPath); err != nil {
		return r.recovery(j)
	}
	if j.PriorHadAuth {
		if err := r.writeBinding(Binding{Email: j.PriorActiveEmail, AccountID: j.PriorAccountID, Generation: j.PriorGeneration, AuthPath: j.AuthPath}); err != nil {
			return r.recovery(j)
		}
	} else if err := os.Remove(r.bindingPath); err != nil && !os.IsNotExist(err) {
		return r.recovery(j)
	}
	return r.abort(dt, j)
}
func (r *sessionReloader) RecoverPendingJournals() error {
	r.gate.Lock()
	defer r.gate.Unlock()
	entries, err := os.ReadDir(r.journalDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return r.recovery(nil)
	}
	if len(entries) == 0 {
		return nil
	}
	if len(entries) != 1 {
		return r.recovery(nil)
	}
	raw, err := credentialio.Read(filepath.Join(r.journalDir, entries[0].Name()))
	if err != nil {
		return r.recovery(nil)
	}
	var j JournalEntry
	if json.Unmarshal(raw, &j) != nil || j.OperationID == "" || entries[0].Name() != j.OperationID+".json" || filepath.Base(j.OperationID) != j.OperationID {
		return r.recovery(nil)
	}
	if j.PriorAuthBackup != "" && j.PriorAuthBackup != filepath.Join(r.backupDir, j.OperationID+"_a.json") {
		return r.recovery(nil)
	}
	switch j.Stage {
	case StagePrepared, StageStopDesktop, StageCaptureFinalA, StageValidateTarget, StageInstallTarget, StageStartDesktop, StageVerifyDesktop, StageCommit, StageRolledBack:
	default:
		return r.recovery(&j)
	}
	dt, state, err := r.probe()
	if err != nil || state.AuthPath != j.AuthPath {
		return r.recovery(&j)
	}
	if j.Stage == StageCommit || j.Stage == StageRolledBack {
		if err := r.cleanup(&j); err != nil {
			return r.recovery(&j)
		}
		r.gate.RecoveryRequired = false
		return nil
	}
	targetInstalled := matchesFile(j.AuthPath, true, j.TargetFingerprint)
	if targetInstalled && state.Running && r.verify(dt, state.PID, j.TargetEmail, j.TargetAccountID) {
		if err := r.writeBinding(Binding{Email: j.TargetEmail, AccountID: j.TargetAccountID, Generation: j.TargetGeneration, AuthPath: j.AuthPath}); err != nil {
			return r.recovery(&j)
		}
		if err := r.stage(&j, StageCommit); err != nil {
			return r.recovery(&j)
		}
		if err := r.cleanup(&j); err != nil {
			return r.recovery(&j)
		}
		r.gate.RecoveryRequired = false
		return nil
	}
	// A crash before the final capture cannot prove ownership from a file alone.
	if j.Stage == StagePrepared || j.Stage == StageStopDesktop || j.Stage == StageCaptureFinalA {
		return r.recovery(&j)
	}
	if matchesFile(j.AuthPath, j.PriorHadAuth, j.PriorFingerprint) {
		if state.Running {
			if !r.verify(dt, state.PID, j.PriorActiveEmail, j.PriorAccountID) {
				return r.recovery(&j)
			}
			if err := r.stage(&j, StageRolledBack); err != nil {
				return r.recovery(&j)
			}
			if err := r.cleanup(&j); err != nil {
				return r.recovery(&j)
			}
		} else if err := r.abort(dt, &j); !errors.Is(err, ErrSwitchFailedRolledBack) {
			return err
		}
		r.gate.RecoveryRequired = false
		return nil
	}
	if !targetInstalled {
		return r.recovery(&j)
	} // external/unknown; do not restore.
	pid := 0
	if state.Running {
		if !r.verify(dt, state.PID, j.TargetEmail, j.TargetAccountID) {
			return r.recovery(&j)
		}
		pid = state.PID
	}
	err = r.rollback(dt, &j, pid)
	if errors.Is(err, ErrSwitchFailedRolledBack) {
		r.gate.RecoveryRequired = false
		return nil
	}
	return err
}
func (r *sessionReloader) saveJournal(j *JournalEntry) error {
	j.UpdatedAt = r.nowFunc()
	raw, err := json.Marshal(j)
	if err != nil {
		return err
	}
	return credentialio.Write(filepath.Join(r.journalDir, j.OperationID+".json"), raw)
}
func (r *sessionReloader) stage(j *JournalEntry, stage SwitchStage) error {
	previous := j.Stage
	j.Stage = stage
	if err := r.saveJournal(j); err != nil {
		j.Stage = previous
		return err
	}
	return nil
}
func (r *sessionReloader) cleanup(j *JournalEntry) error {
	for _, path := range []string{j.PriorAuthBackup, filepath.Join(r.backupDir, j.OperationID+"_b.json")} {
		if path != "" {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return os.Remove(filepath.Join(r.journalDir, j.OperationID+".json"))
}
func matchesFile(path string, exists bool, fingerprint string) bool {
	raw, err := credentialio.Read(path)
	if !exists {
		return os.IsNotExist(err)
	}
	return err == nil && hashBytes(raw) == fingerprint
}
func hashBytes(raw []byte) string                   { return fmt.Sprintf("%x", sha256.Sum256(raw)) }
func atomicWriteFile(path string, raw []byte) error { return credentialio.Write(path, raw) }
