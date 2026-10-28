package automation

import (
	"math"
	"sort"
	"strings"

	"antigravity-switch/internal/engines/antigravity/telemetry"
)

// CandidateAccount represents a candidate account evaluated for automatic switching.
type CandidateAccount struct {
	Email        string                      `json:"email"`
	Index        int                         `json:"index"`  // Original store order
	Status       string                      `json:"status"` // "on" hoặc "off"
	TokenValid   bool                        `json:"token_valid"`
	RefreshToken string                      `json:"-"`
	HasFreshPoll bool                        `json:"has_fresh_poll"`
	Evaluation   ModelQuotaEvaluation        `json:"evaluation"`
	RawQuota     *telemetry.AccountQuotaData `json:"raw_quota,omitempty"`
}

// CandidateExclusion records why an account was disqualified during pre-filtering.
type CandidateExclusion struct {
	Email     string `json:"email"`
	Criterion int    `json:"criterion"` // 1 to 6
	Reason    string `json:"reason"`
}

// FilterEligibleCandidates applies the pre-filtering criteria to exclude unqualified candidates.
//
// 6 Pre-filtering criteria:
// 1. Must be different from active account currently in use.
// 2. Token in accounts_store/ must be valid (can be loaded into WinCred with alive refresh token).
// 3. Not marked as failed in the current execution round.
// 4. Usage quota for the active model group must be above threshold:
//   - For accounts with 5h: 5h > 2% AND weekly > 0%
//   - For accounts with only weekly: weekly > 0%
//
// 5. Quota data must be reliable (not UNKNOWN_UNCERTAIN or error).
// 6. Account must be enabled (Status != "off").
func FilterEligibleCandidates(candidates []CandidateAccount, activeEmail string, failedEmails map[string]bool) (eligible []CandidateAccount, exclusions []CandidateExclusion) {
	for _, cand := range candidates {
		// Criterion 1: Must not be the active account
		if strings.EqualFold(cand.Email, activeEmail) {
			exclusions = append(exclusions, CandidateExclusion{
				Email:     cand.Email,
				Criterion: 1,
				Reason:    "active account currently in use",
			})
			continue
		}

		// Criterion 6: Bỏ qua tài khoản đang ở chế độ OFF
		if strings.EqualFold(cand.Status, "off") {
			exclusions = append(exclusions, CandidateExclusion{
				Email:     cand.Email,
				Criterion: 6,
				Reason:    "tài khoản đang ở chế độ vô hiệu hóa (OFF)",
			})
			continue
		}

		// Criterion 2: Valid token in accounts_store / WinCred with living refresh token
		if !cand.TokenValid {
			exclusions = append(exclusions, CandidateExclusion{
				Email:     cand.Email,
				Criterion: 2,
				Reason:    "token invalid or refresh token missing",
			})
			continue
		}

		// Criterion 3: Not marked failed in the current execution round
		if len(failedEmails) > 0 {
			isFailed := false
			for failed := range failedEmails {
				if strings.EqualFold(cand.Email, failed) {
					isFailed = true
					break
				}
			}
			if isFailed {
				exclusions = append(exclusions, CandidateExclusion{
					Email:     cand.Email,
					Criterion: 3,
					Reason:    "marked as failed in current round",
				})
				continue
			}
		}

		// Criterion 5 (checked before quota thresholds): Reliable quota data
		if cand.Evaluation.State == UnknownUncertain || cand.RawQuota == nil || cand.RawQuota.Status == telemetry.StatusError {
			exclusions = append(exclusions, CandidateExclusion{
				Email:     cand.Email,
				Criterion: 5,
				Reason:    "quota data not reliable or missing (UNKNOWN_UNCERTAIN)",
			})
			continue
		}

		// Criterion 4: Usage threshold for active model group
		if !cand.Evaluation.IsEligible {
			reason := cand.Evaluation.Reason
			if cand.Evaluation.State == ApplicableValid {
				if cand.Evaluation.FiveHourFraction <= Threshold5hLow {
					reason = "5h quota <= 2%"
				} else if cand.Evaluation.WeeklyFraction <= ThresholdWeeklyDepleted {
					reason = "weekly quota <= 1%"
				}
			} else if cand.Evaluation.State == NotApplicable {
				if cand.Evaluation.WeeklyFraction <= ThresholdWeeklyDepleted {
					reason = "weekly quota <= 1%"
				}
			}
			exclusions = append(exclusions, CandidateExclusion{
				Email:     cand.Email,
				Criterion: 4,
				Reason:    reason,
			})
			continue
		}

		eligible = append(eligible, cand)
	}

	return eligible, exclusions
}

// SortCandidatesBurnDown sorts eligible candidates using the 4-tier Burn-down strategy:
// Tier 1: Lowest remaining weekly quota first (Tuần thấp nhất trước: 3% -> 10% -> 50%...)
// Tier 2: Prefer accounts with applicable 5h quota over weekly-only (Ưu tiên tài khoản có 5h)
// Tier 3: Highest remaining 5h quota first (5h cao nhất trước: 80% > 40%)
// Tier 4: Stable fixed directory order (Giữ nguyên thứ tự danh bạ cố định)
func SortCandidatesBurnDown(candidates []CandidateAccount) []CandidateAccount {
	if len(candidates) <= 1 {
		return candidates
	}

	result := make([]CandidateAccount, len(candidates))
	copy(result, candidates)

	sort.SliceStable(result, func(i, j int) bool {
		c1 := result[i]
		c2 := result[j]

		// Tier 1: Lowest remaining weekly quota first (ascending)
		w1 := c1.Evaluation.WeeklyFraction
		w2 := c2.Evaluation.WeeklyFraction
		if math.Abs(w1-w2) > 1e-4 {
			return w1 < w2
		}

		// Tier 2: Prefer accounts with applicable 5h quota over weekly-only
		has5h1 := c1.Evaluation.State == ApplicableValid
		has5h2 := c2.Evaluation.State == ApplicableValid
		if has5h1 != has5h2 {
			return has5h1 // true before false
		}

		// Tier 3: Highest remaining 5h quota first (descending)
		if has5h1 && has5h2 {
			f1 := c1.Evaluation.FiveHourFraction
			f2 := c2.Evaluation.FiveHourFraction
			if math.Abs(f1-f2) > 1e-4 {
				return f1 > f2
			}
		}

		// Tier 4: Stable fixed directory order
		return c1.Index < c2.Index
	})

	return result
}

// BuildCandidateAccount creates a CandidateAccount from account details and evaluates its quota.
func BuildCandidateAccount(email string, index int, tokenValid bool, refreshToken string, quota *telemetry.AccountQuotaData, category QuotaCategory) CandidateAccount {
	eval := ClassifyQuotaState(quota, category)
	return CandidateAccount{
		Email:        email,
		Index:        index,
		Status:       "on",
		TokenValid:   tokenValid,
		RefreshToken: refreshToken,
		Evaluation:   eval,
		RawQuota:     quota,
	}
}
