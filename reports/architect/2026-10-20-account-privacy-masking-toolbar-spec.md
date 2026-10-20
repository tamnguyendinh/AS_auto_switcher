# Đặc Tả Kiến Trúc & Kỹ Thuật: Chế Độ Ẩn Tài Khoản & Chuẩn Hóa Phân Cách Thanh Công Cụ Hàng 2 (Account Privacy Masking & Toolbar Layout Specification)

- **Ngày lập:** 2026-10-20
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Chỉ đạo trực tiếp và đã được phê duyệt từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261020-ACCOUNT-PRIVACY-MASKING-TOOLBAR`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

1. **PHÂN TÁCH TRIỆT ĐỂ PRESENTATION FILTER VÀ BACKEND MODEL (VIEW-ONLY FILTER):**
   - Chế độ "Ẩn tài khoản" (Account Privacy / Masking Mode) **CHỈ LÀ BỘ LỌC HIỂN THỊ Ở TẦNG VIEW**.
   - Tuyệt đối không thay đổi kiểu dữ liệu `Email string` trong Model, Account Store, Quota Cache, Chrome Scanner, Telemetry Loop hay Hot-swap Switcher.
   - Mọi hoạt động nghiệp vụ ngầm (quét Chrome, đổi session, fetch quota, gọi RPC Antigravity) **100% TIẾP TỤC SỬ DỤNG EMAIL THẬT NGUYÊN BẢN**.
2. **QUY TẮC CHE EMAIL GỌN GÀNG TỐI GIẢN (KISS MASKING RULE):**
   - Chỉ giữ lại đúng **1 ký tự đầu** của username + `***` + `@` + `tên miền`.
   - Ví dụ: `tamnguyen@gmail.com` $\rightarrow$ `t***@gmail.com`.
   - Giữ nguyên tooltip hover: khi rê chuột vào email đã che, popup tooltip vẫn hiển thị email thật nguyên bản.
3. **BẢO TỒN NGUYÊN VẸN CÁC THÔNG SỐ HÌNH HỌC ĐÃ KHÓA CỨNG:**
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`** (Compact Mode).
   - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `newSymmetricGridWrap`.
   - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
   - Cập nhật In-place reactive (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. BỐ CỤC THỊ GIÁC & QUY CÁCH HÌNH HỌC (ROW 2 RIGHT CLUSTER)

### 1.1 Bản Vẽ Bố Cục Tổng Thể Hàng 2
```text
┌───────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Hoạt động | PID 18660 | Port 62220  [Danh sách...] [Làm mới...] [📌]                              │
│                                           Ẩn tài khoản: [ ○ ]  │  Tự động: [ ● ]  │  [ ☀️ ▼ ]     │
└───────────────────────────────────────────────────────────────────────────────────────────────────┘
  ◄───────────────── Cụm bên trái (~510px) ─────────────────►   ◄───── Cụm điều khiển phải (~290px) ─────►
```

### 1.2 Chi Tiết Cụm Điều Khiển Phía Phải (Row 2 Right Cluster)
Dàn hàng ngang gồm 5 khối chức năng (7 phần tử Canvas):

```text
┌─────────────────────┐   ┌───┐   ┌─────────────────┐   ┌───┐   ┌────────────┐
│ Ẩn tài khoản: [ ○ ] │   │ │ │   │ Tự động:  [ ● ] │   │ │ │   │  [ ☀️ ▼ ]  │
└─────────────────────┘   └───┘   └─────────────────┘   └───┘   └────────────┘
     (Khối 1: ~105px)    (Div 1)   (Khối 2: ~88px)     (Div 2)  (Khối 3: 58px)
```

1. **Khối 1 — Điều khiển Ẩn tài khoản:**
   - Label: `"Ẩn tài khoản:"` (Font 10pt Bold, widget.LabelWithStyle).
   - Capsule Toggle: Kích thước cố định $23 \times 12\text{ px}$.
     - *Trạng thái OFF (Tắt):* Nền ray màu xám mờ (`palette.BorderSubtle`), nút tròn nằm bên trái ($x=2, y=2$).
     - *Trạng thái ON (Bật):* Nền ray màu xanh thương hiệu (`palette.Primary`), nút tròn nằm bên phải ($x=13, y=2$).
2. **Divider 1 — Thanh phân cách dọc 1:**
   - Đối tượng đồ họa: `canvas.Rectangle`, bọc trong `container.NewGridWrap(fyne.NewSize(1, 14), divRect)` nhằm đảm bảo Layout tính đúng kích thước tối thiểu rộng $1\text{ px}$, cao $14\text{ px}$, màu `palette.BorderSubtle` theo theme hiện hành.
3. **Khối 2 — Điều khiển Tự động đổi tài khoản:**
   - Label: `"Tự động:"` (Font 10pt Bold, widget.LabelWithStyle).
   - Capsule Toggle: Kích thước cố định $23 \times 12\text{ px}$ (sử dụng cùng cấu trúc `CapsuleToggle`).
4. **Divider 2 — Thanh phân cách dọc 2:**
   - Cùng thông số với Divider 1: `container.NewGridWrap(fyne.NewSize(1, 14), divRect)`, rộng $1\text{ px}$, cao $14\text{ px}$, màu `palette.BorderSubtle`.
5. **Khối 3 — Theme Select Dropdown:**
   - Kích thước cố định $58 \times 20\text{ px}$ gồm 3 icon (`🌙`, `☀️`, `☕`).

---

## 2. QUY TẮC CHE EMAIL & XỬ LÝ CHUỖI (MASKING SPECIFICATION)

### 2.1 Thuật toán Che Email (`MaskEmail(email string) string`)
```text
Đầu vào: Email chuỗi (e.g. "tamnguyen@gmail.com")
Bước 1: strings.TrimSpace(email).
Bước 2: Tìm vị trí ký tự '@'.
        Nếu không tìm thấy hoặc ở vị trí đầu tiên (atIdx <= 0):
        -> Trả về email nguyên bản (phòng thủ chuỗi bất thường).
Bước 3: Tách username = email[:atIdx], domain = email[atIdx:] (bao gồm cả ký tự '@').
Bước 4: Chuyển username thành mảng []rune (an toàn với UTF-8 đa byte).
        Nếu len(runes) == 0 -> Trả về email nguyên bản.
Bước 5: Kết quả = string(runes[0]) + "***" + domain.
```

### 2.2 Bảng Kiểm Thử Xử Lý Chuỗi (Test Matrix)
| Đầu vào Email | Đầu ra khi `mask_accounts = true` | Đầu ra khi `mask_accounts = false` | Ghi chú |
|---|---|---|---|
| `tamnguyen@gmail.com` | `t***@gmail.com` | `tamnguyen@gmail.com` | Dạng chuẩn thông dụng |
| `developer.ai@antigravity.io` | `d***@antigravity.io` | `developer.ai@antigravity.io` | Dạng có dấu chấm |
| `a@gmail.com` | `a***@gmail.com` | `a@gmail.com` | Username chỉ có 1 ký tự |
| `admin+tag@sub.domain.com` | `a***@sub.domain.com` | `admin+tag@sub.domain.com` | Domain đa cấp, có alias |
| `invalid-email-string` | `invalid-email-string` | `invalid-email-string` | Chuỗi không hợp lệ (không crash) |
| `""` (Rỗng) | `""` | `""` | Rỗng an toàn |

---

## 3. CƠ CHẾ VẬN HÀNH & LUỒNG DỮ LIỆU (OPERATIONAL MECHANICS)

### 3.1 Khởi Động & Lưu Cấu Hình (Persistence Lifecycle)
- **Vị trí lưu trữ:** `accounts_store/.ui_preferences.json` (định nghĩa bởi hằng số `DefaultUIPreferencesPath`).
- **Mô hình JSON:**
  ```json
  {
    "theme": "dark",
    "mask_accounts": true
  }
  ```
- **Quy tắc nạp:**
  - Khi khởi động `NewAppWithFyne()` $\rightarrow$ gọi `LoadUIPreferences(a.preferencesPath)`.
  - Gán `a.maskAccounts = pref.MaskAccounts`. Nếu không có trường này trong file cũ, Go unmarshal mặc định là `false`.
- **Quy tắc ghi:** Khi người dùng click toggle $\rightarrow$ ghi bất đồng bộ (non-blocking goroutine) qua `SaveUIPreferences(...)` để không bao giờ làm block UI thread.

### 3.2 Luồng Click Toggle & Phản Ứng Tức Thì (In-Place Reactive Update)
```text
[Người dùng click nút gạt Ẩn tài khoản]
                   │
                   ▼
1. Toggle UI đổi vị trí thumb và màu nền ray (Animation mượt mà).
                   │
                   ▼
2. Kích hoạt Callback: a.SetMaskAccounts(isOn)
   ├─ a.mu.Lock(): Cập nhật a.maskAccounts = isOn.
   ├─ Ghi đĩa non-blocking: go SaveUIPreferences(...).
   └─ Duyệt mảng in-place cards trong RAM (Zero Layout Jank):
      for _, card := range a.accountCards {
          if card != nil {
              card.SetMasked(isOn)
          }
      }
                   │
                   ▼
3. Phản hồi trạng thái lên Status Bar (tự ẩn sau 2 giây):
   - Nếu BẬT: "🔒 Đã ẩn thông tin tài khoản"
   - Nếu TẮT: "🔓 Đã hiển thị đầy đủ tài khoản"
```

### 3.3 Hành Vi Tooltip Hover Khi Rê Chuột
- `emailLabelWithTooltip` lưu trữ thuộc tính `fullText: string` cố định luôn là **email thật**.
- Khi `mask_accounts = true`, nhãn hiển thị thông qua `SetText` là `t***@gmail.com`.
- Khi người dùng rê chuột vào (`MouseIn`), popup tooltip bung ra đọc từ `fullText` $\rightarrow$ **hiển thị đầy đủ email thật**.
- Khi rời chuột (`MouseOut`), popup lập tức biến mất.

### 3.4 Bảo Vệ Thanh Trạng Thái (Status Bar Protection)
- Trong các luồng `SwitchAccount()`, `handleAutomationStatus()`, khi hiển thị thông báo chuyển đổi:
  - Nếu `a.IsMaskAccounts() == true`: Tự động thay thế địa chỉ email bằng `MaskEmail(targetEmail)`.
  - *Ví dụ:* `⏳ Đang chuyển sang t***@gmail.com...` và `✅ Đã chuyển sang t***@gmail.com`.

---

## 4. MA TRẬN BẢO TOÀN INVARIANT (SYSTEM INVARIANTS)

1. **Zero Layout Jank:** Việc gạt bật/tắt chế độ ẩn tuyệt đối không được gọi `forceRebuildCards()` hay xóa/thêm đối tượng Canvas trong `accountListContainer`. Chỉ thực hiện set text trên các widget `emailLabel` đã có sẵn thông qua `card.SetMasked(isOn)`.
2. **Theme Synchronization:** Khi người dùng đổi theme (`🌙`, `☀️`, `☕`), cả 2 Capsule Toggle và 2 thanh phân cách `|` phải được cập nhật màu viền/màu nền ăn theo `ThemePalette` mới thông qua `SetPalette(palette)`.
3. **Thread Safety:** Truy cập trạng thái `maskAccounts` trong `App` phải được bảo vệ bằng `a.mu.RLock()` / `a.mu.Lock()`.
4. **Backward Compatibility:** Không thay đổi bất kỳ signature hiện có của `NewAccountCardWithQuota` hay `NewAccountCardWidgetWithQuota` để tránh làm gãy 20+ unit test có sẵn.

---

## 5. BẢN VẼ THI CÔNG CHI TIẾT DÀNH CHO CODER (DETAILED IMPLEMENTATION BLUEPRINT)

Mục này cung cấp vị trí file, khai báo kiểu dữ liệu, hàm và mã nguồn mẫu chính xác để Coder triển khai mà không cần phỏng đoán.

### 5.1 Cập nhật `internal/ui/themes.go`
Mở rộng struct `UIPreferences` để thêm trường `MaskAccounts`:

```go
// UIPreferences stores persistent user interface settings.
type UIPreferences struct {
	Theme        string `json:"theme"`
	MaskAccounts bool   `json:"mask_accounts"`
}
```

### 5.2 Cập nhật `internal/ui/widgets.go`
#### A. Hàm `MaskEmail`
Thêm hàm thuần túy (pure function) xử lý che email UTF-8 an toàn:

```go
// MaskEmail transforms an email like "user@domain.com" into "u***@domain.com".
// If email format is invalid or has no @, original string is returned unchanged.
func MaskEmail(email string) string {
	trimmed := strings.TrimSpace(email)
	atIdx := strings.Index(trimmed, "@")
	if atIdx <= 0 {
		return trimmed
	}
	username := trimmed[:atIdx]
	domain := trimmed[atIdx:]
	runes := []rune(username)
	if len(runes) == 0 {
		return trimmed
	}
	return string(runes[0]) + "***" + domain
}
```

#### B. Phương thức `SetMasked` trên `AccountCardWidget`
Thêm phương thức cập nhật in-place nhãn email của thẻ tài khoản:

```go
// SetMasked toggles the email label between masked and full text in-place.
func (w *AccountCardWidget) SetMasked(masked bool) {
	if w == nil || w.emailLabel == nil {
		return
	}
	if masked {
		w.emailLabel.SetText(MaskEmail(w.Email))
	} else {
		w.emailLabel.SetText(w.Email)
	}
}
```
*(Lưu ý: Giữ nguyên 100% signature của `NewAccountCardWidgetWithQuota` và `NewAccountCardWithQuota`).*

### 5.3 Cập nhật `internal/ui/window.go`
#### A. Chuẩn hóa `CapsuleToggle`
Hiện struct `AutoSwitchToggle` đã được hoàn thiện. Chuẩn hóa thành `CapsuleToggle` dùng chung bằng cách alias type:

```go
// CapsuleToggle represents a 23x12 capsule pill toggle switch.
type CapsuleToggle = AutoSwitchToggle

// newCapsuleToggle creates a standardized capsule toggle.
func newCapsuleToggle(initial bool, onChange func(bool), palette ThemePalette) *CapsuleToggle {
	return newAutoSwitchToggle(initial, onChange, palette)
}
```

#### B. Tạo và bố trí cụm Row 2 Right Cluster
Trong hàm `createWindowContent` hoặc nơi khởi tạo hàng 2 (thay thế cụm `rightCluster` cũ):

```go
// Label Ẩn tài khoản
lblMask := widget.NewLabelWithStyle("Ẩn tài khoản:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
headerRow2MaskLabel = lblMask

// Toggle Ẩn tài khoản
a.maskAccountsToggle = newCapsuleToggle(a.maskAccounts, func(isOn bool) {
	a.SetMaskAccounts(isOn)
}, palette)

// Divider 1 (Bắt buộc bọc GridWrap 1x14 px để không bị MinSize 0x0)
a.row2Divider1 = canvas.NewRectangle(palette.BorderSubtle)
div1Wrap := container.NewGridWrap(fyne.NewSize(1, 14), a.row2Divider1)

// Label Tự động
lblAuto := widget.NewLabelWithStyle("Tự động:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
headerRow2AutoLabel = lblAuto

// Toggle Tự động
a.autoSwitchToggle = newCapsuleToggle(a.autoSwitchEnabled, func(isOn bool) {
	a.SetAutoSwitchEnabled(isOn)
}, palette)

// Divider 2 (Bắt buộc bọc GridWrap 1x14 px)
a.row2Divider2 = canvas.NewRectangle(palette.BorderSubtle)
div2Wrap := container.NewGridWrap(fyne.NewSize(1, 14), a.row2Divider2)

// Bố cục rightCluster 7 phần tử
rightCluster := container.New(row2RightClusterLayout{},
	lblMask,
	a.maskAccountsToggle,
	div1Wrap,
	lblAuto,
	a.autoSwitchToggle,
	div2Wrap,
	themeSelectWrap,
)
```

#### C. Biến toàn cục & Phương thức trợ giúp kiểm thử (Exported Helper Methods)
```go
var headerRow2MaskLabel *widget.Label

// MaskAccountsLabelWidget returns the header row 2 mask accounts label widget.
func (a *App) MaskAccountsLabelWidget() *widget.Label {
	return headerRow2MaskLabel
}

// MaskAccountsToggleWidget returns the header row 2 mask accounts capsule toggle.
func (a *App) MaskAccountsToggleWidget() *CapsuleToggle {
	return a.maskAccountsToggle
}

// Row2Dividers returns the two vertical divider canvas rectangles.
func (a *App) Row2Dividers() (*canvas.Rectangle, *canvas.Rectangle) {
	return a.row2Divider1, a.row2Divider2
}
```

### 5.4 Cập nhật `internal/ui/app.go`
#### A. Bổ sung trường trong struct `App`
```go
type App struct {
	// ... các trường hiện có ...
	maskAccounts       bool
	maskAccountsToggle *CapsuleToggle
	row2Divider1       *canvas.Rectangle
	row2Divider2       *canvas.Rectangle
}
```

#### B. Khởi tạo trạng thái ban đầu (`NewAppWithFyne`)
Trong `NewAppWithFyne(...)`:
```go
pref, err := LoadUIPreferences(a.preferencesPath)
if err == nil && pref != nil {
	a.maskAccounts = pref.MaskAccounts
}
```

#### C. Các phương thức nghiệp vụ trên `App`
```go
// IsMaskAccounts reports whether account privacy masking is currently active.
func (a *App) IsMaskAccounts() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.maskAccounts
}

// SetMaskAccounts updates the privacy masking state and updates all cards in-place.
func (a *App) SetMaskAccounts(mask bool) {
	a.mu.Lock()
	a.maskAccounts = mask
	a.mu.Unlock()

	if a.maskAccountsToggle != nil && a.maskAccountsToggle.IsOn() != mask {
		a.maskAccountsToggle.SetOn(mask)
	}

	// Update cards in-place (Zero layout jank)
	a.mu.RLock()
	cards := append([]*AccountCardWidget(nil), a.accountCards...)
	a.mu.RUnlock()
	for _, card := range cards {
		if card != nil {
			card.SetMasked(mask)
		}
	}

	// Non-blocking disk persistence
	go func() {
		pref, err := LoadUIPreferences(a.preferencesPath)
		if err != nil || pref == nil {
			pref = &UIPreferences{Theme: a.currentTheme}
		}
		pref.MaskAccounts = mask
		_ = SaveUIPreferences(a.preferencesPath, pref)
	}()

	// Status bar transient feedback
	if mask {
		a.showStatusBar("🔒 Đã ẩn thông tin tài khoản")
	} else {
		a.showStatusBar("🔓 Đã hiển thị đầy đủ tài khoản")
	}
	a.autoHideStatusBar(2 * time.Second)
}

// ToggleMaskAccounts flips the account privacy masking state.
func (a *App) ToggleMaskAccounts() {
	a.SetMaskAccounts(!a.IsMaskAccounts())
}
```

#### D. Bảo toàn Mask khi Render Card mới (`UpdateOverview` / `updateCards`)
Mỗi khi khởi tạo card mới (`NewAccountCardWidgetWithQuota`):
```go
cardWidget := NewAccountCardWidgetWithQuota(acc, quotaData, palette, a.SwitchAccount, a.SaveSession, a.RefreshSingleAccountQuota)
if a.maskAccounts {
	cardWidget.SetMasked(true)
}
a.accountCards = append(a.accountCards, cardWidget)
```

#### E. Đồng bộ Theme (`SetPalette` / `SetTheme`)
Trong logic cập nhật theme:
```go
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
```

#### F. Che email trên Status Bar (`SwitchAccount`)
Trong `SwitchAccount(targetEmail string)`:
```go
displayEmail := targetEmail
if a.IsMaskAccounts() {
	displayEmail = MaskEmail(targetEmail)
}
a.showStatusBar(fmt.Sprintf("⏳ Đang chuyển sang %s...", displayEmail))
```
Và khi chuyển thành công:
```go
a.showStatusBar(fmt.Sprintf("✅ Đã chuyển sang %s", displayEmail))
```

---

## 6. MA TRẬN TIÊU CHÍ CHẤP THUẬN & TEST SPEC (ACCEPTANCE CRITERIA)

Coder cần đảm bảo pass toàn bộ 5 nhóm Acceptance Tests trong `internal/ui/ui_test.go`:

1. **`TestMaskEmail_AlgorithmConformance`**:
   - `tamnguyen@gmail.com` $\rightarrow$ `t***@gmail.com`
   - `developer.ai@antigravity.io` $\rightarrow$ `d***@antigravity.io`
   - `a@gmail.com` $\rightarrow$ `a***@gmail.com`
   - `admin+tag@sub.domain.com` $\rightarrow$ `a***@sub.domain.com`
   - `""` và `"invalid-format"` giữ nguyên chuỗi, không hoảng loạn (panic-free).
2. **`TestArchitectSpec_Row2Layout_WithDividers`**:
   - `lblMask` có text `"Ẩn tài khoản:"`, font Bold.
   - `maskAccountsToggle` có MinSize $23 \times 12\text{ px}$.
   - Cả 2 Divider có MinSize đúng $1 \times 14\text{ px}$ (nhờ `container.NewGridWrap`).
   - `rightCluster` chứa đúng 7 đối tượng Canvas theo đúng thứ tự:
     `[lblMask, maskToggle, div1Wrap, lblAuto, autoSwitchToggle, div2Wrap, themeSelectWrap]`.
3. **`TestArchitectSpec_MaskAccountsToggle_Behavior`**:
   - Ban đầu `maskAccounts = false`, `card.emailLabel.Text == "tamnguyen@gmail.com"`.
   - Click toggle $\rightarrow$ `maskAccounts = true`, `card.emailLabel.Text == "t***@gmail.com"`.
   - Kiểm tra tooltip `fullText` vẫn giữ nguyên `"tamnguyen@gmail.com"`.
   - Click toggle lần nữa $\rightarrow$ `card.emailLabel.Text == "tamnguyen@gmail.com"`.
   - Không có phần tử card nào bị recreate (con trỏ `card` giữ nguyên).
4. **`TestArchitectSpec_MaskAccounts_Persistence`**:
   - Khởi tạo app với file `.ui_preferences.json` có `"mask_accounts": true` $\rightarrow$ app tự động bật toggle và che email ngay từ khi mở.
   - Click toggle $\rightarrow$ file `.ui_preferences.json` được cập nhật tương ứng.
5. **`TestArchitectSpec_MaskAccounts_ThemeSync`**:
   - Đổi theme sang `ThemeLight`, `ThemeWarm`, `ThemeDark` $\rightarrow$ `div1.FillColor`, `div2.FillColor`, và màu `palette` của cả 2 toggle cập nhật đúng `palette.BorderSubtle` của theme tương ứng.
