# Báo Cáo Kiến Trúc: Lưu Trữ Bền Vững Trạng Thái Nút Ghim Cửa Sổ Trên Giao Diện
(System Architect Lane Completion Report — Window Always-on-Top Pin Persistence Specification)

- **Thời gian lập:** 2026-10-21 20:35:00 (Asia/Bangkok)
- **Lane:** System Architect
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Chỉ đạo trực tiếp và phê chuẩn từ Human Owner
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261021-ALWAYS-ON-TOP-PERSISTENCE`
- **Mã báo cáo:** `rp_system-architect_261021_203500_by_gemini-2-5-pro_always_on_top_persistence_specification`

---

## I. PHẠM VI CÔNG VIỆC HOÀN THÀNH (SCOPE COMPLETED)

1. **Khảo sát và xác minh hiện trạng kỹ thuật:**
   - Điều tra nút ghim cửa sổ **"📌"** (`alwaysOnTopBtn`) trên thanh công cụ Row 2 của GUI.
   - Xác nhận trạng thái `alwaysOnTop` hiện tại chỉ tồn tại tạm thời trên bộ nhớ RAM (`a.alwaysOnTop`), chưa được liên kết với tệp lưu trữ tùy chọn giao diện `logs/ui_preferences.json`.
   - Mỗi lần khởi động lại ứng dụng, `NewAppWithFyne` luôn gán cứng `alwaysOnTop: true`, khiến người dùng dù đã chủ động bỏ ghim cửa sổ thì lần sau mở app vẫn bị ghim trở lại.

2. **Hoàn thiện giải pháp kiến trúc tối ưu cùng Human Owner:**
   - Đồng bộ cơ chế lưu trữ của nút Ghim 📌 vào `logs/ui_preferences.json` cùng với 3 tùy chọn sẵn có (`theme`, `mask_accounts`, `auto_switch`), hình thành hệ thống **4 tùy chọn UI thống nhất**.
   - **Xử lý triệt để bài toán tương thích ngược và giá trị mặc định:** Sử dụng con trỏ `*bool` (`AlwaysOnTop *bool \`json:"always_on_top,omitempty"\``) trong Go struct. Đối với tệp cấu hình cũ chưa có trường này hoặc tệp chưa tồn tại, giá trị mặc định luôn là `true` (ghim cửa sổ chuẩn mực). Khi người dùng chủ động bỏ ghim, giá trị `false` được lưu và khôi phục chính xác 100%.
   - **Bảo toàn chéo 4 chiều (Cross-Setting Mutual Preservation):** Đảm bảo việc thay đổi bất kỳ tùy chọn nào trong 4 tùy chọn (`Theme`, `MaskAccounts`, `AutoSwitch`, `AlwaysOnTop`) đều không làm mất hoặc ghi đè sai lệch 3 tùy chọn còn lại.
   - **Ghi đĩa non-blocking:** Toàn bộ quá trình ghi đĩa được chạy bất đồng bộ trong goroutine, cam kết không gây giật lag giao diện (Zero UI Jank).

3. **Biên soạn văn kiện đặc tả kỹ thuật chi tiết:**
   - Đã biên soạn và commit tài liệu đặc tả hoàn chỉnh: [2026-10-21-always-on-top-persistence-spec.md](file:///g:/antigravity_auto_switch/reports/architect/2026-10-21-always-on-top-persistence-spec.md) (commit `b909f06`).
   - Cung cấp step-by-step recipe chi tiết cho từng tệp nguồn (`internal/ui/themes.go`, `internal/ui/app.go`, `internal/ui/window.go`), cùng 4 bài test tự động bao quát toàn bộ các kịch bản round-trip, startup restoration, và cross-setting preservation.

---

## II. DANH MỤC TỆP ĐẦU RA (OUTPUT ARTIFACTS)

1. `reports/architect/2026-10-21-always-on-top-persistence-spec.md` (Đặc tả kỹ thuật chi tiết, commit `b909f06`)
2. `reports/system-architect/rp_system-architect_261021_203500_by_gemini-2-5-pro_always_on_top_persistence_specification.md` (Báo cáo hoàn thành lane)

---

## III. QUYẾT ĐỊNH KIẾN TRÚC ĐÃ CHỐT (ARCHITECTURAL DECISIONS)

1. **Schema Data:** Thêm `AlwaysOnTop *bool \`json:"always_on_top,omitempty"\`` vào `UIPreferences`.
2. **Canonical Default:** Xử lý `nil` thành `true` trong `LoadUIPreferences` để bảo toàn giá trị mặc định cho các phiên bản cũ và tệp tạo mới.
3. **Startup Restoration:** Khôi phục `a.alwaysOnTop` từ `prefs.AlwaysOnTop` trong `NewAppWithFyne`, đồng bộ `alwaysOnTopBtn.Importance` (`HighImportance` khi `true`, `MediumImportance` khi `false`), và đồng bộ gọi `setWindowAlwaysOnTop(windowTitle, a.alwaysOnTop)`.
4. **Non-blocking Persistence:** Lưu đĩa bất đồng bộ trong goroutine khi người dùng bấm nút ghim trong `ToggleAlwaysOnTop` và `SetAlwaysOnTop`.
5. **Cross-Setting Mutual Preservation:** Giữ nguyên vẹn trường `AlwaysOnTop` trong cả `SetTheme`, `SetMaskAccounts`, và `SetAutoSwitchEnabled`.

---

## IV. VẤN ĐỀ CÒN TỒN ĐỌNG (RESIDUAL OPEN QUESTIONS)

- **Không có (None).** Văn kiện đặc tả đã giải quyết triệt để mọi trường hợp biên, không để lại bất kỳ nhánh rẽ mập mờ nào.

---

## V. ĐỀ XUẤT BÀN GIAO (HANDOFF RECOMMENDATION)

- **Verdict:** `READY_FOR_CODER`
- **Handoff Target:** Đã sẵn sàng chuyển giao cho Coder Lane triển khai mã nguồn và kiểm thử tự động ngay khi Human Owner ban hành lệnh `gửi CEO`.
