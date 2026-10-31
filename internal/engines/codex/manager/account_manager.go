package manager

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"as-auto-switcher/internal/engines/codex/model"
	"as-auto-switcher/internal/engines/codex/reloader"
	"as-auto-switcher/internal/engines/codex/store"
)

var (
	// ErrCannotDeleteActiveAccount is returned when attempting to delete the currently active account.
	ErrCannotDeleteActiveAccount = errors.New("không thể xóa tài khoản đang hoạt động")

	// ErrCannotToggleUnauthAccount is returned when attempting to toggle status on an account requiring OAuth.
	ErrCannotToggleUnauthAccount = errors.New("cannot toggle status of unauthenticated account")

	// ErrTargetUnauth is returned when attempting to switch to an account that requires OAuth authentication.
	ErrTargetUnauth = errors.New("target account requires oauth authentication before switching")

	// ErrEmptyEmail is returned when an empty email is provided to an account management operation.
	ErrEmptyEmail = errors.New("target email cannot be empty")
)

// CodexCardView represents the projected UI card view for a Codex account.
type CodexCardView struct {
	Email     string             `json:"email"`
	PlanType  string             `json:"plan_type"`
	State     model.AccountState `json:"state"`      // [✔ Dùng], [Chuyển], [Cần OAuth]
	Status    string             `json:"status"`     // "on", "off"
	CanDelete bool               `json:"can_delete"` // False if StateActive (Active card protection)
	CanToggle bool               `json:"can_toggle"` // False if StateUnauth
	CreatedAt time.Time          `json:"created_at"`
}

// SessionReloader defines the contract for desktop restart and switching (implemented in Phase 6).
type SessionReloader interface {
	SwitchSession(targetEmail string) error
	GetActiveSessionEmail() (string, error)
}

// AccountManager defines the contract for Codex account lifecycle and UI card projection.
type AccountManager interface {
	BootstrapActiveSession() error
	ListCards() ([]*CodexCardView, error)
	SwitchAccount(email string) error
	DeleteAccount(email string) error
	ToggleStatus(email string) error
	GetActiveEmail() (string, error)
}

// AccountManagerConfig configures an AccountManager instance.
type AccountManagerConfig struct {
	Store          store.AccountStore
	ActiveAuthPath string
	Reloader       SessionReloader
	ActiveResolver func() (string, error)
	Coordinator    *model.Coordinator
}

type accountManager struct {
	store          store.AccountStore
	activeAuthPath string
	reloader       SessionReloader
	activeResolver func() (string, error)
	mu             sync.RWMutex
	gate           *model.Coordinator
}

// NewAccountManager creates a new AccountManager adhering to Section 11 Phase 4 architecture.
func NewAccountManager(cfg AccountManagerConfig) (AccountManager, error) {
	if cfg.Store == nil {
		return nil, errors.New("missing account store in account manager config")
	}

	if cfg.Coordinator == nil {
		cfg.Coordinator = store.Coordinator(cfg.Store)
	}

	return &accountManager{
		store:          cfg.Store,
		activeAuthPath: cfg.ActiveAuthPath,
		gate:           cfg.Coordinator,
		reloader:       cfg.Reloader,
		activeResolver: cfg.ActiveResolver,
	}, nil
}

// BootstrapActiveSession reconciles the active Codex Desktop credential into the store.
// If no active auth file exists on this machine, it returns nil without error.
func (m *accountManager) BootstrapActiveSession() error {
	if r, ok := m.reloader.(interface{ Reconcile() error }); ok {
		return r.Reconcile()
	}
	return reloader.ErrUnsupportedDesktopVerification
}

func (m *accountManager) resolveActiveEmail() (string, error) {
	if m.activeResolver != nil {
		return m.activeResolver()
	}
	if m.reloader != nil {
		return m.reloader.GetActiveSessionEmail()
	}
	return "", reloader.ErrUnsupportedDesktopVerification
}

// GetActiveEmail returns the currently active account email or empty string if none active.
func (m *accountManager) GetActiveEmail() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resolveActiveEmail()
}

// ListCards returns the projected list of cards with 3 badge states: [✔ Dùng], [Chuyển], [Cần OAuth].
// The list is stably sorted by CreatedAt ascending, then Email ascending.
func (m *accountManager) ListCards() ([]*CodexCardView, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	envelopes, err := m.store.ListAccounts()
	if err != nil {
		return nil, fmt.Errorf("list accounts from store: %w", err)
	}

	activeEmail, activeErr := m.resolveActiveEmail()

	cards := make([]*CodexCardView, 0, len(envelopes))
	for _, env := range envelopes {
		normEmail := strings.ToLower(strings.TrimSpace(env.Email))
		var state model.AccountState
		if activeErr != nil {
			state = model.StateUnknown
		} else if activeEmail != "" && strings.EqualFold(normEmail, activeEmail) {
			state = model.StateActive
		} else {
			state = model.StateReady
		}

		status := strings.ToLower(strings.TrimSpace(env.Status))
		if status != "off" {
			status = "on"
		}

		planType := strings.TrimSpace(env.PlanType)
		if planType == "" {
			planType = "free"
		}

		canDelete := activeErr == nil && state != model.StateActive
		canToggle := true

		cards = append(cards, &CodexCardView{
			Email:     normEmail,
			PlanType:  planType,
			State:     state,
			Status:    status,
			CanDelete: canDelete,
			CanToggle: canToggle,
			CreatedAt: env.CreatedAt,
		})
	}

	// Stable Sort: CreatedAt ascending, then Email ascending
	sort.SliceStable(cards, func(i, j int) bool {
		if !cards[i].CreatedAt.Equal(cards[j].CreatedAt) {
			return cards[i].CreatedAt.Before(cards[j].CreatedAt)
		}
		return cards[i].Email < cards[j].Email
	})

	return cards, nil
}

// DeleteAccount deletes an account from the store after enforcing the Active Card Protection Invariant.
func (m *accountManager) DeleteAccount(email string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return ErrEmptyEmail
	}

	m.gate.Lock()
	defer m.gate.Unlock()
	if err := m.gate.Check(""); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Invariant: Verify target is not active
	activeEmail, activeErr := m.resolveActiveEmail()
	if activeErr != nil {
		return activeErr
	}
	if activeEmail != "" && strings.EqualFold(normEmail, activeEmail) {
		return ErrCannotDeleteActiveAccount
	}

	// Also verify via store record existence
	env, err := m.store.GetAccount(normEmail)
	if err != nil {
		return err
	}

	// Invariant: Ensure state is not StateActive
	if activeEmail != "" && strings.EqualFold(env.Email, activeEmail) {
		return ErrCannotDeleteActiveAccount
	}

	return m.store.DeleteAccount(normEmail)
}

// ToggleStatus toggles the status between "on" and "off" for authenticated accounts.
func (m *accountManager) ToggleStatus(email string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return ErrEmptyEmail
	}

	m.gate.Lock()
	defer m.gate.Unlock()
	if err := m.gate.Check(""); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	env, err := m.store.GetAccount(normEmail)
	if err != nil {
		return err
	}

	tokens, _ := env.GetTokens()
	if tokens == nil || !tokens.IsValid() {
		return ErrCannotToggleUnauthAccount
	}

	currStatus := strings.ToLower(strings.TrimSpace(env.Status))
	newStatus := "on"
	if currStatus != "off" {
		newStatus = "off"
	}

	return m.store.SetAccountStatus(normEmail, newStatus)
}

// SwitchAccount initiates account switching to the target email.
// In Phase 4, it validates existence and credential validity, delegating to SessionReloader when present.
func (m *accountManager) SwitchAccount(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ErrEmptyEmail
	}
	if m.reloader == nil {
		return reloader.ErrUnsupportedDesktopVerification
	}
	return m.reloader.SwitchSession(email)
}
