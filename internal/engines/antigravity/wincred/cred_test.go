package wincred

import (
	"errors"
	"testing"
	"time"

	"antigravity-switch/internal/engines/antigravity/model"
)

const testTargetName = "LegacyGeneric:target=gemini:antigravity_test_p6a"
const testUserName = "antigravity_test"

func TestWincredRoundTripAndDelete(t *testing.T) {
	// Ensure cleanup before and after test
	_ = DeleteTokenTarget(testTargetName)
	t.Cleanup(func() {
		_ = DeleteTokenTarget(testTargetName)
	})

	// 1. Verify non-existent target returns ErrNotFound
	if HasTokenTarget(testTargetName) {
		t.Fatalf("expected HasTokenTarget to return false for non-existent target")
	}
	_, err := ReadTokenTarget(testTargetName)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-existent target, got: %v", err)
	}

	// 2. Write valid token
	now := time.Now().Truncate(time.Second)
	tokenToWrite := &model.TokenData{
		AccessToken:  "ya29.test_access_token_p6a_12345",
		RefreshToken: "1//test_refresh_token_p6a_67890",
		IDToken:      "header.payload_test_tiếng_việt_đặng_thị_mỹ_duyên.signature",
		TokenType:    "Bearer",
		Expiry:       now.Add(1 * time.Hour),
		AuthMethod:   "consumer",
	}

	if err := WriteTokenTarget(testTargetName, testUserName, tokenToWrite); err != nil {
		t.Fatalf("WriteTokenTarget failed: %v", err)
	}

	// 3. Verify HasTokenTarget returns true
	if !HasTokenTarget(testTargetName) {
		t.Fatalf("expected HasTokenTarget to return true after write")
	}

	// 4. Read back and verify fields
	readToken, err := ReadTokenTarget(testTargetName)
	if err != nil {
		t.Fatalf("ReadTokenTarget failed: %v", err)
	}
	if readToken == nil {
		t.Fatalf("expected non-nil token")
	}

	if readToken.AccessToken != tokenToWrite.AccessToken {
		t.Errorf("AccessToken mismatch: expected %q, got %q", tokenToWrite.AccessToken, readToken.AccessToken)
	}
	if readToken.RefreshToken != tokenToWrite.RefreshToken {
		t.Errorf("RefreshToken mismatch: expected %q, got %q", tokenToWrite.RefreshToken, readToken.RefreshToken)
	}
	if readToken.IDToken != tokenToWrite.IDToken {
		t.Errorf("IDToken mismatch: expected %q, got %q", tokenToWrite.IDToken, readToken.IDToken)
	}
	if readToken.TokenType != tokenToWrite.TokenType {
		t.Errorf("TokenType mismatch: expected %q, got %q", tokenToWrite.TokenType, readToken.TokenType)
	}

	// 5. Delete and verify removal
	if err := DeleteTokenTarget(testTargetName); err != nil {
		t.Fatalf("DeleteTokenTarget failed: %v", err)
	}

	if HasTokenTarget(testTargetName) {
		t.Fatalf("expected HasTokenTarget to return false after delete")
	}

	// 6. Delete again should be idempotent (return nil)
	if err := DeleteTokenTarget(testTargetName); err != nil {
		t.Fatalf("expected idempotent DeleteTokenTarget to succeed, got: %v", err)
	}
}

func TestWincredWriteValidation(t *testing.T) {
	// Nil token
	if err := WriteTokenTarget(testTargetName, testUserName, nil); !errors.Is(err, ErrInvalidPayload) {
		t.Errorf("expected ErrInvalidPayload for nil token, got: %v", err)
	}

	// Empty token
	emptyToken := &model.TokenData{}
	if err := WriteTokenTarget(testTargetName, testUserName, emptyToken); !errors.Is(err, ErrInvalidPayload) {
		t.Errorf("expected ErrInvalidPayload for empty token, got: %v", err)
	}

	// Empty target
	validToken := &model.TokenData{AccessToken: "token123"}
	if err := WriteTokenTarget("", testUserName, validToken); err == nil {
		t.Errorf("expected error for empty target name")
	}
}

func TestWincredLiveAntigravityRead(t *testing.T) {
	// Read live default target if Antigravity is active on this host
	if !HasToken() {
		t.Skip("Default Antigravity credential not present on host; skipping live read check")
	}

	token, err := ReadToken()
	if err != nil {
		t.Fatalf("failed to read live Antigravity credential: %v", err)
	}
	if !token.IsValid() {
		t.Errorf("live token is invalid")
	}
	if token.AccessToken == "" && token.RefreshToken == "" {
		t.Errorf("expected at least access_token or refresh_token in live credential")
	}
}
