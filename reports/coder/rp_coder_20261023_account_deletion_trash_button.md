# BÁO CÁO THI CÔNG CODER: NÚT XÓA TÀI KHOẢN (TRASH BUTTON) & ĐỒNG BỘ HAI KHO DỮ LIỆU

- **Mã tài liệu:** `rp_coder_20261023_account_deletion_trash_button.md`
- **Tác giả:** Coder
- **Ngày thi công:** 2026-10-23
- **Đặc tả kiến trúc thẩm quyền (Authority / SSOT):** `reports/architect/2026-10-23-account-deletion-and-trash-button-spec.md` (`SPEC-ARCH-20261023-ACCOUNT-DELETION-AND-TRASH-BUTTON`)
- **Trạng thái:** HOÀN THÀNH 100% (READY FOR SUPERVISOR)

---

## I. BẢN ĐỒ DÒNG DÕI BẤT BIẾN (INVARIANT FAMILY MAP)

- **Tên họ bất biến (Invariant Family):** `FAMILY_ACCOUNT_DELETION_SYNC`
- **Nguồn thẩm quyền (SSOT):** `SPEC-ARCH-20261023-ACCOUNT-DELETION-AND-TRASH-BUTTON`
- **6 Bất biến tối cao (Supreme Invariants Conformance):**
  1. **Invariant 1 — Active-Account Protection Invariant:** Tuyệt đối không hiển thị nút Xóa `[ 🗑 ]` trên thẻ tài khoản đang Active (`IsActive == true`, mang badge `[✔ Dùng]`). Phương thức `RemoveAccount` từ chối ngay lập tức nếu cố tình xóa tài khoản Active với lỗi `ErrCannotDeleteActiveAccount`.
  2. **Invariant 2 — Action-Cluster Order Invariant:** Cụm nút điều khiển bên phải thẻ tài khoản (`actionCluster`) tuân thủ nghiêm ngặt thứ tự 4 vị trí từ trái sang phải:
     $$\text{[ 1. Action Slot ]} \quad \longrightarrow \quad \text{[ 2. Refresh Quota ]} \quad \longrightarrow \quad \text{[ 3. Status ON/OFF ]} \quad \longrightarrow \quad \text{[ 4. Trash Button 🗑 ]}$$
  3. **Invariant 3 — Confirmation Dialog Mandatory Invariant:** Thao tác nhấn `[ 🗑 ]` mở Pop-up xác nhận modal native Fyne (`dialog.NewCustomConfirm`) với tiêu đề "Xác nhận xóa tài khoản", nút "Xóa" (màu đỏ DangerImportance) và nút "Hủy". Chỉ khi người dùng bấm xác nhận "Xóa", tiến trình xóa đĩa ngầm mới được thực thi.
  4. **Invariant 4 — Two-Store Atomic Deletion Invariant:**
     - Thẻ có Token (`HasToken == true`): Xóa file `accounts_store/<email>.json` VÀ đồng thời cập nhật `Added: false` trong `discovered_accounts/<email>.json`.
     - Thẻ `[Cần OAuth]` (`HasToken == false`): Cập nhật `Added: false` trong `discovered_accounts/<email>.json`.
     - Kết quả: Thẻ biến mất hoàn toàn khỏi màn hình chính, quay về danh sách ứng viên trong Modal `[+ Thêm]`.
  5. **Invariant 5 — Status OFF Deletion Invariant:** Khi tài khoản ở trạng thái `Status == "off"`, nút `[ 🗑 ]` vẫn sáng rõ và hoạt động bình thường, cho phép dọn dẹp các tài khoản đã vô hiệu hóa.
  6. **Invariant 6 — Memory Cleanup & Engine Safety Invariant:** Khi xóa thành công:
     - Dọn dẹp cache quota qua `quotaCache.Delete(email)`.
     - Nạp lại `overview` và cập nhật UI trên main thread (`fyne.Do`).
     - Nếu hệ thống đang bật `autoSwitch` và không còn tài khoản ứng viên hợp lệ (`HasToken && Status != "off"`), tự động tắt auto-switch, dừng controller, lưu `ui_preferences.json`, và thông báo trạng thái: `⚠️ Không có tài khoản để chuyển.`.

- **Các bề mặt runtime liên đới (Sibling Surfaces):**
  - `internal/telemetry/cache.go`: `QuotaCache.Delete(email string)`
  - `internal/manager/account_manager.go`: `ErrCannotDeleteActiveAccount`, `RemoveAccount(ctx, email)`
  - `internal/ui/widgets.go`: `AccountCardWidget.onDelete`, `deleteButton`, `deleteContainer`, `DeleteButton()`, `ActionCluster()`, `TriggerDelete()`, `SetOnDelete()`, `NewAccountCardWidgetWithQuota`, `UpdateState`
  - `internal/ui/app.go`: `ConfirmAndDeleteAccount(email)`, `executeRemoveAccount(email, displayEmail)`, `updateUI`, `Close()`
- **Trạng thái đường dẫn lỗi thời / fallback:** Loại bỏ triệt để mọi fallback trái luồng.
- **Bề mặt chưa được kiểm thử (Residual Unverified Surfaces):** `none` (100% được chứng minh qua unit tests và acceptance suite).

---

## II. CHI TIẾT THI CÔNG THEO TOUCH-MAP (5 BƯỚC)

### 1. Bước 1: `internal/telemetry/cache.go`
- Bổ sung phương thức `Delete(email string)` trên struct `QuotaCache`:
  Xóa khóa bộ nhớ đệm quota của tài khoản dưới khóa ghi an toàn `mu.Lock()`.
- Chạy kiểm thử: `go test -v ./internal/telemetry` PASS 100%.

### 2. Bước 2: `internal/manager/account_manager.go`
- Khai báo lỗi chuẩn hóa `ErrCannotDeleteActiveAccount = errors.New("không thể xóa tài khoản đang hoạt động")`.
- Cung cấp getter/setter `DiscoveredStore()` / `SetDiscoveredStore(*store.DiscoveredStore)`.
- Triển khai phương thức `RemoveAccount(ctx context.Context, email string) error`:
  1. Chuẩn hóa email, kiểm tra `strings.TrimSpace(email) == ""` trả về `store.ErrInvalidEmail`.
  2. Kiểm tra `GetCurrentActiveEmail(ctx)` -> Nếu trùng email trả về `ErrCannotDeleteActiveAccount`.
  3. Xóa tệp credentials trong `accounts_store/<email>.json` qua `accStore.DeleteAccount(normEmail)`.
  4. Đọc danh sách `discovered_accounts` và cập nhật `SetDiscoveredAdded(normEmail, false)` nếu record tồn tại.
- Chạy kiểm thử: `go test -v -run TestAccountManager_RemoveAccount ./internal/manager` PASS 100%.

### 3. Bước 3: `internal/ui/widgets.go`
- Mở rộng struct `AccountCardWidget`:
  - `actionCluster *fyne.Container`
  - `deleteButton *widget.Button`
  - `deleteContainer *fyne.Container`
  - `onDelete func(string)`
- Bổ sung các phương thức getter & trigger phục vụ kiểm thử:
  - `DeleteButton() *widget.Button`
  - `ActionCluster() *fyne.Container`
  - `TriggerDelete()`
  - `SetOnDelete(fn func(email string))`
- Cập nhật hàm khởi tạo `NewAccountCardWidgetWithQuota`:
  - Tiếp nhận `extraCallbacks ...func(email string)`: đối số thứ 6 là `toggleFn`, đối số thứ 7 là `deleteFn` (duy trì tương thích ngược 100%).
  - Tạo nút Delete bằng `widget.NewButtonWithIcon("", theme.DeleteIcon(), ...)` bọc trong `compactButton(...)`.
  - Thiết lập `actionCluster = container.NewHBox(actionSlot, compactButton(refreshBtn), statusBtn, deleteWrapper)` đúng chuẩn 4 vị trí Invariant 2.
- Cập nhật hàm `UpdateState`:
  - Nếu `w.IsActive == true`: ẩn `deleteButton` và `deleteContainer` (`Hide()`).
  - Nếu `w.IsActive == false`: hiển thị và kích hoạt `deleteButton` và `deleteContainer` (`Show()`, `Enable()`).
- Chạy kiểm thử: `go test -v -run TestAccountCardWidget_DeleteButton ./internal/ui` PASS 100%.

### 4. Bước 4: `internal/ui/app.go`
- Bổ sung phương thức `ConfirmAndDeleteAccount(email string)`:
  - Kiểm tra trạng thái bận và ngăn chặn xóa tài khoản Active.
  - Phân loại thông điệp xác nhận theo loại thẻ:
    * Thẻ có Token: *"Bạn có chắc chắn muốn xóa tài khoản <email> khỏi hệ thống? Thao tác này sẽ xóa token đã lưu và bạn sẽ cần đăng nhập lại."*
    * Thẻ Cần OAuth: *"Bạn có chắc chắn muốn gỡ tài khoản <email> khỏi màn hình chính không?"*
  - Hỗ trợ che danh tính (`MaskEmail`) nếu bật `maskAccounts`.
  - Khởi tạo `dialog.NewCustomConfirm` với nút "Xóa" (`widget.DangerImportance`) và "Hủy".
- Bổ sung phương thức `executeRemoveAccount(email string, displayEmail string)`:
  - Gọi `mgr.RemoveAccount(context.Background(), email)`.
  - Xóa cache RAM qua `a.quotaCache.Delete(email)`.
  - Đọc overview mới và cập nhật giao diện trên luồng chính (`fyne.Do`).
  - Kiểm tra an toàn Auto-switch (Invariant 6): nếu số ứng viên còn lại bằng 0, kích hoạt `a.stopAutoSwitchWithNoCandidate()`.
- Nâng cấp `Close()` với cơ chế drain background tasks graceful, ngăn chặn tranh chấp khóa file trên Windows khi dọn dẹp thư mục tạm.
- Kết nối `a.ConfirmAndDeleteAccount` vào cả 2 vị trí khởi tạo thẻ trong `app.go` (cập nhật thẻ đơn lẻ và `updateUI`).

### 5. Bước 5: Kiểm Thử Đơn Vị & Chấp Nhận (Acceptance Tests)
- `internal/manager/account_manager_test.go`:
  - `TestAccountManager_RemoveAccount`: Kiểm tra chặn xóa active, kiểm tra xóa đồng bộ 2 kho, kiểm tra thẻ Cần OAuth, kiểm tra email rỗng.
- `internal/ui/widgets_test.go`:
  - `TestAccountCardWidget_DeleteButton_OrderAndHierarchy`: Kiểm tra thứ tự Invariant 2 và phân cấp widget.
  - `TestAccountCardWidget_DeleteButton_VisibilityStates`: Kiểm tra ẩn trên thẻ Active, hiển thị trên thẻ Inactive ON, Inactive OFF, Cần OAuth.
  - `TestAccountCardWidget_TriggerDelete_And_SetOnDelete`: Kiểm tra trigger an toàn.
- `internal/ui/account_deletion_acceptance_test.go`:
  - Bộ 8 Acceptance Test Cases TC-01..TC-08 đạt 100% PASS.

---

## III. KẾT QUẢ BỘ KIỂM THỬ CHẤP NHẬN (TC-01 .. TC-08)

| Mã Test | Phạm Vi Kiểm Thử | Dữ Liệu Đầu Vào | Kết Quả Thực Tế | Trạng Thái |
|---|---|---|---|:---:|
| **TC-01** | Xóa thẻ Inactive Status ON | Thẻ `user1@test.com` (`HasToken: true`, `Status: "on"`) | Nút `[ 🗑 ]` hiển thị. Bấm xóa -> Tệp token bị xóa, `discovered_accounts/` có `Added: false`. Thẻ biến mất khỏi UI. | **PASS** |
| **TC-02** | Xóa thẻ Inactive Status OFF | Thẻ `user2@test.com` (`HasToken: true`, `Status: "off"`) | Nút `[Chuyển]` bị disabled, nút `[ 🗑 ]` vẫn sáng rõ. Bấm xóa -> Tệp token bị xóa. Thẻ biến mất khỏi UI. | **PASS** |
| **TC-03** | Xóa thẻ `[Cần OAuth]` | Thẻ `user3@test.com` (`HasToken: false`, `Added: true`) | Nút `[ 🗑 ]` hiển thị. Dialog hiện câu gỡ thẻ. Bấm xóa -> `discovered_accounts/` đổi `Added: false`. Thẻ biến mất khỏi UI chính. | **PASS** |
| **TC-04** | Chặn xóa thẻ Active | Thẻ `active@test.com` (`IsActive: true`, `[✔ Dùng]`) | Nút `[ 🗑 ]` ẩn hoàn toàn. Gọi `mgr.RemoveAccount` trực tiếp trả về `ErrCannotDeleteActiveAccount`. Gọi UI báo lỗi status bar. | **PASS** |
| **TC-05** | Hủy bỏ Dialog (Dismiss) | Bấm `[ 🗑 ]` trên thẻ `user5@test.com` | Dialog xác nhận hiện lên. Dismiss/Hủy -> Tệp token và thẻ UI được bảo toàn nguyên vẹn. | **PASS** |
| **TC-06** | Thứ tự nút ActionCluster | Kiểm tra cấu trúc phân cấp widget | `Objects[0]` = ActionSlot, `Objects[1]` = RefreshBtn, `Objects[2]` = StatusBtn, `Objects[3]` = DeleteBtn. Đúng 100% Invariant 2. | **PASS** |
| **TC-07** | Dọn dẹp Quota Cache | Quota cache chứa dữ liệu của `user7@test.com` | Sau khi xóa tài khoản -> `a.quotaCache.Get("user7@test.com")` trả về `false`. | **PASS** |
| **TC-08** | Tự động ngắt Auto-Switch khi hết tài khoản | `auto_switch: true`, hệ thống còn duy nhất 1 tài khoản có token | Xóa tài khoản này -> `autoSwitchEnabled` đổi thành `false`, lưu `ui_preferences.json`, status bar hiển thị `⚠️ Không có tài khoản để chuyển.`. | **PASS** |

---

## IV. BẰNG CHỨNG KIỂM TRA TOÀN BỘ REPO & BUILD WINDOWS PE

### 1. Kiểm thử gói trọng tâm:
```powershell
go test -v ./internal/telemetry ./internal/manager ./internal/ui
```
- Kết quả: **PASS 100%** across all packages.

### 2. Kiểm thử toàn bộ Repository:
```powershell
go test -p 1 ./...
```
- Kết quả: **PASS 100%** tất cả các gói trong repo:
  - `cmd/switcher` (PASS)
  - `internal/automation` (PASS)
  - `internal/connector` (PASS)
  - `internal/manager` (PASS)
  - `internal/model` (PASS)
  - `internal/reloader` (PASS)
  - `internal/scanner` (PASS)
  - `internal/store` (PASS)
  - `internal/telemetry` (PASS)
  - `internal/ui` (PASS)
  - `internal/version` (PASS)
  - `internal/wincred` (PASS)

### 3. Build Windows PE:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Clean
```
- Kết quả: **Exit code 0** (BUILD SUCCESSFUL).
- Tệp thực thi: `bin\antigravity-switch.exe` (27.82 MB).
- Xác minh icon PE và tài nguyên nhúng: PASS.

---

## V. KẾT LUẬN & ĐỀ XUẤT HÀNH ĐỘNG

Mã nguồn cho tính năng **Nút Xóa Tài Khoản (Trash Button) & Đồng Bộ Hai Kho Dữ Liệu** đã được hoàn thiện 100% theo đúng kiến trúc phê duyệt tại `SPEC-ARCH-20261023-ACCOUNT-DELETION-AND-TRASH-BUTTON`. Tất cả 6 Supreme Invariants được tuân thủ nghiêm ngặt, không có bất kỳ drift nào ngoài phạm vi Touch-Map.

- **Đánh giá (Verdict):** `PASS / READY_FOR_SUPERVISOR`
