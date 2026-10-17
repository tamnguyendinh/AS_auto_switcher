package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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
	client QuotaClientInterface
	cache  *QuotaCache
	source AccountSource
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

// Cache returns the underlying QuotaCache.
func (p *TelemetryPool) Cache() *QuotaCache {
	return p.cache
}

func (p *TelemetryPool) pollAccountWithToken(ctx context.Context, email, refreshToken string) (*AccountQuotaData, error) {
	cleanEmail := strings.TrimSpace(email)
	cleanToken := strings.TrimSpace(refreshToken)

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
func (p *TelemetryPool) PollAll(ctx context.Context) ([]*AccountQuotaData, error) {
	pollCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		pollCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	accounts, err := p.source.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	if len(accounts) == 0 {
		return []*AccountQuotaData{}, nil
	}

	results := make([]*AccountQuotaData, len(accounts))
	var wg sync.WaitGroup

	for i, acc := range accounts {
		wg.Add(1)
		go func(idx int, info AccountTokenInfo) {
			defer wg.Done()
			data, _ := p.pollAccountWithToken(pollCtx, info.Email, info.RefreshToken)
			results[idx] = data
		}(i, acc)
	}

	wg.Wait()

	// Persist snapshot to disk
	_ = p.cache.SaveToDisk()

	return results, nil
}
