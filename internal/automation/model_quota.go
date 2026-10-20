package automation

import (
	"fmt"
	"strings"
	"time"

	"antigravity-switch/internal/telemetry"
)

// QuotaCategory specifies which model quota family is active for evaluation.
type QuotaCategory string

const (
	CategoryGemini    QuotaCategory = "gemini"
	CategoryClaudeGPT QuotaCategory = "claude_gpt"
)

// QuotaState represents the ternary quota state (Tam Phân Thái).
type QuotaState string

const (
	ApplicableValid  QuotaState = "APPLICABLE_VALID"
	NotApplicable    QuotaState = "NOT_APPLICABLE"
	UnknownUncertain QuotaState = "UNKNOWN_UNCERTAIN"
)

// Threshold constants for proactive switching and candidate eligibility.
const (
	Threshold5hLow          = 0.02 // 5h <= 2% (0.02)
	ThresholdWeeklyDepleted = 0.01 // Weekly <= 1% (0.01)
)

// ModelQuotaEvaluation encapsulates the evaluation of an account quota against the active model category.
type ModelQuotaEvaluation struct {
	AccountEmail     string        `json:"account_email"`
	Category         QuotaCategory `json:"category"`
	ModelID          string        `json:"model_id"`
	ModelLabel       string        `json:"model_label,omitempty"`
	State            QuotaState    `json:"state"`
	FiveHourFraction float64       `json:"five_hour_fraction"`
	FiveHourDisabled bool          `json:"five_hour_disabled"`
	FiveHourReset    time.Time     `json:"five_hour_reset,omitempty"`
	WeeklyFraction   float64       `json:"weekly_fraction"`
	WeeklyDisabled   bool          `json:"weekly_disabled"`
	WeeklyReset      time.Time     `json:"weekly_reset,omitempty"`
	CanonicalReset   time.Time     `json:"canonical_reset,omitempty"`
	ShouldSwitch     bool          `json:"should_switch"` // Active account: threshold reached (5h <= 2% or weekly = 0%)
	IsEligible       bool          `json:"is_eligible"`   // Candidate account: eligible under criterion 4
	Reason           string        `json:"reason,omitempty"`
}

// ResolveActiveModelCategory determines the QuotaCategory for an active model given its ID and/or label.
func ResolveActiveModelCategory(modelID string, modelLabel string) QuotaCategory {
	combined := strings.ToLower(modelID + " " + modelLabel)

	// Check Gemini markers
	if strings.Contains(combined, "gemini") {
		return CategoryGemini
	}

	// Check Claude and GPT markers
	if strings.Contains(combined, "claude") ||
		strings.Contains(combined, "sonnet") ||
		strings.Contains(combined, "opus") ||
		strings.Contains(combined, "haiku") ||
		strings.Contains(combined, "gpt") ||
		strings.Contains(combined, "oss") ||
		strings.Contains(combined, "3p") {
		return CategoryClaudeGPT
	}

	// Default fallback to Gemini as per Antigravity primary engine
	return CategoryGemini
}

// ClassifyQuotaState evaluates the account quota under the given category and determines ternary state and decisions.
func ClassifyQuotaState(quota *telemetry.AccountQuotaData, category QuotaCategory) ModelQuotaEvaluation {
	eval := ModelQuotaEvaluation{
		Category:         category,
		State:            UnknownUncertain,
		FiveHourFraction: -1,
		WeeklyFraction:   -1,
	}

	if quota == nil {
		eval.Reason = "nil quota data"
		return eval
	}

	eval.AccountEmail = quota.Email
	eval.CanonicalReset = quota.ResetTime

	// Check if the telemetry fetch resulted in an error or missing payload
	if quota.Status == telemetry.StatusError || quota.ErrorMessage != "" {
		eval.Reason = fmt.Sprintf("telemetry error: %s", quota.ErrorMessage)
		return eval
	}

	var bucket telemetry.ModelQuotaBucket
	switch category {
	case CategoryGemini:
		bucket = quota.Gemini
	case CategoryClaudeGPT:
		bucket = quota.Claude
	default:
		bucket = quota.Gemini
	}

	eval.ModelID = bucket.ModelID
	eval.FiveHourFraction = bucket.FiveHourFraction
	eval.FiveHourDisabled = bucket.FiveHourDisabled
	eval.FiveHourReset = bucket.FiveHourReset
	eval.WeeklyFraction = bucket.WeeklyFraction
	eval.WeeklyDisabled = bucket.WeeklyDisabled
	eval.WeeklyReset = bucket.WeeklyReset

	// If both fractions are negative (-1), there is no valid quota data for this model family
	if eval.WeeklyFraction < 0 && eval.FiveHourFraction < 0 {
		eval.State = UnknownUncertain
		eval.Reason = "no quota data for model category"
		return eval
	}

	// If WeeklyFraction is negative (-1), data is incomplete / uncertain
	if eval.WeeklyFraction < 0 {
		eval.State = UnknownUncertain
		eval.Reason = "missing weekly quota data"
		return eval
	}

	// Check 5h bucket applicability:
	// A 5h bucket is NOT applicable if:
	// 1. It was explicitly marked disabled: true (on bucket or raw buckets), OR
	// 2. The 5h fraction is absent (-1) in a successful quota response that has weekly data.
	is5hDisabled := eval.FiveHourDisabled || bucket.Disabled
	if !is5hDisabled && len(quota.RawBuckets) > 0 {
		for _, rawB := range quota.RawBuckets {
			lowerID := strings.ToLower(rawB.ModelID)
			lowerToken := strings.ToLower(rawB.TokenType)
			is5h := strings.Contains(lowerID, "5h") || rawB.FiveHourFraction >= 0
			isMatch := false
			if category == CategoryGemini && (strings.Contains(lowerID, "gemini") || lowerToken == "gemini") {
				isMatch = true
			} else if category == CategoryClaudeGPT && (strings.Contains(lowerID, "claude") || strings.Contains(lowerID, "3p") || lowerToken == "claude") {
				isMatch = true
			}
			if isMatch && is5h && (rawB.Disabled || rawB.FiveHourDisabled) {
				is5hDisabled = true
				eval.FiveHourDisabled = true
				break
			}
			_ = isMatch
		}
	}

	has5hFraction := eval.FiveHourFraction >= 0

	if is5hDisabled || !has5hFraction {
		eval.State = NotApplicable
		// Proactive switch condition for NOT_APPLICABLE: weekly <= 0.01
		eval.ShouldSwitch = eval.WeeklyFraction <= ThresholdWeeklyDepleted
		// Candidate eligibility condition for NOT_APPLICABLE: weekly > 0.01
		eval.IsEligible = eval.WeeklyFraction > ThresholdWeeklyDepleted
		if eval.ShouldSwitch {
			eval.Reason = fmt.Sprintf("weekly quota depleted (%.1f%% <= 1%%)", eval.WeeklyFraction*100)
		} else {
			eval.Reason = fmt.Sprintf("weekly quota active (%.1f%%, 5h not applicable)", eval.WeeklyFraction*100)
		}
		return eval
	}

	// Otherwise, 5h is applicable and valid
	eval.State = ApplicableValid
	// Proactive switch condition for APPLICABLE_VALID: 5h <= 2% OR weekly <= 1%
	is5hLow := eval.FiveHourFraction <= Threshold5hLow
	isWeeklyDepleted := eval.WeeklyFraction <= ThresholdWeeklyDepleted
	eval.ShouldSwitch = is5hLow || isWeeklyDepleted

	// Candidate eligibility condition for APPLICABLE_VALID: 5h > 2% AND weekly > 1%
	eval.IsEligible = (eval.FiveHourFraction > Threshold5hLow) && (eval.WeeklyFraction > ThresholdWeeklyDepleted)

	if is5hLow && isWeeklyDepleted {
		eval.Reason = fmt.Sprintf("5h quota low (%.1f%% <= 2%%) and weekly depleted (%.1f%% <= 1%%)", eval.FiveHourFraction*100, eval.WeeklyFraction*100)
	} else if is5hLow {
		eval.Reason = fmt.Sprintf("5h quota low (%.1f%% <= 2%%)", eval.FiveHourFraction*100)
	} else if isWeeklyDepleted {
		eval.Reason = fmt.Sprintf("weekly quota depleted (%.1f%% <= 1%%)", eval.WeeklyFraction*100)
	} else {
		eval.Reason = fmt.Sprintf("quota healthy (5h: %.1f%%, weekly: %.1f%%)", eval.FiveHourFraction*100, eval.WeeklyFraction*100)
	}

	return eval
}

// EvaluateAccountQuota is a convenience function that resolves model category and classifies quota.
func EvaluateAccountQuota(quota *telemetry.AccountQuotaData, modelID string, modelLabel string) ModelQuotaEvaluation {
	category := ResolveActiveModelCategory(modelID, modelLabel)
	eval := ClassifyQuotaState(quota, category)
	eval.ModelID = modelID
	eval.ModelLabel = modelLabel
	return eval
}

