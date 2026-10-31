package connector

import (
	"as-auto-switcher/internal/core/credentialio"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	// ErrConnectionClosed is returned when the underlying transport closes while an RPC operation is in-flight.
	ErrConnectionClosed = errors.New("runtime connection closed")

	// ErrClientClosed is returned when attempting an RPC operation on a closed OAuthClient.
	ErrClientClosed = errors.New("oauth client is closed")

	// ErrRPCTimeout is returned when an RPC call exceeds its timeout deadline.
	ErrRPCTimeout = errors.New("rpc request timed out")
)

// JSONRPCRequest represents a JSON-RPC v2 request frame.
type JSONRPCRequest struct {
	ID     int64       `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// JSONRPCError represents a protocol-level JSON-RPC error.
type JSONRPCError struct {
	Code    int64           `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("Codex RPC error (code %d)", e.Code)
}

// JSONRPCResponse represents a JSON-RPC response frame received from codex.exe.
type JSONRPCResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCNotification represents an incoming or outgoing server notification frame without ID.
type JSONRPCNotification struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// StartLoginResult is the decoded result of an account/login/start RPC call.
type StartLoginResult struct {
	Type    string `json:"type"` // "chatgpt"
	LoginID string `json:"loginId"`
	AuthURL string `json:"authUrl"`
}

// LoginCompletedParams represents the notification parameters emitted upon OAuth web completion.
type LoginCompletedParams struct {
	Success bool    `json:"success"`
	LoginID *string `json:"loginId"`
	Error   *string `json:"error"`
}

// OAuthClient manages communication with the codex.exe app-server via Stdio JSON-RPC v2.
type OAuthClient interface {
	Initialize(ctx context.Context) error
	StartLogin(ctx context.Context) (*StartLoginResult, error)
	WaitForLoginCompleted(ctx context.Context, loginID string) (*LoginCompletedParams, error)
	ReadAccount(ctx context.Context) (email string, planType string, err error)
	CancelLogin(ctx context.Context, loginID string) error
	Close() error
}

// OAuthClientConfig configures an OAuthClient instance.
type OAuthClientConfig struct {
	CodexExePath  string
	CandidateHome string
	RPCTimeout    time.Duration

	// Stream overrides for testing / mocks
	Reader io.Reader
	Writer io.Writer
	Closer io.Closer
}

type oauthClient struct {
	cmd           *exec.Cmd
	isChild       bool
	candidateHome string

	stdinWriter *bufio.Writer
	writeMu     sync.Mutex
	closer      io.Closer

	seq       int64
	pendingMu sync.Mutex
	pending   map[int64]chan *JSONRPCResponse

	notifyMu         sync.Mutex
	completedWaiters map[string]chan *LoginCompletedParams
	recentCompleted  map[string]*LoginCompletedParams

	closed    chan struct{}
	closeOnce sync.Once

	rpcTimeout time.Duration
}

// NewOAuthClient instantiates and initializes a new OAuthClient.
// If Reader and Writer are provided in cfg, it operates over those streams (for mocking/tests).
// Otherwise, it launches codex.exe with CandidateHome isolation.
func NewOAuthClient(cfg OAuthClientConfig) (OAuthClient, error) {
	timeout := cfg.RPCTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	c := &oauthClient{
		pending:          make(map[int64]chan *JSONRPCResponse),
		completedWaiters: make(map[string]chan *LoginCompletedParams),
		recentCompleted:  make(map[string]*LoginCompletedParams),
		closed:           make(chan struct{}),
		rpcTimeout:       timeout,
	}

	if cfg.Reader != nil && cfg.Writer != nil {
		// Mock / stream mode
		c.stdinWriter = bufio.NewWriter(cfg.Writer)
		c.closer = cfg.Closer
		go c.readLoop(cfg.Reader)
		return c, nil
	}

	if strings.TrimSpace(cfg.CodexExePath) == "" {
		return nil, errors.New("missing codex executable path")
	}
	if strings.TrimSpace(cfg.CandidateHome) == "" {
		return nil, errors.New("missing candidate home directory for oauth isolation")
	}

	candidateDir := filepath.Clean(cfg.CandidateHome)
	if err := credentialio.PrivateDir(candidateDir); err != nil {
		return nil, fmt.Errorf("create candidate home (%s): %w", candidateDir, err)
	}

	configTomlPath := filepath.Join(candidateDir, "config.toml")
	if err := credentialio.Write(configTomlPath, []byte("cli_auth_credentials_store = \"file\"\n")); err != nil {
		return nil, err
	}
	cmd := exec.Command(cfg.CodexExePath, "app-server", "--stdio", "-c", "cli_auth_credentials_store=\"file\"")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	c.cmd = cmd
	c.isChild = true
	c.candidateHome = candidateDir

	// Strict Candidate Isolation: Only pass CODEX_HOME and CODEX_CLI_AUTH_CREDENTIALS_STORE to this child
	env := os.Environ()
	filteredEnv := make([]string, 0, len(env)+2)
	for _, e := range env {
		upper := strings.ToUpper(e)
		if !strings.HasPrefix(upper, "CODEX_HOME=") &&
			!strings.HasPrefix(upper, "CODEX_CLI_AUTH_CREDENTIALS_STORE=") && !strings.HasPrefix(upper, "OPENAI_API_KEY=") && !strings.HasPrefix(upper, "CODEX_API_KEY=") {
			filteredEnv = append(filteredEnv, e)
		}
	}
	filteredEnv = append(filteredEnv, "CODEX_HOME="+candidateDir)
	filteredEnv = append(filteredEnv, "CODEX_CLI_AUTH_CREDENTIALS_STORE=file")
	cmd.Env = filteredEnv

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdinPipe.Close()
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		return nil, fmt.Errorf("open stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdinPipe.Close()
		_ = stdoutPipe.Close()
		_ = stderrPipe.Close()
		return nil, fmt.Errorf("spawn codex app-server (%s): %w", cfg.CodexExePath, err)
	}

	// Drain stderr asynchronously to prevent pipe deadlock
	go drainStderr(stderrPipe)

	c.stdinWriter = bufio.NewWriter(stdinPipe)
	c.closer = stdinPipe

	go c.readLoop(stdoutPipe)

	return c, nil
}

// NewStreamOAuthClient creates an OAuthClient using custom reader, writer, and closer (for testing).
func NewStreamOAuthClient(r io.Reader, w io.Writer, c io.Closer) OAuthClient {
	client, _ := NewOAuthClient(OAuthClientConfig{
		Reader: r,
		Writer: w,
		Closer: c,
	})
	return client
}

// drainStderr consumes stderr quietly without leaking secrets
func drainStderr(r io.Reader) { _, _ = io.Copy(io.Discard, r) }

func (c *oauthClient) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *oauthClient) readLoop(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) > 0 {
			c.handleIncomingLine(line)
		}
	}
	_ = c.Close()
}

type incomingEnvelope struct {
	ID     *int64          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *JSONRPCError   `json:"error"`
}

func (c *oauthClient) handleIncomingLine(data []byte) {
	var env incomingEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return
	}

	// 1. Response (ID present, Method empty)
	if env.ID != nil && env.Method == "" {
		c.pendingMu.Lock()
		ch, ok := c.pending[*env.ID]
		c.pendingMu.Unlock()
		if ok {
			select {
			case ch <- &JSONRPCResponse{ID: *env.ID, Result: env.Result, Error: env.Error}:
			default:
			}
		}
		return
	}

	// 2. Server Request (ID present, Method non-empty) -> Respond with MethodNotFound (-32601)
	if env.ID != nil && env.Method != "" {
		errResp := JSONRPCResponse{
			ID: *env.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("unsupported server request: %s", env.Method),
			},
		}
		respBytes, err := json.Marshal(errResp)
		if err == nil {
			respBytes = append(respBytes, '\n')
			ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
			err = c.writeFrame(ctx, respBytes)
			cancel()
			if err != nil {
				_ = c.Close()
			}

		}
		return
	}

	// 3. Notification (ID null/absent, Method non-empty)
	if env.ID == nil && env.Method != "" {
		if env.Method == "account/login/completed" {
			var completed LoginCompletedParams
			if err := json.Unmarshal(env.Params, &completed); err == nil {
				c.handleLoginCompletedNotification(&completed)
			}
		}
		return
	}
}

func (c *oauthClient) handleLoginCompletedNotification(p *LoginCompletedParams) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()

	loginID := ""
	if p.LoginID != nil {
		loginID = *p.LoginID
	}

	// Deliver to matching waiter
	delivered := false
	if ch, ok := c.completedWaiters[loginID]; ok {
		select {
		case ch <- p:
			delivered = true
		default:
		}
	}
	if !delivered {
		if ch, ok := c.completedWaiters[""]; ok {
			select {
			case ch <- p:
				delivered = true
			default:
			}
		}
	}

	// Buffer in recentCompleted for late subscribers
	c.recentCompleted[loginID] = p
	if len(c.recentCompleted) > 100 {
		for k := range c.recentCompleted {
			delete(c.recentCompleted, k)
			break
		}
	}
}

func (c *oauthClient) sendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	if c.isClosed() {
		return nil, ErrClientClosed
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.rpcTimeout)
	defer cancel()

	reqID := atomic.AddInt64(&c.seq, 1)
	req := JSONRPCRequest{
		ID:     reqID,
		Method: method,
		Params: params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')

	resChan := make(chan *JSONRPCResponse, 1)
	c.pendingMu.Lock()
	c.pending[reqID] = resChan
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, reqID)
		c.pendingMu.Unlock()
	}()

	if err := c.writeFrame(reqCtx, data); err != nil {
		return nil, err
	}

	select {
	case res := <-resChan:
		if res == nil {
			return nil, ErrConnectionClosed
		}
		return res, nil
	case <-c.closed:
		return nil, ErrConnectionClosed
	case <-reqCtx.Done():
		if errors.Is(reqCtx.Err(), context.DeadlineExceeded) {
			return nil, ErrRPCTimeout
		}
		return nil, reqCtx.Err()
	}
}

func (c *oauthClient) sendNotification(method string, params interface{}) error {
	if c.isClosed() {
		return ErrClientClosed
	}

	type notif struct {
		Method string      `json:"method"`
		Params interface{} `json:"params"`
	}

	data, err := json.Marshal(notif{
		Method: method,
		Params: params,
	})
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}
	data = append(data, '\n')

	ctx, cancel := context.WithTimeout(context.Background(), c.rpcTimeout)
	defer cancel()
	return c.writeFrame(ctx, data)
}

// Closing the pipe interrupts a blocked write; never wait for writeMu to close it.
func (c *oauthClient) writeFrame(ctx context.Context, data []byte) error {
	done := make(chan error, 1)
	go func() {
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		if c.isClosed() {
			done <- ErrClientClosed
			return
		}
		_, err := c.stdinWriter.Write(data)
		if err == nil {
			err = c.stdinWriter.Flush()
		}
		done <- err
	}()
	select {
	case err := <-done:
		return err
	case <-c.closed:
		return ErrConnectionClosed
	case <-ctx.Done():
		if c.closer != nil {
			_ = c.closer.Close()
		}
		return ctx.Err()
	}
}

// Initialize performs the mandatory handshake: sends "initialize" request, awaits response,
// and then immediately emits the "initialized" notification.
func (c *oauthClient) Initialize(ctx context.Context) error {
	resp, err := c.sendRequest(ctx, "initialize", map[string]interface{}{
		"clientInfo": map[string]string{
			"name":    "as_auto_switcher",
			"version": "1.0.0",
		},
	})
	if err != nil {
		return fmt.Errorf("initialize rpc: %w", err)
	}
	if resp.Error != nil {
		return resp.Error
	}

	if c.isChild {
		var initResult struct {
			CodexHome string `json:"codexHome"`
		}
		if json.Unmarshal(resp.Result, &initResult) != nil || !sameDirectory(initResult.CodexHome, c.candidateHome) {
			return errors.New("Codex runtime did not confirm isolated candidate home")
		}
	}
	if err := c.sendNotification("initialized", map[string]interface{}{}); err != nil {
		return fmt.Errorf("send initialized notification: %w", err)
	}
	return nil
}

// StartLogin initiates an OAuth session via "account/login/start" with type="chatgpt".
func (c *oauthClient) StartLogin(ctx context.Context) (*StartLoginResult, error) {
	resp, err := c.sendRequest(ctx, "account/login/start", map[string]string{
		"type": "chatgpt",
	})
	if err != nil {
		return nil, fmt.Errorf("start login rpc: %w", err)
	}
	if resp.Error != nil {
		return nil, resp.Error
	}

	var res StartLoginResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		return nil, fmt.Errorf("decode start login result: %w", err)
	}
	parsed, parseErr := url.Parse(res.AuthURL)
	if res.Type != "chatgpt" || parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || res.LoginID == "" || res.AuthURL == "" {
		return nil, errors.New("incomplete login start response: missing loginId or authUrl")
	}
	return &res, nil
}

// WaitForLoginCompleted waits for the "account/login/completed" notification matching loginID.
// It also checks buffered notifications if the notification arrived before the wait was registered.
func (c *oauthClient) WaitForLoginCompleted(ctx context.Context, loginID string) (*LoginCompletedParams, error) {
	c.notifyMu.Lock()
	if p, ok := c.recentCompleted[loginID]; ok {
		delete(c.recentCompleted, loginID)
		c.notifyMu.Unlock()
		return p, nil
	}

	ch := make(chan *LoginCompletedParams, 1)
	c.completedWaiters[loginID] = ch
	c.notifyMu.Unlock()

	defer func() {
		c.notifyMu.Lock()
		delete(c.completedWaiters, loginID)
		c.notifyMu.Unlock()
	}()

	select {
	case res := <-ch:
		if res == nil {
			return nil, ErrConnectionClosed
		}
		return res, nil
	case <-c.closed:
		return nil, ErrConnectionClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReadAccount fetches identity and plan metadata via "account/read" with refreshToken=false.
func (c *oauthClient) ReadAccount(ctx context.Context) (email string, planType string, err error) {
	resp, err := c.sendRequest(ctx, "account/read", map[string]bool{
		"refreshToken": false,
	})
	if err != nil {
		return "", "", fmt.Errorf("read account rpc: %w", err)
	}
	if resp.Error != nil {
		return "", "", resp.Error
	}

	type accountReadResult struct {
		Account *struct {
			Type          string  `json:"type"`
			Email         *string `json:"email"`
			PlanType      *string `json:"planType"`
			WorkspaceName *string `json:"workspaceName"`
		} `json:"account"`
		RequiresOpenaiAuth bool `json:"requiresOpenaiAuth"`
	}

	var res accountReadResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		return "", "", fmt.Errorf("decode account read result: %w", err)
	}

	if res.Account == nil || res.Account.Type != "chatgpt" {
		return "", "", errors.New("unsupported or unauthenticated account")
	}
	if res.Account != nil {
		if res.Account.Email != nil {
			email = strings.TrimSpace(*res.Account.Email)
		}
		if res.Account.PlanType != nil {
			planType = strings.TrimSpace(*res.Account.PlanType)
		}
	}
	return email, planType, nil
}

// CancelLogin requests session cancellation via "account/login/cancel".
func (c *oauthClient) CancelLogin(ctx context.Context, loginID string) error {
	resp, err := c.sendRequest(ctx, "account/login/cancel", map[string]string{
		"loginId": loginID,
	})
	if err != nil {
		return fmt.Errorf("cancel login rpc: %w", err)
	}
	if resp.Error != nil {
		return resp.Error
	}
	var result struct {
		Status string `json:"status"`
	}
	if json.Unmarshal(resp.Result, &result) != nil || (result.Status != "canceled" && result.Status != "notFound") {
		return errors.New("invalid login cancellation response")
	}
	return nil
}

// Close gracefully closes the client, notifies all in-flight callers, and terminates child processes.
func (c *oauthClient) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)

		// 1. Close writer/stdin
		if c.closer != nil {
			_ = c.closer.Close()
		}

		// 2. Unblock all pending RPC requests
		c.pendingMu.Lock()
		for id := range c.pending {
			delete(c.pending, id)
		}
		c.pendingMu.Unlock()

		// 3. Unblock all login completion waiters
		c.notifyMu.Lock()
		for id := range c.completedWaiters {
			delete(c.completedWaiters, id)
		}
		c.notifyMu.Unlock()

		// 4. If child process was spawned, wait up to 5s then terminate
		if c.isChild && c.cmd != nil && c.cmd.Process != nil {
			done := make(chan error, 1)
			go func() {
				done <- c.cmd.Wait()
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				_ = c.cmd.Process.Kill()
				<-done
			}
		}
	})
	return nil
}

// Windows may report a long path while os.TempDir returns its 8.3 spelling.
// Compare directory identity rather than accepting a textual approximation.
func sameDirectory(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	left, err := os.Stat(a)
	if err != nil || !left.IsDir() {
		return false
	}
	right, err := os.Stat(b)
	return err == nil && right.IsDir() && os.SameFile(left, right)
}
