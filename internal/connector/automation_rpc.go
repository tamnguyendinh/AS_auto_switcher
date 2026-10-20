package connector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ModelOrAlias represents a model identifier or its alias.
type ModelOrAlias struct {
	Model string `json:"model"`
	Alias string `json:"alias,omitempty"`
}

// ClientModelConfig represents one model entry in clientModelConfigs from LanguageServer.
type ClientModelConfig struct {
	ModelOrAlias ModelOrAlias `json:"modelOrAlias"`
	Label        string       `json:"label,omitempty"`
	Description  string       `json:"description,omitempty"`
	Disabled     bool         `json:"disabled,omitempty"`
	IsSelected   bool         `json:"isSelected,omitempty"`
	Selected     bool         `json:"selected,omitempty"`
}

// CascadeModelConfigData is the response payload from GetCascadeModelConfigData.
type CascadeModelConfigData struct {
	DefaultOverrideModelConfig *ClientModelConfig  `json:"defaultOverrideModelConfig,omitempty"`
	ClientModelConfigs         []ClientModelConfig `json:"clientModelConfigs,omitempty"`
}

// Connect-RPC constants and framing flags
const (
	EndpointStreamAgentStateUpdates = "/exa.language_server_pb.LanguageServerService/StreamAgentStateUpdates"

	ConnectFlagMessage     byte = 0x00
	ConnectFlagEndOfStream byte = 0x02
)

// StreamAgentStateUpdatesRequest encapsulates parameters for subscribing to agent state updates.
type StreamAgentStateUpdatesRequest struct {
	ConversationID     string `json:"conversationId"`
	SubscriberID       string `json:"subscriberId"`
	DisableRehydration bool   `json:"disableRehydration"`
	PageBounds         int    `json:"pageBounds,omitempty"`
}

// StepError represents structured error information in an AgentStateUpdate.
type StepError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
}

// UnmarshalJSON customizes unmarshaling to accept both string and object error representations.
func (e *StepError) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		e.Message = s
		e.Code = s
		return nil
	}
	type alias StepError
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = StepError(a)
	return nil
}

// AgentStateUpdate represents a single update frame from StreamAgentStateUpdates.
type AgentStateUpdate struct {
	ConversationID    string          `json:"conversationId,omitempty"`
	Status            string          `json:"status,omitempty"`
	TerminationReason string          `json:"terminationReason,omitempty"`
	LastStepError     *StepError      `json:"lastStepError,omitempty"`
	ExecutionStatus   string          `json:"executionStatus,omitempty"`
	ExecutionID       string          `json:"executionId,omitempty"`
	StepID            string          `json:"stepId,omitempty"`
	Attempt           int             `json:"attempt,omitempty"`
	RetryAt           string          `json:"retryAt,omitempty"`
	Raw               json.RawMessage `json:"raw,omitempty"`
}

// IsRateLimitOrQuotaExhausted checks if the update indicates rate limit or exhausted quota.
func (u *AgentStateUpdate) IsRateLimitOrQuotaExhausted() bool {
	if u == nil {
		return false
	}
	var textToCheck []string
	if u.LastStepError != nil {
		textToCheck = append(textToCheck, u.LastStepError.Code, u.LastStepError.Message, u.LastStepError.Type)
	}
	textToCheck = append(textToCheck, u.TerminationReason, u.ExecutionStatus)
	joined := strings.ToUpper(strings.Join(textToCheck, " "))

	return strings.Contains(joined, "429") ||
		strings.Contains(joined, "RESOURCE_EXHAUSTED") ||
		strings.Contains(joined, "RATE_LIMIT_EXCEEDED") ||
		strings.Contains(joined, "QUOTA_EXHAUSTED") ||
		strings.Contains(joined, "RATE LIMIT") ||
		strings.Contains(joined, "QUOTA EXHAUSTED")
}

// IsTerminalError returns true if the update represents a terminal error execution state.
func (u *AgentStateUpdate) IsTerminalError() bool {
	if u == nil {
		return false
	}
	status := strings.ToUpper(strings.TrimSpace(u.Status))
	term := strings.ToUpper(strings.TrimSpace(u.TerminationReason))
	return status == "IDLE" && (term == "ERROR" || term == "TERMINATION_REASON_ERROR" || u.LastStepError != nil)
}

// IsSuccessfulCompletion returns true if execution finished normally without errors.
func (u *AgentStateUpdate) IsSuccessfulCompletion() bool {
	if u == nil {
		return false
	}
	status := strings.ToUpper(strings.TrimSpace(u.Status))
	term := strings.ToUpper(strings.TrimSpace(u.TerminationReason))
	return status == "IDLE" && (term == "DONE" || term == "COMPLETED" || (term == "" && u.LastStepError == nil))
}

// HasRetryAt checks if the language server has scheduled an automatic internal retry.
func (u *AgentStateUpdate) HasRetryAt() bool {
	return u != nil && strings.TrimSpace(u.RetryAt) != ""
}

// IsRunning returns true if the turn is currently executing.
func (u *AgentStateUpdate) IsRunning() bool {
	if u == nil {
		return false
	}
	st := strings.ToUpper(strings.TrimSpace(u.Status))
	ex := strings.ToUpper(strings.TrimSpace(u.ExecutionStatus))
	return st == "RUNNING" || ex == "RUNNING"
}

// ConnectFrame represents a framed message in Connect streaming protocol.
type ConnectFrame struct {
	Flags   byte
	Payload []byte
}

// ReadConnectFrame reads a single 5-byte header framed message from r.
func ReadConnectFrame(r io.Reader) (*ConnectFrame, error) {
	var header [5]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}
	flags := header[0]
	length := binary.BigEndian.Uint32(header[1:5])
	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}
	return &ConnectFrame{Flags: flags, Payload: payload}, nil
}

// EncodeConnectFrame wraps a payload in Connect 5-byte framing.
func EncodeConnectFrame(payload []byte, isEndOfStream bool) []byte {
	flags := ConnectFlagMessage
	if isEndOfStream {
		flags = ConnectFlagEndOfStream
	}
	buf := make([]byte, 5+len(payload))
	buf[0] = flags
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(payload)))
	copy(buf[5:], payload)
	return buf
}

// DecodeStreamAgentStateUpdates decodes incoming agent state updates from an io.Reader.
// It supports both Connect streaming protocol 5-byte envelope frames and line-delimited JSON.
func DecodeStreamAgentStateUpdates(r io.Reader, out chan<- AgentStateUpdate) error {
	br := bufio.NewReader(r)

	for {
		firstByte, err := br.Peek(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		// Line-delimited JSON starts with '{' or whitespace
		if firstByte[0] == '{' || firstByte[0] == ' ' || firstByte[0] == '\t' || firstByte[0] == '\r' || firstByte[0] == '\n' {
			line, readErr := br.ReadBytes('\n')
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) > 0 {
				var update AgentStateUpdate
				if unmarshalErr := json.Unmarshal(trimmed, &update); unmarshalErr == nil {
					out <- update
				}
			}
			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					return nil
				}
				return readErr
			}
			continue
		}

		// Connect framing
		frame, frameErr := ReadConnectFrame(br)
		if frameErr != nil {
			if errors.Is(frameErr, io.EOF) {
				return nil
			}
			return frameErr
		}

		if frame.Flags&ConnectFlagEndOfStream != 0 {
			return nil
		}

		if len(frame.Payload) > 0 {
			var update AgentStateUpdate
			if unmarshalErr := json.Unmarshal(frame.Payload, &update); unmarshalErr == nil {
				out <- update
			}
		}
	}
}

// GetCascadeModelConfig queries GetCascadeModelConfigData Connect-RPC endpoint.
func (c *Client) GetCascadeModelConfig(ctx context.Context) (*CascadeModelConfigData, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	resp, err := c.postJSON(ctx, EndpointCascadeModelConfigData, []byte("{}"))
	if err != nil {
		return nil, fmt.Errorf("failed calling GetCascadeModelConfigData: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetCascadeModelConfigData returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading GetCascadeModelConfigData response body: %w", err)
	}

	var data CascadeModelConfigData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed parsing GetCascadeModelConfigData JSON: %w", err)
	}
	return &data, nil
}

// ResolveActiveModel calls DiscoverActiveAgentModel and joins with GetCascadeModelConfig
// to dynamically determine the active model ID, friendly label, and whether it is disabled.
func (c *Client) ResolveActiveModel(ctx context.Context, pbtxtPath string) (modelID string, label string, disabled bool, err error) {
	modelID = c.DiscoverActiveAgentModel(ctx, pbtxtPath)
	if strings.TrimSpace(modelID) == "" {
		modelID = DefaultActiveAgentModel
	}

	cfgData, cfgErr := c.GetCascadeModelConfig(ctx)
	if cfgErr != nil || cfgData == nil {
		return modelID, modelID, false, nil
	}

	// 1. Check clientModelConfigs for direct model/alias match
	for _, cfg := range cfgData.ClientModelConfigs {
		if strings.EqualFold(cfg.ModelOrAlias.Model, modelID) || (cfg.ModelOrAlias.Alias != "" && strings.EqualFold(cfg.ModelOrAlias.Alias, modelID)) {
			lbl := cfg.Label
			if strings.TrimSpace(lbl) == "" {
				lbl = modelID
			}
			return modelID, lbl, cfg.Disabled, nil
		}
	}

	// 2. Check defaultOverrideModelConfig if model matches
	if cfgData.DefaultOverrideModelConfig != nil {
		moa := cfgData.DefaultOverrideModelConfig.ModelOrAlias
		if strings.EqualFold(moa.Model, modelID) || (moa.Alias != "" && strings.EqualFold(moa.Alias, modelID)) {
			lbl := cfgData.DefaultOverrideModelConfig.Label
			if strings.TrimSpace(lbl) == "" {
				lbl = modelID
			}
			return modelID, lbl, cfgData.DefaultOverrideModelConfig.Disabled, nil
		}
	}

	return modelID, modelID, false, nil
}

// SendEmptyItemsRetry dispatches a Connect-RPC SendUserCascadeMessage with an explicit empty items array ([]).
// This triggers Antigravity Language Server's native "Try again" retry mechanism without process restart.
func (c *Client) SendEmptyItemsRetry(ctx context.Context, cascadeID string, modelName string, cascadeConfig interface{}) error {
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
		CascadeID:     cascadeID,
		Items:         []CascadeMessageItem{},
		CascadeConfig: resolvedConfig,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed marshaling empty items retry payload: %w", err)
	}

	resp, err := c.postJSON(ctx, EndpointSendUserCascadeMessage, bodyBytes)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SendEmptyItemsRetry returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// StreamAgentStateUpdates establishes a streaming subscription to StreamAgentStateUpdates.
// It uses the underlying HTTP transport with no unary timeout so long-running streams are bounded only by ctx.
func (c *Client) StreamAgentStateUpdates(ctx context.Context, req StreamAgentStateUpdatesRequest) (<-chan AgentStateUpdate, <-chan error, error) {
	if err := c.validate(); err != nil {
		return nil, nil, err
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed marshaling StreamAgentStateUpdatesRequest: %w", err)
	}

	url := fmt.Sprintf("%s%s", c.info.BaseURL(), EndpointStreamAgentStateUpdates)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/connect+json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")
	httpReq.Header.Set("X-CSRF-Token", c.info.CSRFToken)

	var transport http.RoundTripper
	if c.httpClient != nil {
		transport = c.httpClient.Transport
	}
	streamClient := &http.Client{
		Transport: transport,
		Timeout:   0,
	}

	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("StreamAgentStateUpdates request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("StreamAgentStateUpdates returned HTTP %d", resp.StatusCode)
	}

	updateChan := make(chan AgentStateUpdate, 16)
	errChan := make(chan error, 1)

	go func() {
		defer resp.Body.Close()
		defer close(updateChan)
		defer close(errChan)

		if err := DecodeStreamAgentStateUpdates(resp.Body, updateChan); err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			select {
			case errChan <- err:
			default:
			}
		}
	}()

	return updateChan, errChan, nil
}