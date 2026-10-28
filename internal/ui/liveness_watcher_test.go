package ui

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/core/i18n"
	"antigravity-switch/internal/engines/antigravity/connector"
	"antigravity-switch/internal/engines/antigravity/manager"
)

// TC-LIVE-01: Chuyển đổi trạng thái khi Antigravity tắt (Running -> Stopped)
// Khi Antigravity IDE hoặc language_server.exe bị đóng bên ngoài hệ thống:
// - Liveness watcher phát hiện qua livenessChecker trả về false.
// - InvalidateDiscoveryCache được gọi.
// - overview.LanguageServerRunning chuyển thành false, ServerInfo thành nil.
// - Nhãn Footer UI cập nhật về trạng thái "Antigravity: Không hoạt động  | Dự phòng WinCred".
func TestLivenessWatcher_TC_LIVE_01_RunningToStopped(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.serverStatusLabel = widget.NewLabel("")

	// Thiết lập overview đang Running với PID 9999
	app.mu.Lock()
	app.overview = &manager.AccountOverview{
		LanguageServerRunning: true,
		ServerInfo: &connector.ServerInfo{
			PID:  9999,
			Port: 54321,
		},
	}
	app.mu.Unlock()
	app.renderServerStatusLabel(true, app.overview.ServerInfo)

	// Mock livenessChecker trả về false (tiến trình đã chết ngoài hệ thống)
	app.SetLivenessChecker(func(info *connector.ServerInfo) bool {
		return false
	})
	app.SetServerDiscoverer(func() (*connector.ServerInfo, error) {
		return nil, connector.ErrLanguageServerNotFound
	})

	// Kích hoạt checkLivenessTick()
	app.checkLivenessTick()

	// Assertions
	app.mu.RLock()
	running := app.overview.LanguageServerRunning
	serverInfo := app.overview.ServerInfo
	app.mu.RUnlock()

	if running {
		t.Errorf("TC-LIVE-01: expected overview.LanguageServerRunning to be false, got true")
	}
	if serverInfo != nil {
		t.Errorf("TC-LIVE-01: expected overview.ServerInfo to be nil, got %+v", serverInfo)
	}

	expectedText := i18n.Current().ServerStopped // "Antigravity: Không hoạt động  | Dự phòng WinCred"
	if app.serverStatusLabel.Text != expectedText {
		t.Errorf("TC-LIVE-01: expected label text '%s', got '%s'", expectedText, app.serverStatusLabel.Text)
	}
}

// TC-LIVE-02: Chuyển đổi trạng thái khi Antigravity bật lại (Stopped -> Running)
// Khi Antigravity IDE được người dùng mở lại:
// - Liveness watcher phát hiện qua serverDiscoverer trả về ServerInfo hợp lệ.
// - overview.LanguageServerRunning chuyển thành true, ServerInfo nhận PID/Port mới.
// - Nhãn Footer UI cập nhật về trạng thái hoạt động với đúng PID & Cổng mới.
func TestLivenessWatcher_TC_LIVE_02_StoppedToRunning(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.serverStatusLabel = widget.NewLabel("")

	// Thiết lập overview đang Stopped
	app.mu.Lock()
	app.overview = &manager.AccountOverview{
		LanguageServerRunning: false,
		ServerInfo:            nil,
	}
	app.mu.Unlock()
	app.renderServerStatusLabel(false, nil)

	// Mock serverDiscoverer trả về tiến trình mới PID 8888, Port 55555
	app.SetLivenessChecker(func(info *connector.ServerInfo) bool {
		return true
	})
	app.SetServerDiscoverer(func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:  8888,
			Port: 55555,
		}, nil
	})

	// Kích hoạt checkLivenessTick()
	app.checkLivenessTick()

	// Assertions
	app.mu.RLock()
	running := app.overview.LanguageServerRunning
	info := app.overview.ServerInfo
	app.mu.RUnlock()

	if !running {
		t.Errorf("TC-LIVE-02: expected overview.LanguageServerRunning to be true, got false")
	}
	if info == nil || info.PID != 8888 || info.Port != 55555 {
		t.Fatalf("TC-LIVE-02: expected ServerInfo with PID 8888 and Port 55555, got %+v", info)
	}

	d := i18n.Current()
	expectedText := fmt.Sprintf(d.ServerConnected, 8888, 55555)
	if app.serverStatusLabel.Text != expectedText {
		t.Errorf("TC-LIVE-02: expected label text '%s', got '%s'", expectedText, app.serverStatusLabel.Text)
	}
}

// TC-LIVE-03: Bảo vệ trạng thái khi ứng dụng bận (isBusy Yield)
// Trong suốt quá trình chuyển tài khoản (8 giây) hoặc refresh, watcher phải yield ngay lập tức:
// - Không gọi livenessChecker hay serverDiscoverer.
// - Giữ nguyên tuyệt đối nhãn Footer UI và dữ liệu overview trong RAM.
func TestLivenessWatcher_TC_LIVE_03_BusyYield(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.serverStatusLabel = widget.NewLabel("")

	// Thiết lập overview đang Running và app.isBusy = true
	app.mu.Lock()
	app.isBusy = true
	app.overview = &manager.AccountOverview{
		LanguageServerRunning: true,
		ServerInfo: &connector.ServerInfo{
			PID:  9999,
			Port: 54321,
		},
	}
	app.mu.Unlock()
	app.renderServerStatusLabel(true, app.overview.ServerInfo)
	initialText := app.serverStatusLabel.Text

	// Mock livenessChecker trả về false
	checkerCalled := false
	app.SetLivenessChecker(func(info *connector.ServerInfo) bool {
		checkerCalled = true
		return false
	})

	// Kích hoạt checkLivenessTick() khi isBusy == true
	app.checkLivenessTick()

	// Assertions: checker không được gọi, trạng thái không đổi
	if checkerCalled {
		t.Errorf("TC-LIVE-03: checker must not be called when isBusy is true")
	}

	app.mu.RLock()
	running := app.overview.LanguageServerRunning
	info := app.overview.ServerInfo
	app.mu.RUnlock()

	if !running {
		t.Errorf("TC-LIVE-03: overview.LanguageServerRunning must remain true while busy")
	}
	if info == nil || info.PID != 9999 {
		t.Errorf("TC-LIVE-03: overview.ServerInfo must not be modified while busy")
	}
	if app.serverStatusLabel.Text != initialText {
		t.Errorf("TC-LIVE-03: label text must not change, expected '%s', got '%s'", initialText, app.serverStatusLabel.Text)
	}
}

// TC-LIVE-04: Graceful Shutdown (Không rò rỉ Goroutine)
// Khi ứng dụng Close(), context phải được cancel và goroutine watcher dừng hoàn toàn.
func TestLivenessWatcher_TC_LIVE_04_GracefulTeardown(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)

	// Khởi tạo app.Init()
	app.Init()
	waitForIdle(app, 1*time.Second)

	app.mu.RLock()
	cancel := app.watcherCancel
	app.mu.RUnlock()

	if cancel == nil {
		t.Fatalf("TC-LIVE-04: expected watcherCancel to be non-nil after Init()")
	}

	// Gọi Close()
	app.Close()

	app.mu.RLock()
	canceledWatcher := app.watcherCancel
	app.mu.RUnlock()

	if canceledWatcher != nil {
		t.Errorf("TC-LIVE-04: expected watcherCancel to be nil after Close()")
	}
}

// TC-LIVE-05: Edge-Triggered Debounce (Chống vẽ lặp giao diện khi trạng thái ổn định)
func TestLivenessWatcher_TC_LIVE_05_EdgeTriggeredDebounce(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	mgr := createTestManager(t)
	app := newIsolatedApp(testApp, mgr)
	defer app.Close()

	app.serverStatusLabel = widget.NewLabel("")

	app.mu.Lock()
	app.overview = &manager.AccountOverview{
		LanguageServerRunning: true,
		ServerInfo: &connector.ServerInfo{
			PID:  12345,
			Port: 54321,
		},
	}
	app.mu.Unlock()
	app.renderServerStatusLabel(true, app.overview.ServerInfo)

	// Thay đổi text của label bằng một chuỗi custom để phát hiện nếu label bị vẽ lại
	app.serverStatusLabel.SetText("Custom-Sentinel-Value")

	// Mock livenessChecker trả về true (tiến trình vẫn còn sống bình thường)
	app.SetLivenessChecker(func(info *connector.ServerInfo) bool {
		return true
	})

	// Kích hoạt tick: do trạng thái không đổi, watcher KHÔNG được gọi fyne.Do / renderServerStatusLabel
	app.checkLivenessTick()

	// Text phải giữ nguyên giá trị sentinel
	if app.serverStatusLabel.Text != "Custom-Sentinel-Value" {
		t.Errorf("TC-LIVE-05: expected debounce to prevent redraw, but label text changed to '%s'", app.serverStatusLabel.Text)
	}

	// Trường hợp 2: Khi Stopped và discoverer không tìm thấy tiến trình mới
	app.mu.Lock()
	app.overview.LanguageServerRunning = false
	app.overview.ServerInfo = nil
	app.mu.Unlock()
	app.serverStatusLabel.SetText("Stopped-Sentinel-Value")

	app.SetServerDiscoverer(func() (*connector.ServerInfo, error) {
		return nil, connector.ErrLanguageServerNotFound
	})

	app.checkLivenessTick()

	if app.serverStatusLabel.Text != "Stopped-Sentinel-Value" {
		t.Errorf("TC-LIVE-05: expected debounce to prevent redraw when still stopped, got '%s'", app.serverStatusLabel.Text)
	}
}
