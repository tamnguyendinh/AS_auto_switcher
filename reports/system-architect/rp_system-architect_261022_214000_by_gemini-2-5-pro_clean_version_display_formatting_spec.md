# Báo Cáo Kiến Trúc: Chuẩn Hóa Định Dạng Hiển Thị Phiên Bản Trên Giao Diện (Clean Version Display)
(System Architect Lane Completion Report — Root Cause Analysis & Clean Version Display Specification)

- **Thời gian lập:** 2026-10-22 21:40:00 (Asia/Bangkok)
- **Lane:** System Architect (Nhánh Chiến Lược)
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Tiếp nhận trực tiếp ảnh chụp lỗi `media_1792679112437_0b1f0128.png` từ Human Owner
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261022-CLEAN-VERSION-DISPLAY-SPEC`
- **Tệp đặc tả kỹ thuật:** [`reports/architect/2026-10-22-clean-version-display-formatting-spec.md`](file:///g:/antigravity_auto_switch/reports/architect/2026-10-22-clean-version-display-formatting-spec.md)
- **Mã báo cáo:** `rp_system-architect_261022_214000_by_gemini-2-5-pro_clean_version_display_formatting_spec`

---

## I. KẾT QUẢ QUAN SÁT VÀ PHÂN TÍCH ẢNH BÁO LỖI TỪ HUMAN OWNER

1. **Vị trí và nội dung khoanh vùng:**
   - Trong ảnh `media_1792679112437_0b1f0128.png`, Human Owner đã dùng mũi tên đỏ chỉ trực diện và đóng khung đỏ quanh cụm văn bản ở mép phải Footer Status Bar:
     `Version: 1.0.2.0 (rev 3dd7d69)`
   - Vấn đề được xác định chính xác: **Rò rỉ mã Git commit nội bộ `(rev 3dd7d69)` ra giao diện người dùng chính.**

2. **Bác bỏ nhận định sai lệch trước đó:**
   - Lỗi này **KHÔNG LIÊN QUAN** đến lề, padding hay vị trí căn chỉnh mép phải (layout container đã đúng chuẩn).
   - Vấn đề nằm 100% ở định dạng chuỗi: Ứng dụng Desktop thương mại chuyên nghiệp không bao giờ hiển thị đuôi `(rev ...)` ra UI của người dùng. Giao diện chỉ được phép hiển thị gọn gàng, chuẩn mực: **`Version: 1.0.2.0`**.

---

## II. NGUYÊN NHÂN KỸ THUẬT GỐC RỄ (ROOT CAUSE)

- **`internal/version/version.go`**: Hàm `GetFullVersion()` tự động nối `(rev %s)` khi `GitCommit != ""` (`fmt.Sprintf("Version: %s (rev %s)", Version, GitCommit)`).
- **`scripts/build_windows.ps1`**: Khi build binary, script inject git commit hash qua cờ `-ldflags`: `-X 'antigravity-switch/internal/version.GitCommit=$GitCommit'`. Vì vậy `GitCommit` luôn mang giá trị hash (ở commit gần nhất là `3dd7d69`).
- **`internal/ui/window.go:247`**: Nhãn `a.versionLabel` gọi trực tiếp `version.GetFullVersion()`, khiến giao diện hiển thị đuôi `(rev 3dd7d69)`.
- **Hệ thống Test**: `TestVersion_WithGitCommit` trong `internal/version/version_test.go` trước đó đã cố định assert chuỗi `"Version: 1.0.2.0 (rev a77efb9)"`, hợp thức hóa sai sót này.

---

## III. GIẢI PHÁP KIẾN TRÚC & CHỈ DẪN THI CÔNG

1. **Phân định rõ ràng trách nhiệm:**
   - **Giao diện người dùng (`a.versionLabel`):** 100% hiển thị định dạng sạch `Version: 1.0.2.0` qua hàm `version.GetFullVersion()`. Tuyệt đối cấm hiển thị `(rev ...)`.
   - **Thông tin kỹ thuật / Telemetry:** Biến `GitCommit` vẫn được lưu giữ và inject qua `-ldflags` lúc build để phục vụ log hệ thống hoặc lệnh CLI debug thông qua hàm mới `GetBuildInfo()`, nhưng tách biệt hoàn toàn khỏi UI.
2. **Quy chuẩn kiểm thử:**
   - Cập nhật test case `TestVersion_WithGitCommit` và `TestAcceptance_TC03_VersionDisplayAndFormatting` để khóa chặt chuỗi `"Version: 1.0.2.0"`.

---

## IV. KHÓA TOUCH-MAP CHO CODER LANE

Coder Lane chỉ được phép sửa đúng 3 tệp sau:
1. `internal/version/version.go`
2. `internal/version/version_test.go`
3. `internal/ui/footer_bar_acceptance_test.go`

*(Nghiêm cấm sửa bất kỳ tệp tin nào ngoài danh sách trên)*

---

## V. KẾT LUẬN & BÀN GIAO (VERDICT)

- **Verdict:** `READY_FOR_CODER`
- **Handoff Target:** Chuyển giao tài liệu đặc tả và Touch-Map sang CEO Agent để điều phối Coder Lane thi công lập tức.
