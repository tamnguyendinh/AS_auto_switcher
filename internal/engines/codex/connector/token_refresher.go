package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"as-auto-switcher/internal/engines/codex/model"
)

const (
	// DefaultOpenAITokenEndpoint is the official OpenAI OAuth token endpoint.
	DefaultOpenAITokenEndpoint = "https://auth.openai.com/oauth/token"

	// DefaultCodexClientID is the official application client ID extracted from the binary and telemetry.
	DefaultCodexClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
)

var (
	// ErrAuthTerminal represents permanent authentication errors (token revoked, expired, password changed, invalid grant).
	// When this error occurs, caller must not retry and should keep the current session.
	ErrAuthTerminal = errors.New("auth terminal error: refresh token is revoked or expired")

	// ErrNetworkLoss represents connection loss, DNS resolution failure, server timeout, or transient 429/5xx errors.
	// When this error occurs, caller may retry (e.g., 3 retries x 10s backoff).
	ErrNetworkLoss = errors.New("network connection error")
)

// TokenRefresher defines the interface for cleaning and refreshing OpenAI OAuth tokens.
type TokenRefresher interface {
	// ShouldPurge checks whether the account tokens need to be refreshed based on the 55-minute industry standard threshold.
	ShouldPurge(authData json.RawMessage, now time.Time) (bool, error)

	// RefreshToken sends a form-urlencoded request to OpenAI to refresh all three tokens,
	// preserving RTR and 100% of custom metadata fields.
	RefreshToken(ctx context.Context, authData json.RawMessage) (json.RawMessage, error)
}

type openAITokenRefresher struct {
	endpoint   string
	httpClient *http.Client
	clientID   string
	nowFunc    func() time.Time
}

// NewTokenRefresher creates a default TokenRefresher instance connecting to OpenAI OAuth endpoint.
func NewTokenRefresher() TokenRefresher {
	return NewTokenRefresherWithOptions(
		DefaultOpenAITokenEndpoint,
		&http.Client{Timeout: 15 * time.Second},
		DefaultCodexClientID,
		time.Now,
	)
}

// NewTokenRefresherWithOptions allows customizing endpoint, HTTP client, clientID, and nowFunc for unit tests.
func NewTokenRefresherWithOptions(
	endpoint string,
	client *http.Client,
	clientID string,
	nowFunc func() time.Time,
) TokenRefresher {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		ep = DefaultOpenAITokenEndpoint
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	cid := strings.TrimSpace(clientID)
	if cid == "" {
		cid = DefaultCodexClientID
	}
	if nowFunc == nil {
		nowFunc = time.Now
	}
	return &openAITokenRefresher{
		endpoint:   ep,
		httpClient: client,
		clientID:   cid,
		nowFunc:    nowFunc,
	}
}

// ShouldPurge checks whether the account tokens need to be refreshed based on the 55-minute industry standard threshold.
func (r *openAITokenRefresher) ShouldPurge(raw json.RawMessage, now time.Time) (bool, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return false, errors.New("empty auth data")
	}

	var auth model.CodexAuthJson
	if err := json.Unmarshal(raw, &auth); err != nil {
		return false, fmt.Errorf("unmarshal auth data: %w", err)
	}

	if strings.TrimSpace(auth.Tokens.RefreshToken) == "" {
		return false, errors.New("missing refresh_token")
	}

	// 1. Priority 1: Decode id_token claims to check remaining TTL (standard 60m lifetime)
	if strings.TrimSpace(auth.Tokens.IDToken) != "" {
		claims, err := model.ParseJWTClaims(auth.Tokens.IDToken)
		if err == nil && claims != nil && claims.Exp > 0 {
			expTime := time.Unix(claims.Exp, 0)
			remaining := expTime.Sub(now)
			// Trigger purge when remaining time <= 5 minutes (i.e. age >= 55 minutes)
			if remaining <= 5*time.Minute {
				return true, nil
			}
			return false, nil
		}
	}

	// 2. Priority 2: Check last_refresh if id_token does not have exp or failed to parse
	if strings.TrimSpace(auth.LastRefresh) != "" {
		lastRefresh, err := time.Parse(time.RFC3339, auth.LastRefresh)
		if err == nil {
			if now.Sub(lastRefresh) >= 55*time.Minute {
				return true, nil
			}
			return false, nil
		}
	}

	// 3. Fallback: If refresh_token exists but no id_token and no last_refresh, purge to obtain fresh tokens
	if strings.TrimSpace(auth.Tokens.IDToken) == "" && strings.TrimSpace(auth.LastRefresh) == "" {
		return true, nil
	}

	return false, nil
}

type openAITokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshToken sends a form-urlencoded request to OpenAI to refresh all three tokens,
// preserving RTR and 100% of custom metadata fields.
func (r *openAITokenRefresher) RefreshToken(ctx context.Context, authData json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(authData)) == 0 {
		return nil, errors.New("empty auth data")
	}

	// Preserve custom/unknown metadata fields: unmarshal into map[string]any
	var authMap map[string]any
	if err := json.Unmarshal(authData, &authMap); err != nil {
		return nil, fmt.Errorf("unmarshal auth data to map: %w", err)
	}

	tokensMap, _ := authMap["tokens"].(map[string]any)
	if tokensMap == nil {
		return nil, fmt.Errorf("%w: missing tokens object in auth data", ErrAuthTerminal)
	}

	currentRefreshToken, _ := tokensMap["refresh_token"].(string)
	currentRefreshToken = strings.TrimSpace(currentRefreshToken)
	if currentRefreshToken == "" {
		return nil, fmt.Errorf("%w: missing refresh_token in auth data", ErrAuthTerminal)
	}

	// Determine client_id (extract dynamically from oauth_provenance metadata or fallback to configured clientID)
	clientID := r.clientID
	if prov, ok := tokensMap["oauth_provenance"].(map[string]any); ok {
		if cid, ok := prov["client_id"].(string); ok && strings.TrimSpace(cid) != "" {
			clientID = strings.TrimSpace(cid)
		}
	}

	// Build form url.Values according to RFC 6749
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", currentRefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute HTTP request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkLoss, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit for safety
	if err != nil {
		return nil, fmt.Errorf("%w: read response body: %v", ErrNetworkLoss, err)
	}

	switch {
	case resp.StatusCode == http.StatusOK:
		var tokenResp openAITokenResponse
		if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
			return nil, fmt.Errorf("decode token response JSON: %w", err)
		}

		if strings.TrimSpace(tokenResp.AccessToken) == "" {
			return nil, errors.New("empty access_token in openai token response")
		}

		// Update access_token
		tokensMap["access_token"] = tokenResp.AccessToken
		if strings.TrimSpace(tokenResp.IDToken) != "" {
			tokensMap["id_token"] = tokenResp.IDToken
		}

		// RTR Protection: If server provides new refresh_token, rotate; if empty, retain existing refresh_token
		if strings.TrimSpace(tokenResp.RefreshToken) != "" {
			tokensMap["refresh_token"] = tokenResp.RefreshToken
		}

		// Preserve 100% of other token metadata (account_id, oauth_provenance, login_owner_id...)
		authMap["tokens"] = tokensMap

		// Update last_refresh timestamp
		nowTime := r.nowFunc()
		authMap["last_refresh"] = nowTime.UTC().Format(time.RFC3339)

		return json.Marshal(authMap)

	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		// Permanent authorization failure: refresh token revoked, expired, or invalid grant
		return nil, fmt.Errorf("%w: openai token refresh failed (HTTP %d): %s", ErrAuthTerminal, resp.StatusCode, strings.TrimSpace(string(bodyBytes)))

	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError:
		// Transient network or server failure: rate limit (429) or 5xx server error -> retryable
		return nil, fmt.Errorf("%w: openai server error (HTTP %d): %s", ErrNetworkLoss, resp.StatusCode, strings.TrimSpace(string(bodyBytes)))

	default:
		return nil, fmt.Errorf("%w: unexpected HTTP status %d: %s", ErrNetworkLoss, resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
}
