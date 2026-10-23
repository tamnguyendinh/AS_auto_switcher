# BÁO CÁO ĐẶC TẢ KIẾN TRÚC (ARCHITECTURE SPECIFICATION REPORT)

- **Mã tài liệu:** `rp_architect_20261023_131500_add_modal_delete_and_nonexistent_account_spec`
- **Tác giả:** System Architect Lane (Lane Sạch)
- **Ngày lập:** 2026-10-23
- **Thẩm quyền phê duyệt:** Human Owner
- **Đối tượng thụ hưởng:** Coder Lane, QA Lane, Supervisor Lane
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_CODER`**

---

## 1. MỤC TIÊU & PHẠM VI (GOALS & SCOPE)

### 1.1. Mục tiêu
Tài liệu này đặc tả toàn diện kiến trúc kỹ thuật và giải pháp triển khai cho Modal **"Thêm tài khoản"** (`Add Account Modal`), bao gồm:
1. Bổ sung **Nút [Xóa]** phổ quát trên mọi dòng tài khoản hiển thị trong Modal Thêm tài khoản để xóa file JSON tương ứng trong thư mục `discovered_accounts/`.
2. Bổ sung **Hộp thoại Popup Xác nhận Xóa (Confirmation Dialog)**: Khi người dùng bấm nút [Xóa], hệ thống BẮT BUỘC phải bật popup cảnh báo xác nhận, chỉ khi người dùng bấm "Xóa" trên popup thì thao tác xóa mới được kích hoạt, tuyệt đối ngăn chặn rủi ro xóa nhầm do bấm nhầm chuột.
3. Bổ sung trạng thái mới: **`[Tài khoản không tồn tại]`** xuất hiện khi bấm nút **[Quét Chrome]** đối với những tài khoản hiển thị trên UI modal nhưng file JSON trong `discovered_accounts/` hoàn toàn không tồn tại (thay thế cho nhãn `[Đã add]` / `[Cần OAuth]`), đi kèm nút **[Xóa]** để gỡ dòng đó khỏi danh sách hiển thị Modal.

### 1.2. Ranh giới tuyệt đối (Scope Boundaries & Non-Goals)
- **Tách biệt hai nút Xóa (Deletion Isolation):** Nút [Xóa] trong Modal Thêm tài khoản **chỉ** thao tác trên tầng danh bạ khám phá (`discovered_accounts/`). Tuyệt đối **KHÔNG** xóa token trong `accounts_store/`, và tuyệt đối **KHÔNG** xóa hay can thiệp vào thẻ tài khoản trên màn hình chính (`UI chính`).
- **Không hạ chuẩn sản xuất (No MVP):** Toàn bộ hàm mới phải đảm bảo tính `thread-safe`, `idempotent`, xử lý triệt để các trường hợp biên (`edge cases`).

---

## 2. PHÂN TÍCH HIỆN TRẠNG & NGUYÊN NHÂN GỐC RỄ (ROOT CAUSE ANALYSIS)

Kiến trúc hiện tại của dự án phân tách hai kho dữ liệu riêng biệt:
- `accounts_store/`: Lưu trữ các tài khoản đang vận hành thực tế (chứa token/credential). Xóa tài khoản tại đây do hàm `RemoveAccount()` đảm nhiệm.
- `discovered_accounts/`: Lưu trữ danh bạ tài khoản phát hiện từ Chrome (`DiscoveredAccountEnvelope` gồm `email`, `scanned_at`, `added`).

### Điểm nghẽn hiện tại trong Codebase:
1. **Tại `internal/store/discovered_store.go`:**
   - Chỉ có `SaveDiscovered()`, `SetDiscoveredAdded()`, `ListDiscovered()`, `ListDiscoveredRecords()`.
   - **Thiếu:** Hàm xóa file `DeleteDiscovered(email string) error` và hàm kiểm tra tồn tại `HasDiscovered(email string) bool`.
2. **Tại `internal/manager/account_manager.go`:**
   - Struct `CandidateAccountItem` chỉ có 3 trường: `Email string`, `HasToken bool`, `Added bool`.
   - **Thiếu:** Trường xác định sự tồn tại thực tế của file JSON trên đĩa (`FileExists bool`).
   - Hàm `GetCandidateAccounts()` hiện tại chỉ duyệt qua các records đọc được từ `discoveredStore.ListDiscoveredRecords()`. Nếu một tài khoản đang có mặt trên UI (ví dụ thẻ unauth `[Cần OAuth]` do đã Add trước đó) nhưng file JSON trong `discovered_accounts/` bị xóa/mất -> tài khoản đó bị biến mất hoàn toàn thay vì báo trạng thái không tồn tại.
   - **Thiếu:** Hàm nghiệp vụ `DeleteDiscoveredAccount(email string) error`.
3. **Tại `internal/ui/add_modal.go`:**
   - Chỉ hiển thị 3 trạng thái cố định: `[✔ Đã OAuth]`, `[Đã add]`, `[Cần OAuth] + [ADD]`.
   - **Thiếu:** Nút [Xóa] trên từng dòng tài khoản.
   - **Thiếu:** Trạng thái `[Tài khoản không tồn tại]` khi file JSON trong `discovered_accounts/` không tồn tại.

---

## 3. CÁC NGUYÊN TẮC BẤT BIẾN BẮT BUỘC (ARCHITECTURAL INVARIANTS)

Coder và QA **BẮT BUỘC** tuân thủ 100% các Invariant sau:

1. **`Invariant-Store-Isolation` (Cô lập hai kho):**
   - Nút [Xóa] trong Modal Thêm tài khoản CHỈ xóa file JSON trong `discovered_accounts/`.
   - Không được phép gọi `RemoveAccount()` hay xóa bất kỳ file nào trong `accounts_store/`.
   - Không làm mất thẻ tài khoản đang chạy trên UI chính.
2. **`Invariant-Idempotent-Delete` (Xóa bất biến an toàn):**
   - Khi xóa một tài khoản trong `discovered_accounts/`, nếu file JSON đã không còn tồn tại trên đĩa (`os.IsNotExist`), thao tác xóa coi như thành công và trả về `nil` (không crash, không ném lỗi giả).
3. **`Invariant-Delete-Confirmation-Dialog` (Bắt buộc Popup Xác nhận Xóa):**
   - Khi người dùng bấm nút [Xóa] trên bất kỳ dòng tài khoản nào trong Modal, hệ thống **BẮT BUỘC** phải bật popup hộp thoại xác nhận (`dialog.NewConfirm` hoặc `dialog.NewCustomConfirm`).
   - Tuyệt đối **CẤM** xóa ngay lập tức khi chỉ vừa click nút Xóa trên dòng.
   - Thao tác xóa file và cập nhật UI chỉ được phép thực thi sau khi người dùng bấm nút xác nhận trên Popup. Nếu người dùng bấm "Hủy" hoặc đóng popup, hệ thống giữ nguyên hiện trạng.
4. **`Invariant-Ghost-Account-State` (Phát hiện tài khoản ma):**
   - Khi người dùng nhấn nút **[Quét Chrome]**: Bất kỳ tài khoản nào nằm trong danh sách theo dõi của modal (hoặc đang có thẻ `[Cần OAuth]` trên UI chính) mà kiểm tra trên đĩa cứng KHÔNG tìm thấy file JSON trong `discovered_accounts/` -> BẮT BUỘC phải chuyển thành nhãn **`[Tài khoản không tồn tại]`**.
   - Trạng thái này thay thế cho nhãn `[Đã add]` / `[Cần OAuth]`.
   - Tuyệt đối KHÔNG hiển thị nút `[ADD]` trên dòng `[Tài khoản không tồn tại]`.
5. **`Invariant-Modal-Removal` (Làm sạch danh sách Modal):**
   - Sau khi xác nhận trên popup xóa, dòng tài khoản đó (kể cả dòng `[Tài khoản không tồn tại]`) phải ngay lập tức bị loại bỏ khỏi danh sách hiển thị của Modal.

---

## 4. BẢNG MA TRẬN TRẠNG THÁI MODAL (STATE MATRIX)

| STT | Trạng thái kỹ thuật | Điều kiện nhận diện | Nhãn hiển thị trên Modal | Nút thao tác đi kèm | Hành vi khi bấm [Xóa] |
|:---:|:---|:---|:---:|:---|:---|
| **1** | Đã có Token xác thực | `HasToken == true` && `FileExists == true` | `[✔ Đã OAuth]` *(Màu Success)* | Nút **[Xóa]** | Xóa file trong `discovered_accounts/` & gỡ khỏi danh sách Modal |
| **2** | Đã Add vào UI chính | `Added == true` && `FileExists == true` | `[Đã add]` *(Màu TextMuted)* | Nút **[Xóa]** | Xóa file trong `discovered_accounts/` & gỡ khỏi danh sách Modal *(thẻ UI chính vẫn giữ nguyên)* |
| **3** | Mới quét, chưa Add | `!Added && !HasToken` && `FileExists == true` | `[Cần OAuth]` *(Màu Warning)* | Nút **[ADD]** + Nút **[Xóa]** | Xóa file trong `discovered_accounts/` & gỡ khỏi danh sách Modal |
| **4** | **Tài khoản không tồn tại** | `FileExists == false` *(sau khi Quét Chrome)* | **`[Tài khoản không tồn tại]`** *(Màu TextMuted/Warning)* | Nút **[Xóa]** *(Ẩn nút ADD)* | Gỡ ngay dòng tài khoản khỏi danh sách Modal |

---

## 5. ĐẶC TẢ CHI TIẾT TỪNG TẦNG (DETAILED IMPLEMENTATION SPECIFICATION)

### 5.1. Tầng Store: `internal/store/discovered_store.go`

#### A. Hàm `DeleteDiscovered(email string) error`
```go
// DeleteDiscovered removes the discovered account JSON file for an email.
// It is idempotent: if the file does not exist, it returns nil.
func (s *DiscoveredStore) DeleteDiscovered(email string) error {
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

    if err := os.Remove(path); err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }
    return nil
}
```

#### B. Hàm `HasDiscovered(email string) bool`
```go
// HasDiscovered checks if the discovered account JSON file exists on disk.
func (s *DiscoveredStore) HasDiscovered(email string) bool {
    email = strings.ToLower(strings.TrimSpace(email))
    if email == "" {
        return false
    }
    path, err := SafeAccountPath(s.dir, email)
    if err != nil {
        return false
    }

    s.mu.RLock()
    defer s.mu.RUnlock()

    info, err := os.Stat(path)
    return err == nil && !info.IsDir()
}
```

---

### 5.2. Tầng Manager: `internal/manager/account_manager.go`

#### A. Cập nhật Model `CandidateAccountItem`
```go
type CandidateAccountItem struct {
    Email       string
    HasToken    bool
    Added       bool
    FileExists  bool // true: file JSON trong discovered_accounts tồn tại; false: không tồn tại
}
```

#### B. Thêm Hàm `DeleteDiscoveredAccount(email string) error`
```go
// DeleteDiscoveredAccount deletes the discovered account JSON file from discovered_accounts/.
// It does NOT touch accounts_store/ or main UI active credentials.
func (m *AccountManager) DeleteDiscoveredAccount(email string) error {
    m.mu.RLock()
    discovered := m.discoveredStore
    m.mu.RUnlock()

    if discovered == nil {
        return errors.New("discovered store uninitialized")
    }
    return discovered.DeleteDiscovered(email)
}
```

#### C. Nâng cấp `GetCandidateAccounts()`
Coder cập nhật logic hợp nhất tài khoản:
1. Đọc tất cả `records` từ `discovered.ListDiscoveredRecords()`. Đánh dấu `FileExists: true`.
2. Kiểm tra các tài khoản đang tồn tại trên màn hình chính (ở trạng thái `StateUnauth` hoặc cờ `Added` từ `GetAccountStates`):
   - Nếu có tài khoản nào xuất hiện trong danh sách quản lý của ứng dụng nhưng khi kiểm tra `discovered.HasDiscovered(email)` trả về `false`:
   - Bổ sung tài khoản đó vào danh sách `CandidateAccountItem` với:
     ```go
     CandidateAccountItem{
         Email:      email,
         HasToken:   storedSet[email],
         Added:      true,
         FileExists: false, // Báo hiệu file không tồn tại
     }
     ```
3. Giữ nguyên thuật toán sắp xếp ưu tiên (Token -> Added -> Tên email).

---

### 5.3. Tầng Giao diện UI: `internal/ui/add_modal.go`

#### A. Nút [Xóa] & Hộp thoại Popup Xác nhận Xóa (Confirmation Popup)
- Trên tất cả các dòng tài khoản trong `listContainer`, `actionSlot` phải chứa một nút **[Xóa]** (`theme.DeleteIcon()`, bọc qua `compactButton` để có chiều cao gọn 24px).
- **Hành vi khi bấm [Xóa] (Bắt buộc bật Popup cảnh báo):**
  - Tuyệt đối không xóa ngay lập tức.
  - Khi click nút Xóa, gọi hàm mở popup xác nhận `a.showDeleteCandidateConfirm(candItem, reloadCandidates)`.
  - **Thiết kế Popup Xác nhận:**
    ```go
    func (a *App) showDeleteCandidateConfirm(cand CandidateAccountItem, onConfirmed func()) {
        displayEmail := cand.Email
        if a.IsMaskAccounts() {
            displayEmail = MaskEmail(cand.Email)
        }

        title := "Xác nhận xóa tài khoản"
        msg := fmt.Sprintf("Bạn có chắc chắn muốn xóa tài khoản %s khỏi danh bạ đã quét không?\n\n(Lưu ý: Thao tác này chỉ xóa dữ liệu trong danh bạ khám phá, không xóa tài khoản hay token trên màn hình chính).", displayEmail)

        if !cand.FileExists {
            msg = fmt.Sprintf("Tài khoản %s không có tệp dữ liệu trên hệ thống.\nBạn có muốn gỡ bỏ tài khoản này khỏi danh sách hiển thị không?", displayEmail)
        }

        confirmDlg := dialog.NewConfirm(
            title,
            msg,
            func(confirmed bool) {
                if !confirmed {
                    return
                }
                if cand.FileExists {
                    _ = a.mgr.DeleteDiscoveredAccount(cand.Email)
                }
                fyne.Do(func() {
                    onConfirmed()
                })
            },
            a.win,
        )
        confirmDlg.SetConfirmImportance(widget.DangerImportance)
        confirmDlg.Show()
    }
    ```

#### B. Xử lý hiển thị 4 trạng thái
Trong vòng lặp render danh sách tại `reloadCandidates()`:

```go
var actionSlot *fyne.Container

deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
    a.showDeleteCandidateConfirm(candItem, reloadCandidates)
})
deleteBtn.Importance = widget.LowImportance
compactDel := compactButton(deleteBtn)

if !candItem.FileExists {
    // TRƯỜNG HỢP 4: [Tài khoản không tồn tại] + Nút [Xóa]
    statusBadge := canvas.NewText("[Tài khoản không tồn tại]", palette.TextMuted)
    statusBadge.TextSize = 11
    statusBadge.TextStyle = fyne.TextStyle{Bold: true}

    actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

} else if candItem.HasToken {
    // TRƯỜNG HỢP 1: [✔ Đã OAuth] + Nút [Xóa]
    statusBadge := canvas.NewText("[✔ Đã OAuth]", palette.Success)
    statusBadge.TextSize = 11
    statusBadge.TextStyle = fyne.TextStyle{Bold: true}

    actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

} else if candItem.Added {
    // TRƯỜNG HỢP 2: [Đã add] + Nút [Xóa]
    statusBadge := canvas.NewText("[Đã add]", palette.TextMuted)
    statusBadge.TextSize = 11
    statusBadge.TextStyle = fyne.TextStyle{Bold: true}

    actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactDel)

} else {
    // TRƯỜNG HỢP 3: [Cần OAuth] + Nút [ADD] + Nút [Xóa]
    statusBadge := canvas.NewText("[Cần OAuth]", palette.Warning)
    statusBadge.TextSize = 11
    statusBadge.TextStyle = fyne.TextStyle{Bold: true}

    addBtn := widget.NewButtonWithIcon("ADD", theme.ContentAddIcon(), func() {
        if err := a.mgr.AddDiscoveredAccount(candItem.Email); err == nil {
            // Chuyển đổi tại chỗ: [Cần OAuth] đổi thành [Đã add] và nút ADD biến mất
            doneBadge := canvas.NewText("[Đã add]", palette.TextMuted)
            doneBadge.TextSize = 11
            doneBadge.TextStyle = fyne.TextStyle{Bold: true}
            actionSlot.Objects = []fyne.CanvasObject{container.NewCenter(doneBadge), compactDel}
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

    actionSlot = container.NewHBox(container.NewCenter(statusBadge), compactButton(addBtn), compactDel)
}
```

#### C. Hành vi khi bấm nút [Quét Chrome]
- Nút `scanBtn` khi được nhấn:
  1. Disable nút và hiện `⏳ Đang quét...`.
  2. Chạy goroutine gọi `a.mgr.ScanChromeAccounts()`.
  3. Khi scan hoàn tất, gọi `fyne.Do(func() { reloadCandidates(); scanBtn.SetText("Quét Chrome"); scanBtn.Enable() })`.
  4. Trong quá trình `reloadCandidates()`, các tài khoản hiển thị trên UI modal nhưng không có file trong `discovered_accounts/` sẽ tự động hiển thị trạng thái `[Tài khoản không tồn tại]`.

---

## 6. DANH MỤC KIỂM THỬ BẮT BUỘC (TESTING & VERIFICATION MATRIX)

Coder và QA phải đảm bảo các bộ test sau được viết và chạy PASS 100%:

1. **Unit Test Store (`internal/store/discovered_store_test.go`):**
   - `TestDiscoveredStore_DeleteDiscovered_Success`: Tạo file -> gọi `DeleteDiscovered` -> xác nhận file biến mất trên đĩa.
   - `TestDiscoveredStore_DeleteDiscovered_Idempotent`: Gọi `DeleteDiscovered` trên file không tồn tại -> xác nhận trả về `nil`.
   - `TestDiscoveredStore_HasDiscovered`: Xác nhận trả về `true` khi có file, `false` khi không có file.
2. **Unit Test Manager (`internal/manager/two_store_test.go`):**
   - `TestDeleteDiscoveredAccount_Isolation`: Gọi `DeleteDiscoveredAccount` -> xác nhận file trong `discovered_accounts/` bị xóa nhưng `accounts_store/` tuyệt đối không bị ảnh hưởng.
   - `TestGetCandidateAccounts_NonExistentDetection`: Giả lập tài khoản có trên UI nhưng không có file trong `discovered_accounts/` -> kiểm tra `FileExists == false`.
3. **Acceptance Test UI (`internal/ui/add_modal_test.go` & `add_modal_acceptance_test.go`):**
   - **TC-Modal-Delete-Button-Rendered**: Kiểm tra cả 4 trạng thái dòng tài khoản đều render nút Xóa (`theme.DeleteIcon()`).
   - **TC-Modal-Delete-Click-Shows-Confirm-Dialog**: Nhấn nút Xóa trên dòng tài khoản -> xác nhận popup hộp thoại xác nhận (`dialog.Confirm`) xuất hiện, file chưa bị xóa ngay.
   - **TC-Modal-Delete-Dialog-Cancel**: Nhấn nút "Hủy" trên popup xác nhận -> file không bị xóa và dòng tài khoản vẫn giữ nguyên trên modal.
   - **TC-Modal-Delete-Dialog-Confirm**: Nhấn nút "Xóa" trên popup xác nhận -> file bị xóa khỏi `discovered_accounts/`, dòng biến mất khỏi modal, nhưng thẻ ngoài UI chính không bị mất.
   - **TC-Modal-ScanChrome-NonExistent-Status**: Cho 1 tài khoản có trên UI nhưng cố tình xóa file JSON trong `discovered_accounts/`. Nhấn nút **[Quét Chrome]** -> kiểm tra modal hiển thị nhãn `[Tài khoản không tồn tại]`, nút ADD bị ẩn, nút Xóa hiển thị.
   - **TC-Modal-Delete-NonExistent**: Bấm nút Xóa trên dòng `[Tài khoản không tồn tại]` và xác nhận trên popup -> kiểm tra dòng đó bị xóa sạch khỏi danh sách modal.

---

## 7. KẾT LUẬN & HƯỚNG DẪN BÀN GIAO CHO CODER

Tài liệu này đã giải quyết triệt để 100% bài toán mà Owner đặt ra với đầy đủ cấu trúc dữ liệu, signature hàm, luồng giao diện và các ca kiểm thử cụ thể.

- **Đánh giá của Kiến trúc sư:** **`VERDICT: PASS / READY_FOR_CODER`**
- **Chỉ dẫn cho Coder:** Coder triển khai lần lượt theo thứ tự: Tầng Store (`discovered_store.go`) -> Tầng Manager (`account_manager.go`) -> Tầng UI (`add_modal.go`), sau đó bổ sung unit/acceptance test theo mục 6 để hoàn tất.
