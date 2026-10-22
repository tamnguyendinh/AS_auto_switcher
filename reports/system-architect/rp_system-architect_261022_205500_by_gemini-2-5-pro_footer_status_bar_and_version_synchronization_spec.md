# Báo Cáo Kiến Trúc: Thanh Đáy (Footer Status Bar), Tự Động Đồng Bộ Version & Cơ Chế Ẩn/Hiện Thông Báo
(System Architect Lane Completion Report — Footer Status Bar, Version Synchronization & Full-Width Notification Specification)

- **Thời gian lập:** 2026-10-22 20:55:00 (Asia/Bangkok)
- **Lane:** System Architect (Nhánh Chiến Lược)
- **Model:** gemini-2-5-pro
- **Thẩm quyền:** Trực tiếp thảo luận, lắng nghe và thống nhất giải pháp cùng Human Owner
- **Mã văn kiện đặc tả liên kết:** `SPEC-ARCH-20261022-FOOTER-STATUS-BAR-AND-VERSION-SYNC-SPEC`
- **Tệp đặc tả kỹ thuật:** `reports/architect/2026-10-22-footer-status-bar-and-version-synchronization-spec.md`
- **Mã báo cáo:** `rp_system-architect_261022_205500_by_gemini-2-5-pro_footer_status_bar_and_version_synchronization_spec`

---

## I. PHẠM VI CÔNG VIỆC HOÀN THÀNH (SCOPE COMPLETED)

1. **Khảo sát và xác minh hiện trạng kỹ thuật:**
   - Khảo sát thanh công cụ Row 2 ([`internal/ui/window.go:121-156`](file:///g:/antigravity_auto_switch/internal/ui/window.go#L121-L156)): Phát hiện cụm text trạng thái tiến trình `a.serverStatusLabel` đứng ở đầu góc trái Header Row 2 làm chiếm dụng không gian và phân tán sự tập trung khỏi 4 nút thao tác chính (`[+ Thêm]`, `[Danh sách tài khoản]`, `[Làm mới Quota]`, `[📌]`).
   - Khảo sát cơ chế Versioning: Hiện tại hệ thống chưa có Single Source of Truth cho Version trong mã nguồn Go, script build `scripts/build_windows.ps1` chưa inject version và git commit hash qua `-ldflags`, tệp Windows PE resource `cmd/switcher/app.rc` chưa có khối `VERSIONINFO`.
   - Khảo sát cơ chế hiển thị thông báo (`a.statusBar`): Hoạt động tốt nhưng bị hạn chế bề ngang khi đặt cạnh các thành phần khác.

2. **Thảo luận sâu & Đồng thuận giải pháp cùng Human Owner:**
   - **Tách bạch không gian (Separation of Concerns):** Dời `a.serverStatusLabel` xuống mép trái thanh đáy, giải phóng hoàn toàn Header Row 2 cho các nút thao tác chính.
   - **Định danh ngữ nghĩa rõ ràng:** Thêm tiền tố `Antigravity:` (`Antigravity: Hoạt động | PID 19224 | Port 57868`, `Antigravity: Không hoạt động | Dự phòng WinCred`, `Antigravity: Đang kết nối...`).
   - **Cơ chế Versioning chuẩn:** Tạo package `internal/version` (Single Source of Truth, mặc định `"1.0.2.0"`), inject qua `-ldflags` và tự động đồng bộ hóa sang PE resource `app.rc` để Windows Explorer Properties hiển thị chuẩn.
   - **Cơ chế Ẩn/Hiện thông báo toàn chiều ngang (Full-Width Takeover):** Khi có thông báo tức thời, tạm ẩn cả text bên trái (`Antigravity:...`) và text bên phải (`Version:...`) để thông báo chiếm trọn 100% chiều ngang thanh đáy (căn giữa); khi thông báo tắt, tự động khôi phục lại cả hai nhãn.

3. **Biên soạn văn kiện đặc tả kỹ thuật chi tiết:**
   - Đã biên soạn hoàn chỉnh tài liệu đặc tả: [reports/architect/2026-10-22-footer-status-bar-and-version-synchronization-spec.md](file:///g:/antigravity_auto_switch/reports/architect/2026-10-22-footer-status-bar-and-version-synchronization-spec.md).
   - Phân chia lộ trình thi công 4 Giai đoạn (Phases) theo đúng chỉ đạo của Owner:
     * **Phase 1:** Dời `serverStatusLabel` xuống đáy, thêm tiền tố `Antigravity:`, tinh gọn Header Row 2.
     * **Phase 2:** Hệ thống Versioning & Hiển thị `Version: xxxxx (rev yyyyyyy)` ở mép phải thanh đáy.
     * **Phase 3:** Cơ chế ẩn text trái/phải khi thông báo kích hoạt (Full-Width Notification takeover).
     * **Phase 4:** Bộ Kiểm thử Chấp nhận Toàn diện (TC-01 đến TC-06) & Verification Gate.

---

## II. DANH MỤC TỆP ĐẦU RA (OUTPUT ARTIFACTS)

1. `reports/architect/2026-10-22-footer-status-bar-and-version-synchronization-spec.md` (Văn kiện đặc tả kiến trúc chi tiết, phân chia 4 Phase rõ ràng)
2. `reports/system-architect/rp_system-architect_261022_205500_by_gemini-2-5-pro_footer_status_bar_and_version_synchronization_spec.md` (Báo cáo hoàn thành của System Architect Lane)

---

## III. QUYẾT ĐỊNH KIẾN TRÚC ĐÃ CHỐT (ARCHITECTURAL DECISIONS)

1. **Header Row 2 Layout:** Bắt đầu trực tiếp bằng cụm 4 nút thao tác (`toolbarLeft`), không còn gắn `serverStatusLabel` ở góc trái.
2. **Footer Status Bar Layout:** Sử dụng `container.NewBorder(nil, nil, a.serverStatusLabel, a.versionLabel, a.statusBar)` bọc trong `container.NewVBox(widget.NewSeparator(), container.NewPadded(...))`.
3. **Prefix Định danh:** Bổ sung `Antigravity:` trước các thông điệp trạng thái kết nối Language Server.
4. **Versioning Source of Truth:** Khai báo tại `internal/version/version.go`, inject qua `-ldflags` trong `scripts/build_windows.ps1` (`Version` và `GitCommit`), đồng bộ khối `VERSIONINFO` trong `cmd/switcher/app.rc`.
5. **Notification Takeover Interaction:** Trong `showStatusBar(msg)`, gọi `serverStatusLabel.Hide()` và `versionLabel.Hide()`; trong `clearStatusBar()`, gọi `serverStatusLabel.Show()` và `versionLabel.Show()`.

---

## IV. VẤN ĐỀ CÒN TỒN ĐỌNG (RESIDUAL OPEN QUESTIONS)

- **Không có (None).** Toàn bộ các góc độ về UX, vị trí, tiền tố, chu kỳ hiển thị, cơ chế versioning và chiến lược test chấp nhận đều đã được thảo luận và thống nhất trọn vẹn cùng Human Owner.

---

## V. ĐỀ XUẤT BÀN GIAO & KẾT LUẬN (HANDOFF RECOMMENDATION)

- **Verdict:** `READY_FOR_PLANNER` / `READY_FOR_CODER`
- **Handoff Target:** Chuyển giao gói tài liệu đặc tả cho Strategic Planner lập kế hoạch triển khai chi tiết hoặc CEO Agent trực tiếp điều phối Coder Lane thi công 4 Phases.
