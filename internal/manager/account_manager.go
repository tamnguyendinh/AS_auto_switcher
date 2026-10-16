package manager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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
	IsActive     bool               `json:"is_active"`
	HasToken     bool               `json:"has_token"`
	DisplayBadge string             `json:"display_badge"`
}

// AccountOverview encapsulates the current state of all accounts and language server status.
type AccountOverview struct {
	ActiveEmail           string               `json:"active_email"`
	Accounts              []AccountDisplayItem `json:"accounts"`
	LanguageServerRunning bool                 `json:"language_server_running"`
	ServerInfo            *connector.ServerInfo `json:"server_info,omitempty"`
}

// SwitchResult contains the outcome and latency of an account hot-swap operation.
type SwitchResult struct {
	Success       bool                    `json:"success"`
	Message       string                  `json:"message"`
	TargetEmail   string                  `json:"target_email"`
	PreviousEmail string                  `json:"previous_email"`
	Duration      time.Duration           `json:"duration"`
	ActiveThreads []reloader.ActiveThread `json:"active_threads,omitempty"`
}

// AccountManager orchestrates AccountStore, WinCred, ChromeScanner, Connector, and Reloader.
type AccountManager struct {
	mu            sync.RWMutex
	store         *store.AccountStore
	reloader      *reloader.SessionReloader
	chromeDir     string
	clientFinder  func(ctx context.Context) (*connector.Client, error)
	wincredReader func() (*model.TokenData, error)
	wincredWriter func(token *model.TokenData) error
	oauthRefresher connector.OAuthRefresher
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

	mgr := &AccountManager{
		store:         accStore,
		reloader:      sessionReloader,
		chromeDir:     scanner.GetDefaultChromeUserDataDir(),
		wincredReader: wincred.ReadToken,
		wincredWriter: wincred.WriteToken,
		oauthRefresher: connector.NewGoogleOAuthRefresher(nil),
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

// GetAccountStates returns the comprehensive state of all accounts across Chrome and accounts_store/.
func (m *AccountManager) GetAccountStates(ctx context.Context) (*AccountOverview, error) {
	// Proactively capture active session into store if possible
	_, _ = m.CaptureCurrentSession()

	activeEmail, _ := m.GetCurrentActiveEmail(ctx)

	m.mu.RLock()
	cDir := m.chromeDir
	finder := m.clientFinder
	m.mu.RUnlock()

	// Discover Chrome accounts
	chromeAccounts, _ := scanner.ScanChromeAccountDetails(cDir)

	// Discover stored JSON accounts
	storedEmails, _ := m.store.ListAccounts()
	storedSet := make(map[string]bool)
	for _, email := range storedEmails {
		storedSet[strings.ToLower(strings.TrimSpace(email))] = true
	}

	// Deduplicate accounts
	itemsMap := make(map[string]*AccountDisplayItem)

	// Add Chrome accounts
	for _, ca := range chromeAccounts {
		norm := strings.ToLower(strings.TrimSpace(ca.Email))
		if norm == "" {
			continue
		}

		item := &AccountDisplayItem{
			Email:      norm,
			Name:       ca.Name,
			AvatarURL:  ca.AvatarURL,
			ProfileDir: ca.ProfileDir,
		}

		if activeEmail != "" && norm == activeEmail {
			item.IsActive = true
			item.State = model.StateActive
			item.DisplayBadge = "[ĐANG DÙNG]"
		} else if storedSet[norm] {
			item.HasToken = true
			item.State = model.StateReady
			item.DisplayBadge = "[⚡ Chuyển]"
		} else {
			item.HasToken = false
			item.State = model.StateUnauth
			item.DisplayBadge = "[Chưa lưu]"
		}

		itemsMap[norm] = item
	}

	// Add stored accounts that may not be in Chrome profiles
	for _, email := range storedEmails {
		norm := strings.ToLower(strings.TrimSpace(email))
		if norm == "" {
			continue
		}

		if _, exists := itemsMap[norm]; !exists {
			item := &AccountDisplayItem{
				Email:    norm,
				HasToken: true,
			}
			if activeEmail != "" && norm == activeEmail {
				item.IsActive = true
				item.State = model.StateActive
				item.DisplayBadge = "[ĐANG DÙNG]"
			} else {
				item.State = model.StateReady
				item.DisplayBadge = "[⚡ Chuyển]"
			}
			itemsMap[norm] = item
		} else {
			itemsMap[norm].HasToken = true
		}
	}

	// Sort accounts: Active first, then Ready, then Unauth, then alphabetically by email
	accountsList := make([]AccountDisplayItem, 0, len(itemsMap))
	for _, item := range itemsMap {
		accountsList = append(accountsList, *item)
	}

	sort.Slice(accountsList, func(i, j int) bool {
		if accountsList[i].IsActive != accountsList[j].IsActive {
			return accountsList[i].IsActive // Active first
		}
		if (accountsList[i].State == model.StateReady) != (accountsList[j].State == model.StateReady) {
			return accountsList[i].State == model.StateReady // Ready second
		}
		return accountsList[i].Email < accountsList[j].Email
	})

	overview := &AccountOverview{
		ActiveEmail: activeEmail,
		Accounts:    accountsList,
	}

	// Check Language Server status
	if finder != nil {
		if client, err := finder(ctx); err == nil && client != nil {
			overview.LanguageServerRunning = true
			overview.ServerInfo = client.ServerInfo()
		}
	}

	return overview, nil
}

// SwitchAccount performs a closed-loop hot-swap:
// 1. Probe live RAM -> verify if already active on targetEmail (no-op).
// 2. Snapshot active threads BEFORE switch to preserve in-flight conversation state (P0-04).
// 3. Auto-capture active session to ensure credentials are safe.
// 4. Load target token from store.
// 5. Parity with Python core/state_resolver.py:380-425:
//    Default is False (smooth zero-chrome hot-swap without OAuth popups).
//    NEVER call AuthLogout during hot-swap unless forceLogout is explicitly true! (P0-02).
// 6. Write target token from accounts_store/<email>.json to Windows Credential Manager.
// 7. Pre-patch ToS in app_storage.json (jetski.onboarding.lastLoginIsGcpTos = "false").
// 8. Ensure Antigravity window in foreground.
// 9. Recycle language_server.exe via taskkill /F and wait for monitorLsCrashInternal to respawn (P0-03).
// 10. Re-probe fresh Language Server instance and verify active identity via GetUserStatus.
// 11. In-Flight Session Continuation (SmartFocus -> LoadTrajectory -> SendUserCascadeMessage) (P0-04).
func (m *AccountManager) SwitchAccount(ctx context.Context, targetEmail string, forceLogout ...bool) (*SwitchResult, error) {
	start := time.Now()
	normTarget := strings.ToLower(strings.TrimSpace(targetEmail))
	if normTarget == "" {
		return nil, ErrEmptyTargetEmail
	}

	var shouldLogout bool
	if len(forceLogout) > 0 {
		shouldLogout = forceLogout[0]
	}

	m.mu.RLock()
	finder := m.clientFinder
	writer := m.wincredWriter
	rel := m.reloader
	refresher := m.oauthRefresher
	m.mu.RUnlock()

	// 1. Probe live RAM / current active email
	var client *connector.Client
	if finder != nil {
		client, _ = finder(ctx)
	}

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

	// 2. Snapshot active threads BEFORE recycling to preserve in-flight conversation state (P0-04)
	var activeThreads []reloader.ActiveThread
	if rel != nil {
		_, activeThreads, _ = rel.CaptureActiveThreadsSnapshot("", "", "")
	} else {
		_, activeThreads, _ = reloader.CaptureActiveThreadsSnapshot("", "", "")
	}

	// 3. Auto-capture active session before switching
	_, _ = m.CaptureCurrentSession()

	// 4. Load target token from store
	targetToken, err := m.store.GetAccount(normTarget)
	if err != nil {
		return &SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Chưa có file JSON của %s trong accounts_store/!", normTarget),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, ErrTargetNotFound
	}

	// 4.1 Proactive Token Expiry Verification & Auto-Refresh (Slice 2 P1-B)
	// If target token is expired or within 5 minutes of expiration, proactively refresh via Google OAuth
	// and update both disk store and in-memory targetToken before writing to Windows Credential Manager.
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

	// 5. Parity with Python core/state_resolver.py:380-425:
	// Default is False (smooth zero-chrome hot-swap without OAuth popups).
	// NEVER call AuthLogout during hot-swap unless forceLogout is explicitly true! (P0-02)
	if shouldLogout && client != nil {
		_ = client.AuthLogout(ctx)
	}

	// 5.1 Pre-sync app_storage.json ToS acceptance to eliminate Google Cloud ToS modal dialog (Slice 3 P1-C)
	// Must be performed BEFORE writing to Windows Credential Manager and BEFORE reauth/reload.
	isGcpTos := false
	if targetToken != nil && strings.EqualFold(targetToken.AuthMethod, "gcp") {
		isGcpTos = true
	}
	storagePath := ""
	if rel != nil {
		storagePath = rel.StoragePath
	}
	_ = reloader.SyncAppStorageToS(normTarget, isGcpTos, storagePath)

	// 6. Write target token to Windows Credential Manager
	if writer == nil {
		return nil, errors.New("wincred writer not configured")
	}
	if err := writer(targetToken); err != nil {
		return &SwitchResult{
			Success:       false,
			Message:       fmt.Sprintf("Không thể ghi Token vào Windows Credential Manager: %v", err),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, fmt.Errorf("%w: %v", ErrWriteCredFailed, err)
	}

	// 7. Ensure Antigravity window in foreground
	_ = reloader.EnsureAntigravityForeground()

	// 9. Recycle Language Server: terminates language_server.exe cleanly and waits for
	// Antigravity's monitorLsCrashInternal to respawn fresh instance with new credentials (P0-03)
	var recycleErr error
	if rel != nil {
		recycleErr = rel.RecycleLanguageServer(ctx, 2*time.Second)
	} else {
		recycleErr = reloader.RecycleLanguageServer(ctx, 2*time.Second)
	}
	if recycleErr != nil {
		return &SwitchResult{
			Success:       true,
			Message:       fmt.Sprintf("Đã chuyển sang %s (cảnh báo khởi động lại Language Server: %v)", normTarget, recycleErr),
			TargetEmail:   normTarget,
			PreviousEmail: currentEmail,
			Duration:      time.Since(start),
			ActiveThreads: activeThreads,
		}, nil
	}

	// 10. Re-probe fresh Language Server (new PID, new Port, new CSRF token)
	var newClient *connector.Client
	if finder != nil {
		newClient, _ = finder(ctx)
	} else {
		if newInfo, err := connector.DiscoverLanguageServer(); err == nil && newInfo != nil {
			newClient = connector.NewClient(newInfo)
		}
	}

	// 11. In-Flight Session Continuation (Pure Native Connect-RPC: SmartFocus -> LoadTrajectory -> SendUserCascadeMessage) (P0-04)
	if newClient != nil && len(activeThreads) > 0 {
		if rel != nil {
			_ = rel.PerformSessionContinuation(ctx, newClient, activeThreads, "")
		} else {
			_ = reloader.PerformSessionContinuation(ctx, newClient, activeThreads, "")
		}
	}

	// 12. Re-verify new active email via GetUserStatus
	if newClient != nil {
		if status, err := newClient.GetUserStatus(ctx); err == nil && status != nil && status.Email != "" {
			confirmedEmail := strings.ToLower(strings.TrimSpace(status.Email))
			if confirmedEmail == normTarget {
				return &SwitchResult{
					Success:       true,
					Message:       fmt.Sprintf("Chuyển sang tài khoản %s thành công!", normTarget),
					TargetEmail:   normTarget,
					PreviousEmail: currentEmail,
					Duration:      time.Since(start),
					ActiveThreads: activeThreads,
				}, nil
			}
		}
	}

	return &SwitchResult{
		Success:       true,
		Message:       fmt.Sprintf("Chuyển sang tài khoản %s thành công!", normTarget),
		TargetEmail:   normTarget,
		PreviousEmail: currentEmail,
		Duration:      time.Since(start),
		ActiveThreads: activeThreads,
	}, nil
}
