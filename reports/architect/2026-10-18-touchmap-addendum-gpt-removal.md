# Architectural Decision & Touch-Map Addendum: GPT Removal & Test Suite Governance

**Document ID:** `ADR-20261018-GPT-REMOVAL`  
**Date:** 2026-10-18  
**Author:** Architect Specialist Lane  
**Recipient:** CEO Session (`01a14378-4a08-77c1-88d5-666ad8c3592d`) / Coder Lane (`01a14e00-dc5b-7481-8460-a61eaf27be56`) / Supervisor Lane  
**Authority:** AGENTS.md, Chỉ thị CEO, Báo cáo Supervisor R2 `reports/supervisor/2026-10-18-ui-redesign-r2-supervisor-report.md` (commit `9b4edb2`), Handoff FAST BLOCK từ Coder  
**Status:** APPROVED / OFFICIAL ARCHITECTURAL DIRECTIVE  
**Verdict:** `[VERDICT: PASS / UNBLOCKED_FOR_CODER]`  

---

## 1. Executive Summary & Root Cause of FAST BLOCK

### 1.1 Bối Cảnh Sự Cố (FAST BLOCK)
Trong vòng sửa lỗi UI Redesign F1–F6 (Round 2), Supervisor đã từ chối nghiệm thu (**REJECT**) tại commit `9b4edb2` với lý do:
- **Finding F2 (Chưa đóng):** Coder mới chỉ gán tag `json:"-"` cho trường `GPT` trong `AccountQuotaData`, nhưng trường này vẫn tồn tại trong Go struct contract; reflection (`reflect.TypeOf(*data).FieldByName("GPT")`) vẫn phát hiện trường; REST client vẫn gán giá trị vào `data.GPT`.
- **Finding F1 (Chưa đóng):** Khi payload thiếu bucket hoặc rỗng, hệ thống vẫn dựng quota giả lập (synthesize) thay vì giữ đúng tính trung thực của dữ liệu (`-1` hoặc `--%`).
- **Finding F4 (Chưa đóng):** Còn các UI object cũ (`headerBg`, `statusBarLabel`, `activeStatusLabel`, `activeQuotaLabel`, `NewQuotaCountdownLabel` stub) vẫn được khởi tạo và cập nhật ngầm dù không gắn vào render tree.

Khi Coder nhận lệnh sửa triệt để F2 bằng cách xóa hẳn trường `GPT` khỏi `AccountQuotaData` trong `internal/telemetry/client.go`, Coder gặp phải các rào cản:
1. `internal/telemetry/telemetry_test.go:129`, `:186` bị lỗi compile do assert trực tiếp `data.GPT.RemainingFraction`. Tệp này nằm ngoài Touch-Map hiện tại của Coder.
2. Các tệp probe cũ của Supervisor (`reports/supervisor/evidence/2026-10-18-ui-redesign/runtime_probe.go` và `r2/runtime_probe.go`) chứa tham chiếu tĩnh tới `AccountQuotaData.GPT`.
3. Coder dừng khẩn cấp (FAST BLOCK) để xin quyết định kiến trúc về:
   - Thẩm quyền sửa đổi các tệp test liên quan (`telemetry_test.go`).
   - Ranh giới sở hữu đối với các tệp probe trong `reports/supervisor/evidence/`.

---

## 2. Thẩm Định Tác Động Của Việc Xóa Trường `GPT` (Blast Radius Analysis)

Sau khi kiểm tra toàn bộ đồ thị phụ thuộc mã nguồn và test suite bằng Anvien và AST search, Architect xác định phạm vi ảnh hưởng (blast radius) như sau:

### 2.1 Bản Chất Miền Nghiệp Vụ (Domain Truth)
- API upstream của Google Cloud Code / Language Server (`RetrieveUserQuotaSummary`) hiện chỉ trả về 2 nhóm model chính:
  1. `Gemini Models` (chứa `gemini-5h`, `gemini-weekly`).
  2. `Claude and GPT models` (chứa `3p-5h`, `3p-weekly` — thực chất ánh xạ vào các model bên thứ ba như Claude 4.6).
- Hoàn toàn KHÔNG tồn tại bucket hạn ngạch riêng biệt cho GPT trong upstream.
- Bản vẽ thiết kế của Owner (`reports/problem/images/9180d29a.jpg`) và đặc tả giao diện (`reports/architect/2026-10-17-ui-redesign-spec-from-owner-drawing.md`) chỉ quy định hiển thị 2 cột: Claude và Gemini (mỗi loại 2 chu kỳ: 5 Giờ và Hàng Tuần).
- Việc loại bỏ triệt để trường `GPT` khỏi contract `AccountQuotaData` là hoàn toàn chính xác về mặt kiến trúc, giúp xóa bỏ dữ liệu giả, giảm gánh nặng lưu vết và đáp ứng 100% Invariant F2.

### 2.2 Tác Động Chi Tiết Tới Các Phân Hệ Mã Nguồn

| Phân hệ / Tệp tin | Hiện trạng tham chiếu | Tác động khi xóa `GPT` | Giải pháp kiến trúc bắt buộc |
|---|---|---|---|
| `internal/telemetry/client.go` | Định nghĩa `AccountQuotaData.GPT`, gán `GPT: gpt` trong `RetrieveUserQuota` | Trực tiếp bị xóa | Xóa trường `GPT` khỏi struct `AccountQuotaData`. Xóa dòng gán `GPT: gpt`. Giữ nguyên hằng số `StatusLowGPT` để tương thích trạng thái phân loại khi cần. |
| `internal/telemetry/telemetry_test.go` | Dòng 129, 186 assert `data.GPT`; dòng 442, 1140 khởi tạo struct literal `GPT: ...` | Lỗi compile Go: `data.GPT undefined` và `unknown field GPT` | **Cho phép bổ sung vào Touch-Map**: Xóa các assertion `data.GPT` trên `AccountQuotaData`, xóa field `GPT` trong struct literals. |
| `internal/telemetry/quota_test.go` | Dòng 106 khởi tạo struct literal `GPT: ...` trong test serialization | Lỗi compile Go | (Đã có trong Touch-Map): Xóa field `GPT` trong struct literal; cập nhật test kiểm tra reflection khẳng định trường `GPT` không còn tồn tại (`present == false`). |
| `cmd/switcher/main.go` | Dòng 306 đọc `q.GPT.RemainingFraction`; dòng 315 kiểm tra `q.GPT.RemainingFraction < 0.15` | Lỗi compile Go: `q.GPT undefined` | **Cho phép bổ sung vào Touch-Map**: Xóa khối `if q.GPT.RemainingFraction >= 0` (để `gptStr` giữ nguyên mặc định `"N/A"` hoặc `"--%"`), xóa điều kiện `q.GPT.RemainingFraction < 0.15`. Giữ nguyên header CLI để bảo toàn định dạng bảng tương thích ngược. |
| `cmd/switcher/main_test.go` | Dòng 163, 173, 183, 242 chứa struct literal `GPT: telemetry.ModelQuotaBucket{...}` | Lỗi compile Go: `unknown field GPT in struct literal` | **Cho phép bổ sung vào Touch-Map**: Xóa trường `GPT:` khỏi các composite literal của `telemetry.AccountQuotaData`. |
| `internal/ui/*` | Đã được Coder dọn dẹp sạch sẽ ở vòng 2; chỉ có test xác minh không có chữ GPT | Không bị ảnh hưởng | Tiếp tục duy trì trạng thái sạch, không phát sinh thay đổi struct. |

---

## 3. Ban Hành Addendum Touch-Map Chính Thức

Căn cứ thẩm quyền Kiến trúc sư trưởng và yêu cầu khẩn cấp tháo gỡ cho Coder, ban hành quyết định mở rộng Touch-Map như sau:

### 3.1 Danh Mục Touch-Map Hợp Nhất (Authorized Write Scope for Coder)
Coder được phép sửa đổi duy nhất các tệp tin dưới đây để đóng toàn bộ các finding F1, F2, F4:

1. **Nhóm GUI & Presentation (F1, F3, F4, F5, F6):**
   - `internal/ui/app.go` (Dọn dẹp dead UI objects: `headerBg`, `statusBarLabel`, `activeStatusLabel`, `activeQuotaLabel`).
   - `internal/ui/window.go` (Xử lý layout/scroll view).
   - `internal/ui/widgets.go` (Render dual-cycle badge).
   - `internal/ui/quota_badge.go` (Xử lý chu kỳ độc lập 5h/week, fallback missing bucket `--%`, loại bỏ stub countdown).
   - `internal/ui/ui_test.go` (Cập nhật suite UI test).
   - `cmd/switcher/app_gui.go` (Bảo toàn GUI entrypoint).

2. **Nhóm Telemetry & Contract (F1, F2):**
   - `internal/telemetry/client.go` (Xóa trường `GPT` khỏi `AccountQuotaData`, xóa gán REST, xử lý parsing trung thực).
   - `internal/telemetry/pool.go` (Xử lý lưu trữ raw buckets và fallback).
   - `internal/telemetry/quota_test.go` (Regression test cho dual-cycle và zero-GPT reflection).
   - `internal/connector/api_client.go` (Xử lý lỗi I/O nếu cần).

3. **Nhóm Mới Được Ủy Quyền Bổ Sung (ADDENDUM AUTHORIZATION):**
   - **`internal/telemetry/telemetry_test.go`** — Chỉ được phép: xóa các assertion `data.GPT` và loại bỏ field `GPT` trong composite struct literals của `AccountQuotaData`. Tuyệt đối không thay đổi logic kiểm thử các nhánh khác.
   - **`cmd/switcher/main.go`** — Chỉ được phép: gỡ bỏ truy cập `q.GPT` tại dòng 306 và 315; giữ biến `gptStr = "N/A"` để bảo toàn giao diện CLI.
   - **`cmd/switcher/main_test.go`** — Chỉ được phép: gỡ bỏ trường `GPT:` trong các struct literal của `AccountQuotaData`.

### 3.2 Ranh Giới Nghiêm Cấm (Strict Non-Goals)
- Tuyệt đối **CẤM** Coder chỉnh sửa bất kỳ tệp tin nào ngoài danh mục 13 tệp trên (như `internal/manager/*`, `internal/store/*`, `internal/model/*`, `internal/scanner/*`, `internal/reloader/*`, `internal/wincred/*`).
- Tuyệt đối **CẤM** Coder đổi tên cột hoặc cấu trúc xuất CLI ngoài việc bỏ đọc dữ liệu trường đã xóa.

---

## 4. Quy Định Ranh Giới Sở Hữu Đối Với Các Tệp Probe & Evidence

Về thắc mắc của Coder liên quan đến các probe cũ của Supervisor:

### 4.1 Tính Bất Biến Tuyệt Đối Của Artifact Supervisor (Supervisor Immutability Law)
- Tất cả các tệp trong `reports/supervisor/evidence/*` (ví dụ `reports/supervisor/evidence/2026-10-18-ui-redesign/runtime_probe.go` và `r2/runtime_probe.go`) là **bằng chứng pháp lý - kỹ thuật độc lập bất biến** của Supervisor gắn liền với các commit lịch sử (`fe8ffbb`, `9b4edb2`).
- **LỆNH CẤM:** Coder tuyệt đối **KHÔNG ĐƯỢC PHÉP** chỉnh sửa, xóa, ghi đè hoặc di dời bất kỳ tệp nào trong `reports/supervisor/`.
- Mọi hành vi sửa đổi bằng chứng của Supervisor đều vi phạm nghiêm trọng tính độc lập kiểm định (Independence of Verification) và sẽ bị Supervisor REJECT ngay lập tức.

### 4.2 Cơ Chế Cách Ly Biên Dịch (`//go:build ignore`)
- Cả hai tệp probe của Supervisor đều có directive `//go:build ignore` ở đầu file.
- Do đó:
  - Chúng **KHÔNG** tham gia vào quá trình build hệ thống (`go build ./...`).
  - Chúng **KHÔNG** tham gia vào test suite chuẩn (`go test ./...`).
  - Chúng **KHÔNG** bị kiểm tra bởi `staticcheck -tags gui ./...`.
- Việc các file probe cũ chứa tham chiếu tĩnh `AccountQuotaData.GPT` **HOÀN TOÀN KHÔNG GÂY LỖI BUILD/TEST** cho repository.

### 4.3 Quy Chuẩn Kiểm Thử Của Coder (Coder Verification Standard)
- Coder **KHÔNG ĐƯỢC CHẠY** các probe cũ của Supervisor làm điều kiện nghiệm thu cá nhân nếu probe đó assert contract cũ.
- Khi Coder cần tạo script probe để tự kiểm chứng thực nghiệm trước khi handoff:
  - Coder **PHẢI** tạo probe trong không gian sở hữu của Coder:  
    `reports/coder/evidence/2026-10-18-ui-redesign-f1-f6/` (ví dụ `coder_probe.go`).
  - Tệp probe này phải có `//go:build ignore` và thuộc Touch-Map bằng chứng của Coder.
- **Nguồn Sự Thật Tối Cao (Source of Truth):**
  - Thước đo chất lượng cao nhất của Coder là các test suite chính thống trong repository:
    ```powershell
    go test -v ./internal/telemetry/...
    go test -v ./cmd/switcher/...
    go test -v ./internal/ui/...
    go test ./...
    staticcheck -tags gui ./...
    go build ./...
    go build -tags gui -ldflags "-H=windowsgui" -o antigravity-switch.exe ./cmd/switcher
    ```
- Khi Supervisor bước vào Vòng 3 (Round 3), chính Supervisor sẽ tự tạo thư mục bằng chứng mới (`reports/supervisor/evidence/2026-10-18-ui-redesign-r3/`) và cập nhật script probe độc lập phù hợp với contract mới (không còn `GPT`).

---

## 5. Hướng Dẫn Kỹ Thuật Hành Động Cho Coder Để Đóng F1, F2, F4

Để Coder hoàn thành công việc một cách trơn tru và đạt `[VERDICT: PASS]` từ Supervisor tại Vòng 3:

### 5.1 Đóng F2 (Zero GPT Contract & Serialization)
1. Trong `internal/telemetry/client.go`:
   - Xóa bỏ trường `GPT ModelQuotaBucket` khỏi `AccountQuotaData`.
   - Xóa bỏ việc gán `GPT: gpt` trong hàm `RetrieveUserQuota`.
2. Trong `internal/telemetry/quota_test.go`:
   - Cập nhật test `TestAccountQuotaData_ZeroGPTSerialization`:
     Kiểm tra bằng reflection:
     ```go
     field, present := reflect.TypeOf(AccountQuotaData{}).FieldByName("GPT")
     if present {
         t.Fatalf("CRITICAL VIOLATION: struct AccountQuotaData still contains GPT field: %+v", field)
     }
     ```
3. Trong `internal/telemetry/telemetry_test.go`, `cmd/switcher/main.go`, `cmd/switcher/main_test.go`:
   - Loại bỏ các tham chiếu tới `.GPT` theo Touch-Map Addendum đã cấp ở Mục 3.

### 5.2 Đóng F1 (Truthful Quota & Independent Dual-Cycle)
1. Đảm bảo khi payload API rỗng hoặc thiếu bucket fraction (`-1`):
   - Không được tự ý gán fraction = 1.0 (100%) hoặc fraction = 0 giả tạo.
   - Giao diện UI (`quota_badge.go`) phải format thành `"--%"` khi fraction là âm (`< 0`).
   - Cờ `HasData` chỉ bật `true` khi thực sự có ít nhất một bucket hợp lệ.

### 5.3 Đóng F4 (Dead UI Object Cleanup)
1. Trong `internal/ui/app.go` và `internal/ui/window.go`:
   - Xóa bỏ hoàn toàn các trường và biến không gắn vào render tree: `headerBg`, `statusBarLabel`, `activeStatusLabel`, `activeQuotaLabel`.
   - Xóa bỏ các lệnh cập nhật và đổi màu cho các object này trong `updateUI` và `SetTheme`.
   - Dọn dẹp stub hàm `NewQuotaCountdownLabel` (loại bỏ hoàn toàn nếu không còn caller thực tế, hoặc dọn callers khỏi test).
2. Đảm bảo chạy `staticcheck -tags gui ./...` đạt **0 diagnostics**.

---

## 6. Lệnh Điều Phối Handoff (CEO Directive Forwarding)

1. Văn kiện này chính thức tháo gỡ trạng thái **FAST BLOCK** của Coder.
2. CEO được khuyến nghị chuyển tiếp ngay lập tức Addendum Touch-Map này cho Coder để Coder tiến hành chỉnh sửa mã nguồn theo đúng phạm vi phân công.
3. Architect hoàn thành nhiệm vụ thẩm định, không can thiệp vào mã nguồn sản phẩm, và tiến hành commit văn kiện này vào Git.
