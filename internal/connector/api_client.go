package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	ErrNilServerInfo   = errors.New("language server info is nil or not configured")
	ErrInvalidPort     = errors.New("language server port must be greater than zero")
	ErrEmptyCSRFToken  = errors.New("language server CSRF token is empty")
	ErrRequestFailed   = errors.New("request to language server failed")
	ErrNoEndpointMatch = errors.New("no candidate endpoint responded successfully")
)

var pbtxtModelRegex = regexp.MustCompile(`last_selected_agent_model:\s*"?([a-zA-Z0-9_-]+)"?`)

// Client manages loopback communication with the Antigravity Language Server.
type Client struct {
	info       *ServerInfo
	httpClient *http.Client
}

// NewClient creates a new Client configured for the target ServerInfo with standard timeouts and connection reuse.
func NewClient(info *ServerInfo) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression: false,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}
	return &Client{
		info:       info,
		httpClient: httpClient,
	}
}

// NewClientWithHTTP creates a Client wrapping a caller-supplied *http.Client (useful for unit tests with mock servers).
func NewClientWithHTTP(info *ServerInfo, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{
		info:       info,
		httpClient: httpClient,
	}
}

// ServerInfo returns the active ServerInfo used by this client.
func (c *Client) ServerInfo() *ServerInfo {
	return c.info
}

func (c *Client) validate() error {
	if c.info == nil {
		return ErrNilServerInfo
	}
	if c.info.Port <= 0 {
		return ErrInvalidPort
	}
	if strings.TrimSpace(c.info.CSRFToken) == "" {
		return ErrEmptyCSRFToken
	}
	return nil
}

// postJSON issues an HTTP POST request with CSRF headers to the given relative path.
func (c *Client) postJSON(ctx context.Context, endpoint string, bodyBytes []byte) (*http.Response, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s%s", c.info.BaseURL(), endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed creating request for %s: %w", endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Csrf-Token", c.info.CSRFToken)
	req.Header.Set("x-codeium-csrf-token", c.info.CSRFToken)

	return c.httpClient.Do(req)
}

// GetUserStatus synchronizes and retrieves the active user identity currently running in language_server.exe RAM.
func (c *Client) GetUserStatus(ctx context.Context) (*UserStatus, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	emptyPayload := []byte("{}")
	var lastErr error

	for _, endpoint := range DefaultUserStatusEndpoints {
		resp, err := c.postJSON(ctx, endpoint, emptyPayload)
		if err != nil {
			lastErr = err
			continue
		}

		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response body from %s: %w", endpoint, err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("endpoint %s returned HTTP %d", endpoint, resp.StatusCode)
			continue
		}

		var rawData map[string]interface{}
		if err := json.Unmarshal(body, &rawData); err != nil {
			lastErr = fmt.Errorf("invalid json response from %s: %w", endpoint, err)
			continue
		}

		// Navigate userStatus or user_status envelope
		userObj := rawData
		if sub, ok := rawData["userStatus"].(map[string]interface{}); ok {
			userObj = sub
		} else if sub, ok := rawData["user_status"].(map[string]interface{}); ok {
			userObj = sub
		}

		email, _ := userObj["email"].(string)
		if email == "" {
			email, _ = rawData["email"].(string)
		}

		name, _ := userObj["name"].(string)
		if name == "" {
			name, _ = userObj["display_name"].(string)
		}
		if name == "" {
			name, _ = rawData["name"].(string)
		}
		if name == "" {
			name, _ = rawData["display_name"].(string)
		}

		status, _ := userObj["planStatus"].(string)
		if status == "" {
			status, _ = userObj["status"].(string)
		}
		if status == "" {
			status, _ = rawData["status"].(string)
		}
		if status == "" {
			status = "ACTIVE"
		}

		return &UserStatus{
			Email:       email,
			Name:        name,
			DisplayName: name,
			Status:      status,
			Raw:         rawData,
		}, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoEndpointMatch, lastErr)
	}
	return nil, ErrNoEndpointMatch
}

// AuthLogout purges the active user session and access tokens from language_server.exe RAM.
func (c *Client) AuthLogout(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}

	emptyPayload := []byte("{}")
	var lastErr error

	for _, endpoint := range DefaultAuthLogoutEndpoints {
		resp, err := c.postJSON(ctx, endpoint, emptyPayload)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
			continue
		}

		lastErr = fmt.Errorf("endpoint %s returned HTTP %d", endpoint, resp.StatusCode)
	}

	if lastErr != nil {
		return fmt.Errorf("%w: %v", ErrNoEndpointMatch, lastErr)
	}
	return ErrNoEndpointMatch
}

// TriggerReauth instructs language_server.exe to reload credentials from Windows Credential Manager into RAM.
func (c *Client) TriggerReauth(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}

	emptyPayload := []byte("{}")
	var lastErr error

	for _, endpoint := range DefaultReauthEndpoints {
		resp, err := c.postJSON(ctx, endpoint, emptyPayload)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			return nil
		}
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
			continue
		}

		lastErr = fmt.Errorf("endpoint %s returned HTTP %d", endpoint, resp.StatusCode)
	}

	if lastErr != nil {
		return fmt.Errorf("%w: %v", ErrNoEndpointMatch, lastErr)
	}
	return ErrNoEndpointMatch
}

// LoadTrajectory loads a conversation trajectory into the language server.
func (c *Client) LoadTrajectory(ctx context.Context, cascadeID string) error {
	if err := c.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(cascadeID) == "" {
		return errors.New("cascadeID cannot be empty")
	}

	payload, err := json.Marshal(TrajectoryPayload{CascadeID: strings.TrimSpace(cascadeID)})
	if err != nil {
		return err
	}

	resp, err := c.postJSON(ctx, EndpointLoadTrajectory, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LoadTrajectory returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// SmartFocusConversation focuses a specific conversation in the IDE UI.
func (c *Client) SmartFocusConversation(ctx context.Context, cascadeID string) error {
	if err := c.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(cascadeID) == "" {
		return errors.New("cascadeID cannot be empty")
	}

	payload, err := json.Marshal(TrajectoryPayload{CascadeID: strings.TrimSpace(cascadeID)})
	if err != nil {
		return err
	}

	resp, err := c.postJSON(ctx, EndpointSmartFocusConversation, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SmartFocusConversation returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// DiscoverActiveAgentModel determines the active agent model enum for Cascade.
// Checks antigravity_state.pbtxt first, queries GetCascadeModelConfigData second,
// and falls back to DefaultActiveAgentModel ("MODEL_PLACEHOLDER_M318").
func (c *Client) DiscoverActiveAgentModel(ctx context.Context, pbtxtPath string) string {
	targetPbtxt := pbtxtPath
	if targetPbtxt == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil && homeDir != "" {
			targetPbtxt = filepath.Join(homeDir, ".gemini", "antigravity", "antigravity_state.pbtxt")
		}
	}

	// 1. Check pbtxt file
	if targetPbtxt != "" {
		if contentBytes, err := os.ReadFile(targetPbtxt); err == nil {
			match := pbtxtModelRegex.FindStringSubmatch(string(contentBytes))
			if len(match) > 1 {
				val := strings.TrimSpace(match[1])
				if val != "" {
					return val
				}
			}
		}
	}

	// 2. Query GetCascadeModelConfigData via Connect-RPC if client is valid
	if c.validate() == nil {
		resp, err := c.postJSON(ctx, EndpointCascadeModelConfigData, []byte("{}"))
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				var data map[string]interface{}
				if json.Unmarshal(body, &data) == nil && data != nil {
					// Priority A: defaultOverrideModelConfig
					if override, ok := data["defaultOverrideModelConfig"].(map[string]interface{}); ok {
						if moa, ok := override["modelOrAlias"].(map[string]interface{}); ok {
							if m, ok := moa["model"].(string); ok && strings.TrimSpace(m) != "" {
								return strings.TrimSpace(m)
							}
						}
					}

					// Priority B: explicitly selected in clientModelConfigs
					if configs, ok := data["clientModelConfigs"].([]interface{}); ok {
						for _, cfgRaw := range configs {
							if cfg, ok := cfgRaw.(map[string]interface{}); ok {
								isSelected, _ := cfg["isSelected"].(bool)
								if !isSelected {
									isSelected, _ = cfg["selected"].(bool)
								}
								if isSelected {
									if moa, ok := cfg["modelOrAlias"].(map[string]interface{}); ok {
										if m, ok := moa["model"].(string); ok && strings.TrimSpace(m) != "" {
											return strings.TrimSpace(m)
										}
									}
								}
							}
						}

						// Priority C: first non-disabled model in clientModelConfigs
						for _, cfgRaw := range configs {
							if cfg, ok := cfgRaw.(map[string]interface{}); ok {
								disabled, _ := cfg["disabled"].(bool)
								if !disabled {
									if moa, ok := cfg["modelOrAlias"].(map[string]interface{}); ok {
										if m, ok := moa["model"].(string); ok && strings.TrimSpace(m) != "" {
											return strings.TrimSpace(m)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 3. Fallback default
	return DefaultActiveAgentModel
}

// SendUserCascadeMessage dispatches a prompt into an active Cascade conversation.
func (c *Client) SendUserCascadeMessage(
	ctx context.Context,
	cascadeID string,
	promptText string,
	modelName string,
	cascadeConfig interface{},
) error {
	if err := c.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(cascadeID) == "" {
		return errors.New("cascadeID cannot be empty")
	}

	resolvedConfig := cascadeConfig
	if resolvedConfig == nil {
		model := strings.TrimSpace(modelName)
		if model == "" {
			model = c.DiscoverActiveAgentModel(ctx, "")
		}
		resolvedConfig = CascadeConfig{
			PlannerConfig: PlannerConfig{
				PlanModel: model,
				RequestedModel: map[string]string{
					"model": model,
				},
			},
		}
	}

	payload := SendCascadeMessagePayload{
		CascadeID: strings.TrimSpace(cascadeID),
		Items: []CascadeMessageItem{
			{Text: promptText},
		},
		CascadeConfig: resolvedConfig,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed marshaling send cascade payload: %w", err)
	}

	resp, err := c.postJSON(ctx, EndpointSendUserCascadeMessage, bodyBytes)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SendUserCascadeMessage returned HTTP %d", resp.StatusCode)
	}
	return nil
}
