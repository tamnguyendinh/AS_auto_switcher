package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"as-auto-switcher/internal/core/credentialio"
	"as-auto-switcher/internal/engines/codex/connector"
	"as-auto-switcher/internal/engines/codex/model"
	"as-auto-switcher/internal/engines/codex/store"
)

var (
	ErrOAuthBusy             = model.ErrMutationBusy
	ErrNoActiveAttempt       = errors.New("no active OAuth attempt")
	ErrAttemptNotFound       = errors.New("OAuth attempt not found")
	ErrAlreadyCommitted      = errors.New("OAuth attempt already committed")
	ErrOAuthTimeout          = errors.New("OAuth login timed out")
	ErrOAuthCanceled         = errors.New("OAuth login canceled")
	ErrReauthMismatch        = errors.New("OAuth identity mismatch")
	ErrCannotResolveEmail    = errors.New("cannot verify account identity")
	ErrCandidateAuthNotFound = errors.New("candidate credential not found")
	ErrSaveRetryExpired      = errors.New("save retry deadline expired")
	ErrManagerClosed         = errors.New("OAuth manager closed")
)

type LoginResult struct {
	AttemptID string
	Email     string
	AccountID string
	Stage     string
	Err       error
	Retryable bool
}
type AttemptSnapshot struct {
	LoginResult
	AuthURL  string
	Sequence uint64
}
type OAuthAttempt struct {
	workerDone                         chan struct{}
	cleanupMu                          sync.Mutex
	AttemptID                          string
	Mode                               string
	ExpectedEmail                      string
	ExpectedAccountID                  string
	ExpectedRevision                   uint64
	CandidateAuthPath                  string
	CandidateHome                      string
	LoginID                            string
	AuthURL                            string
	StartTime                          time.Time
	DoneChan                           chan LoginResult
	CancelFn                           context.CancelFunc
	mu                                 sync.Mutex
	stage                              string
	committed, canceled, failed        bool
	client                             connector.OAuthClient
	rawAuthData                        []byte
	planType, resolvedMail, resolvedID string
	lastError                          error
	saveFailedAt                       time.Time
	retryTimer                         *time.Timer
	latest                             AttemptSnapshot
	subscribers                        map[chan AttemptSnapshot]struct{}
}

func (a *OAuthAttempt) GetStage() string          { a.mu.Lock(); defer a.mu.Unlock(); return a.stage }
func (a *OAuthAttempt) IsCommitted() bool         { a.mu.Lock(); defer a.mu.Unlock(); return a.committed }
func (a *OAuthAttempt) IsCanceled() bool          { a.mu.Lock(); defer a.mu.Unlock(); return a.canceled }
func (a *OAuthAttempt) IsFailed() bool            { a.mu.Lock(); defer a.mu.Unlock(); return a.failed }
func (a *OAuthAttempt) Snapshot() AttemptSnapshot { a.mu.Lock(); defer a.mu.Unlock(); return a.latest }

// Subscribe replays state; a disposed/slow view never blocks the worker.
func (a *OAuthAttempt) Subscribe() (<-chan AttemptSnapshot, func()) {
	a.mu.Lock()
	if a.subscribers == nil {
		a.subscribers = make(map[chan AttemptSnapshot]struct{})
	}
	ch := make(chan AttemptSnapshot, 1)
	a.subscribers[ch] = struct{}{}
	ch <- a.latest
	a.mu.Unlock()
	var once sync.Once
	return ch, func() { once.Do(func() { a.mu.Lock(); delete(a.subscribers, ch); a.mu.Unlock() }) }
}
func (a *OAuthAttempt) publishLocked(result LoginResult) {
	a.stage, a.lastError = result.Stage, result.Err
	a.latest = AttemptSnapshot{LoginResult: result, AuthURL: a.AuthURL, Sequence: a.latest.Sequence + 1}
	for ch := range a.subscribers {
		select {
		case <-ch:
		default:
		}
		ch <- a.latest
	}
	if result.Stage == "success" || result.Err != nil {
		select {
		case a.DoneChan <- result:
		default:
		}
	}
}

type OAuthManager interface {
	BeginLogin(context.Context, string, string, string) (*OAuthAttempt, error)
	CancelActiveLogin() error
	IsAttemptActive() bool
	RetrySave(context.Context, string) error
	GetActiveAttempt() *OAuthAttempt
	Close() error
}
type OAuthManagerConfig struct {
	Store                        store.AccountStore
	RuntimeBaseDir               string
	CodexExePath                 string
	CodexExeFinder               func() (string, error)
	ClientFactory                func(connector.OAuthClientConfig) (connector.OAuthClient, error)
	AttemptTimeout, SaveRetryTTL time.Duration
	Coordinator                  *model.Coordinator
}
type oauthManager struct {
	store                        store.AccountStore
	runtimeBaseDir, codexExePath string
	codexExeFinder               func() (string, error)
	clientFactory                func(connector.OAuthClientConfig) (connector.OAuthClient, error)
	attemptTimeout, saveRetryTTL time.Duration
	gate                         *model.Coordinator
	mu                           sync.Mutex
	activeAttempt                *OAuthAttempt
	attempts                     map[string]*OAuthAttempt
	closed                       bool
	workers                      sync.WaitGroup
}

func NewOAuthManager(cfg OAuthManagerConfig) (OAuthManager, error) {
	if cfg.Store == nil {
		return nil, errors.New("missing account store")
	}
	base := cfg.RuntimeBaseDir
	if base == "" {
		base = filepath.Join(cfg.Store.GetStoreDir(), ".runtime", "oauth")
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	if err := credentialio.PrivateDir(base); err != nil {
		return nil, err
	}
	if cfg.CodexExeFinder == nil {
		cfg.CodexExeFinder = func() (string, error) { return connector.FindCodexBinary() }
	}
	if cfg.ClientFactory == nil {
		cfg.ClientFactory = connector.NewOAuthClient
	}
	if cfg.AttemptTimeout <= 0 {
		cfg.AttemptTimeout = 10 * time.Minute
	}
	if cfg.SaveRetryTTL <= 0 {
		cfg.SaveRetryTTL = 10 * time.Minute
	}
	if cfg.Coordinator == nil {
		cfg.Coordinator = store.Coordinator(cfg.Store)
	}
	return &oauthManager{store: cfg.Store, runtimeBaseDir: base, codexExePath: cfg.CodexExePath,
		codexExeFinder: cfg.CodexExeFinder, clientFactory: cfg.ClientFactory, attemptTimeout: cfg.AttemptTimeout,
		saveRetryTTL: cfg.SaveRetryTTL, gate: cfg.Coordinator, attempts: make(map[string]*OAuthAttempt)}, nil
}
func generateAttemptID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("secure randomness unavailable")
	}
	return hex.EncodeToString(b[:])
}

func (m *oauthManager) BeginLogin(ctx context.Context, mode, email, id string) (*OAuthAttempt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if mode == "" {
		mode = "add"
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if mode != "add" && mode != "reauth" {
		return nil, errors.New("invalid login mode")
	}
	m.gate.Lock()
	m.mu.Lock()
	unlock := func() { m.mu.Unlock(); m.gate.Unlock() }
	if m.closed {
		unlock()
		return nil, ErrManagerClosed
	}
	if a := m.activeAttempt; a != nil {
		unlock()
		if a.Mode == mode && a.ExpectedEmail == email && (id == "" || a.ExpectedAccountID == id) {
			return a, nil
		}
		return nil, ErrOAuthBusy
	}
	if err := m.gate.Check(""); err != nil {
		unlock()
		return nil, err
	}
	var revision uint64
	if mode == "reauth" {
		existing, err := m.store.GetAccount(email)
		if err != nil {
			unlock()
			return nil, err
		}
		_, storedID, err := model.ValidateAuth(existing.AuthData)
		if err != nil || (id != "" && id != storedID) {
			unlock()
			return nil, ErrReauthMismatch
		}
		id, revision = storedID, existing.Revision
	}
	attemptCtx, cancel := context.WithTimeout(context.Background(), m.attemptTimeout)
	a := &OAuthAttempt{workerDone: make(chan struct{}), AttemptID: generateAttemptID(), Mode: mode, ExpectedEmail: email, ExpectedAccountID: id,
		ExpectedRevision: revision, StartTime: time.Now(), CancelFn: cancel, DoneChan: make(chan LoginResult, 8)}
	a.CandidateHome = filepath.Join(m.runtimeBaseDir, a.AttemptID)
	a.CandidateAuthPath = filepath.Join(a.CandidateHome, "auth.json")
	a.publishLocked(LoginResult{AttemptID: a.AttemptID, Stage: "preparing"})
	m.attempts = map[string]*OAuthAttempt{a.AttemptID: a}
	m.activeAttempt = a
	m.gate.AttemptID = a.AttemptID
	m.workers.Add(1)
	unlock()
	go func() {
		defer m.workers.Done()
		defer close(a.workerDone)
		defer func() {
			if a.IsCanceled() {
				m.finishCleanup(a)
			}
		}()
		m.run(attemptCtx, a)
	}()
	return a, nil
}

func (m *oauthManager) run(ctx context.Context, a *OAuthAttempt) {
	if err := credentialio.PrivateDir(a.CandidateHome); err != nil {
		m.fail(a, "candidate", errors.New("cannot create private candidate"), false)
		return
	}
	if ctx.Err() != nil {
		m.fail(a, "canceled", ErrOAuthCanceled, false)
		return
	}
	exe := m.codexExePath
	if exe == "" {
		var err error
		exe, err = m.codexExeFinder()
		if err != nil {
			m.fail(a, "runtime", connector.ErrCodexBinaryNotFound, false)
			return
		}
	}
	client, err := m.clientFactory(connector.OAuthClientConfig{CodexExePath: exe, CandidateHome: a.CandidateHome, RPCTimeout: 15 * time.Second})
	if err != nil {
		m.fail(a, "runtime", errors.New("cannot start isolated Codex runtime"), false)
		return
	}
	a.mu.Lock()
	a.client = client
	canceled := a.canceled
	a.mu.Unlock()
	if canceled || ctx.Err() != nil {
		_ = client.Close()
		return
	}
	if err := client.Initialize(ctx); err != nil {
		m.fail(a, "initialize", errors.New("Codex handshake failed"), false)
		return
	}
	res, err := client.StartLogin(ctx)
	if err != nil {
		m.fail(a, "login", errors.New("Codex could not start OAuth"), false)
		return
	}
	m.gate.Lock()
	a.mu.Lock()
	if a.canceled {
		a.mu.Unlock()
		m.gate.Unlock()
		_ = client.Close()
		return
	}
	a.LoginID, a.AuthURL = res.LoginID, res.AuthURL
	a.publishLocked(LoginResult{AttemptID: a.AttemptID, Stage: "waiting_browser"})
	a.mu.Unlock()
	m.gate.Unlock()
	completed, err := client.WaitForLoginCompleted(ctx, res.LoginID)
	if err != nil || completed == nil || !completed.Success || completed.LoginID == nil || *completed.LoginID != res.LoginID {
		failure := errors.New("OAuth provider did not complete authentication")
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			failure = ErrOAuthTimeout
		}
		m.fail(a, "login", failure, false)
		return
	}
	a.mu.Lock()
	if a.canceled {
		a.mu.Unlock()
		return
	}
	a.publishLocked(LoginResult{AttemptID: a.AttemptID, Stage: "verifying"})
	a.mu.Unlock()
	email, plan, err := client.ReadAccount(ctx)
	if err != nil {
		m.fail(a, "account-read", errors.New("Codex account verification failed"), false)
		return
	}
	if err := client.Close(); err != nil {
		m.fail(a, "candidate", errors.New("candidate runtime did not stop"), false)
		return
	}
	raw, err := credentialio.Read(a.CandidateAuthPath)
	if err != nil {
		m.fail(a, "candidate-read", ErrCandidateAuthNotFound, false)
		return
	}
	candidateEmail, candidateID, err := model.ValidateAuth(raw)
	if err != nil || (email != "" && !strings.EqualFold(email, candidateEmail)) {
		m.fail(a, "identity", ErrCannotResolveEmail, false)
		return
	}
	if a.Mode == "reauth" && (candidateEmail != a.ExpectedEmail || candidateID != a.ExpectedAccountID) {
		m.fail(a, "identity", ErrReauthMismatch, false)
		return
	}
	a.mu.Lock()
	if a.canceled {
		a.mu.Unlock()
		return
	}
	a.rawAuthData, a.resolvedMail, a.resolvedID, a.planType = raw, candidateEmail, candidateID, plan
	a.mu.Unlock()
	if err := m.save(ctx, a); err != nil {
		m.fail(a, "saving", err, isStorageFailure(err))
	}
}
func isStorageFailure(err error) bool { var p *os.PathError; return errors.As(err, &p) }

// The durable write and terminal state change share the cancellation gate.
func (m *oauthManager) save(ctx context.Context, a *OAuthAttempt) error {
	m.gate.Lock()
	m.mu.Lock()
	a.mu.Lock()
	unlock := func() { a.mu.Unlock(); m.mu.Unlock(); m.gate.Unlock() }
	if m.closed {
		unlock()
		return ErrManagerClosed
	}
	if a.committed {
		unlock()
		return ErrAlreadyCommitted
	}
	if a.canceled {
		unlock()
		return ErrOAuthCanceled
	}
	if err := ctx.Err(); err != nil {
		unlock()
		return err
	}
	if err := m.gate.Check(a.AttemptID); err != nil {
		unlock()
		return err
	}
	if !a.saveFailedAt.IsZero() && time.Since(a.saveFailedAt) >= m.saveRetryTTL {
		unlock()
		return ErrSaveRetryExpired
	}
	if a.failed && a.stage != "save_failed" {
		unlock()
		return ErrAttemptNotFound
	}
	env := &model.CodexAccountEnvelope{Email: a.resolvedMail, Status: "on", PlanType: a.planType,
		AuthData: append(json.RawMessage(nil), a.rawAuthData...), Generation: "oauth_" + a.AttemptID}
	existing, err := m.store.GetAccount(env.Email)
	if err != nil && !errors.Is(err, store.ErrAccountNotFound) {
		unlock()
		return err
	}
	if a.Mode == "reauth" && (existing == nil || existing.Revision != a.ExpectedRevision) {
		unlock()
		return store.ErrRevisionConflict
	}
	if existing != nil {
		env.Revision = existing.Revision
		env.Status = existing.Status
		env.CreatedAt = existing.CreatedAt
	}
	if err := m.store.SaveAccount(env.Email, env); err != nil {
		unlock()
		return err
	}
	a.committed, a.failed = true, false
	a.AuthURL = ""
	a.rawAuthData = nil
	if a.retryTimer != nil {
		a.retryTimer.Stop()
	}
	a.CancelFn()
	a.publishLocked(LoginResult{AttemptID: a.AttemptID, Email: a.resolvedMail, AccountID: a.resolvedID, Stage: "success"})
	m.activeAttempt = nil
	// gate released after child cleanup
	unlock()
	return m.finishCleanup(a)
}
func (m *oauthManager) fail(a *OAuthAttempt, stage string, err error, retry bool) {
	m.gate.Lock()
	m.mu.Lock()
	a.mu.Lock()
	if a.committed || a.canceled || (a.failed && a.stage != "save_failed") {
		a.mu.Unlock()
		m.mu.Unlock()
		m.gate.Unlock()
		return
	}
	if m.closed {
		retry = false
		err = ErrManagerClosed
	}
	a.failed = true
	if retry {
		stage = "save_failed"
		if a.saveFailedAt.IsZero() {
			a.saveFailedAt = time.Now()
			a.retryTimer = time.AfterFunc(m.saveRetryTTL, func() { m.expire(a) })
		}
	} else {
		a.AuthURL = ""
		a.rawAuthData = nil
		a.CancelFn()
		if m.activeAttempt == a {
			m.activeAttempt = nil
			// gate released after child cleanup
		}
	}
	a.publishLocked(LoginResult{AttemptID: a.AttemptID, Stage: stage, Err: err, Retryable: retry})
	a.mu.Unlock()
	m.mu.Unlock()
	m.gate.Unlock()
	if !retry {
		m.finishCleanup(a)
	}
}
func (m *oauthManager) expire(a *OAuthAttempt) {
	m.gate.Lock()
	m.mu.Lock()
	a.mu.Lock()
	if a.committed || a.canceled || a.stage != "save_failed" {
		a.mu.Unlock()
		m.mu.Unlock()
		m.gate.Unlock()
		return
	}
	a.canceled = true
	a.AuthURL = ""
	a.rawAuthData = nil
	a.CancelFn()
	a.publishLocked(LoginResult{AttemptID: a.AttemptID, Stage: "expired", Err: ErrSaveRetryExpired})
	if m.activeAttempt == a {
		m.activeAttempt = nil
		// gate released after child cleanup
	}
	a.mu.Unlock()
	m.mu.Unlock()
	m.gate.Unlock()
	m.finishCleanup(a)
}
func (m *oauthManager) cleanup(a *OAuthAttempt) error {
	a.cleanupMu.Lock()
	defer a.cleanupMu.Unlock()
	a.mu.Lock()
	client, id := a.client, a.LoginID
	a.mu.Unlock()
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if id != "" && !a.IsCommitted() {
			_ = client.CancelLogin(ctx, id)
		}
		cancel()
		if err := client.Close(); err != nil {
			return errors.New("OAuth runtime cleanup failed")
		}
	}
	rel, err := filepath.Rel(m.runtimeBaseDir, a.CandidateHome)
	if err != nil || rel != a.AttemptID || filepath.Dir(rel) != "." {
		return errors.New("candidate cleanup containment failed")
	}
	if err := credentialio.CheckPath(a.CandidateHome); err != nil {
		return err
	}
	return os.RemoveAll(a.CandidateHome)
}
func (m *oauthManager) finishCleanup(a *OAuthAttempt) error {
	err := m.cleanup(a)
	m.gate.Lock()
	defer m.gate.Unlock()
	if err != nil {
		m.gate.RecoveryRequired = true
		return errors.New("OAuth cleanup requires recovery")
	}
	if m.gate.AttemptID == a.AttemptID {
		m.gate.AttemptID = ""
	}
	return nil
}
func (m *oauthManager) CancelActiveLogin() error {
	m.gate.Lock()
	m.mu.Lock()
	a := m.activeAttempt
	if a == nil {
		for _, old := range m.attempts {
			if old.IsCommitted() {
				m.mu.Unlock()
				m.gate.Unlock()
				return ErrAlreadyCommitted
			}
		}
		m.mu.Unlock()
		m.gate.Unlock()
		return ErrNoActiveAttempt
	}
	a.mu.Lock()
	if a.committed {
		a.mu.Unlock()
		m.mu.Unlock()
		m.gate.Unlock()
		return ErrAlreadyCommitted
	}
	a.canceled, a.failed = true, true
	a.AuthURL = ""
	a.rawAuthData = nil
	a.CancelFn()
	if a.retryTimer != nil {
		a.retryTimer.Stop()
	}
	a.publishLocked(LoginResult{AttemptID: a.AttemptID, Stage: "canceled", Err: ErrOAuthCanceled})
	m.activeAttempt = nil
	// gate released after child cleanup
	a.mu.Unlock()
	m.mu.Unlock()
	m.gate.Unlock()
	m.cleanup(a)
	if a.workerDone != nil {
		select {
		case <-a.workerDone:
		case <-time.After(20 * time.Second):
			return errors.New("OAuth cleanup still in progress")
		}
	}
	return m.finishCleanup(a)
}
func (m *oauthManager) IsAttemptActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeAttempt != nil
}
func (m *oauthManager) GetActiveAttempt() *OAuthAttempt {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeAttempt
}
func (m *oauthManager) RetrySave(ctx context.Context, id string) error {
	m.mu.Lock()
	a := m.attempts[id]
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return ErrManagerClosed
	}
	if a == nil {
		return ErrAttemptNotFound
	}
	if a.GetStage() != "save_failed" {
		if a.IsCommitted() {
			return ErrAlreadyCommitted
		}
		return ErrAttemptNotFound
	}
	err := m.save(ctx, a)
	if err != nil {
		m.fail(a, "saving", err, isStorageFailure(err))
	}
	return err
}
func (m *oauthManager) Close() error {
	m.gate.Lock()
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	m.gate.Unlock()
	cancelErr := m.CancelActiveLogin()
	if errors.Is(cancelErr, ErrNoActiveAttempt) || errors.Is(cancelErr, ErrAlreadyCommitted) {
		cancelErr = nil
	}
	m.workers.Wait()
	m.mu.Lock()
	attempts := m.attempts
	m.attempts = make(map[string]*OAuthAttempt)
	m.mu.Unlock()
	for _, a := range attempts {
		if err := m.finishCleanup(a); err != nil {
			return err
		}
	}
	return cancelErr
}
