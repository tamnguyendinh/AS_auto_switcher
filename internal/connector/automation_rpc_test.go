package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
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

	// 6. Test ResolveActiveModel with dynamic unknown model ID -> falls back gracefully
	_ = os.WriteFile(pbtxt, []byte("some_flag: true\nlast_selected_agent_model: \"MODEL_DYNAMIC_FUTURE_V4\"\n"), 0600)
	modelID, label, disabled, err = client.ResolveActiveModel(ctx, pbtxt)
	if err != nil {
		t.Fatalf("ResolveActiveModel failed: %v", err)
	}
	if modelID != "MODEL_DYNAMIC_FUTURE_V4" || label != "MODEL_DYNAMIC_FUTURE_V4" || disabled {
		t.Errorf("expected fallback for dynamic model, got id=%s label=%s disabled=%v", modelID, label, disabled)
	}
}
