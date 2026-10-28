# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Giai Đoạn 1B: Tái Cấu Trúc Hệ Thống Thư Mục Mã Nguồn — Tách Biệt Backend Antigravity & Tầng Lõi Dùng Chung (Core) Chuẩn Bị Cho Nền Tảng Đa IDE

- **Mã tài liệu:** `rp_system-architect_261028_220000_by_antigravity_phase_1b_antigravity_backend_directory_refactoring_and_core_isolation`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-28
- **Thẩm quyền phê duyệt:** Human Owner (Chủ tịch) phê duyệt trực tiếp trong phiên thảo luận kỹ thuật
- **Trạng thái:** PRODUCTION-READY BLUEPRINT (Khóa cứng Touch-Map 1-1, bàn giao chìa khóa trao tay cho Coder)
- **Mục tiêu cốt tử:** Tái cấu trúc không gian mã nguồn một cách an toàn, có kiểm soát theo từng bước nhỏ, bảo toàn 100% tính năng hiện có của Antigravity và tạo nền móng vững chắc trước khi bước sang Giai đoạn 2 (Tích hợp Codex).

---

# PHẦN 1: BỐI CẢNH & PHÂN TÍCH RỦI RO KIẾN TRÚC

### 1.1. Hiện trạng và Nguy cơ "Chắp vá"
Hiện tại, toàn bộ mã nguồn backend của ứng dụng đang nằm phẳng tại thư mục `internal/`:
- 8 package gồm: `connector`, `reloader`, `scanner`, `telemetry`, `automation`, `manager`, `model`, `store` được thiết kế phục vụ riêng 100% cho Google Antigravity (gọi daemon `language_server.exe`, quét SQLite session tại `.gemini/antigravity/`, quét Chrome profiles, gọi API Quota Gemini).
- Nếu tiếp tục đưa mã nguồn của Codex vào chung các package này, codebase sẽ rơi vào trạng thái spaghetti code, các struct bị phình to, import chéo hỗn loạn và gây rủi ro phá vỡ tính năng của Antigravity.

### 1.2. Phân tích Rủi ro Khi Tái Cấu Trúc (Refactoring Risk Analysis)
Chủ tịch đã chỉ đạo rõ ràng:
> *"Đây là công việc rất nguy hiểm, tuy chỉ là đổi vị trí các file nhưng nếu khai báo sai đường dẫn mới với số lượng file lớn thế này chỉ cần khai đường dẫn mới sai là gãy ngay. Vì thế phải tiến hành theo từng phase, mỗi phase có số lượng thư mục và file nhất định được chuyển đổi. Trong báo cáo phải có 2 bản đồ file chi tiết đến từng file (Bản đồ Cũ vs Bản đồ Mới), tuyệt đối không được viết tắt kiểu dấu ba chấm `...`"*

Do đó, tài liệu đặc tả này:
1. **Kiểm kê chi tiết 100% toàn bộ 105 files trong `cmd/` và `internal/`** thành bảng đối chiếu 1-1 (không có bất kỳ dòng nào dùng `...`).
2. **Chia nhỏ lộ trình thi công thành 5 tiểu giai đoạn (Sub-phases)**: di dời đến đâu, cập nhật import đến đó, chạy test và build ngay lập tức, PASS mới được sang bước tiếp theo.

---

# PHẦN 2: BẢN ĐỒ CHI TIẾT 1-1 TỪNG FILE ĐƠN LẺ (CURRENT ➔ TARGET MAPPING)

---

### KHỐI 1: TẦNG ENTRYPOINT (`cmd/switcher/` — 4 files)
*Giữ nguyên vị trí các file, chỉ cập nhật đường dẫn import bên trong*:

| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) | Phân loại |
|:---:|---|---|---|
| 1 | `cmd/switcher/main.go` | `cmd/switcher/main.go` | Mã nguồn Go Entrypoint |
| 2 | `cmd/switcher/main_test.go` | `cmd/switcher/main_test.go` | Unit Test Entrypoint |
| 3 | `cmd/switcher/app.rc` | `cmd/switcher/app.rc` | Windows Resource Script |
| 4 | `cmd/switcher/rsrc_windows_amd64.syso` | `cmd/switcher/rsrc_windows_amd64.syso` | Windows Resource Binary |

---

### KHỐI 2: TẦNG LÕI DÙNG CHUNG CHO MỌI IDE (`internal/core/` — 13 files)
*Gom các package độc lập, không phụ thuộc vào bất kỳ IDE nào vào phân vùng `internal/core/`*:

| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) | Phân loại |
|:---:|---|---|---|
| 5 | *(File mới tạo)* | `internal/core/engine.go` | Interface trừu tượng IDEEngine |
| 6 | `internal/wincred/cred.go` | `internal/core/wincred/cred.go` | Win32 Credential API |
| 7 | `internal/wincred/cred_test.go` | `internal/core/wincred/cred_test.go` | Test Win32 Credential API |
| 8 | `internal/wincred/types.go` | `internal/core/wincred/types.go` | Types Win32 Credential |
| 9 | `internal/version/version.go` | `internal/core/version/version.go` | Version Info |
| 10 | `internal/version/version_test.go` | `internal/core/version/version_test.go` | Test Version Info |
| 11 | `internal/i18n/dictionary.go` | `internal/core/i18n/dictionary.go` | Từ điển đa ngữ Struct |
| 12 | `internal/i18n/locale.go` | `internal/core/i18n/locale.go` | Quản lý Locale runtime |
| 13 | `internal/i18n/i18n_test.go` | `internal/core/i18n/i18n_test.go` | Test Đa ngữ |
| 14 | `internal/i18n/locales/en.json` | `internal/core/i18n/locales/en.json` | Dữ liệu ngôn ngữ Tiếng Anh |
| 15 | `internal/i18n/locales/vi.json` | `internal/core/i18n/locales/vi.json` | Dữ liệu ngôn ngữ Tiếng Việt |
| 16 | `internal/store/preferences_store.go` | `internal/core/preferences/preferences_store.go` | Lưu trữ cấu hình app |
| 17 | `internal/store/preferences_store_test.go` | `internal/core/preferences/preferences_store_test.go` | Test lưu trữ cấu hình app |

---

### KHỐI 3: TOÀN BỘ 50 FILE BACKEND CỦA ANTIGRAVITY (`internal/engines/antigravity/`)
*Chuyển toàn bộ 50 file mã nguồn và kiểm thử của Google Antigravity vào namespace riêng biệt*:

#### A. Phân hệ Quét Chrome (Scanner — 2 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 18 | `internal/scanner/chrome_scanner.go` | `internal/engines/antigravity/scanner/chrome_scanner.go` |
| 19 | `internal/scanner/chrome_scanner_test.go` | `internal/engines/antigravity/scanner/chrome_scanner_test.go` |

#### B. Phân hệ Quét Session 2h (Reloader — 6 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 20 | `internal/reloader/active_threads.go` | `internal/engines/antigravity/reloader/active_threads.go` |
| 21 | `internal/reloader/active_threads_test.go` | `internal/engines/antigravity/reloader/active_threads_test.go` |
| 22 | `internal/reloader/continuation.go` | `internal/engines/antigravity/reloader/continuation.go` |
| 23 | `internal/reloader/continuation_test.go` | `internal/engines/antigravity/reloader/continuation_test.go` |
| 24 | `internal/reloader/session_reloader.go` | `internal/engines/antigravity/reloader/session_reloader.go` |
| 25 | `internal/reloader/reloader_test.go` | `internal/engines/antigravity/reloader/reloader_test.go` |

#### C. Phân hệ Quota Cloud Gemini (Telemetry — 8 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 26 | `internal/telemetry/client.go` | `internal/engines/antigravity/telemetry/client.go` |
| 27 | `internal/telemetry/cache.go` | `internal/engines/antigravity/telemetry/cache.go` |
| 28 | `internal/telemetry/cache_test.go` | `internal/engines/antigravity/telemetry/cache_test.go` |
| 29 | `internal/telemetry/loop.go` | `internal/engines/antigravity/telemetry/loop.go` |
| 30 | `internal/telemetry/pool.go` | `internal/engines/antigravity/telemetry/pool.go` |
| 31 | `internal/telemetry/tier.go` | `internal/engines/antigravity/telemetry/tier.go` |
| 32 | `internal/telemetry/quota_test.go` | `internal/engines/antigravity/telemetry/quota_test.go` |
| 33 | `internal/telemetry/telemetry_test.go` | `internal/engines/antigravity/telemetry/telemetry_test.go` |

#### D. Phân hệ Kết nối Language Server LSP (Connector — 10 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 34 | `internal/connector/discovery.go` | `internal/engines/antigravity/connector/discovery.go` |
| 35 | `internal/connector/discovery_test.go` | `internal/engines/antigravity/connector/discovery_test.go` |
| 36 | `internal/connector/endpoints.go` | `internal/engines/antigravity/connector/endpoints.go` |
| 37 | `internal/connector/agent_state_stream.go` | `internal/engines/antigravity/connector/agent_state_stream.go` |
| 38 | `internal/connector/api_client.go` | `internal/engines/antigravity/connector/api_client.go` |
| 39 | `internal/connector/automation_rpc.go` | `internal/engines/antigravity/connector/automation_rpc.go` |
| 40 | `internal/connector/automation_rpc_test.go` | `internal/engines/antigravity/connector/automation_rpc_test.go` |
| 41 | `internal/connector/oauth_refresher.go` | `internal/engines/antigravity/connector/oauth_refresher.go` |
| 42 | `internal/connector/oauth_refresher_test.go` | `internal/engines/antigravity/connector/oauth_refresher_test.go` |
| 43 | `internal/connector/connector_test.go` | `internal/engines/antigravity/connector/connector_test.go` |

#### E. Phân hệ Dữ liệu Đặc thù Token Google (Model — 2 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 44 | `internal/model/types.go` | `internal/engines/antigravity/model/types.go` |
| 45 | `internal/model/types_test.go` | `internal/engines/antigravity/model/types_test.go` |

#### F. Phân hệ Lưu trữ File JSON Tài khoản Google (Store — 5 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 46 | `internal/store/account_store.go` | `internal/engines/antigravity/store/account_store.go` |
| 47 | `internal/store/account_store_test.go` | `internal/engines/antigravity/store/account_store_test.go` |
| 48 | `internal/store/discovered_store.go` | `internal/engines/antigravity/store/discovered_store.go` |
| 49 | `internal/store/discovered_store_test.go` | `internal/engines/antigravity/store/discovered_store_test.go` |
| 50 | `internal/store/sanitizer.go` | `internal/engines/antigravity/store/sanitizer.go` |

#### G. Phân hệ Quản lý Tài khoản Antigravity (Manager — 6 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 51 | `internal/manager/account_manager.go` | `internal/engines/antigravity/manager/account_manager.go` |
| 52 | `internal/manager/account_manager_test.go` | `internal/engines/antigravity/manager/account_manager_test.go` |
| 53 | `internal/manager/manager_test.go` | `internal/engines/antigravity/manager/manager_test.go` |
| 54 | `internal/manager/timeline.go` | `internal/engines/antigravity/manager/timeline.go` |
| 55 | `internal/manager/timeline_test.go` | `internal/engines/antigravity/manager/timeline_test.go` |
| 56 | `internal/manager/two_store_test.go` | `internal/engines/antigravity/manager/two_store_test.go` |

#### H. Phân hệ Tự động Chuyển Đổi Quota Gemini (Automation — 12 files)
| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) |
|:---:|---|---|
| 57 | `internal/automation/controller.go` | `internal/engines/antigravity/automation/controller.go` |
| 58 | `internal/automation/controller_test.go` | `internal/engines/antigravity/automation/controller_test.go` |
| 59 | `internal/automation/model_quota.go` | `internal/engines/antigravity/automation/model_quota.go` |
| 60 | `internal/automation/model_quota_test.go` | `internal/engines/antigravity/automation/model_quota_test.go` |
| 61 | `internal/automation/retry.go` | `internal/engines/antigravity/automation/retry.go` |
| 62 | `internal/automation/retry_test.go` | `internal/engines/antigravity/automation/retry_test.go` |
| 63 | `internal/automation/sorter.go` | `internal/engines/antigravity/automation/sorter.go` |
| 64 | `internal/automation/sorter_test.go` | `internal/engines/antigravity/automation/sorter_test.go` |
| 65 | `internal/automation/switch_gate_windows.go` | `internal/engines/antigravity/automation/switch_gate_windows.go` |
| 66 | `internal/automation/switch_gate_windows_test.go` | `internal/engines/antigravity/automation/switch_gate_windows_test.go` |
| 67 | `internal/automation/automation_e2e_test.go` | `internal/engines/antigravity/automation/automation_e2e_test.go` |
| 68 | `internal/automation/runtime_regression_test.go` | `internal/engines/antigravity/automation/runtime_regression_test.go` |

---

### KHỐI 4: TOÀN BỘ 37 FILE TẦNG GIAO DIỆN (`internal/ui/` — Giữ nguyên vị trí)
*Tất cả 37 file của `internal/ui/` giữ nguyên vị trí trên đĩa, chỉ cập nhật import paths*:

| STT | Đường dẫn CŨ (Hiện tại) | Đường dẫn MỚI (Mục tiêu) | Phân loại |
|:---:|---|---|---|
| 69 | `internal/ui/window.go` | `internal/ui/window.go` | Mã nguồn UI |
| 70 | `internal/ui/sidebar.go` | `internal/ui/sidebar.go` | Mã nguồn UI |
| 71 | `internal/ui/app.go` | `internal/ui/app.go` | Mã nguồn UI (Cập nhật Import) |
| 72 | `internal/ui/widgets.go` | `internal/ui/widgets.go` | Mã nguồn UI |
| 73 | `internal/ui/themes.go` | `internal/ui/themes.go` | Mã nguồn UI |
| 74 | `internal/ui/add_modal.go` | `internal/ui/add_modal.go` | Mã nguồn UI |
| 75 | `internal/ui/quota_badge.go` | `internal/ui/quota_badge.go` | Mã nguồn UI |
| 76 | `internal/ui/compact_button.go` | `internal/ui/compact_button.go` | Mã nguồn UI |
| 77 | `internal/ui/tray.go` | `internal/ui/tray.go` | Mã nguồn UI |
| 78 | `internal/ui/assets.go` | `internal/ui/assets.go` | Mã nguồn UI |
| 79 | `internal/ui/single_instance_windows.go` | `internal/ui/single_instance_windows.go` | Mã nguồn UI |
| 80 | `internal/ui/single_instance_other.go` | `internal/ui/single_instance_other.go` | Mã nguồn UI |
| 81 | `internal/ui/frameless_windows.go` | `internal/ui/frameless_windows.go` | Mã nguồn UI |
| 82 | `internal/ui/frameless_other.go` | `internal/ui/frameless_other.go` | Mã nguồn UI |
| 83 | `internal/ui/window_resize.go` | `internal/ui/window_resize.go` | Mã nguồn UI |
| 84 | `internal/ui/window_resize_windows.go` | `internal/ui/window_resize_windows.go` | Mã nguồn UI |
| 85 | `internal/ui/window_resize_other.go` | `internal/ui/window_resize_other.go` | Mã nguồn UI |
| 86 | `internal/ui/assets/antigravity_logo.png` | `internal/ui/assets/antigravity_logo.png` | Asset Ảnh |
| 87 | `internal/ui/assets/logo.png` | `internal/ui/assets/logo.png` | Asset Ảnh |
| 88 | `internal/ui/ui_test.go` | `internal/ui/ui_test.go` | Kiểm thử UI |
| 89 | `internal/ui/phase1_sidebar_test.go` | `internal/ui/phase1_sidebar_test.go` | Kiểm thử UI |
| 90 | `internal/ui/themes_test.go` | `internal/ui/themes_test.go` | Kiểm thử UI |
| 91 | `internal/ui/widgets_test.go` | `internal/ui/widgets_test.go` | Kiểm thử UI |
| 92 | `internal/ui/add_modal_test.go` | `internal/ui/add_modal_test.go` | Kiểm thử UI |
| 93 | `internal/ui/add_modal_acceptance_test.go` | `internal/ui/add_modal_acceptance_test.go` | Kiểm thử UI |
| 94 | `internal/ui/footer_bar_acceptance_test.go` | `internal/ui/footer_bar_acceptance_test.go` | Kiểm thử UI |
| 95 | `internal/ui/i18n_acceptance_test.go` | `internal/ui/i18n_acceptance_test.go` | Kiểm thử UI |
| 96 | `internal/ui/liveness_watcher_test.go` | `internal/ui/liveness_watcher_test.go` | Kiểm thử UI |
| 97 | `internal/ui/account_status_acceptance_test.go` | `internal/ui/account_status_acceptance_test.go` | Kiểm thử UI |
| 98 | `internal/ui/account_deletion_acceptance_test.go` | `internal/ui/account_deletion_acceptance_test.go` | Kiểm thử UI |
| 99 | `internal/ui/automation_binding_test.go` | `internal/ui/automation_binding_test.go` | Kiểm thử UI |
| 100 | `internal/ui/two_store_test.go` | `internal/ui/two_store_test.go` | Kiểm thử UI |
| 101 | `internal/ui/two_store_native_test.go` | `internal/ui/two_store_native_test.go` | Kiểm thử UI |
| 102 | `internal/ui/window_native_test.go` | `internal/ui/window_native_test.go` | Kiểm thử UI |
| 103 | `internal/ui/window_resize_test.go` | `internal/ui/window_resize_test.go` | Kiểm thử UI |
| 104 | `internal/ui/window_resize_native_test.go` | `internal/ui/window_resize_native_test.go` | Kiểm thử UI |
| 105 | `internal/ui/app_test.go` | `internal/ui/app_test.go` | Kiểm thử UI |

---

### KHỐI 5: TẦNG BACKEND CODEX MỚI (`internal/engines/codex/`)
*Theo chỉ đạo trực tiếp của Chủ tịch, tại Giai đoạn 1B chỉ khởi tạo duy nhất thư mục này*:
- **Đường dẫn**: `internal/engines/codex/`
- **Quy tắc**: Thư mục rỗng (hoặc file `.gitkeep` để git nhận diện). Tuyệt đối **không tạo bất kỳ thư mục con nào bên trong**. Cấu trúc chi tiết của Codex sẽ được bàn sau tại Giai đoạn 2.

---

# PHẦN 3: LỘ TRÌNH THI CÔNG TUẦN TỰ TỪNG BƯỚC NHỎ (SUB-PHASES CỦA 1B)

Để đảm bảo an toàn tuyệt đối và zero-downtime, Coder bắt buộc phải thực thi tuần tự theo 5 bước con dưới đây:

### 🟢 BƯỚC 1B.1: Di dời Tầng Lõi Dùng Chung (`internal/core/`) — [13 files]
1. Tạo thư mục `internal/core/` cùng các thư mục con: `wincred/`, `version/`, `i18n/`, `preferences/`.
2. Di chuyển các file từ STT 6 đến STT 17 theo đúng bảng Khối 2.
3. Tạo file `internal/core/engine.go` định nghĩa Interface `IDEEngine`.
4. Cập nhật các import path trong các file đã di dời và các caller phụ thuộc.
5. 🎯 **Cổng Kiểm Tra 1B.1**: Chạy `go test ./internal/core/... -v`. Phải đạt **100% PASS** mới được sang Bước 1B.2.

---

### 🟢 BƯỚC 1B.2: Di dời Phân Hệ Backend Độc Lập của Antigravity — [26 files]
1. Tạo thư mục `internal/engines/antigravity/` cùng các thư mục con: `scanner/`, `reloader/`, `telemetry/`, `connector/`.
2. Di chuyển 26 files từ STT 18 đến STT 43 theo đúng bảng Khối 3.A, 3.B, 3.C, 3.D.
3. Cập nhật import paths nội bộ giữa các package này.
4. 🎯 **Cổng Kiểm Tra 1B.2**: Chạy kiểm thử các package vừa di dời:
   ```powershell
   go test ./internal/engines/antigravity/scanner/... -v
   go test ./internal/engines/antigravity/reloader/... -v
   go test ./internal/engines/antigravity/telemetry/... -v
   go test ./internal/engines/antigravity/connector/... -v
   ```
   Phải đạt **100% PASS** mới được sang Bước 1B.3.

---

### 🟢 BƯỚC 1B.3: Di dời Phân Hệ Backend Lõi của Antigravity — [24 files]
1. Tạo các thư mục con còn lại: `model/`, `store/`, `manager/`, `automation/` trong `internal/engines/antigravity/`.
2. Di chuyển 24 files từ STT 44 đến STT 68 theo đúng bảng Khối 3.E, 3.F, 3.G, 3.H.
3. Cập nhật toàn bộ import paths kết nối toàn bộ hệ thống Backend Antigravity.
4. 🎯 **Cổng Kiểm Tra 1B.3**: Chạy toàn bộ kiểm thử backend Antigravity:
   ```powershell
   go test ./internal/engines/antigravity/... -v
   ```
   Phải đạt **100% PASS** mới được sang Bước 1B.4.

---

### 🟢 BƯỚC 1B.4: Đồng bộ Hóa Tầng Giao Diện `internal/ui/` & `cmd/switcher/`
1. Cập nhật các import paths trong `internal/ui/*.go` và `cmd/switcher/main.go` trỏ về đúng đường dẫn mới:
   - Thay `antigravity-switch/internal/connector` ➔ `antigravity-switch/internal/engines/antigravity/connector`
   - Thay `antigravity-switch/internal/manager` ➔ `antigravity-switch/internal/engines/antigravity/manager`
   - Thay `antigravity-switch/internal/telemetry` ➔ `antigravity-switch/internal/engines/antigravity/telemetry`
   - Thay `antigravity-switch/internal/automation` ➔ `antigravity-switch/internal/engines/antigravity/automation`
   - Thay `antigravity-switch/internal/i18n` ➔ `antigravity-switch/internal/core/i18n`
   - Thay `antigravity-switch/internal/wincred` ➔ `antigravity-switch/internal/core/wincred`
   - Thay `antigravity-switch/internal/version` ➔ `antigravity-switch/internal/core/version`
2. 🎯 **Cổng Kiểm Tra 1B.4**: Chạy `go test ./internal/ui/... -v` và `go test ./cmd/switcher/... -v`. Phải đạt **100% PASS**.

---

### 🟢 BƯỚC 1B.5: Khởi tạo Thư mục Gốc `internal/engines/codex/`
1. Tạo thư mục rỗng `internal/engines/codex/` (tạo file `.gitkeep` bên trong để git ghi nhận).
2. Tuyệt đối không tạo thêm bất kỳ file/folder nào khác trong `codex/`.

---

### 🟢 BƯỚC 1B.6: Biên Dịch Tổng Thể Hệ Thống & Kiểm Tra Toàn Diện
1. Chạy script build Windows chuẩn của repository:
   ```powershell
   .\scripts\build_windows.ps1
   ```
2. Chạy toàn bộ test suite của toàn dự án:
   ```powershell
   go test ./... -v
   ```
3. Chạy file nhị phân vừa build để xác nhận giao diện hoạt động 100% bình thường, không có lỗi runtime.

---

# PHẦN 4: NGUYÊN TẮC BẤT BIẾN KIẾN TRÚC CHO GIAI ĐOẠN 1B

1. **Zero-Ellipsis Invariant (Bất biến không viết tắt):** Mọi tài liệu và chỉ dẫn di dời file phải chỉ đích danh từng file, tuyệt đối cấm dùng dấu ba chấm `...` gây hiểu nhầm.
2. **Zero-Dangling Import Invariant (Bất biến import sạch):** Sau khi hoàn thành Giai đoạn 1B, không còn bất kỳ dòng import nào trỏ tới các package cũ đã di dời.
3. **Empty Codex Engine Invariant (Bất biến Codex rỗng):** Thư mục `internal/engines/codex/` chỉ là thư mục chờ, cấu trúc bên trong sẽ do Chủ tịch và System Architect thảo luận tại Giai đoạn 2.
4. **Clean Worktree Invariant:** Sau khi hoàn tất và chạy build thành công, `git status --porcelain` phải trả về 0 dòng.

---

# PHẦN 5: ĐIỀU KIỆN CHUYỂN SANG GIAI ĐOẠN 2

Sau khi Coder hoàn thành toàn bộ 5 bước của Giai đoạn 1B, được QA và Supervisor kiểm duyệt và nghiệm thu **PASS 100%**:
- Chủ tịch sẽ kiểm tra thực tế cấu trúc repository.
- Sau khi Chủ tịch phê duyệt PASS Giai đoạn 1B, Chủ tịch và System Architect sẽ chính thức bắt đầu phiên thảo luận chuyên sâu về **GIAI ĐOẠN 2 (Thiết kế chi tiết nội bộ Backend Codex và Đưa Tab Codex lên Giao diện)**.
