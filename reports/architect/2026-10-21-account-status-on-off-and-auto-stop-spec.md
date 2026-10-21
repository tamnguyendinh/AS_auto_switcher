# Đặc Tả Kiến Trúc & Chỉ Dẫn Kỹ Thuật Chi Tiết Cho Coder: Trạng Thái Bật/Tắt Tài Khoản và Cơ Chế Dừng An Toàn

(Account On/Off Status & Auto-Stop Safety Specification — Complete Step-by-Step Implementation Recipe)

- **Ngày lập:** 2026-10-21.
- **Múi giờ:** Asia/Bangkok.
- **Phiên bản:** 2.0 (ACTIONABLE RECIPE — Đã bổ sung lộ trình phân chia Phase và đối chiếu codebase chi tiết).
- **Mã văn kiện:** `SPEC-ARCH-20261021-ACCOUNT-STATUS-ON-OFF-AUTO-STOP`.
- **Thẩm quyền:** Phê chuẩn trực tiếp từ Human Owner.
- **Cơ sở đối chiếu source:** commit `b2ee12d`.
- **Trạng thái:** DRAFT (Trình duyệt Owner) — Sẵn sàng cho Coder triển khai tuần tự theo từng Phase không cần suy đoán.
- **Đối tượng áp dụng:** Coder, Planner, QA, Supervisor, Human Owner.

---

## I. MỤC TIÊU VÀ NGUYÊN TẮC CỐT LÕI (OBJECTIVES & INVARIANTS)

Bản đặc tả này chuẩn hóa 2 tính năng gắn kết mật thiết trong hệ thống:
1. Cơ chế Bật/Tắt (Enable/Disable hay `[ON]` / `[OFF]`) độc lập cho từng tài khoản lưu trực tiếp tại kho credential `accounts_store/<email>.json`.
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

## III. BẢNG PHÂN RÃ THỰC THI CHO CODER (CHÍNH XÁC CÁI GÌ CODE TRƯỚC, CÁI GÌ CODE SAU)

Để đảm bảo Coder triển khai mượt mà, không gặp lỗi biên dịch phụ thuộc vòng (circular dependency) hoặc mất dữ liệu, lộ trình lập trình bắt buộc chia làm **5 Phase tuần tự** như sau:

```text
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 1: Tầng Lưu trữ & Thực thể Dữ liệu (Store & Manager)             │
│ [internal/store/account_store.go, internal/manager/account_manager.go] │
│ • Mở rộng StoredAccountEnvelope với Status ("on" / "off")              │
│ • Bổ sung SetAccountStatus & GetAccountStatus                          │
│ • Bảo tồn Status khi SaveAccount ghi đè token                          │
│ • Map Status sang AccountDisplayItem                                   │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 2: Tầng Đánh giá Ứng viên & Telemetry Engine                     │
│ [internal/automation/sorter.go, controller.go, telemetry/pool.go]      │
│ • Mở rộng CandidateAccount có trường Status                            │
│ • Bổ sung Criterion loại trừ tài khoản status == "off"                 │
│ • Bổ sung logic Dừng tự động khi len(eligible) == 0 (ErrNoCandidates)  │
│ • Bỏ qua polling telemetry đối với tài khoản status == "off"           │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 3: Tầng Thành phần Giao diện Thẻ (UI Widgets)                    │
│ [internal/ui/widgets.go]                                               │
│ • Tạo statusTextButton hiển thị [ON] (xanh lá) / [OFF] (xám)           │
│ • Thêm statusTextButton vào bên phải nút [🔄] trong actionCluster       │
│ • Xử lý làm mờ nút [Chuyển] và khóa nút [🔄] khi Status == "off"        │
│ • Ẩn nút [ON/OFF] đối với thẻ [Cần OAuth]                              │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 4: Tầng Điều phối Giao diện & Xử lý Sự kiện (UI App)             │
│ [internal/ui/app.go]                                                   │
│ • Bổ sung hàm callback ToggleAccountStatus(email string)               │
│ • Xử lý logic tắt tài khoản active: kích hoạt chuyển ngay              │
│ • Bắt sự kiện StatusNoCandidate: gạt tắt Toggle, lưu ui_preferences    │
│ • Hiển thị thông báo chuẩn: ⚠️ Không có tài khoản để chuyển.          │
└───────────────────────────────────┬────────────────────────────────────┘
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────────────┐
│ PHASE 5: Kiểm thử Tự động & Nghiệm thu (Testing & Regression)          │
│ [internal/store/*_test.go, automation/*_test.go, ui/*_test.go]         │
│ • Unit test Store lưu trữ, backward compatibility, state preservation  │
│ • Unit test FilterEligibleCandidates loại trừ tài khoản OFF            │
│ • Unit test Controller dừng tự động khi hết ứng viên                   │
│ • UI test thẻ hiển thị đúng, click toggle và cập nhật in-place         │
└────────────────────────────────────────────────────────────────────────┘
```

---

## IV. CHỈ DẪN KỸ THUẬT CHI TIẾT TỪNG PHASE CHO CODER

---

### PHASE 1: TẦNG LƯU TRỮ & THỰC THỂ DỮ LIỆU (STORE & MANAGER)
*(Thực hiện trước tiên vì là nền tảng schema dữ liệu cho toàn bộ các tầng trên)*

#### 1.1. Cập nhật `internal/store/account_store.go`
1. Mở rộng struct `StoredAccountEnvelope`:
   ```go
   type StoredAccountEnvelope struct {
       Email     string           `json:"email"`
       Status    string           `json:"status,omitempty"` // "on" or "off", defaults to "on" if empty
       SavedAt   float64          `json:"saved_at"`
       TokenData *model.TokenData `json:"token_data"`
   }
   ```
2. Cập nhật hàm `SaveAccount(email string, token *model.TokenData) error`:
   - Trước khi tạo temp file để ghi `envelope`, kiểm tra xem file đích `targetPath` đã tồn tại chưa.
   - Nếu đã tồn tại: Đọc nội dung hiện có, unmarshal để lấy `existingStatus`. Nếu `existingStatus == "off"`, gán `envelope.Status = "off"`. Ngược lại gán `envelope.Status = "on"`.
   - Nếu chưa tồn tại: Gán `envelope.Status = "on"`.
   - **Mục đích:** Bảo vệ `Invariant-Token-Update-Preservation` — không làm mất trạng thái `"off"` khi token được làm mới hoặc capture lại.
3. Bổ sung hàm `SetAccountStatus(email string, status string) error`:
   ```go
   // SetAccountStatus atomically updates only the status field ("on" or "off") of an existing account file.
   func (s *AccountStore) SetAccountStatus(email string, status string) error
   ```
   - Chuẩn hóa: `normStatus := strings.ToLower(strings.TrimSpace(status))`. Nếu khác `"off"`, quy về `"on"`.
   - Đọc file hiện hữu: Nếu không tìm thấy file, trả về `ErrAccountNotFound`.
   - Cập nhật trường `envelope.Status = normStatus`.
   - Ghi đè tệp an toàn bằng cơ chế atomic temp file + `os.Rename` hiện có của store.
4. Bổ sung hàm `GetAccountStatus(email string) (string, error)`:
   - Đọc tệp và trả về trường `Status`. Nếu chuỗi rỗng hoặc file chưa có trường này, trả về `"on"`.

#### 1.2. Cập nhật `internal/manager/account_manager.go`
1. Mở rộng `AccountDisplayItem`:
   ```go
   type AccountDisplayItem struct {
       Email        string             `json:"email"`
       Name         string             `json:"name,omitempty"`
       AvatarURL    string             `json:"avatar_url,omitempty"`
       ProfileDir   string             `json:"profile_dir,omitempty"`
       State        model.AccountState `json:"state"`
       Status       string             `json:"status"` // "on" or "off"
       IsActive     bool               `json:"is_active"`
       HasToken     bool               `json:"has_token"`
       DisplayBadge string             `json:"display_badge"`
   }
   ```
2. Trong hàm `GetAccountStates(ctx context.Context)`:
   - Tại vòng lặp duyệt qua `itemsMap` (khoảng dòng 335–347):
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
3. Bổ sung hàm ủy nhiệm trên `AccountManager`:
   ```go
   func (m *AccountManager) SetAccountStatus(email string, status string) error {
       if m.accountStore == nil {
           return errors.New("account store is nil")
       }
       return m.accountStore.SetAccountStatus(email, status)
   }
   ```
4. Bổ sung getter cho `accountStore` (nếu chưa có) để các tầng ngoài truy cập:
   ```go
   func (m *AccountManager) AccountStore() *store.AccountStore {
       return m.accountStore
   }
   ```

---

### PHASE 2: TẦNG ĐÁNH GIÁ ỨNG VIÊN & DỪNG AN TOÀN (AUTOMATION & TELEMETRY)
*(Thực hiện thứ hai để đảm bảo backend tự động nhận thức được trạng thái Bật/Tắt và cơ chế dừng)*

#### 2.1. Cập nhật `internal/automation/sorter.go`
1. Mở rộng struct `CandidateAccount`:
   ```go
   type CandidateAccount struct {
       Email        string                      `json:"email"`
       Index        int                         `json:"index"`
       Status       string                      `json:"status"` // "on" or "off"
       TokenValid   bool                        `json:"token_valid"`
       RefreshToken string                      `json:"-"`
       HasFreshPoll bool                        `json:"has_fresh_poll"`
       Evaluation   ModelQuotaEvaluation        `json:"evaluation"`
       RawQuota     *telemetry.AccountQuotaData `json:"raw_quota,omitempty"`
   }
   ```
2. Cập nhật hàm `FilterEligibleCandidates`:
   - Bổ sung kiểm tra trạng thái vô hiệu hóa:
     ```go
     // Criterion: Must not be disabled (status == "off")
     if strings.EqualFold(cand.Status, "off") {
         exclusions = append(exclusions, CandidateExclusion{
             Email:     cand.Email,
             Criterion: 6,
             Reason:    "tài khoản đang ở chế độ vô hiệu hóa (OFF)",
         })
         continue
     }
     ```

#### 2.2. Cập nhật `internal/automation/controller.go`
1. Khi thu thập ứng viên trong `triggerSwitch` (khoảng dòng 490–515):
   - Đọc trạng thái `status` từ store:
     ```go
     status := "on"
     if c.mgr != nil && c.mgr.AccountStore() != nil {
         if st, err := c.mgr.AccountStore().GetAccountStatus(email); err == nil && st != "" {
             status = st
         }
     }
     cand := BuildCandidateAccount(...)
     cand.Status = status
     candidates = append(candidates, cand)
     ```
2. Cập nhật khối xử lý khi cạn kiệt ứng viên (`len(eligible) == 0` - khoảng dòng 516–521):
   - **Thực thi Dừng an toàn (Safety Stop):**
     ```go
     eligible, exclusions := FilterEligibleCandidates(candidates, activeEmail, failedMap)
     if len(eligible) == 0 {
         // 1. Tắt cờ tự động trong controller
         c.enabled.Store(false)
         c.syncMonitoring()

         // 2. Bắn sự kiện với câu thông báo chuẩn mực của Owner
         c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", exclusions)
         return fmt.Errorf("%w: %+v", ErrNoCandidates, exclusions)
     }
     ```
   - Tương tự tại nhánh preflight tất cả ứng viên đều thất bại (khoảng dòng 568–572):
     ```go
     c.enabled.Store(false)
     c.syncMonitoring()
     c.EmitStatus(StatusNoCandidate, "Không có tài khoản để chuyển.", nil)
     return ErrNoCandidates
     ```

#### 2.3. Cập nhật `internal/telemetry/pool.go`
1. Trong phương thức `PollAllAccounts` (hoặc các luồng định kỳ quét danh sách):
   - Bỏ qua các tài khoản mang trạng thái `status == "off"` để tránh lãng phí request và bảo vệ `Invariant-Disabled-Telemetry-Silence`.

---

### PHASE 3: TẦNG THÀNH PHẦN GIAO DIỆN THẺ (UI WIDGETS)
*(Thực hiện thứ ba để hoàn thiện widget thẻ có khả năng hiển thị `[ON]` / `[OFF]` và phản ứng làm mờ)*

#### 3.1. Cập nhật `internal/ui/widgets.go`
1. Tạo widget nút Text trạng thái `statusTextButton` (tương tự như `switchTextButton`):
   ```go
   type statusTextButton struct {
       widget.BaseWidget
       Status   string // "on" or "off"
       OnTapped func()
       hovered  bool
       palette  ThemePalette
       bg       *canvas.Rectangle
       text     *canvas.Text
   }
   ```
   - Khởi tạo:
     - Chữ hiển thị: Nếu `Status == "off"` thì hiện `"[OFF]"`, màu chữ `palette.TextMuted`. Nếu `Status == "on"` thì hiện `"[ON]"`, màu chữ `palette.Success`. Cỡ chữ 10, in đậm (`Bold: true`), căn giữa.
     - Hiệu ứng rê chuột: Nền hơi sáng nhẹ khi hover. Con trỏ chuột: `desktop.PointerCursor`.
     - Sự kiện `Tapped`: Gọi `OnTapped()` nếu khác nil.
2. Mở rộng `AccountCardWidget`:
   - Thêm trường `Status string`.
   - Thêm trường `statusButton *statusTextButton`.
   - Thêm callback `onToggleStatus func(email string)`.
   - Thêm trường `switchBtn *switchTextButton` để dễ dàng điều khiển trạng thái enable/disable của nút chuyển mà không phải tạo lại object.
3. Cập nhật hàm `NewAccountCardWidgetWithQuota`:
   - Nhận thêm callback `onToggleStatus func(email string)` (hoặc truyền qua setter).
   - Khởi tạo `statusBtn`:
     ```go
     statusBtn := newStatusTextButton(item.Status, func() {
         if w.onToggleStatus != nil {
             w.onToggleStatus(w.Email)
         }
     }, palette)
     ```
   - Cập nhật `actionCluster`: Bố trí nút `statusBtn` nằm ở **bên phải nút `refreshBtn`**:
     ```go
     actionCluster := container.NewHBox(actionSlot, compactButton(refreshBtn), compactButton(statusBtn))
     ```
4. Cập nhật phương thức `UpdateState(item manager.AccountDisplayItem, palette ThemePalette)`:
   - Cập nhật `w.Status = item.Status`.
   - **Xử lý tài khoản `[Cần OAuth]`:**
     - Nếu `!item.HasToken && !item.IsActive`:
       - Gọi `w.statusButton.Hide()` (Ẩn hoàn toàn nút `[ON/OFF]`).
     - Ngược lại:
       - Gọi `w.statusButton.Show()` và cập nhật nhãn `[ON]` hoặc `[OFF]`.
   - **Xử lý nút `[Chuyển]` khi `w.Status == "off"`:**
     - Nếu tài khoản không active và có token (`!w.IsActive && w.HasToken`):
       - Nếu `w.Status == "off"`:
         - Nút `[Chuyển]` được vẽ với màu chữ xám mờ `palette.TextMuted`.
         - Con trỏ chuột là `desktop.DefaultCursor`.
         - Handler click là no-op (không gọi `onSwitch`).
       - Nếu `w.Status == "on"`:
         - Nút `[Chuyển]` vẽ bình thường với màu xanh `palette.Primary`, con trỏ `PointerCursor`, click gọi `onSwitch(w.Email)`.
   - **Xử lý nút `[🔄]`:**
     - Nếu `w.Status == "off"`: Gọi `w.refreshButton.Disable()`.
     - Nếu `w.Status == "on"` và (`w.IsActive || w.HasToken`): Gọi `w.refreshButton.Enable()`.

---

### PHASE 4: TẦNG ĐIỀU PHỐI GIAO DIỆN & XỬ LÝ SỰ KIỆN (UI APP)
*(Thực hiện thứ tư để liên kết sự kiện click với backend và xử lý đồng bộ giao diện)*

#### 4.1. Cập nhật `internal/ui/app.go`
1. Bổ sung hàm xử lý sự kiện Bật/Tắt tài khoản trên `App`:
   ```go
   func (a *App) ToggleAccountStatus(email string)
   ```
   - Lấy trạng thái hiện tại của email từ danh sách overview hoặc store.
   - Trạng thái mới: Nếu đang `"off"` thì đổi thành `"on"`, nếu đang `"on"` thì đổi thành `"off"`.
   - Gọi `a.mgr.SetAccountStatus(email, newStatus)`.
   - Cập nhật trạng thái item tương ứng trong `a.overview.Accounts`.
   - Cập nhật thẻ tương ứng tại chỗ thông qua `a.accountCards[targetIdx].UpdateState(...)`.
   - **Đặc biệt — Xử lý khi TẮT (`"off"`) chính tài khoản đang Active (`email == activeEmail`):**
     ```go
     if email == activeEmail && newStatus == "off" {
         a.mu.RLock()
         isAuto := a.autoSwitchEnabled
         ctrl := a.controller
         a.mu.RUnlock()

         if isAuto && ctrl != nil {
             go func() {
                 _, err := ctrl.TriggerSwitch(context.Background(), "tài khoản active bị tắt")
                 if errors.Is(err, automation.ErrNoCandidates) {
                     // Đã có OnStatusChange xử lý tắt toggle và hiển thị thông báo
                 }
             }()
         }
     }
     ```
2. Cập nhật bộ lắng nghe trạng thái Automation `OnStatusChange` trong `internal/ui/app.go`:
   - Tại nhánh xử lý `case automation.StatusNoCandidate:` (khoảng dòng 451–454):
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

         // 2. Ghi bền vững trạng thái auto_switch: false vào logs/ui_preferences.json
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
3. Truyền hàm callback `a.ToggleAccountStatus` vào các điểm khởi tạo và tái tạo thẻ tài khoản (`NewAccountCardWidgetWithQuota`).

---

### PHASE 5: KIỂM THỬ TỰ ĐỘNG & NGHIỆM THU (TESTING SUITE)
*(Thực hiện cuối cùng để xác nhận không gây hồi quy và đáp ứng 100% tiêu chí nghiệm thu)*

#### 5.1. Kiểm thử Unit Test Store (`internal/store/account_store_test.go`)
- `TestAccountStore_StatusPersistence`: Lưu trạng thái `"off"`, mở lại kiểm tra đúng `"off"`.
- `TestAccountStore_BackwardCompatibility_DefaultOn`: Tạo mock file không có trường `status`, đọc lên kiểm tra trả về `"on"`.
- `TestAccountStore_SaveAccount_PreservesStatus`: Tạo file với `status: "off"`, gọi `SaveAccount` với token mới, kiểm tra `status` vẫn là `"off"`.

#### 5.2. Kiểm thử Automation (`internal/automation/controller_test.go` & `sorter_test.go`)
- `TestFilterEligibleCandidates_ExcludesDisabledAccount`: Đưa 3 ứng viên, trong đó 1 ứng viên có `Status = "off"`. Kiểm tra ứng viên này bị loại và nằm trong danh sách `exclusions`.
- `TestController_SafetyStop_WhenAllDisabled`: Giả lập tất cả ứng viên dự phòng đều có `Status = "off"`. Kích hoạt `TriggerSwitch`:
  - Kiểm tra hàm trả về `ErrNoCandidates`.
  - Kiểm tra `controller.IsEnabled()` chuyển thành `false`.
  - Kiểm tra status phát ra là `StatusNoCandidate` với thông báo `"Không có tài khoản để chuyển."`.

#### 5.3. Kiểm thử Giao diện (`internal/ui/ui_test.go`)
- `TestAccountCardWidget_StatusToggle`: Tạo thẻ với `Status = "on"`, kiểm tra nhãn hiển thị `[ON]`. Bấm nút chuyển sang `[OFF]`, kiểm tra nút `[Chuyển]` bị mờ và không kích hoạt switch callback.
- `TestAccountCardWidget_UnauthHidesStatusButton`: Tạo thẻ cho tài khoản `[Cần OAuth]`, kiểm tra nút status bị ẩn (`statusBtn.Visible() == false`).
- `TestApp_StatusNoCandidate_DisablesAutoSwitchAndSavesPref`: Phát sự kiện `StatusNoCandidate`, kiểm tra `autoSwitchEnabled == false`, `autoSwitchCheck.Checked == false`, và file `ui_preferences.json` được cập nhật `"auto_switch": false`.

---

## V. ĐỐI CHIẾU DANH MỤC TỆP TIN THAY ĐỔI

| STT | Tệp tin | Thư mục | Bản chất thay đổi | Phase |
|:---:|---|---|---|:---:|
| 1 | `account_store.go` | `internal/store/` | Bổ sung `Status`, `SetAccountStatus`, `GetAccountStatus`, logic bảo tồn status trong `SaveAccount`. | Phase 1 |
| 2 | `account_manager.go` | `internal/manager/` | Bổ sung `Status` vào `AccountDisplayItem`, map status trong `GetAccountStates`, thêm `SetAccountStatus`. | Phase 1 |
| 3 | `sorter.go` | `internal/automation/` | Bổ sung `Status` vào `CandidateAccount`, thêm Criterion loại trừ tài khoản `"off"`. | Phase 2 |
| 4 | `controller.go` | `internal/automation/` | Đọc `Status` khi build candidate, kích hoạt Dừng tự động (`enabled = false`) khi cạn ứng viên. | Phase 2 |
| 5 | `pool.go` | `internal/telemetry/` | Bỏ qua quét quota ngầm cho tài khoản `"off"`. | Phase 2 |
| 6 | `widgets.go` | `internal/ui/` | Tạo `statusTextButton`, đặt bên phải `[🔄]`, làm mờ nút `[Chuyển]`, ẩn nút với `[Cần OAuth]`. | Phase 3 |
| 7 | `app.go` | `internal/ui/` | Thêm `ToggleAccountStatus`, xử lý tắt active, bắt `StatusNoCandidate` gạt tắt Toggle và lưu JSON. | Phase 4 |
| 8 | Các tệp `*_test.go` | `internal/*/` | Bộ test tự động bảo đảm bất biến và tiêu chí nghiệm thu. | Phase 5 |

---

## VI. BÀN GIAO VÀ CHỈ DẪN CHO LANE TIẾP THEO

Bản văn kiện này là chỉ dẫn kỹ thuật đầy đủ và trực tiếp nhất cho các Lane kế tiếp:
- **Planner Lane:** Có thể trích xuất trực tiếp danh mục Phase tại **Mục III** và **Mục IV** để lập tiến độ phân rã Jobs mà không cần tái cấu trúc.
- **Coder Lane:** Triển khai tuần tự từ Phase 1 đến Phase 4 theo đúng mã lệnh và phương thức đã chỉ định chi tiết; không cần phải phỏng đoán thứ tự trước sau hay ranh giới dữ liệu.
- **QA & Supervisor:** Sử dụng ma trận tiêu chí tại **Mục V** của phần I và danh mục test tại **Phase 5** làm căn cứ nghiệm thu tính năng.
