package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGetCascadeModelConfig_And_ResolveActiveModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == EndpointCascadeModelConfigData {
			resp := CascadeModelConfigData{
				DefaultOverrideModelConfig: &ClientModelConfig{
					ModelOrAlias: ModelOrAlias{Model: "MODEL_OVERRIDE_PRO"},
					Label:        "Override Pro Label",
					Disabled:     false,
				},
				ClientModelConfigs: []ClientModelConfig{
					{
						ModelOrAlias: ModelOrAlias{Model: "MODEL_PLACEHOLDER_M318", Alias: "gemini-2.5-pro"},
						Label:        "Gemini 2.5 Pro",
						Disabled:     false,
						IsSelected:   true,
					},
					{
						ModelOrAlias: ModelOrAlias{Model: "MODEL_CLAUDE_3_5_SONNET"},
						Label:        "Claude 3.5 Sonnet",
						Disabled:     false,
					},
					{
						ModelOrAlias: ModelOrAlias{Model: "MODEL_DISABLED_TEST"},
						Label:        "Disabled Model",
						Disabled:     true,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	client := NewClient(&ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "test_csrf_token",
	})

	ctx := context.Background()

	// 1. Test GetCascadeModelConfig
	cfg, err := client.GetCascadeModelConfig(ctx)
	if err != nil {
		t.Fatalf("GetCascadeModelConfig failed: %v", err)
	}
	if len(cfg.ClientModelConfigs) != 3 {
		t.Errorf("expected 3 model configs, got %d", len(cfg.ClientModelConfigs))
	}

	tmpDir := t.TempDir()

	// 2. Test ResolveActiveModel with override priority (no pbtxt -> discovers override model)
	modelID, label, disabled, err := client.ResolveActiveModel(ctx, filepath.Join(tmpDir, "non_existent.pbtxt"))
	if err != nil {
		t.Fatalf("ResolveActiveModel failed: %v", err)
	}
	if modelID != "MODEL_OVERRIDE_PRO" {
		t.Errorf("expected modelID MODEL_OVERRIDE_PRO, got %s", modelID)
	}
	if label != "Override Pro Label" {
		t.Errorf("expected label Override Pro Label, got %s", label)
	}
	if disabled {
		t.Errorf("expected disabled=false, got true")
	}

	// 3. Test ResolveActiveModel with M318 in pbtxt -> joins with clientModelConfigs to get friendly label
	pbtxt := filepath.Join(tmpDir, "antigravity_state.pbtxt")
	_ = os.WriteFile(pbtxt, []byte("some_flag: true\nlast_selected_agent_model: \"MODEL_PLACEHOLDER_M318\"\n"), 0600)
	modelID, label, disabled, err = client.ResolveActiveModel(ctx, pbtxt)
	if err != nil {
		t.Fatalf("ResolveActiveModel failed: %v", err)
	}
	if modelID != DefaultActiveAgentModel {
		t.Errorf("expected modelID %s, got %s", DefaultActiveAgentModel, modelID)
	}
	if label != "Gemini 2.5 Pro" {
		t.Errorf("expected label Gemini 2.5 Pro, got %s", label)
	}
	if disabled {
		t.Errorf("expected disabled=false, got true")
	}

	// 4. Test ResolveActiveModel with Claude in pbtxt
	_ = os.WriteFile(pbtxt, []byte("some_flag: true\nlast_selected_agent_model: \"MODEL_CLAUDE_3_5_SONNET\"\n"), 0600)
	modelID, label, disabled, err = client.ResolveActiveModel(ctx, pbtxt)
	if err != nil {
		t.Fatalf("ResolveActiveModel failed: %v", err)
	}
	if modelID != "MODEL_CLAUDE_3_5_SONNET" {
		t.Errorf("expected modelID MODEL_CLAUDE_3_5_SONNET, got %s", modelID)
	}
	if label != "Claude 3.5 Sonnet" {
		t.Errorf("expected label Claude 3.5 Sonnet, got %s", label)
	}
	if disabled {
		t.Errorf("expected disabled=false, got true")
	}

	// 5. Test ResolveActiveModel with Disabled Model
	_ = os.WriteFile(pbtxt, []byte("some_flag: true\nlast_selected_agent_model: \"MODEL_DISABLED_TEST\"\n"), 0600)
	modelID, label, disabled, err = client.ResolveActiveModel(ctx, pbtxt)
	if err != nil {
		t.Fatalf("ResolveActiveModel failed: %v", err)
	}
	if !disabled {
		t.Errorf("expected disabled=true, got false")
	}
	if label != "Disabled Model" {
		t.Errorf("expected label Disabled Model, got %s", label)
	}

	// Unknown IDs must not silently select Gemini.
	_ = os.WriteFile(pbtxt, []byte("last_selected_agent_model: \"MODEL_DYNAMIC_FUTURE_V4\""), 0600)
	_, _, _, err = client.ResolveActiveModel(ctx, pbtxt)
	if err == nil {
		t.Fatal("expected unresolved model error")
	}

}

func TestSendEmptyItemsRetry(t *testing.T) {
	var capturedBody []byte
	var capturedPath string
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedHeaders = r.Header
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	client := NewClient(&ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "test_csrf_token_retry",
	})

	ctx := context.Background()
	err := client.SendEmptyItemsRetry(ctx, "conv-uuid-1234", "gemini-2.5-pro", nil)
	if err != nil {
		t.Fatalf("SendEmptyItemsRetry failed: %v", err)
	}

	if capturedPath != EndpointSendUserCascadeMessage {
		t.Errorf("expected path %s, got %s", EndpointSendUserCascadeMessage, capturedPath)
	}

	if capturedHeaders.Get("X-CSRF-Token") != "test_csrf_token_retry" {
		t.Errorf("expected CSRF token header, got %s", capturedHeaders.Get("X-CSRF-Token"))
	}

	// Verify items: [] explicitly
	bodyStr := string(capturedBody)
	if !strings.Contains(bodyStr, `"items":[]`) {
		t.Errorf("expected JSON to contain '\"items\":[]', got %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"cascadeId":"conv-uuid-1234"`) {
		t.Errorf("expected cascadeId in body, got %s", bodyStr)
	}

	// Verify error on empty cascadeID
	if err := client.SendEmptyItemsRetry(ctx, "", "gemini-2.5-pro", nil); err == nil {
		t.Errorf("expected error on empty cascadeID, got nil")
	}
}

func TestConnectFrame_EncodeDecode(t *testing.T) {
	payload := []byte(`{"conversationId":"conv-1","status":"RUNNING"}`)
	framed := EncodeConnectFrame(payload, false)

	if len(framed) != 5+len(payload) {
		t.Fatalf("expected framed length %d, got %d", 5+len(payload), len(framed))
	}
	if framed[0] != ConnectFlagMessage {
		t.Errorf("expected flag 0x00, got 0x%02x", framed[0])
	}

	buf := bytes.NewReader(framed)
	frame, err := ReadConnectFrame(buf)
	if err != nil {
		t.Fatalf("ReadConnectFrame failed: %v", err)
	}
	if frame.Flags != ConnectFlagMessage {
		t.Errorf("expected frame flag 0x00, got 0x%02x", frame.Flags)
	}
	if string(frame.Payload) != string(payload) {
		t.Errorf("payload mismatch: got %s", string(frame.Payload))
	}

	// Test End-of-Stream frame
	eosPayload := []byte(`{}`)
	eosFramed := EncodeConnectFrame(eosPayload, true)
	if eosFramed[0] != ConnectFlagEndOfStream {
		t.Errorf("expected flag 0x02, got 0x%02x", eosFramed[0])
	}

	eosFrame, err := ReadConnectFrame(bytes.NewReader(eosFramed))
	if err != nil {
		t.Fatalf("ReadConnectFrame EOS failed: %v", err)
	}
	if eosFrame.Flags&ConnectFlagEndOfStream == 0 {
		t.Errorf("expected EOS flag set")
	}
}

func TestStreamAgentStateUpdates_ConnectFramedAndLineDelimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == EndpointStreamAgentStateUpdates {
			if r.Header.Get("x-codeium-csrf-token") != "stream_csrf_test" {
				t.Error("missing native CSRF header")
			}
			requestFrame, err := ReadConnectFrame(r.Body)
			if err != nil {
				t.Errorf("missing request envelope: %v", err)
				http.Error(w, "bad frame", 400)
				return
			}
			var req map[string]any
			if json.Unmarshal(requestFrame.Payload, &req) != nil || req["initialStepsPageBounds"] == nil {
				t.Error("missing native page bounds")
			}

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming not supported", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/connect+json")
			w.WriteHeader(http.StatusOK)

			// Frame 1: RUNNING update via Connect envelope
			u1 := AgentStateUpdate{
				ConversationID:  "conv-abc",
				Status:          "RUNNING",
				ExecutionStatus: "RUNNING",
				ExecutionID:     "exec-1",
				StepID:          "step-1",
			}
			b1 := nativeFixture(t, u1)
			_, _ = w.Write(EncodeConnectFrame(b1, false))
			flusher.Flush()

			// Frame 2: Error 429 update via Connect envelope
			u2 := AgentStateUpdate{
				ConversationID:    "conv-abc",
				Status:            "IDLE",
				TerminationReason: "ERROR",
				LastStepError: &StepError{
					Code:    "RESOURCE_EXHAUSTED",
					Message: "429 Rate limit exceeded",
				},
				ExecutionID: "exec-1",
				StepID:      "step-2",
			}
			b2 := nativeFixture(t, u2)
			_, _ = w.Write(EncodeConnectFrame(b2, false))
			flusher.Flush()

			// Frame 3: EOS frame
			_, _ = w.Write(EncodeConnectFrame([]byte("{}"), true))
			flusher.Flush()
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(u.Port())
	client := NewClient(&ServerInfo{
		PID:       1234,
		Port:      port,
		CSRFToken: "stream_csrf_test",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updates, errs, err := client.StreamAgentStateUpdates(ctx, StreamAgentStateUpdatesRequest{
		ConversationID:     "conv-abc",
		SubscriberID:       "sub-1",
		DisableRehydration: true,
	})
	if err != nil {
		t.Fatalf("StreamAgentStateUpdates failed: %v", err)
	}

	var received []AgentStateUpdate
	for upd := range updates {
		received = append(received, upd)
	}

	if len(received) != 2 {
		t.Fatalf("expected 2 received updates, got %d", len(received))
	}

	// Check update 1: RUNNING
	if !received[0].IsRunning() {
		t.Errorf("expected update 0 to be running")
	}
	if received[0].IsRateLimitOrQuotaExhausted() {
		t.Errorf("update 0 should not be rate limit")
	}

	// Check update 2: 429
	if !received[1].IsRateLimitOrQuotaExhausted() {
		t.Errorf("expected update 1 to indicate rate limit/exhausted")
	}
	if !received[1].IsTerminalError() {
		t.Errorf("expected update 1 to be terminal error")
	}

	// Check error channel
	select {
	case streamErr, ok := <-errs:
		if ok && streamErr != nil {
			t.Errorf("unexpected stream error: %v", streamErr)
		}
	default:
	}
}

func TestAgentStateUpdate_HelpersAndErrorParsing(t *testing.T) {
	// 1. String error representation
	jsonStr := `{"conversationId":"c1","status":"IDLE","terminationReason":"ERROR","lastStepError":"RESOURCE_EXHAUSTED: quota exceeded"}`
	var u1 AgentStateUpdate
	if err := json.Unmarshal([]byte(jsonStr), &u1); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !u1.IsRateLimitOrQuotaExhausted() {
		t.Errorf("expected rate limit exhausted for string error")
	}
	if !u1.IsTerminalError() {
		t.Errorf("expected terminal error")
	}

	// 2. Object error representation
	jsonObj := `{"conversationId":"c1","status":"IDLE","terminationReason":"TERMINATION_REASON_ERROR","lastStepError":{"code":"429","message":"Too Many Requests"}}`
	var u2 AgentStateUpdate
	if err := json.Unmarshal([]byte(jsonObj), &u2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !u2.IsRateLimitOrQuotaExhausted() {
		t.Errorf("expected rate limit exhausted for object error")
	}

	// 3. RetryAt presence
	jsonRetry := `{"conversationId":"c1","status":"PAUSED","retryAt":"2026-10-20T12:00:00Z"}`
	var u3 AgentStateUpdate
	_ = json.Unmarshal([]byte(jsonRetry), &u3)
	u3.RetryAt = time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano)
	if !u3.HasRetryAt() {
		t.Errorf("expected HasRetryAt=true")
	}

	// 4. Successful completion
	jsonSuccess := `{"conversationId":"c1","status":"IDLE","terminationReason":"DONE"}`
	var u4 AgentStateUpdate
	_ = json.Unmarshal([]byte(jsonSuccess), &u4)
	if !u4.IsSuccessfulCompletion() {
		t.Errorf("expected IsSuccessfulCompletion=true")
	}
	if u4.IsRateLimitOrQuotaExhausted() {
		t.Errorf("success should not be rate limit")
	}
}
func nativeFixture(t *testing.T, u AgentStateUpdate) []byte {
	t.Helper()
	m := map[string]any{"conversationId": u.ConversationID, "status": "CASCADE_RUN_STATUS_" + u.Status,
		"executorMetadata":     map[string]any{"executionId": u.ExecutionID, "terminationReason": "EXECUTOR_TERMINATION_REASON_" + u.TerminationReason},
		"mainTrajectoryUpdate": map[string]any{"lastStepError": u.LastStepError, "stepsUpdate": map[string]any{"totalLength": 2}}}
	if u.RetryAt != "" {
		m["executionStatus"] = map[string]any{"modelRequest": map[string]any{"attempt": u.Attempt, "maxAttempts": u.MaxAttempts, "retryAt": u.RetryAt}}
	}
	b, err := json.Marshal(map[string]any{"update": m})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNativeStreamErrorsAndCancellation(t *testing.T) {
	out := make(chan AgentStateUpdate, 2)
	err := DecodeStreamAgentStateUpdates(bytes.NewReader(EncodeConnectFrame([]byte(`{"error":{"code":"unauthenticated","message":"missing CSRF token"}}`), true)), out)
	if err == nil || !strings.Contains(err.Error(), "unauthenticated") {
		t.Fatalf("lost Connect error: %v", err)
	}
	err = DecodeStreamAgentStateUpdates(bytes.NewReader(EncodeConnectFrame([]byte(`{"status":"IDLE"}`), false)), out)
	if err == nil {
		t.Fatal("accepted non-native envelope")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = decodeAgentStateStream(ctx, bytes.NewReader(EncodeConnectFrame(nativeFixture(t, AgentStateUpdate{ConversationID: "c", Status: "IDLE", TerminationReason: "NO_TOOL_CALL"}), false)), make(chan AgentStateUpdate))
	if err != context.Canceled {
		t.Fatalf("blocked decoder did not cancel: %v", err)
	}
}

func TestNativeStreamMergesExecutionPatches(t *testing.T) {
	frames := [][]byte{
		[]byte(`{"update":{"conversationId":"c","status":"CASCADE_RUN_STATUS_RUNNING","executorMetadata":{"executionId":"e1"},"executionStatus":{"modelRequest":{"attempt":1,"maxAttempts":3,"retryAt":"2099-01-01T00:00:00Z"}}}}`),
		[]byte(`{"update":{"status":"CASCADE_RUN_STATUS_IDLE","executorMetadata":{"executionId":"e1","terminationReason":"EXECUTOR_TERMINATION_REASON_ERROR"},"mainTrajectoryUpdate":{"lastStepError":{"errorCode":"RESOURCE_EXHAUSTED","errorMessage":"quota exhausted"},"stepsUpdate":{"totalLength":5}}}}`),
		[]byte(`{"update":{"status":"CASCADE_RUN_STATUS_IDLE","executorMetadata":{"executionId":"e2","terminationReason":"EXECUTOR_TERMINATION_REASON_NO_TOOL_CALL"},"mainTrajectoryUpdate":{"lastStepError":{}}}}`),
	}
	var buf bytes.Buffer
	for _, b := range frames {
		buf.Write(EncodeConnectFrame(b, false))
	}
	out := make(chan AgentStateUpdate, 3)
	if err := DecodeStreamAgentStateUpdates(&buf, out); err != nil {
		t.Fatal(err)
	}
	a, b, c := <-out, <-out, <-out
	if !a.IsRunning() || !a.HasRetryAt() || a.MaxAttempts != 3 {
		t.Fatalf("native retry not decoded: %+v", a)
	}
	if b.ConversationID != "c" || !b.IsTerminalError() || !b.IsRateLimitOrQuotaExhausted() || b.HasRetryAt() {
		t.Fatalf("terminal state incorrect: %+v", b)
	}
	if !c.IsSuccessfulCompletion() || c.LastStepError != nil {
		t.Fatalf("old error leaked to new execution: %+v", c)
	}
}
