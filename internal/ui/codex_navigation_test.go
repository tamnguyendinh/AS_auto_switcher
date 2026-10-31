package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"as-auto-switcher/internal/core/i18n"
	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/manager"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func newCodexTestApp(t *testing.T) *App {
	t.Helper()
	fa := test.NewApp()
	t.Cleanup(fa.Quit)
	a := newIsolatedApp(fa, createTestManager(t))
	t.Cleanup(a.Close)
	a.livenessInterval = 24 * time.Hour
	a.Init()
	waitForIdle(a, 2*time.Second)
	return a
}

func TestCodexNavigationPreservesAntigravity(t *testing.T) {
	a := newCodexTestApp(t)
	view, scroll, loop, controller := a.antigravityContent, a.accountScroll, a.autoLoop, a.controller
	a.accountListContainer.RemoveAll()
	for i := 0; i < 30; i++ {
		a.accountListContainer.Add(widget.NewLabel(fmt.Sprintf("Account %d", i)))
	}
	a.accountListContainer.Refresh()
	scroll.Refresh()
	scroll.ScrollToBottom()
	offset := scroll.Offset
	a.showStatusBar("Antigravity task in progress")
	for i := 0; i < 5; i++ {
		test.Tap(a.sidebar.codexItem)
		if !a.codex.content.Visible() || view.Visible() || !a.sidebar.codexItem.isActive || a.sidebar.antigravityItem.isActive {
			t.Fatal("clicking Codex did not switch body and selection together")
		}
		if a.statusBar.Visible() || a.serverStatusLabel.Visible() || !a.codexStatusLabel.Visible() {
			t.Fatal("Antigravity transient feedback leaked into Codex")
		}
		test.Tap(a.sidebar.antigravityItem)
		if !view.Visible() || a.codex.content.Visible() || !a.statusBar.Visible() {
			t.Fatal("returning to Antigravity did not restore view and pending feedback")
		}
	}
	if a.antigravityContent != view || a.accountScroll != scroll || scroll.Offset != offset || len(a.accountListContainer.Objects) != 30 {
		t.Fatalf("navigation changed content/scroll: view=%t scroll=%t offset=%v want=%v count=%d", a.antigravityContent == view, a.accountScroll == scroll, scroll.Offset, offset, len(a.accountListContainer.Objects))
	}
	if a.autoLoop != loop || !loop.IsRunning() || a.controller != controller {
		t.Fatal("navigation interrupted background service instances")
	}
	a.SwitchTab(NavTab(99))
	if a.sidebar.ActiveTab() != NavTabAntigravity {
		t.Fatal("unknown tab changed navigation")
	}
}

func TestCodexFooterAfterBackgroundMessageExpires(t *testing.T) {
	a := newCodexTestApp(t)
	test.Tap(a.sidebar.codexItem)
	a.showStatusBar("Background Antigravity refresh finished")
	a.clearStatusBar()

	if a.statusBar.Text != "" {
		t.Fatal("status expiry did not run")
	}
	if a.serverStatusLabel.Visible() || a.versionLabel.Visible() || a.statusBar.Visible() || !a.codexStatusLabel.Visible() || !a.codexVersionLabel.Visible() {
		t.Fatal("timeout restored footer labels for the wrong tab")
	}
	test.Tap(a.sidebar.antigravityItem)
	if !a.serverStatusLabel.Visible() || !a.versionLabel.Visible() || a.codexStatusLabel.Visible() || a.statusBar.Visible() {
		t.Fatal("returning after timeout did not restore normal Antigravity footer")
	}
}

func TestCodexPlaceholderLanguageThemeAndSharedPin(t *testing.T) {
	a := newCodexTestApp(t)
	test.Tap(a.sidebar.codexItem)
	if !a.codex.add.Disabled() || !a.codex.refresh.Disabled() {
		t.Fatal("unimplemented account actions must be disabled")
	}
	for _, locale := range []string{"en", "vi"} {
		a.SetLanguage(locale)
		d := i18n.GetDictionary(i18n.Locale(locale))
		if a.codex.add.Text != d.CodexAddAccount || a.codex.description.Text != d.CodexDescription || a.codexStatusLabel.Text != d.CodexStatus || a.codexVersionLabel.Text != d.CodexVersion {
			t.Fatalf("Codex content/footer did not translate to %s", locale)
		}
		for _, name := range []string{ThemeDark, ThemeLight, ThemeWarm} {
			a.SetTheme(name)
			palette := GetThemePalette(name)
			if a.codex.background.FillColor != palette.Card || a.codex.background.StrokeColor != palette.BorderSubtle || a.sidebar.codexItem.palette != palette {
				t.Fatalf("Codex theme mismatch: %s", name)
			}
		}
	}
	before := a.alwaysOnTop
	test.Tap(a.codex.pin)
	if a.alwaysOnTop == before || a.codex.pin.Importance != a.alwaysOnTopBtn.Importance {
		t.Fatal("Codex pin did not update shared window state")
	}
	test.Tap(a.sidebar.antigravityItem)
	// The headless Fyne driver executes fyne.Do synchronously on the caller,
	// unlike native Fyne's serialized UI loop. Await the tab's sync callback.
	waitForIdle(a, 2*time.Second)
	test.Tap(a.alwaysOnTopBtn)
	if a.alwaysOnTop != before || a.codex.pin.Importance != a.alwaysOnTopBtn.Importance {
		t.Fatal("Antigravity pin did not synchronize Codex")
	}
}

func TestCodexTabKeepsLivenessRunning(t *testing.T) {
	fa := test.NewApp()
	defer fa.Quit()
	a := newIsolatedApp(fa, createTestManager(t))
	defer a.Close()
	a.mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, connector.ErrLanguageServerNotFound })
	var calls atomic.Int32
	a.livenessInterval = 10 * time.Millisecond
	a.SetServerDiscoverer(func() (*connector.ServerInfo, error) {
		calls.Add(1)
		return nil, connector.ErrLanguageServerNotFound
	})
	a.overview = &manager.AccountOverview{}
	a.Init()
	waitForIdle(a, 2*time.Second)
	test.Tap(a.sidebar.codexItem)
	before := calls.Load()
	deadline := time.Now().Add(time.Second)
	for calls.Load() < before+3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if calls.Load() < before+3 {
		t.Fatal("liveness watcher stopped while Codex was visible")
	}
	if a.serverStatusLabel.Visible() || !a.codexStatusLabel.Visible() {
		t.Fatal("liveness update changed contextual footer visibility")
	}
}

func TestCodexSkeletonMirror(t *testing.T) {
	t.Skip("skipping skeleton mirror test: superseded by real codex architecture")
	source := filepath.Join("..", "engines", "antigravity")
	target := filepath.Join("..", "engines", "codex")
	count := 0
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(filepath.Join(target, rel))
		if err != nil {
			return err
		}
		want := "package " + filepath.Base(filepath.Dir(rel)) + "\n"
		if string(body) != want {
			return fmt.Errorf("%s must contain only %q", rel, want)
		}
		count++
		return nil
	})
	if err != nil || count != 53 {
		t.Fatalf("mirror: count=%d error=%v", count, err)
	}
	targetCount := 0
	err = filepath.WalkDir(target, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(path, ".go") {
			targetCount++
		}
		return err
	})
	if err != nil || targetCount != count {
		t.Fatalf("extra/missing skeleton files: %d, error=%v", targetCount, err)
	}
}
