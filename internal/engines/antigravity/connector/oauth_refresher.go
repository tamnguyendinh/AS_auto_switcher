package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultGoogleClientID is the verified Google CloudCode OAuth client ID.
	DefaultGoogleClientID = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"

	// DefaultGoogleClientSecret is the verified Google CloudCode OAuth client secret.
	DefaultGoogleClientSecret = "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"

	// DefaultGoogleTokenURL is the authoritative Google OAuth token endpoint.
	DefaultGoogleTokenURL = "https://oauth2.googleapis.com/token"
)

var (
	// ErrEmptyRefreshToken is returned when the refresh token passed is empty.
	ErrEmptyRefreshToken = errors.New("refresh token is empty")

	// ErrTokenRefreshFailed is returned when refreshing the token fails.
	ErrTokenRefreshFailed = errors.New("failed to refresh oauth token")
)

// TokenRefreshResult encapsulates the response from the OAuth refresh endpoint.
type TokenRefreshResult struct {
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry"`
}

// OAuthRefresher defines the interface for refreshing OAuth tokens.
type OAuthRefresher interface {
	RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error)
}

// GoogleOAuthRefresher implements OAuthRefresher for Google CloudCode OAuth credentials.
type GoogleOAuthRefresher struct {
	clientID     string
	clientSecret string
	tokenURL     string
	httpClient   *http.Client
}

// NewGoogleOAuthRefresher creates a GoogleOAuthRefresher with default Google CloudCode credentials.
func NewGoogleOAuthRefresher(httpClient *http.Client) *GoogleOAuthRefresher {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &GoogleOAuthRefresher{
		clientID:     DefaultGoogleClientID,
		clientSecret: DefaultGoogleClientSecret,
		tokenURL:     DefaultGoogleTokenURL,
		httpClient:   httpClient,
	}
}

// NewGoogleOAuthRefresherWithConfig creates a GoogleOAuthRefresher with custom configuration (useful for testing).
func NewGoogleOAuthRefresherWithConfig(clientID, clientSecret, tokenURL string, httpClient *http.Client) *GoogleOAuthRefresher {
	if clientID == "" {
		clientID = DefaultGoogleClientID
	}
	if clientSecret == "" {
		clientSecret = DefaultGoogleClientSecret
	}
	if tokenURL == "" {
		tokenURL = DefaultGoogleTokenURL
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &GoogleOAuthRefresher{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		httpClient:   httpClient,
	}
}

// RefreshToken exchanges an existing refresh token for a fresh access token via Google OAuth2 token endpoint.
func (r *GoogleOAuthRefresher) RefreshToken(ctx context.Context, refreshToken string) (*TokenRefreshResult, error) {
	cleanRefreshToken := strings.TrimSpace(refreshToken)
	if cleanRefreshToken == "" {
		return nil, ErrEmptyRefreshToken
	}

	form := url.Values{}
	form.Set("client_id", r.clientID)
	form.Set("client_secret", r.clientSecret)
	form.Set("refresh_token", cleanRefreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrTokenRefreshFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: http request failed: %v", ErrTokenRefreshFailed, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", ErrTokenRefreshFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: server returned status %d: %s", ErrTokenRefreshFailed, resp.StatusCode, string(bodyBytes))
	}

	var res TokenRefreshResult
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("%w: failed to parse json response: %v", ErrTokenRefreshFailed, err)
	}

	if res.AccessToken == "" {
		return nil, fmt.Errorf("%w: response missing access_token: %s", ErrTokenRefreshFailed, string(bodyBytes))
	}

	if res.ExpiresIn <= 0 {
		res.ExpiresIn = 3600
	}
	res.Expiry = time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)

	return &res, nil
}