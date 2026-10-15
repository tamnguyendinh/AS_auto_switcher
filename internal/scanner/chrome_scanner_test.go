package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIsValidEmail(t *testing.T) {
	testCases := []struct {
		email    string
		expected bool
	}{
		{"user@gmail.com", true},
		{"first.last+tag@company.co.uk", true},
		{"valid_email123@domain.org", true},
		{"", false},
		{"   ", false},
		{"not-an-email", false},
		{"@missinguser.com", false},
		{"missingdomain@", false},
		{"avatar.png", false},
		{"user@googleapis.com", false},
		{"dev@chromium.org", false},
		{"script.js", false},
		{"data.json", false},
	}

	for _, tc := range testCases {
		actual := IsValidEmail(tc.email)
		if actual != tc.expected {
			t.Errorf("IsValidEmail(%q) = %v; expected %v", tc.email, actual, tc.expected)
		}
	}
}

func TestIsBlacklisted(t *testing.T) {
	testCases := []struct {
		email     string
		blacklist []string
		expected  bool
	}{
		{"anvien@gmail.com", nil, true},
		{"ANVIEN@gmail.com", nil, true},
		{"User.AnVien.2026@domain.com", nil, true},
		{"prefix-anvien-suffix@gmail.com", []string{"anvien"}, true},
		{"john.doe@gmail.com", nil, false},
		{"developer@company.com", []string{"anvien"}, false},
		{"corp@restricted.com", []string{"restricted"}, true},
		{"clean@example.com", []string{"restricted"}, false},
	}

	for _, tc := range testCases {
		actual := IsBlacklisted(tc.email, tc.blacklist)
		if actual != tc.expected {
			t.Errorf("IsBlacklisted(%q, %v) = %v; expected %v", tc.email, tc.blacklist, actual, tc.expected)
		}
	}
}

func TestExtractEmailsFromPreferences(t *testing.T) {
	tmpDir := t.TempDir()
	prefPath := filepath.Join(tmpDir, "Preferences")

	prefContent := map[string]interface{}{
		"account_info": []map[string]interface{}{
			{"email": "modern1@gmail.com", "full_name": "Modern User 1"},
			{"email": "shared@gmail.com", "full_name": "Shared User"},
		},
		"account_tracker_service_last_update": map[string]interface{}{
			"acc1": map[string]interface{}{"email": "tracker1@gmail.com", "full_name": "Tracker User"},
			"acc2": map[string]interface{}{"email": "shared@gmail.com", "full_name": "Shared User"},
		},
	}

	bytes, err := json.Marshal(prefContent)
	if err != nil {
		t.Fatalf("Failed to marshal prefContent: %v", err)
	}

	if err := os.WriteFile(prefPath, bytes, 0600); err != nil {
		t.Fatalf("Failed to write prefPath: %v", err)
	}

	emails := ExtractEmailsFromPreferences(prefPath)
	expected := []string{"modern1@gmail.com", "shared@gmail.com", "tracker1@gmail.com"}
	if !reflect.DeepEqual(emails, expected) {
		t.Errorf("ExtractEmailsFromPreferences() = %v; expected %v", emails, expected)
	}

	// Test non-existent file returns nil without panicking
	missingEmails := ExtractEmailsFromPreferences(filepath.Join(tmpDir, "non_existent"))
	if missingEmails != nil {
		t.Errorf("Expected nil for missing file, got %v", missingEmails)
	}
}

func TestScanChromeAccountDetailsAndBlacklist(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create Local State
	localStateData := map[string]interface{}{
		"profile": map[string]interface{}{
			"info_cache": map[string]interface{}{
				"Default": map[string]interface{}{
					"user_name": "alpha@gmail.com",
					"gaia_name": "Alpha User",
					"last_downloaded_gaia_picture_url_with_size": "https://avatar.google.com/alpha.png",
				},
				"Profile 1": map[string]interface{}{
					"user_name": "beta.anvien@gmail.com", // Blacklisted!
					"gaia_name": "Beta Anvien",
				},
				"Profile 2": map[string]interface{}{
					"user_name": "gamma@gmail.com",
					"name":      "Gamma User",
					"picture_url": "https://avatar.google.com/gamma.png",
				},
			},
		},
	}
	lsBytes, _ := json.Marshal(localStateData)
	_ = os.WriteFile(filepath.Join(tmpDir, "Local State"), lsBytes, 0600)

	// 2. Create Default/Preferences
	defaultDir := filepath.Join(tmpDir, "Default")
	_ = os.MkdirAll(defaultDir, 0700)
	defaultPref := map[string]interface{}{
		"account_info": []map[string]interface{}{
			{
				"email":     "alpha@gmail.com",
				"full_name": "Alpha User Updated",
			},
			{
				"email":     "delta@gmail.com",
				"full_name": "Delta User",
			},
			{
				"email":     "blocked.ANVIEN.test@domain.com", // Blacklisted!
				"full_name": "Blocked Account",
			},
		},
	}
	defBytes, _ := json.Marshal(defaultPref)
	_ = os.WriteFile(filepath.Join(defaultDir, "Preferences"), defBytes, 0600)

	// 3. Create Profile 2/Preferences (Regex fallback test)
	prof2Dir := filepath.Join(tmpDir, "Profile 2")
	_ = os.MkdirAll(prof2Dir, 0700)
	rawTextPref := `{"unrelated_config": true, "raw_note": "contact epsilon@gmail.com for help"}`
	_ = os.WriteFile(filepath.Join(prof2Dir, "Preferences"), []byte(rawTextPref), 0600)

	// 4. Create an ignored directory
	ignoredDir := filepath.Join(tmpDir, "Crashpad")
	_ = os.MkdirAll(ignoredDir, 0700)

	// Run scanner
	details, err := ScanChromeAccountDetails(tmpDir, []string{"anvien"})
	if err != nil {
		t.Fatalf("ScanChromeAccountDetails failed: %v", err)
	}

	// Verify blacklisted accounts were completely excluded
	for _, acc := range details {
		if IsBlacklisted(acc.Email, []string{"anvien"}) {
			t.Errorf("Blacklisted account found in output: %s", acc.Email)
		}
	}

	// Verify expected accounts
	expectedEmails := []string{"alpha@gmail.com", "delta@gmail.com", "epsilon@gmail.com", "gamma@gmail.com"}
	var actualEmails []string
	for _, acc := range details {
		actualEmails = append(actualEmails, acc.Email)
	}

	if !reflect.DeepEqual(actualEmails, expectedEmails) {
		t.Fatalf("Expected emails %v, got %v", expectedEmails, actualEmails)
	}

	// Verify ScanChromeProfiles helper returns same email list
	profiles, err := ScanChromeProfiles(tmpDir, []string{"anvien"})
	if err != nil {
		t.Fatalf("ScanChromeProfiles failed: %v", err)
	}
	if !reflect.DeepEqual(profiles, expectedEmails) {
		t.Fatalf("ScanChromeProfiles = %v; expected %v", profiles, expectedEmails)
	}

	// Verify alpha account metadata was merged properly
	var alphaAcc *ChromeAccountInfo
	for i := range details {
		if details[i].Email == "alpha@gmail.com" {
			alphaAcc = &details[i]
			break
		}
	}
	if alphaAcc == nil {
		t.Fatalf("alpha@gmail.com not found in details")
	}
	if alphaAcc.Name != "Alpha User" {
		t.Errorf("alpha Name = %q; expected 'Alpha User'", alphaAcc.Name)
	}
	if alphaAcc.AvatarURL != "https://avatar.google.com/alpha.png" {
		t.Errorf("alpha AvatarURL = %q; expected 'https://avatar.google.com/alpha.png'", alphaAcc.AvatarURL)
	}
}

func TestLiveChromeDiscovery(t *testing.T) {
	// Proactively test against local Chrome User Data if present
	defaultDir := GetDefaultChromeUserDataDir()
	if defaultDir == "" {
		t.Skip("LOCALAPPDATA not set; skipping live test")
	}

	if _, err := os.Stat(defaultDir); os.IsNotExist(err) {
		t.Skip("Chrome User Data does not exist on this machine; skipping live test")
	}

	accounts, err := GetChromeSignedInAccounts(defaultDir)
	if err != nil {
		t.Fatalf("GetChromeSignedInAccounts failed on live directory: %v", err)
	}

	for _, email := range accounts {
		if IsBlacklisted(email, DefaultBlacklist) {
			t.Errorf("Live scan returned blacklisted email: %s", email)
		}
		if !IsValidEmail(email) {
			t.Errorf("Live scan returned invalid email: %s", email)
		}
	}
}
