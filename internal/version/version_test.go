package version

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func TestVersion_Default(t *testing.T) {
	v := GetVersion()
	matched, _ := regexp.MatchString(`^[0-9]+(\.[0-9]+){2,3}$`, v)
	if !matched {
		t.Errorf("expected Version to match quad-part/semver pattern, got: %s", v)
	}
	full := GetFullVersion()
	expectedFull := "Version: " + v
	if full != expectedFull {
		t.Errorf("expected full version '%s', got: %s", expectedFull, full)
	}
	buildInfo := GetBuildInfo()
	expectedBuild := "Version " + v
	if buildInfo != expectedBuild {
		t.Errorf("expected default build info '%s', got: %s", expectedBuild, buildInfo)
	}
}

func TestVersion_WithGitCommit(t *testing.T) {
	oldCommit := GitCommit
	defer func() { GitCommit = oldCommit }()

	GitCommit = "3dd7d69"
	expected := "Version: " + GetVersion()
	if GetFullVersion() != expected {
		t.Errorf("expected clean full version '%s', got '%s'", expected, GetFullVersion())
	}
	if strings.Contains(GetFullVersion(), "rev") || strings.Contains(GetFullVersion(), "3dd7d69") {
		t.Errorf("GetFullVersion() must not contain commit hash or rev, got: %s", GetFullVersion())
	}

	expectedBuildInfo := fmt.Sprintf("Version %s (commit 3dd7d69)", GetVersion())
	if GetBuildInfo() != expectedBuildInfo {
		t.Errorf("expected build info '%s', got '%s'", expectedBuildInfo, GetBuildInfo())
	}
}

func TestVersion_BuildInfo(t *testing.T) {
	oldCommit := GitCommit
	defer func() { GitCommit = oldCommit }()

	GitCommit = ""
	if GetBuildInfo() != "Version "+GetVersion() {
		t.Errorf("expected 'Version %s', got '%s'", GetVersion(), GetBuildInfo())
	}

	GitCommit = "a77efb9"
	expected := fmt.Sprintf("Version %s (commit a77efb9)", GetVersion())
	if GetBuildInfo() != expected {
		t.Errorf("expected '%s', got '%s'", expected, GetBuildInfo())
	}
}

func TestVersion_ReadmeSync(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to locate repo root: %v", err)
	}

	readmePath := filepath.Join(root, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	re := regexp.MustCompile(`\*\*Version\s+([0-9]+(?:\.[0-9]+){2,3})\*\*`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) < 2 {
		t.Fatalf("failed to find version pattern in README.md header")
	}

	readmeVersion := matches[1]
	appVersion := GetVersion()

	if readmeVersion != appVersion {
		t.Fatalf("README.md header version (%s) is out of sync with internal/version.Version (%s). Run '.\\scripts\\build_windows.ps1' or '.\\scripts\\build_windows.ps1 -SyncOnly' to synchronize.", readmeVersion, appVersion)
	}
}
