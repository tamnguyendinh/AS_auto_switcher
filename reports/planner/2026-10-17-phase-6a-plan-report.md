# Báo Cáo Kế Hoạch Thực Thi: Phase 6A — Multi-Account Quota Telemetry Engine, Live Dashboard & Multi-Theme System

**Tài liệu:** `reports/planner/2026-10-17-phase-6a-plan-report.md`  
**Ngày thực hiện:** 2026-10-17  
**Làn đảm nhiệm:** Planner Specialist  
**Cơ quan chỉ đạo:** CEO Agent (Thread ID: `01a14378-4a08-77c1-88d5-666ad8c3592d`)  
**Thẩm quyền & Căn cứ:** Chỉ đạo trực tiếp từ Human Owner, AGENTS.md, Báo cáo Đặc tả Kiến trúc `reports/architect/2026-10-17-phase-6a-quota-telemetry-and-multi-theme-dashboard-architecture.md` đã được Human Owner trực tiếp thảo luận và phê duyệt.  
**Bộ Kế hoạch chi tiết đi kèm:** `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/`  
**Trạng thái bàn giao:** `[VERDICT: READY_FOR_ARCHITECT_REVIEW]`

---

## 1. Tóm tắt Kế hoạch & Mục tiêu chuyển hóa

Thực thi chỉ thị tối cao từ Human Owner ủy thác qua CEO Agent, Planner Lane đã hoàn tất việc chuyển hóa toàn văn Báo cáo Đặc tả Kiến trúc Phase 6A thành bộ Kế hoạch Thực thi hoàn chỉnh gồm 4 tệp chuẩn mực tại thư mục:
`docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/`

Kế hoạch giải quyết triệt để khuyết tật **"Blind Failover" (Chuyển đổi mù quáng)** của hệ thống failover tự động ban đầu, đặt nền móng kiến trúc vững chắc (Phase 6A) làm cầu nối giữa Phase P6 (Go Standalone Switcher) và Phase P7 (Autonomous Quota Failover Daemon). Nhờ đó, trước khi hệ thống kích hoạt tự động switch tài khoản khi gặp HTTP 429, bộ điều khiển đã nắm chắc 100% hạn ngạch thời gian thực và thời điểm Google reset quota của từng tài khoản.

---

## 2. Bảng Phân Rã 3 Slices Thực Thi Độc Lập (Execution Slices)

Kế hoạch được phân rã thành 3 Slices thực thi tuần tự, độc lập và có ranh giới Touch-Map khóa chặt:

| Slice ID | Tên Slice & Trọng tâm | Touch-Map (Khóa chặt Scope) | Tiêu chí Nghiệm thu (Zero-Trust Definition of Done) | Non-Goals (Ranh giới cấm vượt) |
| :---: | :--- | :--- | :--- | :--- |
| **P6A-1** | **Quota Telemetry Engine & 60s Auto-Loop**<br>• Package `internal/telemetry/`<br>• Client Google `v1internal:retrieveUserQuota`<br>• Parallel Worker Pool quét $N$ tài khoản<br>• Two-Tier Cache (RAM + Đĩa)<br>• 60s Auto-Telemetry Loop ngầm | `internal/telemetry/client.go` *(mới)*<br>`internal/telemetry/cache.go` *(mới)*<br>`internal/telemetry/pool.go` *(mới)*<br>`internal/telemetry/loop.go` *(mới)*<br>`internal/telemetry/telemetry_test.go` *(mới)* | • Quét song song toàn bộ tài khoản trong `accounts_store/` với tổng độ trễ < 2.0s (thực nghiệm: 1.69s).<br>• Bóc tách chuẩn xác hạn ngạch Claude 4.6 (Opus/Sonnet), Gemini 2.5 Pro/Flash, GPT-OSS và thời điểm reset UTC.<br>• Single-flight concurrency gate bằng `atomic.Bool` drop 100% request trùng lặp khi chu kỳ 60s bị trễ.<br>• Test suite đạt coverage $> 90\%$, `go test -race` đạt 0 lỗi data race. | CẤM sửa `internal/ui/`, CẤM sửa `cmd/switcher/`, CẤM tự ý kích hoạt chuyển đổi tài khoản (thuộc Phase 7). Không can thiệp mã nguồn reloader hay wincred. |
| **P6A-2** | **Fyne UI Quota Dashboard & Multi-Theme**<br>• Mini Quota Progress Badges 3 màu<br>• Nút Làm mới Quota Tất cả trên Toolbar<br>• Nút `[🔄]` làm mới từng dòng tài khoản<br>• Hệ thống Đa Giao diện (Dark, Light, Warm)<br>• Cơ chế Dispatch phi khóa duy trì 60 FPS | `internal/ui/themes.go` *(mới)*<br>`internal/ui/quota_badge.go` *(mới)*<br>`internal/ui/widgets.go`<br>`internal/ui/window.go`<br>`internal/ui/app.go`<br>`internal/ui/ui_test.go`<br>`internal/ui/themes_test.go` *(mới)* | • Mini Quota Badges hiển thị chính xác trên từng Card với 3 mức màu: Xanh (>50%), Vàng (15-50%), Đỏ (<15%) kèm đếm ngược giờ hồi phục.<br>• Header Toolbar có nút `[📊 Làm mới Quota Tất cả]`, nhãn chu kỳ `Tự động: 60s`, và bộ chọn theme `[🎨 Giao diện: Tối | Sáng | Ấm]`.<br>• Chuyển đổi qua lại giữa Dark (Catppuccin Mocha), Light (Paper Slate), Warm (Amber/Sepia) tức thì (< 50ms) không crash app, lưu preference vào `accounts_store/.ui_preferences.json`.<br>• Thỏa mãn 100% Bảng kiểm tra truy vết giao diện (`UI-TR-01` đến `UI-TR-08`). | CẤM thay đổi kích thước cửa sổ chuẩn (440x560), CẤM gọi Fyne API ngoài main event thread, CẤM làm chậm logic switch tài khoản hiện có. |
| **P6A-3** | **CLI Status Extension & UI Traceability QA**<br>• Lệnh `antigravity-switch.exe --quota`<br>• Bảng xuất Tabwriter ra Terminal Windows<br>• Đồng bộ hóa Dual-Build Target<br>• Nghiệm thu toàn diện UI Matrix & Benchmarks | `cmd/switcher/main.go`<br>`cmd/switcher/app_cli.go`<br>`cmd/switcher/app_gui.go` | • Lệnh CLI `--quota` in ra bảng phân bổ hạn ngạch trực quan gồm: EMAIL, CLAUDE 4.6, GEMINI 2.5, GPT-OSS, RESET TIME (UTC), HEALTH STATUS.<br>• Biên dịch thành công cả 2 target: CLI thuần và GUI.<br>• Bộ nhớ RAM <= 20MB, cold startup CLI <= 50ms.<br>• Toàn bộ 8 tiêu chí `UI-TR-01` .. `UI-TR-08` được đối soát và pass 100%. | Không thay đổi hành vi của các cờ hiện tại (`--status`, `--list`, `--switch`, `--capture`). |

---

## 3. Khóa Chặt Bảng Kiểm Tra Truy Vết Giao Diện (UI Traceability Matrix)

Mọi yêu cầu giao diện từ Kiến trúc sư trưởng đã được chuyển thành các điều kiện nghiệm thu bất di bất dịch trong Kế hoạch:

| Mã Kiểm Thử | Tên Phần Tử Giao Diện | Vị Trí Bố Cục | Tiêu Chuẩn Trực Quan & Hành Vi Nghiệm Thu | Mức Độ Nghiêm Trọng Khi Vi Phạm |
|---|---|---|---|---|
| **UI-TR-01** | Global Quota Refresh Button | Header Toolbar | Nút `[📊 Làm mới Quota Tất cả]`. Bấm vào kích hoạt worker pool quét song song toàn bộ tài khoản, hiển thị trạng thái loading, cập nhật badges khi hoàn thành. | **BLOCKER** |
| **UI-TR-02** | Row-Level Single Refresh Button | Mỗi dòng Account Card | Nút icon `[🔄]` nằm liền kề email. Bấm vào chỉ kích hoạt quét hạn ngạch duy nhất của tài khoản đó. | **BLOCKER** |
| **UI-TR-03** | Auto-Refresh 60s Indicator | Header Toolbar / Status | Nhãn hiển thị chu kỳ `Tự động: 60s` và thời điểm cập nhật gần nhất (`HH:mm:ss`). Tự động làm mới dữ liệu ngầm sau mỗi 60 giây. | **BLOCKER** |
| **UI-TR-04** | Mini Quota Progress Badges | Mỗi dòng Account Card | Hiển thị 3 huy hiệu phần trăm cho Claude, Gemini, GPT với quy tắc 3 màu chuẩn:<br>• Xanh (`#22c55e` / `#a6e3a1`): $> 50\%$<br>• Vàng (`#f59e0b` / `#f9e2af`): $15\% - 50\%$<br>• Đỏ (`#ef4444` / `#f38ba8`): $< 15\%$ | **BLOCKER** |
| **UI-TR-05** | Quota Recovery Countdown Text | Dưới Badges (khi quota $< 15\%$) | Dòng chữ đếm ngược: `Google sẽ hồi phục lúc: HH:mm (còn Xh Yp)` hiển thị chính xác theo timestamp UTC mà Google cung cấp. | **HIGH** |
| **UI-TR-06** | Multi-Theme Switcher Control | Window Header | Bộ chọn 3 theme: `Tối (Dark)`, `Sáng (Light)`, `Ấm (Warm)`. Chuyển đổi mượt mà ngay trên Canvas Fyne mà không cần restart app. | **BLOCKER** |
| **UI-TR-07** | Warm Mode Color Palette | Theme Engine | Warm Theme áp dụng bảng màu amber/sepia (`#fbf0d9` / `#eddcc1` / `#d97706`), lọc triệt để ánh sáng xanh có hại cho mắt lập trình viên. | **HIGH** |
| **UI-TR-08** | Non-Blocking UI Invariant | Toàn bộ vòng đời GUI | Giao diện hoàn toàn không bị đông cứng (freezing) khi đang tải mạng. Luôn duy trì tương tác và cuộn mượt 60 FPS nhờ cơ chế async goroutine và `fyne.Do()`. | **BLOCKER** |

---

## 4. Bằng Chứng Thực Nghiệm & Hiện Trạng Codebase Đối Soát (Empirical Evidence)

1. **Khả thi 100% của Google Quota Endpoint (`E0-P0A-SPEC1`):**
   - Đã kiểm tra thực nghiệm quét đồng thời 7/7 tài khoản đang lưu trong `accounts_store/`.
   - Kết quả: Phản hồi `HTTP 200 OK` 100%, bóc tách trọn vẹn hạn ngạch 3 nhóm model và thời điểm reset UTC trong vòng **1.69 giây**!
2. **Hạ tầng Refresh Token đã sẵn sàng (`E0-P0A-SRC2`):**
   - Module `internal/connector/oauth_refresher.go` đã có sẵn `GoogleOAuthRefresher` đã được kiểm thử pass 100%, cấp mới token trong ~500ms khi token hết hạn.
3. **Môi trường Test Suite hiện tại sạch 100% (`E0-P0A-TEST1`):**
   - Lệnh `go test ./...` trên toàn bộ 8 gói phần mềm hiện có đều pass (connector, manager, model, reloader, scanner, store, ui, wincred), không có lỗi tồn đọng.
4. **Touch-Map được cách ly tuyệt đối:**
   - Các gói nhạy cảm như `internal/reloader/` (reloader & thread preservation) và `internal/wincred/` (Win32 Credential Manager) được bảo vệ tuyệt đối ở chế độ **Preserve-Only**.

---

## 5. Danh Mục Hồ Sơ Kế Hoạch Đã Lập (Standard Plan Set)

Toàn bộ 4 tệp kế hoạch chuẩn theo đặc tả của Skill Planner đã được thiết lập đầy đủ tại:
- `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-plan.md`
- `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-actual-status.md`
- `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-evidence.md`
- `docs/plans/2026-10-17-phase-6a-quota-telemetry-dashboard/2026-10-17-phase-6a-quota-telemetry-dashboard-benchmark.md`
- Báo cáo tổng kết tại: `reports/planner/2026-10-17-phase-6a-plan-report.md`

Mỗi checklist item trong kế hoạch đều tuân thủ nghiêm ngặt chu kỳ thực thi 6 bước (`Code → Inspect → Update tests → Build → QA → Acceptance/Commit`), trong đó Bước 1 (Code) được bẻ gãy thành các leaf atomic task có Action, Inputs, Edit Scope, Outputs, Điều kiện kiểm chứng và Task Checkpoint micro WIP rõ ràng.

---

## 6. Khuyến Nghị Chuyển Giao & Điều Phối Tiếp Theo

1. Hồ sơ Kế hoạch Phase 6A đã hoàn chỉnh 100%, sẵn sàng cho CEO Agent chuyển tiếp sang **Architect Review Lane** hoặc phê duyệt cho **Coder Lane** thi công.
2. Thứ tự thi công khuyến nghị:
   - **Bước 1: Slice P6A-1** thi công phần lõi `internal/telemetry/` (Client, Pool, Cache, 60s Loop) và nghiệm thu với unit test mock server.
   - **Bước 2: Slice P6A-2** thi công giao diện Fyne UI (`themes.go`, `quota_badge.go`, toolbar, card) và đối soát UI Traceability Matrix.
   - **Bước 3: Slice P6A-3** bổ sung cờ `--quota` cho CLI, kiểm tra dual-build, đo đạc benchmark và bàn giao tổng thể.
