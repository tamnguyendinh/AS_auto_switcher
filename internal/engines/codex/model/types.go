package model

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrInvalidJWT is returned when a JWT token string is malformed or cannot be decoded.
	ErrInvalidJWT = errors.New("invalid jwt token")

	// ErrEmailNotFoundInJWT is returned when no email claim exists in the JWT.
	ErrEmailNotFoundInJWT = errors.New("email claim not found in jwt")
)

// AccountState represents the UI card display badge state.
type AccountState string

const (
	StateUnknown AccountState = "[Chưa xác minh]"
	StateActive  AccountState = "[✔ Dùng]"
	StateReady   AccountState = "[Chuyển]"
	StateUnauth  AccountState = "[Cần OAuth]"
)

// ValidateAuth extracts identity only from a complete ChatGPT credential.
// It validates storage shape, not server-side validity.
func ValidateAuth(raw json.RawMessage) (email, accountID string, err error) {
	var auth CodexAuthJson
	if len(strings.TrimSpace(string(raw))) == 0 || strings.TrimSpace(string(raw))[0] != '{' {
		return "", "", errors.New("invalid credential object")
	}
	if err = json.Unmarshal(raw, &auth); err != nil {
		return "", "", errors.New("invalid credential JSON")
	}
	if auth.AuthMode != "chatgpt" || (auth.OpenAIAPIKey != nil && *auth.OpenAIAPIKey != "") ||
		auth.Tokens.AccessToken == "" || auth.Tokens.RefreshToken == "" || auth.Tokens.AccountID == "" {
		return "", "", errors.New("unsupported or incomplete ChatGPT credential")
	}
	email, err = ExtractEmailFromJWT(auth.Tokens.IDToken)
	if err != nil {
		return "", "", err
	}
	return email, auth.Tokens.AccountID, nil
}

func CredentialGeneration(raw json.RawMessage) string {
	var compact bytes.Buffer
	if json.Compact(&compact, raw) == nil {
		raw = compact.Bytes()
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}

// CodexTokens represents projection fields observed from Codex OAuth.
// The raw auth JSON must always be preserved in AuthData.
type CodexTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	AccountID    string `json:"account_id"`
}

// IsValid checks whether this tokens structure contains at least an access token or refresh token.
func (t *CodexTokens) IsValid() bool {
	if t == nil {
		return false
	}
	return strings.TrimSpace(t.AccessToken) != "" || strings.TrimSpace(t.RefreshToken) != ""
}

// CodexAuthJson represents the credential structure observed in ~/.codex/auth.json.
// Used for validation and metadata extraction; unknown fields are preserved via AuthData.
type CodexAuthJson struct {
	OpenAIAPIKey *string     `json:"OPENAI_API_KEY"`
	AuthMode     string      `json:"auth_mode"`
	LastRefresh  string      `json:"last_refresh"`
	Tokens       CodexTokens `json:"tokens"`
}

// CodexAccountEnvelope is the durable storage schema in accounts_store/codex/<email>.json.
type CodexAccountEnvelope struct {
	Email      string          `json:"email"`
	Status     string          `json:"status"`    // "on" or "off" (default: "on")
	PlanType   string          `json:"plan_type"` // "pro", "plus", "free", "team", etc.
	SavedAt    time.Time       `json:"saved_at"`
	AuthData   json.RawMessage `json:"auth_data"`             // Raw credential JSON preserving all original/unknown fields
	CreatedAt  time.Time       `json:"created_at"`            // Preserves insertion order across reauth
	Revision   uint64          `json:"revision"`              // CAS counter to guard against lost updates
	Generation string          `json:"credential_generation"` // Credential generation identifier
}

// Clone creates a deep copy of CodexAccountEnvelope.
func (e *CodexAccountEnvelope) Clone() *CodexAccountEnvelope {
	if e == nil {
		return nil
	}
	clone := *e
	if len(e.AuthData) > 0 {
		clone.AuthData = append(json.RawMessage(nil), e.AuthData...)
	}
	return &clone
}

// GetTokens decodes and returns CodexTokens from AuthData.
func (e *CodexAccountEnvelope) GetTokens() (*CodexTokens, error) {
	if e == nil || len(e.AuthData) == 0 {
		return nil, errors.New("empty auth data")
	}
	var auth CodexAuthJson
	if err := json.Unmarshal(e.AuthData, &auth); err != nil {
		return nil, fmt.Errorf("decode auth data: %w", err)
	}
	return &auth.Tokens, nil
}

// CodexJWTClaims contains standard identity claims extracted from the id_token payload.
type CodexJWTClaims struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"email_verified"`
	Exp           int64  `json:"exp"`
	Iat           int64  `json:"iat"`
	Iss           string `json:"iss"`
	Sub           string `json:"sub"`
}

// ParseJWTClaims decodes the payload of a standard JWT token into CodexJWTClaims without signature verification.
func ParseJWTClaims(idToken string) (*CodexJWTClaims, error) {
	parts := strings.Split(strings.TrimSpace(idToken), ".")
	if len(parts) < 2 {
		return nil, ErrInvalidJWT
	}

	payloadSegment := parts[1]
	// Handle standard URL base64 padding
	switch len(payloadSegment) % 4 {
	case 2:
		payloadSegment += "=="
	case 3:
		payloadSegment += "="
	}

	payloadBytes, err := base64.URLEncoding.DecodeString(payloadSegment)
	if err != nil {
		// Fallback to RawURLEncoding if padding was already absent
		var rawErr error
		payloadBytes, rawErr = base64.RawURLEncoding.DecodeString(parts[1])
		if rawErr != nil {
			return nil, fmt.Errorf("%w: base64 decode failed: %v", ErrInvalidJWT, err)
		}
	}

	var claims CodexJWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("%w: json decode failed: %v", ErrInvalidJWT, err)
	}

	return &claims, nil
}

// ExtractEmailFromJWT extracts and normalizes the email address from a JWT id_token.
func ExtractEmailFromJWT(idToken string) (string, error) {
	claims, err := ParseJWTClaims(idToken)
	if err != nil {
		return "", err
	}
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return "", ErrEmailNotFoundInJWT
	}
	return email, nil
}
