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
	if !strings.HasPrefix(full, "Version: 1.0.2.0") {
		t.Errorf("expected full version to start with 'Version: 1.0.2.0', got: %s", full)
	}
}

func TestVersion_WithGitCommit(t *testing.T) {
	oldCommit := GitCommit
	defer func() { GitCommit = oldCommit }()

	GitCommit = "a77efb9"
	expected := "Version: 1.0.2.0 (rev a77efb9)"
	if GetFullVersion() != expected {
		t.Errorf("expected '%s', got '%s'", expected, GetFullVersion())
	}
}
