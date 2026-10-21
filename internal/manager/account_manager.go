package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/scanner"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/wincred"
)

var (
	ErrTargetNotFound   = errors.New("target account not found in account store")
	ErrTokenInvalid     = errors.New("token payload is empty or invalid")
	ErrWriteCredFailed  = errors.New("failed writing token to Windows Credential Manager")
	ErrReloadFailed     = errors.New("failed reloading session")
	ErrEmptyTargetEmail = errors.New("target email cannot be empty")
)

// AccountDisplayItem represents a unified account card for UI/CLI display.
type AccountDisplayItem struct {
	Email        string             `json:"email"`
	Name         string             `json:"name,omitempty"`
	AvatarURL    string             `json:"avatar_url,omitempty"`
	ProfileDir   string             `json:"profile_dir,omitempty"`
	State        model.AccountState `json:"state"`
	Status       string             `json:"status,omitempty"`
	IsActive     bool               `json:"is_active"`
	HasToken     bool               `json:"has_token"`
	DisplayBadge string             `json:"display_badge"`
}

// AccountOverview encapsulates the current state of all accounts and language server status.
type AccountOverview struct {
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
	mu              sync.RWMutex
	store           *store.AccountStore
	discoveredStore *store.DiscoveredStore
	reloader        *reloader.SessionReloader
	chromeDir       string
	clientFinder    func(ctx context.Context) (*connector.Client, error)
	wincredReader   func() (*model.TokenData, error)
	wincredWriter   func(token *model.TokenData) error
	oauthRefresher  connector.OAuthRefresher
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
	discovered, err := store.NewDiscoveredStore(filepath.Join(filepath.Dir(accStore.GetStoreDir()), "discovered_accounts"))
	if discovered == nil {
		return nil, fmt.Errorf("failed resolving discovered store: %w", err)
	}
	if sessionReloader.OutputDir == "" {
		rootDir := filepath.Dir(accStore.GetStoreDir())
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
			proc, err := connector.DiscoverLanguageServer()
			if err != nil {
				return nil, err
			}
			return connector.NewClient(proc), nil
		},
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

// ExtractEmailFromToken extracts the email address from a TokenData payload or its JWT id_token.
func ExtractEmailFromToken(token *model.TokenData) string {
	if token == nil {
		return ""
	}

	// 1. Try decoding id_token payload
	if token.IDToken != "" {
		parts := strings.Split(token.IDToken, ".")
		if len(parts) >= 2 {
			payloadSegment := parts[1]
			// URL-safe unpadded or padded base64
			var raw []byte
			var err error
			if raw, err = base64.RawURLEncoding.DecodeString(payloadSegment); err != nil {
				raw, err = base64.URLEncoding.DecodeString(payloadSegment)
			}
			if err == nil {
				var claims map[string]interface{}
				if err := json.Unmarshal(raw, &claims); err == nil {
					if email, ok := claims["email"].(string); ok && strings.TrimSpace(email) != "" {
						return strings.ToLower(strings.TrimSpace(email))
					}
				}
			}
		}
	}

	// 2. Try raw payload inspection if present
	if len(token.RawPayload) > 0 {
		var rawMap map[string]interface{}
		if err := json.Unmarshal(token.RawPayload, &rawMap); err == nil {
			if email, ok := rawMap["email"].(string); ok && strings.TrimSpace(email) != "" {
				return strings.ToLower(strings.TrimSpace(email))
			}
		}
	}

	return ""
}

// CaptureCurrentSession captures the active Antigravity OAuth session from Windows Credential Manager
// and persists it into accounts_store/<email>.json.
func (m *AccountManager) CaptureCurrentSession() (string, error) {
	m.mu.RLock()
	reader := m.wincredReader
	m.mu.RUnlock()

	if reader == nil {
		return "", errors.New("wincred reader not configured")
	}

	token, err := reader()
	if err != nil {
		return "", fmt.Errorf("failed reading active credential: %w", err)
	}
	if token == nil || !token.IsValid() {
		return "", ErrTokenInvalid
	}

	email := ExtractEmailFromToken(token)
	if email == "" {
		return "", errors.New("could not extract email from active credential")
	}

	if err := m.store.SaveAccount(email, token); err != nil {
		return "", fmt.Errorf("failed saving account %s: %w", email, err)
	}

	return email, nil
}

// GetCurrentActiveEmail resolves the identity currently running in Antigravity.
// Prioritizes live RAM status from language_server.exe, falling back to Windows Credential Manager.
func (m *AccountManager) GetCurrentActiveEmail(ctx context.Context) (string, error) {
	m.mu.RLock()
	finder := m.clientFinder
	reader := m.wincredReader
	m.mu.RUnlock()

	// 1. Try language server live RAM
	if finder != nil {
		if client, err := finder(ctx); err == nil && client != nil {
			if status, err := client.GetUserStatus(ctx); err == nil && status != nil && status.Email != "" {
				return strings.ToLower(strings.TrimSpace(status.Email)), nil
			}
		}
	}

	// 2. Fallback to Windows Credential Manager
	if reader != nil {
		if token, err := reader(); err == nil && token != nil {
			if email := ExtractEmailFromToken(token); email != "" {
				return email, nil
			}
		}
	}

	return "", nil
}

// GetAccountStates merges live Chrome identities, remembered identities, and
// credentials. Partial capture/catalog errors are returned as visible warnings.
func (m *AccountManager) GetAccountStates(ctx context.Context) (*AccountOverview, error) {
	var warnings []string
	if _, err := m.CaptureCurrentSession(); err != nil && !errors.Is(err, wincred.ErrNotFound) {
		warnings = append(warnings, fmt.Sprintf("Không lưu được phiên hiện tại: %v", err))
	}
	activeEmail, _ := m.GetCurrentActiveEmail(ctx)

	m.mu.RLock()
	cDir, finder, discovered := m.chromeDir, m.clientFinder, m.discoveredStore
	m.mu.RUnlock()

	chromeAccounts, scanErr := scanner.ScanChromeAccountDetails(cDir)
	if scanErr != nil {
		warnings = append(warnings, fmt.Sprintf("Không quét được Chrome: %v", scanErr))
	}
	storedEmails, storeErr := m.store.ListAccounts()
	if storeErr != nil {
		warnings = append(warnings, fmt.Sprintf("Không đọc được kho tài khoản: %v", storeErr))
	}
	storedSet := make(map[string]bool)
	for _, email := range storedEmails {
		storedSet[strings.ToLower(strings.TrimSpace(email))] = true
	}

	itemsMap := make(map[string]*AccountDisplayItem)
	add := func(email string) *AccountDisplayItem {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			return nil
		}
		if item := itemsMap[email]; item != nil {
			return item
		}
		item := &AccountDisplayItem{Email: email}
		itemsMap[email] = item
		return item
	}

	var catalogWriteErr error
	for _, ca := range chromeAccounts {
		item := add(ca.Email)
		if item == nil {
			continue
		}
		item.Name, item.AvatarURL, item.ProfileDir = ca.Name, ca.AvatarURL, ca.ProfileDir
		if discovered != nil {
			if err := discovered.SaveDiscovered(item.Email); err != nil && catalogWriteErr == nil {
				catalogWriteErr = err
			}
		}
	}
	if catalogWriteErr != nil {
		warnings = append(warnings, fmt.Sprintf("Không lưu được danh bạ Chrome: %v", catalogWriteErr))
	}
	if discovered != nil {
		emails, err := discovered.ListDiscovered()
		for _, email := range emails {
			add(email)
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Không đọc đầy đủ danh bạ Chrome: %v", err))
		}
	}
	for _, email := range storedEmails {
		add(email)
	}
	// Live identity is still displayable when its credential cannot be saved.
	add(activeEmail)

	accounts := make([]AccountDisplayItem, 0, len(itemsMap))
	for _, item := range itemsMap {
		item.HasToken = storedSet[item.Email]
		item.IsActive = activeEmail != "" && item.Email == activeEmail
		switch {
		case item.IsActive:
			item.State, item.DisplayBadge = model.StateActive, "[✔ Dùng]"
		case item.HasToken:
			item.State, item.DisplayBadge = model.StateReady, "[Chuyển]"
		default:
			item.State, item.DisplayBadge = model.StateUnauth, "[Cần OAuth]"
		}
		accounts = append(accounts, *item)
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].IsActive != accounts[j].IsActive {
			return accounts[i].IsActive
		}
		if accounts[i].HasToken != accounts[j].HasToken {
			return accounts[i].HasToken
		}
		return accounts[i].Email < accounts[j].Email
	})

	overview := &AccountOverview{ActiveEmail: activeEmail, Accounts: accounts, Warnings: warnings}
	if finder != nil {
		if client, err := finder(ctx); err == nil && client != nil {
			overview.LanguageServerRunning = true
			overview.ServerInfo = client.ServerInfo()
		}
	}
	return overview, nil
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
	start := time.Now()
	normTarget := strings.ToLower(strings.TrimSpace(targetEmail))
	if normTarget == "" {
		return nil, ErrEmptyTargetEmail
	}

	timelineDir := DefaultSwitchTimelineDir
	if m.store != nil && m.store.GetStoreDir() != "" {
		timelineDir = filepath.Join(filepath.Dir(m.store.GetStoreDir()), "logs", "switch_timeline")
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
	_, _ = m.CaptureCurrentSession()

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
		newInfo, clientErr = connector.DiscoverLanguageServer()
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
		rootDir := filepath.Dir(m.store.GetStoreDir())
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
