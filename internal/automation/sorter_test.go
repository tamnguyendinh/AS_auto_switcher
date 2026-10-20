package automation

import (
	"testing"

	"antigravity-switch/internal/telemetry"
)

func TestFilterEligibleCandidates_5Criteria(t *testing.T) {
	activeEmail := "active@example.com"
	failedEmails := map[string]bool{
		"failed_in_round@example.com": true,
	}

	candidates := []CandidateAccount{
		// 1. Active account -> Excluded by Criterion 1
		BuildCandidateAccount("active@example.com", 0, true, "refresh_tok", &telemetry.AccountQuotaData{
			Email: "active@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.80, WeeklyFraction: 0.50},
		}, CategoryGemini),

		// 2. Token invalid -> Excluded by Criterion 2
		BuildCandidateAccount("invalid_token@example.com", 1, false, "", &telemetry.AccountQuotaData{
			Email: "invalid_token@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.80, WeeklyFraction: 0.50},
		}, CategoryGemini),

		// 3. Failed in current round -> Excluded by Criterion 3
		BuildCandidateAccount("failed_in_round@example.com", 2, true, "refresh_tok", &telemetry.AccountQuotaData{
			Email: "failed_in_round@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.80, WeeklyFraction: 0.50},
		}, CategoryGemini),

		// 4. Low 5h quota (1% <= 2%) -> Excluded by Criterion 4
		BuildCandidateAccount("low_5h@example.com", 3, true, "refresh_tok", &telemetry.AccountQuotaData{
			Email: "low_5h@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.01, WeeklyFraction: 0.50},
		}, CategoryGemini),

		// 5. Depleted weekly quota (0%) -> Excluded by Criterion 4
		BuildCandidateAccount("depleted_weekly@example.com", 4, true, "refresh_tok", &telemetry.AccountQuotaData{
			Email: "depleted_weekly@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.80, WeeklyFraction: 0.00},
		}, CategoryGemini),

		// 6. Unknown / Unreliable Quota (telemetry error) -> Excluded by Criterion 5
		BuildCandidateAccount("error_quota@example.com", 5, true, "refresh_tok", &telemetry.AccountQuotaData{
			Email:        "error_quota@example.com",
			Status:       telemetry.StatusError,
			ErrorMessage: "timeout",
		}, CategoryGemini),

		// 7. Eligible Account -> Passed
		BuildCandidateAccount("eligible@example.com", 6, true, "refresh_tok", &telemetry.AccountQuotaData{
			Email: "eligible@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.80, WeeklyFraction: 0.50},
		}, CategoryGemini),
	}

	eligible, exclusions := FilterEligibleCandidates(candidates, activeEmail, failedEmails)

	if len(eligible) != 1 {
		t.Fatalf("expected exactly 1 eligible candidate, got %d", len(eligible))
	}
	if eligible[0].Email != "eligible@example.com" {
		t.Errorf("expected eligible email eligible@example.com, got %s", eligible[0].Email)
	}

	if len(exclusions) != 6 {
		t.Fatalf("expected 6 exclusions, got %d", len(exclusions))
	}

	exclusionMap := make(map[string]CandidateExclusion)
	for _, ex := range exclusions {
		exclusionMap[ex.Email] = ex
	}

	if ex, ok := exclusionMap["active@example.com"]; !ok || ex.Criterion != 1 {
		t.Errorf("expected active account excluded by criterion 1, got %+v", ex)
	}
	if ex, ok := exclusionMap["invalid_token@example.com"]; !ok || ex.Criterion != 2 {
		t.Errorf("expected invalid token excluded by criterion 2, got %+v", ex)
	}
	if ex, ok := exclusionMap["failed_in_round@example.com"]; !ok || ex.Criterion != 3 {
		t.Errorf("expected failed in round excluded by criterion 3, got %+v", ex)
	}
	if ex, ok := exclusionMap["low_5h@example.com"]; !ok || ex.Criterion != 4 {
		t.Errorf("expected low 5h excluded by criterion 4, got %+v", ex)
	}
	if ex, ok := exclusionMap["depleted_weekly@example.com"]; !ok || ex.Criterion != 4 {
		t.Errorf("expected depleted weekly excluded by criterion 4, got %+v", ex)
	}
	if ex, ok := exclusionMap["error_quota@example.com"]; !ok || ex.Criterion != 5 {
		t.Errorf("expected error quota excluded by criterion 5, got %+v", ex)
	}
}

func TestSortCandidatesBurnDown_TableAF_ArchitectureSpecification(t *testing.T) {
	// Exact Table A-F from Architecture Specification Section 6.2:
	// A: Tuần 3% (0.03), 5h 80% (0.80) -> Rank 1
	// B: Tuần 3% (0.03), 5h 40% (0.40) -> Rank 2
	// C: Tuần 3% (0.03), 5h Không áp dụng -> Rank 3
	// D: Tuần 20% (0.20), 5h 90% (0.90) -> Rank 4
	// E: Tuần 1% (0.01), 5h 1% (0.01) -> Bị loại (5h <= 2%)
	// F: Tuần 0% (0.00), 5h 100% (1.00) -> Bị loại (Tuần = 0%)

	candidates := []CandidateAccount{
		BuildCandidateAccount("A@example.com", 0, true, "tokA", &telemetry.AccountQuotaData{
			Email: "A@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.80, WeeklyFraction: 0.03},
		}, CategoryGemini),

		BuildCandidateAccount("B@example.com", 1, true, "tokB", &telemetry.AccountQuotaData{
			Email: "B@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.40, WeeklyFraction: 0.03},
		}, CategoryGemini),

		BuildCandidateAccount("C@example.com", 2, true, "tokC", &telemetry.AccountQuotaData{
			Email: "C@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: -1, FiveHourDisabled: true, WeeklyFraction: 0.03},
		}, CategoryGemini),

		BuildCandidateAccount("D@example.com", 3, true, "tokD", &telemetry.AccountQuotaData{
			Email: "D@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.90, WeeklyFraction: 0.20},
		}, CategoryGemini),

		BuildCandidateAccount("E@example.com", 4, true, "tokE", &telemetry.AccountQuotaData{
			Email: "E@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 0.01, WeeklyFraction: 0.01},
		}, CategoryGemini),

		BuildCandidateAccount("F@example.com", 5, true, "tokF", &telemetry.AccountQuotaData{
			Email: "F@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: 1.00, WeeklyFraction: 0.00},
		}, CategoryGemini),
	}

	// 1. Run Pre-filtering
	eligible, exclusions := FilterEligibleCandidates(candidates, "other_active@example.com", nil)

	if len(eligible) != 4 {
		t.Fatalf("expected 4 eligible candidates (A, B, C, D), got %d", len(eligible))
	}

	if len(exclusions) != 2 {
		t.Fatalf("expected 2 exclusions (E, F), got %d", len(exclusions))
	}

	// Confirm E and F are excluded
	exclusionMap := make(map[string]CandidateExclusion)
	for _, ex := range exclusions {
		exclusionMap[ex.Email] = ex
	}
	if _, ok := exclusionMap["E@example.com"]; !ok {
		t.Errorf("expected E to be excluded")
	}
	if _, ok := exclusionMap["F@example.com"]; !ok {
		t.Errorf("expected F to be excluded")
	}

	// 2. Run Burn-down Sorter on eligible candidates
	sorted := SortCandidatesBurnDown(eligible)

	expectedOrder := []string{
		"A@example.com",
		"B@example.com",
		"C@example.com",
		"D@example.com",
	}

	for i, exp := range expectedOrder {
		if sorted[i].Email != exp {
			t.Errorf("rank %d: expected %s, got %s", i+1, exp, sorted[i].Email)
		}
	}
}

func TestSortCandidatesBurnDown_Tier4TieBreak(t *testing.T) {
	// Two accounts with identical weekly and both lacking 5h:
	// Must preserve fixed index order
	candidates := []CandidateAccount{
		BuildCandidateAccount("second@example.com", 10, true, "tok2", &telemetry.AccountQuotaData{
			Email: "second@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: -1, WeeklyFraction: 0.50},
		}, CategoryGemini),
		BuildCandidateAccount("first@example.com", 5, true, "tok1", &telemetry.AccountQuotaData{
			Email: "first@example.com",
			Gemini: telemetry.ModelQuotaBucket{FiveHourFraction: -1, WeeklyFraction: 0.50},
		}, CategoryGemini),
	}

	sorted := SortCandidatesBurnDown(candidates)
	if sorted[0].Email != "first@example.com" {
		t.Errorf("expected lower index first@example.com to be ranked first, got %s", sorted[0].Email)
	}
	if sorted[1].Email != "second@example.com" {
		t.Errorf("expected higher index second@example.com to be ranked second, got %s", sorted[1].Email)
	}
}
