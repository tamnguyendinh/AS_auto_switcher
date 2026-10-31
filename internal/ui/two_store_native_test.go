//go:build windows && gui && nativeui

package ui

import (
	"context"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"as-auto-switcher/internal/engines/antigravity/connector"
	"as-auto-switcher/internal/engines/antigravity/manager"
	"as-auto-switcher/internal/engines/antigravity/model"
	"as-auto-switcher/internal/engines/antigravity/store"
	"as-auto-switcher/internal/engines/antigravity/wincred"
	"fyne.io/fyne/v2"
)

func TestNativeTwoStoreRefreshAndResize(t *testing.T) {
	a := nativeWindowApp
	root := t.TempDir()
	vault, err := store.NewAccountStore(filepath.Join(root, "accounts_store"))
	if err != nil {
		t.Fatal(err)
	}
	mgr, err := manager.NewAccountManager(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	mgr.SetClientFinder(func(context.Context) (*connector.Client, error) { return nil, errors.New("fixture offline") })
	mgr.SetWincredHandlers(func() (*model.TokenData, error) {
		return &model.TokenData{AccessToken: "fixture", IDToken: makeTestJWT("active@example.com")}, nil
	}, nil)
	profile := filepath.Join(root, "chrome", "Default")
	os.MkdirAll(profile, 0755)
	os.WriteFile(filepath.Join(profile, "Preferences"), []byte(`{"account_info":[{"email":"needs-oauth@example.com"}]}`), 0600)
	mgr.SetChromeDir(filepath.Dir(profile))
	if err := vault.SaveAccount("ready@example.com", &model.TokenData{RefreshToken: "fixture"}); err != nil {
		t.Fatal(err)
	}
	original := readNativeResizeRect()
	fyne.DoAndWait(func() { a.mgr = mgr; a.clearStatusBar() })
	t.Cleanup(func() {
		fyne.DoAndWait(func() {
			a.mgr = nil
			a.overview = nil
			a.accountCards = nil
			a.accountListContainer.RemoveAll()
			a.clearStatusBar()
			applyNativeResizeRect(original)
		})
	})
	clickNativeControl(t, a.refreshBtn)
	waitNativeResize(t, func() bool { return !a.isBusy && len(a.accountCards) == 3 })
	var before *AccountCardWidget
	fyne.DoAndWait(func() {
		if !a.accountCards[0].IsActive || findSwitchButton(a.accountCards[0].Root) != nil {
			t.Error("active action")
		}
		if findSwitchButton(a.accountCards[1].Root) == nil {
			t.Error("ready action")
		}
		if findSwitchButton(a.accountCards[2].Root) != nil || a.accountCards[2].gemini5hPct.Text != "N/A" {
			t.Error("unauth action/quota")
		}
		before = a.accountCards[1]
	})
	for i := 0; i < 3; i++ {
		clickNativeControl(t, a.refreshBtn)
		waitNativeResize(t, func() bool { return !a.isBusy && len(a.accountCards) == 3 })
	}
	fyne.DoAndWait(func() {
		if a.accountCards[1] != before {
			t.Error("repeat refresh replaced card")
		}
	})
	if err := vault.DeleteAccount("ready@example.com"); err != nil {
		t.Fatal(err)
	}
	// Remember this identity before removal from credential store, as a Chrome scan would.
	catalog := mgr.DiscoveredStore()
	catalog.SaveDiscovered("ready@example.com")
	clickNativeControl(t, a.refreshBtn)
	waitNativeResize(t, func() bool {
		if a.isBusy {
			return false
		}
		for _, c := range a.accountCards {
			if c.Email == "ready@example.com" {
				return !c.HasToken && findSwitchButton(c.Root) == nil && c.gemini5hPct.Text == "N/A"
			}
		}
		return false
	})
	if err := vault.SaveAccount("ready@example.com", &model.TokenData{RefreshToken: "fixture"}); err != nil {
		t.Fatal(err)
	}
	clickNativeControl(t, a.refreshBtn)
	waitNativeResize(t, func() bool { return !a.isBusy && len(a.accountCards) == 3 && a.accountCards[1].HasToken })
	var widthBefore float32
	fyne.DoAndWait(func() {
		widthBefore = a.win.Content().MinSize().Width
		a.showStatusBar(strings.Repeat("long-path-or-warning/", 50))
	})
	fyne.DoAndWait(func() {
		if a.win.Content().MinSize().Width > widthBefore {
			t.Error("long warning locked window minimum width")
		}
	})
	// Use the real native resize grips after rendering all three states.
	_, after := dragNativeResizeGrip(t, resizeRight|resizeBottom, 70, 35)
	if after == original {
		t.Fatal("native resize did not move")
	}
	assertNativeResizeCanvas(t)
	fyne.DoAndWait(func() { a.clearStatusBar(); applyNativeResizeRect(original) })
	waitNativeResize(t, func() bool { return readNativeRectForTest() == original })
	if capture := os.Getenv("TWO_STORE_CAPTURE"); capture != "" {
		var captureErr error
		fyne.DoAndWait(func() {
			f, e := os.Create(capture)
			if e != nil {
				captureErr = e
				return
			}
			defer f.Close()
			captureErr = png.Encode(f, a.win.Canvas().Capture())
		})
		if captureErr != nil {
			t.Fatal(captureErr)
		}
	}
	// No session is normal; a genuine capture failure is displayed without a modal.
	mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, wincred.ErrNotFound }, nil)
	clickNativeControl(t, a.refreshBtn)
	waitNativeResize(t, func() bool { return !a.isBusy && a.overview.ActiveEmail == "" })
	mgr.SetWincredHandlers(func() (*model.TokenData, error) { return nil, errors.New("native fixture read denied") }, nil)
	clickNativeControl(t, a.refreshBtn)
	waitNativeResize(t, func() bool { return !a.isBusy && strings.Contains(a.statusBar.Text, "native fixture read denied") })
	if a.win.Canvas().Overlays().Top() != nil {
		t.Fatal("capture warning created a modal")
	}
	t.Log("native refresh: three states, repeated refresh, credential removal/recovery, warning, resize PASS")
	time.Sleep(50 * time.Millisecond)
}

// Called on the Fyne thread by waitNativeResize.
func readNativeRectForTest() resizeRect {
	rect, _, _ := nativeResizeState()
	return rect
}
