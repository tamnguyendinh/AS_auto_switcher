package version

import (
	"strings"
	"testing"
)

func TestVersion_Default(t *testing.T) {
	if GetVersion() != "1.0.2.0" {
		t.Errorf("expected default Version '1.0.2.0', got: %s", GetVersion())
	}
	full := GetFullVersion()
	if full != "Version: 1.0.2.0" {
		t.Errorf("expected full version 'Version: 1.0.2.0', got: %s", full)
	}
	buildInfo := GetBuildInfo()
	if buildInfo != "Version 1.0.2.0" {
		t.Errorf("expected default build info 'Version 1.0.2.0', got: %s", buildInfo)
	}
}

func TestVersion_WithGitCommit(t *testing.T) {
	oldCommit := GitCommit
	defer func() { GitCommit = oldCommit }()

	GitCommit = "3dd7d69"
	expected := "Version: 1.0.2.0"
	if GetFullVersion() != expected {
		t.Errorf("expected clean full version '%s', got '%s'", expected, GetFullVersion())
	}
	if strings.Contains(GetFullVersion(), "rev") || strings.Contains(GetFullVersion(), "3dd7d69") {
		t.Errorf("GetFullVersion() must not contain commit hash or rev, got: %s", GetFullVersion())
	}

	expectedBuildInfo := "Version 1.0.2.0 (commit 3dd7d69)"
	if GetBuildInfo() != expectedBuildInfo {
		t.Errorf("expected build info '%s', got '%s'", expectedBuildInfo, GetBuildInfo())
	}
}

func TestVersion_BuildInfo(t *testing.T) {
	oldCommit := GitCommit
	defer func() { GitCommit = oldCommit }()

	GitCommit = ""
	if GetBuildInfo() != "Version 1.0.2.0" {
		t.Errorf("expected 'Version 1.0.2.0', got '%s'", GetBuildInfo())
	}

	GitCommit = "a77efb9"
	if GetBuildInfo() != "Version 1.0.2.0 (commit a77efb9)" {
		t.Errorf("expected 'Version 1.0.2.0 (commit a77efb9)', got '%s'", GetBuildInfo())
	}
}
