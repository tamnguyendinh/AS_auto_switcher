# Đặc Tả Kiến Trúc & Chỉ Dẫn Kỹ Thuật Chi Tiết Cho Coder: Lưu Trữ Bền Vững Trạng Thái Nút Ghim Cửa Sổ (Always-on-Top / Pin Window)
(Window Always-on-Top Pin Persistence Specification — Complete Implementation Recipe)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-ALWAYS-ON-TOP-PERSISTENCE`
- **Trạng thái:** DRAFT (Trình Human Owner phê duyệt)
- **Đối tượng đọc:** Coder, Supervisor, QA

---

## I. MỤC TIÊU VÀ NGUYÊN TẮC CỐT LÕI (OBJECTIVES & INVARIANTS)

Bản đặc tả này chuẩn hóa cơ chế lưu trữ bền vững (*Persistence*) cho nút ghim cửa sổ **"📌"** (`alwaysOnTopBtn` / `alwaysOnTop`) trên thanh công cụ giao diện (Row 2 Header), tích hợp đầy đủ và đồng bộ hoàn toàn vào hệ thống cấu hình tùy chọn giao diện người dùng `logs/ui_preferences.json`.

### Các điều khoản kỷ luật sắt (Mandatory Invariants):

1. **HỢP NHẤT 4 TÙY CHỌN GIAO DIỆN VÀO MỘT NƠI DUY NHẤT (UNIFIED 4-WAY PREFERENCES INVARIANT):**
   - Tệp `logs/ui_preferences.json` là nguồn chân lý duy nhất (*Single Source of Truth*) quản lý 4 tùy chọn UI của người dùng:
     1. `theme` (chủ đề giao diện: `"dark"`, `"light"`, `"warm"`).
     2. `mask_accounts` (trạng thái ẩn/hiện email tài khoản: `true`/`false`).
     3. `auto_switch` (trạng thái tự động chuyển tài khoản: `true`/`false`).
     4. `always_on_top` (trạng thái ghim cửa sổ luôn nổi lên trên: `true`/`false`).

2. **BẢO TỒN GIÁ TRỊ MẶC ĐỊNH & TƯƠNG THÍCH NGƯỢC (CANONICAL DEFAULT & BACKWARD COMPATIBILITY INVARIANT):**
   - Giá trị mặc định chuẩn mực của Antigravity từ trước tới nay là **ghim cửa sổ luôn nổi** (`alwaysOnTop: true`).
   - Cấu trúc dữ liệu trong Go **bắt buộc** dùng con trỏ boolean: `AlwaysOnTop *bool \`json:"always_on_top,omitempty"\``:
     - Nếu tệp `ui_preferences.json` cũ thiếu trường `always_on_top` (hoặc tệp chưa tồn tại / deserialization trả về `nil`), hệ thống **bắt buộc mặc định là `true`** (ghim cửa sổ). Tuyệt đối không được để Go unmarshal ra zero-value `false` làm mất tính năng ghim mặc định của người dùng cũ.
     - Nếu trường `always_on_top` có giá trị rõ ràng trong tệp (`true` hoặc `false`), hệ thống phải khôi phục chính xác 100% giá trị do người dùng đã chọn.

3. **BẢO TOÀN TRẠNG THÁI KHỞI ĐỘNG (STARTUP RESTORATION INVARIANT):**
   - Trong hàm khởi tạo `NewAppWithFyne`:
     - Giá trị `alwaysOnTop` phải được khôi phục từ `prefs.AlwaysOnTop`.
     - Nút ghim `alwaysOnTopBtn` trên giao diện phải khởi tạo đúng mức độ hiển thị:
       - Nếu `alwaysOnTop == true`: `Importance = widget.HighImportance` (nút nổi bật / đã ghim).
       - Nếu `alwaysOnTop == false`: `Importance = widget.MediumImportance` (nút chìm / chưa ghim).
     - Win32 API `setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)` phải được gọi để áp dụng đúng trạng thái cho cửa sổ Win32 thực tế khi cửa sổ sẵn sàng.

4. **BẢO TOÀN CHÉO KHÔNG GHI ĐÈ LẪN NHAU (CROSS-SETTING MUTUAL PRESERVATION):**
   - Khi người dùng bấm nút ghim 📌, 3 giá trị `theme`, `mask_accounts`, `auto_switch` phải được giữ nguyên vẹn.
   - Khi người dùng đổi Theme, bấm nút Ẩn tài khoản 🔒, hoặc gạt nút Tự động, giá trị `always_on_top` đã lưu trước đó không được phép bị mất hoặc bị reset.

5. **GHI ĐĨA BẤT ĐỒNG BỘ NON-BLOCKING (ZERO UI JANK):**
   - Thao tác ghi đĩa khi người dùng bấm nút ghim phải chạy bất đồng bộ trong goroutine (tương tự như `SetMaskAccounts` và `SetAutoSwitchEnabled`), đảm bảo giao diện phản hồi lập tức dưới 16ms, không giật lag.

---

## II. ĐẶC TẢ SCHEMA DỮ LIỆU ĐÍCH (`logs/ui_preferences.json`)

Tệp `logs/ui_preferences.json` sau khi chuẩn hóa 4 trường:

```json
{
  "theme": "dark",
  "mask_accounts": false,
  "auto_switch": true,
  "always_on_top": true
}
```

Trường hợp người dùng bỏ ghim cửa sổ:
```json
{
  "theme": "dark",
  "mask_accounts": false,
  "auto_switch": true,
  "always_on_top": false
}
```

---

## III. CHỈ DẪN TRIỂN KHAI CHI TIẾT TỪNG BƯỚC CHO CODER (EXHAUSTIVE STEP-BY-STEP RECIPE)

---

### BƯỚC 1: BỔ SUNG TRƯỜNG `AlwaysOnTop` VÀO `internal/ui/themes.go`

Tại file `internal/ui/themes.go` (khoảng dòng 30–35):

1. **Cập nhật struct `UIPreferences`:**
   - **Code hiện tại:**
     ```go
     // UIPreferences stores persistent user interface settings.
     type UIPreferences struct {
         Theme        string `json:"theme"`
         MaskAccounts bool   `json:"mask_accounts"`
         AutoSwitch   bool   `json:"auto_switch"`
     }
     ```
   - **Sửa thành:**
     ```go
     // UIPreferences stores persistent user interface settings.
     type UIPreferences struct {
         Theme        string `json:"theme"`
         MaskAccounts bool   `json:"mask_accounts"`
         AutoSwitch   bool   `json:"auto_switch"`
         AlwaysOnTop  *bool  `json:"always_on_top,omitempty"`
     }
     ```

2. **Cập nhật hàm `LoadUIPreferences` trong `internal/ui/themes.go` (khoảng dòng 262–289):**
   Đảm bảo khi tệp chưa tồn tại hoặc thiếu trường `always_on_top`, giá trị mặc định luôn là `true`:
   - **Code sửa tại các điểm return mặc định:**
     ```go
     defaultTop := true
     ```
     Tại khối file không tồn tại hoặc rỗng:
     ```go
     if os.IsNotExist(err) {
         defaultTop := true
         return &UIPreferences{Theme: ThemeDark, AlwaysOnTop: &defaultTop}, nil
     }
     ```
     Tại khối kiểm tra sau khi unmarshal JSON:
     ```go
     if pref.AlwaysOnTop == nil {
         defaultTop := true
         pref.AlwaysOnTop = &defaultTop
     }
     ```

---

### BƯỚC 2: CẬP NHẬT NẠP CẤU HÌNH KHI KHỞI ĐỘNG TẠI `internal/ui/app.go`

Trong hàm `NewAppWithFyne`:

1. **Tại khối đọc preferences (khoảng dòng 112–119):**
   - **Code hiện tại:**
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
   - **Sửa thành:**
     ```go
     // 2. Load UI Preferences (Theme, Masking, Auto-Switch & Always-on-Top)
     if prefs, err := LoadUIPreferences(a.preferencesPath); err == nil && prefs != nil {
         if prefs.Theme != "" {
             a.currentTheme = prefs.Theme
         }
         a.maskAccounts = prefs.MaskAccounts
         a.autoSwitchEnabled = prefs.AutoSwitch
         if prefs.AlwaysOnTop != nil {
             a.alwaysOnTop = *prefs.AlwaysOnTop
         }
     }
     ```

2. **Tại các khối áp dụng Win32 Always-on-Top khi khởi động (dòng 308–310 và dòng 475–477):**
   - **Code hiện tại:**
     ```go
     if a.alwaysOnTop {
         setWindowAlwaysOnTop(windowTitle, true)
     }
     ```
   - **Sửa thành (áp dụng đúng cả khi `alwaysOnTop == false`):**
     ```go
     setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)
     ```
   *(Hàm `setWindowAlwaysOnTop(title, isTop)` sẵn có sẽ gửi `hwndTopMost` khi `true` và `hwndNoTopMost` khi `false`).*

---

### BƯỚC 3: CẬP NHẬT GHI ĐĨA CHO `ToggleAlwaysOnTop` VÀ `SetAlwaysOnTop` TẠI `internal/ui/app.go`

Tìm các hàm điều khiển Always-on-Top (khoảng dòng 793–826):

1. **Hàm `ToggleAlwaysOnTop`:**
   - **Sửa thành:**
     ```go
     // ToggleAlwaysOnTop switches the state of the Always-on-Top flag and persists the preference.
     func (a *App) ToggleAlwaysOnTop() {
         a.mu.Lock()
         a.alwaysOnTop = !a.alwaysOnTop
         enabled := a.alwaysOnTop
         path := a.preferencesPath
         theme := a.currentTheme
         mask := a.maskAccounts
         autoSwitch := a.autoSwitchEnabled
         a.mu.Unlock()

         setWindowAlwaysOnTop(windowTitle, enabled)
         a.syncAlwaysOnTopBtn(enabled)

         // Non-blocking disk persistence
         go func() {
             pref, err := LoadUIPreferences(path)
             if err != nil || pref == nil {
                 pref = &UIPreferences{
                     Theme:        theme,
                     MaskAccounts: mask,
                     AutoSwitch:   autoSwitch,
                 }
             }
             pref.AlwaysOnTop = &enabled
             _ = SaveUIPreferences(path, pref)
         }()
     }
     ```

2. **Hàm `SetAlwaysOnTop`:**
   - **Sửa thành:**
     ```go
     // SetAlwaysOnTop explicitly sets the Always-on-Top state and persists the preference.
     func (a *App) SetAlwaysOnTop(enabled bool) {
         a.mu.Lock()
         if a.alwaysOnTop == enabled {
             a.mu.Unlock()
             setWindowAlwaysOnTop(windowTitle, enabled)
             a.syncAlwaysOnTopBtn(enabled)
             return
         }
         a.alwaysOnTop = enabled
         path := a.preferencesPath
         theme := a.currentTheme
         mask := a.maskAccounts
         autoSwitch := a.autoSwitchEnabled
         a.mu.Unlock()

         setWindowAlwaysOnTop(windowTitle, enabled)
         a.syncAlwaysOnTopBtn(enabled)

         // Non-blocking disk persistence
         go func() {
             pref, err := LoadUIPreferences(path)
             if err != nil || pref == nil {
                 pref = &UIPreferences{
                     Theme:        theme,
                     MaskAccounts: mask,
                     AutoSwitch:   autoSwitch,
                 }
             }
             pref.AlwaysOnTop = &enabled
             _ = SaveUIPreferences(path, pref)
         }()
     }
     ```

---

### BƯỚC 4: BẢO TOÀN `AlwaysOnTop` TRONG `SetTheme`, `SetMaskAccounts`, VÀ `SetAutoSwitchEnabled` TẠI `internal/ui/app.go`

Đảm bảo khi lưu các cài đặt khác, trường `AlwaysOnTop` không bị mất nếu cần tạo mới struct fallback:

1. **Trong `SetTheme` (khoảng dòng 240–254):**
   ```go
   a.mu.Lock()
   a.currentTheme = themeName
   path := a.preferencesPath
   mask := a.maskAccounts
   autoSwitch := a.autoSwitchEnabled
   alwaysOnTop := a.alwaysOnTop
   a.mu.Unlock()

   prefs, err := LoadUIPreferences(path)
   if err != nil || prefs == nil {
       prefs = &UIPreferences{
           Theme:        themeName,
           MaskAccounts: mask,
           AutoSwitch:   autoSwitch,
           AlwaysOnTop:  &alwaysOnTop,
       }
   }
   prefs.Theme = themeName
   if prefs.AlwaysOnTop == nil {
       prefs.AlwaysOnTop = &alwaysOnTop
   }
   _ = SaveUIPreferences(path, prefs)
   ```

2. **Trong `SetAutoSwitchEnabled` (khoảng dòng 860–880):**
   ```go
   a.mu.Lock()
   ...
   path := a.preferencesPath
   theme := a.currentTheme
   mask := a.maskAccounts
   alwaysOnTop := a.alwaysOnTop
   a.mu.Unlock()
   ...
   go func() {
       pref, err := LoadUIPreferences(path)
       if err != nil || pref == nil {
           pref = &UIPreferences{
               Theme:        theme,
               MaskAccounts: mask,
               AlwaysOnTop:  &alwaysOnTop,
           }
       }
       pref.AutoSwitch = enabled
       if pref.AlwaysOnTop == nil {
           pref.AlwaysOnTop = &alwaysOnTop
       }
       _ = SaveUIPreferences(path, pref)
   }()
   ```

3. **Trong `SetMaskAccounts` (khoảng dòng 903–930):**
   ```go
   a.mu.Lock()
   ...
   path := a.preferencesPath
   theme := a.currentTheme
   autoSwitch := a.autoSwitchEnabled
   alwaysOnTop := a.alwaysOnTop
   a.mu.Unlock()
   ...
   go func() {
       pref, err := LoadUIPreferences(path)
       if err != nil || pref == nil {
           pref = &UIPreferences{
               Theme:       theme,
               AutoSwitch:  autoSwitch,
               AlwaysOnTop: &alwaysOnTop,
           }
       }
       pref.MaskAccounts = mask
       if pref.AlwaysOnTop == nil {
           pref.AlwaysOnTop = &alwaysOnTop
       }
       _ = SaveUIPreferences(path, pref)
   }()
   ```

---

## IV. MA TRẬN KIỂM THỬ TỰ ĐỘNG BẮT BUỘC CHO CODER (TEST MATRIX)

Thêm các bài test toàn diện vào `internal/ui/themes_test.go` hoặc `internal/ui/ui_test.go`:

1. **`TestUIPreferences_JSONSerialization_WithAlwaysOnTop`:**
   - Case A: `AlwaysOnTop = true` -> Marshal ra JSON phải chứa `"always_on_top": true`.
   - Case B: `AlwaysOnTop = false` -> Marshal ra JSON phải chứa `"always_on_top": false`.
   - Case C (Tương thích ngược): JSON cũ không có trường `"always_on_top"` -> Unmarshal vào struct phải ra `AlwaysOnTop == nil`, và qua hàm `LoadUIPreferences` phải tự động có giá trị mặc định là `true`.

2. **`TestApp_AlwaysOnTopPersistence_RoundTrip`:**
   - Khởi tạo `appInstance` với file preferences tạm.
   - Ban đầu `appInstance.IsAlwaysOnTop()` phải là `true`.
   - Gọi `appInstance.ToggleAlwaysOnTop()`.
   - Assert `appInstance.IsAlwaysOnTop() == false`.
   - Đợi 50ms cho goroutine hoàn tất ghi đĩa.
   - Dùng `LoadUIPreferences` đọc trực tiếp từ đĩa: assert `prefs.AlwaysOnTop != nil` và `*prefs.AlwaysOnTop == false`.
   - Gọi tiếp `appInstance.ToggleAlwaysOnTop()`.
   - Assert `appInstance.IsAlwaysOnTop() == true`.
   - Đợi 50ms, đọc lại đĩa: assert `*prefs.AlwaysOnTop == true`.

3. **`TestApp_StartupRestoresAlwaysOnTopState`:**
   - Tạo file `ui_preferences.json` giả lập trên đĩa với nội dung:
     `{"theme":"light","mask_accounts":true,"auto_switch":false,"always_on_top":false}`
   - Khởi tạo `appInstance` mới nạp từ file này.
   - Assert `appInstance.IsAlwaysOnTop() == false`.
   - Nếu `alwaysOnTopBtn` được khởi tạo, assert `appInstance.alwaysOnTopBtn.Importance == widget.MediumImportance`.

4. **`TestApp_CrossSettingPreservation_4Way`:**
   - Tạo preferences với cả 4 giá trị đặc thù (`Theme: "warm"`, `MaskAccounts: true`, `AutoSwitch: true`, `AlwaysOnTop: false`).
   - Gọi `SetTheme("light")` -> Assert `AlwaysOnTop` vẫn là `false`.
   - Gọi `SetMaskAccounts(false)` -> Assert `AlwaysOnTop` vẫn là `false`.
   - Gọi `SetAutoSwitchEnabled(false)` -> Assert `AlwaysOnTop` vẫn là `false`.
   - Gọi `ToggleAlwaysOnTop()` -> Assert `Theme` vẫn là `"light"`, `MaskAccounts` vẫn là `false`, `AutoSwitch` vẫn là `false`.

---

## V. TIÊU CHÍ NGHIỆM THU CUỐI CÙNG (FINAL ACCEPTANCE CRITERIA)

1. `go test ./internal/ui/...` chạy PASS 100% không phát sinh bất kỳ lỗi nào.
2. Khi người dùng bấm nút ghim 📌 trên giao diện để unpin (nút chuyển sang trạng thái chìm / MediumImportance), tệp `logs/ui_preferences.json` được cập nhật ngay lập tức `"always_on_top": false`.
3. Khi người dùng bấm lại nút 📌 để pin (nút chuyển sang trạng thái nổi / HighImportance), tệp `logs/ui_preferences.json` được cập nhật `"always_on_top": true`.
4. Khi tắt ứng dụng hoàn toàn và mở lại, trạng thái ghim/bỏ ghim của cửa sổ được giữ nguyên đúng như phiên trước đó, không bị tự động bật lại về mặc định khi người dùng đã cố ý bỏ ghim.
5. Việc đổi Theme, gạt nút Ẩn tài khoản 🔒, hay gạt nút Tự động không làm mất trạng thái của nút Ghim 📌.
