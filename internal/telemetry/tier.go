package telemetry

import "strings"

// NormalizeTier normalizes various raw tier representations from Google Cloud Code,
// Language Server, and internal models into a canonical 5-character maximum code:
// ULTRA, PRO, BIZ, FREE, or "--".
func NormalizeTier(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(trimmed, "ultra") || strings.Contains(trimmed, "ultimate"):
		return "ULTRA"
	case strings.Contains(trimmed, "pro"):
		return "PRO"
	case strings.Contains(trimmed, "enterprise") || strings.Contains(trimmed, "standard") || strings.Contains(trimmed, "team") || strings.Contains(trimmed, "biz") || strings.Contains(trimmed, "business"):
		return "BIZ"
	case strings.Contains(trimmed, "free"):
		return "FREE"
	default:
		return "--"
	}
}
