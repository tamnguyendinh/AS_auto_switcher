# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Tính năng: Nút Dropdown i18n Chuyển đổi Ngôn ngữ (VN / EN) & Ghi nhớ Cấu hình Bền vững

- **Mã tài liệu:** `rp_system-architect_261024_130500_by_antigravity_i18n_dropdown_language_switcher_and_preferences_persistence`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-24
- **Trạng thái:** PRODUCTION-READY BLUEPRINT (Turnkey Specification for Coder)
- **Mục tiêu:** Cung cấp bản thiết kế kiến trúc chuẩn mực 100% cho tính năng đa ngôn ngữ (i18n): định vị điều khiển dropdown VN/EN trên thanh tiêu đề cạnh nút thu nhỏ, cơ chế lưu bền vững vào `ui_preferences.json`, mô hình từ điển kiểu mạnh (Typed Dictionary), và giải pháp cập nhật giao diện in-place (Zero-Restart, No-Flicker).

---

## 1. TỔNG QUAN YÊU CẦU & BỐI CẢNH (EXECUTIVE SUMMARY)

### 1.1. Hiện trạng
- Ứng dụng `Antigravity Multi-Account Switcher` hiện có giao diện hiển thị hoàn toàn bằng Tiếng Việt cố định (hardcoded strings) trên các nút bấm, nhãn, thanh trạng thái và hộp thoại modal.
- Tài liệu dự án đã chuẩn hóa song ngữ (`README_VN.md` và `README_EN.md`), nhưng giao diện đồ họa Fyne GUI chưa hỗ trợ người dùng quốc tế hoặc người dùng muốn sử dụng Tiếng Anh.
- Cấu hình người dùng (`ui_preferences.json`) hiện đã quản lý `theme`, `mask_accounts`, `auto_switch`, `always_on_top` nhưng chưa có trường dành cho `language`.

### 1.2. Mục tiêu sau khi nâng cấp
1. **Thêm điều khiển i18n Dropdown:** Đặt một dropdown nhỏ gọn (`VN` / `EN`) trên thanh tiêu đề `row1Header`, nằm ngay **bên trái nút thu nhỏ (`btnMin` "—")**.
2. **Ghi nhớ cấu hình bền vững:** Khi người dùng chọn ngôn ngữ, hệ thống lưu ngay vào `logs/ui_preferences.json` và nạp lại chính xác cho các lần khởi động tiếp theo.
3. **Cập nhật tức thì (In-Place Live Refresh):** Chuyển đổi ngôn ngữ diễn ra ngay lập tức trên màn hình đang chạy, không yêu cầu khởi động lại ứng dụng và không gây giật lag / rebuild window.
4. **Bảo toàn 100% Tương thích ngược:** Ngôn ngữ mặc định là `"vi"`. Toàn bộ các test suite kiểm tra chuỗi Tiếng Việt hiện tại tiếp tục vượt qua mà không cần sửa đổi.

---

## 2. NGUYÊN TẮC BẤT BIẾN & RÀNG BUỘC KIẾN TRÚC (INVARIANTS & CONSTRAINTS)

1. **Header Layout & Geometry Invariant:**
   - Điều khiển Dropdown ngôn ngữ phải nằm trong khối `windowControls` ở mép phải của `row1Header`.
   - Thứ tự từ trái qua phải của cụm điều khiển cửa sổ:
     $$\mathbf{[Dropdown\ VN/EN]} \longrightarrow \mathbf{[—\ Thu\ nhỏ]} \longrightarrow \mathbf{[□\ Phóng\ to]} \longrightarrow \mathbf{[✕\ Đóng]}$$
   - Chiều cao điều khiển dropdown được bọc trong `container.NewGridWrap(fyne.NewSize(54, 22), ...)` hoặc `(56, 22)`, đảm bảo không vượt quá chiều cao tiêu chuẩn 26px của `row1Header` và không làm vỡ baseline căn chỉnh.

2. **Typed Catalog Invariant (Nguyên tắc Từ điển Kiểu mạnh):**
   - Nghiêm cấm sử dụng map chuỗi lỏng lẻo (`map[string]string`) làm lõi từ điển để tránh lỗi runtime panic do sai chính tả (typo) hoặc thiếu khóa.
   - Toàn bộ chuỗi ngôn ngữ phải được khai báo dưới dạng struct tĩnh `Dictionary` với các trường cụ thể, có unit test tự động xác minh 100% độ phủ giữa `VI` và `EN`.

3. **Zero-Restart & In-Place Update Invariant:**
   - Chuyển đổi ngôn ngữ không được phép gọi `win.SetContent(...)` hoặc khởi tạo lại cây widget.
   - Phải sử dụng cơ chế cập nhật in-place: cập nhật trực tiếp thuộc tính chuỗi hoặc gọi `.SetText()` trên các widget hiện hữu và gọi `.Refresh()` trên Canvas.

4. **Backward Compatibility & Default Fallback Invariant:**
   - Khi file `ui_preferences.json` chưa có trường `language` (phiên bản cũ) hoặc giá trị không hợp lệ (không phải `"vi"` hay `"en"`), hệ thống bắt buộc fallback về `"vi"`.
   - Tag JSON của trường cấu hình phải là `json:"language,omitempty"`.

5. **Thread-Safe & Non-Blocking Persistence:**
   - Quá trình ghi cấu hình ngôn ngữ xuống đĩa diễn ra trong goroutine nền qua `prefSaveMu`, sử dụng cơ chế atomic write (ghi file tạm `.ui_preferences_tmp_*.json` rồi rename) để ngăn ngừa nghẽn luồng đồ họa chính.

---

## 3. THIẾT KẾ KIẾN TRÚC TỪNG TẦNG (LAYER-BY-LAYER ARCHITECTURE)

```text
 ┌────────────────────────────────────────────────────────┐
 │                      TẦNG GIAO DIỆN (UI)                │
 │  • row1Header: [Dropdown VN/EN] [—] [□] [✕]             │
 │  • a.langSelect (widget.Select: "VN", "EN")            │
 │  • a.applyLanguage(locale) -> cập nhật in-place widget │
 └───────────────────────────┬────────────────────────────┘
                             │
                             ▼ (Gọi gán ngôn ngữ & nạp từ điển)
 ┌────────────────────────────────────────────────────────┐
 │                   TẦNG NGÔN NGỮ (i18n)                 │
 │  • internal/i18n/locale.go                             │
 │  • LocaleVI = "vi", LocaleEN = "en"                    │
 │  • Dictionary struct (Typed Compile-Time Safe Strings) │
 │  • GetDictionary(loc), Current(), SetLocale(loc)       │
 └───────────────────────────┬────────────────────────────┘
                             │
                             ▼ (Ghi bền vững cấu hình)
 ┌────────────────────────────────────────────────────────┐
 │                 TẦNG CẤU HÌNH (PREFERENCES)            │
 │  • UIPreferences.Language ("vi" | "en")                │
 │  • LoadUIPreferences / SaveUIPreferences               │
 │  • logs/ui_preferences.json                            │
 └────────────────────────────────────────────────────────┘
```

---

### 3.1. Tầng Đa ngôn ngữ: `internal/i18n`

Khởi tạo package mới hoàn toàn tách biệt: `internal/i18n/locale.go` và `internal/i18n/dictionary.go`.

#### Định nghĩa kiểu & Struct từ điển:
```go
package i18n

type Locale string

const (
    LocaleVI Locale = "vi"
    LocaleEN Locale = "en"
)

type Dictionary struct {
    // Header & Window
    AppTitle            string
    LangDropdownLabel   string

    // Toolbar Buttons & Controls
    BtnAddAccount       string
    BtnRefreshList      string
    BtnRefreshQuota     string
    BtnAlwaysOnTop      string
    LblMaskAccounts     string
    LblAutoSwitch       string

    // Account Card
    BadgeActive         string
    BadgeSwitch         string
    BadgeNeedOAuth      string
    BtnDeleteAccount    string
    DeleteConfirmTitle  string
    DeleteConfirmMsg    string

    // Server & Connection Status
    StatusServerActive  string
    StatusConnecting    string
    StatusNoCandidate   string
    StatusAutoSwitchOn  string
    StatusAutoSwitchOff string

    // Modals & Dialogs
    ModalAddTitle       string
    ModalClose          string
    ModalEmailLabel     string
    ModalSave           string
    ModalCancel         string

    // Tray Menu
    TrayOpen            string
    TrayExit            string
}
```

#### Bảng dữ liệu song ngữ chuẩn hóa:

| Khóa Từ Điển | Tiếng Việt (`LocaleVI`) | English (`LocaleEN`) |
|---|---|---|
| `AppTitle` | Antigravity Multi-Account Switcher | Antigravity Multi-Account Switcher |
| `BtnAddAccount` | Thêm | Add |
| `BtnRefreshList` | Làm mới danh sách | Refresh List |
| `BtnRefreshQuota` | Làm mới Quota | Refresh Quota |
| `LblMaskAccounts` | Ẩn tài khoản: | Mask Accounts: |
| `LblAutoSwitch` | Tự động: | Auto Switch: |
| `BadgeActive` | [✔ Dùng] | [✔ Active] |
| `BadgeSwitch` | [Chuyển] | [Switch] |
| `BadgeNeedOAuth` | [Cần OAuth] | [Need OAuth] |
| `StatusConnecting` | Đang kết nối... | Connecting... |
| `StatusServerActive` | Antigravity: Hoạt động | Antigravity: Active |
| `StatusNoCandidate` | ⚠️ Không có tài khoản để chuyển. | ⚠️ No accounts available to switch. |
| `StatusAutoSwitchOn` | Tự động chuyển tài khoản: BẬT | Auto-switch accounts: ON |
| `StatusAutoSwitchOff`| Tự động chuyển tài khoản: TẮT | Auto-switch accounts: OFF |
| `ModalAddTitle` | Thêm tài khoản | Add Account |
| `ModalClose` | Đóng | Close |
| `ModalEmailLabel` | Email tài khoản | Account Email |
| `ModalSave` | Lưu | Save |
| `ModalCancel` | Hủy | Cancel |
| `TrayOpen` | Mở giao diện | Open Switcher |
| `TrayExit` | Thoát | Exit |

---

### 3.2. Tầng Cấu hình Bền vững: `internal/ui/themes.go`

1. **Cập nhật `UIPreferences`:**
   ```go
   type UIPreferences struct {
       Theme        string `json:"theme"`
       MaskAccounts bool   `json:"mask_accounts"`
       AutoSwitch   bool   `json:"auto_switch"`
       AlwaysOnTop  *bool  `json:"always_on_top,omitempty"`
       Language     string `json:"language,omitempty"` // "vi" hoặc "en"
   }
   ```

2. **Cập nhật `LoadUIPreferences`:**
   ```go
   // Chuẩn hóa language: nếu rỗng hoặc không hợp lệ -> fallback về "vi"
   if pref.Language != "en" && pref.Language != "vi" {
       pref.Language = "vi"
   }
   ```

3. **Cập nhật `SaveUIPreferences`:**
   - Trường `Language` tự động serialize vào JSON qua tag `json:"language,omitempty"`.

---

### 3.3. Tầng Giao diện Người dùng: `internal/ui/window.go` & `internal/ui/app.go`

#### 1. Định vị và Khởi tạo trong `internal/ui/window.go`:
```go
// Khởi tạo Language Select Dropdown
a.langSelect = widget.NewSelect([]string{"VN", "EN"}, func(selected string) {
    targetLocale := "vi"
    if selected == "EN" {
        targetLocale = "en"
    }
    a.SetLanguage(targetLocale)
})

// Thiết lập giá trị hiển thị ban đầu
if a.currentLocale == "en" {
    a.langSelect.SetSelected("EN")
} else {
    a.langSelect.SetSelected("VN")
}

// Bọc GridWrap với kích thước 54x22 px
langSelectWrap := container.NewGridWrap(fyne.NewSize(54, 22), a.langSelect)

// Ghép vào windowControls: nằm ngay mép trái nút thu nhỏ btnMin
windowControls := container.NewHBox(langSelectWrap, btnMin, btnMax, btnClose)
```

#### 2. Xử lý Logic Cập nhật In-Place trong `internal/ui/app.go`:
```go
// SetLanguage chuyển đổi ngôn ngữ, cập nhật in-place widget và ghi đĩa
func (a *App) SetLanguage(locale string) {
    if locale != "en" && locale != "vi" {
        locale = "vi"
    }

    a.mu.Lock()
    if a.currentLocale == locale {
        a.mu.Unlock()
        return
    }
    a.currentLocale = locale
    theme := a.currentTheme
    mask := a.maskAccounts
    auto := a.autoSwitchEnabled
    top := a.alwaysOnTop
    path := a.preferencesPath
    a.mu.Unlock()

    // 1. Cập nhật trạng thái global i18n
    i18n.SetCurrentLocale(i18n.Locale(locale))
    d := i18n.Current()

    // 2. Cập nhật in-place trên Canvas UI (Main Thread)
    fyne.Do(func() {
        if a.addAccountBtn != nil {
            a.addAccountBtn.SetText(d.BtnAddAccount)
        }
        if a.refreshBtn != nil {
            a.refreshBtn.SetText(d.BtnRefreshList)
        }
        if a.quotaRefreshAllBtn != nil {
            a.quotaRefreshAllBtn.SetText(d.BtnRefreshQuota)
        }
        if headerRow2MaskLabel != nil {
            headerRow2MaskLabel.SetText(d.LblMaskAccounts)
        }
        if headerRow2AutoLabel != nil {
            headerRow2AutoLabel.SetText(d.LblAutoSwitch)
        }
        if a.langSelect != nil {
            expected := "VN"
            if locale == "en" {
                expected = "EN"
            }
            if a.langSelect.Selected != expected {
                a.langSelect.SetSelected(expected)
            }
        }
        // Cập nhật các thẻ tài khoản đang hiển thị
        a.refreshAccountCardsLanguage()
    })

    // 3. Ghi bất đồng bộ xuống ui_preferences.json
    go func() {
        a.prefSaveMu.Lock()
        defer a.prefSaveMu.Unlock()

        pref, err := LoadUIPreferences(path)
        if err != nil || pref == nil {
            pref = &UIPreferences{
                Theme:        theme,
                MaskAccounts: mask,
                AutoSwitch:   auto,
                AlwaysOnTop:  &top,
                Language:     locale,
            }
        } else {
            pref.Language = locale
        }
        _ = SaveUIPreferences(path, pref)
    }()
}
```

#### 3. Cập nhật Thẻ Tài khoản: `internal/ui/widgets.go`:
- Cung cấp method `(w *AccountCardWidget) UpdateLanguage(d *i18n.Dictionary)` để cập nhật nhãn nút `[Chuyển]` / `[Switch]` hoặc `[✔ Dùng]` / `[✔ Active]` mà không làm gián đoạn ma trận Quota và Tier Badge.

---

## 4. KẾ HOẠCH KIỂM THỬ & NGĂN NGỪA HỒI QUY (TEST & REGRESSION PLAN)

1. **`internal/i18n/i18n_test.go`:**
   - `TestI18n_DictionaryCompleteness`: Sử dụng reflection để kiểm tra 100% các trường chuỗi trong `Dictionary` của cả `VI` và `EN` đều có giá trị khác rỗng `""`.
   - `TestI18n_LocaleSwitching`: Kiểm tra chuyển đổi qua lại giữa `vi` và `en` phản hồi đúng struct từ điển tương ứng.

2. **`internal/ui/themes_test.go`:**
   - `TestUIPreferences_LanguagePersistence`: Lưu preferences với `Language: "en"`, nạp lại xác nhận đúng `"en"`.
   - `TestUIPreferences_LanguageDefaultFallback`: Nạp preferences trống hoặc `Language: "invalid"` xác nhận tự động fallback về `"vi"`.

3. **`internal/ui/i18n_acceptance_test.go`:**
   - Kiểm tra vị trí điều khiển: `a.langSelect` nằm ngay cạnh trái của `btnMin`.
   - Kiểm tra phản ứng tức thì: Thay đổi từ `VN` sang `EN` kiểm tra text của `addAccountBtn` chuyển thành `"Add"`, `refreshBtn` thành `"Refresh List"`.
   - Chuyển ngược lại về `VN`, xác nhận text phục hồi thành `"Thêm"`, `"Làm mới danh sách"`.
   - Kiểm tra `ui_preferences.json` lưu giữ chính xác giá trị đã chọn.

---

## 5. BIÊN BẢN BÀN GIAO THỰC THI (CODER HANDOFF CHECKLIST)

Coder nhận bàn giao thực hiện lần lượt theo 5 bước tuần tự sau:
- [ ] **Bước 1:** Khởi tạo gói `internal/i18n/locale.go` và `internal/i18n/dictionary.go` cùng bộ unit test `i18n_test.go`.
- [ ] **Bước 2:** Bổ sung trường `Language` vào `UIPreferences` (`internal/ui/themes.go`) và cập nhật `LoadUIPreferences`.
- [ ] **Bước 3:** Khởi tạo `a.langSelect` trong `internal/ui/window.go` và đưa `langSelectWrap` vào trước `btnMin` trong `windowControls`.
- [ ] **Bước 4:** Xây dựng `a.SetLanguage` và cơ chế cập nhật in-place các widget (`addAccountBtn`, `refreshBtn`, `quotaRefreshAllBtn`, `AccountCardWidget.UpdateLanguage`).
- [ ] **Bước 5:** Nạp ngôn ngữ đã lưu khi khởi tạo `App.Init()`, chạy `go test ./internal/...` xác nhận 100% test pass.

---
*(Bản đặc tả đã được hoàn thiện sản xuất và sẵn sàng lưu vết tại kho lưu trữ).*
