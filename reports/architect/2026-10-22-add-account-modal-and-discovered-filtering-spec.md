# SPEC-ARCH-20261022: Modal Thêm Tài Khoản và Cơ Chế Lọc Danh Bạ Chrome

- **Tài liệu:** Đặc tả Kỹ thuật Kiến trúc Phần mềm (Software Architecture Specification)
- **Tác giả:** System Architect
- **Ngày lập:** 2026-10-22
- **Trạng thái:** APPROVED / READY FOR IMPLEMENTATION
- **Đối tượng đọc:** Coder / Implementer (Đọc vào triển khai trực tiếp không cần suy đoán logic)

---

## 1. Mục Tiêu & Động Lực Kiến Trúc (Architecture Motivation)

1. **Vấn đề cần giải quyết:**
   - Trước đây, `GetAccountStates` tự động quét các profile Chrome trên máy (`ScanChromeAccountDetails`) ở mọi thời điểm: khởi động app, sau khi chuyển tài khoản thủ công, và sau mỗi lần auto-switch.
   - Toàn bộ email trong Chrome đều bị tự động ghi vào `discovered_accounts/` và đẩy thẳng lên UI tổng dưới dạng thẻ `[Cần OAuth]`, gây ô nhiễm thị giác và nghẽn I/O.
2. **Mục tiêu kiến trúc mới:**
   - **Tách rời 100%:** Quét Chrome là thao tác On-Demand (chỉ chạy khi bấm nút trong Modal), không bao giờ tự ý chạy ngầm.
   - **Lọc hiển thị UI tổng:** UI tổng chỉ hiển thị tài khoản đã có token (`accounts_store/`), tài khoản Active, và tài khoản được người dùng chủ động bấm `[ + ADD ]` (`added: true`).
   - **Modal `[+ Thêm]`:** Cung cấp giao diện trực quan với 3 trạng thái phân cấp (`[✔ Đã OAuth]`, `[Cần OAuth]` + `[ + ADD ]`, `[ Đã add ]`) và nút nhỏ `[🔄 Quét Chrome]`.
   - **Ghi nhớ bền vững (Phương án 1):** Cờ `added: true` được lưu đĩa để khi khởi động lại app, thẻ `[Cần OAuth]` vẫn được bảo toàn.

---

## 2. Kiến Trúc Sẵn Có Cần Tận Dụng 100% (Reusable Assets)

Coder **bắt buộc phải tái sử dụng** các thành phần kiến trúc đã hoàn thiện trong codebase, tuyệt đối không viết lại:

1. **Widget Thẻ Tài Khoản (`internal/ui/widgets.go`):**
   - Tận dụng 100% logic hiển thị thẻ `[Cần OAuth]` sẵn có tại dòng 578-586:
     ```go
     text, tint := "[Cần OAuth]", palette.Warning
     badge := canvas.NewText(text, tint)
     // ... actionSlot = container.NewCenter(badge)
     ```
   - Tận dụng logic hiển thị Quota ma trận `N/A` khi thẻ chưa có token (`!w.IsActive && !w.HasToken`) tại dòng 517-524.
   - Tận dụng cơ chế cập nhật tại chỗ `UpdateState` và `UpdateQuota` để tránh giật lag layout.
2. **Scanner Chrome (`internal/scanner/chrome_scanner.go`):**
   - Tái sử dụng hàm quét chuẩn `scanner.ScanChromeAccountDetails(cDir)` và hàm phân giải thư mục `scanner.GetDefaultChromeUserDataDir()`.
3. **Cơ Chế Ghi Tệp An Toàn (Atomic Temp File + Rename Pattern):**
   - Tái sử dụng mẫu ghi tệp nguyên tử bằng `os.CreateTemp` + `Sync()` + `os.Rename` đã chuẩn hóa tại `internal/store/account_store.go` và `internal/store/discovered_store.go`.
4. **Hệ Thống Bảng Màu & Giao Diện (`internal/ui/themes.go`):**
   - Tái sử dụng `ThemePalette`, `GetThemePalette(a.currentTheme)` để Modal tự động đồng bộ theo 3 theme `Dark`, `Light`, `Warm`.

---

## 3. Phân Tích Tác Động & Các Bất Biến Được Bảo Toàn (Impact & Invariants)

### 3.1. Các Kiến Trúc Được Bảo Toàn Tuyệt Đối (Preserved Invariants)
- **Invariant-Auto-Switch-Immunity:** Bộ điều khiển `AutomationController` và thuật toán `Burn-down Sorter` chỉ lọc ứng viên có token sống trong `accounts_store/`. Thẻ `[Cần OAuth]` có `HasToken: false` nên tự động bị loại khỏi danh sách chuyển đổi, đảm bảo an toàn tuyệt đối cho daemon tự động.
- **Invariant-Telemetry-Loop-Safety:** Vòng lặp 60s `AutoTelemetryLoop` chỉ truy vấn quota cho các tài khoản có token trong `accounts_store/`. Thẻ `[Cần OAuth]` không bao giờ kích hoạt request Google API ngầm.
- **Invariant-Hot-Swap-Integrity:** Toàn bộ luồng 9 bước của `SwitchAccount` (Connect-RPC probe, snapshot active threads, WinCred swap, recycle Language Server) giữ nguyên 100%.
- **Invariant-Grid-Layout-Stability:** Lưới thẻ `accountListContainer` (280x130 symmetric scroller) giữ nguyên cấu trúc. Khi nạp thêm thẻ `[Cần OAuth]`, lưới tự động mở rộng theo dòng mà không phá vỡ khung cửa sổ.
- **Invariant-Card-Signature-Compatibility:** Giữ nguyên 100% chữ ký hàm `NewAccountCardWidgetWithQuota` và `NewAccountCardWithQuota`, không gây ảnh hưởng đến 40+ unit test hiện có.

### 3.2. Các Kiến Trúc Bị Thay Đổi (Impacted Surfaces)
1. **`internal/store/discovered_store.go`**:
   - `DiscoveredAccountEnvelope` bổ sung trường `Added bool`.
   - `SaveDiscovered` phải bảo tồn giá trị `Added == true` nếu file đã tồn tại trên đĩa (tránh việc quét lại Chrome làm reset cờ).
2. **`internal/manager/account_manager.go`**:
   - `GetAccountStates`: Cắt bỏ lệnh gọi tự động `scanner.ScanChromeAccountDetails`. Chỉ đọc từ đĩa và chỉ lọc các bản ghi `discovered` có `Added == true`.
   - Bổ sung các phương thức chuyên biệt: `ScanChromeAccounts`, `GetCandidateAccounts`, `AddDiscoveredAccount`.
3. **`internal/ui/window.go`**:
   - Bổ sung nút `[+ Thêm]` vào Toolbar Header Row 2.
4. **`internal/ui/app.go`**:
   - Khởi động app (`Init`) không còn độ trễ quét Chrome.
   - Thêm phương thức điều phối Modal `ShowAddAccountModal`.

---

## 4. Đặc Tả Chi Tiết Từng File & Hàm Cụ Thể (File-by-File Blueprint)

### 4.1. `internal/store/discovered_store.go`

#### A. Cập nhật Struct `DiscoveredAccountEnvelope`:
```go
type DiscoveredAccountEnvelope struct {
	Email     string  `json:"email"`
	ScannedAt float64 `json:"scanned_at"`
	Added     bool    `json:"added,omitempty"`
}
```
*Ghi chú:* Thẻ `omitempty` đảm bảo tương thích ngược 100% với các file JSON cũ trên đĩa (mặc định unmarshal ra `false`).

#### B. Cập nhật `SaveDiscovered(email string) error`:
- Trước khi ghi đè, kiểm tra xem file `<email>.json` đã tồn tại chưa:
  - Nếu đã tồn tại và parse được `Added == true`, bảo toàn `entry.Added = true` (không reset về `false`).
- Thực hiện ghi nguyên tử: `os.CreateTemp` -> `json.Encode` -> `tmp.Sync()` -> `os.Rename`.

#### C. Bổ sung `SetDiscoveredAdded(email string, added bool) error`:
```go
// SetDiscoveredAdded updates the Added flag for an existing or new discovered record.
func (s *DiscoveredStore) SetDiscoveredAdded(email string, added bool) error
```
- Chuẩn hóa `email = strings.ToLower(strings.TrimSpace(email))`.
- Đọc file hiện tại (nếu có để giữ `ScannedAt`), gán `record.Added = added`. Nếu chưa có file, gán `ScannedAt = now`.
- Ghi đĩa nguyên tử dưới khóa `s.mu.Lock()`.

#### D. Bổ sung `ListDiscoveredRecords() ([]DiscoveredAccountEnvelope, error)`:
```go
// ListDiscoveredRecords returns full envelopes including Added flag for all valid records.
func (s *DiscoveredStore) ListDiscoveredRecords() ([]DiscoveredAccountEnvelope, error)
```
- Duyệt thư mục `s.dir`, đọc toàn bộ các file `.json`, trả về danh sách `[]DiscoveredAccountEnvelope` đã sắp xếp theo email.

---

### 4.2. `internal/manager/account_manager.go`

#### A. Tách biệt `GetAccountStates(ctx context.Context) (*AccountOverview, error)`:
- **XÓA BỎ HOÀN TOÀN:**
  ```go
  // CẮT BỎ ĐOẠN NÀY KHỎI GetAccountStates:
  chromeAccounts, scanErr := scanner.ScanChromeAccountDetails(cDir)
  for _, ca := range chromeAccounts { ... discovered.SaveDiscovered ... }
  ```
- **LUỒNG MỚI CỦA `GetAccountStates`:**
  1. `m.CaptureCurrentSession()` (bảo toàn phiên active).
  2. `activeEmail, _ := m.GetCurrentActiveEmail(ctx)`.
  3. Đọc danh sách có token: `storedEmails, _ := m.store.ListAccounts()`.
  4. Đọc danh bạ Chrome: `records, _ := m.discoveredStore.ListDiscoveredRecords()`.
  5. Lọc danh bạ đưa vào UI:
     ```go
     for _, rec := range records {
         // CHỈ nạp vào itemsMap nếu tài khoản đã được người dùng bấm ADD
         if rec.Added {
             add(rec.Email)
         }
     }
     ```
  6. Nạp `storedEmails` và `activeEmail` vào `itemsMap`.
  7. Phân cấp trạng thái thẻ:
     - `item.IsActive` -> `StateActive`, `[✔ Dùng]`
     - `item.HasToken` -> `StateReady`, `[Chuyển]`
     - Còn lại -> `StateUnauth`, `[Cần OAuth]`
  8. Trả về `*AccountOverview`.

#### B. Bổ sung `ScanChromeAccounts() ([]string, error)`:
```go
// ScanChromeAccounts scans Chrome profiles on-demand and syncs new accounts into discoveredStore.
func (m *AccountManager) ScanChromeAccounts() ([]string, error) {
    m.mu.RLock()
    cDir, discovered := m.chromeDir, m.discoveredStore
    m.mu.RUnlock()

    if discovered == nil {
        return nil, errors.New("discovered store uninitialized")
    }

    details, err := scanner.ScanChromeAccountDetails(cDir)
    if err != nil {
        return nil, fmt.Errorf("lỗi quét chrome: %w", err)
    }

    var scannedEmails []string
    for _, d := range details {
        em := strings.ToLower(strings.TrimSpace(d.Email))
        if em == "" {
            continue
        }
        _ = discovered.SaveDiscovered(em) // Đã có logic bảo toàn cờ Added
        scannedEmails = append(scannedEmails, em)
    }
    return scannedEmails, nil
}
```

#### C. Bổ sung `GetCandidateAccounts() ([]CandidateAccountItem, error)`:
```go
type CandidateAccountItem struct {
    Email    string
    HasToken bool
    Added    bool
}

// GetCandidateAccounts returns all discovered accounts enriched with token and added state.
func (m *AccountManager) GetCandidateAccounts() ([]CandidateAccountItem, error)
```
- Lấy `storedSet` từ `m.store.ListAccounts()`.
- Lấy `records` từ `m.discoveredStore.ListDiscoveredRecords()`.
- Ghép nối dữ liệu:
  ```go
  item := CandidateAccountItem{
      Email: rec.Email,
      HasToken: storedSet[rec.Email],
      Added: rec.Added,
  }
  ```
- Sắp xếp: Ưu tiên `HasToken`, tiếp theo đến `Added`, cuối cùng theo `Email` A-Z.

#### D. Bổ sung `AddDiscoveredAccount(email string) error`:
```go
func (m *AccountManager) AddDiscoveredAccount(email string) error {
    m.mu.RLock()
    discovered := m.discoveredStore
    m.mu.RUnlock()
    if discovered == nil {
        return errors.New("discovered store uninitialized")
    }
    return discovered.SetDiscoveredAdded(email, true)
}
```

---

### 4.3. Tạo Mới `internal/ui/add_modal.go`

Module này chịu trách nhiệm hiển thị Modal "Thêm tài khoản" độc lập.

#### A. Cấu trúc và Vòng đời Modal:
```go
type AddAccountModal struct {
    app       *App
    win       fyne.Window
    dialog    dialog.Dialog
    listVBox  *fyne.Container
    refreshBtn *widget.Button
}
```

#### B. Thiết kế Layout Modal:
- **Kích thước khuyến nghị:** Rộng `440px`, Cao `320px`.
- **Header:**
  ```go
  titleLbl := widget.NewLabelWithStyle("Thêm tài khoản", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
  refreshBtn := widget.NewButtonWithIcon("Quét Chrome", theme.ViewRefreshIcon(), func() {
      // 1. Đổi text nút thành "⏳ Đang quét..." và Disable
      // 2. Chạy goroutine gọi m.ScanChromeAccounts()
      // 3. fyne.Do tải lại danh sách trên Modal, khôi phục text nút
  })
  header := container.NewBorder(nil, nil, titleLbl, refreshBtn)
  ```
- **Nội dung cuộn (List VBox inside VScroll):**
  Mỗi tài khoản được biểu diễn bởi một hàng `container.NewBorder(nil, nil, emailLabel, actionCluster)`:
  - `emailLabel`: Truncation Ellipsis, TextStyle Bold nếu `HasToken`.
  - `actionCluster`:
    - **Trường hợp 1 (`item.HasToken == true`):**
      `canvas.NewText("[✔ Đã OAuth]", palette.Success)`
    - **Trường hợp 2 (`!item.HasToken && item.Added`):**
      `canvas.NewText("[Đã add]", palette.TextMuted)`
    - **Trường hợp 3 (`!item.HasToken && !item.Added`):**
      Cụm `container.NewHBox(statusBadge, addBtn)`:
      - `statusBadge := canvas.NewText("[Cần OAuth]", palette.Warning)`
      - `addBtn := widget.NewButtonWithIcon("ADD", theme.ContentAddIcon(), func() { ... })`
      - **Hành vi khi bấm `addBtn`:**
        1. Gọi `m.AddDiscoveredAccount(item.Email)`.
        2. Chuyển đổi giao diện hàng ngay lập tức: Xóa `addBtn`, đổi `statusBadge` thành `canvas.NewText("[Đã add]", palette.TextMuted)`.
        3. Kích hoạt cập nhật UI tổng:
           ```go
           overview, _ := a.mgr.GetAccountStates(context.Background())
           a.mu.Lock()
           a.overview = overview
           a.mu.Unlock()
           a.updateUI(overview)
           ```
- **Footer:** Nút `[Đóng]` gọi `dialog.Hide()`.

---

### 4.4. Cập nhật Toolbar `internal/ui/window.go`

Tại hàm `setupLayout()` ([L141-L150](file:///g:/antigravity_auto_switch/internal/ui/window.go#L141-L150)):
- Bổ sung nút `a.addAccountBtn`:
  ```go
  a.addAccountBtn = widget.NewButtonWithIcon("Thêm", theme.ContentAddIcon(), func() {
      a.ShowAddAccountModal()
  })
  ```
- Sắp xếp lại `toolbarLeft`:
  ```go
  toolbarLeft := container.NewHBox(
      compactButton(a.addAccountBtn),
      compactButton(a.quotaRefreshAllBtn),
      compactButton(a.alwaysOnTopBtn),
  )
  ```
*(Nút `a.refreshBtn` "Danh sách tài khoản" cũ được thay thế hoàn toàn bởi `a.addAccountBtn`, do chức năng quét Chrome đã chuyển trọn vẹn vào trong Modal).*

---

## 5. Quy Chuẩn Kiểm Thử & Tiêu Chí Chấp Nhận (Acceptance Test Matrix)

Coder phải đảm bảo toàn bộ 8 bài kiểm thử sau vượt qua tuyệt đối:

| Mã Test | Kịch Bản Kiểm Thử | Điều Kiện Đầu Vào | Kết Quả Mong Đợi (Pass Criteria) |
|---|---|---|---|
| **TC-01** | Khởi động ứng dụng sạch | `discovered_accounts/` có 5 email (`Added: false`), `accounts_store/` có 2 email | UI tổng hiển thị đúng 2 thẻ từ `accounts_store/`. Tuyệt đối không xuất hiện 5 thẻ từ Chrome. |
| **TC-02** | Mở Modal `[+ Thêm]` | 1 email có token, 1 email đã add, 3 email chưa add | Modal hiển thị đúng 3 trạng thái: 1 dòng `[✔ Đã OAuth]`, 1 dòng `[Đã add]`, 3 dòng `[Cần OAuth]` + `[ + ADD ]`. |
| **TC-03** | Nhấn nút `[ + ADD ]` | Nhấn `[ + ADD ]` tại email `test@gmail.com` | Nhãn `[Cần OAuth]` đổi thành `[Đã add]`, nút `[ + ADD ]` biến mất. Thẻ `test@gmail.com` xuất hiện ngay ngoài UI tổng với badge `[Cần OAuth]` và Quota `N/A`. |
| **TC-04** | Kiểm tra tính bền vững (Restart App) | Tắt app sau khi hoàn thành TC-03 rồi mở lại | Thẻ `test@gmail.com` vẫn hiển thị nguyên vẹn trên UI tổng ở trạng thái `[Cần OAuth]`. |
| **TC-05** | Quét Chrome On-Demand | Mở Modal, bấm nút `[🔄 Quét Chrome]` | Quét các Profile Chrome, ghi nhận email mới vào `discovered_accounts/`. Danh sách Modal nạp thêm dòng mới. UI tổng không bị thêm thẻ tự động. |
| **TC-06** | Hoàn tất OAuth sau khi add | Mở Antigravity đăng nhập email `test@gmail.com` | Token lưu vào `accounts_store/`. Thẻ UI tổng đổi thành `[Chuyển]` / `[✔ Dùng]`. Mở Modal thấy chuyển thành `[✔ Đã OAuth]`. |
| **TC-07** | An toàn Auto-Switch & Telemetry | Chạy Auto-Switch và telemetry loop 60s khi có thẻ `[Cần OAuth]` | Burn-down Sorter tự động bỏ qua thẻ `[Cần OAuth]`. Không có panic, không có crash, không gọi API thừa. |
| **TC-08** | Hiệu năng & Tốc độ Khởi động | Đo thời gian chạy `Init()` | Thời gian nạp UI tổng dưới 50ms, không bị nghẽn bởi I/O của Google Chrome. |
