package automation

import (
	"testing"
	"time"

	"as-auto-switcher/internal/engines/antigravity/telemetry"
)

func TestResolveActiveModelCategory(t *testing.T) {
	tests := []struct {
		name       string
		modelID    string
		modelLabel string
		want       QuotaCategory
	}{
		{"Gemini by ID", "gemini-2.5-pro", "", CategoryGemini},
		{"Gemini by Enum", "MODEL_GEMINI_2_5_FLASH", "", CategoryGemini},
		{"Gemini by Label with M318 enum", "MODEL_PLACEHOLDER_M318", "Gemini 2.5 Pro", CategoryGemini},
		{"Claude by ID", "claude-3-5-sonnet", "", CategoryClaudeGPT},
		{"Claude Sonnet by Label with custom enum", "MODEL_CUSTOM_PRO", "Claude 3.5 Sonnet", CategoryClaudeGPT},
		{"Claude Opus", "claude-3-opus", "", CategoryClaudeGPT},
		{"GPT-4o", "gpt-4o", "", CategoryClaudeGPT},
		{"GPT-OSS", "gpt-oss", "", CategoryClaudeGPT},
		{"3P model", "3p-model-variant", "", CategoryClaudeGPT},
		{"Unknown model", "unknown-custom-model", "", CategoryUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveActiveModelCategory(tt.modelID, tt.modelLabel)
			if got != tt.want {
				t.Errorf("ResolveActiveModelCategory(%q, %q) = %v; want %v", tt.modelID, tt.modelLabel, got, tt.want)
			}
		})
	}
}

func TestClassifyQuotaState_TernaryAndSwitchConditions(t *testing.T) {
	now := time.Now().UTC()

	t.Run("ApplicableValid - Healthy Quota (5h=80%, Weekly=50%)", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				WeeklyFraction:    0.50,
				RemainingFraction: 0.50,
				ResetTime:         now.Add(2 * time.Hour),
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false, got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true, got false")
		}
	})

	t.Run("ApplicableValid - Low 5h Quota <= 2% (5h=1%, Weekly=50%)", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.01,
				WeeklyFraction:    0.50,
				RemainingFraction: 0.01,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for 5h <= 2%%, got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for 5h <= 2%%, got true")
		}
	})

	t.Run("ApplicableValid - Exactly 2% 5h Quota triggers switch", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.02,
				WeeklyFraction:    0.30,
				RemainingFraction: 0.02,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for 5h == 2%%, got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for candidate at 5h == 2%%, got true")
		}
	})

	t.Run("ApplicableValid - Weekly <= 2% (1%) triggers switch even if 5h is high", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.90,
				WeeklyFraction:    0.01,
				RemainingFraction: 0.01,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for weekly=1%% (<= 2%%), got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for weekly=1%% (<= 2%%), got true")
		}
	})

	t.Run("ApplicableValid - Weekly = 0% triggers switch even if 5h is high", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.90,
				WeeklyFraction:    0.00,
				RemainingFraction: 0.00,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for weekly=0%%, got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for weekly=0%%, got true")
		}
	})

	t.Run("ApplicableValid - Exactly 2% Weekly Quota triggers switch even if 5h is high", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				WeeklyFraction:    0.02,
				RemainingFraction: 0.02,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for weekly=2%% (<= 2%%), got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for weekly=2%% (<= 2%%), got true")
		}
	})

	t.Run("ApplicableValid - Marginal depleted weekly (1.5%) triggers switch and is ineligible", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				WeeklyFraction:    0.015,
				RemainingFraction: 0.015,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for weekly=1.5%% (<= 2%%), got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for candidate at weekly=1.5%%, got true")
		}
	})

	t.Run("ApplicableValid - Weekly > 2% (2.1%) does NOT trigger switch if 5h is high", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				WeeklyFraction:    0.021,
				RemainingFraction: 0.021,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false for weekly=2.1%% (> 2%%), got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true for weekly=2.1%% (> 2%%), got false")
		}
	})

	t.Run("ApplicableValid - AND Condition Check: Low 5h (1.9%) with Healthy Weekly (80%) is Ineligible", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.019,
				WeeklyFraction:    0.80,
				RemainingFraction: 0.019,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for 5h=1.9%% <= 2%%, got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for candidate with 5h <= 2%%, got true")
		}
	})

	t.Run("ApplicableValid - AND Condition Check: Both strictly > 2% (5h=2.1%, Weekly=2.1%) is Eligible", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.021,
				WeeklyFraction:    0.021,
				RemainingFraction: 0.021,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false for both > 2%%, got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true for both > 2%%, got false")
		}
	})

	t.Run("NotApplicable - 5h Missing (-1) in successful response, Weekly=10%", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  -1,
				WeeklyFraction:    0.10,
				RemainingFraction: 0.10,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false (weekly 10%% > 0%%), got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true, got false")
		}
	})

	t.Run("NotApplicable - 5h Disabled: true, Weekly=5%", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.01, // Even if 5h fraction is 1%, it is disabled!
				FiveHourDisabled:  true,
				WeeklyFraction:    0.05,
				RemainingFraction: 0.05,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		// When 5h is disabled, 5h <= 2% condition is ignored!
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false because 5h is disabled and weekly=5%% > 2%%, got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true because weekly=5%% > 2%%, got false")
		}
	})

	t.Run("NotApplicable - 5h Disabled: true, Weekly=2% triggers switch and is ineligible", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				FiveHourDisabled:  true,
				WeeklyFraction:    0.02,
				RemainingFraction: 0.02,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true because weekly=2%% (<= 2%%), got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false because weekly=2%% (<= 2%%), got true")
		}
	})

	t.Run("NotApplicable - 5h Disabled: true, Weekly=2.1% is eligible", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				FiveHourDisabled:  true,
				WeeklyFraction:    0.021,
				RemainingFraction: 0.021,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false because weekly=2.1%% (> 2%%), got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true because weekly=2.1%% (> 2%%), got false")
		}
	})

	t.Run("NotApplicable - 5h Disabled: true, Weekly=1%", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				FiveHourDisabled:  true,
				WeeklyFraction:    0.01,
				RemainingFraction: 0.01,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true because weekly=1%% (<= 2%%), got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false because weekly=1%% (<= 2%%), got true")
		}
	})

	t.Run("NotApplicable - 5h Disabled: true, Weekly=0%", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.80,
				FiveHourDisabled:  true,
				WeeklyFraction:    0.00,
				RemainingFraction: 0.00,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true because weekly=0%%, got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false because weekly=0%%, got true")
		}
	})

	t.Run("UnknownUncertain - Nil Quota Data", func(t *testing.T) {
		eval := ClassifyQuotaState(nil, CategoryGemini)
		if eval.State != UnknownUncertain {
			t.Errorf("expected state %s, got %s", UnknownUncertain, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("must NOT switch on uncertain data")
		}
		if eval.IsEligible {
			t.Errorf("must NOT be eligible on uncertain data")
		}
	})

	t.Run("UnknownUncertain - Telemetry Status Error", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email:        "user@example.com",
			Status:       telemetry.StatusError,
			ErrorMessage: "dial tcp: i/o timeout",
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != UnknownUncertain {
			t.Errorf("expected state %s, got %s", UnknownUncertain, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("must NOT switch on error status")
		}
		if eval.IsEligible {
			t.Errorf("must NOT be eligible on error status")
		}
	})

	t.Run("UnknownUncertain - Missing Weekly Fraction (-1)", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "user@example.com",
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2.5",
				FiveHourFraction:  0.50,
				WeeklyFraction:    -1, // Missing!
				RemainingFraction: 0.50,
			},
		}
		eval := ClassifyQuotaState(data, CategoryGemini)
		if eval.State != UnknownUncertain {
			t.Errorf("expected state %s, got %s", UnknownUncertain, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("must NOT switch on missing weekly fraction")
		}
		if eval.IsEligible {
			t.Errorf("must NOT be eligible on missing weekly fraction")
		}
	})
}

func TestActiveModelBinding_ModelIsolation(t *testing.T) {
	// CRITICAL INVARIANT TEST:
	// Phiên đang chạy Gemini thì việc Claude hết hạn ngạch tuần TUYỆT ĐỐI KHÔNG ĐƯỢC CHUYỂN TÀI KHOẢN.
	// Khi người dùng đổi model sang Claude, hệ thống lập tức tính toán lại theo nhóm quota mới.
	data := &telemetry.AccountQuotaData{
		Email: "active_user@example.com",
		Gemini: telemetry.ModelQuotaBucket{
			ModelID:           "gemini-2.5",
			FiveHourFraction:  0.80,
			WeeklyFraction:    0.60,
			RemainingFraction: 0.60,
		},
		Claude: telemetry.ModelQuotaBucket{
			ModelID:           "claude-4-6",
			FiveHourFraction:  0.00,
			WeeklyFraction:    0.00, // Claude is completely exhausted!
			RemainingFraction: 0.00,
		},
	}

	// 1. When active model is Gemini:
	evalGemini := ClassifyQuotaState(data, CategoryGemini)
	if evalGemini.ShouldSwitch {
		t.Fatalf("FATAL INVARIANT VIOLATION: Claude being exhausted triggered switch while active model is Gemini!")
	}
	if !evalGemini.IsEligible {
		t.Fatalf("expected Gemini to be eligible, got false")
	}

	// 2. When active model changes to Claude:
	evalClaude := ClassifyQuotaState(data, CategoryClaudeGPT)
	if !evalClaude.ShouldSwitch {
		t.Fatalf("expected switch triggered when active model changed to Claude, got false")
	}
	if evalClaude.IsEligible {
		t.Fatalf("expected Claude to be ineligible when exhausted, got true")
	}
}

func TestResolveActiveModelCategory_DynamicExtensions(t *testing.T) {
	tests := []struct {
		name             string
		modelID          string
		modelLabel       string
		discoveredTokens []string
		want             QuotaCategory
	}{
		{
			name:             "Argon via discoveredTokens",
			modelID:          "MODEL_PLACEHOLDER_M421",
			modelLabel:       "Gemini 4 Argon",
			discoveredTokens: []string{"argon"},
			want:             QuotaCategory("argon"),
		},
		{
			name:       "Argon fallback without tokens",
			modelID:    "MODEL_PLACEHOLDER_M421",
			modelLabel: "Gemini 4 Argon",
			want:       QuotaCategory("argon"),
		},
		{
			name:       "Argon with qualifier",
			modelID:    "MODEL_PLACEHOLDER_M421",
			modelLabel: "Gemini 4 Argon (Thinking)",
			want:       QuotaCategory("argon"),
		},
		{
			name:       "Argon with tier qualifier and technical ID",
			modelID:    "gemini-4-argon",
			modelLabel: "Gemini 4 Argon (High)",
			want:       QuotaCategory("argon"),
		},
		{
			name:       "Preserve baseline Gemini 3.8 Flash (High)",
			modelID:    "MODEL_PLACEHOLDER_M318",
			modelLabel: "Gemini 3.8 Flash (High)",
			want:       CategoryGemini,
		},
		{
			name:       "Preserve baseline Gemini 2.5 Pro",
			modelID:    "MODEL_PLACEHOLDER_M319",
			modelLabel: "Gemini 2.5 Pro",
			want:       CategoryGemini,
		},
		{
			name:       "Preserve baseline Claude Sonnet 4.6 (Thinking)",
			modelID:    "MODEL_PLACEHOLDER_M25",
			modelLabel: "Claude Sonnet 4.6 (Thinking)",
			want:       CategoryClaudeGPT,
		},
		{
			name:       "Preserve baseline GPT-OSS 120B (Medium)",
			modelID:    "MODEL_OPENAI_GPT_OSS_120B_MEDIUM",
			modelLabel: "GPT-OSS 120B (Medium)",
			want:       CategoryClaudeGPT,
		},
		{
			name:       "Unknown model returns CategoryUnknown",
			modelID:    "MODEL_UNKNOWN_XYZ",
			modelLabel: "Custom Autonomous Model",
			want:       CategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveActiveModelCategory(tt.modelID, tt.modelLabel, tt.discoveredTokens...)
			if got != tt.want {
				t.Errorf("ResolveActiveModelCategory(%q, %q, %v) = %v; want %v", tt.modelID, tt.modelLabel, tt.discoveredTokens, got, tt.want)
			}
		})
	}
}

func TestClassifyQuotaState_DynamicExtensions(t *testing.T) {
	now := time.Now().UTC()

	t.Run("Dual-cycle non-breaking accumulation healthy", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "argon_dual@example.com",
			RawBuckets: []telemetry.ModelQuotaBucket{
				{
					ModelID:          "argon-5h",
					TokenType:        "argon",
					FiveHourFraction: 0.80,
					FiveHourReset:    now.Add(3 * time.Hour),
				},
				{
					ModelID:        "argon-weekly",
					TokenType:      "argon",
					WeeklyFraction: 0.50,
					WeeklyReset:    now.Add(120 * time.Hour),
				},
			},
		}

		eval := ClassifyQuotaState(data, QuotaCategory("argon"))
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false, got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true, got false")
		}
		if eval.FiveHourFraction != 0.80 || eval.WeeklyFraction != 0.50 {
			t.Errorf("expected 5h=0.80 and weekly=0.50, got 5h=%f, weekly=%f", eval.FiveHourFraction, eval.WeeklyFraction)
		}
	})

	t.Run("Dual-cycle 5h low <= 2% triggers switch", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "argon_5h_low@example.com",
			RawBuckets: []telemetry.ModelQuotaBucket{
				{
					ModelID:          "argon-5h",
					TokenType:        "argon",
					FiveHourFraction: 0.01,
					FiveHourReset:    now.Add(1 * time.Hour),
				},
				{
					ModelID:        "argon-weekly",
					TokenType:      "argon",
					WeeklyFraction: 0.50,
					WeeklyReset:    now.Add(100 * time.Hour),
				},
			},
		}

		eval := ClassifyQuotaState(data, QuotaCategory("argon"))
		if eval.State != ApplicableValid {
			t.Errorf("expected state %s, got %s", ApplicableValid, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true, got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false, got true")
		}
	})

	t.Run("Weekly-only Argon pattern (no 5h) marks NotApplicable and does not switch if weekly > 2%", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "argon_weekly_only@example.com",
			RawBuckets: []telemetry.ModelQuotaBucket{
				{
					ModelID:        "argon-weekly",
					TokenType:      "argon",
					WeeklyFraction: 0.85,
					WeeklyReset:    now.Add(140 * time.Hour),
				},
			},
		}

		eval := ClassifyQuotaState(data, QuotaCategory("argon"))
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if !eval.FiveHourDisabled {
			t.Errorf("expected FiveHourDisabled=true when 5h is missing, got false")
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false, got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true, got false")
		}
	})

	t.Run("Weekly-only Argon pattern depleted weekly <= 2% triggers switch", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "argon_depleted@example.com",
			RawBuckets: []telemetry.ModelQuotaBucket{
				{
					ModelID:        "argon-weekly",
					TokenType:      "argon",
					WeeklyFraction: 0.015,
					WeeklyReset:    now.Add(24 * time.Hour),
				},
			},
		}

		eval := ClassifyQuotaState(data, QuotaCategory("argon"))
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true when weekly <= 2%%, got false")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false when weekly <= 2%%, got true")
		}
	})

	t.Run("Missing token in quota buckets returns UnknownUncertain", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "argon_missing@example.com",
			RawBuckets: []telemetry.ModelQuotaBucket{
				{
					ModelID:        "gemini-weekly",
					TokenType:      "gemini",
					WeeklyFraction: 0.90,
				},
			},
		}

		eval := ClassifyQuotaState(data, QuotaCategory("argon"))
		if eval.State != UnknownUncertain {
			t.Errorf("expected state %s, got %s", UnknownUncertain, eval.State)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false, got true")
		}
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false, got true")
		}
	})

	t.Run("Locked or disabled model marks ineligible", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "argon_locked@example.com",
			RawBuckets: []telemetry.ModelQuotaBucket{
				{
					ModelID:        "argon-weekly",
					TokenType:      "argon",
					WeeklyFraction: 0.70,
					Disabled:       true,
				},
			},
		}

		eval := ClassifyQuotaState(data, QuotaCategory("argon"))
		if eval.IsEligible {
			t.Errorf("expected IsEligible=false for locked/disabled model, got true")
		}
		if !eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=true for active locked model, got false")
		}
	})

	t.Run("Lookup via ExtensionBuckets map", func(t *testing.T) {
		data := &telemetry.AccountQuotaData{
			Email: "argon_ext_map@example.com",
			ExtensionBuckets: map[string]telemetry.ModelQuotaBucket{
				"argon": {
					ModelID:        "argon",
					WeeklyFraction: 0.65,
					WeeklyReset:    now.Add(80 * time.Hour),
				},
			},
		}

		eval := ClassifyQuotaState(data, QuotaCategory("argon"))
		if eval.State != NotApplicable {
			t.Errorf("expected state %s, got %s", NotApplicable, eval.State)
		}
		if eval.WeeklyFraction != 0.65 {
			t.Errorf("expected WeeklyFraction=0.65, got %f", eval.WeeklyFraction)
		}
		if eval.ShouldSwitch {
			t.Errorf("expected ShouldSwitch=false, got true")
		}
		if !eval.IsEligible {
			t.Errorf("expected IsEligible=true, got false")
		}
	})
}
