# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Giai Đoạn 2: Tích Hợp Tab Codex Trên Dashboard UI, Khởi Tạo Bộ Khung File Backend Codex (Soi Gương Antigravity) & Cơ Chế Chuyển Tab Động (Contextual Footer)

- **Mã tài liệu:** `rp_system-architect_261029_092000_by_antigravity_phase_2_codex_ui_tab_switching_and_backend_skeleton_mirror`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-29
- **Thẩm quyền phê duyệt:** Trình Human Owner (Chủ tịch) xem xét và phê duyệt
- **Trạng thái:** DRAFT BLUEPRINT FOR OWNER REVIEW (Chờ Chủ tịch đọc và duyệt, TUYỆT ĐỐI CHƯA GỬI CEO)

---

# PHẦN 1: TỔNG QUAN CHIẾN LƯỢC & TRIẾT LÝ THIẾT KẾ CỦA CHỦ TỊCH

### 1.1. Triết lý tương đồng về Trải nghiệm Người dùng (UX Parity)
Dù là quản lý tài khoản cho **Google Antigravity** hay **OpenAI Codex**, bản chất người dùng đều thao tác trên một hệ thống **Auto Switcher**:
- Đều có nhu cầu xem danh sách tài khoản, chuyển đổi tài khoản đang hoạt động (Active).
- Đều có nhu cầu kiểm tra Quota / Hạn mức, tự động đổi tài khoản khi cạn kiệt Quota.
- Đều có nhu cầu theo dõi trạng thái kết nối daemon của IDE ở góc trái thanh đáy và số hiệu phiên bản ở góc phải thanh đáy.

### 1.2. Chiến lược thiết kế Backend Soi Gương (Mirror Architecture)
Chủ tịch đã chỉ đạo rõ ràng:
> *"Về cơ bản thì UX sử dụng Codex hay Antigravity đều giống nhau. Vì thế, tổ chức thư mục và file cũng sẽ giống nhau. Cần tạo hệ thống thư mục và file giống y hệt Antigravity (nhưng nội dung rỗng trước). Phase tiếp theo mới cho code từng phần vào các file, code nào tận dụng được thì copy nguyên từ Antigravity sang, code nào tận dụng 1 phần copy 1 phần."*

Do đó, tại Giai đoạn 2 này:
1. **Dưới Backend**: Tạo đầy đủ toàn bộ hệ thống thư mục và **đúng 50 file khung rỗng (skeleton files)** của Codex tại `internal/engines/codex/`, phản chiếu 1-1 với 50 file của `internal/engines/antigravity/`. Mỗi file chỉ chứa dòng khai báo `package <name>` hợp lệ của Go, hoàn toàn chưa có bất kỳ dòng code logic thực thi nào.
2. **Trên Giao diện UI**:
   - Thêm Tab **Codex** lên thanh Sidebar bên trái (sử dụng **Logo thật chính thức của OpenAI Codex / Codex IDE**).
   - Xây dựng cơ chế chuyển đổi View động (Stack Show/Hide) giữa Antigravity và Codex.
   - Dựng giao diện Content View cho Codex (Toolbar + Khung Empty State/Placeholder sang trọng).
   - Triển khai **Contextual Footer Bar**: Footer tự động thay đổi hiển thị theo Tab đang chọn (ở Tab Antigravity hiện daemon & version của Antigravity; ở Tab Codex hiện daemon & version của Codex).

---

# PHẦN 2: BỐ CỤC GIAO DIỆN GIAI ĐOẠN 2 (TARGET LAYOUT)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────────┐
│ TOP BAR: [AS] Auto Switcher ────────────── (Kéo cửa sổ) ────────────── [VN|EN] [—] [□] [✕]        │
├───────────────┬──────────────────────────────────────────────────────────────────────────────────┤
│ SIDEBAR       │ CONTENT AREA (Chuyển đổi tức thì theo Tab đang chọn)                             │
│ (136px)       │                                                                                  │
│ [Logo] Anti.. │ ► KHI CHỌN TAB "ANTIGRAVITY":                                                    │
│               │   - Toolbar Antigravity: [+ Thêm] [F5] [Quota] [Ghim] [Ẩn] [Tự động] [Theme ☀️]  │
│ [Logo] Codex  │   - Lưới 3 Cột Card Tài Khoản Antigravity (280x130 px, cuộn dọc độc lập)         │
│ (Mới thêm)    │                                                                                  │
│               │ ► KHI CHỌN TAB "CODEX" (UI ONLY):                                                │
│               │   - Toolbar Codex: [+ Thêm tài khoản Codex] [Làm mới] [Ghim]                    │
│               │   - Khung Empty State / Placeholder Codex sang trọng                             │
├───────────────┴──────────────────────────────────────────────────────────────────────────────────┤
│ FOOTER BAR: [Trạng thái Daemon Contextual] ───────────────────── [Version Contextual]            │
│  • Ở Tab Antigravity: [Antigravity Daemon: Đã kết nối] ──────── [v1.0.3.0]                       │
│  • Ở Tab Codex:       [Daemon Codex: Chưa kết nối] ──────────── [Codex v1.0.0]                   │
└──────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

# PHẦN 3: BẢN ĐỒ CHI TIẾT 1-1 TOÀN BỘ 50 FILE KHUNG RỖNG CỦA CODEX BACKEND

*Tất cả 50 file dưới đây sẽ được khởi tạo trong `internal/engines/codex/`, cấu trúc soi gương 1-1 với Antigravity. Mỗi file chỉ chứa 1 dòng khai báo `package <name>` hợp lệ, hoàn toàn không có code logic bên trong*:

### 1. Phân hệ Scanner (`internal/engines/codex/scanner/` — 2 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 1 | `internal/engines/codex/scanner/chrome_scanner.go` | `package scanner` |
| 2 | `internal/engines/codex/scanner/chrome_scanner_test.go` | `package scanner` |

### 2. Phân hệ Reloader (`internal/engines/codex/reloader/` — 6 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 3 | `internal/engines/codex/reloader/active_threads.go` | `package reloader` |
| 4 | `internal/engines/codex/reloader/active_threads_test.go` | `package reloader` |
| 5 | `internal/engines/codex/reloader/continuation.go` | `package reloader` |
| 6 | `internal/engines/codex/reloader/continuation_test.go` | `package reloader` |
| 7 | `internal/engines/codex/reloader/session_reloader.go` | `package reloader` |
| 8 | `internal/engines/codex/reloader/reloader_test.go` | `package reloader` |

### 3. Phân hệ Telemetry (`internal/engines/codex/telemetry/` — 8 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 9 | `internal/engines/codex/telemetry/client.go` | `package telemetry` |
| 10 | `internal/engines/codex/telemetry/cache.go` | `package telemetry` |
| 11 | `internal/engines/codex/telemetry/cache_test.go` | `package telemetry` |
| 12 | `internal/engines/codex/telemetry/loop.go` | `package telemetry` |
| 13 | `internal/engines/codex/telemetry/pool.go` | `package telemetry` |
| 14 | `internal/engines/codex/telemetry/tier.go` | `package telemetry` |
| 15 | `internal/engines/codex/telemetry/quota_test.go` | `package telemetry` |
| 16 | `internal/engines/codex/telemetry/telemetry_test.go` | `package telemetry` |

### 4. Phân hệ Connector (`internal/engines/codex/connector/` — 10 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 17 | `internal/engines/codex/connector/discovery.go` | `package connector` |
| 18 | `internal/engines/codex/connector/discovery_test.go` | `package connector` |
| 19 | `internal/engines/codex/connector/endpoints.go` | `package connector` |
| 20 | `internal/engines/codex/connector/agent_state_stream.go` | `package connector` |
| 21 | `internal/engines/codex/connector/api_client.go` | `package connector` |
| 22 | `internal/engines/codex/connector/automation_rpc.go` | `package connector` |
| 23 | `internal/engines/codex/connector/automation_rpc_test.go` | `package connector` |
| 24 | `internal/engines/codex/connector/oauth_refresher.go` | `package connector` |
| 25 | `internal/engines/codex/connector/oauth_refresher_test.go` | `package connector` |
| 26 | `internal/engines/codex/connector/connector_test.go` | `package connector` |

### 5. Phân hệ Model (`internal/engines/codex/model/` — 2 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 27 | `internal/engines/codex/model/types.go` | `package model` |
| 28 | `internal/engines/codex/model/types_test.go` | `package model` |

### 6. Phân hệ Store (`internal/engines/codex/store/` — 5 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 29 | `internal/engines/codex/store/account_store.go` | `package store` |
| 30 | `internal/engines/codex/store/account_store_test.go` | `package store` |
| 31 | `internal/engines/codex/store/discovered_store.go` | `package store` |
| 32 | `internal/engines/codex/store/discovered_store_test.go` | `package store` |
| 33 | `internal/engines/codex/store/sanitizer.go` | `package store` |

### 7. Phân hệ Manager (`internal/engines/codex/manager/` — 6 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 34 | `internal/engines/codex/manager/account_manager.go` | `package manager` |
| 35 | `internal/engines/codex/manager/account_manager_test.go` | `package manager` |
| 36 | `internal/engines/codex/manager/manager_test.go` | `package manager` |
| 37 | `internal/engines/codex/manager/timeline.go` | `package manager` |
| 38 | `internal/engines/codex/manager/timeline_test.go` | `package manager` |
| 39 | `internal/engines/codex/manager/two_store_test.go` | `package manager` |

### 8. Phân hệ Automation (`internal/engines/codex/automation/` — 12 files rỗng)
| STT | Tên file khung của Codex | Dòng khai báo package |
|:---:|---|---|
| 40 | `internal/engines/codex/automation/controller.go` | `package automation` |
| 41 | `internal/engines/codex/automation/controller_test.go` | `package automation` |
| 42 | `internal/engines/codex/automation/model_quota.go` | `package automation` |
| 43 | `internal/engines/codex/automation/model_quota_test.go` | `package automation` |
| 44 | `internal/engines/codex/automation/retry.go` | `package automation` |
| 45 | `internal/engines/codex/automation/retry_test.go` | `package automation` |
| 46 | `internal/engines/codex/automation/sorter.go` | `package automation` |
| 47 | `internal/engines/codex/automation/sorter_test.go` | `package automation` |
| 48 | `internal/engines/codex/automation/switch_gate_windows.go` | `package automation` |
| 49 | `internal/engines/codex/automation/switch_gate_windows_test.go` | `package automation` |
| 50 | `internal/engines/codex/automation/automation_e2e_test.go` | `package automation` |
| 51 | `internal/engines/codex/automation/runtime_regression_test.go` | `package automation` |

---

# PHẦN 4: THIẾT KẾ CHI TIẾT TẦNG GIAO DIỆN (UI IMPLEMENTATION)

### 4.1. Nhúng Logo Thật của OpenAI Codex / Codex IDE
- Đặt file ảnh logo thật: `internal/ui/assets/codex_logo.png` (transparent PNG, chuẩn nhận diện của OpenAI Codex).
- Khai báo tại `internal/ui/assets.go`:
  ```go
  //go:embed assets/codex_logo.png
  var codexLogoBytes []byte

  // CodexLogoResource cung cấp logo chính thức của OpenAI Codex cho thanh Sidebar.
  var CodexLogoResource = fyne.NewStaticResource("codex_logo.png", codexLogoBytes)
  ```

### 4.2. Bổ sung Tab Codex trên Sidebar (`internal/ui/sidebar.go`)
- Thêm trường `codexItem *sidebarItem` vào struct `SidebarContainer`.
- Khởi tạo `codexItem` với `CodexLogoResource`, nhãn `d.NavCodex`, mặc định `isActive: false`.
- Thêm `sb.codexItem` vào `itemsBox` ngay dưới `sb.antigravityItem`.
- Xây dựng hàm `SwitchTab(target NavTab)`:
  - Khi chọn `NavTabAntigravity`: `antigravityItem.SetActive(true)`, `codexItem.SetActive(false)`.
  - Khi chọn `NavTabCodex`: `antigravityItem.SetActive(false)`, `codexItem.SetActive(true)`.

### 4.3. Dựng Content View cho Tab Codex (`internal/ui/window.go` & `app.go`)
- Tạo container `codexContent`:
  1. **Toolbar Codex (Phía trên)**:
     - Nút `+ Thêm tài khoản Codex` (tạm thời disabled hoặc thông báo toast/dialog "Tính năng sẽ mở ở bản cập nhật tiếp theo").
     - Nút `Làm mới` (icon refresh).
     - Nút `Ghim 📌`.
  2. **Vùng trung tâm (Empty State / Placeholder sang trọng)**:
     - Biểu tượng Logo Codex lớn ở giữa.
     - Tiêu đề: **"Codex Account Switcher"** (in đậm, cỡ lớn).
     - Dòng mô tả: *"Chế độ quản lý tài khoản Codex đang được chuẩn bị tích hợp (Coming Soon)"*.
     - Đường viền bo góc tinh tế theo `ThemePalette.BorderSubtle` và `ThemePalette.CardBg`.
- Vùng thân bên phải `bodyLayout`:
  ```go
  contentStack := container.NewStack(antigravityContent, codexContent)
  bodyLayout := container.NewBorder(nil, nil, a.sidebar, nil, contentStack)
  ```
  - Khi ở Tab Antigravity: `antigravityContent.Show()`, `codexContent.Hide()`.
  - Khi ở Tab Codex: `antigravityContent.Hide()`, `codexContent.Show()`.

### 4.4. Xây dựng Contextual Footer Bar Động
- Trong struct `App`:
  - Giữ nguyên `serverStatusLabel` và `versionLabel` cho Antigravity.
  - Bổ sung `codexStatusLabel` (`"Daemon Codex: Chưa kết nối"`) và `codexVersionLabel` (`"Codex v1.0.0"`).
  - Vùng Footer bên trái bọc trong stack: `leftFooterStack := container.NewStack(a.serverStatusLabel, a.codexStatusLabel)`.
  - Vùng Footer bên phải bọc trong stack: `rightFooterStack := container.NewStack(a.versionLabel, a.codexVersionLabel)`.
- Khi chuyển đổi tab:
  - Tab Antigravity: Hiện `serverStatusLabel` và `versionLabel`, ẩn các label của Codex.
  - Tab Codex: Hiện `codexStatusLabel` và `codexVersionLabel`, ẩn các label của Antigravity.

---

# PHẦN 5: NGUYÊN TẮC BẤT BIẾN KIẾN TRÚC CHO GIAI ĐOẠN 2

1. **Official Brand Logo Invariant:** Tuyệt đối không dùng emoji `🤖` cho Tab Codex. Bắt buộc dùng file ảnh logo chính thức của OpenAI Codex / Codex IDE.
2. **Empty Skeleton Invariant:** Toàn bộ 50 file trong `internal/engines/codex/` là file khung rỗng (chỉ có dòng khai báo package), tuyệt đối chưa nhét code logic backend vào trong giai đoạn này.
3. **Zero-Interruption Invariant:** Khi người dùng chuyển sang xem Tab Codex, toàn bộ các tiến trình chạy ngầm của Antigravity (quét quota, tự động switch tài khoản, liveness check) vẫn duy trì hoạt động ngầm liên tục 100%, không bị hủy hay khởi động lại.
4. **Build & Test Parity Invariant:** Toàn bộ 50 file rỗng của Codex và giao diện mới phải được `go test ./...` và script `.\scripts\build_windows.ps1` biên dịch đạt `PASS 100%`.

---

# PHẦN 6: KẾ HOẠCH BÀN GIAO & DUYỆT BÁO CÁO

Báo cáo này được lập nhằm mục đích **trình Chủ tịch xem xét và phê duyệt trước**.
Chỉ khi nào Chủ tịch đọc kỹ, xác nhận đồng ý 100% và **chính thức ra lệnh bằng văn bản "Gửi CEO"**, System Architect mới tiến hành gửi bàn giao lệnh thi công sang cho CEO.
