# Đặc Tả Kiến Trúc & Chỉ Dẫn Kỹ Thuật Chi Tiết Cho Coder: Lưu Trữ Bền Vững Trạng Thái Nút Gạt Tự Động Trên Giao Diện
(Auto-Switch Toggle Persistence Specification — Complete Implementation Recipe)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-AUTO-SWITCH-TOGGLE-PERSISTENCE`
- **Trạng thái:** DRAFT (Trình Human Owner phê duyệt)
- **Đối tượng đọc:** Coder, Supervisor, QA

---

## I. MỤC TIÊU VÀ NGUYÊN TẮC CỐT LÕI (OBJECTIVES & INVARIANTS)

Bản đặc tả này chuẩn hóa cơ chế lưu trữ bền vững (*Persistence*) cho nút gạt **"Tự động"** (`autoSwitchToggle`) trên thanh công cụ giao diện (Row 2 Header), đồng bộ hoàn toàn với cơ chế lưu trữ của nút **"Ẩn tài khoản"** (`maskAccounts`) và danh sách chọn **"Theme"** vào tệp `logs/ui_preferences.json`.

### Các điều khoản kỷ luật sắt (Mandatory Invariants):
1. **LƯU TRỮ ĐỒNG BỘ 3 TRẠNG THÁI GIAO DIỆN (UNIFIED PREFERENCES INVARIANT):**
   - Tệp `logs/ui_preferences.json` là nơi duy nhất quản lý toàn bộ tùy chọn người dùng:
     1. `theme` (chủ đề giao diện: `"dark"`, `"light"`, `"warm"`).
     2. `mask_accounts` (trạng thái ẩn/hiện email: `true`/`false`).
     3. `auto_switch` (trạng thái tự động chuyển tài khoản: `true`/`false`).
2. **BẢO TỒN TRẠNG THÁI KHỞI ĐỘNG (STARTUP RESTORATION INVARIANT):**
   - Khi ứng dụng khởi động (`NewAppWithFyne`), giá trị `auto_switch` phải được nạp từ đĩa vào `a.autoSwitchEnabled`.
   - Nút gạt `autoSwitchToggle` trên giao diện phải hiển thị đúng trạng thái ON hoặc OFF tương ứng ngay khi mở app.
   - Bộ điều khiển tự động (`automation.AutomationController`) phải được kích hoạt `SetEnabled(true)` nếu `auto_switch` được lưu là `true`.
3. **KHÔNG GHI ĐÈ LẪN NHAU (CROSS-SETTING MUTUAL PRESERVATION):**
   - Khi thay đổi Theme hoặc gạt nút Ẩn tài khoản, giá trị `auto_switch` đã lưu trước đó không được phép bị mất hoặc bị ghi đè về `false`.
   - Ngược lại, khi gạt nút Tự động, `theme` và `mask_accounts` phải được giữ nguyên vẹn.
4. **GHI ĐĨA NON-BLOCKING (ZERO UI JANK):**
   - Thao tác ghi đĩa khi người dùng gạt nút phải chạy bất đồng bộ trong goroutine (như cách làm của `SetMaskAccounts`), đảm bảo phản hồi tức thì trên UI, không giật lag.

---

## II. ĐẶC TẢ SCHEMA DỮ LIỆU ĐÍCH (`logs/ui_preferences.json`)

```json
{
  "theme": "light",
  "mask_accounts": true,
  "auto_switch": true
}
```

---

## III. CHỈ DẪN TRIỂN KHAI CHI TIẾT TỪNG BƯỚC CHO CODER (EXHAUSTIVE STEP-BY-STEP RECIPE)

---

### BƯỚC 1: BỔ SUNG TRƯỜNG `AutoSwitch` VÀO `internal/ui/themes.go`

Tại file `internal/ui/themes.go` (khoảng dòng 30–35):
- **Code hiện tại:**
  ```go
  // UIPreferences stores persistent user interface settings.
  type UIPreferences struct {
      Theme        string `json:"theme"`
      MaskAccounts bool   `json:"mask_accounts"`
  }
  ```
- **Sửa thành:**
  ```go
  // UIPreferences stores persistent user interface settings.
  type UIPreferences struct {
      Theme        string `json:"theme"`
      MaskAccounts bool   `json:"mask_accounts"`
      AutoSwitch   bool   `json:"auto_switch"`
  }
  ```

---

### BƯỚC 2: CẬP NHẬT NẠP CẤU HÌNH KHI KHỞI ĐỘNG TẠI `internal/ui/app.go`

Trong hàm `NewAppWithFyne`:

1. **Tại khối đọc preferences (dòng 112–118):**
   - **Code hiện tại:**
     ```go
     // 2. Load UI Preferences (Theme & Masking)
     if prefs, err := LoadUIPreferences(a.preferencesPath); err == nil && prefs != nil {
         if prefs.Theme != "" {
             a.currentTheme = prefs.Theme
         }
         a.maskAccounts = prefs.MaskAccounts
     }
     ```
   - **Sửa thành:**
     ```go
     // 2. Load UI Preferences (Theme, Masking & Auto-Switch)
     if prefs, err := LoadUIPreferences(a.preferencesPath); err == nil && prefs != nil {
         if prefs.Theme != "" {
             a.currentTheme = prefs.Theme
         }
         a.maskAccounts = prefs.MaskAccounts
         a.autoSwitchEnabled = prefs.AutoSwitch
     }
     ```

2. **Tại khối khởi tạo Automation Controller (dòng 133–136):**
   - **Code hiện tại:**
     ```go
     // 4. Initialize Automation Controller
     if mgr != nil {
         a.controller = automation.NewAutomationController(mgr, a.telemetryPool, a.quotaCache, nil)
     }
     ```
   - **Sửa thành:**
     ```go
     // 4. Initialize Automation Controller
     if mgr != nil {
         a.controller = automation.NewAutomationController(mgr, a.telemetryPool, a.quotaCache, nil)
         if a.autoSwitchEnabled {
             a.controller.SetEnabled(true)
         }
     }
     ```

---

### BƯỚC 3: GHI ĐĨA KHI GẠT NÚT TỰ ĐỘNG TẠI `internal/ui/app.go`

Tìm hàm `SetAutoSwitchEnabled(enabled bool)` (dòng 832–857):
- **Code hiện tại:**
  ```go
  // SetAutoSwitchEnabled explicitly sets the auto-switch state in the UI and controller.
  func (a *App) SetAutoSwitchEnabled(enabled bool) {
      a.mu.Lock()
      if a.autoSwitchEnabled == enabled {
          ctrl := a.controller
          toggle := a.autoSwitchToggle
          a.mu.Unlock()
          if toggle != nil && toggle.IsOn() != enabled {
              toggle.SetOn(enabled)
          }
          if ctrl != nil && ctrl.IsEnabled() != enabled {
              ctrl.SetEnabled(enabled)
          }
          return
      }
      a.autoSwitchEnabled = enabled
      toggle := a.autoSwitchToggle
      ctrl := a.controller
      a.mu.Unlock()

      if toggle != nil && toggle.IsOn() != enabled {
          toggle.SetOn(enabled)
      }
      if ctrl != nil {
          ctrl.SetEnabled(enabled)
      }
  }
  ```
- **Sửa thành (bổ sung ghi đĩa bất đồng bộ):**
  ```go
  // SetAutoSwitchEnabled explicitly sets the auto-switch state in the UI, controller, and disk preferences.
  func (a *App) SetAutoSwitchEnabled(enabled bool) {
      a.mu.Lock()
      if a.autoSwitchEnabled == enabled {
          ctrl := a.controller
          toggle := a.autoSwitchToggle
          a.mu.Unlock()
          if toggle != nil && toggle.IsOn() != enabled {
              toggle.SetOn(enabled)
          }
          if ctrl != nil && ctrl.IsEnabled() != enabled {
              ctrl.SetEnabled(enabled)
          }
          return
      }
      a.autoSwitchEnabled = enabled
      toggle := a.autoSwitchToggle
      ctrl := a.controller
      path := a.preferencesPath
      theme := a.currentTheme
      mask := a.maskAccounts
      a.mu.Unlock()

      if toggle != nil && toggle.IsOn() != enabled {
          toggle.SetOn(enabled)
      }
      if ctrl != nil {
          ctrl.SetEnabled(enabled)
      }

      // Non-blocking disk persistence
      go func() {
          pref, err := LoadUIPreferences(path)
          if err != nil || pref == nil {
              pref = &UIPreferences{Theme: theme, MaskAccounts: mask}
          }
          pref.AutoSwitch = enabled
          _ = SaveUIPreferences(path, pref)
      }()
  }
  ```

---

### BƯỚC 4: BẢO TOÀN `AutoSwitch` TRONG `SetTheme` VÀ `SetMaskAccounts` TẠI `internal/ui/app.go`

1. **Trong `SetTheme` (khoảng dòng 230–243):**
   - **Code hiện tại:**
     ```go
     prefs, err := LoadUIPreferences(path)
     if err != nil || prefs == nil {
         prefs = &UIPreferences{Theme: themeName, MaskAccounts: mask}
     }
     prefs.Theme = themeName
     _ = SaveUIPreferences(path, prefs)
     ```
   - **Sửa thành:** Đảm bảo khi `LoadUIPreferences` lỗi hoặc tạo mới, trường `AutoSwitch` lấy từ `a.autoSwitchEnabled` (không bị reset về false):
     ```go
     a.mu.Lock()
     a.currentTheme = themeName
     path := a.preferencesPath
     mask := a.maskAccounts
     autoSwitch := a.autoSwitchEnabled
     a.mu.Unlock()

     prefs, err := LoadUIPreferences(path)
     if err != nil || prefs == nil {
         prefs = &UIPreferences{Theme: themeName, MaskAccounts: mask, AutoSwitch: autoSwitch}
     }
     prefs.Theme = themeName
     _ = SaveUIPreferences(path, prefs)
     ```

2. **Trong `SetMaskAccounts` (khoảng dòng 898–906):**
   - **Sửa khối goroutine lưu:**
     ```go
     a.mu.Lock()
     ...
     path := a.preferencesPath
     theme := a.currentTheme
     autoSwitch := a.autoSwitchEnabled
     a.mu.Unlock()
     ...
     go func() {
         pref, err := LoadUIPreferences(path)
         if err != nil || pref == nil {
             pref = &UIPreferences{Theme: theme, AutoSwitch: autoSwitch}
         }
         pref.MaskAccounts = mask
         _ = SaveUIPreferences(path, pref)
     }()
     ```

---

## IV. MA TRẬN KIỂM THỬ TỰ ĐỘNG BẮT BUỘC CHO CODER (TEST MATRIX)

Thêm các bài test vào `internal/ui/themes_test.go` hoặc `internal/ui/ui_test.go`:

1. **`TestUIPreferences_JSONSerialization_WithAutoSwitch`:**
   - Tạo struct `UIPreferences{Theme: "warm", MaskAccounts: true, AutoSwitch: true}`.
   - Marshal sang JSON và kiểm tra chuỗi có `"auto_switch":true`.
   - Unmarshal ngược lại và assert `pref.AutoSwitch == true`.
2. **`TestApp_AutoSwitchPersistence_RoundTrip`:**
   - Khởi tạo `appInstance` với `tempPrefPath`.
   - Gọi `appInstance.SetAutoSwitchEnabled(true)`.
   - Đợi 50ms để goroutine lưu xong.
   - Đọc trực tiếp file trên đĩa qua `LoadUIPreferences(tempPrefPath)`, assert `prefs.AutoSwitch == true`.
   - Gọi tiếp `appInstance.SetAutoSwitchEnabled(false)`.
   - Đợi 50ms, đọc lại file trên đĩa, assert `prefs.AutoSwitch == false`.
3. **`TestApp_StartupRestoresAutoSwitchState`:**
   - Tạo file test `ui_preferences.json` có sẵn `{"theme":"dark","mask_accounts":false,"auto_switch":true}`.
   - Khởi tạo app instance mới trỏ vào file này.
   - Assert `app.IsAutoSwitchEnabled() == true`.
   - Nếu cửa sổ được tạo, assert `app.autoSwitchToggle.IsOn() == true`.
   - Assert `app.Controller().IsEnabled() == true`.

---

## V. TIÊU CHÍ NGHIỆM THU CUỐI CÙNG (FINAL ACCEPTANCE CRITERIA)

1. `go test ./internal/ui/...` chạy PASS 100% không có bất kỳ lỗi nào.
2. Khi người dùng gạt nút Tự động ON/OFF trên giao diện thực tế, mở file `logs/ui_preferences.json` ra thấy trường `"auto_switch": true` (hoặc `false`) được cập nhật ngay lập tức.
3. Khi tắt app và mở lại, nút Tự động giữ nguyên đúng vị trí ON hoặc OFF của phiên làm việc trước đó.
4. Nút Ẩn tài khoản và Dropdown Theme vẫn hoạt động bình thường, không làm mất trạng thái của nút Tự động.
