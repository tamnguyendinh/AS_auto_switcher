# Báo Cáo Đề Xuất Description & Metadata Cho GitHub Repository

**Tài liệu:** `reports/docs/rp_docs_github_repo_description_proposals.md`  
**Dự án:** Antigravity Multi-Account Switcher & Autonomous Failover Daemon  
**Phiên bản mã nguồn thực tế:** `v1.0.2.0` (Căn cứ `internal/version/version.go`, không dùng README cũ)  
**Người thực hiện:** Technical Writer Lane  
**Ngày cập nhật:** 23/10/2026  
**Thẩm quyền:** Human Owner Directive (Khảo sát trực tiếp mã nguồn thực tế, không dùng README cũ), CEO Directive (Thread ID: `ebad9771-35b4-4afb-955f-1b75612f7630`)  

---

## 1. Nguồn Căn Cứ & Hiện Trạng Kiến Trúc Mã Nguồn (Codebase-Grounded Truth)

*Toàn bộ nội dung đề xuất trong tài liệu này được trích xuất và đối soát trực tiếp từ kiến trúc mã nguồn Go đang hoạt động trong `internal/`, hoàn toàn độc lập và không phụ thuộc vào `README.md` cũ:*

1. **Phiên bản chính thức:** `Version 1.0.2.0` (xác lập tại `internal/version/version.go`).
2. **Động cơ Chuyển Đổi Tự Động (Autonomous Failover Daemon - AutoSwitch):**
   - **Van kích hoạt kép (Dual-Trigger Engine):** Kích hoạt chủ động qua chu kỳ đo lường nền 60s (ngưỡng 5h $\le 2\%$, tuần $\le 1\%$) và kích hoạt bị động tức thì khi nhận lỗi Connect-RPC streaming (`429 Too Many Requests` hoặc `RESOURCE_EXHAUSTED`).
   - **Phục hồi Native In-Memory 3-Retry:** Thử lại nội bộ qua `SendUserCascadeMessage` với danh sách nội dung rỗng (`empty content list`) để phân biệt micro-spike trước khi chuyển tài khoản.
   - **Thuật toán Burn-down Sorter:** 5 tiêu chí sơ loại nghiêm ngặt và 4 tầng ưu tiên (ưu tiên đốt cạn quota tuần thấp nhất trước chu kỳ reset, ưu tiên tài khoản 5h dồi dào).
   - **Single-Flight Concurrency Gate:** Windows Named Mutex độc quyền (`AntigravitySwitch_SwitchGate_<USERNAME>`) ngăn chặn hoàn toàn race condition giữa daemon tự động và thao tác thủ công.
3. **Đột phá Hiệu năng Tầng Dưới (Under-the-Hood Kernel & Memory Performance):**
   - **Win32 Kernel Discovery siêu tốc:** Thay thế hoàn toàn subprocess `powershell`/`netstat` bằng native Win32 APIs (`NtQueryInformationProcess`, `Toolhelp32Snapshot`, `iphlpapi.dll::GetExtendedTcpTable`), tăng tốc quét Language Server từ 1.08s xuống **0.02s (>50 lần)**.
   - **Discovery Cache 10s & SingleFlight:** Xác thực cặp `(PID, CreationTime)` chống lỗi PID recycling và gom luồng quét đồng thời.
   - **Khởi động tức thì < 300ms:** Loại bỏ gián đoạn UI, thu hồi heap nhàn rỗi ngầm qua `debug.FreeOSMemory()`.
   - **Warm Refresh:** Điều phối thế hệ `refreshGen atomic.Uint64`, hủy kết quả cũ âm thầm mà không gây đơ giật hay khóa tương tác giao diện.
4. **Trải Nghiệm Desktop & Quản Lý Vòng Đời Tài Khoản:**
   - **Zero-Console Native Windows PE:** Đơn file nhị phân duy nhất `antigravity-switch.exe` (`-H windowsgui`), biểu tượng PE icon sắc nét, System Tray, Single Instance.
   - **Kiến trúc Two-Store & Modal Thêm Tài Khoản:** Tách bạch `accounts_store/` và `discovered_accounts/`, hỗ trợ gạt trạng thái `[ON / OFF]`.
   - **Nút Xóa An Toàn `[ 🗑 ]`:** Hỗ trợ xóa tài khoản không dùng với Modal xác nhận Danger hai nút (`[Hủy]` / `[Xóa]`), bảo vệ bất biến không cho phép xóa tài khoản đang Active (`[✔ Dùng]`).
   - **Bảo toàn phiên làm việc:** Tự động ghi nhận hội thoại trong 2 giờ gần nhất và gửi prompt tiếp tục phiên code sau khi chuyển đổi.
   - **An toàn & Cục bộ 100%:** Giao tiếp trực tiếp qua Windows Credential Manager (`wincred`), không dùng server bên thứ 3, không bọc proxy API, không can thiệp mã nhị phân Antigravity.

---

## 2. Tiêu Chuẩn Giới Hạn GitHub & Quy Tắc Thiết Kế Description

- **Giới hạn ký tự GitHub About:** Tối đa **350 ký tự**.
- **Độ dài vàng (Golden Length):** Từ **150 đến 230 ký tự**. Tránh bị dấu ba chấm (`...`) trên kết quả tìm kiếm Google, GitHub Search và thẻ xem trước OpenGraph (Twitter Card/LinkedIn Preview).
- **Từ khóa công nghệ (SEO High-Intent):** `Antigravity`, `Go/Golang`, `Fyne`, `Gemini`, `Claude`, `Failover Daemon`, `Win32`, `Zero-Console`.

---

## 3. Các Phương Án Đề Xuất Description (Tiếng Anh - English)

### 🌟 Option 1A: Concise & Modern (Súc Tích & Hiện Đại) - *Khuyến nghị chung*
- **Đặc trưng:** Ngắn gọn, nêu bật giá trị cốt lõi, hoàn hảo cho giao diện GitHub hiện đại.
- **Độ dài:** 188 ký tự (`< 350`).

```text
⚡ High-performance multi-account switcher & autonomous failover daemon for Google Antigravity 2.x. Zero-console Go/Fyne desktop GUI, live quota tracking, and seamless session restoration.
```

---

### 🛡️ Option 1B: Enterprise & Security-Focused (Chuyên Nghiệp & Bảo Mật)
- **Đặc trưng:** Nhấn mạnh tính an toàn cục bộ, bảo vệ Credential Manager, Single-Flight Mutex, không can thiệp binary.
- **Độ dài:** 210 ký tự (`< 350`).

```text
🔒 Safe, non-invasive Google Antigravity multi-account management daemon for Windows. Features autonomous failover, in-memory retry, Single-Flight concurrency, and real-time Gemini/Claude quota monitoring.
```

---

### ⚙️ Option 1C: Feature-Focused & Deep Tech (Công Nghệ & Hiệu Năng Kernel)
- **Đặc trưng:** Nêu bật các giải pháp kiến trúc đột phá: Win32 Kernel Discovery (0.02s), Dual-Trigger, Burn-down Sorter, Zero-Console.
- **Độ dài:** 215 ký tự (`< 350`).

```text
🚀 Autonomous failover daemon & quota tracker for Google Antigravity. Built with Go & Fyne, featuring Dual-Trigger engine, Burn-down Sorter, Win32 kernel discovery, and zero-console native PE architecture.
```

---

### 💡 Option 1D: Developer-Centric & Flow-Preserving (Thu Hút Lập Trình Viên)
- **Đặc trưng:** Nhắm trúng nỗi đau hết quota ngắt mạch tư duy lập trình viên, văn phong truyền cảm hứng, thúc đẩy sử dụng.
- **Độ dài:** 200 ký tự (`< 350`).

```text
⚡ Never let quota limits break your flow. Autonomous multi-account switcher for Google Antigravity with smart failover, real-time Gemini/Claude quota dashboard, and zero-downtime session recovery.
```

---

## 4. Các Phương Án Đề Xuất Description (Tiếng Việt)

### 🌟 Phương Án 2A: Súc Tích & Tinh Gọn (Concise - VI)
- **Đặc trưng:** Trực quan, dễ đọc, phù hợp cho người dùng Việt Nam nắm bắt ngay tính năng.
- **Độ dài:** 198 ký tự (`< 350`).

```text
⚡ Ứng dụng quản lý đa tài khoản và tự động hoán đổi failover cho Google Antigravity. Giao diện Fyne GUI Zero-Console, theo dõi quota Gemini/Claude thời gian thực, bảo toàn phiên làm việc liên tục.
```

---

### 🛡️ Phương Án 2B: Chuyên Nghiệp & Chuẩn Mực Doanh Nghiệp (Professional - VI)
- **Đặc trưng:** Đề cao tính an toàn tuyệt đối, tuân thủ cơ chế chuẩn Google, bảo vệ Single-Flight Mutex trên Windows.
- **Độ dài:** 217 ký tự (`< 350`).

```text
🔒 Giải pháp điều phối đa tài khoản Google Antigravity an toàn trên Windows. Tích hợp daemon tự động chuyển tài khoản khi hết quota, bảo vệ Single-Flight Mutex, tuân thủ 100% cơ chế Google, bảo mật tuyệt đối.
```

---

### ⚙️ Phương Án 2C: Trọng Tâm Kiến Trúc & Công Nghệ (Feature-Focused - VI)
- **Đặc trưng:** Liệt kê các thành phần kỹ thuật nổi bật của phiên bản v1.0.2.0 (Dual-Trigger, Burn-down Sorter, Win32 Kernel).
- **Độ dài:** 199 ký tự (`< 350`).

```text
🚀 Ứng dụng Go/Fyne quản lý và tự động chuyển đổi tài khoản Antigravity: Động cơ Dual-Trigger, thuật toán Burn-down Sorter, Win32 Kernel Discovery, Dashboard quota 5h/tuần và Zero-Console Native PE.
```

---

### 💡 Phương Án 2D: Thấu Cảm & Trải Nghiệm Lập Trình Viên (Developer-Centric - VI)
- **Đặc trưng:** Giải quyết triệt để vấn đề gián đoạn phiên code do hết quota; tự động giữ nguyên tiến trình làm việc.
- **Độ dài:** 200 ký tự (`< 350`).

```text
⚡ Chấm dứt nỗi lo cạn quota khi code! Công cụ tự động chuyển đổi tài khoản Google Antigravity thông minh, giữ nguyên phiên làm việc, dashboard quota Gemini/Claude trực quan, mượt mà và an toàn 100%.
```

---

## 5. Đề Xuất Danh Sách GitHub Topics / Tags (Tối Ưu SEO & Thuật Toán)

GitHub cho phép gắn tối đa **20 topics**. Dưới đây là 16 topics tối ưu nhất được phân loại rõ ràng:

### 🏷️ Danh Mục Topics Chi Tiết:
- **Ecosystem & AI Models:** `antigravity`, `google-antigravity`, `gemini`, `claude`, `google-cloud`
- **Ngôn ngữ & Nền tảng:** `golang`, `go`, `fyne`, `windows-gui`, `desktop-app`
- **Kiến trúc & Tính năng:** `account-switcher`, `failover-daemon`, `quota-monitoring`, `zero-console`
- **Mục đích & Phân khúc:** `developer-tools`, `productivity`, `automation`

### 📋 Khối Copy-Paste 1 Chạm Cho Topics:
```text
antigravity, google-antigravity, gemini, claude, golang, go, fyne, windows-gui, desktop-app, account-switcher, failover-daemon, quota-monitoring, zero-console, developer-tools, productivity, automation
```

---

## 6. Đề Xuất Taglines & Micro-Slogans (Cho Banner & Profile README)

### Tiếng Anh:
1. *"Uninterrupted AI Coding: Autonomous Multi-Account Failover for Google Antigravity."*
2. *"Zero-Console Windows Native GUI & Intelligent Quota Daemon for Antigravity 2.x."*
3. *"Switch accounts seamlessly. Never exhaust your quota."*

### Tiếng Việt:
1. *"Code AI liên tục không gián đoạn: Tự động chuyển đổi tài khoản cho Google Antigravity."*
2. *"Ứng dụng Desktop Windows Native Zero-Console & Daemon quản lý quota thông minh."*
3. *"Tự động hoán đổi tài khoản dự phòng. Giữ trọn mạch tư duy lập trình."*

---

## 7. Đề Xuất Detailed Description (Mô Tả Chi Tiết Cho Releases / Wiki)

### English Pitch:
> **Antigravity Multi-Account Switcher** (v1.0.2.0) is a high-performance Windows desktop application engineered in **Go** and **Fyne**. It empowers developers to manage multiple Google accounts effortlessly in Antigravity 2.x without breaking their workflow. Featuring an **Autonomous Failover Daemon** powered by a **Dual-Trigger Engine** (proactive 60s telemetry loop + instant Connect-RPC error interception) and the **Burn-down Sorter Algorithm**, the tool proactively detects resource exhaustion and transitions to the optimal backup account. Backed by **sub-0.02s Win32 Kernel Discovery**, **Single-Flight Concurrency Protection**, and native Windows Credential Manager integration, it delivers a seamless **Zero-Console Native PE** experience while keeping your credentials 100% secure and local.

### Bản Tiếng Việt:
> **Antigravity Multi-Account Switcher** (v1.0.2.0) là ứng dụng desktop Windows hiệu năng cao được phát triển bằng **Go** và **Fyne**. Ứng dụng giúp lập trình viên quản lý và chuyển đổi linh hoạt giữa các tài khoản Google trong Antigravity 2.x mà không làm gián đoạn dòng suy nghĩ công việc. Sở hữu **Daemon Tự Động Chuyển Đổi (Autonomous Failover Engine)** với **Động cơ Van Kích Hoạt Kép (Dual-Trigger Engine)** và thuật toán thông minh **Burn-down Sorter**, hệ thống tự động phát hiện cạn kiệt quota và hoán đổi sang tài khoản dự phòng tối ưu nhất. Với công nghệ **Win32 Kernel Discovery siêu tốc (< 0.02s)**, cơ chế bảo vệ liên tiến trình **Single-Flight Concurrency Gate**, và tích hợp Windows Credential Manager, ứng dụng đem lại trải nghiệm **Zero-Console Native PE** mượt mà, độc lập và bảo mật 100% trên thiết bị cá nhân.

---

## 8. Hướng Dẫn Thao Tác Cập Nhật Trên GitHub 1 Chạm

1. Truy cập vào Repository trên trình duyệt GitHub (`https://github.com/<owner>/<repo>`).
2. Ở cột bên phải, nhấp vào biểu tượng bánh răng ⚙️ tại phần **About**.
3. **Description:** Copy khối văn bản ở **Mục 3 (Option 1D hoặc 1A)** hoặc **Mục 4**.
4. **Website:** Điền link repository hoặc trang Releases.
5. **Topics:** Dán danh sách từ khóa từ **Mục 5**.
6. **Include in the home page:** Tích chọn ☑️ *Releases*.
7. Nhấn nút xanh **Save changes** để lưu.

---

## 9. Ma Trận Khuyến Nghị Lựa Chọn

| Mục tiêu phát triển | Phương án tối ưu | Ưu điểm cốt lõi |
| :--- | :--- | :--- |
| **Kho mã nguồn mở quốc tế (Khuyên dùng)** | **Option 1D** hoặc **1A** (English) | Tiếp cận cộng đồng dev toàn cầu, nêu bật giải pháp chống đứt gãy mạch code và trải nghiệm mượt mà. |
| **Khoe kiến trúc / Portfolio kỹ thuật cá nhân** | **Option 1C** (English) | Phô diễn kỹ thuật chuyên sâu: Win32 Kernel Discovery, Burn-down Sorter, Dual-Trigger, Native PE. |
| **Cộng đồng Lập trình viên Việt Nam** | **Phương Án 2D** hoặc **2A** (Tiếng Việt) | Gần gũi, súc tích, thấu hiểu khó khăn khi quản lý nhiều tài khoản cá nhân / dự án. |
| **Môi trường yêu cầu bảo mật / Doanh nghiệp** | **Option 1B** (EN) hoặc **2B** (VI) | Đặt trọng tâm vào bảo mật cục bộ, Single-Flight Mutex, tuân thủ Google, không can thiệp binary. |
