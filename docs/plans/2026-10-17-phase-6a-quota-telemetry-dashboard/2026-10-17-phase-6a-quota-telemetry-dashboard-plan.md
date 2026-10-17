# Phase 6A: Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System Plan

## Metadata

- Date: `2026-10-17`
- Status: `in_progress (Slice P6A-1 completed; next: Slice P6A-2)`
- Plan: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
- Evidence: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`
- Benchmark: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-benchmark.md`
- Actual status: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-actual-status.md`

## Goal

Chuyển hóa toàn văn Báo cáo Đặc tả Kiến trúc Phase 6A (`reports/architect/2026-10-17-phase-6a-quota-telemetry-and-multi-theme-dashboard-architecture.md`) thành kế hoạch thực thi chuẩn mực, giải quyết triệt để vấn đề "Blind Failover" (chuyển đổi mù quáng) của hệ thống chuyển đổi tài khoản Antigravity.

Kế hoạch phân rã thành 3 Slices thực thi độc lập:
1. **Slice P6A-1: Quota Telemetry Engine & 60s Auto-Telemetry Loop (`internal/telemetry/`)**
   - Xây dựng HTTP client truy vấn Google Cloud Code endpoint `v1internal:retrieveUserQuota`.
   - Bóc tách hạn ngạch chính xác theo nhóm model: Claude 4.6 (Opus/Sonnet), Gemini 2.5 Pro/Flash, GPT-OSS; ghi nhận thời điểm Google reset quota (UTC).
   - Xây dựng concurrent worker pool quét đồng thời $N$ tài khoản với độ trễ < 2s.
   - Triển khai bộ nhớ đệm hai tầng (RAM `QuotaMatrix` + đĩa `accounts_store/.quota_cache.json`).
   - Xây dựng vòng lặp ngầm 60s (Auto-Telemetry Loop) với cổng chống dồn lệnh `atomic.Bool` single-flight.
2. **Slice P6A-2: Fyne UI Quota Dashboard, Refresh Actions & Multi-Theme System (`internal/ui/`)**
   - Mở rộng giao diện Fyne Native GUI: hiển thị Mini Quota Progress Badges cho từng tài khoản với 3 mức màu trực quan: Xanh (>50%), Vàng (15-50%), Đỏ (<15%).
   - Bổ sung nút "Làm mới toàn bộ" (`[📊 Làm mới Quota Tất cả]`) trên Header Toolbar và nút làm mới từng tài khoản (`[🔄]`) trên mỗi Card.
   - Triển khai Hệ thống Đa Giao diện (Multi-Theme System): Cung cấp 3 theme chuẩn mực (Dark Mode Catppuccin Mocha mặc định, Light Mode paper slate, Warm Mode amber/sepia chống mỏi mắt) với bộ chọn chuyển đổi mượt mà không crash app, lưu cấu hình vào `accounts_store/.ui_preferences.json`.
   - Bảo đảm giao diện bất đồng bộ không khóa (Non-blocking UI), giữ tốc độ khung hình 60 FPS khi đang quét mạng.
3. **Slice P6A-3: CLI Status Extension & UI Traceability Matrix Verification (`cmd/switcher/`)**
   - Bổ sung cờ CLI `antigravity-switch.exe --quota` xuất bảng hạn ngạch trực quan ra terminal Windows.
   - Đối soát toàn diện bảng truy vết giao diện UI Traceability Matrix (từ `UI-TR-01` đến `UI-TR-08`) theo tiêu chuẩn Zero-Trust Definition of Done.

## Rules

- Hoàn thành P0 actual status trước khi tiến hành thực thi code.
- Mỗi phase/slice phải tuân thủ nghiêm ngặt chu kỳ thực thi 6 bước (`Code → Inspect → Update tests → Build → QA → Acceptance/Commit`).
- Bước 1 (Code) phải được chia nhỏ thành các leaf atomic task có Action, Inputs, Allowed Edit Scope, Outputs, Điều kiện kiểm chứng và Task Checkpoint (`wip(<slice>): task 1.x - <outcome>`) rõ ràng.
- Tuyệt đối tuân thủ nguyên tắc "Code first, test second": Bước 1 chỉ sửa file mã nguồn sản phẩm, Bước 3 mới viết/cập nhật unit & integration tests.
- Tuyệt đối không sửa mã nguồn ngoài Touch-Map đã khóa cho từng Slice.
- Không phá vỡ các chức năng chuyển đổi tài khoản (`internal/manager/`), quét Chrome (`internal/scanner/`), và reloader (`internal/reloader/`) hiện có.
- Cấm lạm dụng fallback ẩn (hidden fallback). Lỗi từ Google Quota API phải được ghi nhận rõ ràng vào model quota của tài khoản đó.
- Cấm tự ý mở thêm tính năng ngoài phạm vi Phase 6A (đặc biệt: Phase 7 Autonomous Failover Daemon là phạm vi của giai đoạn tiếp theo).

## Problem

Theo Báo cáo Đặc tả Kiến trúc `reports/architect/2026-10-17-phase-6a-quota-telemetry-and-multi-theme-dashboard-architecture.md`:
1. **Khuyết tật "Blind Failover" (Chuyển đổi mù quáng):**
   - Khi gặp mã lỗi HTTP 429 Quota Exhausted, cơ chế failover ban đầu chuyển ngẫu nhiên hoặc round-robin sang tài khoản kế tiếp mà hoàn toàn không biết tài khoản đó có còn hạn ngạch hay không.
   - Dẫn đến 3 thảm họa vận hành:
     + **Bão chuyển đổi liên hoàn (Cascading Flapping Storm):** Tài khoản tiếp theo cũng cạn quota -> lại kích hoạt 429 -> lại switch -> khởi động lại Language Server liên tục làm crash IDE.
     + **Đoán mò thời gian hồi phục (Speculative Cooldown):** Gán thời gian chờ tĩnh (15m, 1h, 2h) vô căn cứ, trong khi Google Cloud Code có thời điểm reset quota (UTC) cụ thể.
     + **Bất đối xứng hạn ngạch đa model (Multi-Model Asymmetry):** Tài khoản có thể cạn 100% quota Gemini nhưng vẫn còn nguyên 100% Claude 4.6 hay GPT-OSS. Chuyển đổi mù làm lãng phí nghiêm trọng tài nguyên.
2. **Giao diện đơn điệu và thiếu thông tin hạn ngạch:**
   - GUI hiện tại chỉ hiển thị trạng thái tài khoản cơ bản (`ACTIVE`, `READY`, `UNAUTH`), lập trình viên không biết tài khoản nào sắp hết quota để chủ động điều phối.
   - GUI chỉ có 1 Dark Theme cố định, chưa đáp ứng nhu cầu làm việc ban ngày (Light Theme) hay làm việc đêm chống mỏi mắt (Warm Theme).

## Scope

### In-Scope (Touch-Map cho 3 Slices)
- **Slice P6A-1:**
  - `internal/telemetry/client.go` (tạo mới)
  - `internal/telemetry/cache.go` (tạo mới)
  - `internal/telemetry/pool.go` (tạo mới)
  - `internal/telemetry/loop.go` (tạo mới)
  - `internal/telemetry/telemetry_test.go` (tạo mới)
- **Slice P6A-2:**
  - `internal/ui/themes.go` (tạo mới)
  - `internal/ui/quota_badge.go` (tạo mới)
  - `internal/ui/widgets.go` (mở rộng `NewAccountCard`)
  - `internal/ui/window.go` (mở rộng Toolbar & Layout)
  - `internal/ui/app.go` (tích hợp telemetry loop & async dispatch)
  - `internal/ui/ui_test.go` & `internal/ui/themes_test.go` (cập nhật test)
- **Slice P6A-3:**
  - `cmd/switcher/main.go` (bổ sung cờ `--quota` và bảng xuất tabwriter)
  - `cmd/switcher/app_cli.go` & `app_gui.go` (đồng bộ build tags)

### Out-of-Scope (Non-Goals)
- CẤM sửa mã nguồn `internal/reloader/` (Session reloader và thread preservation).
- CẤM sửa mã nguồn `internal/wincred/` (Windows Credential Manager).
- CẤM sửa mã nguồn `internal/scanner/` (Chrome profile scanner).
- CẤM sửa định dạng lưu trữ token WinCred (`LegacyGeneric:target=gemini:antigravity`).
- CẤM triển khai daemon tự động failover (thuộc phạm vi Phase 7).

## Technical Invariants

1. **Google Quota Endpoint Invariant:** Sử dụng endpoint chính thức `https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota` với phương thức POST, bearer token từ `internal/connector.GoogleOAuthRefresher`.
2. **Single-Flight Concurrency Gate Invariant:** Vòng lặp 60s phải được bảo vệ bằng `atomic.Bool`. Nếu lần quét trước chưa hoàn tất, lần tick 60s tiếp theo bị DROP ngay lập tức (`CompareAndSwap(false, true)`), triệt tiêu hoàn toàn hiện tượng nghẽn mạng hay chồng lấn request.
3. **Thread-Safe Two-Tier Cache Invariant:** `QuotaMatrix` trong RAM bảo vệ bằng `sync.RWMutex`. Đĩa snapshot tại `accounts_store/.quota_cache.json` cho phép khởi động tức thì mà không cần chờ mạng.
4. **Non-Blocking UI Invariant:** Toàn bộ tác vụ mạng và refresh hạn ngạch phải chạy trên goroutine nền. Mọi cập nhật Canvas UI phải điều phối qua `fyne.Do()` trên main thread, duy trì tốc độ 60 FPS mượt mà.
5. **Three-Tier Color Threshold Invariant:** Màu sắc hiển thị badge quota bắt buộc tuân theo:
   - Xanh (`#22c55e` / `#a6e3a1`): Hạn ngạch $> 50%$
   - Vàng/Hổ phách (`#f59e0b` / `#f9e2af`): Hạn ngạch $15\% - 50\%$
   - Đỏ (`#ef4444` / `#f38ba8`): Hạn ngạch $< 15\%$ kèm dòng chữ đếm ngược thời gian hồi phục của Google.
6. **Pure Fyne Theme Hot-Switch Invariant:** Cả 3 theme (`DarkTheme`, `LightTheme`, `WarmTheme`) tuân thủ `fyne.Theme`, chuyển đổi tức thời qua `app.Settings().SetTheme()` mà không làm giật cửa sổ, không cần khởi động lại tiến trình.
7. **UI Traceability Matrix Invariant:** Bắt buộc thỏa mãn 8/8 mục kiểm thử từ `UI-TR-01` đến `UI-TR-08`.
8. **Zero-Regression Invariant:** Việc bổ sung telemetry và theme mới không được làm suy giảm độ ổn định và thời gian chuyển đổi tài khoản hiện có (< 1s).

## Acceptance Criteria (Definition of Done)

- **AC-P6A-1 (Telemetry Engine & Loop):**
  - Quét song song 7 tài khoản trong `accounts_store/` với thời gian hoàn thành < 2.0s (kết quả thực nghiệm: 1.69s).
  - Bóc tách đầy đủ hạn ngạch Claude 4.6, Gemini 2.5, GPT-OSS và thời điểm reset UTC.
  - Vòng lặp 60s tự động cập nhật ngầm, single-flight gate drop 100% khi có xung đột.
  - Test suite `internal/telemetry/` đạt độ bao phủ $> 90\%$, `go test -race` không phát hiện data race.
- **AC-P6A-2 (UI Dashboard & Multi-Theme):**
  - Giao diện Fyne hiển thị đầy đủ Mini Quota Badges trên từng card tài khoản với 3 màu chuẩn mực.
  - Header Toolbar có nút `[📊 Làm mới Quota Tất cả]` và nhãn `Tự động: 60s (cập nhật lúc HH:mm:ss)`.
  - Mỗi dòng tài khoản có nút `[🔄]` làm mới riêng biệt cho tài khoản đó.
  - Bộ chọn theme `[🎨 Giao diện: Tối | Sáng | Ấm]` chuyển đổi tức thì giao diện không crash, lưu cấu hình vào `accounts_store/.ui_preferences.json`.
  - Thỏa mãn 100% các tiêu chí từ `UI-TR-01` đến `UI-TR-08`.
- **AC-P6A-3 (CLI Extension & Dual-Build):**
  - Lệnh `antigravity-switch.exe --quota` in bảng phân bổ hạn ngạch trực quan ra terminal Windows.
  - Độ trễ khởi động CLI <= 50ms, bộ nhớ RAM <= 20MB.
  - Biên dịch thành công cả 2 target: CLI thuần (`go build ./cmd/switcher`) và GUI (`go build -tags gui ./cmd/switcher`).

---

## Checklist

- [x] P0-A: Complete actual status before implementation work.
  - Goal: Khảo sát hiện trạng codebase, xác định các thành phần `missing`, `partial`, `correct`, và lập các tệp kế hoạch theo chuẩn.
  - Work Steps: Kiểm tra `internal/connector/oauth_refresher.go`, `internal/ui/`, `cmd/switcher/`, `accounts_store/`, chạy baseline test suite `go test ./...`.
  - Implementation Gate: Tệp `actual-status.md` hoàn thành với bảng ma trận chi tiết và evidence IDs.
  - Acceptance: 8/8 gói phần mềm hiện tại pass test, 7 tài khoản trong store sẵn sàng, xác định rõ Touch-Map cho 3 Slices.

### P1: Slice P6A-1 — Quota Telemetry Engine & 60s Auto-Telemetry Loop (`internal/telemetry/`)

- Phase Goal: Xây dựng package `internal/telemetry/` hoàn chỉnh gồm HTTP client gọi Google `v1internal:retrieveUserQuota`, concurrent worker pool quét $N$ tài khoản trong < 2s, cache 2 tầng RAM/đĩa, và vòng lặp tự động 60s ngầm bảo vệ bằng atomic single-flight gate.
- Phase Boundary:
  - In scope: `internal/telemetry/client.go`, `internal/telemetry/cache.go`, `internal/telemetry/pool.go`, `internal/telemetry/loop.go`, `internal/telemetry/telemetry_test.go`.
  - Out of scope: Không sửa `internal/ui/`, không sửa `cmd/switcher/`.
  - Dependencies: Tái sử dụng `internal/connector.GoogleOAuthRefresher` và dữ liệu từ `accounts_store/`.
- Phase Implementation Rule: Thực hiện tuần tự Step 1 (Code leaf tasks), Step 2 (Inspect & Lint), Step 3 (Update tests), Step 4 (Build), Step 5 (Mini QA), Step 6 (Acceptance & Commit).
- Ordered Slice List:
  - P1-A: Quota Telemetry Client, Worker Pool, Cache, & 60s Loop.

- [x] P1-A: Quota Telemetry Client, Worker Pool, Cache, & 60s Loop.
  - Goal: Cung cấp đầy đủ hạ tầng thu thập, chuẩn hóa, lưu trữ và tự động làm mới hạn ngạch Google Cloud Code cho toàn bộ tài khoản trong pool.
  - Scope Boundary:
    - Editable: `internal/telemetry/client.go`, `internal/telemetry/cache.go`, `internal/telemetry/pool.go`, `internal/telemetry/loop.go`, `internal/telemetry/telemetry_test.go`.
    - Inspect-only: `internal/connector/oauth_refresher.go`, `accounts_store/*.json`.
    - Preserve-only: `internal/reloader/`, `internal/wincred/`, `internal/scanner/`.
    - Out of scope: Mã nguồn UI và CLI flags.
  - Non-Goals: Không thực hiện tự động chuyển đổi tài khoản khi hết quota (thuộc Phase 7).
  - Pre-flight Questions:
    - Data source: Google Cloud Code endpoint `https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota` với access token từ `internal/connector.GoogleOAuthRefresher`.
    - Display permission: N/A (Backend telemetry).
    - DB read flow: Đọc refresh token từ `accounts_store/<email>.json` và snapshot từ `accounts_store/.quota_cache.json`.
    - DB write flow: Ghi snapshot vào `accounts_store/.quota_cache.json` bằng atomic write.
    - Render location: N/A.
    - UI behavior flow: N/A.
    - Docker runtime: N/A (Windows native binary).
    - Playwright target: N/A.
    - Behavior test: Unit test với mock HTTP server mô phỏng Google Cloud Code và test race condition với `-race`.
    - Cleanup/quarantine: Dọn dẹp tệp cache tạm trong test runner.
    - External side effects: Gửi HTTP POST tới Google Cloud Code API (chỉ tiêu tốn < 0.1% rate limit).
    - N/A notes: Docker runtime và Playwright N/A cho backend Go telemetry.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Quota HTTP Client & Model Aggregator
         + Action: Xây dựng `internal/telemetry/client.go` với struct `QuotaClient`, hàm `RetrieveUserQuota(ctx, refreshToken)`, gọi Google OAuth Refresher lấy access token, POST tới Google endpoint, phân tích JSON payload, nhóm hạn ngạch vào Claude 4.6, Gemini 2.5, GPT-OSS và bóc tách thời gian reset UTC.
         + Inputs: `connector.GoogleOAuthRefresher`, HTTP client, refreshToken string.
         + Allowed Edit Scope: `internal/telemetry/client.go`.
         + Outputs: `AccountQuotaData`, `ModelQuotaBucket`, và `QuotaClient`.
         + Verification Condition: `go vet ./internal/telemetry/...`
         + Task Checkpoint: `wip(P6A-1): task 1.1 - implement google quota client and bucket aggregator`
       - Task 1.2: Two-Tier Quota Cache (RAM + Disk)
         + Action: Xây dựng `internal/telemetry/cache.go` với `QuotaCache` quản lý in-memory `QuotaMatrix` có `sync.RWMutex`, cung cấp `Get`, `GetAll`, `Update`, `SaveToDisk(accounts_store/.quota_cache.json)`, `LoadFromDisk()`.
         + Inputs: `AccountQuotaData`, file path `accounts_store/.quota_cache.json`.
         + Allowed Edit Scope: `internal/telemetry/cache.go`.
         + Outputs: `QuotaCache` và atomic persistence logic.
         + Verification Condition: `go vet ./internal/telemetry/...`
         + Task Checkpoint: `wip(P6A-1): task 1.2 - implement thread-safe two-tier quota cache`
       - Task 1.3: Concurrent Worker Pool
         + Action: Xây dựng `internal/telemetry/pool.go` với `TelemetryPool` điều phối goroutines quét đồng thời $N$ tài khoản với `context.WithTimeout(ctx, 5*time.Second)`, tổng thời gian quét < 2s cho 7 tài khoản.
         + Inputs: Danh sách tài khoản lưu trong `accounts_store/`, `QuotaClient`, `QuotaCache`.
         + Allowed Edit Scope: `internal/telemetry/pool.go`.
         + Outputs: `PollAll(ctx)` và `PollAccount(ctx, email)`.
         + Verification Condition: `go vet ./internal/telemetry/...`
         + Task Checkpoint: `wip(P6A-1): task 1.3 - implement parallel worker pool for multi-account telemetry`
       - Task 1.4: 60-Second Auto-Telemetry Loop
         + Action: Xây dựng `internal/telemetry/loop.go` với `AutoTelemetryLoop` chạy `time.NewTicker(60 * time.Second)`. Quản lý trạng thái bằng `atomic.Bool isRunning` (`CompareAndSwap(false, true)`). Cung cấp `TriggerNow()` để reset ticker khi bấm thủ công, và `Stop()` để hủy an toàn.
         + Inputs: `TelemetryPool`, interval 60s, context lifecycle.
         + Allowed Edit Scope: `internal/telemetry/loop.go`.
         + Outputs: `AutoTelemetryLoop` struct và lifecycle methods.
         + Verification Condition: `go vet ./internal/telemetry/...`
         + Task Checkpoint: `wip(P6A-1): task 1.4 - implement 60s auto-telemetry ticker loop with single-flight gate`
    2. Code Inspection & Lint:
       - Kiểm tra diff, định dạng mã (`gofmt`), typecheck, và bảo đảm không sửa ngoài `internal/telemetry/`.
    3. Update Tests (Code first, test second):
       - Tạo `internal/telemetry/telemetry_test.go` với mock HTTP server kiểm tra:
         + Phân tích đúng 100% quota, 0% quota, depleted quota.
         + Bóc tách chính xác ISO 8601 UTC reset time.
         + Xử lý lỗi HTTP 401, 403, 500, network timeout.
         + Kiểm thử cổng `atomic.Bool` single-flight drop overlap.
         + Kiểm thử race condition với `go test -race ./internal/telemetry/...`.
    4. Build:
       - Chạy `go build ./internal/telemetry/...` và `go test -v ./internal/telemetry/...`.
    5. QA (Mini QA / Runtime Validation):
       - Chạy benchmark đo độ trễ worker pool quét 7 tài khoản giả lập bảo đảm < 2.0s.
       - Evidence target: `E1-P1A-TEST1`, `E1-P1A-TEST2`, `E1-P1A-TEST3`.
    6. Slice Acceptance & Official Milestone Commit:
       - Kiểm tra các tiêu chuẩn hoàn thành của P1-A, cập nhật `evidence.md`, cập nhật hàng `internal/telemetry/` từ `missing -> correct` trong `actual-status.md`, tạo commit:
         `feat(P6A-1): implement quota telemetry engine with 60s auto-loop and two-tier cache`.
  - Implementation Gate:
    - Kiểm tra `accounts_store/` có tối thiểu 1 tài khoản với refresh token hợp lệ trước khi chạy tích hợp.
  - Acceptance:
    - Source: Package `internal/telemetry/` biên dịch sạch, không cảnh báo linter.
    - Runtime: Mock worker pool hoàn thành quét $N$ tài khoản trong < 2s.
    - Data: Đọc/ghi snapshot `.quota_cache.json` hoạt động chuẩn xác.
    - Behavior test: `go test -race ./internal/telemetry/...` vượt qua 100%, test coverage $> 90\%$.
    - Cleanup/quarantine: Không lưu rác trên hệ thống file ngoài `accounts_store/`.
    - Evidence IDs: `E1-P1A-CODE1..CODE4`, `E1-P1A-TEST1..TEST3`.
    - Actual-status rows refreshed: `internal/telemetry/` chuyển sang `correct`.
  - Evidence Targets: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`.
  - Actual-status Update: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-actual-status.md`.
  - Commit Boundary: Commit chính thức sau khi hoàn tất Slice P6A-1.

---

### P2: Slice P6A-2 — Fyne UI Quota Dashboard, Refresh Actions & Multi-Theme System (`internal/ui/`)

- Phase Goal: Mở rộng giao diện Fyne Native GUI (`internal/ui/`) với Quota Progress Badges cho từng tài khoản, nút làm mới toàn bộ trên Header Toolbar, nút làm mới từng dòng, và Hệ thống Đa Giao diện (Dark, Light, Warm Mode) chuyển đổi mượt mà không crash app, duy trì tốc độ 60 FPS non-blocking.
- Phase Boundary:
  - In scope: `internal/ui/themes.go`, `internal/ui/quota_badge.go`, `internal/ui/widgets.go`, `internal/ui/window.go`, `internal/ui/app.go`, `internal/ui/ui_test.go`, `internal/ui/themes_test.go`.
  - Out of scope: Không sửa `internal/telemetry/`, không sửa `internal/reloader/`.
  - Dependencies: Dữ liệu từ `internal/telemetry/`.
- Phase Implementation Rule: Thực hiện tuần tự Step 1 (Code leaf tasks), Step 2 (Inspect & Lint), Step 3 (Update tests), Step 4 (Build), Step 5 (Mini QA), Step 6 (Acceptance & Commit).
- Ordered Slice List:
  - P2-A: Multi-Theme Engine, Mini Quota Badges, Toolbar Refresh & Individual Refresh Buttons.

- [ ] P2-A: Multi-Theme Engine, Mini Quota Badges, Toolbar Refresh & Individual Refresh Buttons.
  - Goal: Nâng cấp trải nghiệm người dùng với bảng điều khiển hạn ngạch trực quan, các nút điều khiển làm mới linh hoạt, và 3 bộ giao diện thẩm mỹ cao.
  - Scope Boundary:
    - Editable: `internal/ui/themes.go`, `internal/ui/quota_badge.go`, `internal/ui/widgets.go`, `internal/ui/window.go`, `internal/ui/app.go`, `internal/ui/ui_test.go`, `internal/ui/themes_test.go`.
    - Inspect-only: `internal/telemetry/cache.go`, `internal/manager/account_manager.go`.
    - Preserve-only: `internal/reloader/`, `internal/wincred/`, `internal/scanner/`.
    - Out of scope: CLI commands và backend telemetry engine.
  - Non-Goals: Không làm thay đổi kích thước cửa sổ chuẩn (440x560), không làm chậm chuyển đổi tài khoản.
  - Pre-flight Questions:
    - Data source: `telemetry.QuotaCache` và `manager.AccountOverview`.
    - Display permission: Người dùng chạy ứng dụng desktop.
    - DB read flow: Đọc cấu hình giao diện từ `accounts_store/.ui_preferences.json`.
    - DB write flow: Ghi lựa chọn theme vào `accounts_store/.ui_preferences.json`.
    - Render location: Fyne Canvas (Window, Toolbar, Cards, Status bar).
    - UI behavior flow:
      1. Bấm `[📊 Làm mới Quota Tất cả]` -> kích hoạt worker pool ngầm -> hiển thị loading indicator -> cập nhật badges qua `fyne.Do()`.
      2. Bấm `[🔄]` trên card -> chỉ làm mới hạn ngạch của tài khoản đó.
      3. Bấm chọn Theme (`Tối`, `Sáng`, `Ấm`) -> gọi `SetTheme()` tức thì mà không restart.
    - Docker runtime: N/A (Windows native Fyne desktop GUI).
    - Playwright target: N/A.
    - Behavior test: Unit test với Fyne headless test app kiểm tra render card, theme switching, và badge thresholds.
    - Cleanup/quarantine: Dọn dẹp tệp `.ui_preferences.json` sau test.
    - External side effects: Không có (tác vụ mạng do telemetry engine đảm nhiệm).
    - N/A notes: Docker runtime và Playwright N/A cho Fyne native Windows GUI.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: Multi-Theme Engine (Dark, Light, Warm)
         + Action: Xây dựng `internal/ui/themes.go` định nghĩa `DarkTheme` (Catppuccin Mocha), `LightTheme` (paper slate), và `WarmTheme` (amber/sepia lọc ánh sáng xanh) thỏa mãn `fyne.Theme`. Cung cấp hàm chuyển theme tức thời và lưu cấu hình vào `accounts_store/.ui_preferences.json`.
         + Inputs: Fyne theme interface, mã màu chuẩn từ kiến trúc.
         + Allowed Edit Scope: `internal/ui/themes.go`.
         + Outputs: Các struct Theme và theme manager.
         + Verification Condition: `go vet ./internal/ui/...`
         + Task Checkpoint: `wip(P6A-2): task 1.1 - implement multi-theme system (dark, light, warm)`
       - Task 1.2: Mini Quota Badges & Recovery Countdown Widget
         + Action: Xây dựng `internal/ui/quota_badge.go` tạo component hiển thị hạn ngạch Claude, Gemini, GPT với 3 mức màu: Xanh (>50%), Vàng (15-50%), Đỏ (<15%). Nếu hạn ngạch < 15%, hiển thị dòng đếm ngược: `Google sẽ hồi phục lúc: HH:mm (còn Xh Yp)`.
         + Inputs: Giá trị phần trăm quota, thời điểm reset Google.
         + Allowed Edit Scope: `internal/ui/quota_badge.go`.
         + Outputs: `NewQuotaBadgeGroup` và `NewQuotaCountdownLabel`.
         + Verification Condition: `go vet ./internal/ui/...`
         + Task Checkpoint: `wip(P6A-2): task 1.2 - create mini quota progress badges and countdown widget`
       - Task 1.3: Account Card Quota & Refresh Button Integration
         + Action: Cập nhật `NewAccountCard` trong `internal/ui/widgets.go` để tích hợp mini quota badges và nút icon `[🔄]` làm mới riêng biệt cho từng tài khoản đặt cạnh email.
         + Inputs: `AccountDisplayItem`, `AccountQuotaData`, callback làm mới.
         + Allowed Edit Scope: `internal/ui/widgets.go`.
         + Outputs: Card tài khoản nâng cấp với quota badges.
         + Verification Condition: `go vet ./internal/ui/...`
         + Task Checkpoint: `wip(P6A-2): task 1.3 - integrate quota badges and per-row refresh button into account card`
       - Task 1.4: Header Toolbar Controls Extension
         + Action: Cập nhật `setupLayout()` trong `internal/ui/window.go` để bổ sung:
           1. Nút `[📊 Làm mới Quota Tất cả]` trên toolbar.
           2. Nhãn hiển thị chu kỳ `Tự động: 60s` và timestamp `HH:mm:ss`.
           3. Bộ chọn Theme `[🎨 Giao diện: Tối | Sáng | Ấm]`.
         + Inputs: Window layout, toolbar buttons, theme selector widget.
         + Allowed Edit Scope: `internal/ui/window.go`.
         + Outputs: Header toolbar mở rộng đầy đủ các điều khiển mới.
         + Verification Condition: `go vet ./internal/ui/...`
         + Task Checkpoint: `wip(P6A-2): task 1.4 - add toolbar quota refresh button, 60s indicator, and theme selector`
       - Task 1.5: App Lifecycle Telemetry Wiring & Non-Blocking Dispatch
         + Action: Cập nhật `internal/ui/app.go` để khởi tạo `AutoTelemetryLoop`, bắt sự kiện refresh quota toàn bộ hoặc đơn lẻ, dispatch dữ liệu lên giao diện qua `fyne.Do()`, duy trì UI responsive 60 FPS trong mọi hoàn cảnh.
         + Inputs: `App` struct, `telemetry.TelemetryPool`, `telemetry.AutoTelemetryLoop`.
         + Allowed Edit Scope: `internal/ui/app.go`.
         + Outputs: App controller tích hợp hoàn chỉnh với telemetry loop.
         + Verification Condition: `go vet ./internal/ui/...`
         + Task Checkpoint: `wip(P6A-2): task 1.5 - wire telemetry loop and non-blocking event dispatch in app controller`
    2. Code Inspection & Lint:
       - Kiểm tra diff, bảo đảm không rò rỉ bộ nhớ, không gọi Fyne Canvas API ngoài main thread (`fyne.Do()` bắt buộc).
    3. Update Tests (Code first, test second):
       - Tạo `internal/ui/themes_test.go` kiểm tra đổi qua lại giữa 3 theme không gây panic.
       - Cập nhật `internal/ui/ui_test.go` kiểm tra tạo Card có quota badge, nhãn countdown hiển thị đúng khi < 15%.
    4. Build:
       - Chạy `go build -tags gui ./internal/ui/...` và `go test -v ./internal/ui/...`.
    5. QA (Mini QA / Runtime Validation):
       - Chạy headless Fyne test app kiểm tra render đầy đủ các phần tử theo UI Traceability Matrix (`UI-TR-01` đến `UI-TR-08`).
       - Evidence target: `E2-P2A-CODE1..CODE5`, `E2-P2A-TEST1`, `E2-P2A-QA1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Kiểm tra các tiêu chuẩn hoàn thành của P2-A, cập nhật `evidence.md`, cập nhật hàng `internal/ui/` từ `partial -> correct` trong `actual-status.md`, tạo commit:
         `feat(P6A-2): implement fyne ui quota dashboard and multi-theme system`.
  - Implementation Gate:
    - Slice P6A-1 phải hoàn thành và pass test trước khi bắt đầu Slice P6A-2.
  - Acceptance:
    - Source: Mã nguồn UI biên dịch sạch cả với tag `gui` và mặc định.
    - Runtime/UI: Đổi theme tức thời trong < 50ms, giao diện không giật, hiển thị chuẩn màu 3 mức quota.
    - Traceability: Đạt 8/8 mục kiểm thử từ `UI-TR-01` đến `UI-TR-08`.
    - Behavior test: `go test ./internal/ui/...` pass 100%.
    - Evidence IDs: `E2-P2A-CODE1..CODE5`, `E2-P2A-TEST1`, `E2-P2A-QA1`.
    - Actual-status rows refreshed: `internal/ui/themes.go`, `internal/ui/quota_badge.go`, `internal/ui/widgets.go`, `internal/ui/window.go`, `internal/ui/app.go` chuyển sang `correct`.
  - Evidence Targets: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`.
  - Actual-status Update: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-actual-status.md`.
  - Commit Boundary: Commit chính thức sau khi hoàn tất Slice P6A-2.

---

### P3: Slice P6A-3 — CLI Status Extension & UI Traceability Matrix Verification (`cmd/switcher/`)

- Phase Goal: Mở rộng CLI với cờ `antigravity-switch.exe --quota` xuất bảng phân bổ hạn ngạch toàn bộ tài khoản ra terminal Windows, xác thực tính tương thích dual-build (CLI thuần và GUI), đối soát toàn bộ UI Traceability Matrix, và đo đạc các chỉ số benchmark hiệu năng.
- Phase Boundary:
  - In scope: `cmd/switcher/main.go`, `cmd/switcher/app_cli.go`, `cmd/switcher/app_gui.go`.
  - Out of scope: Không sửa backend telemetry, không sửa GUI widgets.
  - Dependencies: Dữ liệu từ `internal/telemetry/` và `internal/ui/`.
- Phase Implementation Rule: Thực hiện tuần tự Step 1 (Code leaf tasks), Step 2 (Inspect & Lint), Step 3 (Update tests), Step 4 (Build), Step 5 (Mini QA), Step 6 (Acceptance & Commit).
- Ordered Slice List:
  - P3-A: CLI `--quota` Flag & UI Traceability Matrix Complete Verification.

- [ ] P3-A: CLI `--quota` Flag & UI Traceability Matrix Complete Verification.
  - Goal: Cung cấp giao diện dòng lệnh mạnh mẽ để kiểm tra hạn ngạch nhanh chóng mà không cần mở GUI, đồng thời nghiệm thu chất lượng bàn giao toàn diện cho Phase 6A.
  - Scope Boundary:
    - Editable: `cmd/switcher/main.go`, `cmd/switcher/app_cli.go`, `cmd/switcher/app_gui.go`.
    - Inspect-only: `internal/telemetry/`, `internal/ui/`.
    - Preserve-only: `internal/reloader/`, `internal/wincred/`, `internal/scanner/`.
    - Out of scope: Các gói internal khác.
  - Non-Goals: Không thay đổi hành vi của các cờ hiện tại (`--status`, `--list`, `--switch`, `--capture`).
  - Pre-flight Questions:
    - Data source: `telemetry.TelemetryPool` và `telemetry.QuotaCache`.
    - Display permission: Standard output terminal.
    - DB read flow: Đọc snapshot từ `accounts_store/.quota_cache.json` nếu có cờ offline, hoặc quét trực tiếp qua pool.
    - DB write flow: N/A.
    - Render location: `os.Stdout` qua `text/tabwriter`.
    - UI behavior flow: N/A.
    - Docker runtime: N/A.
    - Playwright target: N/A.
    - Behavior test: Kiểm thử chạy lệnh CLI với cờ `--quota` và xác thực định dạng bảng in ra.
    - Cleanup/quarantine: N/A.
    - External side effects: HTTP calls tới Google Quota API nếu cần quét trực tiếp.
    - N/A notes: Docker runtime và Playwright N/A.
  - Work Steps:
    1. Code (Execute leaf atomic tasks in core source files only):
       - Task 1.1: CLI `--quota` Flag & Tabular Formatter
         + Action: Thêm cờ `flagQuota = flag.Bool("quota", false, "Display detailed multi-account quota telemetry table")` vào `cmd/switcher/main.go`. Khi cờ được bật, gọi telemetry pool quét hạn ngạch và in bảng qua `tabwriter` với các cột: `EMAIL`, `CLAUDE 4.6`, `GEMINI 2.5`, `GPT-OSS`, `RESET TIME (UTC)`, `HEALTH STATUS`.
         + Inputs: CLI arguments, `telemetry.TelemetryPool`.
         + Allowed Edit Scope: `cmd/switcher/main.go`.
         + Outputs: Tabular console renderer cho `--quota`.
         + Verification Condition: `go vet ./cmd/switcher/...`
         + Task Checkpoint: `wip(P6A-3): task 1.1 - implement --quota cli flag and tabular console renderer`
       - Task 1.2: Dual-Build Target Sync
         + Action: Đồng bộ `cmd/switcher/app_cli.go` và `app_gui.go` để cờ `--quota` hoạt động nhất quán trên cả bản build CLI thuần và bản build Fyne GUI.
         + Inputs: Build tags `//go:build !gui` và `//go:build gui`.
         + Allowed Edit Scope: `cmd/switcher/app_cli.go`, `cmd/switcher/app_gui.go`.
         + Outputs: Dual-mode binary target.
         + Verification Condition: `go build ./cmd/switcher` và `go build -tags gui ./cmd/switcher`
         + Task Checkpoint: `wip(P6A-3): task 1.2 - ensure dual-build CLI/GUI compatibility for --quota`
    2. Code Inspection & Lint:
       - Kiểm tra diff, format mã, xác nhận cả 2 lệnh build đều pass mà không cảnh báo.
    3. Update Tests (Code first, test second):
       - Tạo unit/integration test trong `cmd/switcher/` kiểm tra cờ `--quota` in đúng bảng.
    4. Build:
       - Biên dịch:
         + `go build -o bin/antigravity-switch-cli.exe ./cmd/switcher`
         + `go build -tags gui -o bin/antigravity-switch.exe ./cmd/switcher`
    5. QA (Mini QA / Runtime Validation):
       - Chạy `bin/antigravity-switch-cli.exe --quota` trên terminal Windows thực tế.
       - Kiểm tra định dạng bảng, màu sắc, thời gian reset, và độ trễ phản hồi <= 2.5s.
       - Thực hiện đối soát 8/8 mục kiểm thử trong UI Traceability Matrix (`UI-TR-01` đến `UI-TR-08`).
       - Đo đạc RAM footprint (<= 20MB) và startup latency (<= 50ms).
       - Evidence target: `E3-P3A-CODE1..CODE2`, `E3-P3A-TEST1`, `E3-P3A-BENCH1`.
    6. Slice Acceptance & Official Milestone Commit:
       - Kiểm tra các tiêu chuẩn hoàn thành của P3-A, cập nhật `evidence.md`, cập nhật hàng `cmd/switcher/` từ `partial -> correct` trong `actual-status.md`, tạo commit:
         `feat(P6A-3): extend cli with --quota table and complete ui traceability matrix`.
  - Implementation Gate:
    - Slice P6A-1 và P6A-2 phải hoàn tất trước khi tiến hành nghiệm thu Slice P6A-3.
  - Acceptance:
    - Source: CLI in ra bảng hạn ngạch rõ ràng, sạch sẽ, không lỗi runtime.
    - Performance: RAM footprint <= 20MB, CLI startup <= 50ms.
    - Traceability: 8/8 mục trong UI Traceability Matrix được nghiệm thu đạt 100%.
    - Evidence IDs: `E3-P3A-CODE1..CODE2`, `E3-P3A-TEST1`, `E3-P3A-BENCH1`.
    - Actual-status rows refreshed: `cmd/switcher/main.go` chuyển sang `correct`.
  - Evidence Targets: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`.
  - Actual-status Update: `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-actual-status.md`.
  - Commit Boundary: Commit chính thức sau khi hoàn tất Slice P6A-3.

---

- [ ] P4-A: Call supervisor for the implemented-plan acceptance loop.
  - Goal: Đánh giá và nghiệm thu toàn diện công việc của Phase 6A đối soát với bản kế hoạch đã phê duyệt, ma trận actual-status, evidence ledger, benchmark ledger, và UI Traceability Matrix trước khi chuyển giao sang Phase 7.
  - Work Steps: Supervisor kiểm tra mã nguồn, chạy toàn bộ test suite, kiểm tra tính toàn vẹn của UI Traceability Matrix, xác nhận zero regression trên account switcher và reloader, và xác nhận kết quả nghiệm thu.
  - Implementation Gate: Cả 3 Slices P6A-1, P6A-2, P6A-3 đều phải có commit chính thức và evidence đầy đủ.
  - Acceptance: Báo cáo nghiệm thu Supervisor xác nhận đạt 100% tiêu chí Zero-Trust Definition of Done.

