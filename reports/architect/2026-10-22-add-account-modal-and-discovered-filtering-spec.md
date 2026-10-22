# ĐẶC TẢ KIẾN TRÚC & KỸ THUẬT TOÀN DIỆN: MODAL THÊM TÀI KHOẢN & PHÂN TÁCH QUÉT CHROME

- **Tài liệu:** Đặc tả Kỹ thuật Kiến trúc & Hướng dẫn Triển khai (Comprehensive Software Architecture & Implementation Blueprint)
- **Tác giả:** System Architect
- **Ngày lập:** 2026-10-22
- **Trạng thái:** APPROVED / READY FOR IMPLEMENTATION
- **Đối tượng đọc:** Coder / Implementer (Đọc vào triển khai trực tiếp không cần suy đoán logic)

---

# PHẦN I: BỐI CẢNH, NGUYÊN NHÂN GỐC RỄ & THIẾT KẾ TRẢI NGHIỆM NGƯỜI DÙNG (UX SPECIFICATION)

## 1. Bối Cảnh & Nguyên Nhân Gốc Rễ (Root Cause)

### 1.1. Hiện trạng Codebase cũ
Hệ thống hiện tại đang bị **trộn lẫn** giữa hai trách nhiệm hoàn toàn khác nhau: **Quét dữ liệu Google Chrome** và **Hiển thị danh sách lên Giao diện UI tổng**.

Cụ thể, hàm `GetAccountStates()` trong `internal/manager/account_manager.go` đang bị kích hoạt tự động ngầm ở **4 thời điểm**:
1. **Khi mở ứng dụng (Startup):** Ngay khi hàm `Init()` chạy, hệ thống quét toàn bộ thư mục Chrome trên máy.
2. **Sau khi chuyển tài khoản thủ công:** Người dùng bấm chuyển tài khoản xong, hệ thống lại quét Chrome.
3. **Sau khi Auto-Switch chuyển tài khoản tự động:** Khi sự kiện failover diễn ra, hệ thống lại quét Chrome.
4. **Khi bấm nút "Danh sách tài khoản" trên Toolbar.**

### 1.2. Hai hệ quả nghiêm trọng
- **Ô nhiễm thị giác (UI Pollution):** Trình duyệt Chrome của người dùng thường có rất nhiều tài khoản (email cá nhân, mail mua sắm, mail học tập...). Hệ thống quét tự động và ép **toàn bộ** các email này hiển thị kín màn hình dưới dạng thẻ `[Cần OAuth]`, dù người dùng không có nhu cầu sử dụng chúng cho Antigravity.
- **Thẻ "đội mồ sống lại":** Do Chrome liên tục bị quét ngầm ở mọi sự kiện, bất kỳ thao tác nào nhằm gỡ hay xoá thẻ `[Cần OAuth]` đều vô ích, vì chỉ cần khởi động lại app hoặc chuyển tài khoản là Chrome lại quét ra và tự động ghi đè trở lại.

---

## 2. Nguyên Lý Kiến Trúc Mới (Separation of Concerns)

Tách rời triệt để hệ thống thành **3 tầng độc lập**:

```mermaid
flowchart TD
    subgraph Tầng_1["TẦNG 1: KHO ĐỆM ỨNG VIÊN (Candidate Pool)"]
        Chrome["Google Chrome Profiles trên máy"] -->|Bấm [🔄 Quét Chrome] trong Modal| Discovered["discovered_accounts/<email>.json<br/>(Chỉ là danh bạ ứng viên, KHÔNG tự ý lên UI)"]
    end

    subgraph Tầng_2["TẦNG 2: BẢNG ĐIỀU KHIỂN CHÍNH (Working Dashboard)"]
        Discovered -->|Bấm [+ ADD] trong Modal| Pending["Thẻ [Cần OAuth] trên UI tổng<br/>(Lưu cờ bền vững: added = true)"]
        Pending -->|Đăng nhập Antigravity IDE| Vault["accounts_store/<email>.json<br/>(Kho Token/Credential chính thức)"]
        Vault -->|Render tự động| Active["Thẻ [Chuyển] hoặc [✔ Dùng]"]
    end
```

1. **Tầng 1 — Kho đệm ứng viên (`discovered_accounts/`):**
   - Đóng vai trò là danh bạ ứng viên tạm thời.
   - Việc quét Chrome chuyển hẳn sang cơ chế **On-Demand (Chỉ quét khi được yêu cầu)** thông qua nút Refresh trong Modal Thêm tài khoản.
   - Tuyệt đối không tự ý đẩy dữ liệu lên UI tổng.
2. **Tầng 2 — Bảng điều khiển làm việc (UI Tổng):**
   - Hoàn toàn không can thiệp hay đọc từ Chrome.
   - Tốc độ khởi động tức thì (< 50ms) vì chỉ đọc từ đĩa cục bộ.
   - **Chỉ hiển thị 3 nguồn tài khoản hợp lệ:**
     - Các tài khoản đã có token trong `accounts_store/` (hiển thị thẻ `[Chuyển]`).
     - Tài khoản đang Active trong Windows Credential Manager (hiển thị thẻ `[✔ Dùng]`).
     - Các tài khoản mà người dùng **chủ động bấm `[ + ADD ]`** từ Modal (hiển thị thẻ `[Cần OAuth]`).

---

## 3. Thiết Kế Chi Tiết Giao Diện & Trải Nghiệm Người Dùng (UX Blueprint)

### 3.1. Điểm kích hoạt trên Giao diện chính (UI Tổng)
- Trên thanh Toolbar (Header Row 2), thay thế nút "Danh sách tài khoản" cũ bằng nút: **`[+ Thêm]`** (với icon `theme.ContentAddIcon()`), đặt cạnh cụm điều khiển:
  ```text
  [Hoạt động | PID 1234]    [ [+ Thêm]  [Làm mới Quota]  [📌] ] ...
  ```
- Khi người dùng nhấn nút này, **Modal "Thêm tài khoản"** sẽ hiển thị chính giữa cửa sổ ứng dụng.

---

### 3.2. Sơ đồ Giao diện & Hành vi tương tác trong Modal

```text
┌─────────────────────────────────────────────────────────────────┐
│  Thêm tài khoản                              [ 🔄 Quét Chrome ] │
├─────────────────────────────────────────────────────────────────┤
│  user1@company.com                           [✔ Đã OAuth]       │
│                                                                 │
│  user2@gmail.com             [Cần OAuth]     [ + ADD ]          │
│                              (Bấm -> [Cần OAuth] chuyển thành   │
│                               [Đã add] và nút add biến mất)     │
│                                                                 │
│  user3@school.edu                            [ Đã add ]         │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                         [ Đóng ]│
└─────────────────────────────────────────────────────────────────┘
```

#### A. Phần Đầu (Header Modal):
- **Tiêu đề:** *"Thêm tài khoản"*.
- **Nút nhỏ `[🔄 Quét Chrome]`:**
  - Nằm ở góc trên bên phải của Modal.
  - Khi bấm: Quét các profile Chrome trên máy tính để tìm các email mới và ghi nhận vào `discovered_accounts/` (mặc định cờ `added: false`).
  - Danh sách trong Modal tự động làm mới tức thì mà không ảnh hưởng gì tới các thẻ ngoài UI tổng.

#### B. Danh sách Tài khoản & 3 Trạng thái phân cấp:
Đọc toàn bộ các email trong `discovered_accounts/`, mỗi dòng hiển thị 1 email với hành vi phân cấp:

1. **Trạng thái 1: `[✔ Đã OAuth]`** (Ví dụ: `user1@company.com`):
   - **Điều kiện:** Đã có file token trong `accounts_store/` hoặc đang là tài khoản Active.
   - **Hiển thị:** Bên phải hiển thị nhãn `[✔ Đã OAuth]` (màu xanh lá hoặc xám dịu).
   - **Tương tác:** Không có nút `[ + ADD ]` (vì tài khoản đã nằm sẵn trên UI tổng).

2. **Trạng thái 2: `[Cần OAuth]` + Nút `[ + ADD ]`** (Ví dụ: `user2@gmail.com`):
   - **Điều kiện:** Email tìm thấy từ Chrome nhưng chưa có token trong `accounts_store/` và chưa được đưa lên UI tổng (`added == false`).
   - **Hiển thị:** Hiển thị nhãn cảnh báo `[Cần OAuth]` và nút bấm `[ + ADD ]`.
   - **Tương tác khi người dùng click `[ + ADD ]`:**
     1. Ghi nhận cờ `added: true` vào file đĩa `discovered_accounts/user2@gmail.com.json`.
     2. Nhãn `[Cần OAuth]` tại dòng đó **lập tức chuyển thành nhãn `[Đã add]`**.
     3. Nút `[ + ADD ]` **biến mất hoàn toàn**.
     4. Ngoài UI tổng lập tức xuất hiện thêm một thẻ tài khoản mới mang trạng thái `[Cần OAuth]` (với hạn ngạch hiển thị N/A).

3. **Trạng thái 3: `[ Đã add ]`** (Ví dụ: `user3@school.edu`):
   - **Điều kiện:** Tài khoản chưa nạp token nhưng đã được người dùng nhấn `[ + ADD ]` từ trước (`added == true` trên đĩa).
   - **Hiển thị:** Chỉ hiển thị duy nhất nhãn `[ Đã add ]`, không hiển thị nút bấm nào khác.

#### C. Phần Đuôi (Footer):
- Nút **`[Đóng]`** để đóng modal quay lại giao diện chính.

---

### 3.3. Vòng đời Hoàn chỉnh của một Tài khoản (Lifecycle Journey)
```text
[Profile Chrome trên máy]
           │
           ▼ (Người dùng mở Modal, bấm "Quét Chrome")
[discovered_accounts/email.json (added: false)]
           │
           ▼ (Người dùng bấm nút "[ + ADD ]")
[discovered_accounts/email.json (added: true)] ──(Lưu bền vững trên đĩa)
           │
           ▼ (Hệ thống nạp thẻ lên UI tổng)
[Thẻ UI tổng: [Cần OAuth], Quota N/A]
           │
           ▼ (Người dùng mở Antigravity IDE, đăng nhập tài khoản đó)
[Hệ thống tự bắt credential -> Lưu vào accounts_store/email.json]
           │
           ▼ (Tự động cập nhật trạng thái)
[Thẻ UI tổng chuyển thành: [Chuyển] hoặc [✔ Dùng]]
[Trong Modal chuyển thành: [✔ Đã OAuth]]
```

---

# PHẦN II: KIẾN TRÚC SẴN CÓ TÁI SỬ DỤNG 100% (REUSABLE ASSETS)

Coder **bắt buộc phải tái sử dụng** các thành phần kiến trúc đã hoàn thiện trong codebase, tuyệt đối không viết lại:

1. **Widget Thẻ Tài Khoản (`internal/ui/widgets.go`):**
   - Tận dụng 100% logic hiển thị thẻ `[Cần OAuth]` sẵn có tại dòng 578-586:
     ```go
     text, tint := "[Cần OAuth]", palette.Warning
     badge := canvas.NewText(text, tint)
     // ... actionSlot = container.NewCenter(badge)
     ```
   - Tận dụng logic hiển thị Quota ma trận `N/A` khi thẻ chưa có token (`!w.IsActive && !w.HasToken`) tại dòng 517-524.
   - Tận dụng cơ chế cập nhật phản ứng tại chỗ `UpdateState` và `UpdateQuota` để tránh giật lag layout.
2. **Scanner Chrome (`internal/scanner/chrome_scanner.go`):**
   - Tái sử dụng hàm quét chuẩn `scanner.ScanChromeAccountDetails(cDir)` và hàm phân giải thư mục `scanner.GetDefaultChromeUserDataDir()`.
3. **Cơ Chế Ghi Tệp An Toàn (Atomic Temp File + Rename Pattern):**
   - Tái sử dụng mẫu ghi tệp nguyên tử bằng `os.CreateTemp` + `Sync()` + `os.Rename` đã chuẩn hóa tại `internal/store/account_store.go` và `internal/store/discovered_store.go`.
4. **Hệ Thống Bảng Màu & Giao Diện (`internal/ui/themes.go`):**
   - Tái sử dụng `ThemePalette`, `GetThemePalette(a.currentTheme)` để Modal tự động đồng bộ theo 3 theme `Dark`, `Light`, `Warm`.

---

# PHẦN III: KIẾN TRÚC CŨ ĐƯỢC BẢO TOÀN TUYỆT ĐỐI 100% (PRESERVED INVARIANTS)

Coder **tuyệt đối không được làm thay đổi** các bất biến kiến trúc sau:

1. **Invariant-Auto-Switch-Immunity (Bảo vệ Daemon chuyển đổi tự động):**
   - Bộ điều khiển `AutomationController` và thuật toán `Burn-down Sorter` chỉ lọc ứng viên có token sống trong `accounts_store/`. Thẻ `[Cần OAuth]` có `HasToken: false` nên **tự động bị loại khỏi danh sách chuyển đổi**, đảm bảo an toàn tuyệt đối cho daemon tự động.
2. **Invariant-Telemetry-Loop-Safety (Bảo vệ vòng lặp đo lường):**
   - Vòng lặp 60s `AutoTelemetryLoop` chỉ truy vấn quota cho các tài khoản có token trong `accounts_store/`. Thẻ `[Cần OAuth]` không bao giờ kích hoạt request Google API ngầm.
3. **Invariant-Hot-Swap-Integrity (Bảo toàn quy trình chuyển đổi 9 bước):**
   - Toàn bộ luồng 9 bước của `SwitchAccount` (Connect-RPC probe, snapshot active threads, WinCred swap, recycle Language Server) giữ nguyên 100%.
4. **Invariant-Grid-Layout-Stability (Bảo toàn lưới thẻ giao diện):**
   - Lưới thẻ `accountListContainer` (280x130 symmetric scroller) giữ nguyên cấu trúc. Khi nạp thêm thẻ `[Cần OAuth]`, lưới tự động mở rộng theo dòng mà không phá vỡ khung cửa sổ.
5. **Invariant-Card-Signature-Compatibility (Bảo toàn tương thích ngược chữ ký hàm):**
   - Giữ nguyên 100% chữ ký hàm `NewAccountCardWidgetWithQuota` và `NewAccountCardWithQuota`, không gây ảnh hưởng đến 40+ unit test hiện có.

---

# PHẦN IV: KIẾN TRÚC BỊ TÁC ĐỘNG & CHI TIẾT TỪNG FILE, STRUCT, HÀM

### 1. `internal/store/discovered_store.go`

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
  - Nếu đã tồn tại và parse được `Added == true`, bảo toàn `entry.Added = true` (không reset về `false` khi Chrome quét lại).
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

### 2. `internal/manager/account_manager.go`

#### A. Tái cấu trúc `GetAccountStates(ctx context.Context) (*AccountOverview, error)`:
- **CẮT BỎ HOÀN TOÀN** đoạn code sau khỏi `GetAccountStates`:
  ```go
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

### 3. Tạo Mới `internal/ui/add_modal.go`

Module này chịu trách nhiệm hiển thị Modal "Thêm tài khoản" độc lập.

#### A. Cấu trúc và Vòng đời Modal:
```go
type AddAccountModal struct {
    app        *App
    win        fyne.Window
    dialog     dialog.Dialog
    listVBox   *fyne.Container
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

### 4. Cập nhật Toolbar `internal/ui/window.go`

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

### 5. Cập nhật `internal/ui/app.go`
- Tối ưu `Init()`: Khởi động app nạp UI ngay từ đĩa (dưới 50ms), không còn bị nghẽn bởi Chrome I/O.
- Bổ sung phương thức điều phối: `func (a *App) ShowAddAccountModal()`.

---

# PHẦN V: CÁCH CODER TRIỂN KHAI TỪNG BƯỚC (STEP-BY-STEP IMPLEMENTATION ROADMAP)

Coder triển khai tuần tự theo 5 bước sau:

```text
┌────────────────────────────────────────────────────────┐
│ BƯỚC 1: Mở rộng Store (discovered_store.go)           │
│ • Thêm trường Added bool vào DiscoveredAccountEnvelope │
│ • Cập nhật SaveDiscovered để bảo toàn cờ Added         │
│ • Thêm SetDiscoveredAdded & ListDiscoveredRecords      │
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ BƯỚC 2: Tách rời Manager (account_manager.go)          │
│ • Xóa scan Chrome khỏi GetAccountStates                │
│ • Lọc rec.Added == true khi nạp vào itemsMap           │
│ • Thêm ScanChromeAccounts & GetCandidateAccounts       │
│ • Thêm AddDiscoveredAccount                            │
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ BƯỚC 3: Tạo giao diện Modal (internal/ui/add_modal.go) │
│ • Dựng AddAccountModal với kích thước 440x320          │
│ • Bố trí Header (nút quét Chrome), List cuộn, Footer   │
│ • Xử lý sự kiện click [ + ADD ]: đổi nhãn sang [Đã add]│
│   và ẩn nút ADD, cập nhật UI tổng ngay lập tức         │
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ BƯỚC 4: Tích hợp UI Tổng (window.go & app.go)          │
│ • Gắn nút [+ Thêm] vào Toolbar Header Row 2            │
│ • Kết nối callback ShowAddAccountModal                 │
└───────────────────────────┬────────────────────────────┘
                            │
┌───────────────────────────▼────────────────────────────┐
│ BƯỚC 5: Chạy Kiểm Thử Đơn Vị & Tích Hợp (go test)      │
│ • Chạy kiểm thử store, manager, ui                     │
│ • Kiểm chứng 8 bài kiểm thử chấp nhận (TC-01 -> TC-08) │
└────────────────────────────────────────────────────────┘
```

---

# PHẦN VI: BỘ TIÊU CHUẨN KIỂM THỬ CHẤP NHẬN (ACCEPTANCE TEST MATRIX)

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
