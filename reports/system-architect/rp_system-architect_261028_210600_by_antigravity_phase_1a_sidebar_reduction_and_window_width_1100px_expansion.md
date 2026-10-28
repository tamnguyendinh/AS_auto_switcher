# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Tinh Chỉnh Giao Diện Giai Đoạn 1A: Thu Gọn Độ Rộng Sidebar Xuống 80% (136px) & Mở Rộng Độ Rộng Cửa Sổ Tổng Lên 1100px (Khai Phóng Lưới 3 Cột Card)

- **Mã tài liệu:** `rp_system-architect_261028_210600_by_antigravity_phase_1a_sidebar_reduction_and_window_width_1100px_expansion`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-28
- **Thẩm quyền phê duyệt:** Human Owner (Chủ tịch) phê duyệt trực tiếp trong phiên thảo luận kỹ thuật
- **Trạng thái:** PRODUCTION-READY BLUEPRINT (Khóa cứng Touch-Map, bàn giao chìa khóa trao tay cho Coder)
- **Điều kiện kích hoạt thi công:** Triển khai ngay sau khi nhánh công việc "Vấn đề 2h" (`active conversation threads window 2h`) được Coder hoàn tất, QA/Supervisor kiểm duyệt và nghiệm thu PASS hoàn toàn.

---

# PHẦN 1: BỐI CẢNH & PHÂN TÍCH KIẾN TRÚC GIAI ĐOẠN 1A

### 1.1. Đánh giá sau khi nghiệm thu Giai đoạn 1
Giai đoạn 1 đã triển khai thành công việc đổi tên ứng dụng thành **Auto Switcher** và tạo lập cấu trúc Sidebar bên trái với Tab Antigravity. Tuy nhiên, qua quá trình trực quan hóa trên màn hình thực tế, Chủ tịch đã chỉ đạo tinh chỉnh tối ưu hóa giao diện (Giai đoạn 1A) trước khi bước sang Giai đoạn 2:
1. **Sidebar hiện tại (170px) còn hơi chiếm diện tích**: Cần thu gọn xuống còn **80%** kích thước ban đầu, tức đưa về **136px**.
2. **Độ rộng cửa sổ tổng (980px) chưa tối ưu mật độ hiển thị card**: Cần mở rộng lên **1100px** để tạo không gian thoáng đãng cho vùng Content Area, đưa lưới hiển thị tài khoản từ 2 cột lên **3 CỘT CARD (3-column grid)**.

### 1.2. Phân tích Không gian & Tỷ lệ Bố cục Mới

$$\text{Tổng bề ngang cửa sổ} = \mathbf{1100\text{ px}}$$

Phân bổ không gian:
- **Sidebar bên trái (`SidebarWidth`)**: $170 \times 0.8 = \mathbf{136\text{ px}}$
- **Đường viền ngăn cách (BorderSubtle)**: $\mathbf{1\text{ px}}$
- **Khung Content Area bên phải**:
  $$\text{Không gian làm việc khả dụng} = 1100 - 136 - 1 = \mathbf{963\text{ px}}$$

### 1.3. Lợi thế Kiến trúc Đột phá của Khung Content 963px
- Mỗi card tài khoản có kích thước chuẩn $280 \times 130\text{ px}$.
- Khi xếp 3 cột card:
  $$\text{Chiều rộng 3 card} = 280 \times 3 = 840\text{ px}$$
- Khoảng cách còn lại: $963 - 840 = 123\text{ px}$ hoàn toàn lý tưởng cho padding mép trái/phải và gap giữa các cột (~20px mỗi khoảng cách).
- **Kết quả UX**: Người dùng nhìn thấy 3 tài khoản trên mỗi hàng, giảm đáng kể nhu cầu cuộn dọc, tăng tốc độ quan sát và chuyển đổi tài khoản.

---

# PHẦN 2: BỐ CỤC GIAO DIỆN GIAI ĐOẠN 1A (TARGET LAYOUT)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ TOP BAR: [AS] Auto Switcher ────────────── (Kéo cửa sổ) ────────────── [VN|EN] [—] [□] [✕]        │
├───────────────┬──────────────────────────────────────────────────────────────────────────────────┤
│ SIDEBAR       │ CONTENT AREA (Khung làm việc Antigravity: Rộng 963px - Hiển thị 3 Cột Card)      │
│ (136px - 80%) │                                                                                  │
│               │ ┌──────────────────────────────────────────────────────────────────────────────┐ │
│ [Logo] Anti.. │ │ Toolbar: [+ Thêm] [F5] [Quota] [Ghim] │ [Ẩn] │ [Tự động] │ [Theme 🌙]        │ │
│ (Active Tab)  │ └──────────────────────────────────────────────────────────────────────────────┘ │
│               │ ┌──────────────────────────────────────────────────────────────────────────────┐ │
│               │ │ ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                       │ │
│               │ │ │ Card 1       │   │ Card 2       │   │ Card 3       │                       │ │
│ (Chưa có      │ │ │ [280x130 px] │   │ [280x130 px] │   │ [280x130 px] │                       │ │
│  Codex)       │ │ └──────────────┘   └──────────────┘   └──────────────┘                       │ │
│               │ │ ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                       │ │
│               │ │ │ Card 4       │   │ Card 5       │   │ Card 6       │                       │ │
│               │ │ └──────────────┘   └──────────────┘   └──────────────┘                       │ │
│               │ └──────────────────────────────────────────────────────────────────────────────┘ │
├───────────────┴──────────────────────────────────────────────────────────────────────────────────┤
│ FOOTER BAR: [Trạng thái Daemon / Liveness Antigravity] ────────────────────────── [v1.0.3.0]     │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

# PHẦN 3: NGUYÊN TẮC BẤT BIẾN KIẾN TRÚC (ARCHITECTURAL INVARIANTS)

1. **Exact Dimension Invariant:**
   - Hằng số `SidebarWidth` bắt buộc phải là `136.0` (chính xác 80% của 170.0).
   - Hằng số `DefaultWindowWidth` bắt buộc phải là `1100.0` (tăng từ 980.0).
   - `DefaultWindowHeight` giữ nguyên `630.0`.
   - `MinWindowWidth` giữ nguyên `640.0`.
2. **Text Integrity in 136px Sidebar:**
   - Chữ `"Antigravity"` trên Tab Sidebar (font size 13, bold) đi kèm icon $20\times 20\text{ px}$ phải hiển thị gọn gàng, căn chỉnh trung tâm/dọc thẩm mỹ, không bị tràn viền (clipping) hay xuống dòng.
3. **Card Grid Reflow Invariant:**
   - Lưới card `accountListContainer = newSymmetricGridWrap(fyne.NewSize(280, 130))` phải tự động thích ứng với chiều rộng mới và dàn thành 3 cột đều đặn khi ở kích thước mặc định 1100px.
4. **Test Suite Parity Invariant:**
   - 100% các assertion kiểm tra `DefaultWindowWidth` và `SidebarWidth` trong các tệp test (`phase1_sidebar_test.go`, `ui_test.go`) phải được cập nhật đồng bộ sang `1100.0` và `136.0`, đạt trạng thái `PASS` tuyệt đối.

---

# PHẦN 4: HƯỚNG DẪN THI CÔNG CHI TIẾT DÀNH CHO CODER (TOUCH-MAP)

### 4.1. Tệp 1: [`internal/ui/sidebar.go`](file:///g:/antigravity_auto_switch/internal/ui/sidebar.go)
Cập nhật hằng số độ rộng Sidebar:
```go
// SidebarWidth định nghĩa độ rộng chuẩn của thanh Sidebar (thu gọn xuống 80% còn 136px theo Phase 1A)
const SidebarWidth float32 = 136.0
```

### 4.2. Tệp 2: [`internal/ui/window.go`](file:///g:/antigravity_auto_switch/internal/ui/window.go)
Cập nhật hằng số độ rộng cửa sổ mặc định:
```go
const (
	DefaultWindowWidth  float32 = 1100.0 // Mở rộng từ 980.0 lên 1100.0 theo Phase 1A
	DefaultWindowHeight float32 = 630.0
	MinWindowWidth      float32 = 640.0
	MinWindowHeight     float32 = 360.0
)
```

### 4.3. Tệp 3: [`internal/ui/phase1_sidebar_test.go`](file:///g:/antigravity_auto_switch/internal/ui/phase1_sidebar_test.go)
Đồng bộ các assertion kiểm tra độ rộng:
```go
// Kiểm tra SidebarWidth
if SidebarWidth != 136.0 {
    t.Errorf("expected SidebarWidth 136.0 (80%% of 170.0), got %f", SidebarWidth)
}

// Kiểm tra DefaultWindowWidth
if DefaultWindowWidth != 1100.0 {
    t.Errorf("expected DefaultWindowWidth 1100.0, got %f", DefaultWindowWidth)
}
```

### 4.4. Tệp 4: [`internal/ui/ui_test.go`](file:///g:/antigravity_auto_switch/internal/ui/ui_test.go)
Tìm tất cả các điểm kiểm tra `DefaultWindowWidth == 980.0` và cập nhật thành `1100.0`:
- Dòng ~1024, ~2253, ~2355, ~2535, ~2570, ~2579, ~2939, ~3296, ~3383, ~3423, ~3503.
- Đảm bảo tất cả các test case kích thước canvas và snapshot layout khớp với kích thước mới `1100x630`.

---

# PHẦN 5: CỔNG KIỂM TRA CHẤT LƯỢNG & NGHIỆM THU (SAFETY GATE 1A)

Trước khi gửi báo cáo hoàn thành, Coder phải vượt qua đầy đủ các bước kiểm tra sau:

1. **Kiểm thử tự động:**
   ```powershell
   go test ./internal/ui/... -v
   ```
   *Yêu cầu:* 100% test case PASS, không có bất kỳ failure hay regression nào.

2. **Biên dịch toàn hệ thống:**
   ```powershell
   .\scripts\build_windows.ps1
   ```
   *Yêu cầu:* Tạo file nhị phân thành công không cảnh báo.

3. **Xác minh trực quan thực tế:**
   - Chạy ứng dụng.
   - Sidebar bên trái hiển thị thon gọn (136px), chữ Antigravity và logo cân đối.
   - Bề ngang cửa sổ rộng rãi (1100px), danh sách tài khoản dàn đều thành 3 cột thẻ trực quan, rõ ràng.
   - Các chức năng chuyển theme, chuyển ngôn ngữ, đổi tài khoản hoạt động mượt mà.

---

# PHẦN 6: ĐIỀU KIỆN TIẾP TỤC SANG GIAI ĐOẠN 2

Theo chỉ thị trực tiếp từ Chủ tịch:
1. Sau khi Giai đoạn 1A được Coder triển khai và QA/Supervisor nghiệm thu PASS hoàn toàn, **Chủ tịch sẽ trực tiếp review thủ công giao diện thực tế**.
2. Khi Chủ tịch xác nhận review thủ công đạt yêu cầu (PASS), Chủ tịch và System Architect sẽ chính thức bắt đầu phiên thảo luận chi tiết để thiết kế **GIAI ĐOẠN 2 (Bổ sung Tab Codex và cơ chế chuyển đổi view)**.
