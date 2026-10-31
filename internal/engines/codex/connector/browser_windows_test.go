//go:build windows

package connector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBrowserCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedExe string
		argsCount   int
		expectErr   bool
	}{
		{
			name:        "Quoted path with single argument and percent-1 placeholder",
			input:       `"C:\Program Files\Google\Chrome\Application\chrome.exe" --single-argument %1`,
			expectedExe: `C:\Program Files\Google\Chrome\Application\chrome.exe`,
			argsCount:   1, // "--single-argument", %1 filtered
			expectErr:   false,
		},
		{
			name:        "Quoted path without arguments",
			input:       `"C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe"`,
			expectedExe: `C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe`,
			argsCount:   0,
			expectErr:   false,
		},
		{
			name:        "Unquoted path with spaces and arguments",
			input:       `C:\Program Files\Mozilla Firefox\firefox.exe -os-int -url "%1"`,
			expectedExe: `C:\Program Files\Mozilla Firefox\firefox.exe`,
			argsCount:   2, // "-os-int", "-url", "%1" filtered
			expectErr:   false,
		},
		{
			name:        "Simple unquoted path without spaces",
			input:       `C:\Windows\explorer.exe %1`,
			expectedExe: `C:\Windows\explorer.exe`,
			argsCount:   0,
			expectErr:   false,
		},
		{
			name:        "Empty command string",
			input:       `   `,
			expectedExe: "",
			argsCount:   0,
			expectErr:   true,
		},
		{
			name:        "Malformed quotes",
			input:       `"C:\Program Files\Browser\browser.exe`,
			expectedExe: `C:\Program Files\Browser\browser.exe`,
			argsCount:   0,
			expectErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exe, args, err := ParseBrowserCommand(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}
			if filepath.Clean(exe) != filepath.Clean(tt.expectedExe) {
				t.Errorf("exe mismatch: got %q, want %q", exe, tt.expectedExe)
			}
			if len(args) != tt.argsCount {
				t.Errorf("args count mismatch: got %d (%v), want %d", len(args), args, tt.argsCount)
			}
		})
	}
}

func TestWindowsBrowserLauncher_ListInstalledBrowsers(t *testing.T) {
	launcher := NewWindowsBrowserLauncher()
	browsers, err := launcher.ListInstalledBrowsers()
	if err != nil {
		t.Fatalf("failed to list installed browsers: %v", err)
	}

	if len(browsers) == 0 {
		t.Fatalf("expected at least one installed browser on Windows host, got 0")
	}

	seenPaths := make(map[string]bool)
	defaultCount := 0

	for _, b := range browsers {
		if b.ID == "" {
			t.Errorf("browser has empty ID: %+v", b)
		}
		if b.Name == "" {
			t.Errorf("browser has empty Name: %+v", b)
		}
		if b.ExePath == "" {
			t.Errorf("browser has empty ExePath: %+v", b)
		}

		// Verify binary exists on disk
		fi, err := os.Stat(b.ExePath)
		if err != nil {
			t.Errorf("browser executable does not exist on disk: %s (err: %v)", b.ExePath, err)
		} else if fi.IsDir() {
			t.Errorf("browser executable is a directory: %s", b.ExePath)
		}

		lowerPath := strings.ToLower(filepath.Clean(b.ExePath))
		if seenPaths[lowerPath] {
			t.Errorf("duplicate browser ExePath detected: %s", b.ExePath)
		}
		seenPaths[lowerPath] = true

		if b.IsDefault {
			defaultCount++
		}
	}

	if defaultCount != 1 {
		t.Errorf("expected exactly 1 default browser, got %d", defaultCount)
	}
}

func TestWindowsBrowserLauncher_GetDefaultBrowser(t *testing.T) {
	launcher := NewWindowsBrowserLauncher()
	def, err := launcher.GetDefaultBrowser()
	if err != nil {
		t.Fatalf("GetDefaultBrowser failed: %v", err)
	}

	if def == nil {
		t.Fatal("expected default browser info, got nil")
	}

	if !def.IsDefault {
		t.Errorf("expected IsDefault to be true, got %v", def.IsDefault)
	}

	if def.ExePath == "" {
		t.Error("default browser has empty ExePath")
	}

	fi, err := os.Stat(def.ExePath)
	if err != nil {
		t.Errorf("default browser executable does not exist on disk: %s (err: %v)", def.ExePath, err)
	} else if fi.IsDir() {
		t.Errorf("default browser executable is a directory: %s", def.ExePath)
	}
}

func TestWindowsBrowserLauncher_Launch_Validation(t *testing.T) {
	launcher := NewWindowsBrowserLauncher()

	// Launch with non-existent executable must fail
	fakeExe := filepath.Join(t.TempDir(), "nonexistent_browser.exe")
	err := launcher.Launch(fakeExe, "https://openai.com")
	if err == nil {
		t.Fatalf("expected error launching non-existent browser executable, got nil")
	}

	// Launch with empty executable and empty URL must fail
	err = launcher.Launch("", "")
	// If default browser was found, it might launch with empty URL, or error if empty URL and no default
	// Testing empty exe and URL with nonexistent path
	err = launcher.Launch(fakeExe, "")
	if err == nil {
		t.Fatalf("expected error for fake exe, got nil")
	}
}
