# Báo Cáo Quyết Định Kiến Trúc & Addendum Touch-Map: Xử Lý FAST BLOCK Slice P1-C

- **Mã văn kiện**: `ADR-20261020-P1C-FAST-BLOCK-RESOLUTION`
- **Tác giả**: System Architect Lane
- **Ngày ban hành**: 2026-10-20
- **Trạng thái**: **APPROVED / ACTIVE**
- **Đối tượng áp dụng**: Coder Lane (Slice P1-C), Supervisor Lane, CEO Lane
- **Hồ sơ tham chiếu**:
  - Báo động FAST BLOCK từ Coder tại Slice P1-C (`internal/ui` & `cmd/switcher`)
  - Kế hoạch triển khai: `docs/plans/2026-10-20-autonomous-account-switching/2026-10-20-autonomous-account-switching-plan.md`
  - Quy chuẩn kiểm thử: `nguyen-ly-test-file-tach-biet` & `AGENTS.md`
  - Script build Windows: `scripts/build_windows.ps1`

---

## 1. Tóm Tắt & Phán Quyết Kiến Trúc (Executive Summary & Verdict)

- **Phán quyết**: **[VERDICT: READY_FOR_CODER]**
- **Phương án lựa chọn**: **Phương án A — Ban hành Addendum bổ sung `internal/manager/account_manager_test.go` vào Touch-Map của Slice P1-C**.
- **Mục tiêu**: Cho phép Coder bổ sung mock `rel.ServerDiscoverer` cho 3 unit tests cũ trong `internal/manager/account_manager_test.go` nhằm khôi phục tính hermetic (cô lập 100% khỏi môi trường tiến trình OS bên ngoài), giúp pass hoàn toàn lệnh build & test bắt buộc `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`.
- **Ranh giới bất biến (Invariants)**:
  - **TUYỆT ĐỐI KHÔNG** sửa mã nguồn sản phẩm (`internal/manager/account_manager.go` hoặc các file core khác ngoài Touch-Map).
  - **CHỈ ĐƯỢC PHÉP** tiêm mock `rel.ServerDiscoverer` vào đúng 3 test function đã định danh.
  - **KHÔNG ĐƯỢC PHÉP** nới lỏng hoặc sửa đổi script `scripts/build_windows.ps1` để bỏ qua kiểm thử core packages.

---

## 2. Khảo Sát Nguyên Nhân Gốc Rễ (Root Cause Analysis)

### 2.1. Triệu chứng lỗi
Khi chạy lệnh build & test tích hợp bắt buộc của dự án:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
```
Giai đoạn Pre-flight pure Go verification chạy `go test ./internal/manager/...` và thất bại với 3 unit test:
1. `TestSwitchAccount` (`account_manager_test.go:324`)
2. `TestSwitchAccount_AtomicAppStorageSync` (`account_manager_test.go:781`)
3. `TestSwitchAccount_CivilizedReAuth_HotSwapped` (`account_manager_test.go:880`)

Thông báo lỗi cụ thể:
```text
SwitchAccount failed: failed recycling language server: failed discovering respawned language_server.exe: language_server.exe process not found or not ready
```

### 2.2. Cơ chế kích hoạt lỗi trong mã nguồn
1. Trong `internal/manager/account_manager.go:490-505`, hàm `SwitchAccount` thực hiện tái khởi động language server qua `rel.RecycleLanguageServer(ctx, waitDuration)`.
2. Trong `internal/reloader/session_reloader.go:128-132`, hàm `RecycleLanguageServer` thực hiện:
   ```go
   if killer == nil {
       killer = DefaultKillLanguageServer
   }
   if discoverer == nil {
       discoverer = connector.DiscoverLanguageServer
   }
   if sleeper == nil {
       sleeper = defaultSleep
   }
   ```
3. Sau khi kill tiến trình và sleep thời gian chờ, hàm gọi `info, err := discoverer()`.
4. Nếu `r.ServerDiscoverer` là `nil` (chưa được mock), nó rơi vào hàm mặc định `connector.DiscoverLanguageServer()`. Hàm này quét bảng tiến trình và cổng TCP thực tế trên hệ điều hành Windows để tìm `language_server.exe`.
5. Trong môi trường kiểm thử (CI hoặc máy phát triển không chạy Antigravity IDE thực tế), `connector.DiscoverLanguageServer()` trả về lỗi: `language_server.exe process not found or not ready`.
6. Hậu quả là `RecycleLanguageServer` trả về lỗi và `SwitchAccount` bị hủy bỏ.

### 2.3. Đánh giá tính cô lập & tiền lệ trong codebase
- Trong cùng file `internal/manager/account_manager_test.go`, tất cả các unit test khác như:
  - `TestAccountManager_Workflow` (dòng 131)
  - `TestSwitchAccount_WincredFailure` (dòng 506)
  - `TestSwitchAccount_QuotaNotification` (dòng 576)
  - `TestSwitchAccount_ActiveThreads` (dòng 937)
  - `TestSwitchAccount_ContextCancelled` (dòng 1012)
  - `TestSwitchAccount_AppStorageSyncError` (dòng 1062)
  - Ngay cả các kịch bản phụ sau dòng 382 & 402 của chính hàm `TestSwitchAccount`!
  đều đã chủ động mock `rel.ServerDiscoverer = func() (*connector.ServerInfo, error) { ... }`.
- Việc 3 test `TestSwitchAccount` (đoạn đầu), `TestSwitchAccount_AtomicAppStorageSync`, và `TestSwitchAccount_CivilizedReAuth_HotSwapped` chỉ mock `rel.ProcessKiller` và `rel.Sleeper` nhưng bỏ sót mock `rel.ServerDiscoverer` là một lỗ hổng thiết lập test fixture cũ (test harness oversight), khiến test bị phụ thuộc vào môi trường OS thật (rò rỉ side-effect).
- Mã nguồn sản phẩm của Slice P1-C (`internal/ui/app.go`, `cmd/switcher/main.go`) hoàn toàn không gây ra lỗi này; việc thực thi `build_windows.ps1 -Mode all` là bước nghiệm thu toàn diện nên đã phát hiện ra lỗ hổng kiểm thử cũ này.

---

## 3. Thẩm Định Hai Phương Án Xử Lý

### Phương án A: Ban hành Addendum bổ sung `internal/manager/account_manager_test.go` vào Touch-Map để Coder mock `rel.ServerDiscoverer` đồng bộ
- **Bản chất**: Trực tiếp sửa chữa fixture của 3 test bị rò rỉ môi trường, đưa test suite trở về trạng thái thuần hermetic theo chuẩn Go unit test.
- **Ưu điểm**:
  - Giải quyết triệt để tận gốc (root-cause resolution).
  - Không thay đổi bất kỳ dòng mã nghiệp vụ sản phẩm nào.
  - Đảm bảo toàn bộ lệnh `go test ./...` và `build_windows.ps1` chạy độc lập, xanh 100% trên mọi máy phát triển và môi trường CI/CD không có Antigravity đang chạy.
  - Phù hợp hoàn toàn với nguyên tắc kiểm thử độc lập (`nguyen-ly-test-file-tach-biet`).
- **Nhược điểm**: Cần mở rộng Touch-Map thêm 1 file test ngoài phạm vi ban đầu của Slice P1-C.
- **Đánh giá rủi ro**: Rủi ro = 0 đối với mã runtime sản phẩm.

### Phương án B: Quy định phân tách điều kiện kiểm thử script build theo đúng nguyên tắc build isolation
- **Bản chất**: Điều chỉnh `scripts/build_windows.ps1` để bỏ bớt package `internal/manager/...` hoặc thêm cờ skip tests khi build binary P1-C.
- **Ưu điểm**: Giữ nguyên Touch-Map hẹp ban đầu của P1-C.
- **Nhược điểm & Rủi ro**:
  - Che giấu lỗi kiểm thử thật trong package cốt lõi (`internal/manager`).
  - Vi phạm nguyên tắc bảo toàn chất lượng trước khi release (`build_windows.ps1` đóng vai trò là release gate).
  - Khiến các đợt chạy test sau này hoặc lập trình viên khác chạy `go test ./...` vẫn tiếp tục bị fail nếu không có tiến trình language server đang chạy trên máy.
- **Đánh giá kiến trúc**: **Bác bỏ**. Không được giải quyết lỗi test bằng cách hạ thấp tiêu chuẩn kiểm thử của build script.

---

## 4. Quyết Định Kiến Trúc Chính Thức

1. **CHẤP THUẬN PHƯƠNG ÁN A**:
   Phê chuẩn giải pháp bổ sung mock `rel.ServerDiscoverer` vào 3 test hàm trong `internal/manager/account_manager_test.go`.
2. **GIỮ NGUYÊN NGUYÊN TẮC BUILD GATE**:
   Lệnh `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all` tiếp tục là điều kiện tiên quyết bắt buộc để hoàn tất Slice P1-C, không bị nới lỏng hay cắt xén.

---

## 5. Addendum Touch-Map Cho Coder (Slice P1-C)

### 5.1. Bảng phân định Touch-Map cập nhật

| Đường dẫn tệp tin | Vai trò / Quyền hạn | Giới hạn & Ràng buộc |
|---|---|---|
| `internal/ui/app.go` | **Editable (Cũ)** | Triển khai binding controller, lifecycle |
| `internal/ui/window.go` | **Editable (Cũ)** | Triển khai callback giao diện |
| `cmd/switcher/main.go` | **Editable (Cũ)** | Triển khai CLI `--switch` wrapper |
| `internal/ui/ui_test.go` / `automation_binding_test.go` | **Editable (Cũ)** | Tests giao diện & controller binding |
| `cmd/switcher/main_test.go` | **Editable (Cũ)** | Tests CLI binding |
| **`internal/manager/account_manager_test.go`** | **Editable (ADDENDUM MỚI)** | **CHỈ ĐƯỢC PHÉP** tiêm mock `rel.ServerDiscoverer` trong 3 test hàm: `TestSwitchAccount`, `TestSwitchAccount_AtomicAppStorageSync`, `TestSwitchAccount_CivilizedReAuth_HotSwapped`. **CẤM** sửa các hàm khác. |
| `internal/manager/account_manager.go` | **STRICTLY FORBIDDEN (CẤM)** | Mã nguồn sản phẩm — Tuyệt đối không được chạm vào |
| `scripts/build_windows.ps1` | **READ-ONLY (CẤM SỬA)** | Giữ nguyên release gate kiểm thử toàn diện |

---

## 6. Hướng Dẫn Kỹ Thuật Chi Tiết Cho Coder (Surgical Implementation Guide)

Coder thực hiện thêm đoạn code khởi tạo mock `rel.ServerDiscoverer` vào đúng 3 vị trí sau trong file `internal/manager/account_manager_test.go`:

### Vị trí 1: Trong hàm `TestSwitchAccount` (sau khi tạo `client` tại khoảng dòng 296-305)
```go
	// Thiết lập mock ServerDiscoverer để tránh rò rỉ ra tìm tiến trình language_server.exe thật trên OS
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       9999,
			Port:      port,
			CSRFToken: "test-csrf-token",
		}, nil
	}
```
*(Lưu ý: Đặt trước lệnh gọi `mgr.SwitchAccount(context.Background(), targetEmail, false)` đầu tiên).*

### Vị trí 2: Trong hàm `TestSwitchAccount_AtomicAppStorageSync` (sau khoảng dòng 755)
```go
	// Thiết lập mock ServerDiscoverer đồng bộ với atomicServer
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       1234,
			Port:      portAtomic,
			CSRFToken: "tok",
		}, nil
	}
```

### Vị trí 3: Trong hàm `TestSwitchAccount_CivilizedReAuth_HotSwapped` (sau khoảng dòng 855-860)
```go
	// Thiết lập mock ServerDiscoverer đồng bộ với client server
	rel.ServerDiscoverer = func() (*connector.ServerInfo, error) {
		return &connector.ServerInfo{
			PID:       8888,
			Port:      port,
			CSRFToken: "test-reauth-csrf",
		}, nil
	}
```

---

## 7. Tiêu Chí Nghiệm Thu Sau Cùng (Acceptance Criteria)

Coder cần xác nhận đạt đủ 2 điều kiện sau trước khi bàn giao Slice P1-C:
1. `go test ./internal/manager/...` vượt qua 100% với mã trả về 0 (exit code 0).
2. Lệnh bắt buộc hoàn tất thành công:
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all
   ```
   Tạo ra đầy đủ cả 2 file nhị phân trong thư mục `bin/`:
   - `bin/antigravity-switch.exe` (CLI pure Go)
   - `bin/antigravity-switch-gui.exe` (GUI Fyne với icon PE)

---

**Văn kiện có hiệu lực thi hành ngay lập tức kể từ thời điểm commit.**
