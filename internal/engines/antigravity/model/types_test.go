package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTokenData_UpdateTokens(t *testing.T) {
	initialJSON := `{
		"access_token": "old-access-token",
		"refresh_token": "old-refresh-token",
		"id_token": "old-id-token",
		"token_type": "Bearer",
		"expiry": "2026-01-01T00:00:00Z",
		"token": {
			"access_token": "old-access-token",
			"refresh_token": "old-refresh-token",
			"token_type": "Bearer",
			"expiry": "2026-01-01T00:00:00Z"
		}
	}`

	var token TokenData
	if err := json.Unmarshal([]byte(initialJSON), &token); err != nil {
		t.Fatalf("Failed to unmarshal initial JSON: %v", err)
	}

	if len(token.RawPayload) == 0 {
		t.Fatal("Expected RawPayload to be populated on UnmarshalJSON")
	}

	newExpiry := time.Now().Add(1 * time.Hour)
	token.UpdateTokens("new-access-token", "new-refresh-token", "new-id-token", newExpiry)

	if len(token.RawPayload) != 0 {
		t.Errorf("Expected RawPayload to be nil/empty after UpdateTokens, got len=%d", len(token.RawPayload))
	}
	if token.AccessToken != "new-access-token" {
		t.Errorf("Expected AccessToken new-access-token, got %s", token.AccessToken)
	}
	if token.RefreshToken != "new-refresh-token" {
		t.Errorf("Expected RefreshToken new-refresh-token, got %s", token.RefreshToken)
	}
	if token.IDToken != "new-id-token" {
		t.Errorf("Expected IDToken new-id-token, got %s", token.IDToken)
	}
	if !token.Expiry.Equal(newExpiry) {
		t.Errorf("Expected Expiry %v, got %v", newExpiry, token.Expiry)
	}

	marshaled, err := json.Marshal(&token)
	if err != nil {
		t.Fatalf("Failed to marshal updated token: %v", err)
	}

	marshaledStr := string(marshaled)
	if strings.Contains(marshaledStr, "old-access-token") {
		t.Error("Marshaled JSON still contains old-access-token from old RawPayload")
	}
	if !strings.Contains(marshaledStr, "new-access-token") {
		t.Error("Marshaled JSON missing new-access-token")
	}
	if !strings.Contains(marshaledStr, "new-refresh-token") {
		t.Error("Marshaled JSON missing new-refresh-token")
	}

	// Test partial update (empty strings don't overwrite existing)
	token.UpdateTokens("another-access-token", "", "", time.Time{})
	if token.AccessToken != "another-access-token" {
		t.Errorf("Expected AccessToken another-access-token, got %s", token.AccessToken)
	}
	if token.RefreshToken != "new-refresh-token" {
		t.Errorf("Expected RefreshToken to remain new-refresh-token, got %s", token.RefreshToken)
	}
	if token.IDToken != "new-id-token" {
		t.Errorf("Expected IDToken to remain new-id-token, got %s", token.IDToken)
	}

	// Test nil safety
	var nilToken *TokenData
	nilToken.UpdateTokens("x", "y", "z", time.Now())
}