package automation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"as-auto-switcher/internal/engines/antigravity/telemetry"
)

// QuotaCategory specifies which model quota family is active for evaluation.
type QuotaCategory string

const (
	CategoryUnknown   QuotaCategory = "unknown"
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
	ThresholdWeeklyDepleted = 0.02 // Weekly <= 2% (0.02) - Chuẩn hóa đồng nhất mốc 2%
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
	ShouldSwitch     bool          `json:"should_switch"` // Active account: threshold reached (5h <= 2% or weekly <= 2%)
	IsEligible       bool          `json:"is_eligible"`   // Candidate account: eligible under criterion 4 (> 2%)
	Reason           string        `json:"reason,omitempty"`
}

var rxQualifier = regexp.MustCompile(`\s*\([^)]*\)\s*`)

// ResolveActiveModelCategory determines the QuotaCategory for an active model given its ID and/or label.
// discoveredTokens is an optional list of model tokens discovered from the language server.
func ResolveActiveModelCategory(modelID string, modelLabel string, discoveredTokens ...string) QuotaCategory {
	combined := strings.ToLower(modelID + " " + modelLabel)
	cleanCombined := strings.TrimSpace(rxQualifier.ReplaceAllString(combined, " "))

	// 1. Kiểm tra ưu tiên nếu active model khớp với bất kỳ token nào trong danh sách mở rộng đã phát hiện
	for _, token := range discoveredTokens {
		t := strings.ToLower(strings.TrimSpace(token))
		if t != "" && strings.Contains(cleanCombined, t) {
			return QuotaCategory(t)
		}
	}

	// 2. Nhận diện Baseline 3P/Claude/GPT: Nếu chứa "claude" hoặc "gpt-oss", hoặc các markers 3p chuẩn
	if strings.Contains(cleanCombined, "claude") ||
		strings.Contains(cleanCombined, "sonnet") ||
		strings.Contains(cleanCombined, "opus") ||
		strings.Contains(cleanCombined, "haiku") ||
		strings.Contains(cleanCombined, "gpt") ||
		strings.Contains(cleanCombined, "oss") ||
		strings.Contains(cleanCombined, "3p") {
		return CategoryClaudeGPT
	}

	// 3. Nhận diện Baseline Gemini chuẩn:
	// Nếu chứa "gemini" VÀ chứa ("flash" HOẶC "pro") VÀ không chứa số phiên bản >= 4 hoặc các từ tố tên mã mở rộng -> CategoryGemini
	if strings.Contains(cleanCombined, "gemini") {
		isExtension := strings.Contains(cleanCombined, "argon")
		if !isExtension {
			// Kiểm tra nếu có version >= 4 (như "4", "v4", "4.0")
			for _, f := range strings.Fields(cleanCombined) {
				cleanF := strings.Trim(f, "v-._")
				if len(cleanF) > 0 && cleanF[0] >= '4' && cleanF[0] <= '9' {
					isExtension = true
					break
				}
			}
		}

		if !isExtension && (strings.Contains(cleanCombined, "flash") || strings.Contains(cleanCombined, "pro") || cleanCombined == "gemini") {
			return CategoryGemini
		}

		// Đối với Model Mở rộng: Trích xuất token chữ thường bằng cách loại bỏ qualifier trong ngoặc đơn, lấy từ cuối cùng (vd: "argon")
		words := strings.Fields(cleanCombined)
		if len(words) > 0 {
			lastWord := words[len(words)-1]
			if lastWord != "gemini" {
				return QuotaCategory(lastWord)
			}
		}

		if !isExtension {
			return CategoryGemini
		}
	}

	// An unresolved placeholder must never select a quota family implicitly.
	return CategoryUnknown
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
		token := strings.ToLower(string(category))
		if token == "" || token == string(CategoryUnknown) {
			eval.Reason = "unknown model quota family"
			return eval
		}

		var matched5h, matchedWeek *telemetry.ModelQuotaBucket
		hasMatch := false

		// Tìm kiếm bucket mở rộng từ quota.ExtensionBuckets[string(category)]
		if quota.ExtensionBuckets != nil {
			if extB, ok := quota.ExtensionBuckets[token]; ok {
				hasMatch = true
				bCopy := extB
				if bCopy.FiveHourFraction > 0 || (bCopy.FiveHourFraction == 0 && !bCopy.FiveHourReset.IsZero()) {
					matched5h = &bCopy
				}
				if bCopy.WeeklyFraction >= 0 {
					matchedWeek = &bCopy
				}
				if bCopy.FiveHourDisabled {
					eval.FiveHourDisabled = true
				}
				if bCopy.WeeklyDisabled {
					eval.WeeklyDisabled = true
				}
				if bCopy.Disabled {
					if matched5h == nil {
						matched5h = &bCopy
					}
					if matchedWeek == nil {
						matchedWeek = &bCopy
					}
				}
			}
		}

		// INV-P1-04: Duyệt toàn bộ rawBuckets và tích lũy dữ liệu cho cả 2 chu kỳ (TUYỆT ĐỐI KHÔNG DÙNG BREAK SỚM)
		for i := range quota.RawBuckets {
			rawB := &quota.RawBuckets[i]
			lowerID := strings.ToLower(rawB.ModelID)
			lowerToken := strings.ToLower(rawB.TokenType)
			if strings.Contains(lowerID, token) || strings.Contains(lowerToken, token) {
				hasMatch = true
				is5h := strings.Contains(lowerID, "5h")
				isWeek := strings.Contains(lowerID, "week")

				if !is5h && !isWeek {
					if rawB.FiveHourFraction > 0 || (rawB.FiveHourFraction == 0 && !rawB.FiveHourReset.IsZero()) {
						is5h = true
					}
					if rawB.WeeklyFraction >= 0 {
						isWeek = true
					}
				}

				if is5h {
					matched5h = rawB
				}
				if isWeek {
					matchedWeek = rawB
				}
			}
		}

		if !hasMatch {
			eval.State = UnknownUncertain
			eval.Reason = fmt.Sprintf("no quota bucket matching model token %q", token)
			return eval
		}

		eval.ModelID = token
		if matched5h != nil {
			eval.FiveHourFraction = matched5h.FiveHourFraction
			eval.FiveHourDisabled = matched5h.FiveHourDisabled || matched5h.Disabled
			eval.FiveHourReset = matched5h.FiveHourReset
		} else {
			eval.FiveHourFraction = -1
		}

		if matchedWeek != nil {
			eval.WeeklyFraction = matchedWeek.WeeklyFraction
			eval.WeeklyDisabled = matchedWeek.WeeklyDisabled || matchedWeek.Disabled
			eval.WeeklyReset = matchedWeek.WeeklyReset
		} else {
			eval.WeeklyFraction = -1
		}

		// Kiểm tra cờ disabled / locked
		isLocked := (matched5h != nil && (matched5h.Disabled || (matched5h.FiveHourDisabled && matchedWeek == nil))) ||
			(matchedWeek != nil && (matchedWeek.Disabled || matchedWeek.WeeklyDisabled))

		// Nếu model bị khóa/disabled: IsEligible = false
		if isLocked {
			eval.State = NotApplicable
			eval.FiveHourDisabled = true
			eval.WeeklyDisabled = true
			eval.ShouldSwitch = true
			eval.IsEligible = false
			eval.Reason = fmt.Sprintf("model %s is disabled/locked", token)
			return eval
		}

		// Nếu thiếu dữ liệu tuần (-1), chuyển sang trạng thái bất định
		if eval.WeeklyFraction < 0 {
			eval.State = UnknownUncertain
			eval.Reason = "missing weekly quota data"
			return eval
		}

		is5hDisabled := eval.FiveHourDisabled || !(eval.FiveHourFraction >= 0)

		// INV-P1-07: Nếu không có chu kỳ 5h (FiveHourFraction < 0, như Argon) -> FiveHourDisabled = true, NotApplicable
		if is5hDisabled {
			eval.State = NotApplicable
			eval.FiveHourDisabled = true
			eval.ShouldSwitch = eval.WeeklyFraction <= ThresholdWeeklyDepleted
			eval.IsEligible = eval.WeeklyFraction > ThresholdWeeklyDepleted
			if eval.ShouldSwitch {
				eval.Reason = fmt.Sprintf("weekly quota depleted (%.1f%% <= 2%%)", eval.WeeklyFraction*100)
			} else {
				eval.Reason = fmt.Sprintf("weekly quota active (%.1f%%, 5h not applicable)", eval.WeeklyFraction*100)
			}
			return eval
		}

		// Ngược lại, có đầy đủ cả 2 chu kỳ:
		eval.State = ApplicableValid
		is5hLow := eval.FiveHourFraction <= Threshold5hLow
		isWeeklyDepleted := eval.WeeklyFraction <= ThresholdWeeklyDepleted
		eval.ShouldSwitch = is5hLow || isWeeklyDepleted
		eval.IsEligible = (eval.FiveHourFraction > Threshold5hLow) && (eval.WeeklyFraction > ThresholdWeeklyDepleted)

		if is5hLow && isWeeklyDepleted {
			eval.Reason = fmt.Sprintf("5h quota low (%.1f%% <= 2%%) and weekly depleted (%.1f%% <= 2%%)", eval.FiveHourFraction*100, eval.WeeklyFraction*100)
		} else if is5hLow {
			eval.Reason = fmt.Sprintf("5h quota low (%.1f%% <= 2%%)", eval.FiveHourFraction*100)
		} else if isWeeklyDepleted {
			eval.Reason = fmt.Sprintf("weekly quota depleted (%.1f%% <= 2%%)", eval.WeeklyFraction*100)
		} else {
			eval.Reason = fmt.Sprintf("quota healthy (5h: %.1f%%, weekly: %.1f%%)", eval.FiveHourFraction*100, eval.WeeklyFraction*100)
		}

		return eval
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
		}
	}

	has5hFraction := eval.FiveHourFraction >= 0

	if is5hDisabled || !has5hFraction {
		eval.State = NotApplicable
		// Giai Đoạn 1 (Phát hiện): Kích hoạt chuyển khi tài khoản active cạn tuần: weekly <= 0.02 (<= 2%)
		eval.ShouldSwitch = eval.WeeklyFraction <= ThresholdWeeklyDepleted

		// Giai Đoạn 2 (Lọc sạch): Điều kiện hạn mức nền tảng để ứng viên hợp lệ: weekly > 0.02 (nghiêm ngặt > 2%)
		eval.IsEligible = eval.WeeklyFraction > ThresholdWeeklyDepleted
		if eval.ShouldSwitch {
			eval.Reason = fmt.Sprintf("weekly quota depleted (%.1f%% <= 2%%)", eval.WeeklyFraction*100)
		} else {
			eval.Reason = fmt.Sprintf("weekly quota active (%.1f%%, 5h not applicable)", eval.WeeklyFraction*100)
		}
		return eval
	}

	// Otherwise, 5h is applicable and valid
	eval.State = ApplicableValid
	// Giai Đoạn 1 (Phát hiện): Kích hoạt chuyển khi tài khoản active chạm ngưỡng: 5h <= 2% HOẶC weekly <= 2%
	is5hLow := eval.FiveHourFraction <= Threshold5hLow
	isWeeklyDepleted := eval.WeeklyFraction <= ThresholdWeeklyDepleted
	eval.ShouldSwitch = is5hLow || isWeeklyDepleted

	// Giai Đoạn 2 (Lọc sạch): Điều kiện hạn mức nền tảng bắt buộc dùng chữ VÀ: 5h > 2% VÀ weekly > 2% (nghiêm ngặt > 2%)
	eval.IsEligible = (eval.FiveHourFraction > Threshold5hLow) && (eval.WeeklyFraction > ThresholdWeeklyDepleted)

	if is5hLow && isWeeklyDepleted {
		eval.Reason = fmt.Sprintf("5h quota low (%.1f%% <= 2%%) and weekly depleted (%.1f%% <= 2%%)", eval.FiveHourFraction*100, eval.WeeklyFraction*100)
	} else if is5hLow {
		eval.Reason = fmt.Sprintf("5h quota low (%.1f%% <= 2%%)", eval.FiveHourFraction*100)
	} else if isWeeklyDepleted {
		eval.Reason = fmt.Sprintf("weekly quota depleted (%.1f%% <= 2%%)", eval.WeeklyFraction*100)
	} else {
		eval.Reason = fmt.Sprintf("quota healthy (5h: %.1f%%, weekly: %.1f%%)", eval.FiveHourFraction*100, eval.WeeklyFraction*100)
	}

	return eval
}

// EvaluateAccountQuota is a convenience function that resolves model category and classifies quota.
func EvaluateAccountQuota(quota *telemetry.AccountQuotaData, modelID string, modelLabel string, discoveredTokens ...string) ModelQuotaEvaluation {
	category := ResolveActiveModelCategory(modelID, modelLabel, discoveredTokens...)
	eval := ClassifyQuotaState(quota, category)
	eval.ModelID = modelID
	eval.ModelLabel = modelLabel
	return eval
}
