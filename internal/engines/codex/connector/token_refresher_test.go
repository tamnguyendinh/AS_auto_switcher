package connector

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// helper to create a dummy JWT with an expiration timestamp
func makeTestJWT(exp int64, email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(map[string]any{
		"exp":   exp,
		"email": email,
		"iat":   exp - 3600,
	})
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return fmt.Sprintf("%s.%s.dummy-sig", header, payload)
}

func makeAuthJSON(idToken, refreshToken, lastRefresh string) json.RawMessage {
	data := map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]any{
			"id_token":      idToken,
			"refresh_token": refreshToken,
			"access_token":  "dummy-access",
			"account_id":    "test-account-id",
		},
	}
	if lastRefresh != "" {
		data["last_refresh"] = lastRefresh
	}
	raw, _ := json.Marshal(data)
	return raw
}

// 1. TestTokenRefresher_55MinuteRule_ExpRemaining
// Dummy JWT id_token with exp remaining <= 5 min -> ShouldPurge true
// Exp remaining > 5 min -> ShouldPurge false
// Exp already expired (< 0) -> ShouldPurge true
func TestTokenRefresher_55MinuteRule_ExpRemaining(t *testing.T) {
	refresher := NewTokenRefresher()
	fixedNow := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		remaining time.Duration
		expected  bool
	}{
		{
			name:      "Remaining 4 minutes (under 5 min threshold)",
			remaining: 4 * time.Minute,
			expected:  true,
		},
		{
			name:      "Remaining exactly 5 minutes (threshold boundary)",
			remaining: 5 * time.Minute,
			expected:  true,
		},
		{
			name:      "Remaining 10 minutes (fresh token)",
			remaining: 10 * time.Minute,
			expected:  false,
		},
		{
			name:      "Remaining 45 minutes (very fresh token)",
			remaining: 45 * time.Minute,
			expected:  false,
		},
		{
			name:      "Expired 2 minutes ago (negative remaining)",
			remaining: -2 * time.Minute,
			expected:  true,
		},
		{
			name:      "Expired 1 hour ago",
			remaining: -1 * time.Hour,
			expected:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exp := fixedNow.Add(tc.remaining).Unix()
			jwt := makeTestJWT(exp, "user@example.com")
			raw := makeAuthJSON(jwt, "rt.test.dummy", "")

			purge, err := refresher.ShouldPurge(raw, fixedNow)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if purge != tc.expected {
				t.Errorf("ShouldPurge(%v) = %v, want %v", tc.remaining, purge, tc.expected)
			}
		})
	}
}

// 2. TestTokenRefresher_55MinuteRule_LastRefresh
// No id_token, last_refresh 56m ago -> true
// last_refresh 20m ago -> false
func TestTokenRefresher_55MinuteRule_LastRefresh(t *testing.T) {
	refresher := NewTokenRefresher()
	fixedNow := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		elapsed  time.Duration
		expected bool
	}{
		{
			name:     "Last refresh 56 minutes ago (>= 55m threshold)",
			elapsed:  56 * time.Minute,
			expected: true,
		},
		{
			name:     "Last refresh exactly 55 minutes ago (threshold boundary)",
			elapsed:  55 * time.Minute,
			expected: true,
		},
		{
			name:     "Last refresh 20 minutes ago (< 55m threshold)",
			elapsed:  20 * time.Minute,
			expected: false,
		},
		{
			name:     "Last refresh 2 hours ago",
			elapsed:  2 * time.Hour,
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lastRefreshStr := fixedNow.Add(-tc.elapsed).Format(time.RFC3339)
			raw := makeAuthJSON("", "rt.test.dummy", lastRefreshStr)

			purge, err := refresher.ShouldPurge(raw, fixedNow)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if purge != tc.expected {
				t.Errorf("ShouldPurge(elapsed %v) = %v, want %v", tc.elapsed, purge, tc.expected)
			}
		})
	}

	// Fallback case: Has refresh_token, but both id_token and last_refresh are empty
	t.Run("Fallback empty id_token and empty last_refresh", func(t *testing.T) {
		raw := makeAuthJSON("", "rt.test.dummy", "")
		purge, err := refresher.ShouldPurge(raw, fixedNow)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !purge {
			t.Errorf("ShouldPurge(no id_token & no last_refresh) = false, want true")
		}
	})
}

// 3. TestTokenRefresher_ShouldPurge_Validation
// Empty authData, malformed JSON, or missing refresh_token -> returns error
func TestTokenRefresher_ShouldPurge_Validation(t *testing.T) {
	refresher := NewTokenRefresher()
	now := time.Now()

	t.Run("Empty raw message", func(t *testing.T) {
		_, err := refresher.ShouldPurge(nil, now)
		if err == nil {
			t.Error("expected error for nil raw, got nil")
		}
		_, err = refresher.ShouldPurge(json.RawMessage("   "), now)
		if err == nil {
			t.Error("expected error for whitespace raw, got nil")
		}
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		_, err := refresher.ShouldPurge(json.RawMessage("{malformed json"), now)
		if err == nil {
			t.Error("expected error for malformed json, got nil")
		}
	})

	t.Run("Missing refresh_token", func(t *testing.T) {
		raw := json.RawMessage(`{"tokens": {"refresh_token": ""}}`)
		_, err := refresher.ShouldPurge(raw, now)
		if err == nil {
			t.Fatal("expected error for empty refresh_token, got nil")
		}
		if !strings.Contains(err.Error(), "missing refresh_token") {
			t.Errorf("error %q should contain 'missing refresh_token'", err.Error())
		}
	})
}

// 4. TestTokenRefresher_RFC6749_FormUrlEncodedRequest
// Verify HTTP method POST, Content-Type application/x-www-form-urlencoded, and form fields
func TestTokenRefresher_RFC6749_FormUrlEncodedRequest(t *testing.T) {
	var (
		receivedMethod      string
		receivedContentType string
		receivedGrantType   string
		receivedClientID    string
		receivedToken       string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		receivedGrantType = r.PostForm.Get("grant_type")
		receivedClientID = r.PostForm.Get("client_id")
		receivedToken = r.PostForm.Get("refresh_token")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "new-access-123",
			"refresh_token": "new-refresh-456",
			"id_token": "new-id-789",
			"token_type": "Bearer",
			"expires_in": 3600
		}`))
	}))
	defer server.Close()

	refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)
	input := makeAuthJSON("old-id", "test-rt-string", "")

	result, err := refresher.RefreshToken(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST method, got %q", receivedMethod)
	}
	if receivedContentType != "application/x-www-form-urlencoded" {
		t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %q", receivedContentType)
	}
	if receivedGrantType != "refresh_token" {
		t.Errorf("expected grant_type refresh_token, got %q", receivedGrantType)
	}
	if receivedClientID != DefaultCodexClientID {
		t.Errorf("expected client_id %q, got %q", DefaultCodexClientID, receivedClientID)
	}
	if receivedToken != "test-rt-string" {
		t.Errorf("expected refresh_token test-rt-string, got %q", receivedToken)
	}

	var outputMap map[string]any
	if err := json.Unmarshal(result, &outputMap); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	tokens := outputMap["tokens"].(map[string]any)
	if tokens["access_token"] != "new-access-123" {
		t.Errorf("access_token = %v, want new-access-123", tokens["access_token"])
	}
	if tokens["refresh_token"] != "new-refresh-456" {
		t.Errorf("refresh_token = %v, want new-refresh-456", tokens["refresh_token"])
	}
	if tokens["id_token"] != "new-id-789" {
		t.Errorf("id_token = %v, want new-id-789", tokens["id_token"])
	}
}

// 5. TestTokenRefresher_RTR_PreserveRefreshTokenWhenEmpty
// Server returns 200 OK with empty refresh_token -> output must preserve old refresh_token
func TestTokenRefresher_RTR_PreserveRefreshTokenWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "new-access-token",
			"refresh_token": "",
			"id_token": "new-id-token"
		}`))
	}))
	defer server.Close()

	refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)
	input := makeAuthJSON("old-id", "rt.original.preserved.token", "")

	result, err := refresher.RefreshToken(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var outputMap map[string]any
	if err := json.Unmarshal(result, &outputMap); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	tokens := outputMap["tokens"].(map[string]any)
	if tokens["refresh_token"] != "rt.original.preserved.token" {
		t.Errorf("expected refresh_token to be preserved as 'rt.original.preserved.token', got %v", tokens["refresh_token"])
	}
	if tokens["access_token"] != "new-access-token" {
		t.Errorf("expected access_token to be 'new-access-token', got %v", tokens["access_token"])
	}
}

// 6. TestTokenRefresher_RTR_RotateRefreshTokenWhenProvided
// Server returns 200 OK with new rotated refresh_token -> output must update to new token
func TestTokenRefresher_RTR_RotateRefreshTokenWhenProvided(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "new-access-token",
			"refresh_token": "rt.2.new-rotated-token",
			"id_token": "new-id-token"
		}`))
	}))
	defer server.Close()

	refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)
	input := makeAuthJSON("old-id", "rt.1.old-token", "")

	result, err := refresher.RefreshToken(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var outputMap map[string]any
	if err := json.Unmarshal(result, &outputMap); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	tokens := outputMap["tokens"].(map[string]any)
	if tokens["refresh_token"] != "rt.2.new-rotated-token" {
		t.Errorf("expected rotated refresh_token 'rt.2.new-rotated-token', got %v", tokens["refresh_token"])
	}
}

// 7. TestTokenRefresher_PreserveUnknownMetadataFields
// Input has custom fields in root and tokens (e.g. oauth_provenance, login_owner_id, custom_field)
// All must be preserved 100% in output JSON
func TestTokenRefresher_PreserveUnknownMetadataFields(t *testing.T) {
	fixedNow := time.Date(2026, 10, 31, 15, 30, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "refreshed-access",
			"refresh_token": "refreshed-refresh",
			"id_token": "refreshed-id"
		}`))
	}))
	defer server.Close()

	refresher := NewTokenRefresherWithOptions(
		server.URL,
		server.Client(),
		DefaultCodexClientID,
		func() time.Time { return fixedNow },
	)

	inputJSON := `{
		"auth_mode": "chatgpt",
		"OPENAI_API_KEY": null,
		"custom_root_property": "safe_and_sound",
		"tokens": {
			"access_token": "old-access",
			"refresh_token": "old-refresh",
			"id_token": "old-id",
			"account_id": "account-uuid-9999",
			"oauth_provenance": {
				"issuer": "https://auth.openai.com",
				"client_id": "app_EMoamEEZ73f0CkXaXp7hrann"
			},
			"login_owner_id": "owner-abc-xyz-123",
			"custom_metadata_num": 42
		},
		"last_refresh": "2026-10-30T00:00:00Z"
	}`

	result, err := refresher.RefreshToken(context.Background(), json.RawMessage(inputJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// Verify root preservation
	if out["auth_mode"] != "chatgpt" {
		t.Errorf("auth_mode = %v, want chatgpt", out["auth_mode"])
	}
	if out["custom_root_property"] != "safe_and_sound" {
		t.Errorf("custom_root_property = %v, want safe_and_sound", out["custom_root_property"])
	}
	if out["last_refresh"] != fixedNow.Format(time.RFC3339) {
		t.Errorf("last_refresh = %v, want %v", out["last_refresh"], fixedNow.Format(time.RFC3339))
	}

	// Verify tokens preservation
	tokens, ok := out["tokens"].(map[string]any)
	if !ok {
		t.Fatalf("tokens is not a map")
	}
	if tokens["access_token"] != "refreshed-access" {
		t.Errorf("access_token = %v, want refreshed-access", tokens["access_token"])
	}
	if tokens["refresh_token"] != "refreshed-refresh" {
		t.Errorf("refresh_token = %v, want refreshed-refresh", tokens["refresh_token"])
	}
	if tokens["id_token"] != "refreshed-id" {
		t.Errorf("id_token = %v, want refreshed-id", tokens["id_token"])
	}
	if tokens["account_id"] != "account-uuid-9999" {
		t.Errorf("account_id = %v, want account-uuid-9999", tokens["account_id"])
	}
	if tokens["login_owner_id"] != "owner-abc-xyz-123" {
		t.Errorf("login_owner_id = %v, want owner-abc-xyz-123", tokens["login_owner_id"])
	}
	if tokens["custom_metadata_num"] != float64(42) {
		t.Errorf("custom_metadata_num = %v, want 42", tokens["custom_metadata_num"])
	}

	prov, ok := tokens["oauth_provenance"].(map[string]any)
	if !ok {
		t.Fatalf("oauth_provenance is missing or not a map")
	}
	if prov["issuer"] != "https://auth.openai.com" || prov["client_id"] != "app_EMoamEEZ73f0CkXaXp7hrann" {
		t.Errorf("oauth_provenance corrupted: %v", prov)
	}
}

// 8. TestTokenRefresher_ErrorClassification_ErrAuthTerminal
// HTTP 400, 401, 403 or missing tokens/refresh_token -> errors.Is(err, ErrAuthTerminal) == true
// errors.Is(err, ErrNetworkLoss) == false
func TestTokenRefresher_ErrorClassification_ErrAuthTerminal(t *testing.T) {
	statusCodes := []struct {
		code int
		body string
	}{
		{http.StatusBadRequest, `{"error":"invalid_grant","error_description":"refresh token is revoked"}`},
		{http.StatusUnauthorized, `{"error":"unauthorized"}`},
		{http.StatusForbidden, `{"error":"forbidden"}`},
	}

	for _, sc := range statusCodes {
		t.Run(fmt.Sprintf("HTTP_%d", sc.code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(sc.code)
				_, _ = w.Write([]byte(sc.body))
			}))
			defer server.Close()

			refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)
			input := makeAuthJSON("id", "rt.bad.token", "")

			_, err := refresher.RefreshToken(context.Background(), input)
			if err == nil {
				t.Fatalf("expected error for HTTP %d, got nil", sc.code)
			}
			if !errors.Is(err, ErrAuthTerminal) {
				t.Errorf("expected errors.Is(err, ErrAuthTerminal) == true, got: %v", err)
			}
			if errors.Is(err, ErrNetworkLoss) {
				t.Errorf("expected errors.Is(err, ErrNetworkLoss) == false, got true")
			}
		})
	}

	t.Run("Missing tokens object in input", func(t *testing.T) {
		refresher := NewTokenRefresher()
		_, err := refresher.RefreshToken(context.Background(), json.RawMessage(`{"auth_mode":"chatgpt"}`))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrAuthTerminal) {
			t.Errorf("expected ErrAuthTerminal for missing tokens, got %v", err)
		}
	})

	t.Run("Empty refresh_token in input", func(t *testing.T) {
		refresher := NewTokenRefresher()
		_, err := refresher.RefreshToken(context.Background(), json.RawMessage(`{"tokens":{"refresh_token":""}}`))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrAuthTerminal) {
			t.Errorf("expected ErrAuthTerminal for empty refresh_token, got %v", err)
		}
	})
}

// 9. TestTokenRefresher_ErrorClassification_ErrNetworkLoss
// HTTP 429, 500, 503, connection dropped, context canceled -> errors.Is(err, ErrNetworkLoss) == true
// errors.Is(err, ErrAuthTerminal) == false
func TestTokenRefresher_ErrorClassification_ErrNetworkLoss(t *testing.T) {
	statusCodes := []struct {
		code int
		body string
	}{
		{http.StatusTooManyRequests, `{"error":"rate_limited"}`},
		{http.StatusInternalServerError, `{"error":"internal_server_error"}`},
		{http.StatusBadGateway, `{"error":"bad_gateway"}`},
		{http.StatusServiceUnavailable, `{"error":"service_unavailable"}`},
	}

	for _, sc := range statusCodes {
		t.Run(fmt.Sprintf("HTTP_%d", sc.code), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(sc.code)
				_, _ = w.Write([]byte(sc.body))
			}))
			defer server.Close()

			refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)
			input := makeAuthJSON("id", "rt.retryable", "")

			_, err := refresher.RefreshToken(context.Background(), input)
			if err == nil {
				t.Fatalf("expected error for HTTP %d, got nil", sc.code)
			}
			if !errors.Is(err, ErrNetworkLoss) {
				t.Errorf("expected errors.Is(err, ErrNetworkLoss) == true, got: %v", err)
			}
			if errors.Is(err, ErrAuthTerminal) {
				t.Errorf("expected errors.Is(err, ErrAuthTerminal) == false, got true")
			}
		})
	}

	t.Run("Connection Refused / Closed Server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		serverURL := server.URL
		server.Close() // explicitly close to simulate connection drop

		refresher := NewTokenRefresherWithOptions(serverURL, http.DefaultClient, DefaultCodexClientID, nil)
		input := makeAuthJSON("id", "rt.test", "")

		_, err := refresher.RefreshToken(context.Background(), input)
		if err == nil {
			t.Fatal("expected error on closed connection, got nil")
		}
		if !errors.Is(err, ErrNetworkLoss) {
			t.Errorf("expected ErrNetworkLoss on closed connection, got: %v", err)
		}
		if errors.Is(err, ErrAuthTerminal) {
			t.Errorf("expected ErrAuthTerminal == false, got true")
		}
	})

	t.Run("Context Timeout or Cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
		}))
		defer server.Close()

		refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)
		input := makeAuthJSON("id", "rt.test", "")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		_, err := refresher.RefreshToken(ctx, input)
		if err == nil {
			t.Fatal("expected error on cancelled context, got nil")
		}
		if !errors.Is(err, ErrNetworkLoss) {
			t.Errorf("expected ErrNetworkLoss on cancelled context, got: %v", err)
		}
	})
}

// 10. TestTokenRefresher_EmptyAccessTokenRejection
// Mock server returns 200 OK with empty access_token -> must reject with error
func TestTokenRefresher_EmptyAccessTokenRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "",
			"refresh_token": "some-rt",
			"id_token": "some-id"
		}`))
	}))
	defer server.Close()

	refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)
	input := makeAuthJSON("id", "rt.test", "")

	_, err := refresher.RefreshToken(context.Background(), input)
	if err == nil {
		t.Fatal("expected error for empty access_token, got nil")
	}
	if !strings.Contains(err.Error(), "empty access_token") {
		t.Errorf("error %q should contain 'empty access_token'", err.Error())
	}
}

// 11. TestTokenRefresher_DynamicClientIDExtraction
// Extracts client_id from tokens.oauth_provenance or falls back to DefaultCodexClientID
func TestTokenRefresher_DynamicClientIDExtraction(t *testing.T) {
	var capturedClientID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		capturedClientID = r.PostForm.Get("client_id")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"access_token": "new-at",
			"refresh_token": "new-rt"
		}`))
	}))
	defer server.Close()

	refresher := NewTokenRefresherWithOptions(server.URL, server.Client(), DefaultCodexClientID, nil)

	t.Run("Extract dynamically from oauth_provenance", func(t *testing.T) {
		capturedClientID = ""
		raw := json.RawMessage(`{
			"tokens": {
				"refresh_token": "rt.sample",
				"oauth_provenance": {
					"client_id": "app_custom_dynamic_client_id_999"
				}
			}
		}`)

		_, err := refresher.RefreshToken(context.Background(), raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedClientID != "app_custom_dynamic_client_id_999" {
			t.Errorf("captured client_id = %q, want app_custom_dynamic_client_id_999", capturedClientID)
		}
	})

	t.Run("Fallback to configured clientID when oauth_provenance is absent", func(t *testing.T) {
		capturedClientID = ""
		raw := json.RawMessage(`{
			"tokens": {
				"refresh_token": "rt.sample"
			}
		}`)

		_, err := refresher.RefreshToken(context.Background(), raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedClientID != DefaultCodexClientID {
			t.Errorf("captured client_id = %q, want default %q", capturedClientID, DefaultCodexClientID)
		}
	})
}
