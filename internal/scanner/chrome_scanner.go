package scanner

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EmailRegex validates standard email syntax.
var EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$`)

// EmailExtractRegex scans raw strings for potential Google/Workspace email occurrences.
var EmailExtractRegex = regexp.MustCompile(`[a-zA-Z0-9_.+-]+@gmail\.com|[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+`)

// DefaultBlacklist defines mandatory excluded account keywords.
var DefaultBlacklist = []string{"anvien"}

// ChromeAccountInfo holds rich metadata for a signed-in Google Chrome account.
type ChromeAccountInfo struct {
	Email      string `json:"email"`
	Name       string `json:"name,omitempty"`
	ProfileDir string `json:"profile_dir,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
}

// GetDefaultChromeUserDataDir returns the standard Chrome User Data folder on Windows.
func GetDefaultChromeUserDataDir() string {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return ""
	}
	return filepath.Join(localAppData, "Google", "Chrome", "User Data")
}

// IsValidEmail verifies that the email is syntactically sound and not a known false positive.
func IsValidEmail(email string) bool {
	norm := strings.ToLower(strings.TrimSpace(email))
	if norm == "" {
		return false
	}

	// Filter out common false positives (assets, internal schemas, googleapis endpoints)
	forbiddenSuffixes := []string{
		".png", ".jpg", ".jpeg", ".js", ".css", ".json",
		".googleapis.com", "@googleapis.com",
		".chromium.org", "@chromium.org",
	}
	for _, suffix := range forbiddenSuffixes {
		if strings.HasSuffix(norm, suffix) {
			return false
		}
	}

	return EmailRegex.MatchString(norm)
}

// IsBlacklisted performs strict case-insensitive filtering against blacklisted terms.
func IsBlacklisted(email string, blacklist []string) bool {
	if len(blacklist) == 0 {
		blacklist = DefaultBlacklist
	}

	norm := strings.ToLower(strings.TrimSpace(email))
	for _, term := range blacklist {
		cleanTerm := strings.ToLower(strings.TrimSpace(term))
		if cleanTerm != "" && strings.Contains(norm, cleanTerm) {
			return true
		}
	}
	return false
}

// ExtractEmailsFromPreferences extracts valid Google account emails from a Chrome profile Preferences file.
func ExtractEmailsFromPreferences(prefPath string) []string {
	if prefPath == "" {
		return nil
	}

	contentBytes, err := os.ReadFile(prefPath)
	if err != nil {
		return nil
	}
	content := string(contentBytes)

	emailSet := make(map[string]struct{})

	// 1. Regex search on raw preferences text
	matches := EmailExtractRegex.FindAllString(content, -1)
	for _, match := range matches {
		norm := strings.ToLower(strings.TrimSpace(match))
		if IsValidEmail(norm) {
			emailSet[norm] = struct{}{}
		}
	}

	// 2. Structured JSON parsing for high precision
	var prefData map[string]interface{}
	if err := json.Unmarshal(contentBytes, &prefData); err == nil && prefData != nil {
		// Modern Chrome: account_info list of dicts
		if accInfoList, ok := prefData["account_info"].([]interface{}); ok {
			for _, item := range accInfoList {
				if accMap, ok := item.(map[string]interface{}); ok {
					if rawEmail, ok := accMap["email"].(string); ok {
						norm := strings.ToLower(strings.TrimSpace(rawEmail))
						if IsValidEmail(norm) {
							emailSet[norm] = struct{}{}
						}
					}
				}
			}
		}

		// Older Chrome: account_tracker_service_last_update dict
		if tracker, ok := prefData["account_tracker_service_last_update"].(map[string]interface{}); ok {
			for _, item := range tracker {
				if accMap, ok := item.(map[string]interface{}); ok {
					if rawEmail, ok := accMap["email"].(string); ok {
						norm := strings.ToLower(strings.TrimSpace(rawEmail))
						if IsValidEmail(norm) {
							emailSet[norm] = struct{}{}
						}
					}
				}
			}
		}
	}

	emails := make([]string, 0, len(emailSet))
	for email := range emailSet {
		emails = append(emails, email)
	}
	sort.Strings(emails)
	return emails
}

// ScanChromeAccountDetails scans all Chrome profiles and returns rich account details,
// filtering out any accounts matching the blacklist.
func ScanChromeAccountDetails(userDataDir string, blacklist []string) ([]ChromeAccountInfo, error) {
	targetDir := userDataDir
	if targetDir == "" {
		targetDir = GetDefaultChromeUserDataDir()
	}
	if targetDir == "" {
		return []ChromeAccountInfo{}, nil
	}

	stat, err := os.Stat(targetDir)
	if err != nil || !stat.IsDir() {
		return []ChromeAccountInfo{}, nil
	}

	accountsMap := make(map[string]*ChromeAccountInfo)
	knownProfileDirs := make(map[string]struct{})

	// 1. Inspect Local State
	localStatePath := filepath.Join(targetDir, "Local State")
	if lsBytes, err := os.ReadFile(localStatePath); err == nil {
		var lsData map[string]interface{}
		if err := json.Unmarshal(lsBytes, &lsData); err == nil && lsData != nil {
			if profSection, ok := lsData["profile"].(map[string]interface{}); ok {
				var infoCache map[string]interface{}
				if ic, ok := profSection["info_cache"].(map[string]interface{}); ok {
					infoCache = ic
				} else if pa, ok := profSection["profiles_attributes"].(map[string]interface{}); ok {
					infoCache = pa
				}

				for profDir, rawPData := range infoCache {
					knownProfileDirs[profDir] = struct{}{}
					pData, ok := rawPData.(map[string]interface{})
					if !ok {
						continue
					}

					userName, _ := pData["user_name"].(string)
					norm := strings.ToLower(strings.TrimSpace(userName))
					if !IsValidEmail(norm) || IsBlacklisted(norm, blacklist) {
						continue
					}

					name, _ := pData["gaia_name"].(string)
					if name == "" {
						name, _ = pData["name"].(string)
					}
					name = strings.TrimSpace(name)

					avatar, _ := pData["last_downloaded_gaia_picture_url_with_size"].(string)
					if avatar == "" {
						avatar, _ = pData["picture_url"].(string)
					}

					accountsMap[norm] = &ChromeAccountInfo{
						Email:      norm,
						Name:       name,
						ProfileDir: profDir,
						AvatarURL:  avatar,
					}
				}
			}
		}
	}

	// 2. Discover Profile directories
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		entries = []fs.DirEntry{}
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entryName := entry.Name()

		_, isKnown := knownProfileDirs[entryName]
		if entryName != "Default" && !strings.HasPrefix(entryName, "Profile") && !isKnown {
			continue
		}

		fullProfileDir := filepath.Join(targetDir, entryName)
		prefPath := filepath.Join(fullProfileDir, "Preferences")
		prefBytes, err := os.ReadFile(prefPath)
		if err != nil {
			continue
		}

		var prefData map[string]interface{}
		_ = json.Unmarshal(prefBytes, &prefData)

		if prefData != nil {
			// Modern Chrome: account_info list
			if accList, ok := prefData["account_info"].([]interface{}); ok {
				for _, item := range accList {
					accMap, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					rawEmail, _ := accMap["email"].(string)
					norm := strings.ToLower(strings.TrimSpace(rawEmail))
					if !IsValidEmail(norm) || IsBlacklisted(norm, blacklist) {
						continue
					}

					name, _ := accMap["full_name"].(string)
					if name == "" {
						name, _ = accMap["given_name"].(string)
					}
					name = strings.TrimSpace(name)

					pic, _ := accMap["picture_url"].(string)
					if pic == "" {
						pic, _ = accMap["last_downloaded_image_url_with_size"].(string)
					}

					if existing, exists := accountsMap[norm]; !exists {
						accountsMap[norm] = &ChromeAccountInfo{
							Email:      norm,
							Name:       name,
							ProfileDir: entryName,
							AvatarURL:  pic,
						}
					} else {
						if existing.Name == "" && name != "" {
							existing.Name = name
						}
						if existing.AvatarURL == "" && pic != "" {
							existing.AvatarURL = pic
						}
						if existing.ProfileDir == "" {
							existing.ProfileDir = entryName
						}
					}
				}
			}

			// Older Chrome: account_tracker_service_last_update
			if tracker, ok := prefData["account_tracker_service_last_update"].(map[string]interface{}); ok {
				for _, item := range tracker {
					accMap, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					rawEmail, _ := accMap["email"].(string)
					norm := strings.ToLower(strings.TrimSpace(rawEmail))
					if !IsValidEmail(norm) || IsBlacklisted(norm, blacklist) {
						continue
					}

					name, _ := accMap["full_name"].(string)
					if name == "" {
						name, _ = accMap["given_name"].(string)
					}
					name = strings.TrimSpace(name)

					pic, _ := accMap["picture_url"].(string)

					if existing, exists := accountsMap[norm]; !exists {
						accountsMap[norm] = &ChromeAccountInfo{
							Email:      norm,
							Name:       name,
							ProfileDir: entryName,
							AvatarURL:  pic,
						}
					} else {
						if existing.Name == "" && name != "" {
							existing.Name = name
						}
						if existing.AvatarURL == "" && pic != "" {
							existing.AvatarURL = pic
						}
						if existing.ProfileDir == "" {
							existing.ProfileDir = entryName
						}
					}
				}
			}
		}

		// Fallback regex search on Preferences content
		regexEmails := ExtractEmailsFromPreferences(prefPath)
		for _, email := range regexEmails {
			norm := strings.ToLower(strings.TrimSpace(email))
			if !IsValidEmail(norm) || IsBlacklisted(norm, blacklist) {
				continue
			}

			if existing, exists := accountsMap[norm]; !exists {
				accountsMap[norm] = &ChromeAccountInfo{
					Email:      norm,
					ProfileDir: entryName,
				}
			} else if existing.ProfileDir == "" {
				existing.ProfileDir = entryName
			}
		}
	}

	// Filter and convert to sorted slice
	result := make([]ChromeAccountInfo, 0, len(accountsMap))
	for _, acc := range accountsMap {
		if IsValidEmail(acc.Email) && !IsBlacklisted(acc.Email, blacklist) {
			result = append(result, *acc)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Email < result[j].Email
	})

	return result, nil
}

// ScanChromeProfiles scans Chrome profiles and returns a unique, sorted list of signed-in emails,
// strictly excluding any blacklisted emails.
func ScanChromeProfiles(userDataDir string, blacklist []string) ([]string, error) {
	details, err := ScanChromeAccountDetails(userDataDir, blacklist)
	if err != nil {
		return nil, err
	}

	emails := make([]string, 0, len(details))
	for _, acc := range details {
		emails = append(emails, acc.Email)
	}
	return emails, nil
}

// GetChromeSignedInAccounts returns all signed-in Google accounts using DefaultBlacklist.
func GetChromeSignedInAccounts(userDataDir string) ([]string, error) {
	return ScanChromeProfiles(userDataDir, DefaultBlacklist)
}
