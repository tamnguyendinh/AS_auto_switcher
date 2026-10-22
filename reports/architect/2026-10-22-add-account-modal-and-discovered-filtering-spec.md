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

Cụ thể, hàm `GetAccountStates()` trong `internal/manager/account_manager.go:308` đang bị kích hoạt tự động ngầm ở **4 thời điểm**:
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

Coder **tuyệt đối không được làm thay đổi** các bất biến kiến trúc sau để đảm bảo ứng dụng vận hành bình thường:

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

# PHẦN IV: KIẾN TRÚC BỊ TÁC ĐỘNG & ĐẶC TẢ MÃ NGUỒN CHI TIẾT (CODE-LEVEL BLUEPRINT)

Coder triển khai mã nguồn cụ thể theo từng file dưới đây:

### 1. `internal/store/discovered_store.go`

#### A. Cập nhật Struct `DiscoveredAccountEnvelope`:
```go
type DiscoveredAccountEnvelope struct {
	Email     string  `json:"email"`
	ScannedAt float64 `json:"scanned_at"`
	Added     bool    `json:"added,omitempty"`
}
```

#### B. Cập nhật `SaveDiscovered(email string) error`:
```go
func (s *DiscoveredStore) SaveDiscovered(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	path, err := SafeAccountPath(s.dir, email)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	// BẢO TỒN CỜ Added nếu đã tồn tại trước đó:
	added := false
	if existingData, readErr := os.ReadFile(path); readErr == nil {
		var existingEnv DiscoveredAccountEnvelope
		if jsonErr := json.Unmarshal(existingData, &existingEnv); jsonErr == nil {
			added = existingEnv.Added
		}
	}

	tmp, err := os.CreateTemp(s.dir, ".discovered-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	entry := DiscoveredAccountEnvelope{
		Email:     email,
		ScannedAt: float64(time.Now().UnixNano()) / 1e9,
		Added:     added,
	}
	if err := json.NewEncoder(tmp).Encode(entry); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
```

#### C. Bổ sung `SetDiscoveredAdded(email string, added bool) error`:
```go
// SetDiscoveredAdded updates the Added flag for an existing or new discovered record.
func (s *DiscoveredStore) SetDiscoveredAdded(email string, added bool) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ErrInvalidEmail
	}
	path, err := SafeAccountPath(s.dir, email)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	scannedAt := float64(time.Now().UnixNano()) / 1e9
	if existingData, readErr := os.ReadFile(path); readErr == nil {
		var existingEnv DiscoveredAccountEnvelope
		if jsonErr := json.Unmarshal(existingData, &existingEnv); jsonErr == nil {
			if existingEnv.ScannedAt > 0 {
				scannedAt = existingEnv.ScannedAt
			}
		}
	}

	tmp, err := os.CreateTemp(s.dir, ".discovered-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	entry := DiscoveredAccountEnvelope{
		Email:     email,
		ScannedAt: scannedAt,
		Added:     added,
	}
	if err := json.NewEncoder(tmp).Encode(entry); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
```

#### D. Bổ sung `ListDiscoveredRecords() ([]DiscoveredAccountEnvelope, error)`:
```go
// ListDiscoveredRecords returns full envelopes including Added flag for all valid records.
func (s *DiscoveredStore) ListDiscoveredRecords() ([]DiscoveredAccountEnvelope, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var records []DiscoveredAccountEnvelope
	seen := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}
		var record DiscoveredAccountEnvelope
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(record.Email))
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Email < records[j].Email
	})
	return records, nil
}
```

---

### 2. `internal/manager/account_manager.go`

#### A. Cắt bỏ Quét Chrome và Lọc `rec.Added == true` trong `GetAccountStates`:
```go
func (m *AccountManager) GetAccountStates(ctx context.Context) (*AccountOverview, error) {
	var warnings []string
	if _, err := m.CaptureCurrentSession(); err != nil && !errors.Is(err, wincred.ErrNotFound) {
		warnings = append(warnings, fmt.Sprintf("Không lưu được phiên hiện tại: %v", err))
	}
	activeEmail, _ := m.GetCurrentActiveEmail(ctx)

	m.mu.RLock()
	finder, discovered := m.clientFinder, m.discoveredStore
	m.mu.RUnlock()

	storedEmails, storeErr := m.store.ListAccounts()
	if storeErr != nil {
		warnings = append(warnings, fmt.Sprintf("Không đọc được kho tài khoản: %v", storeErr))
	}
	storedSet := make(map[string]bool)
	for _, email := range storedEmails {
		storedSet[strings.ToLower(strings.TrimSpace(email))] = true
	}

	itemsMap := make(map[string]*AccountDisplayItem)
	add := func(email string) *AccountDisplayItem {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			return nil
		}
		if item := itemsMap[email]; item != nil {
			return item
		}
		item := &AccountDisplayItem{Email: email}
		itemsMap[email] = item
		return item
	}

	// ĐỌC DANH BẠ DISCOVERED VÀ CHỈ NẠP TÀI KHOẢN CÓ Added == true:
	if discovered != nil {
		records, err := discovered.ListDiscoveredRecords()
		for _, rec := range records {
			if rec.Added {
				add(rec.Email)
			}
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Không đọc đầy đủ danh bạ Chrome: %v", err))
		}
	}

	// NẠP TÀI KHOẢN ĐÃ CÓ TOKEN VÀ ACTIVE:
	for _, email := range storedEmails {
		add(email)
	}
	add(activeEmail)

	accounts := make([]AccountDisplayItem, 0, len(itemsMap))
	for _, item := range itemsMap {
		item.HasToken = storedSet[item.Email]
		item.IsActive = activeEmail != "" && item.Email == activeEmail
		item.Status = "on"
		if item.HasToken {
			item.Status = m.GetAccountStatus(item.Email)
		}
		switch {
		case item.IsActive:
			item.State, item.DisplayBadge = model.StateActive, "[✔ Dùng]"
		case item.HasToken:
			item.State, item.DisplayBadge = model.StateReady, "[Chuyển]"
		default:
			item.State, item.DisplayBadge = model.StateUnauth, "[Cần OAuth]"
		}
		accounts = append(accounts, *item)
	}
	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].IsActive != accounts[j].IsActive {
			return accounts[i].IsActive
		}
		if accounts[i].HasToken != accounts[j].HasToken {
			return accounts[i].HasToken
		}
		return accounts[i].Email < accounts[j].Email
	})

	overview := &AccountOverview{ActiveEmail: activeEmail, Accounts: accounts, Warnings: warnings}
	if finder != nil {
		if client, err := finder(ctx); err == nil && client != nil {
			overview.LanguageServerRunning = true
			overview.ServerInfo = client.ServerInfo()
		}
	}
	return overview, nil
}
```

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

	chromeAccounts, err := scanner.ScanChromeAccountDetails(cDir)
	if err != nil {
		return nil, fmt.Errorf("không quét được Chrome: %w", err)
	}

	var scannedEmails []string
	for _, ca := range chromeAccounts {
		em := strings.ToLower(strings.TrimSpace(ca.Email))
		if em == "" {
			continue
		}
		_ = discovered.SaveDiscovered(em) // Đã có logic bảo toàn cờ Added
		scannedEmails = append(scannedEmails, em)
	}
	return scannedEmails, nil
}
```

#### C. Bổ sung `CandidateAccountItem` & `GetCandidateAccounts()`:
```go
type CandidateAccountItem struct {
	Email    string
	HasToken bool
	Added    bool
}

// GetCandidateAccounts returns all discovered accounts enriched with token and added state.
func (m *AccountManager) GetCandidateAccounts() ([]CandidateAccountItem, error) {
	m.mu.RLock()
	st, discovered := m.store, m.discoveredStore
	m.mu.RUnlock()

	if discovered == nil {
		return nil, errors.New("discovered store uninitialized")
	}

	records, err := discovered.ListDiscoveredRecords()
	if err != nil {
		return nil, err
	}

	storedSet := make(map[string]bool)
	if st != nil {
		if storedEmails, err := st.ListAccounts(); err == nil {
			for _, em := range storedEmails {
				storedSet[strings.ToLower(strings.TrimSpace(em))] = true
			}
		}
	}

	items := make([]CandidateAccountItem, 0, len(records))
	for _, r := range records {
		items = append(items, CandidateAccountItem{
			Email:    r.Email,
			HasToken: storedSet[r.Email],
			Added:    r.Added,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].HasToken != items[j].HasToken {
			return items[i].HasToken
		}
		if items[i].Added != items[j].Added {
			return items[i].Added
		}
		return items[i].Email < items[j].Email
	})
	return items, nil
}
```

#### D. Bổ sung `AddDiscoveredAccount(email string) error`:
```go
// AddDiscoveredAccount marks an account as added to the main UI.
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

```go
package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ShowAddAccountModal displays the Add Account dialog.
func (a *App) ShowAddAccountModal() {
	if a == nil || a.win == nil || a.mgr == nil {
		return
	}

	palette := GetThemePalette(a.currentTheme)

	listContainer := container.NewVBox()
	scrollContent := container.NewVScroll(container.NewPadded(listContainer))
	scrollContent.SetMinSize(fyne.NewSize(420, 260))

	var d dialog.Dialog

	reloadCandidates := func() {
		candidates, err := a.mgr.GetCandidateAccounts()
		if err != nil {
			return
		}
		listContainer.Objects = nil
		if len(candidates) == 0 {
			emptyLbl := widget.NewLabelWithStyle("Không có tài khoản nào trong danh bạ.", fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
			listContainer.Add(emptyLbl)
			listContainer.Refresh()
			return
		}

		for _, cand := range candidates {
			candItem := cand // capture
			emailLbl := widget.NewLabel(candItem.Email)
			if a.IsMaskAccounts() {
				emailLbl.SetText(MaskEmail(candItem.Email))
			}
			emailLbl.Truncation = fyne.TextTruncateEllipsis

			var actionSlot *fyne.Container

			if candItem.HasToken {
				// Trường hợp 1: [✔ Đã OAuth]
				statusBadge := canvas.NewText("[✔ Đã OAuth]", palette.Success)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}
				actionSlot = container.NewCenter(statusBadge)
			} else if candItem.Added {
				// Trường hợp 3: [ Đã add ]
				statusBadge := canvas.NewText("[Đã add]", palette.TextMuted)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}
				actionSlot = container.NewCenter(statusBadge)
			} else {
				// Trường hợp 2: [Cần OAuth] + Nút [ + ADD ]
				statusBadge := canvas.NewText("[Cần OAuth]", palette.Warning)
				statusBadge.TextSize = 11
				statusBadge.TextStyle = fyne.TextStyle{Bold: true}

				var addBtn *widget.Button
				addBtn = widget.NewButtonWithIcon("ADD", theme.ContentAddIcon(), func() {
					if err := a.mgr.AddDiscoveredAccount(candItem.Email); err == nil {
						// Chuyển đổi tại chỗ: [Cần OAuth] đổi thành [Đã add] và nút ADD biến mất
						doneBadge := canvas.NewText("[Đã add]", palette.TextMuted)
						doneBadge.TextSize = 11
						doneBadge.TextStyle = fyne.TextStyle{Bold: true}
						actionSlot.Objects = []fyne.CanvasObject{doneBadge}
						actionSlot.Refresh()

						// Nạp ngay thẻ mới lên UI tổng
						go func() {
							overview, _ := a.mgr.GetAccountStates(context.Background())
							a.mu.Lock()
							a.overview = overview
							a.mu.Unlock()
							fyne.Do(func() {
								a.updateUI(overview)
							})
						}()
					}
				})
				addBtn.Importance = widget.HighImportance
				actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactButton(addBtn))
			}

			row := container.NewBorder(nil, nil, nil, actionSlot, emailLbl)
			listContainer.Add(row)
		}
		listContainer.Refresh()
	}

	var scanBtn *widget.Button
	scanBtn = widget.NewButtonWithIcon("Quét Chrome", theme.ViewRefreshIcon(), func() {
		scanBtn.Disable()
		scanBtn.SetText("⏳ Đang quét...")
		go func() {
			_, _ = a.mgr.ScanChromeAccounts()
			fyne.Do(func() {
				reloadCandidates()
				scanBtn.SetText("Quét Chrome")
				scanBtn.Enable()
			})
		}()
	})

	headerRow := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Thêm tài khoản", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		compactButton(scanBtn),
	)

	contentBox := container.NewBorder(
		container.NewVBox(headerRow, widget.NewSeparator()),
		nil, nil, nil,
		scrollContent,
	)

	reloadCandidates()

	d = dialog.NewCustom("Thêm tài khoản", "Đóng", contentBox, a.win)
	d.Resize(fyne.NewSize(460, 340))
	d.Show()
}
```

---

### 4. Cập nhật Toolbar `internal/ui/window.go`

Thay thế nút `a.refreshBtn` ("Danh sách tài khoản" cũ) bằng nút `a.addAccountBtn`:
```go
	a.addAccountBtn = widget.NewButtonWithIcon("Thêm", theme.ContentAddIcon(), func() {
		a.ShowAddAccountModal()
	})

	toolbarLeft := container.NewHBox(
		compactButton(a.addAccountBtn),
		compactButton(a.quotaRefreshAllBtn),
		compactButton(a.alwaysOnTopBtn),
	)
```

---

### 5. Cập nhật `internal/ui/app.go`
- Tối ưu `Init()`: Khởi động app nạp UI ngay từ đĩa (dưới 50ms), không còn bị nghẽn bởi Chrome I/O.
- Tích hợp hàm `func (a *App) ShowAddAccountModal()` đã tạo ở `add_modal.go`.

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
