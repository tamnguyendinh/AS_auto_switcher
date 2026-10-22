# Đặc Tả Kiến Trúc & Chỉ Dẫn Kỹ Thuật Chi Tiết Cho Coder: Đồng Bộ Tỷ Lệ Hiển Thị & Thanh Lọc Texture Cache Khi Khởi Động Ứng Dụng (Startup UI Scale & Texture Synchronization — Complete Implementation Recipe)

- **Ngày lập:** 2026-10-22
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261022-STARTUP-UI-SCALE-TEXTURE-SYNC`
- **Trạng thái:** DRAFT (Trình Human Owner xem xét)
- **Đối tượng đọc:** Coder, Supervisor, QA

---

## I. BỐI CẢNH & PHÂN TÍCH NGUYÊN NHÂN GỐC RỄ (CONTEXT & ROOT CAUSE ANALYSIS)

### 1. Hiện tượng thực tế (Observed Problem)
Khi người dùng mở ứng dụng `antigravity-switch`:
- **Giai đoạn khởi đầu (Initial Startup):** Giao diện tổng thể nhìn khá đồng đều nhưng có cảm giác **nở to, nét chữ dày và hơi nhòe/mờ nhẹ (kém sắc nét)**.
- **Giai đoạn sau khi đổi Theme tại runtime (Runtime Theme Switch):** Khi người dùng bấm vào danh sách chọn chủ đề giao diện (🌙 Tối, ☀️ Sáng, ☕ Ấm), toàn bộ UI ngay lập tức **thu nhỏ lại về đúng tỷ lệ chuẩn, thanh mảnh, viền khung và phông chữ trở nên cực kỳ sắc nét (crisp) và tinh tế**.

### 2. Nguyên nhân kỹ thuật gốc rễ (Technical Root Cause)
1. **Lệch pha Vòng đời Fyne GLFW Event Loop (Lifecycle Desynchronization):**
   - Trong driver OpenGL/GLFW của Fyne (`internal/driver/glfw/loop.go`), bộ lắng nghe thay đổi cấu hình hiển thị và phông chữ (`Settings().AddListener(...)`) **chỉ được đăng ký sau khi `driver.Run()` (bên trong `a.app.Run()`) chính thức chạy**.
   - Trong khi đó, hàm `a.Init()` cài đặt Theme (`a.app.Settings().SetTheme(...)`) và dựng cây Widget lại diễn ra **trước khi `a.app.Run()` được gọi**.
   - Do đó, tại thời điểm mở app và vẽ frame đầu tiên, bộ lắng nghe `AddListener` chưa chạy, các texture chữ (`textTextures`) ban đầu được FreeType rasterize khi cửa sổ GLFW chưa hoàn tất handshake với Windows về Per-Monitor DPI V2 (120 DPI = 125% scale, hệ số scale thực tế là `1.3`). Texture chữ bị lấy mẫu nội suy (linear filter sampling), khiến chữ bị nở to và mờ nét.
2. **Cơ chế đặc biệt khi đổi Theme tại Runtime:**
   Khi người dùng click đổi theme lúc ứng dụng đang chạy:
   - Fyne kích hoạt listener trong `loop.go`:
     - `painter.ClearFontCache()`
     - `cache.ResetThemeCaches()`
     - `cache.DeleteTextTexturesFor(w.canvas)` (Xóa sạch toàn bộ texture chữ cũ, ép vẽ lại từng glyph ở đúng pixel density thực tế `pixScale = 1.3`).
     - `c.reloadScale()` và `RescaleContext()` (Căn chỉnh lại kích thước viewport và lưới điểm ảnh OpenGL).
   - Hàm `a.SetTheme()` trong `internal/ui/app.go` gọi `a.forceRebuildCards(overview)`, xóa cache thẻ cũ và vẽ lại mới toàn bộ danh sách tài khoản theo lưới pixel chuẩn.
   - 👉 Chuỗi thao tác này đã "gọt giũa" giao diện, làm cho UI co lại đúng kích thước chuẩn và sắc nét 100%.

---

## II. MỤC TIÊU VÀ NGUYÊN TẮC CỐT LÕI (OBJECTIVES & INVARIANTS)

Bản đặc tả này thiết lập cơ chế **"Tự động đồng bộ Texture & Viewport hậu khởi động" (Post-Show Visual Synchronization Lifecycle)**: Ngay khi cửa sổ hiển thị và Event Loop GLFW bắt đầu hoạt động, hệ thống tự động kích hoạt chu trình làm sạch texture cache và đồng bộ viewport, đảm bảo người dùng vừa mở app lên là đã có ngay giao diện nhỏ gọn, sắc nét chuẩn 100%, đồng nhất tuyệt đối mà không cần phải bấm đổi theme thủ công.

### Các điều khoản kỷ luật sắt (Mandatory Invariants):

1. **BẢO TỒN NGUYÊN VẸN CẤU HÌNH NGƯỜI DÙNG (ZERO PREFERENCES MUTATION INVARIANT):**
   - Chu trình đồng bộ hậu khởi động **tuyệt đối không được ghi đè hay thay đổi** tệp `logs/ui_preferences.json`.
   - Theme đang kích hoạt của người dùng (Dark, Light hoặc Warm) phải được giữ nguyên vẹn 100%. Thao tác đồng bộ chỉ làm tươi texture đồ họa và viewport trên canvas, không làm thay đổi trạng thái cài đặt.

2. **AN TOÀN ĐA LUỒNG & NON-BLOCKING (THREAD-SAFETY & ZERO UI JANK INVARIANT):**
   - Thao tác đồng bộ hậu khởi động bắt buộc phải được kích hoạt từ một Goroutine nền nhẹ và đưa vào luồng chính của Fyne thông qua `fyne.Do(...)`.
   - Tuyệt đối không được gọi trực tiếp các hàm thao tác GUI hay GLFW context ngoài main thread của Fyne (tránh triệt để crash OpenGL / Access Violation).

3. **TƯƠNG THÍCH MỌI KỊCH BẢN TẢI DỮ LIỆU TÀI KHOẢN (RACE-CONDITION FREE OVERVIEW INVARIANT):**
   - Hệ thống phải xử lý trơn tru cả 2 kịch bản:
     - **Kịch bản A (Overview về sau):** Nếu `a.Refresh()` chưa xong lúc đồng bộ chạy, canvas được thanh lọc trước; khi overview về, `updateUI` sẽ vẽ các thẻ lên canvas đã sắc nét.
     - **Kịch bản B (Overview về trước):** Nếu `a.Refresh()` đã hoàn tất và `a.overview != nil`, chu trình đồng bộ sẽ tự động gọi làm tươi lại các thẻ (`forceRebuildCards`) để áp dụng ngay bộ texture sắc nét mới.

4. **BẢO TOÀN KÍCH THƯỚC CỐ ĐỊNH & SINGLE INSTANCE (WINDOW GEOMETRY & LOCK INVARIANT):**
   - Kích thước thẻ tài khoản cố định `280x130` và lưới co giãn đối xứng `newSymmetricGridWrap` không bị thay đổi logic.
   - Cơ chế Single Instance Mutex trên Windows không bị ảnh hưởng.

---

## III. SƠ ĐỒ KIẾN TRÚC & QUY TRÌNH THỰC THI (ARCHITECTURE WORKFLOW)

```text
                     [Khởi chạy: cmd/switcher/main.go]
                                     │
                                     ▼
                            [ui.NewApp(mgr)]
                   (Nạp Theme đã lưu từ ui_preferences.json)
                                     │
                                     ▼
                                [a.Init()]
       (Tạo Splash Window, cấu hình Theme ban đầu, win.Resize(920, 630))
                                     │
                                     ▼
                               [a.win.Show()]
                    (GLFW tạo HWND và hiển thị lên màn hình)
                                     │
                                     ▼
                        [bindMainWindowHandle(a.win)]
                        [setWindowAlwaysOnTop(...)]
                                     │
                                     ▼
             [Khởi chạy Post-Show Visual Sync Goroutine]
                                     │
                    ┌────────────────┴────────────────┐
                    │                                 │
                    ▼                                 ▼
             [a.app.Run()]                  [Goroutine nền]
      (Event Loop GLFW bắt đầu;                       │
       AddListener đã sẵn sàng)           (Chờ 40-60ms cho GLFW map xong)
                    │                                 │
                    │◄─────── fyne.Do(...) ───────────┘
                    │   • a.SyncVisualScaleAndTextures()
                    │     - a.app.Settings().SetTheme(NewThemeInstance(theme))
                    │       -> painter.ClearFontCache()
                    │       -> cache.ResetThemeCaches()
                    │       -> cache.DeleteTextTexturesFor(canvas)
                    │       -> c.reloadScale() / RescaleContext()
                    │     - Làm tươi các Canvas Object thanh công cụ
                    │     - Nếu overview != nil: a.forceRebuildCards(overview)
                    ▼
     [KẾT QUẢ: Toàn bộ Text & Card đạt 100% độ sắc nét & kích thước chuẩn ngay từ giây đầu tiên]
```

---

## IV. CHỈ DẪN TRIỂN KHAI CHI TIẾT TỪNG BƯỚC CHO CODER (EXHAUSTIVE IMPLEMENTATION RECIPE)

---

### BƯỚC 1: XÂY DỰNG PHƯƠNG THỨC `SyncVisualScaleAndTextures()` TRONG `internal/ui/app.go`

Tại file `internal/ui/app.go` (ngay bên dưới hàm `SetTheme`, khoảng dòng 308):

**Coder bổ sung phương thức mới:**

```go
// SyncVisualScaleAndTextures purges initial unscaled font textures and synchronizes
// the active theme palette, canvas scaling, and viewport on the visible window.
// It executes identical visual sharpening to a runtime theme switch without modifying
// or re-saving the user's stored preferences.
func (a *App) SyncVisualScaleAndTextures() {
	a.mu.RLock()
	themeName := a.currentTheme
	overview := a.overview
	a.mu.RUnlock()

	// 1. Kích hoạt Fyne GLFW driver thanh lọc toàn bộ text textures cũ và rescale context
	if a.app != nil {
		a.app.Settings().SetTheme(NewThemeInstance(themeName))
	}

	palette := GetThemePalette(themeName)

	// 2. Làm tươi các phần tử Canvas trên Header để đồng bộ màu sắc và texture sắc nét
	if a.monogramBg != nil {
		a.monogramBg.FillColor = palette.Primary
		a.monogramBg.Refresh()
	}
	if a.autoSwitchToggle != nil {
		a.autoSwitchToggle.SetPalette(palette)
	}
	if a.autoSwitchLabel != nil {
		a.autoSwitchLabel.Color = palette.Text
		a.autoSwitchLabel.Refresh()
	}
	if a.maskAccountsToggle != nil {
		a.maskAccountsToggle.SetPalette(palette)
	}
	if a.row2Divider1 != nil {
		a.row2Divider1.FillColor = palette.BorderSubtle
		a.row2Divider1.Refresh()
	}
	if a.row2Divider2 != nil {
		a.row2Divider2.FillColor = palette.BorderSubtle
		a.row2Divider2.Refresh()
	}

	// 3. Nếu danh sách tài khoản đã được tải xong từ trước, buộc dựng lại toàn bộ thẻ
	// với bộ đo đạc font chữ sắc nét mới
	if overview != nil {
		a.forceRebuildCards(overview)
	}
}
```

---

### BƯỚC 2: TÍCH HỢP GOROUTINE ĐỒNG BỘ VÀO `ShowAndRun()` TRONG `internal/ui/app.go`

Tại file `internal/ui/app.go`, tìm hàm `ShowAndRun()` (khoảng dòng 548–566):

**Code hiện tại:**
```go
// ShowAndRun displays the window and runs the Fyne event loop.
func (a *App) ShowAndRun() {
	if a.win == nil {
		a.Init()
	}
	defer a.Close()

	// 1. Ép GLFW tạo HWND và hiển thị cửa sổ lên Windows
	a.win.Show()

	// The driver created the window without decorations. Only retain its native
	// handle for the custom drag and minimize controls; never rewrite its style.
	bindMainWindowHandle(a.win)

	setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)

	// Chạy vòng lặp nhận sự kiện của Fyne
	a.app.Run()
}
```

**Coder sửa thành:**
```go
// ShowAndRun displays the window and runs the Fyne event loop.
func (a *App) ShowAndRun() {
	if a.win == nil {
		a.Init()
	}
	defer a.Close()

	// 1. Ép GLFW tạo HWND và hiển thị cửa sổ lên Windows
	a.win.Show()

	// The driver created the window without decorations. Only retain its native
	// handle for the custom drag and minimize controls; never rewrite its style.
	bindMainWindowHandle(a.win)

	setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)

	// 2. Kích hoạt đồng bộ phông chữ, texture cache và viewport hậu khởi động
	// Chạy trong Goroutine nền nhẹ, chờ GLFW event loop sẵn sàng rồi dispatch vào main thread qua fyne.Do.
	go func() {
		// Nhịp nghỉ 50ms đảm bảo GLFW đã đăng ký xong AddListener trong driver.Run()
		time.Sleep(50 * time.Millisecond)
		fyne.Do(func() {
			a.SyncVisualScaleAndTextures()
		})
	}()

	// 3. Chạy vòng lặp nhận sự kiện của Fyne
	a.app.Run()
}
```

---

### BƯỚC 3: ĐẢM BẢO TÍNH TOÀN VẸN CỦA `Init()` VÀ KHÔNG GÂY LỆCH STATE

Tại `Init()` trong `internal/ui/app.go`:
- Giữ nguyên các lời gọi hiện tại.
- Hàm `SyncVisualScaleAndTextures()` được xuất khẩu (exported: chữ `S` hoa) để:
  1. `ShowAndRun()` gọi khi chạy thực tế.
  2. Các hàm kiểm thử tự động (Unit Test / Integration Test) có thể gọi trực tiếp để xác minh mà không cần phải chạy GUI blocking.

---

## V. KỊCH BẢN KIỂM THỬ TỰ ĐỘNG & TIÊU CHÍ CHẤP THUẬN (ACCEPTANCE TEST SUITE)

Coder bắt buộc bổ sung các bài test tự động vào file `internal/ui/themes_test.go` (hoặc `app_test.go`):

### Test Case 1: Xác minh `SyncVisualScaleAndTextures` không làm bẩn file cấu hình
- **Tên hàm test:** `TestSyncVisualScaleAndTextures_PreservesPreferences`
- **Mục tiêu:**
  - Khởi tạo `App` với tệp preferences tạm có `theme: "dark"`, `mask_accounts: true`, `auto_switch: false`, `always_on_top: true`.
  - Gọi `a.Init()`.
  - Gọi `a.SyncVisualScaleAndTextures()`.
  - Nạp lại file preferences từ đĩa.
  - **Khẳng định:** Toàn bộ 4 giá trị preferences trên đĩa phải giữ nguyên 100%, không bị reset hay thay đổi timestamp bất thường.

### Test Case 2: Xác minh `SyncVisualScaleAndTextures` tái dựng thẻ tài khoản nếu overview đã tồn tại
- **Tên hàm test:** `TestSyncVisualScaleAndTextures_RebuildsCardsWhenOverviewPresent`
- **Mục tiêu:**
  - Nạp overview giả lập gồm 2 tài khoản.
  - Gọi `a.updateUI(overview)` -> ghi nhận con trỏ của `a.accountCards[0]`.
  - Gọi `a.SyncVisualScaleAndTextures()`.
  - **Khẳng định:** `len(a.accountCards) == 2`, và con trỏ `a.accountCards[0]` phải là đối tượng mới được tạo lại (`rebuilt`), đảm bảo texture mới đã được áp dụng.

### Test Case 3: Xác minh `SyncVisualScaleAndTextures` hoạt động an toàn khi overview là nil
- **Tên hàm test:** `TestSyncVisualScaleAndTextures_SafeWhenOverviewNil`
- **Mục tiêu:**
  - Khởi tạo `App`, gọi `a.Init()` nhưng không cung cấp overview.
  - Gọi `a.SyncVisualScaleAndTextures()`.
  - **Khẳng định:** Không xảy ra panic, các widget header (`monogramBg`, `autoSwitchToggle`, `maskAccountsToggle`) được làm tươi bình thường.

---

## VI. TIÊU CHÍ NGHIỆM THU CHẤT LƯỢNG (ACCEPTANCE CRITERIA)

| Mã AC | Nội dung tiêu chí | Phương thức kiểm tra |
|---|---|---|
| **AC-1** | Giao diện khi vừa mở app lập tức đạt kích thước nhỏ gọn, nét chữ và viền khung sắc nét 100%, không còn hiện tượng nở to hay mờ nét. | Kiểm tra trực quan trên màn hình Windows DPI 125% (120 DPI). |
| **AC-2** | Bấm đổi Theme qua lại giữa 🌙 ☀️ ☕ tại runtime hoạt động mượt mà, không bị giật lag, giữ nguyên độ sắc nét đồng nhất. | Kiểm tra thủ công và chạy test suite `TestAppSetThemeRuntime`. |
| **AC-3** | Không làm thay đổi hay ghi đè bất kỳ giá trị nào trong `logs/ui_preferences.json` lúc khởi động. | Chạy test tự động `TestSyncVisualScaleAndTextures_PreservesPreferences`. |
| **AC-4** | Không phát sinh data race hay deadlock giữa goroutine khởi động và luồng nạp dữ liệu tài khoản `Refresh()`. | `go test -race ./internal/ui/...` đạt PASS 100%. |
| **AC-5** | Toàn bộ các bài kiểm thử hiện có của dự án (`go test ./cmd/switcher/... ./internal/...`) đều đạt PASS 100%. | Chạy toàn bộ test suite dự án. |

---

## VII. MA TRẬN TÁC ĐỘNG TỆP TIN (FILE IMPACT MATRIX)

Coder chỉ được phép tác động lên các tệp tin được cấp phép sau:

| Tệp tin | Loại tác động | Mục đích thay đổi |
|---|---|---|
| `internal/ui/app.go` | Bổ sung hàm & tích hợp | Bổ sung `SyncVisualScaleAndTextures()` và gọi trong goroutine hậu khởi động tại `ShowAndRun()`. |
| `internal/ui/themes_test.go` | Bổ sung Test Cases | Thêm bộ 3 test cases tự động kiểm tra tính toàn vẹn và an toàn của chu trình đồng bộ. |
| `scripts/build_windows.ps1` | **CẤM CHẠM VÀO (UNTOUCHED)** | Giữ nguyên tuyệt đối 100%. |
| Tất cả các package khác (`store`, `manager`, `telemetry`, `cmd`) | **CẤM THAY ĐỔI** | 0 dòng code thay đổi ngoài UI. |

---
*Văn kiện đặc tả kiến trúc được biên soạn hoàn chỉnh bởi System Architect phục vụ bàn giao trực tiếp cho Coder triển khai không cần suy nghĩ.*
