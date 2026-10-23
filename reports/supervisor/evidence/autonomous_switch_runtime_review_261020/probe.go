package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"antigravity-switch/internal/automation"
	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/model"
	"antigravity-switch/internal/store"
	"antigravity-switch/internal/telemetry"
)

// Read-only diagnosis: synthetic identity, empty isolated account store, isolated
// mutex, mock Retry endpoint. No production SwitchAccount call can find a target.
func main() {
	result := map[string]any{}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil { panic(err) }
	updates := make(chan connector.AgentStateUpdate, 2)
	err = connector.DecodeStreamAgentStateUpdates(bytes.NewReader(connector.EncodeConnectFrame(raw, false)), updates)
	update := <-updates
	result["native_response_decoded"] = map[string]any{"error": fmt.Sprint(err), "status": update.Status, "conversation_id_empty": update.ConversationID == "", "successful_completion": update.IsSuccessfulCompletion()}
	eos := connector.EncodeConnectFrame([]byte(`{"error":{"code":"unauthenticated","message":"missing CSRF token"}}`), true)
	err = connector.DecodeStreamAgentStateUpdates(bytes.NewReader(eos), make(chan connector.AgentStateUpdate, 2))
	result["connect_error_trailer_returned_error"] = err != nil

	var retryCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { retryCalls.Add(1); w.Write([]byte(`{}`)) }))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	client := connector.NewClientWithHTTP(&connector.ServerInfo{Port: port, CSRFToken: "synthetic"}, srv.Client())
	off := automation.NewAutomationController(nil, nil, nil, nil)
	off.SetClientFinder(func(context.Context) (*connector.Client, error) { return client, nil })
	off.RetryCoordinator().SetConversationsScanner(func(context.Context) ([]string, error) { return nil, nil })
	off.RetryCoordinator().SetSleeper(func(context.Context, time.Duration) error { return nil })
	off.RetryCoordinator().SetModelName("synthetic-model")
	off.Start(context.Background())
	off.RetryCoordinator().HandleAgentStateUpdate(context.Background(), connector.AgentStateUpdate{ConversationID: "synthetic-conversation", Status: "IDLE", TerminationReason: "RESOURCE_EXHAUSTED", ExecutionID: "synthetic-execution"})
	deadline := time.Now().Add(time.Second)
	for retryCalls.Load() == 0 && time.Now().Before(deadline) { time.Sleep(time.Millisecond) }
	result["disabled_controller_retry"] = map[string]any{"enabled": off.IsEnabled(), "mock_retry_calls": retryCalls.Load()}
	off.Stop()

	tmp, err := os.MkdirTemp(".tmp", "autoswitch-review-store-")
	if err != nil { panic(err) }
	defer os.Remove(tmp) // Empty directory only; never recursive removal.
	st, err := store.NewAccountStore(tmp)
	if err != nil { panic(err) }
	mgr, err := manager.NewAccountManager(st, nil)
	if err != nil { panic(err) }
	var identityReads atomic.Int32
	mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("synthetic unavailable") })
	mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		identityReads.Add(1)
		return &model.TokenData{RawPayload: []byte(`{"email":"active@example.invalid"}`)}, nil
	}, func(*model.TokenData) error { panic("credential write forbidden in diagnostic") })
	gate := automation.NewSwitchGate(fmt.Sprintf("Local\\ReviewOnly_%d", time.Now().UnixNano()))
	cache := telemetry.NewQuotaCache(filepath.Join(tmp, "unused-cache.json"))
	ctrl := automation.NewAutomationController(mgr, nil, cache, gate)
	ctrl.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("synthetic unavailable") })
	ctrl.SetEnabled(true)
	// SetEnabled launches an empty-cache evaluation; wait for that read to settle.
	time.Sleep(30*time.Millisecond)
	identityReads.Store(0)
	locked := make(chan struct{})
	release := make(chan struct{})
	gateDone := make(chan struct{})
	go func() { defer close(gateDone); gate.Execute(context.Background(), func(bool) error { close(locked); <-release; return nil }) }()
	<-locked
	ctrl.OnTelemetryUpdate([]*telemetry.AccountQuotaData{{Email: "active@example.invalid", Status: telemetry.StatusLow, Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.01, WeeklyFraction: 0.5}}})
	var events []string
	timer := time.NewTimer(300*time.Millisecond)
collect:
	for { select {
	case e := <-ctrl.StatusChannel(): events = append(events, string(e.Code))
	case <-timer.C: break collect
	} }
	result["proactive_waiting_on_gate"] = map[string]any{"events": events, "identity_reads": identityReads.Load(), "is_switching_while_gate_held": ctrl.IsSwitching(), "expected_if_context_alive": "switch remains pending at gate"}
	close(release)
	<-gateDone
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}
