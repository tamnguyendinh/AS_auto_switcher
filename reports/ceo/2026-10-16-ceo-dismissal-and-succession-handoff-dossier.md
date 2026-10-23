# BẢN BÀN GIAO ĐIỀU PHỐI CHIẾN DỊCH & BÃI NHIỆM CEO (CEO HANDOFF DOSSIER)
**Dự án:** Antigravity Multi-Account Switcher & Autonomous Quota Failover Daemon  
**Thời điểm bàn giao:** 2026-10-16T14:00:00+07:00  
**Tác giả bàn giao:** Tiền nhiệm CEO Agent (Thread ID: `01a1428b-bf9f-7853-b10d-d2040f6fe08c`)  
**Đối tượng tiếp nhận:** Tân CEO Agent (Kế nhiệm / Next CEO Session)  
**Lý do bàn giao:** Human Owner bãi nhiệm Tiền nhiệm CEO do thiếu sót năng lực phân tích yêu cầu, lẫn lộn thẩm quyền và vi phạm quy trình điều hành lane.  
**Trạng thái Git HEAD:** Commit `a7794c8` (`docs(supervisor): zero-trust acceptance review rejecting gui switch flow due to ux modal and console flashing`)  
**Trạng thái Worktree:** 100% SẠCH (`git status --porcelain` = 0 dòng)  

---

## 1. TỔNG QUAN CHIẾN DỊCH & HIỆN TRẠNG TOÀN DIỆN

### 1.1 Mục tiêu dự án
1. Xây dựng công cụ chuyển đổi tài khoản Google độc lập, siêu nhẹ bằng Go thuần (Native Windows, Fyne GUI), hot-swap ngầm dưới 2 giây, không mất thread context, và **hoàn toàn không mở trình duyệt Chrome khi chuyển đổi**.
2. **Kỷ luật đồng bộ Python ➔ Go:** Bản Go phải kế thừa 100% tính năng, hành vi và tài nguyên của bản Python gốc (`core/`, `scanner/`, `accounts_store/`). Tuyệt đối cấm sáng tác hay suy đoán làm lệch hành vi.

### 1.2 Tiến độ các hạng mục đã hoàn thành & đã nghiệm thu Zero-Trust
| Hạng mục / Lỗi khắc phục | Coder Commit | Supervisor Zero-Trust Commit | Đánh giá |
| :--- | :---: | :---: | :--- |
| **P0-01 (Lệch đường dẫn accounts_store/)** | `bce8164` | `fc9fc60` | **PASS** - Phân giải đa cấp luôn neo về `G:\antigravity_auto_switch\accounts_store\`, không tạo `bin\accounts_store\`. |
| **P0-02 (Bỏ hoàn toàn lệnh AuthLogout)** | `1ed7e6f` | `84183e6` | **PASS** - Xóa 100% lời gọi AuthLogout, triệt tiêu nguyên nhân Electron IDE bật Chrome Google OAuth. |
| **P0-03 (Recycle LS thay vì Ctrl+R)** | `75664d8` | `ab55c71` | **PASS** - Kill language_server.exe để Antigravity cha tự spawn lại nạp WinCred ngầm dưới 2s, xóa bỏ phím Ctrl+R. |
| **P0-04 (Continuation & Thread Snapshot)** | `e56fd0d` | `5d2892c` | **PASS** - Lưu active thread snapshot, gọi RPC SmartFocus + LoadTrajectory khôi phục phiên chat dở dang. |
| **P1-02 (Khôi phục Always on Top)** | `a7265d0` | `9b51dd1` | **PASS** - Khôi phục nút Ghim trên cùng bằng Win32 SetWindowPos, xóa bỏ 100% nút/biến autoFailover tự chế. |
| **Logo thương hiệu & EXE Icon** | `c59f8e4` | `88e84f4` | **PASS** - Tạo `logo/app.ico`, nhúng vào Fyne GUI và PE Resource của 2 file .exe qua windres. |
| **Sập GUI Startup Panic (Interface Conversion)** | `741410c` | `2db3788` | **PASS** - Xóa bỏ struct bọc switcherWindow, giữ nguyên fyne.Window gốc, triệt tiêu lỗi sập GUI khi khởi chạy. |

---

## 2. BÀI HỌC QUẢN TRỊ XƯƠNG MÁU & NGUYÊN TẮC CỐT TỬ TỪ HUMAN OWNER

Tân CEO **BẮT BUỘC PHẢI KHẮC CỐT GHI TÂM** các nguyên tắc điều hành sau:

1. **Phân tích câu nói & ý định thực sự của Owner:**
   - Việc của CEO không phải là đi soi mã nguồn hay phân tích lỗi kỹ thuật (đó là việc của thợ).
   - **Việc tối thượng của CEO là phân tích chính xác câu nói của Owner, xem Owner THỰC SỰ ĐANG MUỐN GÌ**, từ đó phân loại đúng nhiệm vụ và đẩy việc về đúng Lane!
2. **Kỷ luật trước khi Coder code:**
   - **"Trước khi Coder code thì PHẢI CÓ LANE CHUYÊN LO KỸ THUẬT KIẾN TRÚC, CHUYÊN REVIEW. Coder chỉ được code, không được suy nghĩ!"**
   - Tuyệt đối không bao giờ nghe Owner báo lỗi một cái là xua ngay Coder đi sửa mù quáng. Mọi lỗi phải qua Lane Review / Kiến trúc sư thẩm định, ra báo cáo giải pháp được Owner duyệt rồi mới chuyển cho Coder.
3. **Mở lane mới đúng việc, dùng lane cũ đúng việc:**
   - Khi công việc cùng bối cảnh đang dở dang (ví dụ Owner yêu cầu review thêm trong lúc Supervisor đang làm việc): **Gửi ngay vào chính Lane đang chạy đó qua `send_message_to_thread`**, không mở lane mới bừa bãi!
   - Chỉ mở lane mới (`create_thread`) khi chuyển giao vai trò đối lập (Handoff Coder ➔ Supervisor để đảm bảo Zero-Trust) hoặc mở một mặt trận hoàn toàn độc lập.
4. **Quy tắc Zero-Trust thực chất:**
   - Tuyệt đối không tin bất cứ file test nào xanh là pass. Test xanh không có nghĩa là tính năng đúng.
   - Không tin cái gì cả, chỉ tin vào tính năng và hành vi đúng thực tế trên Codebase và Runtime thật của file `.exe`!
5. **Đồng bộ tài nguyên tuyệt đối với Python:**
   - Python dùng cái gì, thư mục nào (`accounts_store/`), format nào thì Go phải dùng lại y hệt, cấm tuyệt đối suy đoán hay tự tiện sáng tác.

---

## 3. ĐIỂM DỪNG HIỆN TẠI (FRONTIER) & NHIỆM VỤ TIẾP THEO CỦA TÂN CEO

### 3.1 Trạng thái đang chờ xử lý
Supervisor Lane vừa ban hành Báo Cáo Phán Quyết **REJECT** tại commit `a7794c8` (`reports/supervisor/2026-10-16-gui-switch-flow-and-ux-supervisor-review.md`) đối với 2 lỗi UX mà Owner phát hiện khi test thực tế:
1. **Khuyết tật 1 (Modal popup thành công):**
   - Vị trí: `internal/ui/app.go` dòng 197 và 235 đang gọi `dialog.ShowInformation(...)`.
   - Chuẩn mực Python (`ui/app_window.py:508`): Cấm mọi modal popup thành công, chỉ cập nhật dòng chữ trên `statusBar` ngầm.
2. **Khuyết tật 2 (Cửa sổ terminal đen chớp tắt khi chạy ngầm):**
   - Vị trí: Các lệnh `exec.Command` thiếu thuộc tính Win32 `CreationFlags: 0x08000000` (`CREATE_NO_WINDOW`) và `HideWindow: true` tại:
     + `internal/connector/discovery.go` dòng 72, 195.
     + `internal/reloader/session_reloader.go` dòng 71.
   - Chuẩn mực Python (`core/session_reloader.py:86-88`): Luôn truyền `creationflags = subprocess.CREATE_NO_WINDOW`.

### 3.2 Hành động cụ thể của Tân CEO (Turnkey Action)
Ngay khi tiếp nhận quyền chỉ huy:
1. Xác nhận tiếp nhận vai trò CEO và cam kết tuân thủ 100% các nguyên tắc quản trị của Owner.
2. Mở một **Coder Lane** (hoặc giao tiếp vào thread coder nếu thích hợp) với hợp đồng chuẩn sửa dứt điểm 2 lỗi UX nói trên:
   - Xóa bỏ 2 dialog popup trong `internal/ui/app.go`.
   - Gắn cờ `CREATE_NO_WINDOW` cho toàn bộ các lệnh `exec.Command`.
3. Sau khi Coder hoàn thành, mở **Supervisor Lane** độc lập để kiểm chứng Zero-Trust.
4. Bàn giao file `.exe` hoàn chỉnh cho Owner trực tiếp kiểm nghiệm runtime!

---
*Bản bàn giao này được lập chính thức, lưu trữ bền vững trên Git repository.*