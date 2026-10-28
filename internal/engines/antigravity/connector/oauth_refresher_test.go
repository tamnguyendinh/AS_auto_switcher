package connector

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGoogleOAuthRefresher_Success(t *testing.T) {
	expectedClientID := "test-client-id"
	expectedClientSecret := "test-client-secret"
	expectedRefreshToken := "test-refresh-token"
	expectedAccessToken := "new-refreshed-access-token"
	expectedExpiresIn := 3600
	expectedTokenType := "Bearer"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %v", err)
		}

		form, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("failed parsing form body: %v", err)
		}

		if form.Get("client_id") != expectedClientID {
			t.Errorf("expected client_id %s, got %s", expectedClientID, form.Get("client_id"))
		}
		if form.Get("client_secret") != expectedClientSecret {
			t.Errorf("expected client_secret %s, got %s", expectedClientSecret, form.Get("client_secret"))
		}
		if form.Get("refresh_token") != expectedRefreshToken {
			t.Errorf("expected refresh_token %s, got %s", expectedRefreshToken, form.Get("refresh_token"))
		}
		if form.Get("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type refresh_token, got %s", form.Get("grant_type"))
		}

		resp := map[string]interface{}{
			"access_token":  expectedAccessToken,
			"expires_in":    expectedExpiresIn,
			"token_type":    expectedTokenType,
			"scope":         "https://www.googleapis.com/auth/cloud-platform",
			"refresh_token": "rotated-refresh-token",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	refresher := NewGoogleOAuthRefresherWithConfig(expectedClientID, expectedClientSecret, server.URL, server.Client())

	ctx := context.Background()
	result, err := refresher.RefreshToken(ctx, expectedRefreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AccessToken != expectedAccessToken {
		t.Errorf("expected access token %s, got %s", expectedAccessToken, result.AccessToken)
	}
	if result.ExpiresIn != expectedExpiresIn {
		t.Errorf("expected expires_in %d, got %d", expectedExpiresIn, result.ExpiresIn)
	}
	if result.TokenType != expectedTokenType {
		t.Errorf("expected token_type %s, got %s", expectedTokenType, result.TokenType)
	}
	if result.RefreshToken != "rotated-refresh-token" {
		t.Errorf("expected refresh_token rotated-refresh-token, got %s", result.RefreshToken)
	}
	if time.Until(result.Expiry) < 50*time.Minute || time.Until(result.Expiry) > 65*time.Minute {
		t.Errorf("unexpected expiry duration: %v", time.Until(result.Expiry))
	}
}

func TestGoogleOAuthRefresher_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "invalid_grant", "error_description": "Bad Request"}`))
	}))
	defer server.Close()

	refresher := NewGoogleOAuthRefresherWithConfig("cid", "sec", server.URL, server.Client())
	ctx := context.Background()
	result, err := refresher.RefreshToken(ctx, "bad-refresh-token")

	if err == nil {
		t.Fatal("expected error for HTTP 400, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
	if !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("expected error message to contain status 400 and invalid_grant, got: %v", err)
	}
}

func TestGoogleOAuthRefresher_NetworkFailure(t *testing.T) {
	client := &http.Client{Timeout: 50 * time.Millisecond}
	// Use unreachable local port
	refresher := NewGoogleOAuthRefresherWithConfig("cid", "sec", "http://127.0.0.1:59999/unreachable", client)

	ctx := context.Background()
	_, err := refresher.RefreshToken(ctx, "test-refresh-token")
	if err == nil {
		t.Fatal("expected network failure error, got nil")
	}
}

func TestGoogleOAuthRefresher_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	refresher := NewGoogleOAuthRefresherWithConfig("cid", "sec", server.URL, server.Client())
	ctx := context.Background()
	_, err := refresher.RefreshToken(ctx, "test-refresh-token")
	if err == nil {
		t.Fatal("expected json unmarshal error, got nil")
	}
}

func TestGoogleOAuthRefresher_EmptyRefreshToken(t *testing.T) {
	refresher := NewGoogleOAuthRefresher(nil)
	ctx := context.Background()
	_, err := refresher.RefreshToken(ctx, "   ")
	if err != ErrEmptyRefreshToken {
		t.Fatalf("expected ErrEmptyRefreshToken, got: %v", err)
	}
}