# ĐẶC TẢ KIẾN TRÚC & KỸ THUẬT TOÀN DIỆN: NÚT XÓA TÀI KHOẢN (TRASH BUTTON) & ĐỒNG BỘ HAI KHO DỮ LIỆU

- **Tài liệu:** Đặc tả Kỹ thuật Kiến trúc & Hướng dẫn Triển khai (Comprehensive Software Architecture & Implementation Blueprint)
- **Mã định danh:** `SPEC-ARCH-20261023-ACCOUNT-DELETION-AND-TRASH-BUTTON`
- **Tác giả:** System Architect (Nhánh Chiến Lược)
- **Ngày lập:** 2026-10-23
- **Trạng thái:** APPROVED BY HUMAN OWNER / READY FOR IMPLEMENTATION
- **Đối tượng đọc:** Coder / Implementer (Đọc vào triển khai trực tiếp 100%, không cần suy đoán logic)

---

# PHẦN I: BỐI CẢNH, NGUYÊN TẮC THIẾT KẾ & BẢN ĐỒ KIẾN TRÚC

## 1. Bối Cảnh & Nhu Cầu Người Dùng

Hiện tại, ứng dụng Antigravity Auto Switcher cho phép nạp tài khoản qua hai hình thức:
1. Đăng nhập trực tiếp qua Antigravity IDE (tự động capture session vào `accounts_store/<email>.json`).
2. Quét tài khoản từ Google Chrome Profiles và bấm `[+ ADD]` trong Modal Thêm tài khoản để đưa thẻ `[Cần OAuth]` lên giao diện chính.

**Vấn đề tồn tại:** 
Người dùng không có bất kỳ cơ chế trực quan nào trên giao diện chính để:
- Thu hồi, dọn dẹp hoặc xóa vĩnh viễn các tài khoản không còn sử dụng khỏi kho lưu trữ `accounts_store/`.
- Gỡ bỏ các thẻ đang ở trạng thái `[Cần OAuth]` ra khỏi màn hình chính nếu người dùng không muốn tiếp tục liên kết.

Khi một tài khoản đã được nạp vào, nó sẽ tồn tại vĩnh viễn trên UI trừ khi người dùng tự mở thư mục tệp tin hệ thống và xóa file `.json` thủ công bằng tay.

---

## 2. Nguyên Tắc Thiết Kế Cốt Lõi (Architecture Invariants)

Bản đặc tả này thiết lập **6 Nguyên Tắc Bất Di Bất Dịch (Supreme Invariants)** mà Coder bắt buộc phải tuân thủ nghiêm ngặt:

1. **Invariant 1 — Bất Khả Xâm Phạm Tài Khoản Active (`Active-Account Protection Invariant`):**
   Tuyệt đối **KHÔNG BAO GIỜ** hiển thị nút Xóa `[ 🗑 ]` trên thẻ tài khoản đang Active (`IsActive == true`, mang badge `[✔ Dùng]`). Cấm triệt để việc xóa tài khoản đang hoạt động vì phiên Windows Credential Manager (`wincred`) sẽ tự động capture lại token ở chu kỳ kế tiếp, gây xung đột trạng thái dữ liệu (race condition). Muốn xóa tài khoản này, người dùng bắt buộc phải chuyển sang một tài khoản khác trước.

2. **Invariant 2 — Nhất Quán Thứ Tự Thanh Hành Động (`Action-Cluster Order Invariant`):**
   Trên mọi thẻ tài khoản, cụm nút điều khiển bên phải (`actionCluster`) phải luôn tuân thủ đúng thứ tự 4 vị trí từ trái sang phải:
   $$\text{[ 1. Action Slot ]} \quad \longrightarrow \quad \text{[ 2. Refresh Quota ]} \quad \longrightarrow \quad \text{[ 3. Status ON/OFF ]} \quad \longrightarrow \quad \text{[ 4. Trash Button 🗑 ]}$$

3. **Invariant 3 — Bắt Buộc Xác Nhận An Toàn (`Confirmation Dialog Mandatory Invariant`):**
   Hành động nhấn icon `[ 🗑 ]` **TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP** xóa dữ liệu ngay lập tức. Bắt buộc phải bật Pop-up xác nhận modal native của Fyne (`dialog.NewCustomConfirm`) với 2 lựa chọn: `[Hủy]` và `[Xóa]` (màu đỏ Danger). Chỉ khi người dùng bấm xác nhận `[Xóa]`, tiến trình xóa đĩa mới được phép thực thi.

4. **Invariant 4 — Đồng Bộ Triệt Để Hai Kho (Two-Store Atomic Deletion Invariant):**
   Áp dụng **Kịch bản B (Gỡ hoàn toàn thẻ khỏi màn hình chính)**:
   - Với tài khoản có Token trong `accounts_store/`: Xóa file `accounts_store/<email>.json` VÀ đồng thời cập nhật `added: false` trong `discovered_accounts/<email>.json`.
   - Với tài khoản `[Cần OAuth]`: Cập nhật `added: false` trong `discovered_accounts/<email>.json`.
   - Kết quả: Thẻ tài khoản biến mất hoàn toàn khỏi màn hình chính và quay về danh sách ứng viên trong Modal `[+ Thêm]`.

5. **Invariant 5 — Hỗ Trợ Đầy Đủ Thẻ OFF (`Status OFF Deletion Invariant`):**
   Khi tài khoản ở trạng thái `Status == "off"` (nút `[Chuyển]` bị mờ disabled), nút Xóa `[ 🗑 ]` **vẫn sáng rõ và hoạt động bình thường**, cho phép người dùng dọn dẹp các tài khoản đã vô hiệu hóa.

6. **Invariant 6 — Dọn Dẹp Bộ Nhớ & An Toàn Tự Động Hóa (`Memory Cleanup & Engine Safety Invariant`):**
   Sau khi xóa thành công:
   - Xóa ngay cache quota của email đó trong `a.quotaCache`.
   - Nạp lại `overview` và refresh UI trên luồng chính (`fyne.Do`).
   - Nếu hệ thống đang bật Tự động chuyển (Auto-switch) và không còn tài khoản nào khả dụng, tự động tắt auto-switch an toàn và cập nhật thanh trạng thái: `⚠️ Không có tài khoản để chuyển.`.

---

# PHẦN II: THIẾT KẾ GIAO DIỆN & TRẢI NGHIỆM NGƯỜI DÙNG (UI/UX SPECIFICATION)

## 1. Bố Cục Trực Quan Thanh Hành Động (`actionCluster`)

Thanh hành động nằm ở hàng trên bên phải của mỗi thẻ tài khoản (`AccountCardWidget`):

```text
┌────────────────────────────────────────────────────────────────────────┐
│  user@example.com           [ [Chuyển] ]  [ 🔄 ]  [ [ON] ]  [ 🗑 ]     │
├────────────────────────────────────────────────────────────────────────┤
│  Gemini Pro    [████████░░░░] 65.0%   (5h: 01:23:45)                   │
│  Claude 3.5    [████████████] 100.0%  (5h: 04:12:00)                   │
└────────────────────────────────────────────────────────────────────────┘
```

### Chi tiết trạng thái hiển thị của 4 vị trí nút:

| Loại Thẻ | Vị trí 1: Action Slot | Vị trí 2: Refresh Quota | Vị trí 3: Status Button | Vị trí 4: Delete Button `[ 🗑 ]` |
|---|---|---|---|---|
| **Inactive (Status ON)** | Nút phẳng `[Chuyển]` (Primary, click chuyển) | Nút icon `[ 🔄 ]` (Enabled) | Nút phẳng `[ON]` (Xanh lá) | **Nút icon `[ 🗑 ]` (Sáng rõ, click mở dialog)** |
| **Inactive (Status OFF)** | Nút phẳng `[Chuyển]` (Mờ TextMuted, Disabled) | Nút icon `[ 🔄 ]` (Disabled) | Nút phẳng `[OFF]` (Xám TextMuted) | **Nút icon `[ 🗑 ]` (Sáng rõ, click mở dialog)** |
| **Cần OAuth (`HasToken=false`)** | Nhãn vàng `[Cần OAuth]` | Nút icon `[ 🔄 ]` (Disabled) | **Ẩn hoàn toàn (`Hide()`)** | **Nút icon `[ 🗑 ]` (Sáng rõ, click mở dialog)** |
| **Active (`IsActive=true`)** | Nhãn xanh `[✔ Dùng]` | Nút icon `[ 🔄 ]` (Enabled) | Nút phẳng `[ON]` / `[OFF]` | **Ẩn hoàn toàn (`Hide()`)** |

---

## 2. Thông Số Kỹ Thuật Nút Thùng Rác (`deleteButton`)

- **Loại Widget:** `*widget.Button` bọc trong `compactButton(...)` (chuẩn hóa độ cao tối đa 24px đồng bộ với nút `[ 🔄 ]`).
- **Icon:** `theme.DeleteIcon()` (Icon thùng rác chuẩn Fyne).
- **Text:** Chuỗi rỗng `""` (chỉ hiển thị icon).
- **Importance:** `widget.LowImportance`.
- **Màu sắc / Hover:** Thừa hưởng style của `compactButton`.
- **Hành vi khi Click:** Kích hoạt callback `onDelete(email)`.

---

## 3. Thiết Kế Hộp Thoại Xác Nhận (`dialog.NewCustomConfirm`)

Hộp thoại xác nhận được dựng qua hàm `dialog.NewCustomConfirm` gắn trực tiếp vào cửa sổ chính `a.win`:

```text
┌──────────────────────────────────────────────────────────────┐
│  Xác nhận xóa tài khoản                                      │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Bạn có chắc chắn muốn xóa tài khoản user@example.com khỏi    │
│  hệ thống? Thao tác này sẽ xóa token đã lưu và bạn sẽ cần    │
│  đăng nhập lại.                                              │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│                                      [ Hủy ]    [ Xóa ]      │
└──────────────────────────────────────────────────────────────┘
```

### Chi tiết cấu hình Dialog:
- **Title:** `"Xác nhận xóa tài khoản"`
- **Confirm Text:** `"Xóa"`
- **Dismiss Text:** `"Hủy"`
- **Confirm Button Importance:** `widget.DangerImportance` (Màu đỏ cảnh báo bắt mắt).
- **Nội dung thông báo (Phân cấp theo loại thẻ):**
  - **Trường hợp A — Thẻ đã có Token (`HasToken == true`):**
    $$\text{"Bạn có chắc chắn muốn xóa tài khoản <email> khỏi hệ thống? Thao tác này sẽ xóa token đã lưu và bạn sẽ cần đăng nhập lại."}$$
  - **Trường hợp B — Thẻ `[Cần OAuth]` (`HasToken == false`):**
    $$\text{"Bạn có chắc chắn muốn gỡ tài khoản <email> khỏi màn hình chính không?"}$$
- **Xử lý Privacy Masking:**
  Nếu `a.maskAccounts == true`, giá trị `<email>` trong thông điệp bắt buộc phải được bọc qua hàm `MaskEmail(email)` (ví dụ: `u***@gmail.com`).

---

# PHẦN III: THIẾT KẾ KỸ THUẬT & CONTRACTS (DATA & LOGIC ARCHITECTURE)

## 1. Tầng Store (`internal/store/`)

Hiện tại cả hai phương thức cần thiết đã có sẵn và chạy ổn định tại:
- [`internal/store/account_store.go:303`](file:///g:/antigravity_auto_switch/internal/store/account_store.go#L303):
  `func (s *AccountStore) DeleteAccount(email string) error`
- [`internal/store/discovered_store.go:88`](file:///g:/antigravity_auto_switch/internal/store/discovered_store.go#L88):
  `func (s *DiscoveredStore) SetDiscoveredAdded(email string, added bool) error`

Coder **không cần sửa đổi tầng Store**, chỉ cần gọi đúng các API trên.

---

## 2. Tầng Quản Lý Điều Phối (`internal/manager/account_manager.go`)

Bổ sung phương thức điều phối nghiệp vụ xóa tài khoản toàn diện:

```go
// ErrCannotDeleteActiveAccount is returned when attempting to delete the currently active account.
var ErrCannotDeleteActiveAccount = errors.New("không thể xóa tài khoản đang hoạt động")

// RemoveAccount orchestrates the two-store deletion workflow:
// 1. Rejects deletion if the target email is currently active.
// 2. Deletes the token credentials file in accounts_store/<email>.json.
// 3. Resets the Added flag to false in discovered_accounts/<email>.json (if record exists).
func (m *AccountManager) RemoveAccount(ctx context.Context, email string) error {
	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return store.ErrInvalidEmail
	}

	// Invariant 1: Cấm xóa tài khoản đang Active
	activeEmail, _ := m.GetCurrentActiveEmail(ctx)
	if activeEmail != "" && strings.EqualFold(activeEmail, normEmail) {
		return ErrCannotDeleteActiveAccount
	}

	m.mu.Lock()
	accStore, discStore := m.store, m.discoveredStore
	m.mu.Unlock()

	// 1. Xóa khỏi accounts_store
	if accStore != nil {
		if err := accStore.DeleteAccount(normEmail); err != nil {
			return fmt.Errorf("lỗi xóa tệp tài khoản: %w", err)
		}
	}

	// 2. Cập nhật added: false trong discovered_accounts nếu record đã tồn tại
	if discStore != nil {
		records, err := discStore.ListDiscoveredRecords()
		if err == nil {
			for _, rec := range records {
				if strings.EqualFold(rec.Email, normEmail) {
					_ = discStore.SetDiscoveredAdded(normEmail, false)
					break
				}
			}
		}
	}

	return nil
}
```

---

## 3. Tầng Bộ Nhớ Đệm Quota (`internal/telemetry/cache.go`)

Để ngăn chặn rò rỉ bộ nhớ hoặc hiển thị quota cũ sau khi xóa tài khoản, bổ sung phương thức `Delete` trên struct `QuotaCache`:

```go
// Delete removes cached quota data for an email from memory.
func (c *QuotaCache) Delete(email string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.matrix, strings.ToLower(strings.TrimSpace(email)))
}
```

---

## 4. Tầng Widget Thẻ Giao Diện (`internal/ui/widgets.go`)

### 4.1. Mở rộng Struct `AccountCardWidget`

```go
type AccountCardWidget struct {
	Root           fyne.CanvasObject
	Email          string
	IsActive       bool
	HasToken       bool
	Status         string
	masked         bool
	onSwitch       func(email string)
	onRefreshSingle func(email string)
	onToggleStatus func(email string)
	onDelete       func(email string) // <-- MỚI: Callback xóa tài khoản

	cardBg         *canvas.Rectangle
	emailLabel     *tappableEmailLabel
	actionCluster  *fyne.Container
	actionSlot     *fyne.Container
	refreshButton  *widget.Button
	statusButton   *statusTextButton
	deleteButton   *widget.Button     // <-- MỚI: Widget nút thùng rác

	// Matrix Quota Cells...
}
```

### 4.2. Getter & Trigger phục vụ Kiểm thử Đơn vị (Unit Tests)

```go
// DeleteButton returns the internal delete button for testing and inspection.
func (w *AccountCardWidget) DeleteButton() *widget.Button {
	return w.deleteButton
}

// TriggerDelete invokes the onDelete callback directly (used in tests).
func (w *AccountCardWidget) TriggerDelete() {
	if w != nil && !w.IsActive && w.onDelete != nil {
		w.onDelete(w.Email)
	}
}

// SetOnDelete updates or injects the onDelete callback.
func (w *AccountCardWidget) SetOnDelete(fn func(email string)) {
	if w != nil {
		w.onDelete = fn
	}
}
```

### 4.3. Cập nhật Hàm Khởi Tạo `NewAccountCardWidgetWithQuota`

Hàm khởi tạo duy trì tính **tương thích ngược 100% (Backward Compatibility)** với tất cả unit test hiện hữu:

```go
func NewAccountCardWidgetWithQuota(
	item manager.AccountDisplayItem,
	quota *telemetry.AccountQuotaData,
	palette ThemePalette,
	onSwitch func(email string),
	onRefreshSingle func(email string),
	extraCallbacks ...func(email string),
) *AccountCardWidget {
	var toggleFn func(email string)
	var deleteFn func(email string)

	if len(extraCallbacks) > 0 {
		toggleFn = extraCallbacks[0]
	}
	if len(extraCallbacks) > 1 {
		deleteFn = extraCallbacks[1]
	}

	st := strings.ToLower(strings.TrimSpace(item.Status))
	if st != "off" {
		st = "on"
	}

	w := &AccountCardWidget{
		Email:          item.Email,
		Status:         st,
		onSwitch:       onSwitch,
		onRefreshSingle: onRefreshSingle,
		onToggleStatus: toggleFn,
		onDelete:       deleteFn,
	}

	emailLabel := newEmailLabelWithTooltip(item.Email, item.IsActive)
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if (w.IsActive || w.HasToken) && w.Status != "off" && onRefreshSingle != nil {
			onRefreshSingle(w.Email)
		}
	})
	refreshBtn.Importance = widget.LowImportance

	statusBtn := newStatusTextButton(st, func() {
		if w.onToggleStatus != nil {
			w.onToggleStatus(w.Email)
		}
	}, palette)

	// Tạo nút Delete với theme.DeleteIcon()
	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if !w.IsActive && w.onDelete != nil {
			w.onDelete(w.Email)
		}
	})
	deleteBtn.Importance = widget.LowImportance

	actionSlot := container.NewStack()
	// Bố trí đúng Invariant 2: [Action] [Refresh] [Status] [Delete]
	actionCluster := container.NewHBox(
		actionSlot,
		compactButton(refreshBtn),
		statusBtn,
		compactButton(deleteBtn),
	)
	topRow := container.NewBorder(nil, nil, nil, actionCluster, emailLabel)

	// ... phần tạo matrixTable giữ nguyên ...

	w.Root, w.cardBg, w.emailLabel = root, cardBg, emailLabel
	w.actionCluster, w.actionSlot, w.refreshButton, w.statusButton, w.deleteButton = actionCluster, actionSlot, refreshBtn, statusBtn, deleteBtn

	// ... phần quota cell mapping giữ nguyên ...

	w.UpdateState(item, palette)
	w.UpdateQuota(quota, palette)
	return w
}
```

### 4.4. Cập nhật Logic `UpdateState`

Trong `UpdateState(item manager.AccountDisplayItem, palette ThemePalette)`:

```go
// 1. Quản lý trạng thái nút Delete
if w.deleteButton != nil {
	if w.IsActive {
		// Invariant 1: Thẻ Active TUYỆT ĐỐI KHÔNG HIỂN THỊ nút Xóa
		w.deleteButton.Hide()
	} else {
		// Thẻ Inactive (Status ON/OFF hoặc Cần OAuth): Luôn hiển thị
		w.deleteButton.Show()
		w.deleteButton.Enable()
	}
}
```

---

## 5. Tầng Ứng Dụng Chính (`internal/ui/app.go`)

### 5.1. Bổ sung Hàm Điều Khiển `ConfirmAndDeleteAccount`

```go
// ConfirmAndDeleteAccount displays a confirmation dialog before executing two-store deletion.
func (a *App) ConfirmAndDeleteAccount(email string) {
	a.mu.Lock()
	if a.isBusy {
		a.mu.Unlock()
		return
	}
	mgr := a.mgr
	overview := a.overview
	win := a.win
	maskAccounts := a.maskAccounts
	a.mu.Unlock()

	if mgr == nil || win == nil {
		return
	}

	normEmail := strings.ToLower(strings.TrimSpace(email))
	if normEmail == "" {
		return
	}

	// Invariant 1: Ngăn chặn xóa tài khoản Active
	if overview != nil && strings.EqualFold(overview.ActiveEmail, normEmail) {
		fyne.Do(func() {
			a.showStatusBar("⚠️ Không thể xóa tài khoản đang hoạt động.")
		})
		return
	}

	// Phân loại thẻ có token hay thẻ [Cần OAuth]
	hasToken := false
	if overview != nil {
		for _, acc := range overview.Accounts {
			if strings.EqualFold(acc.Email, normEmail) {
				hasToken = acc.HasToken
				break
			}
		}
	}

	displayEmail := normEmail
	if maskAccounts {
		displayEmail = MaskEmail(normEmail)
	}

	title := "Xác nhận xóa tài khoản"
	msg := fmt.Sprintf("Bạn có chắc chắn muốn gỡ tài khoản %s khỏi màn hình chính không?", displayEmail)
	if hasToken {
		msg = fmt.Sprintf("Bạn có chắc chắn muốn xóa tài khoản %s khỏi hệ thống? Thao tác này sẽ xóa token đã lưu và bạn sẽ cần đăng nhập lại.", displayEmail)
	}

	msgLabel := widget.NewLabel(msg)
	msgLabel.Wrapping = fyne.TextWrapWord

	confirmDlg := dialog.NewCustomConfirm(
		title,
		"Xóa",
		"Hủy",
		msgLabel,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			go a.executeRemoveAccount(normEmail, displayEmail)
		},
		win,
	)
	confirmDlg.SetConfirmImportance(widget.DangerImportance)
	confirmDlg.Show()
}
```

### 5.2. Hàm Thực Thi Xóa Ngầm `executeRemoveAccount`

```go
func (a *App) executeRemoveAccount(email string, displayEmail string) {
	a.mu.Lock()
	a.isBusy = true
	mgr := a.mgr
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isBusy = false
		a.mu.Unlock()
	}()

	// 1. Thực thi xóa nghiệp vụ qua Manager
	if err := mgr.RemoveAccount(context.Background(), email); err != nil {
		fyne.Do(func() {
			a.showStatusBar(fmt.Sprintf("❌ Lỗi xóa tài khoản: %v", err))
		})
		return
	}

	// 2. Dọn dẹp RAM Cache
	if a.quotaCache != nil {
		a.quotaCache.Delete(email)
	}

	// 3. Đọc lại overview mới từ Manager
	newOverview, err := mgr.GetAccountStates(context.Background())
	if err != nil || newOverview == nil {
		return
	}

	a.mu.Lock()
	a.overview = newOverview
	a.mu.Unlock()

	// 4. Cập nhật UI trên main thread
	fyne.Do(func() {
		a.updateUI(newOverview)
		a.showStatusBar(fmt.Sprintf("🗑 Đã xóa tài khoản %s khỏi hệ thống.", displayEmail))
	})

	// 5. Invariant 6: Kiểm tra an toàn Auto-switch
	// Đếm số tài khoản hợp lệ còn lại có Token và Status == "on"
	validCandidates := 0
	for _, acc := range newOverview.Accounts {
		if acc.HasToken && strings.ToLower(strings.TrimSpace(acc.Status)) != "off" {
			validCandidates++
		}
	}

	if validCandidates == 0 {
		a.mu.Lock()
		if a.autoSwitch {
			a.autoSwitch = false
			if a.autoSwitchCheck != nil {
				a.autoSwitchCheck.SetChecked(false)
			}
			if a.autoController != nil {
				a.autoController.Stop()
			}
			// Lưu cấu hình auto_switch: false vào ui_preferences.json
			_ = a.savePreferences()
			fyne.Do(func() {
				a.showStatusBar("⚠️ Không có tài khoản để chuyển.")
			})
		}
		a.mu.Unlock()
	}
}
```

### 5.3. Kết Nối Callback tại các vị trí sinh Widget Thẻ trong `app.go`

Trong [`internal/ui/app.go`](file:///g:/antigravity_auto_switch/internal/ui/app.go):
- Tại dòng **769-776** (cập nhật thẻ đơn lẻ):
  ```go
  cardWidget := NewAccountCardWidgetWithQuota(
      targetItem,
      quotaData,
      palette,
      a.SwitchAccount,
      a.RefreshSingleAccountQuota,
      a.ToggleAccountStatus,
      a.ConfirmAndDeleteAccount, // <-- BỔ SUNG ĐỐI SỐ THỨ 7
  )
  ```
- Tại dòng **1337** (dựng danh sách thẻ trong `updateUI`):
  ```go
  cardWidget := NewAccountCardWidgetWithQuota(
      acc,
      quotaData,
      palette,
      a.SwitchAccount,
      a.RefreshSingleAccountQuota,
      a.ToggleAccountStatus,
      a.ConfirmAndDeleteAccount, // <-- BỔ SUNG ĐỐI SỐ THỨ 7
  )
  ```

---

# PHẦN IV: HƯỚNG DẪN TRIỂN KHAI CHO CODER (TOUCH-MAP & CODE SNIPPETS)

Coder thực hiện tuần tự theo 5 bước chính:

```text
┌────────────────────────────────────────────────────────┐
│ BƯỚC 1: internal/telemetry/cache.go                    │
│ • Thêm method Delete(email string) cho QuotaCache      │
├────────────────────────────────────────────────────────┤
│ BƯỚC 2: internal/manager/account_manager.go            │
│ • Thêm ErrCannotDeleteActiveAccount                    │
│ • Thêm method RemoveAccount(ctx, email)                │
├────────────────────────────────────────────────────────┤
│ BƯỚC 3: internal/ui/widgets.go                         │
│ • Thêm field onDelete, deleteButton vào struct         │
│ • Thêm nút theme.DeleteIcon() vào vị trí 4 actionCluster│
│ • Cập nhật NewAccountCardWidgetWithQuota & UpdateState │
│ • Thêm getter DeleteButton(), TriggerDelete()          │
├────────────────────────────────────────────────────────┤
│ BƯỚC 4: internal/ui/app.go                             │
│ • Thêm method ConfirmAndDeleteAccount(email string)    │
│ • Thêm method executeRemoveAccount(email, display)     │
│ • Truyền a.ConfirmAndDeleteAccount vào các chỗ gọi thẻ │
├────────────────────────────────────────────────────────┤
│ BƯỚC 5: Viết Kiểm Thử Đơn Vị & Chấp Nhận               │
│ • internal/manager/account_manager_test.go             │
│ • internal/ui/widgets_test.go                          │
│ • internal/ui/account_deletion_acceptance_test.go      │
└────────────────────────────────────────────────────────┘
```

---

# PHẦN V: BỘ TIÊU CHUẨN KIỂM THỬ CHẤP NHẬN (ACCEPTANCE TEST CRITERIA)

Coder bắt buộc phải viết các test case sau và đảm bảo `go test ./...` đạt 100% PASS:

| Mã Test | Phạm Vi Kiểm Thử | Dữ Liệu Đầu Vào | Hành Vi & Kết Quả Kỳ Vọng |
|---|---|---|---|
| **TC-01** | **Xóa thẻ Inactive Status ON** | Thẻ `user1@test.com` (`HasToken: true`, `Status: "on"`, `IsActive: false`) | Nút `[ 🗑 ]` hiển thị. Bấm nút -> Pop-up xác nhận hiện nội dung xóa token. Bấm `[Xóa]` -> File `accounts_store/user1@test.com.json` bị xóa, `discovered_accounts/` có `added: false`. Thẻ biến mất khỏi UI. |
| **TC-02** | **Xóa thẻ Inactive Status OFF** | Thẻ `user2@test.com` (`HasToken: true`, `Status: "off"`, `IsActive: false`) | Nút `[Chuyển]` bị disabled, nhưng nút `[ 🗑 ]` vẫn sáng rõ và click được. Bấm `[Xóa]` -> File token bị xóa. Thẻ biến mất khỏi UI. |
| **TC-03** | **Xóa thẻ `[Cần OAuth]`** | Thẻ `user3@test.com` (`HasToken: false`, `IsActive: false`, `added: true`) | Nút `[ 🗑 ]` hiển thị. Bấm nút -> Pop-up hiện thông điệp gỡ thẻ khỏi màn hình chính. Bấm `[Xóa]` -> `discovered_accounts/` đổi `added: false`. Thẻ biến mất khỏi UI chính, quay về Modal. |
| **TC-04** | **Chặn xóa thẻ Active** | Thẻ `active@test.com` (`IsActive: true`, mang nhãn `[✔ Dùng]`) | Nút `[ 🗑 ]` bị ẩn hoàn toàn (`deleteButton.Visible() == false`). Nếu cố tình gọi `mgr.RemoveAccount` trực tiếp từ code -> Trả về lỗi `ErrCannotDeleteActiveAccount`. |
| **TC-05** | **Hủy bỏ Dialog (Dismiss)** | Bấm nút `[ 🗑 ]` trên bất kỳ thẻ nào | Hộp thoại xác nhận hiện ra. Bấm nút `[Hủy]` -> Hộp thoại đóng lại. File trên đĩa không bị thay đổi, thẻ trên UI vẫn giữ nguyên vị trí. |
| **TC-06** | **Thứ tự nút trên ActionCluster** | Kiểm tra cấu trúc phân cấp widget thẻ | Kiểm tra `actionCluster.Objects`: vị trí 0 là `actionSlot`, vị trí 1 là `refreshBtn`, vị trí 2 là `statusBtn`, vị trí 3 là `deleteBtn`. Đúng 100% Invariant 2. |
| **TC-07** | **Dọn dẹp Quota Cache** | Quota cache đang lưu dữ liệu của `user1@test.com` | Xóa `user1@test.com` -> `a.quotaCache.Get("user1@test.com")` trả về `false`. |
| **TC-08** | **Tự động ngắt Auto-Switch khi hết tài khoản** | `auto_switch: true`, hệ thống chỉ còn 1 tài khoản duy nhất có token | Người dùng xóa tài khoản này -> Hệ thống tự động chuyển `auto_switch: false`, lưu `ui_preferences.json`, thanh trạng thái hiển thị `⚠️ Không có tài khoản để chuyển.`. |

---
**Bản đặc tả này đã được khóa nội dung (Architecture Locked). Coder triển khai chính xác theo từng bước mà không được tùy tiện làm sai lệch các Invariant.**
