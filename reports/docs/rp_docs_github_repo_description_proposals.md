# Báo Cáo Đề Xuất Description & Metadata Cho GitHub Repository

**Tài liệu:** `reports/docs/rp_docs_github_repo_description_proposals.md`  
**Dự án:** Antigravity Multi-Account Switcher & Autonomous Failover Daemon  
**Phiên bản mã nguồn thực tế:** `v1.0.2.0` (Căn cứ `internal/version/version.go`, không dùng README cũ)  
**Người thực hiện:** Technical Writer Lane  
**Ngày cập nhật:** 23/10/2026  
**Thẩm quyền:** Human Owner Directive (Tối ưu hóa Description chuẩn 350 ký tự GitHub, khảo sát trực tiếp mã nguồn thực tế, không dùng README cũ), CEO Directive (Thread ID: `ebad9771-35b4-4afb-955f-1b75612f7630`)  

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

## 2. Tiêu Chuẩn Giới Hạn GitHub 350 Ký Tự (350 Characters Limit)

Trường **About Description** trên GitHub có giới hạn cứng là **350 ký tự** (khi nhập trên web, GitHub hiển thị bộ đếm ngược: *"350 characters remaining"*).

Tài liệu này phân chia các đề xuất thành **3 nhóm độ dài rõ rệt** để Human Owner tùy chọn:
- **Nhóm Maximized (320 – 335 ký tự, còn lại 15 – 30 ký tự):** Khai thác tối đa không gian cho phép, đưa đầy đủ các tính năng công nghệ đỉnh cao và giá trị cốt lõi.
- **Nhóm Balanced Sweet Spot (190 – 220 ký tự):** Độ dài tiêu chuẩn vàng, vừa vặn không bị cắt ba chấm (`...`) trên kết quả tìm kiếm Google & GitHub search.
- **Nhóm Compact (150 – 180 ký tự):** Súc tích, ngắn gọn, vào thẳng công năng chính.

---

## 3. Các Phương Án Đề Xuất Description (Tiếng Anh - English)

### 🚀 [MAXIMIZED] Nhóm Tận Dụng Tối Đa Giới Hạn 350 Ký Tự (320 - 335 chars)

#### 🌟 Option 1A-Max: Feature-Focused & Deep Tech (Kỹ thuật Toàn diện)
- **Độ dài:** **332 ký tự** *(Còn lại 18 ký tự)*
- **Định vị:** Phô diễn toàn bộ hệ thống tính năng cao cấp từ Dual-Trigger, Win32 Kernel, Burn-down Sorter đến Single-Flight Mutex.

```text
⚡ Autonomous multi-account switcher & failover daemon for Google Antigravity 2.x. Features Dual-Trigger engine (telemetry + Connect-RPC 429), in-memory retry, Burn-down Sorter, Win32 kernel discovery, Single-Flight Mutex, and live Gemini/Claude quota tracking in a zero-console Go/Fyne desktop GUI. Zero downtime, 100% safe & local.
```

---

#### 💡 Option 1B-Max: Developer Flow & Zero Downtime (Thu hút Lập trình viên)
- **Độ dài:** **333 ký tự** *(Còn lại 17 ký tự)*
- **Định vị:** Nhấn mạnh bảo toàn mạch tư duy code không lo gián đoạn do hết quota, kết hợp đầy đủ daemon tự động.

```text
⚡ Never let quota exhaustion break your flow. Autonomous multi-account switcher for Google Antigravity 2.x with smart failover daemon, proactive & reactive triggers, in-memory retry, Burn-down Sorter, live Gemini & Claude quota monitoring, and seamless session recovery. 100% safe, local, and zero-console Go/Fyne desktop experience.
```

---

#### 🛡️ Option 1C-Max: Enterprise & Security-First (Bảo Mật & Chuẩn Mực Doanh Nghiệp)
- **Độ dài:** **327 ký tự** *(Còn lại 23 ký tự)*
- **Định vị:** Đề cao tính an toàn cục bộ, bảo vệ Credential Manager, Single-Flight Mutex, không can thiệp binary.

```text
🔒 Safe, non-invasive Google Antigravity 2.x multi-account management daemon for Windows. Features autonomous failover, Dual-Trigger engine, in-memory retry, Single-Flight concurrency gate, and real-time Gemini/Claude quota tracking in a zero-console Go/Fyne native PE app. 100% on-device, adheres strictly to Google standards.
```

---

### ⚖️ [BALANCED] Nhóm Cân Bằng Chuẩn Vàng (190 - 220 chars)

#### Option 1D-Balanced: Modern & Punchy (Hiện Đại & Trực Diện)
- **Độ dài:** **188 ký tự** *(Còn lại 162 ký tự)*

```text
⚡ High-performance multi-account switcher & autonomous failover daemon for Google Antigravity 2.x. Zero-console Go/Fyne desktop GUI, live quota tracking, and seamless session restoration.
```

#### Option 1E-Balanced: Developer-Centric (Lập Trình Viên)
- **Độ dài:** **200 ký tự** *(Còn lại 150 ký tự)*

```text
⚡ Never let quota limits break your flow. Autonomous multi-account switcher for Google Antigravity with smart failover, real-time Gemini/Claude quota dashboard, and zero-downtime session recovery.
```

---

## 4. Các Phương Án Đề Xuất Description (Tiếng Việt)

### 🚀 [MAXIMIZED] Nhóm Tận Dụng Tối Đa Giới Hạn 350 Ký Tự (320 - 335 ký tự)

#### 🌟 Phương Án 2A-Max: Điểm Nhấn Tính Năng Kỹ Thuật Toàn Diện
- **Độ dài:** **333 ký tự** *(Còn lại 17 ký tự)*
- **Định vị:** Đầy đủ các điểm sáng công nghệ: Dual-Trigger, In-memory Retry, Burn-down Sorter, Win32 Kernel Discovery 0.02s, Single-Flight Mutex.

```text
⚡ Quản lý đa tài khoản & daemon tự động chuyển đổi thông minh cho Google Antigravity 2.x. Tích hợp động cơ kích hoạt kép Dual-Trigger, phục hồi in-memory retry, thuật toán Burn-down Sorter, Win32 Kernel Discovery siêu tốc 0.02s, Single-Flight Mutex và dashboard quota Gemini/Claude 5h/tuần. Giao diện Fyne Zero-Console, an toàn 100%.
```

---

#### 💡 Phương Án 2B-Max: Trải Nghiệm Lập Trình Viên & Giữ Trọn Mạch Code
- **Độ dài:** **331 ký tự** *(Còn lại 19 ký tự)*
- **Định vị:** Chấm dứt sự cố đứt gãy công việc do hết quota, chuyển đổi mượt mà, phục hồi phiên tức thì.

```text
⚡ Giữ trọn mạch code không lo cạn quota! Ứng dụng quản lý đa tài khoản và tự động hoán đổi failover cho Google Antigravity 2.x. Tự động chuyển tài khoản khi chạm ngưỡng hoặc gặp lỗi 429, phục hồi phiên làm việc tức thì, theo dõi quota Gemini/Claude thời gian thực. Giao diện Fyne GUI Zero-Console siêu nhẹ, an toàn và bảo mật 100%.
```

---

#### 🛡️ Phương Án 2C-Max: Chuyên Nghiệp & Bảo Mật Doanh Nghiệp
- **Độ dài:** **331 ký tự** *(Còn lại 19 ký tự)*
- **Định vị:** An toàn dữ liệu, Single-Flight Mutex, xác thực Windows Credential Manager cục bộ, tuân thủ 100% cơ chế Google.

```text
🔒 Giải pháp quản lý và điều phối đa tài khoản Google Antigravity 2.x an toàn trên Windows. Tích hợp daemon tự động chuyển tài khoản khi hết quota, thuật toán Burn-down Sorter, bảo vệ Single-Flight Mutex, xác thực Windows Credential Manager cục bộ, không can thiệp binary, tuân thủ 100% chuẩn Google. Fyne GUI Zero-Console mượt mà.
```

---

### ⚖️ [BALANCED] Nhóm Cân Bằng Chuẩn Vàng Tiếng Việt (190 - 220 ký tự)

#### Phương Án 2D-Balanced: Súc Tích & Tinh Gọn
- **Độ dài:** **198 ký tự** *(Còn lại 152 ký tự)*

```text
⚡ Ứng dụng quản lý đa tài khoản và tự động hoán đổi failover cho Google Antigravity. Giao diện Fyne GUI Zero-Console, theo dõi quota Gemini/Claude thời gian thực, bảo toàn phiên làm việc liên tục.
```

#### Phương Án 2E-Balanced: Hướng Lập Trình Viên
- **Độ dài:** **200 ký tự** *(Còn lại 150 ký tự)*

```text
⚡ Chấm dứt nỗi lo cạn quota khi code! Công cụ tự động chuyển đổi tài khoản Google Antigravity thông minh, giữ nguyên phiên làm việc, dashboard quota Gemini/Claude trực quan, mượt mà và an toàn 100%.
```

---

## 5. Đề Xuất Danh Sách GitHub Topics / Tags (Tối Ưu SEO & Thuật Toán)

GitHub cho phép gắn tối đa **20 topics**. Dưới đây là 16 topics tối ưu nhất:

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
3. **Description:** Copy khối văn bản ở **Mục 3 (Option 1A-Max hoặc 1B-Max)** hoặc **Mục 4**.
4. **Website:** Điền link repository hoặc trang Releases.
5. **Topics:** Dán danh sách từ khóa từ **Mục 5**.
6. **Include in the home page:** Tích chọn ☑️ *Releases*.
7. Nhấn nút xanh **Save changes** để lưu.

---

## 9. Bảng So Sánh & Khuyến Nghị Lựa Chọn

| Mục tiêu phát triển | Phương án tối ưu | Số ký tự | Ký tự còn lại (trên 350) |
| :--- | :--- | :---: | :---: |
| **Tận dụng tối đa 350 ký tự (Tech & Features - EN)** | **Option 1A-Max** | **332** | **18 còn lại** |
| **Tận dụng tối đa 350 ký tự (Developer Flow - EN)** | **Option 1B-Max** | **333** | **17 còn lại** |
| **Tận dụng tối đa 350 ký tự (Enterprise Safe - EN)** | **Option 1C-Max** | **327** | **23 còn lại** |
| **Tận dụng tối đa 350 ký tự (Tính năng toàn diện - VI)** | **Phương Án 2A-Max** | **333** | **17 còn lại** |
| **Tận dụng tối đa 350 ký tự (Mạch code liên tục - VI)** | **Phương Án 2B-Max** | **331** | **19 còn lại** |
| **Chuẩn vàng cân bằng không bị cắt ngắn (EN)** | **Option 1D-Balanced** | **188** | **162 còn lại** |
| **Chuẩn vàng cân bằng không bị cắt ngắn (VI)** | **Phương Án 2D-Balanced** | **198** | **152 còn lại** |
