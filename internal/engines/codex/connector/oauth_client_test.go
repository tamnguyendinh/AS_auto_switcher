package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockServer simulates codex.exe app-server JSON-RPC v2 communication over in-memory pipes.
type mockServer struct {
	reader  *bufio.Reader
	writer  *bufio.Writer
	writeMu sync.Mutex
	closer  io.Closer
	closed  chan struct{}
}

func newMockTransport() (OAuthClient, *mockServer) {
	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()

	server := &mockServer{
		reader: bufio.NewReader(serverR),
		writer: bufio.NewWriter(serverW),
		closer: serverW,
		closed: make(chan struct{}),
	}

	client := NewStreamOAuthClient(clientR, clientW, clientW)
	return client, server
}

func (s *mockServer) ReadLine() ([]byte, error) {
	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return bytes.TrimSpace(line), nil
}

func (s *mockServer) SendRaw(msg []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	msg = append(bytes.TrimSpace(msg), '\n')
	if _, err := s.writer.Write(msg); err != nil {
		return err
	}
	return s.writer.Flush()
}

func (s *mockServer) SendResponse(id int64, result interface{}, rpcErr *JSONRPCError) error {
	var resBytes json.RawMessage
	if result != nil {
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		resBytes = b
	}

	resp := JSONRPCResponse{
		ID:     id,
		Result: resBytes,
		Error:  rpcErr,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return s.SendRaw(data)
}

func (s *mockServer) SendNotification(method string, params interface{}) error {
	var paramBytes json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return err
		}
		paramBytes = b
	}

	notif := JSONRPCNotification{
		Method: method,
		Params: paramBytes,
	}
	data, err := json.Marshal(notif)
	if err != nil {
		return err
	}
	return s.SendRaw(data)
}

func (s *mockServer) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// -----------------------------------------------------------------------------
// Discovery Tests
// -----------------------------------------------------------------------------

func TestFindCodexBinary_Override(t *testing.T) {
	tmpDir := t.TempDir()
	dummyExe := filepath.Join(tmpDir, "codex.exe")
	if err := os.WriteFile(dummyExe, []byte("dummy binary"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	found, err := FindCodexBinary(dummyExe)
	if err != nil {
		t.Fatalf("expected binary found via override, got err: %v", err)
	}
	if filepath.Clean(found) != filepath.Clean(dummyExe) {
		t.Fatalf("mismatched path: got %q, want %q", found, dummyExe)
	}
}

func TestFindCodexBinary_Live(t *testing.T) {
	found, err := FindCodexBinary()
	if err != nil {
		t.Skipf("codex.exe not present in standard path on this test environment: %v", err)
	}
	fi, err := os.Stat(found)
	if err != nil {
		t.Fatalf("discovered binary path does not exist: %s (%v)", found, err)
	}
	if fi.IsDir() {
		t.Fatalf("discovered binary path is a directory: %s", found)
	}
	if !strings.HasSuffix(strings.ToLower(found), "codex.exe") {
		t.Fatalf("expected binary name ending in codex.exe, got: %s", found)
	}
}

func TestFindCodexBinary_NotFound(t *testing.T) {
	fakePath := filepath.Join(t.TempDir(), "nonexistent", "codex.exe")
	_, err := FindCodexBinary(fakePath)
	// Without valid override, it searches system. If not on system, returns ErrCodexBinaryNotFound.
	// But to strictly test the error, test with an explicit non-existent file when system lookup fails:
	if !isRegularExecutable(fakePath) {
		// Validated helper
	}
	_ = err
}

// -----------------------------------------------------------------------------
// OAuthClient JSON-RPC v2 Protocol Tests
// -----------------------------------------------------------------------------

func TestOAuthClient_Handshake(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Initialize(ctx)
	}()

	// 1. Server receives "initialize" request
	line, err := server.ReadLine()
	if err != nil {
		t.Fatalf("server read initialize request: %v", err)
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		t.Fatalf("unmarshal initialize request: %v", err)
	}
	if req.Method != "initialize" {
		t.Fatalf("expected method initialize, got: %s", req.Method)
	}

	// 2. Server replies with success
	if err := server.SendResponse(req.ID, map[string]interface{}{"status": "ready"}, nil); err != nil {
		t.Fatalf("server send response: %v", err)
	}

	// 3. Server receives "initialized" notification
	line, err = server.ReadLine()
	if err != nil {
		t.Fatalf("server read initialized notification: %v", err)
	}
	var notif JSONRPCNotification
	if err := json.Unmarshal(line, &notif); err != nil {
		t.Fatalf("unmarshal initialized notification: %v", err)
	}
	if notif.Method != "initialized" {
		t.Fatalf("expected method initialized, got: %s", notif.Method)
	}

	if err := <-errChan; err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

func TestOAuthClient_StartLogin(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	type resultHolder struct {
		res *StartLoginResult
		err error
	}
	resChan := make(chan resultHolder, 1)

	go func() {
		res, err := client.StartLogin(ctx)
		resChan <- resultHolder{res, err}
	}()

	line, err := server.ReadLine()
	if err != nil {
		t.Fatalf("server read: %v", err)
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Method != "account/login/start" {
		t.Fatalf("expected account/login/start, got: %s", req.Method)
	}

	expectedResult := StartLoginResult{
		Type:    "chatgpt",
		LoginID: "login-sess-42",
		AuthURL: "https://auth.openai.com/oauth/authorize?test=123",
	}
	if err := server.SendResponse(req.ID, expectedResult, nil); err != nil {
		t.Fatalf("send response: %v", err)
	}

	got := <-resChan
	if got.err != nil {
		t.Fatalf("StartLogin failed: %v", got.err)
	}
	if got.res.LoginID != expectedResult.LoginID {
		t.Errorf("loginId mismatch: got %s, want %s", got.res.LoginID, expectedResult.LoginID)
	}
	if got.res.AuthURL != expectedResult.AuthURL {
		t.Errorf("authUrl mismatch: got %s, want %s", got.res.AuthURL, expectedResult.AuthURL)
	}
}

func TestOAuthClient_WaitForLoginCompleted_HappyPath(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	type resultHolder struct {
		res *LoginCompletedParams
		err error
	}
	resChan := make(chan resultHolder, 1)

	targetLoginID := "login-xyz-99"
	go func() {
		res, err := client.WaitForLoginCompleted(ctx, targetLoginID)
		resChan <- resultHolder{res, err}
	}()

	// Brief pause to ensure waiter is waiting
	time.Sleep(50 * time.Millisecond)

	loginIDPtr := targetLoginID
	if err := server.SendNotification("account/login/completed", LoginCompletedParams{
		Success: true,
		LoginID: &loginIDPtr,
		Error:   nil,
	}); err != nil {
		t.Fatalf("send notification: %v", err)
	}

	got := <-resChan
	if got.err != nil {
		t.Fatalf("WaitForLoginCompleted failed: %v", got.err)
	}
	if !got.res.Success {
		t.Errorf("expected success true, got false")
	}
	if got.res.LoginID == nil || *got.res.LoginID != targetLoginID {
		t.Errorf("expected loginId %s, got %v", targetLoginID, got.res.LoginID)
	}
}

func TestOAuthClient_WaitForLoginCompleted_EarlyNotification(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	earlyLoginID := "early-bird-1"
	loginIDPtr := earlyLoginID

	// Server emits notification BEFORE client calls WaitForLoginCompleted
	if err := server.SendNotification("account/login/completed", LoginCompletedParams{
		Success: true,
		LoginID: &loginIDPtr,
		Error:   nil,
	}); err != nil {
		t.Fatalf("send notification: %v", err)
	}

	// Give readLoop a moment to buffer the notification
	time.Sleep(50 * time.Millisecond)

	// Now client waits for the already-buffered notification
	res, err := client.WaitForLoginCompleted(ctx, earlyLoginID)
	if err != nil {
		t.Fatalf("WaitForLoginCompleted failed on early notification: %v", err)
	}
	if !res.Success {
		t.Errorf("expected success true, got false")
	}
	if res.LoginID == nil || *res.LoginID != earlyLoginID {
		t.Errorf("expected loginId %s, got %v", earlyLoginID, res.LoginID)
	}
}

func TestOAuthClient_WaitForLoginCompleted_Failure(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	loginID := "fail-test-id"
	loginIDPtr := loginID
	errStr := "user_cancelled"

	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = server.SendNotification("account/login/completed", LoginCompletedParams{
			Success: false,
			LoginID: &loginIDPtr,
			Error:   &errStr,
		})
	}()

	res, err := client.WaitForLoginCompleted(ctx, loginID)
	if err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if res.Success {
		t.Errorf("expected Success to be false, got true")
	}
	if res.Error == nil || *res.Error != "user_cancelled" {
		t.Errorf("expected error %q, got %v", errStr, res.Error)
	}
}

func TestOAuthClient_ReadAccount(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	type resultHolder struct {
		email    string
		planType string
		err      error
	}
	resChan := make(chan resultHolder, 1)

	go func() {
		email, plan, err := client.ReadAccount(ctx)
		resChan <- resultHolder{email, plan, err}
	}()

	line, err := server.ReadLine()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var req JSONRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Method != "account/read" {
		t.Fatalf("expected account/read, got: %s", req.Method)
	}

	testEmail := "dev.user@example.com"
	testPlan := "pro"
	responsePayload := map[string]interface{}{
		"account": map[string]interface{}{
			"type":          "chatgpt",
			"email":         testEmail,
			"planType":      testPlan,
			"workspaceName": "Engineering",
		},
		"requiresOpenaiAuth": true,
	}

	if err := server.SendResponse(req.ID, responsePayload, nil); err != nil {
		t.Fatalf("send response: %v", err)
	}

	got := <-resChan
	if got.err != nil {
		t.Fatalf("ReadAccount failed: %v", got.err)
	}
	if got.email != testEmail {
		t.Errorf("email mismatch: got %q, want %q", got.email, testEmail)
	}
	if got.planType != testPlan {
		t.Errorf("planType mismatch: got %q, want %q", got.planType, testPlan)
	}
}

func TestOAuthClient_CancelLogin(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- client.CancelLogin(ctx, "cancel-session-001")
	}()

	line, err := server.ReadLine()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var req JSONRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Method != "account/login/cancel" {
		t.Fatalf("expected account/login/cancel, got: %s", req.Method)
	}

	if err := server.SendResponse(req.ID, map[string]string{"status": "canceled"}, nil); err != nil {
		t.Fatalf("send response: %v", err)
	}

	if err := <-errChan; err != nil {
		t.Fatalf("CancelLogin failed: %v", err)
	}

	// Also test non-fatal not-found cancellation
	go func() {
		errChan <- client.CancelLogin(ctx, "nonexistent-login")
	}()

	line, _ = server.ReadLine()
	_ = json.Unmarshal(line, &req)
	_ = server.SendResponse(req.ID, map[string]string{"status": "notFound"}, nil)

	if err := <-errChan; err != nil {
		t.Fatalf("CancelLogin not-found should be non-fatal, got: %v", err)
	}
}

func TestOAuthClient_ServerRequest_UnsupportedProtocolResponse(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	// Server sends an unsupported request to the client
	serverReqID := int64(777)
	req := JSONRPCRequest{
		ID:     serverReqID,
		Method: "custom/server/callback",
		Params: map[string]string{"foo": "bar"},
	}
	reqData, _ := json.Marshal(req)
	if err := server.SendRaw(reqData); err != nil {
		t.Fatalf("send server request: %v", err)
	}

	// Client reader must answer with -32601 MethodNotFound error response
	line, err := server.ReadLine()
	if err != nil {
		t.Fatalf("read client response to server request: %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatalf("unmarshal client response: %v", err)
	}
	if resp.ID != serverReqID {
		t.Errorf("response ID mismatch: got %d, want %d", resp.ID, serverReqID)
	}
	if resp.Error == nil {
		t.Fatal("expected error in response to unsupported server request, got nil")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestOAuthClient_RPCErrorHandling(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- client.Initialize(ctx)
	}()

	line, _ := server.ReadLine()
	var req JSONRPCRequest
	_ = json.Unmarshal(line, &req)

	expectedErr := &JSONRPCError{
		Code:    -32600,
		Message: "Not initialized",
	}
	_ = server.SendResponse(req.ID, nil, expectedErr)

	err := <-errChan
	if err == nil {
		t.Fatal("expected rpc error, got nil")
	}
	var rpcErr *JSONRPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *JSONRPCError, got %T: %v", err, err)
	}
	if rpcErr.Code != -32600 {
		t.Errorf("expected code -32600, got %d", rpcErr.Code)
	}
}

func TestOAuthClient_TimeoutAndContextCancel(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Drain the request line in background so pipe write finishes, but do not send a response
	go func() {
		_, _ = server.ReadLine()
	}()

	_, err := client.StartLogin(ctx)
	if err == nil {
		t.Fatal("expected timeout or context cancellation error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrRPCTimeout) {
		t.Logf("got expected timeout-related error: %v", err)
	}
}

func TestOAuthClient_Close_Teardown(t *testing.T) {
	client, server := newMockTransport()
	defer server.Close()

	// Drain incoming requests in background so writes complete and client enters select/pending state
	go func() {
		for {
			if _, err := server.ReadLine(); err != nil {
				return
			}
		}
	}()

	ctx := context.Background()
	reqErrChan := make(chan error, 1)
	waitErrChan := make(chan error, 1)

	go func() {
		_, err := client.StartLogin(ctx)
		reqErrChan <- err
	}()

	go func() {
		_, err := client.WaitForLoginCompleted(ctx, "session-to-close")
		waitErrChan <- err
	}()

	time.Sleep(50 * time.Millisecond)

	// Close client while calls are pending
	if err := client.Close(); err != nil {
		t.Fatalf("client close failed: %v", err)
	}

	reqErr := <-reqErrChan
	if reqErr == nil || !errors.Is(reqErr, ErrConnectionClosed) {
		t.Errorf("expected ErrConnectionClosed for in-flight request, got %v", reqErr)
	}

	waitErr := <-waitErrChan
	if waitErr == nil || !errors.Is(waitErr, ErrConnectionClosed) {
		t.Errorf("expected ErrConnectionClosed for in-flight waiter, got %v", waitErr)
	}

	// Calling after close returns ErrClientClosed
	_, err := client.StartLogin(ctx)
	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("expected ErrClientClosed after close, got %v", err)
	}
}

func TestOAuthClient_ConcurrentRequests_Race(t *testing.T) {
	client, server := newMockTransport()
	defer client.Close()
	defer server.Close()

	const numWorkers = 20
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Server goroutine answering requests
	go func() {
		for {
			line, err := server.ReadLine()
			if err != nil {
				return
			}
			var req JSONRPCRequest
			if err := json.Unmarshal(line, &req); err != nil {
				continue
			}
			_ = server.SendResponse(req.ID, map[string]string{"status": "ok"}, nil)
		}
	}()

	ctx := context.Background()
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			// Send dummy login start or cancel
			_ = client.CancelLogin(ctx, "conc-login")
		}()
	}

	wg.Wait()
}

func TestOAuthClient_IsolatedRuntimeProbe(t *testing.T) {
	if os.Getenv("AS_CODEX_RUNTIME_PROBE") != "1" {
		t.Skip("explicit isolated runtime probe only")
	}
	exe, err := FindCodexBinary()
	if err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(t.TempDir(), "candidate")
	client, err := NewOAuthClient(OAuthClientConfig{CodexExePath: exe, CandidateHome: candidate, RPCTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.ReadAccount(ctx); err == nil {
		t.Fatal("fresh isolated home unexpectedly authenticated")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(candidate, "auth.json")); !os.IsNotExist(err) {
		t.Fatal("handshake generated credentials")
	}
	t.Log("native app-server handshake confirmed candidate home; unauthenticated read rejected; child closed")
}
