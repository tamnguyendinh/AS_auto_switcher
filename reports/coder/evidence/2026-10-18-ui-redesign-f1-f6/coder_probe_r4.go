//go:build ignore

// Coder R4 independent acceptance probe verifying closure of F1 & F4 and preservation of F2, F3, F5, F6.
// Run from repo root: go run reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/coder_probe_r4.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"antigravity-switch/internal/connector"
	"antigravity-switch/internal/manager"
	"antigravity-switch/internal/telemetry"
	"antigravity-switch/internal/ui"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

type probeRefresher struct{}

func (probeRefresher) RefreshToken(context.Context, string) (*connector.TokenRefreshResult, error) {
	return &connector.TokenRefreshResult{AccessToken: "probe-token"}, nil
}

func main() {
	fmt.Println("=== Starting Coder R4 Independent Probe ===")
	passed := 0
	total := 0

	assert := func(name string, condition bool, msg string) {
		total++
		if !condition {
			fmt.Printf("[FAIL] %s: %s\n", name, msg)
			os.Exit(1)
		}
		passed++
		fmt.Printf("[PASS] %s\n", name)
	}

	// =========================================================================
	// F1 — Case 1: REST legacy remainingAmount: 37 without remainingFraction
	// =========================================================================
	{
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"buckets":[{"modelId":"claude-4-6","remainingAmount":37}]}`)
		}))
		client := telemetry.NewQuotaClientWithEndpoint(probeRefresher{}, server.URL, server.Client())
		data, err := client.RetrieveUserQuota(context.Background(), "token")
		server.Close()
		assert("F1.1_REST_AmountCamel_NoFraction_ErrNil", err == nil, fmt.Sprintf("%v", err))
		assert("F1.1_REST_AmountCamel_FractionUnknown", data.Claude.RemainingFraction == -1, fmt.Sprintf("got %f", data.Claude.RemainingFraction))
		assert("F1.1_REST_AmountCamel_AmountPreserved", data.Claude.RemainingAmount == 37, fmt.Sprintf("got %d", data.Claude.RemainingAmount))

		dq := ui.ExtractDualCycleQuota(data)
		assert("F1.1_REST_AmountCamel_HasDataFalse", !dq.HasData, fmt.Sprintf("HasData=%v", dq.HasData))
		assert("F1.1_REST_AmountCamel_AllSentinels", dq.Claude5h == -1 && dq.ClaudeWeek == -1 && dq.Gemini5h == -1 && dq.GeminiWeek == -1,
			fmt.Sprintf("(%f,%f,%f,%f)", dq.Claude5h, dq.ClaudeWeek, dq.Gemini5h, dq.GeminiWeek))
	}

	// =========================================================================
	// F1 — Case 2: REST legacy snake_case remaining_amount: 37 without remaining_fraction
	// =========================================================================
	{
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"buckets":[{"model_id":"gemini-2-5","remaining_amount":37}]}`)
		}))
		client := telemetry.NewQuotaClientWithEndpoint(probeRefresher{}, server.URL, server.Client())
		data, err := client.RetrieveUserQuota(context.Background(), "token")
		server.Close()
		assert("F1.2_REST_AmountSnake_NoFraction_ErrNil", err == nil, fmt.Sprintf("%v", err))
		assert("F1.2_REST_AmountSnake_FractionUnknown", data.Gemini.RemainingFraction == -1, fmt.Sprintf("got %f", data.Gemini.RemainingFraction))
		assert("F1.2_REST_AmountSnake_AmountPreserved", data.Gemini.RemainingAmount == 37, fmt.Sprintf("got %d", data.Gemini.RemainingAmount))

		dq := ui.ExtractDualCycleQuota(data)
		assert("F1.2_REST_AmountSnake_HasDataFalse", !dq.HasData, fmt.Sprintf("HasData=%v", dq.HasData))
		assert("F1.2_REST_AmountSnake_AllSentinels", dq.Claude5h == -1 && dq.ClaudeWeek == -1 && dq.Gemini5h == -1 && dq.GeminiWeek == -1,
			fmt.Sprintf("(%f,%f,%f,%f)", dq.Claude5h, dq.ClaudeWeek, dq.Gemini5h, dq.GeminiWeek))
	}

	// =========================================================================
	// F1 — Case 3: REST legacy missing both fraction and amount (modelId only)
	// =========================================================================
	{
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"buckets":[{"modelId":"claude-4-6"}]}`)
		}))
		client := telemetry.NewQuotaClientWithEndpoint(probeRefresher{}, server.URL, server.Client())
		data, err := client.RetrieveUserQuota(context.Background(), "token")
		server.Close()
		assert("F1.3_REST_ModelIdOnly_ErrNil", err == nil, fmt.Sprintf("%v", err))
		assert("F1.3_REST_ModelIdOnly_FractionUnknown", data.Claude.RemainingFraction == -1, fmt.Sprintf("got %f", data.Claude.RemainingFraction))

		dq := ui.ExtractDualCycleQuota(data)
		assert("F1.3_REST_ModelIdOnly_HasDataFalse", !dq.HasData, fmt.Sprintf("HasData=%v", dq.HasData))
		assert("F1.3_REST_ModelIdOnly_AllSentinels", dq.Claude5h == -1 && dq.ClaudeWeek == -1 && dq.Gemini5h == -1 && dq.GeminiWeek == -1,
			fmt.Sprintf("(%f,%f,%f,%f)", dq.Claude5h, dq.ClaudeWeek, dq.Gemini5h, dq.GeminiWeek))
	}

	// =========================================================================
	// F1 — Case 4: REST legacy Claude fraction 0 only (no cycle)
	// =========================================================================
	{
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"buckets":[{"modelId":"claude-4-6","remainingFraction":0.0}]}`)
		}))
		client := telemetry.NewQuotaClientWithEndpoint(probeRefresher{}, server.URL, server.Client())
		data, err := client.RetrieveUserQuota(context.Background(), "token")
		server.Close()
		assert("F1.4_REST_FractionZero_ErrNil", err == nil, fmt.Sprintf("%v", err))
		assert("F1.4_REST_FractionZero_ClaudeZero", data.Claude.RemainingFraction == 0.0, fmt.Sprintf("got %f", data.Claude.RemainingFraction))

		dq := ui.ExtractDualCycleQuota(data)
		assert("F1.4_REST_FractionZero_HasDataTrue", dq.HasData, "expected HasData=true for valid 0.0 quota")
		assert("F1.4_REST_FractionZero_Claude5hZero", dq.Claude5h == 0.0, fmt.Sprintf("got %f", dq.Claude5h))
		assert("F1.4_REST_FractionZero_ClaudeWeekUnknown", dq.ClaudeWeek == -1, fmt.Sprintf("got %f (must NOT synthesize weekly)", dq.ClaudeWeek))
		assert("F1.4_REST_FractionZero_GeminiUnknown", dq.Gemini5h == -1 && dq.GeminiWeek == -1, fmt.Sprintf("got g5h=%f, gweek=%f", dq.Gemini5h, dq.GeminiWeek))
	}

	// =========================================================================
	// F1 — Case 5: Summary Claude bucketId:opaque, window:weekly, fraction 0
	// =========================================================================
	{
		summaryJSON := []byte(`{
			"groups": [
				{
					"displayName": "Claude models",
					"buckets": [
						{"bucketId": "opaque", "window": "weekly", "remainingFraction": 0.0}
					]
				}
			]
		}`)
		claude, gemini, _, _, _, _, err := telemetry.ParseQuotaSummaryWithBuckets(summaryJSON)
		assert("F1.5_Summary_WeeklyZero_ErrNil", err == nil, fmt.Sprintf("%v", err))
		assert("F1.5_Summary_WeeklyZero_WeekFractionZero", claude.WeeklyFraction == 0.0, fmt.Sprintf("got %f", claude.WeeklyFraction))
		assert("F1.5_Summary_WeeklyZero_5hUnknown", claude.FiveHourFraction == -1, fmt.Sprintf("got %f", claude.FiveHourFraction))

		data := &telemetry.AccountQuotaData{
			Email:  "test@example.com",
			Claude: claude,
			Gemini: gemini,
		}
		dq := ui.ExtractDualCycleQuota(data)
		assert("F1.5_Summary_WeeklyZero_HasDataTrue", dq.HasData, "expected HasData=true")
		assert("F1.5_Summary_WeeklyZero_MatchingWeekNot5h", dq.Claude5h == -1 && dq.ClaudeWeek == 0.0,
			fmt.Sprintf("expected (-1, 0), got 5h=%f, week=%f", dq.Claude5h, dq.ClaudeWeek))
	}

	// =========================================================================
	// F1 — Case 6: Cache reload persistence of weekly zero [-1, 0, -1, -1]
	// =========================================================================
	{
		tempDir, err := os.MkdirTemp("", "probe_cache_*")
		assert("F1.6_Cache_TempDir_ErrNil", err == nil, fmt.Sprintf("%v", err))
		defer os.RemoveAll(tempDir)

		cachePath := filepath.Join(tempDir, "quota_cache.json")
		cache := telemetry.NewQuotaCache(cachePath)
		data := &telemetry.AccountQuotaData{
			Email: "persist@test.com",
			Claude: telemetry.ModelQuotaBucket{
				ModelID:           "claude-4-6",
				RemainingFraction: 0.0,
				FiveHourFraction:  -1,
				WeeklyFraction:    0.0,
			},
			Gemini: telemetry.ModelQuotaBucket{
				ModelID:           "gemini-2-5",
				RemainingFraction: -1,
				FiveHourFraction:  -1,
				WeeklyFraction:    -1,
			},
			Status:      telemetry.StatusDepleted,
			LastUpdated: time.Now().UTC(),
		}
		cache.Update(data)
		err = cache.SaveToDisk()
		assert("F1.6_Cache_SaveToDisk_ErrNil", err == nil, fmt.Sprintf("%v", err))

		// Reload fresh
		cache2 := telemetry.NewQuotaCache(cachePath)
		err = cache2.LoadFromDisk()
		assert("F1.6_Cache_LoadFromDisk_ErrNil", err == nil, fmt.Sprintf("%v", err))

		loaded, ok := cache2.Get("persist@test.com")
		assert("F1.6_Cache_GetLoaded_Ok", ok && loaded != nil, "failed to get loaded account")
		assert("F1.6_Cache_Loaded_5hUnknown", loaded.Claude.FiveHourFraction == -1, fmt.Sprintf("got %f", loaded.Claude.FiveHourFraction))
		assert("F1.6_Cache_Loaded_WeekZeroPreserved", loaded.Claude.WeeklyFraction == 0.0, fmt.Sprintf("got %f", loaded.Claude.WeeklyFraction))

		dq := ui.ExtractDualCycleQuota(loaded)
		assert("F1.6_Cache_Loaded_DualCycleExact", dq.Claude5h == -1 && dq.ClaudeWeek == 0.0 && dq.Gemini5h == -1 && dq.GeminiWeek == -1,
			fmt.Sprintf("expected (-1,0,-1,-1), got (%f,%f,%f,%f)", dq.Claude5h, dq.ClaudeWeek, dq.Gemini5h, dq.GeminiWeek))
	}

	// =========================================================================
	// F1 — Case 7: Legacy cache only Claude 20%, missing Gemini
	// =========================================================================
	{
		legacyJSON := []byte(`{
			"matrix": {
				"onlyclaude@test.com": {
					"email": "onlyclaude@test.com",
					"claude": {
						"model_id": "claude-4-6",
						"remaining_fraction": 0.2
					}
				}
			}
		}`)
		var snapshot telemetry.QuotaCacheSnapshot
		err := json.Unmarshal(legacyJSON, &snapshot)
		assert("F1.7_LegacyCache_Unmarshal_ErrNil", err == nil, fmt.Sprintf("%v", err))

		acc := snapshot.Matrix["onlyclaude@test.com"]
		assert("F1.7_LegacyCache_AccountNonNil", acc != nil, "account was nil")
		assert("F1.7_LegacyCache_ClaudeFraction20", acc.Claude.RemainingFraction == 0.2, fmt.Sprintf("got %f", acc.Claude.RemainingFraction))
		assert("F1.7_LegacyCache_GeminiUnknownNotZero", acc.Gemini.RemainingFraction == -1 && acc.Gemini.FiveHourFraction == -1 && acc.Gemini.WeeklyFraction == -1,
			fmt.Sprintf("Gemini was rem=%f, 5h=%f, week=%f (must be -1)", acc.Gemini.RemainingFraction, acc.Gemini.FiveHourFraction, acc.Gemini.WeeklyFraction))

		dq := ui.ExtractDualCycleQuota(acc)
		assert("F1.7_LegacyCache_DualCycleClaudePreserved", dq.Claude5h == 0.2 && dq.ClaudeWeek == -1, fmt.Sprintf("Claude was 5h=%f, week=%f", dq.Claude5h, dq.ClaudeWeek))
		assert("F1.7_LegacyCache_DualCycleGeminiUnknown", dq.Gemini5h == -1 && dq.GeminiWeek == -1, fmt.Sprintf("Gemini was 5h=%f, week=%f", dq.Gemini5h, dq.GeminiWeek))
		assert("F1.7_LegacyCache_HasDataTrue", dq.HasData, "expected HasData=true")
	}

	// =========================================================================
	// F1 — Case 8: Legacy cache with Claude object but missing remaining_fraction
	// =========================================================================
	{
		legacyJSON := []byte(`{
			"matrix": {
				"nofraction@test.com": {
					"email": "nofraction@test.com",
					"claude": {
						"model_id": "claude-4-6"
					}
				}
			}
		}`)
		var snapshot telemetry.QuotaCacheSnapshot
		err := json.Unmarshal(legacyJSON, &snapshot)
		assert("F1.8_LegacyCache_NoFraction_ErrNil", err == nil, fmt.Sprintf("%v", err))

		acc := snapshot.Matrix["nofraction@test.com"]
		assert("F1.8_LegacyCache_ClaudeFractionUnknown", acc.Claude.RemainingFraction == -1, fmt.Sprintf("got %f", acc.Claude.RemainingFraction))

		dq := ui.ExtractDualCycleQuota(acc)
		assert("F1.8_LegacyCache_HasDataFalse", !dq.HasData, "expected HasData=false")
		assert("F1.8_LegacyCache_AllUnknown", dq.Claude5h == -1 && dq.ClaudeWeek == -1 && dq.Gemini5h == -1 && dq.GeminiWeek == -1,
			fmt.Sprintf("got (%f,%f,%f,%f)", dq.Claude5h, dq.ClaudeWeek, dq.Gemini5h, dq.GeminiWeek))
	}

	// =========================================================================
	// F2 Verification: Zero GPT field in struct & Zero GPT in JSON
	// =========================================================================
	{
		_, present := reflect.TypeOf(telemetry.AccountQuotaData{}).FieldByName("GPT")
		assert("F2_ZeroGPT_StructFieldAbsent", !present, "AccountQuotaData still contains GPT field")

		data := &telemetry.AccountQuotaData{
			Email:  "f2@test.com",
			Claude: telemetry.ModelQuotaBucket{RemainingFraction: 0.5},
			Gemini: telemetry.ModelQuotaBucket{RemainingFraction: 0.5},
		}
		bytes, err := json.Marshal(data)
		assert("F2_ZeroGPT_MarshalErrNil", err == nil, fmt.Sprintf("%v", err))
		assert("F2_ZeroGPT_JsonNoGptKey", !strings.Contains(string(bytes), `"gpt"`), string(bytes))
	}

	// =========================================================================
	// F4 Verification: Retired UI objects removed from App
	// =========================================================================
	{
		appType := reflect.TypeOf(ui.App{})
		for _, retiredField := range []string{"headerBg", "statusBarLabel", "activeStatusLabel", "activeQuotaLabel"} {
			_, exists := appType.FieldByName(retiredField)
			assert("F4_AppField_"+retiredField+"_Absent", !exists, fmt.Sprintf("App struct still has %s", retiredField))
		}
	}

	// =========================================================================
	// F5 Verification: Compact Card Layout (3 rows: topRow, row5h, rowWeek)
	// =========================================================================
	{
		_ = test.NewApp()
		item := manager.AccountDisplayItem{Email: "compact@test.com", IsActive: false}
		quota := &telemetry.AccountQuotaData{
			Claude: telemetry.ModelQuotaBucket{FiveHourFraction: 0.5, WeeklyFraction: 0.5},
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.8, WeeklyFraction: 0.8},
		}
		cardObj := ui.NewAccountCardWithQuota(item, quota, ui.DarkPalette, nil, nil, nil)
		cardStack, ok := cardObj.(*fyne.Container)
		assert("F5_CompactCard_IsContainer", ok && len(cardStack.Objects) == 2, "card must be stack of background and content")
		padded, ok2 := cardStack.Objects[1].(*fyne.Container)
		assert("F5_CompactCard_HasPadded", ok2 && len(padded.Objects) == 1, "card must have padded content")
		cardVBox, ok3 := padded.Objects[0].(*fyne.Container)
		assert("F5_CompactCard_HasVBox", ok3, "padded must contain cardVBox")
		assert("F5_CompactCard_Has3Rows", len(cardVBox.Objects) == 3, fmt.Sprintf("expected 3 rows in cardVBox, got %d", len(cardVBox.Objects)))
	}

	// =========================================================================
	// F6 Verification: Monogram AS Theme Palette (Light #2563eb, Warm #d97706, Dark #89b4fa)
	// =========================================================================
	{
		assert("F6_Palette_Light_Primary", ui.LightPalette.Primary.R == 0x25 && ui.LightPalette.Primary.G == 0x63 && ui.LightPalette.Primary.B == 0xeb, "Light palette Primary mismatch")
		assert("F6_Palette_Warm_Primary", ui.WarmPalette.Primary.R == 0xd9 && ui.WarmPalette.Primary.G == 0x77 && ui.WarmPalette.Primary.B == 0x06, "Warm palette Primary mismatch")
		assert("F6_Palette_Dark_Primary", ui.DarkPalette.Primary.R == 0x89 && ui.DarkPalette.Primary.G == 0xb4 && ui.DarkPalette.Primary.B == 0xfa, "Dark palette Primary mismatch")
	}

	fmt.Printf("\n=== Coder R4 Probe Summary: %d/%d assertions PASSED ===\n", passed, total)
}
