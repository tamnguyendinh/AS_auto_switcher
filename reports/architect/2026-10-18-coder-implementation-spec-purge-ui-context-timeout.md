# BẢN ĐẶC TẢ THI CÔNG (CODER IMPLEMENTATION SPEC)
## XÓA BỎ HOÀN TOÀN TÀN DƯ CONTEXT TIMEOUT 15s TẠI TẦNG UI & BẢO TOÀN NGUYÊN VẸN QUY TRÌNH 4 BƯỚC CHUẨN

- **Tác giả:** Architect Lane
- **Ngày:** 2026-10-18
- **Phiên bản:** 1.0.0
- **Đối tượng thi công:** Coder Lane
- **Phê duyệt bởi:** Human Owner

---

## 1. MỤC TIÊU VÀ NGUYÊN TẮC CỐT LÕI (IRON LAWS)

1. **Xóa bỏ vĩnh viễn tàn dư kỹ thuật Context Timeout 15s tại tầng UI:**
   - Xóa bỏ hoàn toàn dòng `ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)` tại `internal/ui/app.go:545`.
   - **LỆNH THÉP TỪ HUMAN OWNER:** Tuyệt đối **KHÔNG ĐƯỢC PHÉP** tự ý thay thế bằng bất kỳ con số timeout nhân tạo nào khác (dù là 10s, 30s hay 60s). Quy trình chạy ngầm trong Goroutine phải được chạy tự nhiên từ đầu đến cuối cho đến khi gửi xong thread cuối cùng.

2. **Bảo toàn nguyên vẹn 100% Quy trình 4 bước chuẩn (KHÔNG ĐƯỢC LÀM VẸO QUY TRÌNH):**
   - **Bước 1:** Chốt danh sách active threads bằng snapshot SQLite 2h trước khi chuyển tài khoản (lưu tệp JSON).
   - **Bước 2:** Chuyển tài khoản chuẩn Antigravity (Đồng bộ ToS + Ghi WinCred + Foreground + Taskkill LS + **Chờ đủ 8 giây** + Dò tìm LS mới & **Xác nhận danh tính trong RAM** qua `GetUserStatus`).
   - **Bước 3:** Đọc lại đúng tệp JSON snapshot của Bước 1.
   - **Bước 4:** Gửi tin nhắn tuần tự vào từng thread (SmartFocus -> LoadTrajectory -> SendUserCascadeMessage) với **nghỉ 3 giây pacing delay** giữa các thread.
   - Cấm thay đổi trật tự, cấm gộp bước, cấm bỏ bớt bất kỳ bước nào trong 4 bước này.

3. **Sửa dứt điểm rủi ro chọn nhầm Cổng HTTPS tại `internal/connector/discovery.go`:**
   - Tăng thời gian probe HTTP từ `250ms` lên `1*time.Second`.
   - Khi probe không khớp, **bắt buộc mặc định chọn cổng số lớn nhất** (`ports[len(ports)-1]`), tuyệt đối không được chọn cổng số nhỏ (`ports[0]`) vì cổng số nhỏ trong Antigravity là HTTPS listener (từ chối HTTP thuần bằng mã `400 Bad Request`).

4. **Minh bạch hóa kết quả tiếp tục phiên trên UI Status Bar:**
   - Hiển thị số thread đã gửi thành công / tổng số thread ngay trên Status Bar sau khi hoàn tất.

---

## 2. CHI TIẾT CÁC TỆP CẦN CAN THIỆP VÀ DIFF CỤ THỂ

### 2.1 Tệp 1: `internal/ui/app.go`
**Vị trí:** Trong phương thức `SwitchAccount(targetEmail string)` tại `internal/ui/app.go:530-580`.

#### Mã nguồn hiện tại (BEFORE):
```go
		if a.mgr == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		res, err := a.mgr.SwitchAccount(ctx, targetEmail, false) // forceLogout = false: smooth zero-chrome hot-swap

		if err != nil {
			fyne.Do(func() {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Chuyển thất bại: %v", err))
				}
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
			})
			return
		}

		// Refresh account states in RAM and update cards/header immediately
		overview, _ := a.mgr.GetAccountStates(ctx)
		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			if a.statusBarLabel != nil {
				a.statusBarLabel.SetText(fmt.Sprintf("Đã chuyển sang %s (%s)", res.TargetEmail, res.Duration.Round(time.Millisecond)))
			}
		})
```

#### Mã nguồn sau điều chỉnh (AFTER):
```go
		if a.mgr == nil {
			return
		}

		// Xóa bỏ hoàn toàn bẫy context.WithTimeout(15s).
		// Quy trình 4 bước chạy tự nhiên từ đầu đến thread cuối cùng qua context.Background().
		res, err := a.mgr.SwitchAccount(context.Background(), targetEmail, false) // forceLogout = false: smooth zero-chrome hot-swap

		if err != nil {
			fyne.Do(func() {
				if a.statusBarLabel != nil {
					a.statusBarLabel.SetText(fmt.Sprintf("Chuyển thất bại: %v", err))
				}
				if a.win != nil {
					dialog.ShowError(err, a.win)
				}
			})
			return
		}

		// Refresh account states in RAM and update cards/header immediately
		overview, _ := a.mgr.GetAccountStates(context.Background())
		a.mu.Lock()
		a.overview = overview
		a.mu.Unlock()

		fyne.Do(func() {
			a.updateUI(overview)
			if a.statusBarLabel != nil {
				statusText := fmt.Sprintf("Đã chuyển sang %s (%s)", res.TargetEmail, res.Duration.Round(time.Millisecond))
				if res != nil && res.ContinuationSummary != nil && res.ContinuationSummary.TotalThreads > 0 {
					statusText = fmt.Sprintf("Đã chuyển sang %s (%s) • Đã tiếp tục %d/%d phiên",
						res.TargetEmail, res.Duration.Round(time.Millisecond),
						res.ContinuationSummary.SuccessCount, res.ContinuationSummary.TotalThreads)
				}
				a.statusBarLabel.SetText(statusText)
			}
		})
```

---

### 2.2 Tệp 2: `internal/connector/discovery.go`
**Vị trí:** Trong hàm `DiscoverListeningPort(pid int)` tại dòng 280-300.

#### Mã nguồn hiện tại (BEFORE):
```go
// DiscoverListeningPort finds the primary Connect-RPC HTTP listening port for the PID.
// When multiple ports are listening, it probes each port and selects the one returning HTTP 200.
func DiscoverListeningPort(pid int) (int, error) {
	ports, err := DiscoverListeningPorts(pid)
	if err != nil || len(ports) == 0 {
		return 0, ErrNoListeningPort
	}

	if len(ports) == 1 {
		return ports[0], nil
	}

	// Probe each port to find HTTP 200 listener
	for _, p := range ports {
		if ProbeHTTPPort(p, 250*time.Millisecond) {
			return p, nil
		}
	}

	// Default to lowest port if probing does not match
	return ports[0], nil
}
```

#### Mã nguồn sau điều chỉnh (AFTER):
```go
// DiscoverListeningPort finds the primary Connect-RPC HTTP listening port for the PID.
// When multiple ports are listening, it probes each port and selects the one returning HTTP 200.
func DiscoverListeningPort(pid int) (int, error) {
	ports, err := DiscoverListeningPorts(pid)
	if err != nil || len(ports) == 0 {
		return 0, ErrNoListeningPort
	}

	if len(ports) == 1 {
		return ports[0], nil
	}

	// Probe each port to find HTTP 200 listener (probe timeout 1s)
	for _, p := range ports {
		if ProbeHTTPPort(p, 1*time.Second) {
			return p, nil
		}
	}

	// In Antigravity, the lower port is the HTTPS listener (which rejects plain HTTP with 400),
	// while the higher port is always the Connect-RPC HTTP listener. Default to the highest port!
	return ports[len(ports)-1], nil
}
```

---

### 2.3 Tệp 3: `internal/ui/ui_test.go`
**Yêu cầu kiểm thử:**
- Cập nhật các test assertion nếu cần để xác nhận `app.statusBarLabel.Text` hiển thị đầy đủ thông điệp chuyển tài khoản và số phiên tiếp tục.
- Đảm bảo toàn bộ test suite của package `internal/ui`, `internal/manager`, `internal/connector` đều `PASS 100%`.

---

## 3. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

1. **Zero Context Timeout Choke:** Tuyệt đối không còn `context.WithTimeout(..., 15*time.Second)` bên trong `a.SwitchAccount`.
2. **Quy trình 4 bước nguyên vẹn:**
   - Bước 1: Vẫn snapshot đủ danh sách thread.
   - Bước 2: Vẫn đổi WinCred, kill LS, chờ đủ 8 giây, xác thực RAM.
   - Bước 3: Vẫn đọc lại file snapshot của Bước 1.
   - Bước 4: Vẫn gửi tuần tự từng thread với pacing delay 3s giữa các thread cho đến thread cuối cùng.
3. **Không kẹt cổng HTTPS:** `DiscoverListeningPort` luôn ưu tiên cổng Connect-RPC HTTP (`ports[len(ports)-1]`).
4. **Compile & Test Pass:** `go test ./...` chạy thành công không một lỗi lầm.

---

## 4. KẾT LUẬN VÀ BÀN GIAO
Bản đặc tả này là mệnh lệnh kỹ thuật thi công chính thức. Yêu cầu Coder Lane thực hiện sửa đổi chính xác từng dòng theo đúng mã mẫu ở trên, kiểm tra kiểm thử và báo cáo hoàn thành sang Supervisor Lane.
