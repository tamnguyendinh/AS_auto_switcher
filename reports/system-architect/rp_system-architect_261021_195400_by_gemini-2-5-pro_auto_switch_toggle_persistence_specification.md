# Báo Cáo Kiến Trúc: Lưu Trữ Bền Vững Trạng Thái Nút Gạt Tự Động Trên Giao Diện
(System Architect Lane Completion Report — Auto-Switch Toggle Persistence Specification)

- **Thời gian lập:** 2026-10-21 19:54:00 (Asia/Bangkok)
- **Lane:** System Architect
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Chỉ đạo trực tiếp và phê duyệt từ Human Owner
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261021-AUTO-SWITCH-TOGGLE-PERSISTENCE`
- **Mã báo cáo:** `rp_system-architect_261021_195400_by_gemini-2-5-pro_auto_switch_toggle_persistence_specification`

---

## I. PHẠM VI CÔNG VIỆC HOÀN THÀNH (SCOPE COMPLETED)

1. **Khảo sát và phân tích hiện trạng:**
   - Điều tra nút gạt "Tự động" (`autoSwitchToggle`) trên thanh công cụ Row 2 của GUI.
   - Xác nhận nút gạt hiện tại chỉ lưu trạng thái trên bộ nhớ RAM (`a.autoSwitchEnabled`), hoàn toàn chưa được kết nối với file lưu trữ cấu hình `logs/ui_preferences.json`.
   - Mỗi lần tắt ứng dụng và mở lại, nút Tự động luôn bị trả về giá trị mặc định ban đầu là `OFF (false)`.
2. **Thảo luận và thống nhất giải pháp cùng Human Owner:**
   - Thống nhất mở rộng `UIPreferences` để đồng bộ cơ chế lưu trữ của nút Tự động tương tự như nút "Ẩn tài khoản" và "Theme".
   - Tự động nạp và kích hoạt trạng thái đã lưu khi app khởi động lại.
   - Bảo toàn chéo giá trị `AutoSwitch` khi đổi Theme hoặc gạt nút Ẩn tài khoản.
3. **Biên soạn văn kiện đặc tả kỹ thuật chi tiết:**
   - Hoàn thành tài liệu [2026-10-21-auto-switch-toggle-persistence-spec.md](file:///g:/antigravity_auto_switch/reports/architect/2026-10-21-auto-switch-toggle-persistence-spec.md) (commit `ca1430c`).
   - Cung cấp step-by-step recipe cho từng file: `internal/ui/themes.go`, `internal/ui/app.go`, và 3 bài test tự động.
4. **Phê chuẩn từ Owner:**
   - Human Owner đã trực tiếp kiểm tra và ban hành lệnh: `"gui ceo"`.

---

## II. DANH MỤC TỆP ĐẦU RA (OUTPUT ARTIFACTS)

1. `reports/architect/2026-10-21-auto-switch-toggle-persistence-spec.md` (Đặc tả kỹ thuật, commit `ca1430c`)
2. `reports/system-architect/rp_system-architect_261021_195400_by_gemini-2-5-pro_auto_switch_toggle_persistence_specification.md` (Báo cáo hoàn thành lane)

---

## III. QUYẾT ĐỊNH KIẾN TRÚC ĐÃ CHỐT (ARCHITECTURAL DECISIONS)

1. **Schema Data:** Thêm `AutoSwitch bool `json:"auto_switch"`` vào `UIPreferences`.
2. **Startup Restoration:** Khôi phục `a.autoSwitchEnabled` từ `prefs.AutoSwitch` khi khởi động (`NewAppWithFyne`) và đồng bộ `a.controller.SetEnabled(true)` nếu trạng thái lưu là `true`.
3. **Non-blocking Persistence:** Lưu đĩa bất đồng bộ trong goroutine khi người dùng thao tác gạt nút tại `SetAutoSwitchEnabled`.
4. **Mutual Preservation:** Giữ nguyên vẹn trường `AutoSwitch` khi người dùng thay đổi Theme hoặc gạt nút Ẩn tài khoản.

---

## IV. VẤN ĐỀ CÒN TỒN ĐỌNG (RESIDUAL OPEN QUESTIONS)

- **Không có (None).** Văn kiện đặc tả đã được Owner phê chuẩn toàn bộ.

---

## V. ĐỀ XUẤT BÀN GIAO (HANDOFF RECOMMENDATION)

- **Verdict:** `READY_FOR_CODER`
- **Handoff Target:** Coder Lane nhận bản đặc tả `reports/architect/2026-10-21-auto-switch-toggle-persistence-spec.md` để triển khai mã nguồn và bộ test tự động tại `internal/ui/`.
