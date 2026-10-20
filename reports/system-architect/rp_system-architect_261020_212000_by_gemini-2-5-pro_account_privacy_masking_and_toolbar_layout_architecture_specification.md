# Báo Cáo Kiến Trúc Hệ Thống: Đặc Tả Toàn Diện Chế Độ Ẩn Thông Tin Tài Khoản & Chuẩn Hóa Phân Cách Thanh Công Cụ Hàng 2 (Account Privacy Masking & Toolbar Layout Specification)

**Mã tài liệu:** `rp_system-architect_261020_212000_by_gemini-2-5-pro_account_privacy_masking_and_toolbar_layout_architecture_specification`  
**Ngày ban hành:** 2026-10-20  
**Làn thực hiện:** System Architect (Kiến trúc sư Hệ thống)  
**Mô hình thực hiện:** `gemini-2-5-pro`  
**Chỉ đạo tối cao:** Human Owner (Thảo luận & Thống nhất trực tiếp)  
**Trạng thái:** HOÀN TẤT & ĐÃ PHÊ DUYỆT BỞI HUMAN OWNER (APPROVED ARCHITECTURAL AUTHORITY)  
**Tài liệu đặc tả đính kèm:** `reports/architect/2026-10-20-account-privacy-masking-toolbar-spec.md`  
**Điểm đối soát mã nguồn:**
- Giao diện Fyne: `internal/ui/window.go`, `internal/ui/app.go`, `internal/ui/widgets.go`, `internal/ui/themes.go`
- Cấu hình lưu trữ: `accounts_store/.ui_preferences.json`

---

## 1. TỔNG QUAN, BỐI CẢNH & NHU CẦU THỰC TẾ

### 1.1 Vấn Đề Thực Tế (Problem Statement)
Trong quá trình vận hành thực tế Antigravity Multi-Account Switcher:
1. **Rủi ro rò rỉ dữ liệu cá nhân khi chia sẻ màn hình / quay video:** Danh sách email Google được hiển thị đầy đủ trên các thẻ card (`AccountCardWidget`). Khi người dùng quay clip hướng dẫn, livestream trên mạng xã hội, hoặc chụp ảnh màn hình để trao đổi công việc, toàn bộ địa chỉ email cá nhân đều bị lộ công khai, tạo nguy cơ bị spam hoặc rủi ro an ninh mạng.
2. **Bố cục Hàng 2 (Row 2 Header) chưa có sự phân tách rõ ràng:** Cụm điều khiển phía bên phải hiện chỉ gồm nhãn `"Tự động:"`, nút gạt `AutoSwitchToggle` và dropdown Theme đặt cạnh nhau. Giao diện thiếu sự mạch lạc, chưa có các vách ngăn trực quan (visual separators) để phân định các nhóm tính năng độc lập.

### 1.2 Yêu Cầu Chỉ Đạo Trực Tiếp Từ Human Owner
Human Owner đã trực tiếp đưa ra yêu cầu cải tiến kiến trúc:
> *"email tren thẻ hien thị tren UI tôi thay can co them 1 chuc năng che bót thong tin emial(thong itn tài khaonr) can co 1 nút canh nút 'Tự động' nhu sau:*  
> *`Ẩn tài khoản: <Nút gạt giống tự động> | Tự động: <Nút gạt> | dropdown theme`*  
> *bo trí và pahn cach nhu the se dẹp và chueyn hgheipj hon, chcu angn này cung giúp app chuyen đoi này nhin an toàn hơn"*

---

## 2. CÁC QUYẾT ĐỊNH KIẾN TRÚC ĐÃ THỐNG NHẤT VỚI HUMAN OWNER

Sau phiên thảo luận chi tiết giữa Human Owner và System Architect, các quyết định kiến trúc cốt lõi đã được xác lập:

| Hạng mục quyết định | Nội dung thống nhất | Lý do & Lợi ích kiến trúc |
|---|---|---|
| **Quy tắc che email (Masking Rule)** | Giữ lại đúng **1 ký tự đầu** của username + `***` + `@` + `tên miền`.<br>Ví dụ: `tamnguyen@gmail.com` $\rightarrow$ `t***@gmail.com`. | Cực kỳ gọn gàng, đồng nhất cho mọi tài khoản bất kể độ dài tên; chống tràn layout thẻ 280x130 px; đơn giản hóa code xử lý chuỗi (KISS). |
| **Hành vi Rê chuột (Tooltip Hover)** | **Giữ nguyên tooltip gốc:** Khi rê chuột vào email đã che, popup tooltip vẫn hiển thị đầy đủ email thật. | Giúp người dùng kiểm tra nhanh email mà không cần tắt chế độ ẩn; không cần viết thêm logic rẽ nhánh tắt/bật tooltip. |
| **Bố cục Hàng 2 (Row 2 Layout)** | `Ẩn tài khoản: [Toggle] │ Tự động: [Toggle] │ [Dropdown Theme]` | 3 nhóm tính năng được phân cách bằng vách ngăn dọc `│` thanh mảnh (1px), cân đối thị giác, chuyên nghiệp. |
| **Kiểu dáng nút gạt (Toggle Widget)** | Tái sử dụng thiết kế Capsule Toggle ($23 \times 12\text{ px}$, track bo góc $6\text{px}$, thumb tròn $8\text{px}$) đồng bộ 100% với nút Tự động. | Nhất quán ngôn ngữ thiết kế (Design Consistency), đổi màu theo bảng màu Theme (`palette.Primary` khi ON, `palette.BorderSubtle` khi OFF). |
| **Lưu cấu hình (Persistence)** | Lưu trường `mask_accounts: bool` vào `accounts_store/.ui_preferences.json`. Mặc định lần đầu cài là `false`. | Ghi nhớ trạng thái của người dùng qua các lần tắt/mở ứng dụng. |
| **Hiệu năng cập nhật (Reactive Performance)** | Gạt nút $\rightarrow$ cập nhật trực tiếp `emailLabel` trên mảng `accountCards` có sẵn trong RAM. | Phản hồi dưới 1 mili-giây, zero jank, không giật màn hình, không làm gián đoạn Quota countdown. |
| **Bảo vệ Thanh trạng thái (Status Bar Protection)** | Tự động che email trong các thông báo chuyển đổi ở Status Bar (`⏳ Đang chuyển sang t***@gmail.com...`). | Chống rò rỉ email thật qua dòng thông báo phía đáy cửa sổ. |

---

## 3. BẢN VẼ BỐ CỤC HÌNH HỌC & ĐẶC TẢ CHI TIẾT (VISUAL SPECIFICATION)

### 3.1 Sơ Đồ Bố Cục Hàng 2 Tổng Thể
```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ [AS] Antigravity Multi-Account Switcher (10pt Bold)                                                      _ |_| X │ ◄── Hàng 1
├──────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ Hoạt động | PID 18660 | Port 62220  [Danh sách...] [Làm mới...] [📌]                                            │
│                                           Ẩn tài khoản: [ ○ ]  │  Tự động: [ ● ]  │  [ ☀️ ▼ ]                    │ ◄── Hàng 2
└──────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
  ◄───────────────── Cụm bên trái (~510px) ─────────────────►   ◄────── Cụm điều khiển phải (~290px) ──────►
```

### 3.2 Đặc Tả Hình Học Cụm Phía Phải (Row 2 Right Cluster Layout)
Cụm được bố trí bằng Custom Layout `row2RightClusterLayout` gồm 5 phần tử nối tiếp nhau theo chiều ngang:

```text
┌───────────────────────┐   ┌───┐   ┌───────────────────┐   ┌───┐   ┌────────────┐
│ Ẩn tài khoản:  [ ○ ]  │   │ │ │   │   Tự động:  [ ● ] │   │ │ │   │  [ ☀️ ▼ ]  │
└───────────────────────┘   └───┘   └───────────────────┘   └───┘   └────────────┘
     (Khối 1: ~105px)      (Div 1)     (Khối 2: ~88px)     (Div 2)  (Khối 3: 58px)
```

1. **Khối 1 — "Ẩn tài khoản:" + Capsule Toggle ($105 \times 20\text{ px}$):**
   - Label: `widget.NewLabelWithStyle("Ẩn tài khoản:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})`.
   - Toggle: `newCapsuleToggle(initial bool, onChange func(bool), palette ThemePalette)` kích thước $23 \times 12\text{ px}$.
2. **Divider 1 — Thanh phân cách dọc 1 ($1 \times 14\text{ px}$):**
   - Đối tượng `canvas.Rectangle`, rộng 1px, cao 14px, màu `palette.BorderSubtle`. Căn giữa theo trục dọc Hàng 2 ($y = (\text{Height} - 14) / 2$).
3. **Khối 2 — "Tự động:" + Capsule Toggle ($88 \times 20\text{ px}$):**
   - Label: `widget.NewLabelWithStyle("Tự động:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})`.
   - Toggle: `AutoSwitchToggle` hiện tại ($23 \times 12\text{ px}$).
4. **Divider 2 — Thanh phân cách dọc 2 ($1 \times 14\text{ px}$):**
   - Cùng thông số với Divider 1: rộng 1px, cao 14px, màu `palette.BorderSubtle`.
5. **Khối 3 — Theme Select Dropdown ($58 \times 20\text{ px}$):**
   - Dropdown gồm 3 icon theme thu gọn tối đa.

---

## 4. CHI TIẾT CƠ CHẾ VẬN HÀNH DỮ LIỆU & LUỒNG SỰ KIỆN (OPERATIONAL MECHANICS)

### 4.1 Thuật Toán Xử Lý Chuỗi Che Email (`maskEmail`)
```text
Hàm: maskEmail(email string) string

1. atIndex = strings.Index(email, "@")
2. Nếu atIndex <= 0:
      Trả về email nguyên bản (phòng thủ an toàn chuỗi dị thường, không bao giờ panic).
3. username = email[:atIndex]
4. domain = email[atIndex:] (chuỗi bắt đầu từ ký tự '@')
5. runes = []rune(username)
6. firstChar = string(runes[0])
7. Trả về: firstChar + "***" + domain
```

**Bảng kết quả mẫu:**
- `tamnguyen123@gmail.com` $\rightarrow$ `t***@gmail.com`
- `dev@antigravity.io` $\rightarrow$ `d***@antigravity.io`
- `a@gmail.com` $\rightarrow$ `a***@gmail.com`
- `admin.test+work@sub.domain.com` $\rightarrow$ `a***@sub.domain.com`

### 4.2 Luồng Vòng Đời Khởi Động & Ghi Nhớ Cấu Hình (Persistence Lifecycle)
```text
[Khởi động NewAppWithFyne()]
         │
         ▼
[Đọc file accounts_store/.ui_preferences.json]
         │
         ├─► Nếu file có "mask_accounts": true  ──► a.maskAccounts = true
         └─► Nếu file chưa có trường này        ──► a.maskAccounts = false (Mặc định)
         │
         ▼
[Dựng giao diện setupLayout()]
         │
         ├─► Khởi tạo maskAccountsToggle ở trạng thái tương ứng.
         └─► Khi tạo các AccountCardWidget ban đầu:
             Nếu a.maskAccounts == true:
                 card.emailLabel hiển thị t***@gmail.com
                 card.emailLabel.fullText = tamnguyen123@gmail.com (Email gốc)
```

### 4.3 Luồng Tương Tác Bấm Gạt Nút (In-Place Reactive Update)
```text
[Người dùng click nút gạt Ẩn tài khoản]
                    │
                    ▼
1. Toggle kích hoạt animation nội bộ:
   - Thumb trượt sang vị trí mới (ON: x=13, OFF: x=2).
   - Track đổi màu (ON: palette.Primary, OFF: palette.BorderSubtle).
                    │
                    ▼
2. Gọi Callback a.SetMaskAccounts(isOn):
   ├─ a.mu.Lock(): a.maskAccounts = isOn.
   ├─ Ghi đĩa không đồng bộ: Ghi mask_accounts vào .ui_preferences.json.
   └─ In-place RAM Loop (Zero Layout Jank):
      Duyệt mảng a.accountCards:
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
3. Thông báo phản hồi nhẹ trên Status Bar (tự ẩn sau 2 giây):
   - Khi BẬT: "🔒 Đã ẩn thông tin tài khoản"
   - Khi TẮT: "🔓 Đã hiển thị đầy đủ tài khoản"
```

### 4.4 Phân Định Ranh Giới Giữa View Filter và Backend Invariant
> **NGUYÊN TẮC BẤT BIẾN TỐI CAO:**  
> Chế độ ẩn tài khoản là một **Presentation View Filter đơn thuần**.  
> Tất cả các thành phần Backend, Quota Client, Hot-swap Switcher, Chrome Cookie Sync, Windows Credential Vault và Automation Controller **100% TIẾP TỤC DÙNG EMAIL THẬT NGUYÊN BẢN**. Không có bất kỳ sự thay đổi nào đối với model dữ liệu hay logic chuyển đổi tài khoản bên dưới.

---

## 5. MA TRẬN BẢO TOÀN INVARIANT (SYSTEM INVARIANTS)

1. **Khung Kích Thước Bất Biến (Fixed Dimensions):**
   - Kích thước cửa sổ: Cố định $920 \times 630\text{ px}$.
   - Kích thước thẻ Card: Cố định $280 \times 130\text{ px}$.
   - Kích thước bảng Quota: Cố định $228 \times 77\text{ px}$.
   - Không gây co giãn cửa sổ khi gạt nút bật/tắt ẩn tài khoản.
2. **Đồng Bộ Đa Theme (Theme Synchronization):**
   - Khi người dùng đổi theme (`🌙`, `☀️`, `☕`), cả hai nút gạt Capsule Toggle và hai thanh phân cách `│` phải được cập nhật màu sắc ăn khớp theo `palette.Primary` và `palette.BorderSubtle`.
3. **An Toàn Đa Luồng (Thread Safety):**
   - Trạng thái `a.maskAccounts` được đồng bộ qua `sync.RWMutex` (`a.mu.RLock()` / `a.mu.Lock()`).

---

## 6. KẾ HOẠCH BÀN GIAO & PHÂN RÃ CÔNG VIỆC TIẾP THEO

Tài liệu này là căn cứ tối cao đã được Human Owner phê duyệt. Bàn giao sang CEO và các làn chuyên trách tiếp theo:
1. **Làn Planner:** Lập kế hoạch thực thi chi tiết (Execution Plan) dựa trên tài liệu đặc tả này, phân rã công việc giao cho Coder.
2. **Làn Coder:** Triển khai các thành phần UI:
   - Bổ sung `maskEmail()` và hàm `SetMasked()` trong `internal/ui/widgets.go`.
   - Cập nhật `setupLayout()` và `row2RightClusterLayout` trong `internal/ui/window.go`.
   - Cập nhật `App.SetMaskAccounts()` và lưu `UIPreferences.MaskAccounts` trong `internal/ui/app.go`, `internal/ui/themes.go`.
   - Cập nhật hiển thị Status Bar khi chuyển tài khoản.
3. **Làn QA & Supervisor:** Kiểm thử tự động trên Fyne headless/native frame:
   - Kiểm tra hiển thị masked email `t***@gmail.com`.
   - Kiểm tra tooltip hover hiển thị đúng email thật.
   - Kiểm tra ghi nhớ trạng thái trên file `.ui_preferences.json`.
   - Kiểm tra không xảy ra layout jank hay vỡ khung viền.
