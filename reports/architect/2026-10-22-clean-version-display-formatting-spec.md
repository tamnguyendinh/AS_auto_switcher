# ĐẶC TẢ KIẾN TRÚC & KỸ THUẬT: CHUẨN HÓA ĐỊNH DẠNG HIỂN THỊ PHIÊN BẢN TRÊN GIAO DIỆN (CLEAN VERSION DISPLAY)

- **Tài liệu:** Đặc tả Kỹ thuật Kiến trúc & Hướng dẫn Khắc phục Lỗi Giao diện (Software Architecture & Remediation Specification)
- **Mã định danh:** `SPEC-ARCH-20261022-CLEAN-VERSION-DISPLAY-SPEC`
- **Tác giả:** System Architect Lane
- **Ngày ban hành:** 2026-10-22
- **Trạng thái:** APPROVED / LOCKED FOR CODER
- **Bằng chứng đầu vào:** Ảnh chụp lỗi thực tế từ Human Owner tại `media_1792679112437_0b1f0128.png`
- **Đối tượng tiếp nhận:** CEO Agent, Coder Lane, Supervisor Lane

---

## PHẦN I: PHÂN TÍCH VÀ XÁC ĐỊNH LỖI TỪ HÌNH ẢNH CỦA HUMAN OWNER

### 1. Quan sát trực quan trên bức ảnh `media_1792679112437_0b1f0128.png`
- **Vị trí được đánh dấu:** Góc dưới cùng bên phải của cửa sổ ứng dụng (khu vực mép phải của Footer Status Bar).
- **Ký hiệu chỉ định của Owner:** 
  - Một mũi tên màu đỏ lớn hướng trực diện vào góc phải thanh đáy.
  - Một khung viền chữ nhật màu đỏ bao bọc trọn vẹn cụm văn bản hiển thị:
    ```text
    Version: 1.0.2.0 (rev 3dd7d69)
    ```

### 2. Định nghĩa bản chất lỗi kỹ thuật (Defect Identification)
- **Vấn đề cốt lõi:** Chuỗi hiển thị phiên bản đang phơi bày mã băm Git commit nội bộ `(rev 3dd7d69)` ra giao diện người dùng cuối.
- **Tiêu chuẩn sản phẩm Desktop thương mại:**
  - Không có một ứng dụng Desktop production chuyên nghiệp nào lại hiển thị mã băm commit nội bộ kiểu `(rev 3dd7d69)` trên thanh trạng thái chính dành cho người dùng.
  - Việc gắn thêm đuôi `(rev <hash>)` làm giao diện trở nên rườm rà, thiếu tính thương mại và tạo ấn tượng đây là bản thử nghiệm nội bộ chưa hoàn thiện.
  - Giao diện người dùng cuối (User-Facing UI) chỉ được phép hiển thị phiên bản phát hành sạch sẽ: **`Version: 1.0.2.0`**.
- **Chấn chỉnh nhận định sai lệch trước đó:**
  - Lỗi này **HOÀN TOÀN KHÔNG PHẢI** là lỗi căn lề, khoảng cách lề (padding/margin) hay việc nhãn có nằm sát mép hay không.
  - Lỗi nằm ở chính nội dung văn bản: **Hiển thị chuỗi `(rev 3dd7d69)` trên UI**.

---

## PHẦN II: NGUYÊN NHÂN KỸ THUẬT GỐC RỄ (ROOT CAUSE ANALYSIS)

Sau khi khảo sát toàn bộ codebase liên quan đến hệ thống Versioning và UI Footer, nguyên nhân gốc rễ được chỉ rõ như sau:

1. **Logic định dạng trong `internal/version/version.go`:**
   ```go
   // Code hiện tại:
   func GetFullVersion() string {
       if GitCommit != "" {
           return fmt.Sprintf("Version: %s (rev %s)", Version, GitCommit)
       }
       return fmt.Sprintf("Version: %s", Version)
   }
   ```
   Hàm `GetFullVersion()` hiện đang tự động nối thêm `(rev %s)` bất cứ khi nào biến `GitCommit` mang giá trị khác rỗng.

2. **Cơ chế Build chính thức trong `scripts/build_windows.ps1`:**
   Tại dòng 124–129 của `scripts/build_windows.ps1`:
   ```powershell
   $GitCommit = (git rev-parse --short HEAD 2>$null).Trim()
   $ldflags = "-H windowsgui -s -w -X 'antigravity-switch/internal/version.Version=$AppVersion' -X 'antigravity-switch/internal/version.GitCommit=$GitCommit'"
   ```
   Khi biên dịch bằng script chuẩn của repository, biến `GitCommit` luôn được truyền giá trị commit hash hiện tại (ở commit vừa qua là `3dd7d69`).

3. **Gắn trực tiếp vào giao diện trong `internal/ui/window.go`:**
   Tại dòng 247:
   ```go
   a.versionLabel = widget.NewLabelWithStyle(version.GetFullVersion(), fyne.TextAlignTrailing, fyne.TextStyle{})
   ```
   Do `GetFullVersion()` trả về `Version: 1.0.2.0 (rev 3dd7d69)`, nhãn trên thanh đáy đã hiển thị trực tiếp chuỗi này ra màn hình người dùng.

4. **Sự tiếp tay của Test Suite cũ:**
   Trong `internal/version/version_test.go` (`TestVersion_WithGitCommit`), bài test cũ đã ấn định kỳ vọng sai lầm:
   ```go
   expected := "Version: 1.0.2.0 (rev a77efb9)"
   if GetFullVersion() != expected { ... }
   ```
   Điều này khiến logic hiển thị sai lệch bị hợp thức hóa qua kiểm thử.

---

## PHẦN III: GIẢI PHÁP KIẾN TRÚC CHUẨN HÓA (ARCHITECTURAL SOLUTION)

### 1. Nguyên lý phân tách trách nhiệm (Separation of Concerns)
- **User-Facing UI Display:** Thuộc về người dùng cuối. Phải luôn hiển thị sạch, thanh lịch, chuẩn mực sản phẩm: **`Version: 1.0.2.0`**. Tuyệt đối không phơi bày thông tin Git commit, build revision ra UI.
- **Diagnostic & Telemetry Metadata:** Dành cho kỹ sư, nhật ký log tiến trình hoặc CLI debug. Biến `GitCommit` vẫn tiếp tục được lưu giữ trong package `internal/version` và inject qua `-ldflags` lúc build để phục vụ việc ghi log chẩn đoán khi cần, nhưng được cách ly hoàn toàn khỏi chuỗi hiển thị trên UI.

### 2. Thiết kế API chuẩn trong `internal/version/version.go`
- **`GetVersion() string`**: Trả về số phiên bản thuần túy (`"1.0.2.0"`).
- **`GetFullVersion() string`**: Dành riêng cho giao diện người dùng (UI). Luôn trả về:
  ```go
  fmt.Sprintf("Version: %s", Version) // Kết quả: "Version: 1.0.2.0"
  ```
  Bất kể `GitCommit` có rỗng hay chứa commit hash, `GetFullVersion()` luôn bảo đảm định dạng thương mại sạch sẽ cho UI.
- **`GetBuildInfo() string` (Mới - Tùy chọn cho Telemetry/Diagnostics):**
  Trả về đầy đủ thông tin kỹ thuật gồm cả commit hash:
  ```go
  func GetBuildInfo() string {
      if GitCommit != "" {
          return fmt.Sprintf("Version %s (commit %s)", Version, GitCommit)
      }
      return fmt.Sprintf("Version %s", Version)
  }
  ```

### 3. Đồng bộ hóa Kiểm thử (Test Suite Synchronization)
- **`internal/version/version_test.go`:**
  - `TestVersion_Default`: Xác nhận `GetVersion() == "1.0.2.0"` và `GetFullVersion() == "Version: 1.0.2.0"`.
  - `TestVersion_WithGitCommit`: Khi gán `GitCommit = "3dd7d69"`, `GetFullVersion()` **VẪN PHẢI BẢO ĐẢM** trả về `"Version: 1.0.2.0"`, không được chứa `rev` hay mã commit.
  - `TestVersion_BuildInfo`: Kiểm tra `GetBuildInfo()` trả về thông tin đầy đủ kèm commit.
- **`internal/ui/footer_bar_acceptance_test.go` (`TestAcceptance_TC03_VersionDisplayAndFormatting`):**
  - Khẳng định chặt chẽ: `app.versionLabel.Text == "Version: 1.0.2.0"`.
  - Khẳng định ngăn ngừa hồi quy: `!strings.Contains(app.versionLabel.Text, "rev")` và `!strings.Contains(app.versionLabel.Text, "(")`.

---

## PHẦN IV: TIÊU CHÍ NGHIỆM THU CHẤP NHẬN (ACCEPTANCE CRITERIA - AC)

| Mã AC | Tiêu chí Kiểm định | Kỳ vọng Chi tiết |
|---|---|---|
| **AC-01** | **Clean UI Version Text** | `a.versionLabel.Text` trên giao diện thanh đáy hiển thị chính xác là `"Version: 1.0.2.0"`. Tuyệt đối không chứa `(rev ...)` hoặc chuỗi commit hash. |
| **AC-02** | **SSOT Function Invariant** | `version.GetFullVersion()` luôn trả về `"Version: 1.0.2.0"` trong mọi trường hợp (kể cả khi `GitCommit` được inject lúc build). |
| **AC-03** | **Acceptance Test Suite Pass** | Bài test `TestAcceptance_TC03_VersionDisplayAndFormatting` và các test trong `internal/version/version_test.go` chạy `go test ./...` đạt 100% PASS với định dạng mới. |
| **AC-04** | **Windows Build & PE Verification** | Biên dịch `pwsh .\scripts\build_windows.ps1 -Clean` thành công (Exit Code 0), binary sinh ra có đầy đủ Application Icon, PE VERSIONINFO là `1.0.2.0` và khi khởi chạy giao diện chỉ hiển thị `Version: 1.0.2.0`. |

---

## PHẦN V: KHÓA TOUCH-MAP CHO CODER LANE (STRICT TOUCH-MAP)

Coder Lane được chỉ định thực thi nghiêm ngặt và **CHỈ ĐƯỢC PHÉP CHẠM** vào các tệp tin trong danh sách sau:

1. **`internal/version/version.go`**: Cập nhật hàm `GetFullVersion()` trả về `"Version: 1.0.2.0"` sạch sẽ; bổ sung `GetBuildInfo()`.
2. **`internal/version/version_test.go`**: Cập nhật test case khẳng định `GetFullVersion()` không rò rỉ commit hash lên UI.
3. **`internal/ui/footer_bar_acceptance_test.go`**: Cập nhật assertion của `TestAcceptance_TC03_VersionDisplayAndFormatting` để khóa chặt chuỗi `"Version: 1.0.2.0"`.

> [!CAUTION]
> **CẤM TUYỆT ĐỐI**: Coder Lane không được phép chỉnh sửa bất kỳ tệp tin nào nằm ngoài Touch-Map trên (đặc biệt không sửa `internal/ui/window.go`, `cmd/`, `internal/store/`, hoặc cấu trúc layout thanh đáy đã hoàn thiện ở commit trước).
