# Đặc Tả Kiến Trúc: Hệ Thống Lưu Trữ 2 Tầng (Two-Store) & Chuẩn Hóa 3 Trạng Thái Thẻ Giao Diện
(Two-Store Architecture & UI 3-State Card Specification & Execution Blueprint)

- **Ngày lập:** 2026-10-21
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Chỉ đạo kỹ thuật trực tiếp từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261021-TWO-STORE-AND-UI-3-STATE`
- **Trạng thái:** DỰ THẢO BÁO CÁO CHI TIẾT TRÌNH OWNER DUYỆT (DETAILED PROPOSAL FOR OWNER REVIEW)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY INVARIANTS)

1. **PHÂN TÁCH TRIỆT ĐỂ RANH GIỚI 2 THƯ MỤC DỮ LIỆU (TWO-STORE SEPARATION):**
   - **`discovered_accounts/`:** Là danh bạ nhận diện danh tính quét được từ Google Chrome trên máy tính. Tài khoản ở đây là **CHƯA OAUTH ("chỉ nằm đó")**. Tuyệt đối không chứa `token_data`.
   - **`accounts_store/`:** Là kho chứng chỉ bảo mật (Token Vault) của Antigravity. Tài khoản ở đây là **ĐÃ OAUTH THẬT VỚI GOOGLE ("mới dùng được")**. Chỉ những tài khoản đã thực sự đăng nhập vào Antigravity qua Chrome mới được phép xuất hiện ở thư mục này.
   - Cả 2 thư mục đều bắt buộc phải nằm trong `.gitignore` để bảo mật 100% dữ liệu người dùng.
2. **TRIỆT TIÊU LỖI POPUP TRÊN GIAO DIỆN (ZERO-ERROR UI CONTRACT):**
   - Tuyệt đối cấm hiển thị nút `[Chuyển]` cho các tài khoản chưa có token trong `accounts_store/`.
   - Tài khoản chưa OAuth bắt buộc phải hiển thị nhãn **`[Cần OAuth]`** và vô hiệu hóa chuyển đổi để loại bỏ hoàn toàn lỗi văng popup `target account not found in store`.
3. **NGUYÊN TẮC TINH GIẢN TỐI ĐA (KISS JSON STORAGE):**
   - File JSON trong `discovered_accounts/` chỉ lưu đúng 2 trường: `email` và `scanned_at`. Loại bỏ hoàn toàn các trường thừa không dùng (`name`, `profile_dir`, `avatar_url`).
4. **BẢO TỒN KÍCH THƯỚC HÌNH HỌC ĐÃ KHÓA CỨNG:**
   - Cửa sổ ứng dụng: `920 x 630 px`.
   - Thẻ Card: `280 x 130 px` trong `newSymmetricGridWrap`. Cập nhật In-place reactive (Zero Jank).

---

## PHẦN I: KIẾN TRÚC LƯU TRỮ 2 THƯ MỤC (TWO-STORE ARCHITECTURE)

```text
 ┌────────────────────────────────────────────────────────────────────────────────────────┐
 │ 1. THƯ MỤC `discovered_accounts/` (Danh bạ Chrome — Nằm trong .gitignore)             │
 │ • Nguồn gốc: Quét từ `Local State` và thư mục Profile của Google Chrome.               │
 │ • Cấu trúc tệp `discovered_accounts/<email>.json`:                                    │
 │   {                                                                                    │
 │     "email": "developer@gmail.com",                                                    │
 │     "scanned_at": 1729500000                                                           │
 │   }                                                                                    │
 │ • Vai trò: Nhận diện danh tính tài khoản trên máy. Không có quyền chạy Antigravity.    │
 └───────────────────────────────────────────┬────────────────────────────────────────────┘
                                             │
                                             ▼ (Người dùng đăng nhập trên Antigravity qua Chrome)
                                    [OAUTH THẬT VỚI GOOGLE]
                                             │
                                             ▼ (Tự động bắt phiên từ Windows Credential)
 ┌────────────────────────────────────────────────────────────────────────────────────────┐
 │ 2. THƯ MỤC `accounts_store/` (Kho Chìa khóa Token Antigravity — Nằm trong .gitignore) │
 │ • Nguồn gốc: Bắt phiên từ Windows Credential Manager (`target=gemini:antigravity`).   │
 │ • Cấu trúc tệp `accounts_store/<email>.json`:                                          │
 │   {                                                                                    │
 │     "email": "developer@gmail.com",                                                    │
 │     "saved_at": 1729500000,                                                            │
 │     "token_data": {                                                                    │
 │       "access_token": "ya29...",                                                       │
 │       "refresh_token": "1//04...",                                                     │
 │       "id_token": "eyJhbG...",                                                         │
 │       "expiry": "2026-10-21T10:30:00Z"                                                 │
 │     }                                                                                  │
 │   }                                                                                    │
 │ • Vai trò: Cung cấp token để đo Quota telemetry và thực hiện Zero-Chrome Hot-swap.     │
 └────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## PHẦN II: QUY CÁCH HIỂN THỊ THẺ CARD GIAO DIỆN (UI 3-STATE SPECIFICATION)

Cụm điều khiển góc phải phía trên của Thẻ Card (hàng 1, cạnh nút làm mới `[🔄]`) hiển thị nghiêm ngặt theo 3 trạng thái phân nhánh:

| Trạng thái | Điều kiện logic | Hiển thị trên Thẻ Card | Màu sắc / Kiểu dáng | Hành vi tương tác |
|---|---|:---:|---|---|
| **1. Đang dùng** | `item.IsActive == true` | **`[✔ Dùng]`** | Font 10pt Bold, màu Primary (Xanh lá thương hiệu) | Đang là phiên active trong Antigravity. Không cần nút chuyển. |
| **2. Đã OAuth (Sẵn sàng)** | `!item.IsActive && item.HasToken == true` | **`[Chuyển]`** | Nút bấm chữ nhật màu Primary (`newSwitchTextButton`) | Bấm vào thực hiện Zero-Chrome Hot-swap tức thì (< 1.5s). Quota hiển thị đầy đủ. |
| **3. Chưa OAuth** | `!item.IsActive && item.HasToken == false` | **`[Cần OAuth]`** | Nhãn chữ đậm Font 10pt Bold, màu Hổ phách/Xám mờ | **KHÔNG HIỂN THỊ NÚT CHUYỂN**. Người dùng biết cần đăng nhập Antigravity trước. Quota hiện `N/A`. |

---

## PHẦN III: CHỈ DẪN THỰC THI CHI TIẾT TỪNG TỆP (IMPLEMENTATION BLUEPRINT)

### 1. File: `internal/store/discovered_store.go` (Tạo mới)
Tạo package quản lý riêng cho thư mục `discovered_accounts/`:
- Định nghĩa cấu trúc `DiscoveredAccountEnvelope`:
  ```go
  type DiscoveredAccountEnvelope struct {
      Email     string  `json:"email"`
      ScannedAt float64 `json:"scanned_at"`
  }
  ```
- Hàm `NewDiscoveredStore(dir string)`: Khởi tạo và tự động `os.MkdirAll(dir, 0755)`.
- Hàm `SaveDiscovered(email string) error`: Ghi file nguyên tử (Atomic write qua temp file + rename) lưu `discovered_accounts/<email>.json`.
- Hàm `ListDiscovered() ([]string, error)`: Đọc toàn bộ danh sách email đã quét trong `discovered_accounts/`.

---

### 2. File: `internal/manager/account_manager.go` (Sửa đổi)
- Bổ sung trường `discoveredStore *store.DiscoveredStore` vào `AccountManager`.
- Trong hàm `GetAccountStates(ctx context.Context)`:
  1. Giữ nguyên lệnh proactive capture phiên active: `_, _ = m.CaptureCurrentSession()`.
  2. Quét Chrome: `chromeAccounts, _ := scanner.ScanChromeAccountDetails(cDir)`.
  3. **ĐỒNG BỘ VÀO DISCOVERED STORE:** Với mỗi tài khoản trong `chromeAccounts`, tự động gọi `m.discoveredStore.SaveDiscovered(ca.Email)`.
  4. Nạp danh sách đã OAuth: `storedEmails, _ := m.store.ListAccounts()`.
  5. Đánh dấu trạng thái tài khoản:
     - Nếu `norm == activeEmail`: `IsActive = true`, `State = model.StateActive`, `DisplayBadge = "[✔ Dùng]"`.
     - Nếu `storedSet[norm]`: `HasToken = true`, `State = model.StateReady`, `DisplayBadge = "[Chuyển]"`.
     - Ngược lại: `HasToken = false`, `State = model.StateUnauth`, `DisplayBadge = "[Cần OAuth]"`.

---

### 3. File: `internal/ui/widgets.go` (Sửa đổi dứt điểm lỗi UI)
- Sửa hàm `NewAccountCardWidgetWithQuota`:
  - **Xóa bỏ tham số thừa:** Bỏ tham số `onSave func()` khỏi chữ ký hàm.
  - **Sửa logic vẽ Cụm Action Cluster tại dòng 348:**
    ```go
    var actionCluster fyne.CanvasObject
    if item.IsActive {
        activeBadge := widget.NewLabelWithStyle("[✔ Dùng]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
        actionCluster = container.NewHBox(activeBadge, compactRefresh)
    } else if item.HasToken {
        switchBtn := newSwitchTextButton(func() {
            if onSwitch != nil {
                onSwitch(item.Email)
            }
        }, palette)
        actionCluster = container.NewHBox(switchBtn, compactRefresh)
    } else {
        needOAuthBadge := widget.NewLabelWithStyle("[Cần OAuth]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
        actionCluster = container.NewHBox(needOAuthBadge, compactRefresh)
    }
    ```

---

### 4. File: `internal/ui/app.go` & `internal/ui/tray.go` (Dọn dẹp Dead Code)
- **Trong `app.go`:**
  - Xóa bỏ hoàn toàn hàm `SaveSession()` (dòng 761 - 810).
  - Cập nhật các lời gọi `NewAccountCardWidgetWithQuota` bỏ tham số `a.SaveSession`.
- **Trong `tray.go`:**
  - Xóa bỏ menu item `"Lưu phiên hiện tại"` (dòng 31).

---

### 5. File: `internal/store/sanitizer.go` (Chuẩn hóa Đường dẫn Portable)
- Chuẩn hóa hàm `GetDefaultStoreDir()` và hàm lấy đường dẫn `discovered_accounts/`:
  - Luôn luôn ưu tiên xác định vị trí file thực thi:
    ```go
    if exePath, err := os.Executable(); err == nil {
        exeDir := filepath.Dir(exePath)
        return filepath.Join(exeDir, storeName)
    }
    ```
  - Xóa bỏ triệt để đường dẫn máy cá nhân hardcode: `defaultWorkspaceStore := 'G:\antigravity_auto_switch\accounts_store'`.
  - Đảm bảo app chạy ở chế độ Portable an toàn trên mọi máy tính, không bao giờ văng file ra Desktop hay System32.

---

### 6. File: `.gitignore` (Cập nhật)
Bổ sung thư mục mới vào danh sách loại trừ Git:
```gitignore
accounts_store/
discovered_accounts/
```

---

## PHẦN IV: TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

Coder chỉ được tuyên bố hoàn thành khi đạt 100% các tiêu chí sau:
1. **Nghiệm thu Lưu trữ:**
   - Bấm "Danh sách tài khoản": Thư mục `discovered_accounts/` tự động được tạo và ghi nhận đầy đủ các file JSON (chỉ chứa `email` và `scanned_at`) của mọi tài khoản Chrome trên máy.
   - Thư mục `accounts_store/` chỉ chứa đúng file JSON của tài khoản đã từng đăng nhập Antigravity (có `token_data`).
2. **Nghiệm thu Giao diện (Zero-Error):**
   - Tài khoản đang đăng nhập trong Antigravity: Hiện **`[✔ Dùng]`**.
   - Tài khoản đã từng đăng nhập Antigravity (đã lưu token): Hiện nút **`[Chuyển]`**, click đúp chuyển đổi ngầm trơn tru.
   - Tài khoản mới từ Chrome (chưa từng đăng nhập Antigravity): Hiện nhãn **`[Cần OAuth]`**, hoàn toàn KHÔNG có nút Chuyển, không thể click gây lỗi popup.
3. **Nghiệm thu Dead Code & Test:**
   - Xóa sạch `onSave` và `SaveSession()`, không còn cảnh báo biến thừa.
   - Toàn bộ test suite `go test ./...` chạy pass 100%.

---

Văn bản đặc tả này đã được lưu chính thức tại kho lưu trữ phục vụ cho quá trình thực thi tiếp theo.