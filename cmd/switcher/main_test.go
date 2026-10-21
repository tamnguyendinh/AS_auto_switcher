package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"antigravity-switch/internal/automation"
	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/reloader"
	"antigravity-switch/internal/store"
)

func makeTestJWT(email string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims := map[string]interface{}{
		"email": email,
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}
	claimsBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signature := base64.RawURLEncoding.EncodeToString([]byte("signature"))
	return header + "." + payload + "." + signature
}

// -----------------------------------------------------------------------------
// Switch Gate Coordination Tests
// -----------------------------------------------------------------------------

func TestSwitch_WithGate(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "accounts_store")
	accStore, err := store.NewAccountStore(storePath)
	if err != nil {
		t.Fatalf("failed initializing test store: %v", err)
	}

	targetEmail := "target@test.com"
	activeToken := &model.TokenData{
		AccessToken: "test_access",
		IDToken:     makeTestJWT("initial@test.com"),
	}
	targetToken := &model.TokenData{
		AccessToken: "target_access",
		IDToken:     makeTestJWT(targetEmail),
	}
	if err := accStore.SaveAccount(targetEmail, targetToken); err != nil {
		t.Fatalf("failed saving target account: %v", err)
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		email := manager.ExtractEmailFromToken(activeToken)
		if email == "" {
			email = targetEmail
		}
		_, _ = w.Write([]byte(`{"email":"` + email + `"}`))
	}))
	defer testServer.Close()

	u, _ := url.Parse(testServer.URL)
	port, _ := strconv.Atoi(u.Port())
	testClient := connector.NewClientWithHTTP(&connector.ServerInfo{
		PID:       12345,
		Port:      port,
		CSRFToken: "test-csrf",
	}, testServer.Client())

	rel := reloader.NewSessionReloader()
	rel.ProcessKiller = func(ctx context.Context) error { return nil }
	rel.Sleeper = func(ctx context.Context, d time.Duration) error { return nil }
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       12345,
			Port:      port,
			CSRFToken: "test-csrf",
		}, nil
	}

	mgr, err := manager.NewAccountManager(accStore, rel)
	if err != nil {
		t.Fatalf("failed initializing manager: %v", err)
	}
	mgr.SetWincredHandlers(
		func() (*model.TokenData, error) { return activeToken, nil },
		func(token *model.TokenData) error { activeToken = token; return nil },
	)
	mgr.SetClientFinder(func(ctx context.Context) (*connector.Client, error) {
		return testClient, nil
	})

	gate := automation.NewSwitchGate()
	res, err := automation.ExecuteSwitchWithGate(context.Background(), gate, mgr, targetEmail, false)
	if err != nil {
		t.Fatalf("ExecuteSwitchWithGate failed: %v", err)
	}
	if res == nil || !res.Success {
		t.Fatalf("expected successful switch, got: %+v", res)
	}
	if res.TargetEmail != targetEmail {
		t.Errorf("expected target email %s, got %s", targetEmail, res.TargetEmail)
	}
}

func TestSwitch_GateContention(t *testing.T) {
	gate := automation.NewSwitchGate()
	held := make(chan struct{})
	release := make(chan struct{})

	go func() {
		_ = gate.Execute(context.Background(), func(wasAbandoned bool) error {
			close(held)
			<-release
			return nil
		})
	}()

	<-held
	executed, err := gate.TryExecute(func(wasAbandoned bool) error {
		return nil
	})
	if executed || err != automation.ErrGateBusy {
		t.Errorf("expected TryExecute to fail with ErrGateBusy while gate is held, got executed=%v, err=%v", executed, err)
	}
	close(release)
}

func TestSwitch_TwoWayContention(t *testing.T) {
	gate := automation.NewSwitchGate()

	// 1. Manual first: manual switch holds gate, concurrent TryExecute is rejected
	held := make(chan struct{})
	release := make(chan struct{})

	go func() {
		_ = gate.Execute(context.Background(), func(wasAbandoned bool) error {
			close(held)
			<-release
			return nil
		})
	}()

	<-held

	executed, err := gate.TryExecute(func(wasAbandoned bool) error {
		return nil
	})
	if executed || err != automation.ErrGateBusy {
		t.Errorf("expected TryExecute to fail with ErrGateBusy, got %v", err)
	}
	close(release)
	time.Sleep(20 * time.Millisecond)

	// 2. Auto first: auto holds gate, manual TryExecute is rejected
	autoHeld := make(chan struct{})
	autoRelease := make(chan struct{})

	go func() {
		_ = gate.Execute(context.Background(), func(wasAbandoned bool) error {
			close(autoHeld)
			<-autoRelease
			return nil
		})
	}()

	<-autoHeld
	executedManual, errManual := gate.TryExecute(func(wasAbandoned bool) error {
		return nil
	})
	if executedManual || errManual != automation.ErrGateBusy {
		t.Errorf("expected TryExecute to fail with ErrGateBusy when auto holds gate, got %v", errManual)
	}
	close(autoRelease)
}
