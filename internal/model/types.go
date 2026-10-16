package model

import (
	"encoding/json"
	"strings"
	"sync"
	"time"
)

// AccountState represents the lifecycle state of an account in Antigravity Switcher.
type AccountState string

const (
	StateActive         AccountState = "ACTIVE"
	StateReady          AccountState = "READY"
	StateQuotaExhausted AccountState = "QUOTA_EXHAUSTED"
	StateUnauth         AccountState = "UNAUTH"
	StateError          AccountState = "ERROR"
)

// TokenData holds OAuth credential tokens and metadata.
// It is designed to transparently handle both flat JSON schemas and Antigravity's nested
// {"token": {...}, "id_token": "...", "auth_method": "..."} format.
type TokenData struct {
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	TokenType    string    `json:"token_type,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
	AuthMethod   string    `json:"auth_method,omitempty"`

	// RawPayload stores the unmodified raw JSON bytes for exact round-trip preservation.
	RawPayload json.RawMessage `json:"-"`
}

// nestedTokenPayload represents the inner token object in Antigravity's credential format.
type nestedTokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expiry       string `json:"expiry"`
}

type fullCredentialEnvelope struct {
	Token        *nestedTokenPayload `json:"token,omitempty"`
	AccessToken  string              `json:"access_token,omitempty"`
	RefreshToken string              `json:"refresh_token,omitempty"`
	TokenType    string              `json:"token_type,omitempty"`
	Expiry       interface{}         `json:"expiry,omitempty"`
	IDToken      string              `json:"id_token,omitempty"`
	AuthMethod   string              `json:"auth_method,omitempty"`
}

// UnmarshalJSON parses both flat and nested token structures.
func (t *TokenData) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}

	var env fullCredentialEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}

	// Preserve raw bytes
	t.RawPayload = make(json.RawMessage, len(data))
	copy(t.RawPayload, data)

	t.IDToken = env.IDToken
	t.AuthMethod = env.AuthMethod

	// Check if nested "token" object is present (standard Antigravity format)
	if env.Token != nil {
		t.AccessToken = env.Token.AccessToken
		t.RefreshToken = env.Token.RefreshToken
		t.TokenType = env.Token.TokenType
		if env.Token.Expiry != "" {
			if parsedExpiry, err := time.Parse(time.RFC3339Nano, env.Token.Expiry); err == nil {
				t.Expiry = parsedExpiry
			} else if parsedExpiry, err := time.Parse(time.RFC3339, env.Token.Expiry); err == nil {
				t.Expiry = parsedExpiry
			}
		}
	}

	// Fallback to top-level fields if not set from nested object
	if t.AccessToken == "" {
		t.AccessToken = env.AccessToken
	}
	if t.RefreshToken == "" {
		t.RefreshToken = env.RefreshToken
	}
	if t.TokenType == "" {
		t.TokenType = env.TokenType
	}
	if t.Expiry.IsZero() && env.Expiry != nil {
		switch v := env.Expiry.(type) {
		case string:
			if parsedExpiry, err := time.Parse(time.RFC3339Nano, v); err == nil {
				t.Expiry = parsedExpiry
			} else if parsedExpiry, err := time.Parse(time.RFC3339, v); err == nil {
				t.Expiry = parsedExpiry
			}
		case float64:
			t.Expiry = time.Unix(int64(v), 0)
		}
	}

	if t.TokenType == "" {
		t.TokenType = "Bearer"
	}

	return nil
}

// MarshalJSON serializes the token. If RawPayload is present, it writes the exact raw JSON
// to maintain byte-level fidelity. Otherwise, it marshals a dual envelope containing both
// top-level fields and the nested "token" object expected by Antigravity language_server.exe.
func (t *TokenData) MarshalJSON() ([]byte, error) {
	if len(t.RawPayload) > 0 {
		return t.RawPayload, nil
	}

	var expiryStr string
	if !t.Expiry.IsZero() {
		expiryStr = t.Expiry.Format(time.RFC3339Nano)
	}

	tokenType := t.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	authMethod := t.AuthMethod
	if authMethod == "" {
		authMethod = "consumer"
	}

	output := map[string]interface{}{
		"access_token":  t.AccessToken,
		"refresh_token": t.RefreshToken,
		"token_type":    tokenType,
		"id_token":      t.IDToken,
		"auth_method":   authMethod,
		"token": map[string]interface{}{
			"access_token":  t.AccessToken,
			"refresh_token": t.RefreshToken,
			"token_type":    tokenType,
			"expiry":        expiryStr,
		},
	}
	if !t.Expiry.IsZero() {
		output["expiry"] = expiryStr
	}

	return json.Marshal(output)
}

// IsValid checks whether this token contains at least an access token or refresh token.
func (t *TokenData) IsValid() bool {
	if t == nil {
		return false
	}
	return strings.TrimSpace(t.AccessToken) != "" || strings.TrimSpace(t.RefreshToken) != ""
}

// UpdateTokens updates the token fields and clears RawPayload to ensure subsequent JSON serialization
// outputs the newly refreshed tokens.
func (t *TokenData) UpdateTokens(accessToken, refreshToken, idToken string, expiry time.Time) {
	if t == nil {
		return
	}
	if accessToken != "" {
		t.AccessToken = accessToken
	}
	if refreshToken != "" {
		t.RefreshToken = refreshToken
	}
	if idToken != "" {
		t.IDToken = idToken
	}
	if !expiry.IsZero() {
		t.Expiry = expiry
	}
	t.RawPayload = nil
}

// AccountRecord represents a single account managed by Antigravity Switcher.
type AccountRecord struct {
	Email                  string       `json:"email"`
	State                  AccountState `json:"state"`
	Token                  *TokenData   `json:"token,omitempty"`
	LastSwitchedAt         time.Time    `json:"last_switched_at"`
	CooldownUntil          time.Time    `json:"cooldown_until"`
	ConsecutiveExhaustions int          `json:"consecutive_exhaustions"`
	TotalSwitches          int          `json:"total_switches"`
	ErrorMessage           string       `json:"error_message,omitempty"`
}

// IsEligibleForSwitch returns true if the account is ready for hot-swap.
func (a *AccountRecord) IsEligibleForSwitch() bool {
	if a == nil {
		return false
	}
	return a.State == StateReady
}

// MarkExhausted updates the account state to QUOTA_EXHAUSTED with exponential backoff:
// 1x, 2x, 4x, 8x capped at 2 hours.
func (a *AccountRecord) MarkExhausted(baseCooldown time.Duration) {
	if a == nil {
		return
	}
	a.State = StateQuotaExhausted
	a.ConsecutiveExhaustions++
	shift := a.ConsecutiveExhaustions - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 3 {
		shift = 3 // 1 << 3 = 8x multiplier maximum
	}
	multiplier := 1 << shift
	duration := baseCooldown * time.Duration(multiplier)
	maxCooldown := 2 * time.Hour
	if duration > maxCooldown {
		duration = maxCooldown
	}
	a.CooldownUntil = time.Now().Add(duration)
}

// CheckCooldownExpiry checks if the cooldown period has ended and restores StateReady.
func (a *AccountRecord) CheckCooldownExpiry() bool {
	if a == nil {
		return false
	}
	if a.State == StateQuotaExhausted && !a.CooldownUntil.IsZero() && time.Now().After(a.CooldownUntil) {
		a.State = StateReady
		a.CooldownUntil = time.Time{}
		return true
	}
	return false
}

// AccountPool manages the concurrent, thread-safe set of account records.
type AccountPool struct {
	mu       sync.RWMutex
	accounts map[string]*AccountRecord
	order    []string
}

// NewAccountPool creates an empty, initialized AccountPool.
func NewAccountPool() *AccountPool {
	return &AccountPool{
		accounts: make(map[string]*AccountRecord),
		order:    make([]string, 0),
	}
}

// AddAccount adds or updates an account record in the pool.
func (p *AccountPool) AddAccount(record *AccountRecord) {
	if record == nil || record.Email == "" {
		return
	}
	norm := strings.ToLower(strings.TrimSpace(record.Email))
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.accounts[norm]; !exists {
		p.order = append(p.order, norm)
	}
	p.accounts[norm] = record
}

// GetAccount retrieves an account by email.
func (p *AccountPool) GetAccount(email string) (*AccountRecord, bool) {
	norm := strings.ToLower(strings.TrimSpace(email))
	p.mu.RLock()
	defer p.mu.RUnlock()

	acc, exists := p.accounts[norm]
	return acc, exists
}

// GetActiveAccount returns the single currently active account, if any.
func (p *AccountPool) GetActiveAccount() (*AccountRecord, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, acc := range p.accounts {
		if acc.State == StateActive {
			return acc, true
		}
	}
	return nil, false
}

// GetReadyAccounts returns all accounts in StateReady.
func (p *AccountPool) GetReadyAccounts() []*AccountRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ready := make([]*AccountRecord, 0)
	for _, email := range p.order {
		if acc, ok := p.accounts[email]; ok && acc.State == StateReady {
			ready = append(ready, acc)
		}
	}
	return ready
}

// GetAllAccounts returns all accounts in the pool ordered by insertion.
func (p *AccountPool) GetAllAccounts() []*AccountRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()

	res := make([]*AccountRecord, 0, len(p.order))
	for _, email := range p.order {
		if acc, ok := p.accounts[email]; ok {
			res = append(res, acc)
		}
	}
	return res
}

// MarkActive marks the given account as StateActive and resets any other active account to StateReady.
func (p *AccountPool) MarkActive(email string) {
	norm := strings.ToLower(strings.TrimSpace(email))
	p.mu.Lock()
	defer p.mu.Unlock()

	for e, acc := range p.accounts {
		if e == norm {
			acc.State = StateActive
			acc.LastSwitchedAt = time.Now()
			acc.TotalSwitches++
		} else if acc.State == StateActive {
			acc.State = StateReady
		}
	}
}

// MarkExhausted marks the given account as StateQuotaExhausted.
func (p *AccountPool) MarkExhausted(email string, baseCooldown time.Duration) {
	norm := strings.ToLower(strings.TrimSpace(email))
	p.mu.Lock()
	defer p.mu.Unlock()

	if acc, ok := p.accounts[norm]; ok {
		acc.MarkExhausted(baseCooldown)
	}
}

// FailoverConfig defines runtime parameters for autonomous failover.
type FailoverConfig struct {
	MinDwellTime          time.Duration `json:"min_dwell_time"`           // Default: 15s
	MaxSwitchesPerWindow  int           `json:"max_switches_per_window"`  // Default: 4
	SlidingWindowDuration time.Duration `json:"sliding_window_duration"` // Default: 120s
	DefaultCooldown       time.Duration `json:"default_cooldown"`        // Default: 15m
	AutoWindowReload      bool          `json:"auto_window_reload"`       // Default: false
	EnableNotifications   bool          `json:"enable_notifications"`    // Default: true
}

// DefaultFailoverConfig returns standard failover safety settings.
func DefaultFailoverConfig() FailoverConfig {
	return FailoverConfig{
		MinDwellTime:          15 * time.Second,
		MaxSwitchesPerWindow:  4,
		SlidingWindowDuration: 120 * time.Second,
		DefaultCooldown:       15 * time.Minute,
		AutoWindowReload:      false,
		EnableNotifications:   true,
	}
}
