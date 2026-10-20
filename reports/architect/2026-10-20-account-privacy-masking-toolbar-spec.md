# Đặc Tả Kiến Trúc & Kỹ Thuật: Chế Độ Ẩn Tài Khoản & Chuẩn Hóa Phân Cách Thanh Công Cụ Hàng 2 (Account Privacy Masking & Toolbar Layout Specification)

- **Ngày lập:** 2026-10-20
- **Vai trò:** System Architect (Kiến trúc sư Hệ thống)
- **Thẩm quyền:** Chỉ đạo trực tiếp và đã được phê duyệt từ Human Owner
- **Mã văn kiện:** `SPEC-ARCH-20261020-ACCOUNT-PRIVACY-MASKING-TOOLBAR`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

1. **PHÂN TÁCH TRIỆT ĐỂ PRESENTATION FILTER VÀ BACKEND MODEL (VIEW-ONLY FILTER):**
   - Chế độ "Ẩn tài khoản" (Account Privacy / Masking Mode) **CHỈ LÀ BỘ LỌC HIỂN THỊ Ở TẦNG VIEW**.
   - Tuyệt đối không thay đổi kiểu dữ liệu `Email string` trong Model, Account Store, Quota Cache, Chrome Scanner, Telemetry Loop hay Hot-swap Switcher.
   - Mọi hoạt động nghiệp vụ ngầm (quét Chrome, đổi session, fetch quota, gọi RPC Antigravity) **100% TIẾP TỤC SỬ DỤNG EMAIL THẬT NGUYÊN BẢN**.
2. **QUY TẮC CHE EMAIL GỌN GÀNG TỐI GIẢN (KISS MASKING RULE):**
   - Chỉ giữ lại đúng **1 ký tự đầu** của username + `***` + `@` + `tên miền`.
   - Ví dụ: `tamnguyen@gmail.com` $\rightarrow$ `t***@gmail.com`.
   - Giữ nguyên tooltip hover: khi rê chuột vào email đã che, popup tooltip vẫn hiển thị email thật nguyên bản.
3. **BẢO TỒN NGUYÊN VẸN CÁC THÔNG SỐ HÌNH HỌC ĐÃ KHÓA CỨNG:**
   - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`** (Compact Mode).
   - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** trong `newSymmetricGridWrap`.
   - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
   - Cập nhật In-place reactive (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. BỐ CỤC THỊ GIÁC & QUY CÁCH HÌNH HỌC (ROW 2 RIGHT CLUSTER)

### 1.1 Bản Vẽ Bố Cục Tổng Thể Hàng 2
```text
┌───────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Hoạt động | PID 18660 | Port 62220  [Danh sách...] [Làm mới...] [📌]                              │
│                                           Ẩn tài khoản: [ ○ ]  │  Tự động: [ ● ]  │  [ ☀️ ▼ ]     │
└───────────────────────────────────────────────────────────────────────────────────────────────────┘
  ◄───────────────── Cụm bên trái (~510px) ─────────────────►   ◄───── Cụm điều khiển phải (~290px) ─────►
```

### 1.2 Chi Tiết Cụm Điều Khiển Phía Phải (Row 2 Right Cluster)
Dàn hàng ngang gồm 5 phần tử:

```text
┌─────────────────────┐   ┌───┐   ┌─────────────────┐   ┌───┐   ┌────────────┐
│ Ẩn tài khoản: [ ○ ] │   │ │ │   │ Tự động:  [ ● ] │   │ │ │   │  [ ☀️ ▼ ]  │
└─────────────────────┘   └───┘   └─────────────────┘   └───┘   └────────────┘
     (Khối 1: ~105px)    (Div 1)   (Khối 2: ~88px)     (Div 2)  (Khối 3: 58px)
```

1. **Khối 1 — Điều khiển Ẩn tài khoản:**
   - Label: `"Ẩn tài khoản:"` (Font 10pt Bold, căn lề thẳng hàng với label `"Tự động:"`).
   - Capsule Toggle: Kích thước cố định $23 \times 12\text{ px}$.
     - *Trạng thái OFF (Tắt):* Nền ray màu xám mờ (`palette.BorderSubtle`), nút tròn nằm bên trái ($x=2, y=2$).
     - *Trạng thái ON (Bật):* Nền ray màu xanh thương hiệu (`palette.Primary`), nút tròn nằm bên phải ($x=13, y=2$).
2. **Divider 1 — Thanh phân cách dọc 1:**
   - Đối tượng đồ họa `canvas.Rectangle`, rộng $1\text{ px}$, cao $14\text{ px}$, màu `palette.BorderSubtle` ăn theo theme đang kích hoạt.
3. **Khối 2 — Điều khiển Tự động đổi tài khoản:**
   - Label: `"Tự động:"` (Font 10pt Bold).
   - Capsule Toggle: Kích thước cố định $23 \times 12\text{ px}$ (sử dụng `AutoSwitchToggle` hiện có).
4. **Divider 2 — Thanh phân cách dọc 2:**
   - Cùng thông số với Divider 1: rộng $1\text{ px}$, cao $14\text{ px}$, màu `palette.BorderSubtle`.
5. **Khối 3 — Theme Select Dropdown:**
   - Kích thước cố định $58 \times 20\text{ px}$ gồm 3 icon (`🌙`, `☀️`, `☕`).

---

## 2. QUY TẮC CHE EMAIL & XỬ LÝ CHUỖI (MASKING SPECIFICATION)

### 2.1 Thuật toán Che Email (`maskEmail(email string) string`)
```text
Đầu vào: Email chuỗi (e.g. "tamnguyen@gmail.com")
Bước 1: Tìm vị trí ký tự '@'.
        Nếu không tìm thấy hoặc ở vị trí đầu tiên (atIdx <= 0):
        -> Trả về email nguyên bản (phòng thủ chuỗi bất thường).
Bước 2: Tách username = email[:atIdx], domain = email[atIdx:] (bao gồm cả ký tự '@').
Bước 3: Lấy ký tự Rune đầu tiên của username (an toàn với Unicode UTF-8).
Bước 4: Kết quả = string(firstRune) + "***" + domain.
```

### 2.2 Bảng Kiểm Thử Xử Lý Chuỗi (Test Matrix)
| Đầu vào Email | Đầu ra khi `mask_accounts = true` | Đầu ra khi `mask_accounts = false` | Ghi chú |
|---|---|---|---|
| `tamnguyen@gmail.com` | `t***@gmail.com` | `tamnguyen@gmail.com` | Dạng chuẩn thông dụng |
| `developer.ai@antigravity.io` | `d***@antigravity.io` | `developer.ai@antigravity.io` | Dạng có dấu chấm |
| `a@gmail.com` | `a***@gmail.com` | `a@gmail.com` | Username chỉ có 1 ký tự |
| `admin+tag@sub.domain.com` | `a***@sub.domain.com` | `admin+tag@sub.domain.com` | Domain đa cấp, có alias |
| `invalid-email-string` | `invalid-email-string` | `invalid-email-string` | Chuỗi không hợp lệ (không crash) |
| `""` (Rỗng) | `""` | `""` | Rỗng an toàn |

---

## 3. CƠ CHẾ VẬN HÀNH & LUỒNG DỮ LIỆU (OPERATIONAL MECHANICS)

### 3.1 Khởi Động & Lưu Cấu Hình (Persistence Lifecycle)
- **Vị trí lưu trữ:** `accounts_store/.ui_preferences.json`
- **Mô hình JSON:**
  ```json
  {
    "theme": "dark",
    "mask_accounts": true
  }
  ```
- **Quy tắc nạp:**
  - Khi khởi động `NewAppWithFyne()` $\rightarrow$ đọc `LoadUIPreferences()`.
  - Nếu `mask_accounts` là `true` $\rightarrow$ gán `a.maskAccounts = true`.
  - Mặc định khi chưa có file cấu hình: `mask_accounts = false`.
- **Quy tắc ghi:** Khi người dùng click toggle $\rightarrow$ ghi bất đồng bộ ra đĩa để không chặn main UI goroutine.

### 3.2 Luồng Click Toggle & Phản Ứng Tức Thì (In-Place Reactive Update)
```text
[Người dùng click nút gạt Ẩn tài khoản]
                   │
                   ▼
1. Toggle UI đổi vị trí thumb và màu nền ray (Animation mượt mà).
                   │
                   ▼
2. Kích hoạt Callback: a.SetMaskAccounts(isOn)
   ├─ a.mu.Lock(): Cập nhật a.maskAccounts = isOn.
   ├─ Ghi đĩa non-blocking: SaveUIPreferences().
   └─ Duyệt mảng in-place cards trong RAM (Zero Layout Jank):
      for _, card := range a.accountCards {
          if card != nil && card.emailLabel != nil {
              if isOn {
                  card.emailLabel.SetText(maskEmail(card.Email))
              } else {
                  card.emailLabel.SetText(card.Email)
              }
              card.emailLabel.Refresh()
          }
      }
                   │
                   ▼
3. Phản hồi trạng thái lên Status Bar (trong 2 giây):
   - Nếu BẬT: "🔒 Đã ẩn thông tin tài khoản"
   - Nếu TẮT: "🔓 Đã hiển thị đầy đủ tài khoản"
```

### 3.3 Hành Vi Tooltip Hover Khi Rê Chuột
- `emailLabelWithTooltip` lưu trữ thuộc tính `fullText: string` cố định luôn là **email thật**.
- Khi `mask_accounts = true`, nhãn hiển thị là `t***@gmail.com`.
- Khi người dùng rê chuột vào (`MouseIn`), popup tooltip bung ra đọc từ `fullText` $\rightarrow$ **hiển thị đầy đủ email thật**.
- Khi rời chuột (`MouseOut`), popup lập tức biến mất.

### 3.4 Bảo Vệ Thanh Trạng Thái (Status Bar Protection)
- Trong các luồng `SwitchAccount()`, `handleAutomationStatus()`, khi hiển thị thông báo chuyển đổi:
  - Nếu `a.maskAccounts == true`: Tự động thay thế địa chỉ email bằng `maskEmail(targetEmail)`.
  - *Ví dụ:* `⏳ Đang chuyển sang t***@gmail.com...` và `✅ Đã chuyển sang t***@gmail.com`.

---

## 4. MA TRẬN BẢO TOÀN INVARIANT (SYSTEM INVARIANTS)

1. **Zero Layout Jank:** Việc gạt bật/tắt chế độ ẩn tuyệt đối không được gọi `forceRebuildCards()` hay xóa/thêm đối tượng Canvas trong `accountListContainer`. Chỉ thực hiện set text trên các widget `emailLabel` đã có sẵn.
2. **Theme Synchronization:** Khi người dùng đổi theme (`🌙`, `☀️`, `☕`), cả 2 Capsule Toggle và 2 thanh phân cách `|` phải được cập nhật màu viền/màu nền ăn theo `ThemePalette` mới thông qua hàm `SetPalette()`.
3. **Thread Safety:** Truy cập trạng thái `maskAccounts` trong `App` phải được bảo vệ bằng `a.mu.RLock()` / `a.mu.Lock()`.
