# Coder Implementation Report: Cập Nhật Tài Liệu README.md Bổ Sung Toàn Diện Hệ Thống Chuyển Đổi Tự Động (Version 1.0.1.8)

- **Date:** 2026-10-20
- **Role:** Coder
- **Authority / SSOT:**
  - Chỉ đạo trực tiếp từ Human Owner
  - `README.md`
  - `AGENTS.md` & `working-rules`
  - `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
  - `reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md`
- **Scope (Touch-Map Khóa Cứng):**
  - `README.md`
  - `reports/coder/2026-10-20-readme-auto-switch-report.md`
- **Non-Goals (Tuân Thủ Kỷ Luật Sắt):**
  - TUYỆT ĐỐI CẤM sửa bất kỳ file mã nguồn nào trong `internal/`, `cmd/`, `scripts/`.
  - TUYỆT ĐỐI CẤM dùng công cụ `anvien`.
  - Cấm tự ý sửa đổi các file ngoài Touch-Map đã khóa.
  - Cấm spawn subagents hoặc gọi `request_user_input`.

---

## 1. Invariant Family Map

| Thuộc tính | Chi tiết |
|---|---|
| **Family Name** | `Documentation & User Guidance Alignment for Autonomous Failover Engine (Version 1.0.1.8)` |
| **SSOT / Authority** | Chỉ đạo trực tiếp của Human Owner; `docs/plans/2026-10-20-autonomous-account-switching/` |
| **Sibling Surfaces** | - Header Version Badge (`Version 1.0.1.8`)<br>- Mô tả tổng quan & Tính năng hiện có<br>- Mục chuyên sâu `Cơ chế Chuyển Đổi Tự Động (Autonomous Failover Engine)`<br>- Hướng dẫn sử dụng CLI (`--switch`, `--list`, `--quota`, `--status`, `--capture`)<br>- Hướng dẫn sử dụng GUI (`AutoSwitchToggle` Row 2, status bar)<br>- Quy trình kiểm thử mở rộng (Race detector, Hermetic E2E 14 edge cases) |
| **Forbidden Legacy Fallback** | Loại bỏ hoàn toàn nhận định cũ: *"Vòng lặp nền tự động làm mới quota, chưa tự chọn tài khoản khác khi hết quota"* |
| **Verification Matrix** | - Cấu trúc tài liệu chuẩn mực tiếng Việt, rõ ràng, mạch lạc.<br>- UTF-8 không BOM.<br>- Git diff sạch sẽ, đúng touch-map khóa cứng. |

---

## 2. Chi Tiết Các Nội Dung Đã Cập Nhật Vào README.md

### 2.1. Ghi nhận Phiên Bản Sản Phẩm
- Đã bổ sung rõ ràng tiêu đề phụ và nhãn phiên bản: **`Phiên bản: Version 1.0.1.8`**.
- Cập nhật đoạn giới thiệu làm nổi bật sự xuất hiện của hệ thống **Chuyển Đổi Tài Khoản Tự Động (Autonomous Multi-Account Switcher & Failover Daemon)**.

### 2.2. Van Kích Hoạt Kép (Dual-Trigger Engine)
- **Kích hoạt chủ động (Proactive Trigger):**
  - Giám sát đồng bộ theo chu kỳ telemetry 60 giây.
  - Ánh xạ model hoạt động (Gemini / Claude) sang nhóm quota tương ứng.
  - Ngưỡng chuyển chủ động:
    + Tài khoản có 5h: Quota 5h <= 2% (`Threshold5hLow = 0.02`) hoặc Quota Tuần <= 1% (`ThresholdWeeklyDepleted = 0.01`).
    + Tài khoản chỉ có tuần: Quota Tuần <= 1% (`ThresholdWeeklyDepleted = 0.01`).
  - Nguyên tắc an toàn: Không chuyển đổi mù quáng khi gặp lỗi mạng hoặc trạng thái không chắc chắn `UNKNOWN_UNCERTAIN`.
- **Kích hoạt bị động tức thì (Reactive Trigger):**
  - Lắng nghe luồng Connect-RPC stream giữa IDE và Language Server.
  - Bắt tức thì khi gặp lỗi cạn kiệt tài nguyên (HTTP 429 / Connect-RPC `RESOURCE_EXHAUSTED`).
  - Thẩm định kết quả thực thi trên stream thật (execution outcome), loại bỏ cảnh báo giả từ HTTP ACK.

### 2.3. Cơ Chế Phục Hồi Native In-Memory 3-Retry
- Tự động thử lại nội bộ trước khi kích hoạt quy trình chuyển tài khoản.
- Gửi yêu cầu `SendUserCascadeMessage` với danh sách nội dung rỗng (`empty content list`) qua Connect-RPC.
- Ngân sách tối đa **3 lần thử lại** (`MaxRecoveryAttempts = 3`) độc lập trên từng hội thoại.
- Chỉ kích hoạt chuyển tài khoản khi toàn bộ 3 lần thử đều xác nhận thất bại do `RESOURCE_EXHAUSTED`.

### 2.4. Thuật Toán Sắp Xếp Thông Minh Burn-down Sorter
- **5 Tiêu chí sơ loại (Pre-filtering):**
  1. Khác tài khoản đang dùng (`Criterion 1`).
  2. Token và refresh token hợp lệ trong kho `accounts_store/` (`Criterion 2`).
  3. Không nằm trong danh sách thất bại của phiên hiện tại (`Criterion 3`).
  4. Quota model khả dụng vượt ngưỡng kích hoạt: có 5h thì 5h > 2% và Tuần > 1%; chỉ có tuần thì Tuần > 1% (`Criterion 4`).
  5. Dữ liệu quota tin cậy, không phải `UNKNOWN_UNCERTAIN` (`Criterion 5`).
- **4 Tầng ưu tiên sắp xếp (Burn-down Sorter Tiers):**
  - Tầng 1: Tuần thấp nhất trước (ưu tiên "burn-down" hết quota tuần sắp cạn: 3% -> 10% -> 50%...).
  - Tầng 2: Ưu tiên tài khoản có chu kỳ 5h so với tài khoản chỉ có tuần.
  - Tầng 3: 5h cao nhất trước (giữa các tài khoản có 5h, ưu tiên tài khoản có tỷ lệ 5h dồi dào nhất: 80% > 40%).
  - Tầng 4: Thứ tự danh bạ cố định ban đầu để đảm bảo tính tất định.

### 2.5. Bảo Vệ Liên Tiến Trình Single-Flight Concurrency Gate
- Sử dụng **Windows Named Mutex** theo Windows Session người dùng (`AntigravitySwitch_SwitchGate_<USERNAME>`) kết hợp in-process semaphore.
- Đồng bộ đa kênh an toàn: Tự động chuyển (Failover Daemon), nút bấm chuyển thủ công trên GUI, và lệnh gọi CLI `antigravity-switch --switch <email>`.
- Ngăn chặn hoàn toàn xung đột ghi đè credential, file lock và race condition giữa các tiến trình.

### 2.6. Tích Hợp Fyne GUI & Trạng Thái Thời Gian Thực
- Công tắc gạt con nhộng **AutoSwitchToggle** đặt tại Toolbar Header Row 2 (ngang hàng nhãn "Tự động" và theme select).
- Thanh trạng thái thời gian thực hiển thị minh bạch các bước: `IDLE`, `EVALUATING`, `RETRYING`, `SWITCHING`, `SWITCHED`, `NO_CANDIDATE`, `ERROR`.

### 2.7. Hướng Dẫn Sử Dụng & Kiểm Thử
- Bổ sung hướng dẫn đầy đủ các lệnh CLI (`--list`, `--status`, `--quota`, `--capture`, `--switch`, `--gui`).
- Bổ sung hướng dẫn kiểm thử chi tiết bao gồm `go test -v -race -count=1 ./internal/automation/...` bao phủ toàn diện 14 tình huống biên (edge cases).

---

## 3. Xác Minh Kỹ Thuật (Verification Evidence)

1. **Kiểm tra File Encoding:**
   - File `README.md` được ghi theo chuẩn UTF-8 không BOM (byte đầu tiên là `0x23 0x20 0xE2 0x9A`, đại diện cho `# ⚡`).
2. **Kiểm tra Touch-Map:**
   - Chỉ có duy nhất 2 file được thay đổi/tạo mới:
     + `README.md` (modified)
     + `reports/coder/2026-10-20-readme-auto-switch-report.md` (untracked/new)
   - Hoàn toàn KHÔNG chỉnh sửa bất kỳ file mã nguồn nào trong `internal/`, `cmd/`, `scripts/`.
3. **Tuân thủ điều cấm:**
   - Không sử dụng công cụ `anvien`.
   - Không gọi `request_user_input`.
   - Không mở lane phụ / subagent.

---

## 4. Kết Luận & Handoff

Hạng mục cập nhật tài liệu `README.md` bổ sung tính năng Chuyển Đổi Tự Động cho phiên bản `Version 1.0.1.8` đã hoàn thành xuất sắc, đầy đủ 100% các yêu cầu chỉ đạo từ Human Owner và CEO.

- **Verdict:** `PASS / READY_FOR_SUPERVISOR`