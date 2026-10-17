package telemetry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/store"
)

var (
	// ErrNilClient is returned when QuotaClient is nil.
	ErrNilClient = errors.New("quota client cannot be nil")

	// ErrNilCache is returned when QuotaCache is nil.
	ErrNilCache = errors.New("quota cache cannot be nil")

	// ErrNilSource is returned when AccountSource is nil.
	ErrNilSource = errors.New("account source cannot be nil")
)

// AccountTokenInfo holds minimal account identifier and refresh token data for polling.
type AccountTokenInfo struct {
	Email        string
	RefreshToken string
}

// AccountSource defines the interface for retrieving accounts and credentials.
type AccountSource interface {
	ListAccounts() ([]AccountTokenInfo, error)
	GetRefreshToken(email string) (string, error)
}

// StoreAccountSource adapts store.AccountStore to the AccountSource interface.
type StoreAccountSource struct {
	store *store.AccountStore
}

// NewStoreAccountSource creates an AccountSource backed by an AccountStore.
func NewStoreAccountSource(s *store.AccountStore) *StoreAccountSource {
	return &StoreAccountSource{store: s}
}

// ListAccounts retrieves all stored accounts and their refresh tokens from the store.
func (s *StoreAccountSource) ListAccounts() ([]AccountTokenInfo, error) {
	if s.store == nil {
		return nil, errors.New("account store is nil")
	}
	emails, err := s.store.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts from store: %w", err)
	}

	var results []AccountTokenInfo
	for _, email := range emails {
		clean := strings.TrimSpace(email)
		if clean == "" {
			continue
		}
		rt := ""
		token, err := s.store.GetAccount(clean)
		if err == nil && token != nil {
			rt = token.RefreshToken
		}
		results = append(results, AccountTokenInfo{
			Email:        clean,
			RefreshToken: rt,
		})
	}
	return results, nil
}

// GetRefreshToken retrieves the refresh token for a specific email.
func (s *StoreAccountSource) GetRefreshToken(email string) (string, error) {
	if s.store == nil {
		return "", errors.New("account store is nil")
	}
	token, err := s.store.GetAccount(email)
	if err != nil {
		return "", fmt.Errorf("failed to get account %s: %w", email, err)
	}
	if token == nil || strings.TrimSpace(token.RefreshToken) == "" {
		return "", fmt.Errorf("no refresh token for account %s", email)
	}
	return token.RefreshToken, nil
}

// TelemetryPool manages concurrent querying of quota telemetry across multiple accounts.
type TelemetryPool struct {
	client            QuotaClientInterface
	cache             *QuotaCache
	source            AccountSource
	localQuotaFetcher func(ctx context.Context, email string) (*AccountQuotaData, error)
	serverInfoMu      sync.RWMutex
	cachedServerInfo  *connector.ServerInfo
	cachedServerTime  time.Time
}

func (p *TelemetryPool) getServerInfo() (*connector.ServerInfo, error) {
	p.serverInfoMu.RLock()
	if time.Since(p.cachedServerTime) < 5*time.Second {
		info := p.cachedServerInfo
		p.serverInfoMu.RUnlock()
		if info == nil {
			return nil, errors.New("language server not running")
		}
		return info, nil
	}
	p.serverInfoMu.RUnlock()

	p.serverInfoMu.Lock()
	defer p.serverInfoMu.Unlock()
	if time.Since(p.cachedServerTime) < 5*time.Second {
		if p.cachedServerInfo == nil {
			return nil, errors.New("language server not running")
		}
		return p.cachedServerInfo, nil
	}

	info, err := connector.DiscoverLanguageServer()
	p.cachedServerTime = time.Now()
	p.cachedServerInfo = info
	if err != nil || info == nil {
		return nil, errors.New("language server not running")
	}
	return info, nil
}

// QuotaClientInterface allows mocking QuotaClient in tests.
type QuotaClientInterface interface {
	RetrieveUserQuota(ctx context.Context, refreshToken string) (*AccountQuotaData, error)
}

// NewTelemetryPool initializes a TelemetryPool backed by store.AccountStore.
func NewTelemetryPool(client QuotaClientInterface, cache *QuotaCache, accountStore *store.AccountStore) (*TelemetryPool, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	if cache == nil {
		return nil, ErrNilCache
	}
	return &TelemetryPool{
		client: client,
		cache:  cache,
		source: NewStoreAccountSource(accountStore),
	}, nil
}

// NewTelemetryPoolWithSource initializes a TelemetryPool backed by any AccountSource.
func NewTelemetryPoolWithSource(client QuotaClientInterface, cache *QuotaCache, source AccountSource) (*TelemetryPool, error) {
	if client == nil {
		return nil, ErrNilClient
	}
	if cache == nil {
		return nil, ErrNilCache
	}
	if source == nil {
		return nil, ErrNilSource
	}
	return &TelemetryPool{
		client: client,
		cache:  cache,
		source: source,
	}, nil
}

// SetLocalQuotaFetcher overrides the local language server quota fetcher (used in unit tests).
func (p *TelemetryPool) SetLocalQuotaFetcher(fn func(ctx context.Context, email string) (*AccountQuotaData, error)) {
	p.localQuotaFetcher = fn
}

// Cache returns the underlying QuotaCache.
func (p *TelemetryPool) Cache() *QuotaCache {
	return p.cache
}

func (p *TelemetryPool) pollFromLanguageServer(ctx context.Context, email string) (*AccountQuotaData, error) {
	if p.localQuotaFetcher != nil {
		return p.localQuotaFetcher(ctx, email)
	}

	// 1. Discover language server (cached)
	serverInfo, err := p.getServerInfo()
	if err != nil || serverInfo == nil {
		return nil, errors.New("language server not running")
	}

	// 2. Check active user identity
	client := connector.NewClient(serverInfo)
	userStatus, err := client.GetUserStatus(ctx)
	if err != nil || userStatus == nil || strings.TrimSpace(userStatus.Email) == "" {
		return nil, errors.New("cannot determine active user from language server")
	}

	// Only active account is queried via Connect-RPC
	if !strings.EqualFold(strings.TrimSpace(userStatus.Email), strings.TrimSpace(email)) {
		return nil, errors.New("account is not active in language server")
	}

	// 3. Query Connect-RPC endpoint
	url := fmt.Sprintf("%s/exa.language_server_pb.LanguageServerService/RetrieveUserQuotaSummary", serverInfo.BaseURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString("{}"))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", serverInfo.CSRFToken)
	req.Header.Set("x-codeium-csrf-token", serverInfo.CSRFToken)

	httpClient := &http.Client{Timeout: 3 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("language server returned HTTP %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	claude, gemini, gpt, resetTime, status, err := ParseQuotaSummary(bodyBytes)
	if err != nil {
		return nil, err
	}

	return &AccountQuotaData{
		Email:       email,
		Claude:      claude,
		Gemini:      gemini,
		GPT:         gpt,
		ResetTime:   resetTime,
		Status:      status,
		LastUpdated: time.Now().UTC(),
	}, nil
}

func (p *TelemetryPool) pollAccountWithToken(ctx context.Context, email, refreshToken string) (*AccountQuotaData, error) {
	cleanEmail := strings.TrimSpace(email)
	cleanToken := strings.TrimSpace(refreshToken)

	// Hybrid Telemetry Step 1: For active account running in language_server.exe,
	// query Connect-RPC /exa.language_server_pb.LanguageServerService/RetrieveUserQuotaSummary
	if data, err := p.pollFromLanguageServer(ctx, cleanEmail); err == nil && data != nil {
		data.Email = cleanEmail
		p.cache.Update(data)
		return data, nil
	}

	// Step 2: Fallback or inactive account -> Google Cloud Code API with dynamic User-Agent
	if cleanToken == "" {
		err := fmt.Errorf("no refresh token available for account %s", cleanEmail)
		errData := &AccountQuotaData{
			Email:        cleanEmail,
			Status:       StatusError,
			ErrorMessage: err.Error(),
			LastUpdated:  time.Now().UTC(),
		}
		p.cache.Update(errData)
		return errData, err
	}

	data, err := p.client.RetrieveUserQuota(ctx, cleanToken)
	if err != nil {
		errData := &AccountQuotaData{
			Email:        cleanEmail,
			Status:       StatusError,
			ErrorMessage: err.Error(),
			LastUpdated:  time.Now().UTC(),
		}
		p.cache.Update(errData)
		return errData, err
	}

	data.Email = cleanEmail
	p.cache.Update(data)
	return data, nil
}

// PollAccount refreshes quota telemetry for a single account.
func (p *TelemetryPool) PollAccount(ctx context.Context, email string) (*AccountQuotaData, error) {
	cleanEmail := strings.TrimSpace(email)
	if cleanEmail == "" {
		return nil, errors.New("account email cannot be empty")
	}

	pollCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	refreshToken, err := p.source.GetRefreshToken(cleanEmail)
	if err != nil {
		// Even if store lookup fails, check local language server if this is the active user
		if data, lsErr := p.pollFromLanguageServer(pollCtx, cleanEmail); lsErr == nil && data != nil {
			data.Email = cleanEmail
			p.cache.Update(data)
			_ = p.cache.SaveToDisk()
			return data, nil
		}

		errData := &AccountQuotaData{
			Email:        cleanEmail,
			Status:       StatusError,
			ErrorMessage: err.Error(),
			LastUpdated:  time.Now().UTC(),
		}
		p.cache.Update(errData)
		_ = p.cache.SaveToDisk()
		return errData, err
	}

	data, err := p.pollAccountWithToken(pollCtx, cleanEmail, refreshToken)
	_ = p.cache.SaveToDisk()
	return data, err
}

// PollAll concurrently queries quota telemetry for all stored accounts.
// Enforces a 5-second per-batch deadline to guarantee total completion in < 2s for normal operations.
// Returns results, count of successful polls, count of errors, and fatal error.
func (p *TelemetryPool) PollAll(ctx context.Context) ([]*AccountQuotaData, int, int, error) {
	pollCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	accounts, err := p.source.ListAccounts()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		return []*AccountQuotaData{}, 0, 0, nil
	}

	results := make([]*AccountQuotaData, len(accounts))
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	for i, acc := range accounts {
		wg.Add(1)
		go func(idx int, info AccountTokenInfo) {
			defer wg.Done()
			data, err := p.pollAccountWithToken(pollCtx, info.Email, info.RefreshToken)
			results[idx] = data
			if err != nil || (data != nil && data.Status == StatusError) {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i, acc)
	}

	wg.Wait()

	// Persist snapshot to disk
	_ = p.cache.SaveToDisk()

	return results, int(successCount.Load()), int(errorCount.Load()), nil
}