# Đặc Tả Kiến Trúc & Chỉ Dẫn Kỹ Thuật Chi Tiết Cho Coder: Trạng Thái Bật/Tắt Tài Khoản và Cơ Chế Dừng An Toàn

(Account On/Off Status & Auto-Stop Safety Specification — Complete Step-by-Step Implementation Recipe)

- **Ngày lập:** 2026-10-21.
- **Múi giờ:** Asia/Bangkok.
- **Phiên bản:** 2.1 (UI-FIRST ACTIONABLE RECIPE — Chuẩn hóa lộ trình: Tầng Giao diện Thẻ code đầu tiên theo chỉ đạo Owner).
- **Mã văn kiện:** `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`.
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner.
- **Cơ sở đối chiếu source:** commit `c11c479`.
- **Trạng thái:** DRAFT (Trình duyệt Owner) — Sẵn sàng cho Coder triển khai tuần tự theo từng Phase không cần suy đoán.
- **Đối tượng áp dụng:** Coder, Planner, QA, Supervisor, Human Owner.

---

## I. MỤC TIÊU VÀ NGUYÊN TẮC CỐT LÕI (OBJECTIVES & INVARIANTS)

Bản đặc tả này chuẩn hóa 2 tính năng gắn kết mật thiết trong hệ thống theo phương pháp tiếp cận **Giao diện làm trung tâm (UI-First)**:
1. Cơ chế Bật/Tắt (Enable/Disable hay `[ON]` / `[OFF]`) trực quan trên từng thẻ tài khoản, lưu bền vững tại kho credential `accounts_store/<email>.json`.
2. Cơ chế Dừng an toàn (Fail-Safe Auto-Stop) tự động ngắt chế độ Tự động (`auto_switch = false`) và thông báo chuẩn mực khi hệ thống không còn bất kỳ tài khoản nào đủ điều kiện để chuyển đổi.

### 7 Bất biến hệ thống bắt buộc (Mandatory Invariants):

1. **`Invariant-Account-Status-Persistence`:** Trạng thái Bật/Tắt của tài khoản phải được lưu bền vững vào chính tệp `accounts_store/<email>.json` dưới dạng trường `"status": "on"` hoặc `"status": "off"`. Khởi động lại ứng dụng bắt buộc phải khôi phục chính xác 100% trạng thái này.
2. **`Invariant-Backward-Compatibility-Default-On`:** Mọi tệp tài khoản hiện hữu trong `accounts_store/` chưa có trường `status` phải luôn được nạp với giá trị mặc định là `"on"`. Tuyệt đối không yêu cầu script migration hay làm lỗi các tài khoản cũ.
3. **`Invariant-Token-Update-Preservation`:** Các tác vụ tự động ghi tệp tài khoản do cập nhật token OAuth, refresh token hoặc capture credential từ Windows Credential Manager bắt buộc phải đọc trước và giữ nguyên giá trị `status` đang có trên đĩa. Tuyệt đối không được reset về `"on"` khi tài khoản đang bị người dùng tắt (`"off"`).
4. **`Invariant-Unauth-Card-Isolation`:** Thẻ tài khoản chưa có credential (`[Cần OAuth]`) chỉ tồn tại dưới dạng danh bạ được phát hiện từ Chrome, chưa thể sử dụng nên **tuyệt đối KHÔNG hiển thị nút `[ON/OFF]`**.
5. **`Invariant-Disabled-Switch-Lockout`:** Tài khoản mang trạng thái `status == "off"` bị ngăn chặn chuyển đổi 100% ở cả 2 tầng:
   - **Tầng Giao diện (UI):** Nút `[Chuyển]` bị làm mờ (màu `TextMuted`), con trỏ chuột mặc định, bấm vào không có tác dụng.
   - **Tầng Bộ điều khiển (Controller):** Bị bộ lọc ứng viên `FilterEligibleCandidates` loại bỏ ngay từ vòng duyệt đầu tiên.
6. **`Invariant-No-Candidate-Auto-Halt`:** Khi phát hiện không có ứng viên nào đủ điều kiện chuyển đổi (do tất cả tài khoản dự phòng đều bị Tắt `[OFF]`, hết hạn mức, hoặc lỗi preflight):
   - Hệ thống lập tức ngắt cờ Tự động (`auto_switch = false`).
   - Cập nhật và lưu bền vững `"auto_switch": false` vào tệp [logs/ui_preferences.json](file:///g:/antigravity_auto_switch/logs/ui_preferences.json).
   - Gạt công tắc "Tự động" trên UI về `OFF`.
   - Hiển thị thông báo trạng thái chuẩn hóa: `⚠️ Không có tài khoản để chuyển.`.
7. **`Invariant-Disabled-Telemetry-Silence`:** Không thực hiện quét quota định kỳ (Telemetry loop) hoặc gửi request mạng tới Google API cho bất kỳ tài khoản nào đang ở trạng thái `status == "off"`. Nút làm mới đơn vị `[🔄]` trên thẻ bị vô hiệu hóa (`disabled`).

---

## II. ĐẶC TẢ SCHEMA DỮ LIỆU ĐÍCH

### 1. Tệp tài khoản `accounts_store/<email>.json`

```json
{
  "email": "daohien204ccyersin@gmail.com",
  "status": "on",
  "saved_at": 1792502031.4247375,
  "token_data": {
    "token": {
      "access_token": "ya29...",
      "token_type": "Bearer",
      "refresh_token": "1//...",
      "expiry": "2026-10-21T21:00:58+07:00"
    },
    "auth_method": "consumer"
  }
}
```

Trường hợp tài khoản bị tắt:
```json
{
  "email": "daohien204ccyersin@gmail.com",
  "status": "off",
  "saved_at": 1792502031.4247375,
  "token_data": {
    "token": { ... }
  }
}
```

- Trường `status`: chuỗi văn bản (`string`), nhận một trong hai giá trị `"on"` hoặc `"off"`.
- Nếu thiếu trường này hoặc mang chuỗi rỗng `""` $\rightarrow$ Mặc định chuẩn là `"on"`.

### 2. Tệp cấu hình giao diện `logs/ui_preferences.json`

Khi cơ chế Dừng an toàn kích hoạt:
```json
{
  "theme": "dark",
  "mask_accounts": true,
  "auto_switch": false,
  "always_on_top": true
}
```

---

## III. BẢNG PHÂN RÃ THỰC THI (LỘ TRÌNH UI-FIRST: CÁI GÌ CODE TRƯỚC, CÁI GÌ CODE SAU)

Theo chỉ đạo trực tiếp của Human Owner, hệ thống áp dụng phương pháp **UI-First (Tầng Giao diện Thẻ code đầu tiên)** nhằm đảm bảo hợp đồng tương tác người dùng, diện mạo trực quan của thẻ và các nút bấm được chốt chuẩn xác ngay từ bước đầu:

```text
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 1 (CODE ĐẦU TIÊN): Tầng Giao diện Thẻ (UI Widgets)               │
│ [internal/ui/widgets.go]                                               │
│ • Tạo widget statusTextButton hiển thị [ON] (xanh lá) / [OFF] (xám)     │
│ • Bố trí nút [ON/OFF] nằm bên phải nút [🔄] trong actionCluster        │
│ • Xử lý làm mờ nút [Chuyển] và khóa nút [🔄] khi Status == "off"        │
│ • Ẩn hoàn toàn nút [ON/OFF] đối với thẻ [Cần OAuth]                    │
│ • Mở rộng AccountCardWidget và UpdateState để cập nhật tại chỗ         │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 2 (CODE THỨ HAI): Tầng Lưu trữ & Thực thể Dữ liệu                │
│ [internal/store/account_store.go, internal/manager/account_manager.go] │
│ • Mở rộng StoredAccountEnvelope với trường Status ("on" / "off")       │
│ • Bổ sung SetAccountStatus & GetAccountStatus (mặc định cũ là "on")    │
│ • Bảo tồn Status khi SaveAccount ghi đè / capture token                │
│ • Bổ sung Status vào AccountDisplayItem và map trong GetAccountStates  │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 3 (CODE THỨ BA): Tầng Điều phối Giao diện & Xử lý Sự kiện       │
│ [internal/ui/app.go]                                                   │
│ • Bổ sung hàm callback ToggleAccountStatus(email string)               │
│ • Nối sự kiện click nút [ON/OFF] vào AccountManager.SetAccountStatus   │
│ • Xử lý logic tắt tài khoản active: kích hoạt chuyển ngay              │
│ • Bắt sự kiện StatusNoCandidate: gạt tắt Toggle, lưu ui_preferences    │
│ • Hiển thị thông báo chuẩn: ⚠️ Không có tài khoản để chuyển.          │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 4 (CODE THỨ TƯ): Tầng Tự động Chuyển đổi & Telemetry Engine      │
│ [internal/automation/sorter.go, controller.go, telemetry/pool.go]      │
│ • Mở rộng CandidateAccount có trường Status                            │
│ • Bổ sung Criterion loại trừ tài khoản status == "off"                 │
│ • Kích hoạt Dừng tự động (enabled = false) khi len(eligible) == 0      │
│ • Bỏ qua polling telemetry đối với tài khoản status == "off"           │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 5 (CODE THỨ NĂM): Kiểm thử Tự động & Nghiệm thu                  │
│ [internal/ui/*_test.go, store/*_test.go, automation/*_test.go]         │
│ • UI test thẻ hiển thị [ON]/[OFF], click toggle, làm mờ nút Chuyển     │
│ • Store unit test lưu trữ, backward compatibility, state preservation  │
│ • Automation unit test loại trừ tài khoản OFF, dừng khi hết ứng viên   │
│ • E2E test chu trình tắt tài khoản active và tự động dừng an toàn      │
└────────────────────────────────────────────────────────────────────────┘
```

---

## IV. CHỈ DẪN KỸ THUẬT CHI TIẾT TỪNG PHASE CHO CODER

---

### PHASE 1 (CODE ĐẦU TIÊN): TẦNG GIAO DIỆN THẺ (UI WIDGETS)
*(Bắt đầu tại đây theo chỉ đạo của Owner: Hoàn thiện diện mạo thẻ, bố cục nút và hiệu ứng làm mờ)*

#### 1.1. Cập nhật `internal/ui/widgets.go`
1. **Xây dựng widget nút Text trạng thái `statusTextButton`:**
   Tạo widget phẳng tương tự như `switchTextButton` để đảm bảo phong cách hiển thị đồng bộ với giao diện hiện có:
   ```go
   type statusTextButton struct {
       widget.BaseWidget
       Status   string // "on" hoặc "off"
       OnTapped func()
       hovered  bool
       palette  ThemePalette
       bg       *canvas.Rectangle
       text     *canvas.Text
   }
   ```
   - **Quy tắc hiển thị văn bản:**
     - Khi `Status == "on"`: Nhãn hiển thị `"[ON]"`, màu chữ `palette.Success` (xanh lá), in đậm (`Bold: true`), cỡ chữ 10, căn giữa.
     - Khi `Status == "off"`: Nhãn hiển thị `"[OFF]"`, màu chữ `palette.TextMuted` (xám mờ), in đậm (`Bold: true`), cỡ chữ 10, căn giữa.
   - **Hiệu ứng chuột (Hover & Cursor):**
     - Con trỏ chuột: `desktop.PointerCursor` (bàn tay chỉ tay).
     - Khi rê chuột vào: Nền `bg` sáng nhẹ 14% theo màu tương ứng, nhãn sáng nhẹ tương tự nút `[Chuyển]`.
   - **Xử lý sự kiện click:**
     - Hàm `Tapped(*fyne.PointEvent)` gọi `OnTapped()` nếu khác nil.

2. **Mở rộng `AccountCardWidget`:**
   - Thêm trường `Status string` (mặc định khởi tạo là `"on"`).
   - Thêm trường tham chiếu: `statusButton *statusTextButton`.
   - Thêm trường tham chiếu callback: `onToggleStatus func(email string)`.

3. **Bố cục Hàng trên (Top Row) trong `NewAccountCardWidgetWithQuota`:**
   - Nhận thêm callback `onToggleStatus func(email string)` vào hàm khởi tạo thẻ.
   - Khởi tạo `statusButton`:
     ```go
     statusBtn := newStatusTextButton(item.Status, func() {
         if w.onToggleStatus != nil {
             w.onToggleStatus(w.Email)
         }
     }, palette)
     ```
   - **Vị trí nút:** Đặt `statusBtn` nằm ở **bên phải nút `refreshBtn`**:
     ```go
     // Thứ tự: [actionSlot (Chuyển/Dùng/OAuth)] -> [Nút 🔄 Làm mới] -> [Nút ON/OFF]
     actionCluster := container.NewHBox(actionSlot, compactButton(refreshBtn), compactButton(statusBtn))
     ```

4. **Cập nhật phương thức phản ứng tại chỗ `UpdateState(item, palette)`:**
   - Cập nhật trường nội bộ: `w.Status = item.Status`.
   - **Quy tắc đối với thẻ `[Cần OAuth]` (`!item.HasToken && !item.IsActive`):**
     - Gọi `w.statusButton.Hide()` $\rightarrow$ Tuyệt đối không hiển thị nút `[ON/OFF]` trên thẻ chưa có credential.
   - **Quy tắc đối với thẻ đã có credential hoặc đang Active:**
     - Gọi `w.statusButton.Show()`.
     - Cập nhật trạng thái và màu sắc của `statusButton` theo `item.Status` (`[ON]` xanh lá hoặc `[OFF]` xám).
   - **Xử lý làm mờ nút `[Chuyển]` khi `w.Status == "off"`:**
     - Khi tài khoản không active và có token (`!w.IsActive && w.HasToken`):
       - Nếu `w.Status == "off"`:
         - Khởi tạo nút chuyển ở trạng thái bị vô hiệu hóa: Màu chữ `palette.TextMuted`, không có hover effect, con trỏ chuột là `desktop.DefaultCursor`, click không thực hiện hành động nào (no-op).
       - Nếu `w.Status == "on"`:
         - Nút `[Chuyển]` khởi tạo bình thường: Màu chữ `palette.Primary`, con trỏ `PointerCursor`, click gọi `w.onSwitch(w.Email)`.
   - **Xử lý nút làm mới `[🔄]` khi `w.Status == "off"`:**
     - Nếu `w.Status == "off"`: Gọi `w.refreshButton.Disable()`.
     - Nếu `w.Status == "on"` và (`w.IsActive || w.HasToken`): Gọi `w.refreshButton.Enable()`.

---

### PHASE 2 (CODE THỨ HAI): TẦNG LƯU TRỮ & THỰC THỂ DỮ LIỆU (STORE & MANAGER)
*(Thực hiện thứ hai để cung cấp khả năng nạp/lưu trạng thái bền vững phục vụ giao diện)*

#### 2.1. Cập nhật `internal/store/account_store.go`
1. Mở rộng struct `StoredAccountEnvelope`:
   ```go
   type StoredAccountEnvelope struct {
       Email     string           `json:"email"`
       Status    string           `json:"status,omitempty"` // "on" hoặc "off", mặc định là "on" nếu rỗng
       SavedAt   float64          `json:"saved_at"`
       TokenData *model.TokenData `json:"token_data"`
   }
   ```
2. Cập nhật hàm `SaveAccount(email string, token *model.TokenData) error`:
   - Trước khi ghi đè tệp tài khoản, kiểm tra xem tệp `targetPath` đã có sẵn trên đĩa hay chưa.
   - Nếu tệp đã tồn tại: Đọc tệp và trích xuất `existingStatus`. Nếu `existingStatus == "off"`, giữ nguyên `envelope.Status = "off"`. Ngược lại gán `"on"`.
   - Nếu tệp chưa tồn tại: Gán `envelope.Status = "on"`.
   - **Mục đích:** Bảo vệ `Invariant-Token-Update-Preservation` — không làm mất trạng thái `"off"` khi token được capture hoặc refresh lại.
3. Bổ sung hàm cập nhật trạng thái `SetAccountStatus`:
   ```go
   // SetAccountStatus atomically updates only the status field ("on" or "off") of an existing account file.
   func (s *AccountStore) SetAccountStatus(email string, status string) error {
       if strings.TrimSpace(email) == "" {
           return ErrInvalidEmail
       }
       normStatus := strings.ToLower(strings.TrimSpace(status))
       if normStatus != "off" {
           normStatus = "on"
       }
       targetPath, err := SafeAccountPath(s.storeDir, email)
       if err != nil {
           return err
       }

       s.mu.Lock()
       defer s.mu.Unlock()

       data, err := os.ReadFile(targetPath)
       if err != nil {
           if os.IsNotExist(err) {
               return ErrAccountNotFound
           }
           return fmt.Errorf("failed to read account file: %w", err)
       }

       var env StoredAccountEnvelope
       if err := json.Unmarshal(data, &env); err != nil {
           return ErrCorruptedStoreFile
       }

       env.Status = normStatus

       // Ghi tệp an toàn bằng atomic temp file + os.Rename
       tmpFile, err := os.CreateTemp(s.storeDir, "acc_tmp_*.json")
       if err != nil {
           return err
       }
       tmpPath := tmpFile.Name()
       defer func() { _ = tmpFile.Close(); _ = os.Remove(tmpPath) }()

       encoder := json.NewEncoder(tmpFile)
       encoder.SetIndent("", "  ")
       if err := encoder.Encode(env); err != nil {
           return err
       }
       _ = tmpFile.Sync()
       _ = tmpFile.Close()

       _ = os.Remove(targetPath)
       return os.Rename(tmpPath, targetPath)
   }
   ```
4. Bổ sung hàm đọc trạng thái `GetAccountStatus`:
   ```go
   func (s *AccountStore) GetAccountStatus(email string) (string, error) {
       targetPath, err := SafeAccountPath(s.storeDir, email)
       if err != nil {
           return "on", err
       }
       s.mu.RLock()
       defer s.mu.RUnlock()

       data, err := os.ReadFile(targetPath)
       if err != nil {
           return "on", err
       }
       var env StoredAccountEnvelope
       if err := json.Unmarshal(data, &env); err != nil || env.Status == "" {
           return "on", nil
       }
       return strings.ToLower(env.Status), nil
   }
   ```

#### 2.2. Cập nhật `internal/manager/account_manager.go`
1. Mở rộng struct `AccountDisplayItem`:
   ```go
   type AccountDisplayItem struct {
       Email        string             `json:"email"`
       Name         string             `json:"name,omitempty"`
       AvatarURL    string             `json:"avatar_url,omitempty"`
       ProfileDir   string             `json:"profile_dir,omitempty"`
       State        model.AccountState `json:"state"`
       Status       string             `json:"status"` // "on" hoặc "off"
       IsActive     bool               `json:"is_active"`
       HasToken     bool               `json:"has_token"`
       DisplayBadge string             `json:"display_badge"`
   }
   ```
2. Trong hàm `GetAccountStates`:
   - Đọc trạng thái `status` từ store cho từng tài khoản có credential (khoảng dòng 335–347):
     ```go
     for _, item := range itemsMap {
         item.HasToken = storedSet[item.Email]
         item.IsActive = activeEmail != "" && item.Email == activeEmail
         item.Status = "on"
         if item.HasToken && m.accountStore != nil {
             if st, err := m.accountStore.GetAccountStatus(item.Email); err == nil && st != "" {
                 item.Status = st
             }
         }
         ...
     ```
3. Bổ sung hàm ủy quyền trên `AccountManager`:
   ```go
   func (m *AccountManager) SetAccountStatus(email string, status string) error {
       if m.accountStore == nil {
           return errors.New("account store is nil")
       }
       return m.accountStore.SetAccountStatus(email, status)
   }

   func (m *AccountManager) AccountStore() *store.AccountStore {
       return m.accountStore
   }
   ```

---

### PHASE 3 (CODE THỨ BA): TẦNG ĐIỀU PHỐI GIAO DIỆN & XỬ LÝ SỰ KIỆN (UI APP)
*(Thực hiện thứ ba để kết nối sự kiện click từ widget của Phase 1 với store của Phase 2)*

#### 3.1. Cập nhật `internal/ui/app.go`
1. **Bổ sung phương thức `ToggleAccountStatus(email string)`:**
   ```go
   func (a *App) ToggleAccountStatus(email string) {
       a.mu.Lock()
       if a.isBusy {
           a.mu.Unlock()
           return
       }
       mgr := a.mgr
       overview := a.overview
       currentTheme := a.currentTheme
       activeEmail := ""
       if overview != nil {
           activeEmail = overview.ActiveEmail
       }
       a.mu.Unlock()

       if mgr == nil || overview == nil {
           return
       }

       // 1. Xác định trạng thái hiện tại và đảo trạng thái
       targetIdx := -1
       currentStatus := "on"
       for idx, acc := range overview.Accounts {
           if acc.Email == email {
               targetIdx = idx
               if acc.Status != "" {
                   currentStatus = acc.Status
               }
               break
           }
       }

       newStatus := "off"
       if currentStatus == "off" {
           newStatus = "on"
       }

       // 2. Ghi bền vững vào kho accounts_store/<email>.json
       if err := mgr.SetAccountStatus(email, newStatus); err != nil {
           a.showStatusBar(fmt.Sprintf("❌ Lỗi cập nhật trạng thái: %v", err))
           return
       }

       // 3. Cập nhật overview và widget thẻ tại chỗ
       a.mu.Lock()
       if targetIdx >= 0 && targetIdx < len(overview.Accounts) {
           overview.Accounts[targetIdx].Status = newStatus
       }
       a.mu.Unlock()

       palette := GetThemePalette(currentTheme)
       fyne.Do(func() {
           if targetIdx >= 0 && targetIdx < len(a.accountCards) && a.accountCards[targetIdx] != nil {
               a.accountCards[targetIdx].UpdateState(overview.Accounts[targetIdx], palette)
           }
       })

       // 4. Xử lý đặc biệt: Nếu người dùng TẮT chính tài khoản đang Active
       if email == activeEmail && newStatus == "off" {
           a.mu.RLock()
           isAuto := a.autoSwitchEnabled
           ctrl := a.controller
           a.mu.RUnlock()

           if isAuto && ctrl != nil {
               go func() {
                   _, _ = ctrl.TriggerSwitch(context.Background(), "tài khoản active bị tắt")
               }()
           }
       }
   }
   ```

2. **Cập nhật bộ lắng nghe `OnStatusChange` của Automation:**
   Tại nhánh xử lý `case automation.StatusNoCandidate:` (khoảng dòng 451–454):
   ```go
   case automation.StatusNoCandidate:
       // 1. Tắt giao diện công tắc Tự động
       a.mu.Lock()
       a.autoSwitchEnabled = false
       if a.autoSwitchCheck != nil {
           a.autoSwitchCheck.SetChecked(false)
       }
       prefPath := a.preferencesPath
       a.mu.Unlock()

       // 2. Ghi bền vững auto_switch: false vào logs/ui_preferences.json
       go func() {
           pref, err := LoadUIPreferences(prefPath)
           if err != nil || pref == nil {
               defaultTop := true
               pref = &UIPreferences{Theme: ThemeDark, AutoSwitch: false, AlwaysOnTop: &defaultTop}
           } else {
               pref.AutoSwitch = false
           }
           _ = SaveUIPreferences(prefPath, pref)
       }()

       // 3. Hiển thị thông báo chuẩn hóa theo chỉ đạo của Owner
       a.showStatusBar("⚠️ Không có tài khoản để chuyển.")
       a.autoHideStatusBar(4 * time.Second)
   ```

3. Truyền callback `a.ToggleAccountStatus` vào tất cả các điểm tạo `AccountCardWidget` trong `app.go`.

---

### PHASE 4 (CODE THỨ TƯ): TẦNG TỰ ĐỘNG CHUYỂN ĐỔI & TELEMETRY ENGINE
*(Thực hiện thứ tư để bổ sung quy tắc loại trừ và kích hoạt dừng an toàn khi đánh giá ứng viên)*

#### 4.1. Cập nhật `internal/automation/sorter.go`
1. Mở rộng `CandidateAccount`:
   ```go
   type CandidateAccount struct {
       Email        string                      `json:"email"`
       Index        int                         `json:"index"`
       Status       string                      `json:"status"` // "on" hoặc "off"
       TokenValid   bool                        `json:"token_valid"`
       RefreshToken string                      `json:"-"`
       HasFreshPoll bool                        `json:"has_fresh_poll"`
       Evaluation   ModelQuotaEvaluation        `json:"evaluation"`
       RawQuota     *telemetry.AccountQuotaData `json:"raw_quota,omitempty"`
   }
   ```
2. Cập nhật hàm `FilterEligibleCandidates`:
   ```go
   // Criterion: Bỏ qua tài khoản đang ở chế độ OFF
   if strings.EqualFold(cand.Status, "off") {
       exclusions = append(exclusions, CandidateExclusion{
           Email:     cand.Email,
           Criterion: 6,
           Reason:    "tài khoản đang ở chế độ vô hiệu hóa (OFF)",
       })
       continue
   }
   ```

#### 4.2. Cập nhật `internal/automation/controller.go`
1. Trong `triggerSwitch`, khi nạp candidates từ store:
   - Gán `cand.Status = status` (lấy qua `GetAccountStatus`).
2. Tại khối `len(eligible) == 0` (hết ứng viên):
   ```go
   eligible, exclusions := FilterEligibleCandidates(candidates, activeEmail, failedMap)
   if len(eligible) == 0 {
       // Kích hoạt dừng an toàn
       c.enabled.Store(false)
       c.syncMonitoring()
       c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", exclusions)
       return fmt.Errorf("%w: %+v", ErrNoCandidates, exclusions)
   }
   ```
   Tương tự tại điểm tất cả ứng viên đều thất bại preflight:
   ```go
   c.enabled.Store(false)
   c.syncMonitoring()
   c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", nil)
   return ErrNoCandidates
   ```

#### 4.3. Cập nhật `internal/telemetry/pool.go`
1. Trong các vòng lặp polling quét quota tự động:
   - Bỏ qua các tài khoản có `status == "off"`.

---

### PHASE 5 (CODE THỨ NĂM): KIỂM THỬ TỰ ĐỘNG & NGHIỆM THU
*(Thực hiện cuối cùng để bảo đảm mọi bất biến hoạt động ổn định và không hồi quy)*

1. **Kiểm thử Widget Thẻ Giao diện (`internal/ui/ui_test.go`):**
   - `TestAccountCardWidget_StatusButton_RendersCorrectly`: Khởi tạo thẻ với status `"on"`, kiểm tra nhãn `[ON]`, nút `[Chuyển]` hoạt động. Đổi sang `"off"`, kiểm tra nhãn `[OFF]`, nút `[Chuyển]` mờ đi và click không kích hoạt chuyển.
   - `TestAccountCardWidget_UnauthAccount_HidesStatusButton`: Thẻ `[Cần OAuth]` kiểm tra `statusBtn.Visible() == false`.
2. **Kiểm thử Store (`internal/store/account_store_test.go`):**
   - `TestAccountStore_Status_Persistence_And_Preservation`: Kiểm tra nạp file cũ mặc định là `"on"`, cập nhật `"off"` ghi đúng đĩa, và gọi `SaveAccount` ghi đè token không làm mất cờ `"off"`.
3. **Kiểm thử Controller & Safety Stop (`internal/automation/controller_test.go`):**
   - `TestController_SafetyStop_WhenNoEligibleCandidates`: Giả lập tất cả tài khoản phụ đều mang `Status = "off"`. Kích hoạt chuyển đổi, kiểm tra controller tự động gạt `enabled = false`, phát sự kiện `StatusNoCandidate` với thông báo `"Không có tài khoản để chuyển."`.
4. **Kiểm thử Đồng bộ Tùy chọn (`internal/ui/ui_test.go`):**
   - `TestApp_StatusNoCandidate_TurnsOffAutoSwitch`: Kiểm tra phát sự kiện `StatusNoCandidate` làm tắt toggle UI và lưu bền vững `"auto_switch": false` vào `ui_preferences.json`.

---

## V. ĐỐI CHIẾU DANH MỤC TỆP TIN THAY ĐỔI

| Thứ tự | Tệp tin | Thư mục | Trách nhiệm | Phase |
|:---:|---|---|---|:---:|
| **1** | `widgets.go` | `internal/ui/` | Tạo `statusTextButton`, đặt bên phải `[🔄]`, làm mờ nút `[Chuyển]`, ẩn nút với `[Cần OAuth]`. | **Phase 1 (Code đầu tiên)** |
| **2** | `account_store.go` | `internal/store/` | Thêm `Status`, `SetAccountStatus`, `GetAccountStatus`, bảo tồn status khi `SaveAccount`. | **Phase 2 (Code thứ hai)** |
| **3** | `account_manager.go` | `internal/manager/` | Thêm `Status` vào `AccountDisplayItem`, map status trong `GetAccountStates`, thêm `SetAccountStatus`. | **Phase 2 (Code thứ hai)** |
| **4** | `app.go` | `internal/ui/` | Thêm `ToggleAccountStatus`, xử lý tắt active, bắt `StatusNoCandidate` gạt tắt Toggle và lưu JSON. | **Phase 3 (Code thứ ba)** |
| **5** | `sorter.go` | `internal/automation/` | Thêm `Status` vào `CandidateAccount`, thêm Criterion loại trừ tài khoản `"off"`. | **Phase 4 (Code thứ tư)** |
| **6** | `controller.go` | `internal/automation/` | Đọc `Status` khi build candidate, kích hoạt Dừng tự động (`enabled = false`) khi cạn ứng viên. | **Phase 4 (Code thứ tư)** |
| **7** | `pool.go` | `internal/telemetry/` | Bỏ qua quét quota ngầm cho tài khoản `"off"`. | **Phase 4 (Code thứ tư)** |
| **8** | Các tệp `*_test.go` | `internal/*/` | Bộ test tự động bảo đảm 7 bất biến và tiêu chí nghiệm thu. | **Phase 5 (Code cuối cùng)** |

---

## VI. BÀN GIAO VÀ CHỈ DẪN CHO LANE TIẾP THEO

- **Planner Lane:** Trích xuất trực tiếp bảng lộ trình UI-First tại **Mục III** và các Phase chi tiết tại **Mục IV** để lập danh sách Jobs cho Coder.
- **Coder Lane:** Bắt đầu code ngay từ **Phase 1: `internal/ui/widgets.go`**, sau đó đi tiếp tuần tự sang Phase 2, Phase 3, Phase 4 và Phase 5.
- **QA & Supervisor Lane:** Sử dụng 7 Bất biến hệ thống tại **Mục I** và danh mục kiểm thử tại **Phase 5** làm tiêu chuẩn nghiệm thu chặt chẽ.
