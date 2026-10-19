# Báo Cáo Đặc Tả Kiến Trúc: Khắc Phục Lỗi Tự Động Co Giãn Giao Diện, Cập Nhật Quota Tại Chỗ & Lưu Vết Kích Thước Cửa Sổ

- **Ngày ban hành:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261019-UI-RESIZE-STABILITY`
- **Trạng thái:** APPROVED BY OWNER — READY FOR CODER IMPLEMENTATION

---

## 1. Bối Cảnh & Vấn Đề Kỹ Thuật (Problem Statement)

Human Owner trực tiếp phản ánh hiện tượng:
1. Giao diện tool cứ một chốc lại tự động thay đổi kích thước (co giãn) dù không hề có ai đang đụng vào code.
2. Khi người dùng dùng chuột kéo thay đổi kích thước UI của tool thì kích thước không cố định như ý muốn mà cứ tự động co giãn / giật lại kích thước cũ mỗi khi Quota được tự động refresh.

### Phân tích nguyên nhân gốc rễ (Root Cause Analysis):
1. **Chu kỳ 60s của `AutoTelemetryLoop` (`internal/ui/app.go:213-234`):**
   - Vòng lặp nền quét Quota từ Google API mỗi 60 giây và kích hoạt `fyne.Do(func() { a.updateUI(overview) })`.
2. **Cơ chế hủy và tái tạo toàn bộ Canvas (`internal/ui/app.go:660-685`):**
   - Trong `updateUI`, lệnh `a.accountListContainer.Objects = nil` xóa sạch toàn bộ các thẻ tài khoản hiện hữu và khởi tạo lại từ đầu (`NewAccountCardWithQuota`).
   - Các Card mới sinh ra có kích thước `0x0`, chuỗi text Quota chuyển từ `--%` sang `100.0%` hoặc `43.8%`, làm thay đổi MinSize đột ngột và kích hoạt chuỗi re-layout toàn bộ cây giao diện, gây hiện tượng giật khung hình (Layout Jank) và co giãn thị giác.
3. **Cơ chế `EnsureMinSize` & `fitContent` của Fyne GLFW Driver:**
   - Trong `internal/ui/window.go`, `rootContainer` bị khóa cứng `minSizeLayout{min: fyne.NewSize(840, 525)}`. Trên màn hình Windows có Display Scaling (125% - 150%), giá trị này tương đương `1050x656` đến `1260x787` px thực tế.
   - Khi người dùng kéo thu nhỏ cửa sổ (< 840x525), mỗi lần Quota refresh gọi `Refresh()`, Fyne driver phát hiện kích thước cửa sổ hiện tại nhỏ hơn `minSizeOnScreen()`, liền kích hoạt cờ `w.shouldExpand = true` và gọi `view.SetSize()`, **ép cửa sổ bật tung to trở lại**.
4. **Cột Quota bị khóa cứng 88px (`internal/ui/quota_badge.go:522`):**
   - Khóa cứng `colCycleW = 88.0` khiến thẻ Card khi giãn rộng không tự phân bổ không gian đều cho 2 cột 5h và Week, gây lệch tỷ lệ và khoảng trống thừa.
5. **Thiếu cơ chế lưu vết kích thước cửa sổ:**
   - `UIPreferences` chỉ lưu `theme`, hoàn toàn không lưu `WindowWidth` và `WindowHeight` của người dùng.

---

## 2. Các Bất Biến Kiến Trúc (Architectural Invariants)

- **INV-01 (In-Place Reactive Quota Updates):** Tuyệt đối KHÔNG xóa cây Canvas (`Objects = nil`) trong chu kỳ 60s của `updateUI` nếu danh sách tài khoản không thay đổi. Phải cập nhật trực tiếp dữ liệu vào các widget Text và FillBar đã tồn tại.
- **INV-02 (Freedom of Window Resizing):** Ngưỡng kích thước tối thiểu của cửa sổ (`MinWindowWidth`, `MinWindowHeight`) phải hạ xuống mức an toàn `480 x 360 px`. Khi người dùng thu nhỏ, `VScroll` tự động kích hoạt thanh cuộn, Fyne GLFW driver tuyệt đối không được tự ý phóng to cửa sổ.
- **INV-03 (Window Geometry Persistence):** Kích thước cửa sổ (`WindowWidth`, `WindowHeight`) do người dùng tùy biến phải được lưu trữ bền vững vào `accounts_store/.ui_preferences.json` và khôi phục chính xác khi mở lại ứng dụng.
- **INV-04 (Elastic Matrix Quota Table):** Bảng Quota phải tự động co giãn theo tỷ lệ bề rộng thực tế của thẻ Card (`clamp(88.0, (cardWidth - fixedWidth)/2, 125.0)`).

---

## 3. Thiết Kế Chi Tiết & Giải Pháp Kỹ Thuật

### 3.1. Phân hệ 1: Lưu vết kích thước cửa sổ (`internal/ui/themes.go`)

Mở rộng struct `UIPreferences`:
```go
type UIPreferences struct {
    Theme        string  `json:"theme"`
    WindowWidth  float32 `json:"window_width,omitempty"`
    WindowHeight float32 `json:"window_height,omitempty"`
}
```

### 3.2. Phân hệ 2: Hạ ngưỡng MinSize và Debounced Window Size Persistence (`internal/ui/window.go`)

1. Khai báo hằng số kích thước:
```go
const (
    DefaultWindowWidth  float32 = 1008.0
    DefaultWindowHeight float32 = 630.0
    MinWindowWidth      float32 = 480.0
    MinWindowHeight     float32 = 360.0
)
```

2. Cập nhật `setupLayout()`:
```go
rootContainer := container.New(&minSizeLayout{min: fyne.NewSize(MinWindowWidth, MinWindowHeight), app: a}, mainLayout)
a.win.SetContent(rootContainer)
```

3. Ghi nhận kích thước khi người dùng resize trong `minSizeLayout.Layout`:
```go
func (m *minSizeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
    for _, o := range objects {
        o.Resize(size)
        o.Move(fyne.NewPos(0, 0))
    }
    if m.app != nil && size.Width >= MinWindowWidth && size.Height >= MinWindowHeight {
        m.app.OnWindowResized(size.Width, size.Height)
    }
}
```

4. Cơ chế Debounce Lưu Disk trong `internal/ui/app.go`:
- Khi nhận sự kiện resize, sử dụng timer debounce 300ms. Khi người dùng dừng kéo chuột, lưu `WindowWidth` và `WindowHeight` vào `a.preferencesPath`.
- Tại `Init()`: Nếu `prefs.WindowWidth > 0 && prefs.WindowHeight > 0`, áp dụng `a.win.Resize(fyne.NewSize(prefs.WindowWidth, prefs.WindowHeight))`.

### 3.3. Phân hệ 3: Cập nhật Quota tại chỗ (In-Place Reactive Data Binding) (`internal/ui/widgets.go` & `app.go`)

1. Cấu trúc `AccountCardWidget`:
Lưu giữ trực tiếp con trỏ tới các widget hiển thị:
- `gemini5hPct`, `gemini5hBar`
- `geminiWeekPct`, `geminiWeekBar`
- `claude5hPct`, `claude5hBar`
- `claudeWeekPct`, `claudeWeekBar`
- `emailLabel`, `actionCluster`, `cardBg`

2. Phương thức `UpdateQuota(quota *telemetry.AccountQuotaData, palette ThemePalette)`:
- Trích xuất `dq := ExtractDualCycleQuota(quota)`.
- Cập nhật text chuỗi `%` và tính toán lại `fillBar.Resize()` cục bộ.
- Gọi `Refresh()` cục bộ trên các phần tử con, **tuyệt đối không gọi `Refresh()` trên toàn bộ `accountListContainer`**.

3. Tái cấu trúc `updateUI` trong `internal/ui/app.go`:
- Giữ một slice `accountCards []*AccountCardWidget` trong `App`.
- So sánh danh sách `overview.Accounts` với `accountCards`:
  + Nếu số lượng và email khớp 100%: Lặp qua và gọi `a.accountCards[i].UpdateQuota(quotaData, palette)`. Hoàn toàn không đụng vào `a.accountListContainer.Objects`.
  + Chỉ khi có email mới hoặc xóa bớt email: Mới thực hiện dựng lại danh sách Card và gán lại `Objects`.

### 3.4. Phân hệ 4: Bảng Quota co giãn đàn hồi (`internal/ui/quota_badge.go`)

Trong `matrixTableLayout.Layout`:
```go
fixedW := col1W + 2*sepW + 4*colGap // ~60px
remainW := size.Width - fixedW
colCycleW := float32(88.0)
if remainW > 176.0 {
    // Phân bổ đều cho 2 cột 5h và Week, chặn trên 125px để bảo toàn typography
    colCycleW = float32(math.Min(float64(remainW / 2.0), 125.0))
}
```

---

## 4. Khóa Phạm Vi Chỉnh Sửa (Touch-Map Khóa Cứng)

Coder Lane khi triển khai TUYỆT ĐỐI chỉ được phép chỉnh sửa đúng 5 file sau:

| STT | Đường dẫn File | Mục đích sửa đổi |
|---|---|---|
| 1 | `internal/ui/themes.go` | Mở rộng `UIPreferences` bổ sung `WindowWidth`, `WindowHeight` |
| 2 | `internal/ui/quota_badge.go` | Tính toán đàn hồi `colCycleW` trong `matrixTableLayout.Layout` |
| 3 | `internal/ui/widgets.go` | Cấu trúc `AccountCardWidget` và hàm `UpdateQuota` in-place |
| 4 | `internal/ui/window.go` | Hạ MinWindowWidth/Height về `480x360`, nối hook resize |
| 5 | `internal/ui/app.go` | Khôi phục window size khi mở, debounce lưu disk, in-place `updateUI` |
| 6 | `internal/ui/ui_test.go` | Bổ sung test kiểm thử In-Place update, Geometry persistence, MinSize |

---

## 5. Ranh Giới Nghiêm Ngặt (Non-Goals)

- KHÔNG thay đổi logic nghiệp vụ của `telemetryPool`, `QuotaClient` hay Google API connector.
- KHÔNG thay đổi thuật toán hoán đổi tài khoản (hot-swap) trong `manager`.
- KHÔNG thay đổi giao diện dòng lệnh (CLI `--quota`, `--status`).
- KHÔNG thay đổi màu sắc bảng màu hoặc font chữ đã nghiệm thu.

---

## 6. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **AC-01 (In-Place Quota Update):** Khi `autoLoop` 60s quét Quota xong, các số liệu % và thanh bar trên thẻ card cập nhật tức thì; không có hiện tượng giật khung hình (zero layout jank), không reset thanh cuộn.
2. **AC-02 (Window Resize Freedom):** Người dùng có thể kéo thu nhỏ cửa sổ tự do xuống tới `480 x 360 px`. Khi Quota tự động refresh sau 60s, cửa sổ phải **đứng yên 100%**, tuyệt đối không tự ý phóng to trở lại kích thước cũ.
3. **AC-03 (Window Size Persistence):** Kích thước cửa sổ sau khi chỉnh được lưu vào `accounts_store/.ui_preferences.json` và khôi phục chuẩn xác khi khởi động lại ứng dụng.
4. **AC-04 (Responsive Card Width):** Khi kéo dãn cửa sổ rộng ra, 2 cột `5h` và `Week` trong thẻ Card tự động mở rộng cân đối, không để lại khoảng trống bất hợp lý.
5. **AC-05 (Build & Test Discipline):**
   - Chạy `go test -tags gui ./internal/ui/...` đạt **100% PASS**.
   - Chạy `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` đạt **Exit code = 0**.
