package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/model"
	"as-auto-switcher/internal/engines/antigravity/reloader"
	"as-auto-switcher/internal/engines/antigravity/scanner"
	"as-auto-switcher/internal/engines/antigravity/store"
	"as-auto-switcher/internal/engines/antigravity/telemetry"
	"as-auto-switcher/internal/engines/antigravity/wincred"
)

var (
	ErrTargetNotFound            = errors.New("target account not found in account store")
	ErrTokenInvalid              = errors.New("token payload is empty or invalid")
	ErrWriteCredFailed           = errors.New("failed writing token to Windows Credential Manager")
	ErrReloadFailed              = errors.New("failed reloading session")
	ErrEmptyTargetEmail          = errors.New("target email cannot be empty")
	ErrCannotDeleteActiveAccount = errors.New("không thể xóa tài khoản đang hoạt động")
)

// AccountDisplayItem represents a unified account card for UI/CLI display.
type AccountDisplayItem struct {
	Email        string             `json:"email"`
	Name         string             `json:"name,omitempty"`
	AvatarURL    string             `json:"avatar_url,omitempty"`
	ProfileDir   string             `json:"profile_dir,omitempty"`
	State        model.AccountState `json:"state"`
	Status       string             `json:"status"` // "on" or "off"
	IsActive     bool               `json:"is_active"`
	HasToken     bool               `json:"has_token"`
	DisplayBadge string             `json:"display_badge"`
	Tier         string             `json:"tier,omitempty"` // "PRO", "ULTRA", "FREE", "BIZ", "--"
}

// AccountOverview encapsulates the current state of all accounts and language server status.
type AccountOverview struct {
	Versioned             bool                  `json:"-"`
	Revision              uint64                `json:"revision"`
	IdentitySource        string                `json:"identity_source"`
	WarningCodes          []string              `json:"warning_codes,omitempty"`
	ActiveEmail           string                `json:"active_email"`
	Accounts              []AccountDisplayItem  `json:"accounts"`
	LanguageServerRunning bool                  `json:"language_server_running"`
	ServerInfo            *connector.ServerInfo `json:"server_info,omitempty"`
	Warnings              []string              `json:"warnings,omitempty"`
}

// SwitchResult contains the outcome and latency of an account hot-swap operation.
type SwitchResult struct {
	Success             bool                          `json:"success"`
	Message             string                        `json:"message"`
	TargetEmail         string                        `json:"target_email"`
	PreviousEmail       string                        `json:"previous_email"`
	Duration            time.Duration                 `json:"duration"`
	ActiveThreads       []reloader.ActiveThread       `json:"active_threads,omitempty"`
	HotSwapped          bool                          `json:"hot_swapped"`
	ContinuationSummary *reloader.ContinuationSummary `json:"continuation_summary,omitempty"`
}

// AccountManager orchestrates AccountStore, WinCred, ChromeScanner, Connector, and Reloader.
type AccountManager struct {
	session            sessionGuard
	storedSnapshot     map[string]AccountDisplayItem
	discoveredSnapshot map[string]bool
	mu                 sync.RWMutex
	store              *store.AccountStore
	discoveredStore    *store.DiscoveredStore
	reloader           *reloader.SessionReloader
	chromeDir          string
	clientFinder       func(ctx context.Context) (*connector.Client, error)
	wincredReader      func() (*model.TokenData, error)
	wincredWriter      func(token *model.TokenData) error
	oauthRefresher     connector.OAuthRefresher
	trackedAccounts    map[string]bool
}

// NewAccountManager creates a fully initialized AccountManager with default production dependencies.
func NewAccountManager(accStore *store.AccountStore, sessionReloader *reloader.SessionReloader) (*AccountManager, error) {
	if accStore == nil {
		var err error
		accStore, err = store.NewAccountStore("")
		if err != nil {
			return nil, fmt.Errorf("failed initializing account store: %w", err)
		}
	}
	if sessionReloader == nil {
		sessionReloader = reloader.NewSessionReloader()
	}
	// The catalog shares the selected data location, never the launch CWD.
	// A temporary catalog I/O failure must not prevent use of saved credentials.
	discovered, err := store.NewDiscoveredStore(accStore.GetDiscoveredStoreDir())
	if discovered == nil {
		return nil, fmt.Errorf("failed resolving discovered store: %w", err)
	}
	if sessionReloader.OutputDir == "" {
		rootDir := accStore.GetDataRootDir()
		sessionReloader.OutputDir = filepath.Join(rootDir, "logs", "active_threads_history")
	}

	mgr := &AccountManager{
		store:           accStore,
		discoveredStore: discovered,
		reloader:        sessionReloader,
		chromeDir:       scanner.GetDefaultChromeUserDataDir(),
		wincredReader:   wincred.ReadToken,
		wincredWriter:   wincred.WriteToken,
		oauthRefresher:  connector.NewGoogleOAuthRefresher(nil),
		clientFinder: func(ctx context.Context) (*connector.Client, error) {
			proc, err := connector.DiscoverLanguageServerContext(ctx)
			if err != nil {
				return nil, err
			}
			return connector.NewClient(proc), nil
		},
		trackedAccounts: make(map[string]bool),
	}

	return mgr, nil
}

// SetClientFinder overrides the language server client factory (useful for unit testing).
func (m *AccountManager) SetClientFinder(fn func(ctx context.Context) (*connector.Client, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientFinder = fn
}

// SetWincredHandlers overrides credential read/write operations (useful for unit testing).
func (m *AccountManager) SetWincredHandlers(reader func() (*model.TokenData, error), writer func(token *model.TokenData) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wincredReader = reader
	m.wincredWriter = writer
}

// SetOAuthRefresher overrides the OAuth token refresher (useful for unit testing).
func (m *AccountManager) SetOAuthRefresher(refresher connector.OAuthRefresher) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oauthRefresher = refresher
}

// SetChromeDir overrides the Chrome User Data path.
func (m *AccountManager) SetChromeDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chromeDir = dir
}

func (m *AccountManager) SetReloader(rel *reloader.SessionReloader) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloader = rel
}

// Store returns the underlying AccountStore instance.
func (m *AccountManager) Store() *store.AccountStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store
}

// AccountStore returns the underlying AccountStore instance (alias for Store).
func (m *AccountManager) AccountStore() *store.AccountStore {
	return m.Store()
}

// DiscoveredStore returns the underlying DiscoveredStore instance.
func (m *AccountManager) DiscoveredStore() *store.DiscoveredStore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.discoveredStore
}

// SetDiscoveredStore sets the DiscoveredStore instance.
func (m *AccountManager) SetDiscoveredStore(ds *store.DiscoveredStore) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discoveredStore = ds
}

// SetAccountStatus updates the status ("on" or "off") of an account in the store.
func (m *AccountManager) SetAccountStatus(email string, status string) error {
	release, lockErr := m.session.mutate(context.Background())
	if lockErr != nil {
		return lockErr
	}
	defer release()
	m.mu.RLock()
	st := m.store
	m.mu.RUnlock()
	if st == nil {
		return errors.New("account store is nil")
	}
	return st.SetAccountStatus(email, status)
}

// GetAccountStatus returns the status ("on" or "off") of an account, defaulting to "on" on error.
func (m *AccountManager) GetAccountStatus(email string) string {
	m.mu.RLock()
	st := m.store
	m.mu.RUnlock()
	if st == nil {
		return "on"
	}
	status, err := st.GetAccountStatus(email)
	if err != nil || status == "" {
		return "on"
	}
	return status
}

// RemoveAccount orchestrates the two-store deletion workflow:
// 1. Rejects deletion if the target email is currently active.
// 2. Deletes the token credentials file in accounts_store/<email>.json.
// 3. Resets the Added flag to false in discovered_accounts/<email>.json (if record exists).
func (m *AccountManager) RemoveAccount(ctx context.Context, email string) error {
	release, lockErr := m.session.mutate(ctx)
	if lockErr != nil {
		return lockErr
	}
	defer release()
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return store.ErrInvalidEmail
	}

	// Invariant 1: Cấm xóa tài khoản đang Active
	activeEmail, _ := m.GetCurrentActiveEmail(ctx)
	if activeEmail != "" && strings.EqualFold(activeEmail, normEmail) {
		return ErrCannotDeleteActiveAccount
	}

	m.mu.Lock()
	accStore, discStore := m.store, m.discoveredStore
	m.mu.Unlock()

	// Invariant: Double-check kiểm tra activeEmail trước khi thực hiện xóa
	activeEmail2, _ := m.GetCurrentActiveEmail(ctx)
	if activeEmail2 != "" && strings.EqualFold(activeEmail2, normEmail) {
		return ErrCannotDeleteActiveAccount
	}

	// 1. Xóa khỏi accounts_store
	if accStore != nil {
		if err := accStore.DeleteAccount(normEmail); err != nil {
			return fmt.Errorf("lỗi xóa tệp tài khoản: %w", err)
		}
		delete(m.storedSnapshot, normEmail)
	}

	// 2. Cập nhật added: false trong discovered_accounts nếu record đã tồn tại
	if discStore != nil {
		records, err := discStore.ListDiscoveredRecords()
		if err == nil {
			for _, rec := range records {
				if strings.EqualFold(rec.Email, normEmail) {
					if err := discStore.SetDiscoveredAdded(normEmail, false); err != nil {
						return err
					}
					delete(m.discoveredSnapshot, normEmail)
					break
				}
			}
		}
	}

	return nil
}

// ScanChromeAccounts scans Chrome profiles on-demand and syncs new accounts into discoveredStore.
func (m *AccountManager) ScanChromeAccounts() ([]string, error) {
	release, lockErr := m.session.mutate(context.Background())
	if lockErr != nil {
		return nil, lockErr
	}
	defer release()
	m.mu.RLock()
	cDir, discovered := m.chromeDir, m.discoveredStore
	m.mu.RUnlock()

	if discovered == nil {
		return nil, errors.New("discovered store uninitialized")
	}

	chromeAccounts, err := scanner.ScanChromeAccountDetails(cDir)
	if err != nil {
		return nil, fmt.Errorf("không quét được Chrome: %w", err)
	}

	var scannedEmails []string
	for _, ca := range chromeAccounts {
		em := strings.ToLower(strings.TrimSpace(ca.Email))
		if em == "" {
			continue
		}
		_ = discovered.SaveDiscovered(em) // Đã có logic bảo toàn cờ Added
		scannedEmails = append(scannedEmails, em)
	}
	return scannedEmails, nil
}

// CandidateAccountItem represents a discovered account candidate with token, added, and file existence state.
type CandidateAccountItem struct {
	Email      string
	HasToken   bool
	Added      bool
	FileExists bool // true: file JSON trong discovered_accounts tồn tại; false: không tồn tại
}

// GetCandidateAccounts returns all discovered accounts enriched with token, added, and file existence state.
// It detects non-existent accounts that appear in UI/management tracking but lack a JSON file on disk.
func (m *AccountManager) GetCandidateAccounts(knownEmails ...string) ([]CandidateAccountItem, error) {
	m.mu.RLock()
	st, discovered := m.store, m.discoveredStore
	tracked := make(map[string]bool, len(m.trackedAccounts))
	for k, v := range m.trackedAccounts {
		tracked[k] = v
	}
	m.mu.RUnlock()

	if discovered == nil {
		return nil, errors.New("discovered store uninitialized")
	}

	records, err := discovered.ListDiscoveredRecords()
	if err != nil {
		return nil, err
	}

	storedSet := make(map[string]bool)
	var storedEmails []string
	if st != nil {
		if stored, err := st.ListAccounts(); err == nil {
			storedEmails = stored
			for _, em := range storedEmails {
				storedSet[strings.ToLower(strings.TrimSpace(em))] = true
			}
		}
	}

	items := make([]CandidateAccountItem, 0, len(records))
	seen := make(map[string]bool)
	for _, r := range records {
		norm := strings.ToLower(strings.TrimSpace(r.Email))
		seen[norm] = true
		items = append(items, CandidateAccountItem{
			Email:      norm,
			HasToken:   storedSet[norm],
			Added:      r.Added,
			FileExists: true,
		})
	}

	// Kiểm tra các tài khoản đang tồn tại trên màn hình chính (tracked, known):
	checkGhost := func(email string) {
		norm := strings.ToLower(strings.TrimSpace(email))
		if norm == "" || seen[norm] {
			return
		}
		seen[norm] = true
		if !discovered.HasDiscovered(norm) {
			items = append(items, CandidateAccountItem{
				Email:      norm,
				HasToken:   storedSet[norm],
				Added:      true,
				FileExists: false,
			})
		}
	}

	for em := range tracked {
		checkGhost(em)
	}
	for _, em := range knownEmails {
		checkGhost(em)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].HasToken != items[j].HasToken {
			return items[i].HasToken
		}
		if items[i].Added != items[j].Added {
			return items[i].Added
		}
		return items[i].Email < items[j].Email
	})
	return items, nil
}

// AddDiscoveredAccount marks an account as added to the main UI.
func (m *AccountManager) AddDiscoveredAccount(email string) error {
	release, lockErr := m.session.mutate(context.Background())
	if lockErr != nil {
		return lockErr
	}
	defer release()
	normEmail := strings.ToLower(strings.TrimSpace(email))
	m.mu.Lock()
	if m.trackedAccounts == nil {
		m.trackedAccounts = make(map[string]bool)
	}
	m.trackedAccounts[normEmail] = true
	discovered := m.discoveredStore
	m.mu.Unlock()

	if discovered == nil {
		return errors.New("discovered store uninitialized")
	}
	if err := discovered.SetDiscoveredAdded(normEmail, true); err != nil {
		return err
	}
	if m.discoveredSnapshot == nil {
		m.discoveredSnapshot = make(map[string]bool)
	}
	m.discoveredSnapshot[normEmail] = true
	return nil
}

// DeleteDiscoveredAccount deletes the discovered account JSON file from discovered_accounts/.
// It does NOT touch accounts_store/ or main UI active credentials.
func (m *AccountManager) DeleteDiscoveredAccount(email string) error {
	release, lockErr := m.session.mutate(context.Background())
	if lockErr != nil {
		return lockErr
	}
	defer release()
	normEmail := strings.ToLower(strings.TrimSpace(email))
	m.mu.Lock()
	discovered := m.discoveredStore
	if m.trackedAccounts != nil {
		delete(m.trackedAccounts, normEmail)
	}
	m.mu.Unlock()

	if discovered == nil {
		return errors.New("discovered store uninitialized")
	}
	if err := discovered.DeleteDiscovered(normEmail); err != nil {
		return err
	}
	delete(m.discoveredSnapshot, normEmail)
	return nil
}

// TrackAccount registers an account as tracked by the application UI/management.
func (m *AccountManager) TrackAccount(email string) {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.trackedAccounts == nil {
		m.trackedAccounts = make(map[string]bool)
	}
	m.trackedAccounts[normEmail] = true
}

// UntrackAccount unregisters an account from tracked UI accounts.
func (m *AccountManager) UntrackAccount(email string) {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.trackedAccounts != nil {
		delete(m.trackedAccounts, normEmail)
	}
}

// SwitchAccount performs a closed-loop hot-swap:
//  1. Probe live RAM -> verify if already active on targetEmail (no-op).
//  2. Snapshot active threads BEFORE switch to preserve in-flight conversation state (P0-04).
//  3. Auto-capture active session to ensure credentials are safe.
//  4. Load target token from store.
//  5. Parity with Python core/state_resolver.py:380-425:
//     Default is False (smooth zero-chrome hot-swap without OAuth popups).
//     NEVER call AuthLogout during hot-swap unless forceLogout is explicitly true! (P0-02).
//  6. Write target token from accounts_store/<email>.json to Windows Credential Manager.
//  7. Pre-patch ToS in app_storage.json (jetski.onboarding.lastLoginIsGcpTos = "false").
//  8. Ensure Antigravity window in foreground.
//  9. Recycle language_server.exe via taskkill /F and wait for monitorLsCrashInternal to respawn (P0-03).
//  10. Re-probe fresh Language Server instance and verify active identity via GetUserStatus.
//  11. In-Flight Session Continuation (SmartFocus -> LoadTrajectory -> SendUserCascadeMessage) (P0-04).
func (m *AccountManager) SwitchAccount(ctx context.Context, targetEmail string, forceLogout ...bool) (*SwitchResult, error) {
	release, lockErr := m.session.mutate(ctx)
	if lockErr != nil {
		return nil, lockErr
	}
	defer release()
	start := time.Now()
	normTarget := strings.ToLower(strings.TrimSpace(targetEmail))
	if normTarget == "" {
		return nil, ErrEmptyTargetEmail
	}

	timelineDir := DefaultSwitchTimelineDir
	if m.store != nil && m.store.GetStoreDir() != "" {
		timelineDir = filepath.Join(m.store.GetDataRootDir(), "logs", "switch_timeline")
	}

	// 1. Kiểm tra tài khoản hiện tại trong RAM, nếu đã là targetEmail thì no-op
	currentEmail, _ := m.GetCurrentActiveEmail(ctx)
	if currentEmail != "" && currentEmail == normTarget {
		return &SwitchResult{
			Success:       true,
			Message:       fmt.Sprintf("Tài khoản %s hiện đang hoạt động.", normTarget),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
		}, nil
	}

	LogSwitchEvent(timelineDir, "SWITCH_START", fmt.Sprintf("Bắt đầu chuyển tài khoản từ %s sang %s", currentEmail, normTarget))

	failSwitch := func(res *SwitchResult, err error) (*SwitchResult, error) {
		LogSwitchEvent(timelineDir, "SWITCH_FAILED", fmt.Sprintf("Thất bại: %v", err), time.Since(start))
		return res, err
	}

	m.mu.RLock()
	rel := m.reloader
	writer := m.wincredWriter
	refresher := m.oauthRefresher
	finder := m.clientFinder
	m.mu.RUnlock()

	var shouldLogout bool
	if len(forceLogout) > 0 {
		shouldLogout = forceLogout[0]
	}
	if shouldLogout && finder != nil {
		if client, err := finder(ctx); err == nil && client != nil {
			_ = client.AuthLogout(ctx)
		}
	}

	// =========================================================================
	// BƯỚC 1: Chốt danh sách trước khi chuyển tài khoản (Snapshot SQLite 2 Giờ)
	// =========================================================================
	var snapshotPath string
	var activeThreads []reloader.ActiveThread
	if rel != nil {
		snapshotPath, activeThreads, _ = rel.CaptureActiveThreadsSnapshot("", "", "")
	} else {
		snapshotPath, activeThreads, _ = reloader.CaptureActiveThreadsSnapshot("", "", "")
	}
	LogSwitchEvent(timelineDir, "SNAPSHOT_DONE", fmt.Sprintf("Đã snapshot %d thread(s) đang hoạt động", len(activeThreads)), time.Since(start))

	// Tự động lưu phiên làm việc hiện tại
	_, _ = m.captureCurrentSession(ctx)

	// Tải token của tài khoản mục tiêu từ accounts_store/
	targetToken, err := m.store.GetAccount(normTarget)
	if err != nil {
		return failSwitch(&SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Chưa có file JSON của %s trong accounts_store/!", normTarget),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, ErrTargetNotFound)
	}

	// Auto-refresh token nếu sắp hết hạn trong 5 phút
	if targetToken != nil && (targetToken.Expiry.IsZero() || time.Until(targetToken.Expiry) < 5*time.Minute) {
		if targetToken.RefreshToken != "" {
			activeRefresher := refresher
			if activeRefresher == nil {
				activeRefresher = connector.NewGoogleOAuthRefresher(nil)
			}
			refreshRes, refErr := activeRefresher.RefreshToken(ctx, targetToken.RefreshToken)
			if refErr == nil && refreshRes != nil && refreshRes.AccessToken != "" {
				targetToken.UpdateTokens(refreshRes.AccessToken, refreshRes.RefreshToken, refreshRes.IDToken, refreshRes.Expiry)
				_ = m.store.SaveAccount(normTarget, targetToken)
			}
		}
	}

	// =========================================================================
	// BƯỚC 2: Chuyển tài khoản bằng cơ chế chuẩn của Antigravity
	// =========================================================================
	// 2.1 Đồng bộ ToS vào app_storage.json
	isGcpTos := strings.EqualFold(targetToken.AuthMethod, "gcp")
	storagePath := ""
	if rel != nil {
		storagePath = rel.StoragePath
	}
	_ = reloader.SyncAppStorageToS(normTarget, isGcpTos, storagePath)

	// 2.2 Ghi token vào Windows Credential Manager
	if writer == nil {
		return failSwitch(nil, errors.New("wincred writer not configured"))
	}
	if err := writer(targetToken); err != nil {
		return failSwitch(&SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Không thể ghi Token vào Windows Credential Manager: %v", err),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, fmt.Errorf("%w: %v", ErrWriteCredFailed, err))
	}
	LogSwitchEvent(timelineDir, "WINCRED_WRITE", fmt.Sprintf("Đã ghi token mới của %s vào WinCred", normTarget), time.Since(start))

	// 2.3 Đưa cửa sổ Antigravity lên foreground
	_ = reloader.EnsureAntigravityForeground()

	// 2.4 Dừng language_server.exe để Antigravity monitorLsCrashInternal tự khởi động lại
	// LƯU Ý BẮT BUỘC: Xóa bỏ 100% nhánh gọi RPC ReAuth ảo!
	// BẮT ĐẦU CHỜ ĐỦ 8 GIÂY rồi mới qua bước tiếp theo
	waitDuration := 8 * time.Second
	var recycleErr error
	if rel != nil {
		recycleErr = rel.RecycleLanguageServer(ctx, waitDuration)
	} else {
		recycleErr = reloader.RecycleLanguageServer(ctx, waitDuration)
	}
	if recycleErr != nil {
		return failSwitch(&SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Khởi động lại Language Server thất bại: %v", recycleErr),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, fmt.Errorf("failed recycling language server: %w", recycleErr))
	}
	LogSwitchEvent(timelineDir, "KILL_LS", "Language Server đã được yêu cầu khởi động lại", time.Since(start))

	// 2.5 Dò tìm Language Server mới sau 8 giây ổn định và xác nhận danh tính trong RAM (F1)
	var newClient *connector.Client
	var clientErr error
	if finder != nil {
		newClient, clientErr = finder(ctx)
	} else {
		var newInfo *connector.ServerInfo
		newInfo, clientErr = connector.DiscoverLanguageServerContext(ctx)
		if clientErr == nil && newInfo != nil {
			newClient = connector.NewClient(newInfo)
		}
	}

	// R2-F1 [P1]: Bắt buộc mọi trường hợp newClient == nil đều trả Success = false
	if newClient == nil {
		errMsg := "không tìm thấy language server mới sau khi khởi động lại"
		if clientErr != nil {
			return failSwitch(&SwitchResult{
				Success:       false,
				Message:       fmt.Sprintf("Không thể kết nối với Language Server mới sau khi chuyển sang %s: %v", normTarget, clientErr),
				TargetEmail:   normTarget,
				PreviousEmail: currentEmail,
				Duration:      time.Since(start),
				ActiveThreads: activeThreads,
			}, fmt.Errorf("%s: %w", errMsg, clientErr))
		}
		return failSwitch(&SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Không thể kết nối với Language Server mới sau khi chuyển sang %s", normTarget),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, errors.New(errMsg))
	}

	// F1 [P1 - CỐT TỬ]: Bắt buộc xác thực email trong RAM qua GetUserStatus trước khi gửi tin nhắn hoặc báo thành công
	status, statusErr := newClient.GetUserStatus(ctx)
	if statusErr != nil {
		return failSwitch(&SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Không thể xác thực trạng thái người dùng trong RAM sau khi đổi tài khoản: %v", statusErr),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, fmt.Errorf("failed verifying user status in RAM: %w", statusErr))
	}

	if status == nil || strings.TrimSpace(status.Email) == "" {
		return failSwitch(&SwitchResult{
			Success:       false,
			Message:       "Không thể xác thực trạng thái người dùng trong RAM sau khi đổi tài khoản: email rỗng hoặc trạng thái không hợp lệ",
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, errors.New("failed verifying user status in RAM: empty status or email"))
	}

	ramEmail := strings.ToLower(strings.TrimSpace(status.Email))
	if ramEmail != normTarget {
		return failSwitch(&SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Xác thực RAM thất bại: Language Server vẫn đang chạy tài khoản %s, không khớp mục tiêu %s", ramEmail, normTarget),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, fmt.Errorf("identity mismatch in RAM: expected %s, got %s", normTarget, ramEmail))
	}
	LogSwitchEvent(timelineDir, "RAM_CHECK_DONE", fmt.Sprintf("Xác thực RAM thành công: %s", ramEmail), time.Since(start))

	// =========================================================================
	// BƯỚC 3: Đọc đúng file JSON của bước 1
	// =========================================================================
	threadsToContinue := activeThreads
	if snapshotPath != "" {
		if diskThreads, readErr := reloader.ReadActiveThreadsSnapshot(snapshotPath); readErr == nil && len(diskThreads) > 0 {
			threadsToContinue = diskThreads
		}
	}

	// =========================================================================
	// BƯỚC 4: Gửi tuần tự toàn bộ danh sách (Nghỉ 3s giữa các thread, không ngắt khi lỗi)
	// =========================================================================
	var contSummary *reloader.ContinuationSummary
	if newClient != nil && len(threadsToContinue) > 0 {
		LogSwitchEvent(timelineDir, "DISPATCH_START", fmt.Sprintf("Bắt đầu gửi tin nhắn tiếp tục phiên vào %d thread", len(threadsToContinue)), time.Since(start))
		if rel != nil {
			contSummary, _ = rel.PerformSessionContinuation(ctx, newClient, threadsToContinue, "")
		} else {
			contSummary, _ = reloader.PerformSessionContinuation(ctx, newClient, threadsToContinue, "")
		}
		if contSummary != nil {
			LogSwitchEvent(timelineDir, "DISPATCH_DONE", fmt.Sprintf("Gửi tiếp tục phiên hoàn tất: %d/%d thành công", contSummary.SuccessCount, contSummary.TotalThreads), time.Since(start))
		}
	}

	// =========================================================================
	// BƯỚC 5: Tự động dọn dẹp lịch sử snapshot (Chỉ giữ đúng 2 file mới nhất)
	// =========================================================================
	historyDir := ""
	if rel != nil && rel.OutputDir != "" {
		historyDir = rel.OutputDir
	} else if m.store != nil {
		rootDir := m.store.GetDataRootDir()
		historyDir = filepath.Join(rootDir, "logs", "active_threads_history")
	} else {
		historyDir = reloader.GetDefaultThreadsHistoryDir()
	}
	_, _ = reloader.PruneActiveThreadsHistory(historyDir, reloader.DefaultHistoryKeepCount)

	// Cập nhật thông điệp hoàn tất minh bạch
	msg := fmt.Sprintf("Chuyển sang tài khoản %s thành công!", normTarget)
	if contSummary != nil && contSummary.TotalThreads > 0 {
		msg = fmt.Sprintf("Chuyển sang tài khoản %s thành công! Đã gửi prompt tiếp tục phiên vào %d/%d thread.",
			normTarget, contSummary.SuccessCount, contSummary.TotalThreads)
	}

	LogSwitchEvent(timelineDir, "SWITCH_SUCCESS", fmt.Sprintf("Chuyển sang %s thành công hoàn tất", normTarget), time.Since(start))

	return &SwitchResult{
		Success:             true,
		Message:             msg,
		TargetEmail:         normTarget,
		PreviousEmail:       currentEmail,
		Duration:            time.Since(start),
		ActiveThreads:       threadsToContinue,
		ContinuationSummary: contSummary,
	}, nil
}

// sessionGuard serializes capture and all account mutations, including switches
// initiated by automation. Revision changes at both mutation boundaries prevent
// an old UI result from being applied while a mutation waits for a reader.
type sessionGuard struct {
	once       sync.Once
	gate       chan struct{}
	revision   atomic.Uint64
	pending    atomic.Int32
	cancelMu   sync.Mutex
	readCancel context.CancelFunc
}

func (g *sessionGuard) lock(ctx context.Context) error {
	g.once.Do(func() { g.gate = make(chan struct{}, 1) })
	select {
	case g.gate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			g.unlock()
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (g *sessionGuard) unlock() { <-g.gate }
func (g *sessionGuard) mutate(ctx context.Context) (func(), error) {
	g.pending.Add(1)
	g.revision.Add(1)
	g.cancelMu.Lock()
	if g.readCancel != nil {
		g.readCancel()
	}
	g.cancelMu.Unlock()
	if err := g.lock(ctx); err != nil {
		g.revision.Add(1)
		g.pending.Add(-1)
		return nil, err
	}
	return func() { g.revision.Add(1); g.pending.Add(-1); g.unlock() }, nil
}
func (g *sessionGuard) read(ctx context.Context) (context.Context, func(), error) {
	if err := g.lock(ctx); err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	g.cancelMu.Lock()
	g.readCancel = cancel
	if g.pending.Load() > 0 {
		cancel()
	}
	g.cancelMu.Unlock()
	return ctx, func() {
		g.cancelMu.Lock()
		g.readCancel = nil
		cancel()
		g.cancelMu.Unlock()
		g.unlock()
	}, nil
}

func (m *AccountManager) Revision() uint64 { return m.session.revision.Load() }
func (m *AccountManager) IsMutating() bool { return m.session.pending.Load() > 0 }

var (
	ErrCredentialEmailMissing  = errors.New("could not extract email from active credential")
	ErrCredentialEmailConflict = errors.New("credential identity fields disagree")
	ErrNoActiveSession         = errors.New("no active session")
	ErrCaptureRead             = errors.New("credential read failed")
	ErrCaptureWrite            = errors.New("credential save failed")
)

func normalizeIdentity(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if strings.Count(value, "@") != 1 || strings.ContainsAny(value, " \t\r\n/\\") {
		return ""
	}
	parts := strings.SplitN(value, "@", 2)
	if parts[0] == "" || parts[1] == "" {
		return ""
	}
	return value
}

func credentialEmail(token *model.TokenData) (string, error) {
	if token == nil {
		return "", ErrCredentialEmailMissing
	}
	candidates := []string{}
	if parts := strings.Split(token.IDToken, "."); len(parts) >= 2 {
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
		if err == nil {
			var claims map[string]interface{}
			if json.Unmarshal(raw, &claims) == nil {
				if email, ok := claims["email"].(string); ok {
					candidates = append(candidates, email)
				}
			}
		}
	}
	candidates = append(candidates, token.IDTokenEmail, token.Email)
	var raw map[string]interface{}
	if json.Unmarshal(token.RawPayload, &raw) == nil {
		for _, key := range []string{"id_token_email", "email"} {
			if email, ok := raw[key].(string); ok {
				candidates = append(candidates, email)
			}
		}
	}
	result := ""
	for _, candidate := range candidates {
		email := normalizeIdentity(candidate)
		if email == "" {
			continue
		}
		if result != "" && result != email {
			return "", ErrCredentialEmailConflict
		}
		result = email
	}
	if result == "" {
		return "", ErrCredentialEmailMissing
	}
	return result, nil
}

func ExtractEmailFromToken(token *model.TokenData) string {
	email, _ := credentialEmail(token)
	return email
}

func (m *AccountManager) CaptureCurrentSession() (string, error) {
	ctx, release, err := m.session.read(context.Background())
	if err != nil {
		return "", err
	}
	defer release()
	return m.captureCurrentSession(ctx)
}

func (m *AccountManager) captureCurrentSession(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.RLock()
	reader, st := m.wincredReader, m.store
	m.mu.RUnlock()
	if reader == nil {
		return "", ErrCaptureRead
	}
	token, err := reader()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCaptureRead, err)
	}
	if token == nil || !token.IsValid() {
		return "", ErrTokenInvalid
	}
	email, err := credentialEmail(token)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if st == nil {
		return "", ErrCaptureWrite
	}
	if err := st.SaveAccountIfChanged(email, token); err != nil {
		return "", fmt.Errorf("%w: %w", ErrCaptureWrite, err)
	}
	return email, nil
}

func credentialErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrCredentialEmailConflict):
		return "credential_identity_conflict"
	case errors.Is(err, ErrCredentialEmailMissing):
		return "credential_identity_missing"
	case errors.Is(err, ErrCaptureWrite):
		return "credential_write"
	case errors.Is(err, ErrTokenInvalid), errors.Is(err, wincred.ErrInvalidPayload):
		return "credential_invalid"
	default:
		return "credential_read"
	}
}

func (m *AccountManager) GetCurrentActiveEmail(ctx context.Context) (string, error) {
	email, _, _, err := m.resolveActiveIdentity(ctx)
	return email, err
}

func (m *AccountManager) resolveActiveIdentity(ctx context.Context) (string, string, *connector.ServerInfo, error) {
	m.mu.RLock()
	finder, reader := m.clientFinder, m.wincredReader
	m.mu.RUnlock()
	var info *connector.ServerInfo
	var failures []error
	if err := ctx.Err(); err != nil {
		return "", "unknown", nil, err
	}
	if finder != nil {
		client, err := finder(ctx)
		if err == nil && client != nil {
			info = client.ServerInfo()
			status, err := client.GetUserStatus(ctx)
			if err == nil && status != nil {
				if email := normalizeIdentity(status.Email); email != "" {
					return email, "live", info, nil
				}
			}
			failures = append(failures, errors.New("live identity unavailable"))
		} else if err != nil && !errors.Is(err, connector.ErrLanguageServerNotFound) {
			failures = append(failures, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return "", "unknown", info, err
	}
	if reader != nil {
		token, err := reader()
		if err == nil && token != nil && token.IsValid() {
			email, err := credentialEmail(token)
			if err == nil {
				return email, "credential", info, nil
			}
			failures = append(failures, err)
		} else if err != nil && !errors.Is(err, wincred.ErrNotFound) {
			failures = append(failures, ErrCaptureRead)
		}
	}
	if len(failures) == 0 {
		return "", "none", info, ErrNoActiveSession
	}
	return "", "unknown", info, errors.Join(failures...)
}

// GetAccountStates captures and reads one serialized session transaction. Source
// snapshots retain only unreadable data, never turn a complete deletion into a
// stale account, and are not a substitute for reading the two on-disk stores.
func (m *AccountManager) GetAccountStates(ctx context.Context) (*AccountOverview, error) {
	ctx, release, err := m.session.read(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	view := &AccountOverview{Revision: m.Revision(), Versioned: true}
	warn := func(code, message string, err error) {
		view.WarningCodes = append(view.WarningCodes, code)
		view.Warnings = append(view.Warnings, fmt.Sprintf("%s: %v", message, err))
	}
	if _, err := m.captureCurrentSession(ctx); err != nil && !errors.Is(err, wincred.ErrNotFound) {
		warn(credentialErrorCode(err), "Không lưu được phiên hiện tại", err)
	}
	email, source, info, identityErr := m.resolveActiveIdentity(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	view.ActiveEmail, view.IdentitySource, view.ServerInfo = email, source, info
	view.LanguageServerRunning = info != nil
	if identityErr != nil && !errors.Is(identityErr, ErrNoActiveSession) {
		warn("identity_unavailable", "Không xác định được phiên hiện tại", identityErr)
	}
	m.mu.RLock()
	st, catalogStore := m.store, m.discoveredStore
	m.mu.RUnlock()
	stored := make(map[string]AccountDisplayItem)
	emails, storeErr := st.ListAccountsWithErrors()
	if storeErr != nil {
		for email, item := range m.storedSnapshot {
			path, pathErr := store.SafeAccountPath(st.GetStoreDir(), email)
			_, statErr := os.Stat(path)
			if pathErr == nil && !os.IsNotExist(statErr) {
				stored[email] = item
			}
		}
		warn("credential_store_read", "Không đọc đầy đủ kho tài khoản", storeErr)
	}
	for _, email := range emails {
		email = strings.ToLower(strings.TrimSpace(email))
		env, err := st.GetAccountEnvelope(email)
		if err != nil {
			if item, ok := m.storedSnapshot[email]; ok {
				stored[email] = item
			}
			warn("credential_store_read", "Không đọc đầy đủ kho tài khoản", err)
			continue
		}
		status := "on"
		if env.Status == "off" {
			status = "off"
		}
		stored[email] = AccountDisplayItem{Email: email, HasToken: true, Status: status, Tier: env.Tier}
	}
	catalog := make(map[string]bool)
	if catalogStore != nil {
		records, err := catalogStore.ListDiscoveredRecords()
		if err != nil {
			for email, added := range m.discoveredSnapshot {
				if !catalogStore.RecordMissing(email) {
					catalog[email] = added
				}
			}
			warn("catalog_read", "Không đọc đầy đủ danh bạ Chrome", err)
		}
		for _, record := range records {
			catalog[record.Email] = record.Added
		}
	}
	items := make(map[string]AccountDisplayItem)
	for email, added := range catalog {
		if added {
			items[email] = AccountDisplayItem{Email: email, Status: "on"}
		}
	}
	for email, item := range stored {
		items[email] = item
	}
	if email != "" {
		if _, ok := items[email]; !ok {
			items[email] = AccountDisplayItem{Email: email, Status: "on"}
		}
	}
	for _, item := range items {
		item.IsActive = email != "" && item.Email == email
		item.Tier = telemetry.NormalizeTier(item.Tier)
		switch {
		case item.IsActive:
			item.State, item.DisplayBadge = model.StateActive, "[✔ Dùng]"
		case item.HasToken:
			item.State, item.DisplayBadge = model.StateReady, "[Chuyển]"
		default:
			item.State, item.DisplayBadge = model.StateUnauth, "[Cần OAuth]"
		}
		view.Accounts = append(view.Accounts, item)
	}
	sort.Slice(view.Accounts, func(i, j int) bool {
		a, b := view.Accounts[i], view.Accounts[j]
		if a.IsActive != b.IsActive {
			return a.IsActive
		}
		if a.HasToken != b.HasToken {
			return a.HasToken
		}
		return a.Email < b.Email
	})
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.storedSnapshot, m.discoveredSnapshot = stored, catalog
	m.mu.Lock()
	if m.trackedAccounts == nil {
		m.trackedAccounts = make(map[string]bool)
	}
	for _, item := range view.Accounts {
		if !item.HasToken {
			m.trackedAccounts[item.Email] = true
		}
	}
	m.mu.Unlock()
	return view, nil
}
