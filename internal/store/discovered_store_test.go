package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestDiscoveredStoreRepeatReloadAndPartialRead(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "discovered_accounts")
	s, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.SaveDiscovered(" USER@example.com "); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	raw, err := os.ReadFile(filepath.Join(dir, "user@example.com.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatal(err)
	}
	if len(record) != 2 || record["email"] != "user@example.com" || record["scanned_at"].(float64) <= 0 {
		t.Fatalf("unexpected catalog schema: %v", record)
	}
	reopened, err := NewDiscoveredStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		emails, err := reopened.ListDiscovered()
		if err != nil || !reflect.DeepEqual(emails, []string{"user@example.com"}) {
			t.Fatalf("%v %v", emails, err)
		}
	}
	after, _ := os.ReadFile(filepath.Join(dir, "user@example.com.json"))
	if string(raw) != string(after) {
		t.Fatal("reading changed scanned_at")
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	emails, err := s.ListDiscovered()
	if err == nil || len(emails) != 1 {
		t.Fatalf("partial read lost valid record or hid error: %v %v", emails, err)
	}
}

func TestDiscoveredStoreUnavailableAndFailedReplacement(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "catalog")
	if err := os.WriteFile(dir, []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	s, err := NewDiscoveredStore(dir)
	if s == nil || err == nil {
		t.Fatal("expected usable path handle with visible I/O error")
	}
	if err := s.SaveDiscovered("user@example.com"); err == nil {
		t.Fatal("write should fail")
	}
	if _, err := s.ListDiscovered(); err == nil {
		t.Fatal("read should fail")
	}
	dir = filepath.Join(root, "working")
	s, err = NewDiscoveredStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "user@example.com.json")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "keep")
	os.WriteFile(marker, []byte("preserve"), 0600)
	if err := s.SaveDiscovered("user@example.com"); err == nil {
		t.Fatal("replacement should fail")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("failed replacement destroyed target")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("temporary file leaked: %v", entries)
	}
}

func TestResolveStoreLocationsAndInvalidOverride(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	unrelated := filepath.Join(root, "cwd")
	for _, p := range []string{repo, unrelated} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module fixture"), 0600)
	repoStore := filepath.Join(repo, "accounts_store")
	if got := resolveStoreDir("", unrelated, filepath.Join(repo, "bin", "app.exe")); got != repoStore {
		t.Fatalf("repo executable: %s", got)
	}
	if got := resolveStoreDir("", filepath.Join(repo, "internal", "store"), filepath.Join(root, "go-build", "app.exe")); got != repoStore {
		t.Fatalf("go run: %s", got)
	}
	portable := filepath.Join(root, "portable")
	if got := resolveStoreDir("", unrelated, filepath.Join(portable, "app.exe")); got != filepath.Join(portable, "accounts_store") {
		t.Fatalf("portable: %s", got)
	}
	os.MkdirAll(filepath.Join(portable, "accounts_store"), 0755)
	if got := resolveStoreDir("", unrelated, filepath.Join(portable, "bin", "app.exe")); got != filepath.Join(portable, "accounts_store") {
		t.Fatalf("legacy portable: %s", got)
	}
	override := filepath.Join(root, "custom", "vault")
	t.Setenv("ANTIGRAVITY_STORE_DIR", override)
	s, err := NewAccountStore("")
	if err != nil || s.GetStoreDir() != override {
		t.Fatalf("new override: %v %v", s, err)
	}
	occupied := filepath.Join(root, "occupied")
	os.WriteFile(occupied, []byte("keep"), 0600)
	t.Setenv("ANTIGRAVITY_STORE_DIR", occupied)
	if _, err := NewAccountStore(""); err == nil {
		t.Fatal("invalid override silently accepted/fell back")
	}
	if got := resolveStoreDir("", "", ""); got != "" {
		t.Fatalf("unexpected cwd fallback: %s", got)
	}
}
