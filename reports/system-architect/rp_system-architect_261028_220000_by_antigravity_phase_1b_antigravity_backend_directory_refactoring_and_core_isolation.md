# BÁO CÁO KIẾN TRÚC VÀ CHỈ DẪN TRIỂN KHAI — GIAI ĐOẠN 1B
## Tách backend Antigravity và core dùng chung, bảo toàn hành vi hiện tại

- **Mã tài liệu:** `rp_system-architect_261028_220000_by_antigravity_phase_1b_antigravity_backend_directory_refactoring_and_core_isolation`
- **Ngày lập / cập nhật:** 2026-10-28.
- **Bản cập nhật:** R2, điều tra source và cập nhật trực tiếp theo yêu cầu Owner.
- **Baseline đã đối chiếu:** commit `f8bed972c35bff013c3d360c856f75712ace5bc2`, repository `G:\antigravity_auto_switch`.
- **Trạng thái:** chỉ dẫn triển khai đã bổ sung; chưa thực hiện refactor, chưa có build/test/runtime PASS của phase 1B. Kết quả review tài liệu ghi ở cuối.
- **Phạm vi:** phase 1B; không tích hợp Codex, không đổi thuật toán switch/quota/retry/continuation, schema dữ liệu hoặc thiết kế UI.
- **Thẩm quyền:** báo cáo gốc ghi nhận hướng phân vùng được Owner chấp thuận; R2 sửa các khoảng trống từ source theo yêu cầu cập nhật của Owner. Không coi báo cáo là bằng chứng implementation đã được nghiệm thu.
- **Cách đọc:** Phần 1 quy định ranh giới; Phần 2–5 là hướng dẫn thi công cụ thể theo yêu cầu Owner. Không tự mở rộng thiết kế đa IDE từ hướng dẫn này.

---

# PHẦN 1: KẾT QUẢ ĐIỀU TRA VÀ QUYẾT ĐỊNH

## 1.1. Các sai lệch đã xác nhận

| Phát hiện | Bằng chứng baseline | Xử lý R2 |
|---|---|---|
| Wincred chưa độc lập IDE | `internal/wincred/cred.go` import model; `types.go` chứa target/user Antigravity | Tách raw Windows API vào core, token adapter vào engine ở 1B.1 |
| Build script bị bỏ sót | `scripts/build_windows.ps1` đọc version path cũ và hai linker symbol cũ | Sửa cả ba ngay khi move version |
| Caller bị trì hoãn tới 1B.4 | Manager dùng connector/reloader/scanner/telemetry; telemetry test dùng store | Move và cập nhật mọi caller trong cùng bước |
| Preferences cần tách symbol | `internal/ui/themes.go` dùng preferences qua store; `app_test.go` dùng account store | Đổi package preferences và qualifier đúng caller; không replace store toàn cục |
| IDEEngine chưa có contract | Chỉ có tên interface, không có method/type/lifecycle | Không tạo engine.go trong 1B; thiết kế ở phase 2 |
| Native tests bị bỏ qua | Ba file UI dùng tag `windows && gui && nativeui` | Native gate riêng trên Windows desktop tương tác |
| Kiểm kê sai | 104 file cũ: 103 tracked và 1 syso generated; backend tám package có 51 file | Sửa map và số lượng, phân biệt runtime data |
| Đếm bước sai | Có 1B.1–1B.6, không phải năm bước | Sáu bước và preflight 1B.0 |

Ba file ignored tại `internal/ui/accounts_store/.ui_preferences.json`, `internal/ui/logs/.ui_preferences.json`, `internal/ui/logs/ui_preferences.json` là dữ liệu runtime có mặt lúc điều tra, không phải source cần move. Không đọc nội dung hoặc xóa chúng để làm sạch worktree.

## 1.2. Ranh giới và hướng phụ thuộc

- Core gồm `wincred` raw, `preferences`, `i18n`, `version`; không chứa Google token model, default credential Antigravity hoặc lifecycle IDE.
- Antigravity gồm chín package: `scanner`, `reloader`, `telemetry`, `connector`, `model`, `store`, `manager`, `automation`, `wincred` token adapter.
- Cho phép: UI/entrypoint → engine và core; engine → core. Core không được phụ thuộc engine/UI/entrypoint/domain cũ trực tiếp hoặc gián tiếp, kể cả test dependency.
- UI và entrypoint vẫn gọi Antigravity concrete implementation; phase này không thêm registry/router/façade/interface rỗng.
- Không đưa Google model vào core để né việc tách adapter.
- `internal/engines/codex/` chỉ có `.gitkeep` rỗng, không thư mục con hoặc package Go.
- **IDEEngine:** hoãn toàn bộ contract và lifecycle đa IDE sang phase 2. Không tạo `internal/core/engine.go` ở phase 1B. Việc hoãn không làm thay đổi hành vi đang có vì baseline chưa có interface/consumer này.

## 1.3. Hợp đồng tách credential

| File đích | Nội dung bắt buộc |
|---|---|
| `internal/core/wincred/cred.go` | Tách raw API từ file cũ: binding advapi32, ReadRaw, WriteRaw, DeleteRaw, HasRaw và hai sentinel ErrNotFound/ErrInvalidPayload |
| `internal/core/wincred/types.go` | CREDENTIALW, CredTypeGeneric, CredPersistLocalMachine; bỏ hai hằng target/user Antigravity |
| `internal/core/wincred/cred_test.go` | Test mới raw byte API, không dùng model hoặc credential thật |
| `internal/engines/antigravity/wincred/cred.go` | Phần token còn lại từ cred.go cũ; nhận hai hằng target/user từ types.go; gọi raw core qua alias corewincred |
| `internal/engines/antigravity/wincred/cred_test.go` | Di chuyển ba test token hiện hữu; bảo toàn assertions, đổi import model; bổ sung kiểm chứng lỗi nếu còn thiếu |

Adapter giữ nguyên chữ ký và semantics của tám hàm: `ReadToken`, `ReadTokenTarget`, `WriteToken`, `WriteTokenTarget`, `DeleteToken`, `DeleteTokenTarget`, `HasToken`, `HasTokenTarget`.

- Giữ `DefaultTargetName = "LegacyGeneric:target=gemini:antigravity"`, `DefaultUserName = "antigravity"` trong adapter.
- Giữ identity lỗi bằng alias: `ErrNotFound = corewincred.ErrNotFound`, `ErrInvalidPayload = corewincred.ErrInvalidPayload`. Không tạo errors.New mới cùng message.
- Raw core giữ validation target/blob, native error propagation, copy blob, CredFree, persist mode, delete idempotent và hành vi empty target như baseline.
- Adapter giữ JSON parse/serialize, TokenData.IsValid, nil-token và empty-blob behavior. Không copy Win32 syscall trở lại engine; không cần tái xuất raw API từ adapter.
- Trong 1B.1 adapter tạm import `antigravity-switch/internal/model`; tới 1B.3 đổi sang `antigravity-switch/internal/engines/antigravity/model`. Core không bao giờ dùng model này.
- Toàn bộ caller token cũ chuyển import sang adapter engine, không chuyển sang raw core.
- Test raw dùng target thử nghiệm riêng, cleanup đúng target test; không ghi/xóa default credential. Kiểm byte roundtrip, absent/ErrNotFound, repeated delete, empty target/blob.
- Test token giữ roundtrip/validation/live-read; thêm assertion identity của hai sentinel và parse invalid JSON/empty blob nếu coverage chưa có. Không sửa expected behavior để ép PASS.
- Test live-read hiện có chứa nhánh lỗi in token: khi move phải thay message đó bằng thông báo không chứa payload, giữ nguyên điều kiện assertion. Không đưa token/credential vào evidence.

## 1.4. Hợp đồng preferences, build và dữ liệu

- Hai file preferences chuyển thành `package preferences`, giữ public API/schema/default/path/atomic-save behavior.
- `internal/ui/themes.go` import `antigravity-switch/internal/core/preferences`; thay qualifier của DefaultUIPreferencesPath, UIPreferences, LoadUIPreferences, SaveUIPreferences từ store sang preferences.
- Giữ `logs/ui_preferences.json`, mọi field JSON và mặc định hiện hữu. Các account/discovered store vẫn là `package store`; không replace store toàn cục.
- Build script đổi SourcePath thành `internal\core\version\version.go`; đổi cả hai linker symbol thành `antigravity-switch/internal/core/version.Version` và `antigravity-switch/internal/core/version.GitCommit`.
- Giữ module, Go/dependency version, version ứng dụng, icon, app ID, executable name, single-instance title và working directory của bản phát hành.
- I18n JSON di chuyển cùng package; giữ `go:embed locales/vi.json` và `go:embed locales/en.json` cùng nguyên nội dung.
- Không di dời account stores, logs, profile IDE, SQLite session hoặc credential của người dùng. Đây là refactor source, không phải data migration.

---

# PHẦN 2: BẢN ĐỒ ĐẦY ĐỦ FILE CŨ → MỚI

## 2.1. Tổng kiểm kê và quy ước

- Baseline: 4 entrypoint/resource (gồm syso generated), 12 file wincred/version/i18n/preferences, 51 file backend thuộc tám package, 37 UI = **104 đường dẫn cũ**.
- Đích: 4 entrypoint/resource, 12 core, 53 Antigravity, 37 UI và 1 Codex gitkeep = **107 đường dẫn**, gồm 106 tracked sau khi add và 1 syso generated.
- Ngoài tổng trên: build script và báo cáo này; evidence/ledger đếm riêng.
- Bảng giữ đủ từng file. Riêng cred.go là tách nội dung, không phải move 1–1 thuần túy; Phần 1.3 và bảng 2.2 mô tả đầy đủ output bổ sung.
- Bước move: nhóm core/wincred/preferences tại 1B.1; scanner/reloader/telemetry/connector tại 1B.2; model/store còn lại/manager/automation tại 1B.3; UI và entrypoint giữ vị trí, cập nhật import tại bước phát sinh.

| STT | Đường dẫn cũ | Đường dẫn đích |
|---:|---|---|
| 1 | `cmd/switcher/main.go` | `cmd/switcher/main.go` |
| 2 | `cmd/switcher/main_test.go` | `cmd/switcher/main_test.go` |
| 3 | `cmd/switcher/app.rc` | `cmd/switcher/app.rc` |
| 4 | `cmd/switcher/rsrc_windows_amd64.syso` | `cmd/switcher/rsrc_windows_amd64.syso` |
| 5 | `internal/wincred/cred.go` | `internal/engines/antigravity/wincred/cred.go` |
| 6 | `internal/wincred/cred_test.go` | `internal/engines/antigravity/wincred/cred_test.go` |
| 7 | `internal/wincred/types.go` | `internal/core/wincred/types.go` |
| 8 | `internal/version/version.go` | `internal/core/version/version.go` |
| 9 | `internal/version/version_test.go` | `internal/core/version/version_test.go` |
| 10 | `internal/i18n/dictionary.go` | `internal/core/i18n/dictionary.go` |
| 11 | `internal/i18n/locale.go` | `internal/core/i18n/locale.go` |
| 12 | `internal/i18n/i18n_test.go` | `internal/core/i18n/i18n_test.go` |
| 13 | `internal/i18n/locales/en.json` | `internal/core/i18n/locales/en.json` |
| 14 | `internal/i18n/locales/vi.json` | `internal/core/i18n/locales/vi.json` |
| 15 | `internal/store/preferences_store.go` | `internal/core/preferences/preferences_store.go` |
| 16 | `internal/store/preferences_store_test.go` | `internal/core/preferences/preferences_store_test.go` |
| 17 | `internal/scanner/chrome_scanner.go` | `internal/engines/antigravity/scanner/chrome_scanner.go` |
| 18 | `internal/scanner/chrome_scanner_test.go` | `internal/engines/antigravity/scanner/chrome_scanner_test.go` |
| 19 | `internal/reloader/active_threads.go` | `internal/engines/antigravity/reloader/active_threads.go` |
| 20 | `internal/reloader/active_threads_test.go` | `internal/engines/antigravity/reloader/active_threads_test.go` |
| 21 | `internal/reloader/continuation.go` | `internal/engines/antigravity/reloader/continuation.go` |
| 22 | `internal/reloader/continuation_test.go` | `internal/engines/antigravity/reloader/continuation_test.go` |
| 23 | `internal/reloader/session_reloader.go` | `internal/engines/antigravity/reloader/session_reloader.go` |
| 24 | `internal/reloader/reloader_test.go` | `internal/engines/antigravity/reloader/reloader_test.go` |
| 25 | `internal/telemetry/client.go` | `internal/engines/antigravity/telemetry/client.go` |
| 26 | `internal/telemetry/cache.go` | `internal/engines/antigravity/telemetry/cache.go` |
| 27 | `internal/telemetry/cache_test.go` | `internal/engines/antigravity/telemetry/cache_test.go` |
| 28 | `internal/telemetry/loop.go` | `internal/engines/antigravity/telemetry/loop.go` |
| 29 | `internal/telemetry/pool.go` | `internal/engines/antigravity/telemetry/pool.go` |
| 30 | `internal/telemetry/tier.go` | `internal/engines/antigravity/telemetry/tier.go` |
| 31 | `internal/telemetry/quota_test.go` | `internal/engines/antigravity/telemetry/quota_test.go` |
| 32 | `internal/telemetry/telemetry_test.go` | `internal/engines/antigravity/telemetry/telemetry_test.go` |
| 33 | `internal/connector/discovery.go` | `internal/engines/antigravity/connector/discovery.go` |
| 34 | `internal/connector/discovery_test.go` | `internal/engines/antigravity/connector/discovery_test.go` |
| 35 | `internal/connector/endpoints.go` | `internal/engines/antigravity/connector/endpoints.go` |
| 36 | `internal/connector/agent_state_stream.go` | `internal/engines/antigravity/connector/agent_state_stream.go` |
| 37 | `internal/connector/api_client.go` | `internal/engines/antigravity/connector/api_client.go` |
| 38 | `internal/connector/automation_rpc.go` | `internal/engines/antigravity/connector/automation_rpc.go` |
| 39 | `internal/connector/automation_rpc_test.go` | `internal/engines/antigravity/connector/automation_rpc_test.go` |
| 40 | `internal/connector/oauth_refresher.go` | `internal/engines/antigravity/connector/oauth_refresher.go` |
| 41 | `internal/connector/oauth_refresher_test.go` | `internal/engines/antigravity/connector/oauth_refresher_test.go` |
| 42 | `internal/connector/connector_test.go` | `internal/engines/antigravity/connector/connector_test.go` |
| 43 | `internal/model/types.go` | `internal/engines/antigravity/model/types.go` |
| 44 | `internal/model/types_test.go` | `internal/engines/antigravity/model/types_test.go` |
| 45 | `internal/store/account_store.go` | `internal/engines/antigravity/store/account_store.go` |
| 46 | `internal/store/account_store_test.go` | `internal/engines/antigravity/store/account_store_test.go` |
| 47 | `internal/store/discovered_store.go` | `internal/engines/antigravity/store/discovered_store.go` |
| 48 | `internal/store/discovered_store_test.go` | `internal/engines/antigravity/store/discovered_store_test.go` |
| 49 | `internal/store/sanitizer.go` | `internal/engines/antigravity/store/sanitizer.go` |
| 50 | `internal/manager/account_manager.go` | `internal/engines/antigravity/manager/account_manager.go` |
| 51 | `internal/manager/account_manager_test.go` | `internal/engines/antigravity/manager/account_manager_test.go` |
| 52 | `internal/manager/manager_test.go` | `internal/engines/antigravity/manager/manager_test.go` |
| 53 | `internal/manager/timeline.go` | `internal/engines/antigravity/manager/timeline.go` |
| 54 | `internal/manager/timeline_test.go` | `internal/engines/antigravity/manager/timeline_test.go` |
| 55 | `internal/manager/two_store_test.go` | `internal/engines/antigravity/manager/two_store_test.go` |
| 56 | `internal/automation/controller.go` | `internal/engines/antigravity/automation/controller.go` |
| 57 | `internal/automation/controller_test.go` | `internal/engines/antigravity/automation/controller_test.go` |
| 58 | `internal/automation/model_quota.go` | `internal/engines/antigravity/automation/model_quota.go` |
| 59 | `internal/automation/model_quota_test.go` | `internal/engines/antigravity/automation/model_quota_test.go` |
| 60 | `internal/automation/retry.go` | `internal/engines/antigravity/automation/retry.go` |
| 61 | `internal/automation/retry_test.go` | `internal/engines/antigravity/automation/retry_test.go` |
| 62 | `internal/automation/sorter.go` | `internal/engines/antigravity/automation/sorter.go` |
| 63 | `internal/automation/sorter_test.go` | `internal/engines/antigravity/automation/sorter_test.go` |
| 64 | `internal/automation/switch_gate_windows.go` | `internal/engines/antigravity/automation/switch_gate_windows.go` |
| 65 | `internal/automation/switch_gate_windows_test.go` | `internal/engines/antigravity/automation/switch_gate_windows_test.go` |
| 66 | `internal/automation/automation_e2e_test.go` | `internal/engines/antigravity/automation/automation_e2e_test.go` |
| 67 | `internal/automation/runtime_regression_test.go` | `internal/engines/antigravity/automation/runtime_regression_test.go` |
| 68 | `internal/ui/window.go` | `internal/ui/window.go` |
| 69 | `internal/ui/sidebar.go` | `internal/ui/sidebar.go` |
| 70 | `internal/ui/app.go` | `internal/ui/app.go` |
| 71 | `internal/ui/widgets.go` | `internal/ui/widgets.go` |
| 72 | `internal/ui/themes.go` | `internal/ui/themes.go` |
| 73 | `internal/ui/add_modal.go` | `internal/ui/add_modal.go` |
| 74 | `internal/ui/quota_badge.go` | `internal/ui/quota_badge.go` |
| 75 | `internal/ui/compact_button.go` | `internal/ui/compact_button.go` |
| 76 | `internal/ui/tray.go` | `internal/ui/tray.go` |
| 77 | `internal/ui/assets.go` | `internal/ui/assets.go` |
| 78 | `internal/ui/single_instance_windows.go` | `internal/ui/single_instance_windows.go` |
| 79 | `internal/ui/single_instance_other.go` | `internal/ui/single_instance_other.go` |
| 80 | `internal/ui/frameless_windows.go` | `internal/ui/frameless_windows.go` |
| 81 | `internal/ui/frameless_other.go` | `internal/ui/frameless_other.go` |
| 82 | `internal/ui/window_resize.go` | `internal/ui/window_resize.go` |
| 83 | `internal/ui/window_resize_windows.go` | `internal/ui/window_resize_windows.go` |
| 84 | `internal/ui/window_resize_other.go` | `internal/ui/window_resize_other.go` |
| 85 | `internal/ui/assets/antigravity_logo.png` | `internal/ui/assets/antigravity_logo.png` |
| 86 | `internal/ui/assets/logo.png` | `internal/ui/assets/logo.png` |
| 87 | `internal/ui/ui_test.go` | `internal/ui/ui_test.go` |
| 88 | `internal/ui/phase1_sidebar_test.go` | `internal/ui/phase1_sidebar_test.go` |
| 89 | `internal/ui/themes_test.go` | `internal/ui/themes_test.go` |
| 90 | `internal/ui/widgets_test.go` | `internal/ui/widgets_test.go` |
| 91 | `internal/ui/add_modal_test.go` | `internal/ui/add_modal_test.go` |
| 92 | `internal/ui/add_modal_acceptance_test.go` | `internal/ui/add_modal_acceptance_test.go` |
| 93 | `internal/ui/footer_bar_acceptance_test.go` | `internal/ui/footer_bar_acceptance_test.go` |
| 94 | `internal/ui/i18n_acceptance_test.go` | `internal/ui/i18n_acceptance_test.go` |
| 95 | `internal/ui/liveness_watcher_test.go` | `internal/ui/liveness_watcher_test.go` |
| 96 | `internal/ui/account_status_acceptance_test.go` | `internal/ui/account_status_acceptance_test.go` |
| 97 | `internal/ui/account_deletion_acceptance_test.go` | `internal/ui/account_deletion_acceptance_test.go` |
| 98 | `internal/ui/automation_binding_test.go` | `internal/ui/automation_binding_test.go` |
| 99 | `internal/ui/two_store_test.go` | `internal/ui/two_store_test.go` |
| 100 | `internal/ui/two_store_native_test.go` | `internal/ui/two_store_native_test.go` |
| 101 | `internal/ui/window_native_test.go` | `internal/ui/window_native_test.go` |
| 102 | `internal/ui/window_resize_test.go` | `internal/ui/window_resize_test.go` |
| 103 | `internal/ui/window_resize_native_test.go` | `internal/ui/window_resize_native_test.go` |
| 104 | `internal/ui/app_test.go` | `internal/ui/app_test.go` |

## 2.2. File bổ sung và file ngoài cây source

| Loại | Đường dẫn chính xác | Bước / hành động |
|---|---|---|
| Tách mới từ cred.go cũ | `internal/core/wincred/cred.go` | 1B.1 raw API |
| Test mới | `internal/core/wincred/cred_test.go` | 1B.1 kiểm raw contract |
| File mới 0 byte | `internal/engines/codex/.gitkeep` | 1B.5 duy nhất file này |
| Giữ vị trí, sửa | `scripts/build_windows.ps1` | 1B.1 version path và hai linker symbol |
| Giữ vị trí, sửa | `reports/system-architect/rp_system-architect_261028_220000_by_antigravity_phase_1b_antigravity_backend_directory_refactoring_and_core_isolation.md` | R2 theo yêu cầu Owner |

Không tạo `internal/core/engine.go`, không giữ compatibility package tại đường dẫn cũ. Alias sentinel trong token adapter chỉ để bảo toàn identity lỗi.

---

# PHẦN 3: CALLER TOUCH-MAP THEO BƯỚC

Paths dưới đây là vị trí baseline; nếu file đã move, tra bảng 2.1 để lấy vị trí đang có. Bảng lấy từ import source thực; trước thi công phải quét lại source drift. Mỗi bước phải cập nhật cả production và test caller, không đợi 1B.4.

## 1B.1 — caller phải cập nhật

| File tại baseline | Import/symbol bị ảnh hưởng |
|---|---|
| `internal/manager/account_manager_test.go` | wincred |
| `internal/manager/account_manager.go` | wincred |
| `internal/manager/two_store_test.go` | wincred |
| `internal/ui/account_deletion_acceptance_test.go` | wincred |
| `internal/ui/add_modal_acceptance_test.go` | wincred |
| `internal/ui/add_modal_test.go` | wincred |
| `internal/ui/add_modal.go` | i18n |
| `internal/ui/app.go` | i18n |
| `internal/ui/i18n_acceptance_test.go` | i18n |
| `internal/ui/liveness_watcher_test.go` | i18n |
| `internal/ui/phase1_sidebar_test.go` | i18n, version |
| `internal/ui/quota_badge.go` | i18n |
| `internal/ui/sidebar.go` | i18n |
| `internal/ui/themes.go` | preferences: bốn qualifier tại Phần 1.4 |
| `internal/ui/tray.go` | i18n |
| `internal/ui/two_store_native_test.go` | wincred |
| `internal/ui/two_store_test.go` | wincred |
| `internal/ui/widgets.go` | i18n |
| `internal/ui/window.go` | i18n, version |

## 1B.2 — caller phải cập nhật

| File tại baseline | Import/symbol bị ảnh hưởng |
|---|---|
| `cmd/switcher/main_test.go` | connector, reloader |
| `cmd/switcher/main.go` | reloader |
| `internal/automation/automation_e2e_test.go` | connector, reloader, telemetry |
| `internal/automation/controller_test.go` | connector, reloader, telemetry |
| `internal/automation/controller.go` | connector, telemetry |
| `internal/automation/model_quota_test.go` | telemetry |
| `internal/automation/model_quota.go` | telemetry |
| `internal/automation/retry_test.go` | connector |
| `internal/automation/retry.go` | connector, reloader |
| `internal/automation/runtime_regression_test.go` | connector, telemetry |
| `internal/automation/sorter_test.go` | telemetry |
| `internal/automation/sorter.go` | telemetry |
| `internal/manager/account_manager_test.go` | connector, reloader |
| `internal/manager/account_manager.go` | connector, reloader, scanner, telemetry |
| `internal/manager/manager_test.go` | connector, reloader |
| `internal/manager/timeline_test.go` | connector, reloader |
| `internal/manager/two_store_test.go` | connector |
| `internal/reloader/continuation_test.go` | connector |
| `internal/reloader/continuation.go` | connector |
| `internal/reloader/reloader_test.go` | connector |
| `internal/reloader/session_reloader.go` | connector |
| `internal/telemetry/client.go` | connector |
| `internal/telemetry/pool.go` | connector |
| `internal/telemetry/quota_test.go` | connector |
| `internal/telemetry/telemetry_test.go` | connector |
| `internal/ui/account_deletion_acceptance_test.go` | connector, reloader, telemetry |
| `internal/ui/account_status_acceptance_test.go` | connector, reloader, telemetry |
| `internal/ui/add_modal_acceptance_test.go` | connector, reloader, telemetry |
| `internal/ui/add_modal_test.go` | connector |
| `internal/ui/app.go` | connector, telemetry |
| `internal/ui/automation_binding_test.go` | connector, telemetry |
| `internal/ui/footer_bar_acceptance_test.go` | connector |
| `internal/ui/liveness_watcher_test.go` | connector |
| `internal/ui/quota_badge.go` | telemetry |
| `internal/ui/themes_test.go` | telemetry |
| `internal/ui/two_store_native_test.go` | connector |
| `internal/ui/two_store_test.go` | connector |
| `internal/ui/ui_test.go` | connector, reloader, telemetry |
| `internal/ui/widgets.go` | telemetry |

## 1B.3 — caller phải cập nhật

| File tại baseline | Import/symbol bị ảnh hưởng |
|---|---|
| `cmd/switcher/main_test.go` | automation, manager, model, store |
| `cmd/switcher/main.go` | manager, store |
| `internal/automation/automation_e2e_test.go` | manager, model, store |
| `internal/automation/controller_test.go` | manager, model, store |
| `internal/automation/controller.go` | manager |
| `internal/automation/switch_gate_windows.go` | manager |
| `internal/manager/account_manager_test.go` | model, store |
| `internal/manager/account_manager.go` | model, store |
| `internal/manager/manager_test.go` | model, store |
| `internal/manager/timeline_test.go` | model, store |
| `internal/manager/two_store_test.go` | model, store |
| `internal/store/account_store_test.go` | model |
| `internal/store/account_store.go` | model |
| `internal/telemetry/pool.go` | store |
| `internal/telemetry/telemetry_test.go` | model, store |
| `internal/ui/account_deletion_acceptance_test.go` | manager, model, store |
| `internal/ui/account_status_acceptance_test.go` | automation, manager, model, store |
| `internal/ui/add_modal_acceptance_test.go` | automation, manager, model, store |
| `internal/ui/add_modal_test.go` | manager, model, store |
| `internal/ui/add_modal.go` | manager, model |
| `internal/ui/app_test.go` | automation, model, store |
| `internal/ui/app.go` | automation, manager |
| `internal/ui/automation_binding_test.go` | automation, manager |
| `internal/ui/footer_bar_acceptance_test.go` | manager |
| `internal/ui/i18n_acceptance_test.go` | manager, model |
| `internal/ui/liveness_watcher_test.go` | manager |
| `internal/ui/phase1_sidebar_test.go` | manager |
| `internal/ui/themes_test.go` | manager |
| `internal/ui/two_store_native_test.go` | manager, model, store |
| `internal/ui/two_store_test.go` | manager, model, store |
| `internal/ui/ui_test.go` | automation, manager, model, store |
| `internal/ui/widgets_test.go` | manager |
| `internal/ui/widgets.go` | manager |
| `internal/ui/window_native_test.go` | manager |
| `internal/wincred/cred_test.go` | model |
| `internal/wincred/cred.go` | model |


Lưu ý bảng 1B.3: các dòng wincred được sửa tại vị trí token adapter mới; raw core không được có import model. Hai file preferences chỉ đổi package declaration ở 1B.1 và không được chuyển lại vào engine store.

---

# PHẦN 4: LỘ TRÌNH VÀ CỔNG KIỂM TRA

## 4.1. Quy tắc chung và cổng G

1. Chạy lệnh từ root `G:\antigravity_auto_switch` bằng PowerShell. Sau mỗi native command kiểm tra `$LASTEXITCODE`; khác 0 là FAIL, trừ quy ước không-match của rg ở 5.1. Không chạy tiếp để che exit code.
2. Mỗi slice gồm move/tách source, sửa mọi caller, gofmt file Go đổi, kiểm tra, QA/Supervisor và commit riêng. Không commit trạng thái import gãy.
3. Không giảm assertions, skip test hoặc thay logic nghiệp vụ để làm gate xanh. Baseline fail phải được ghi và xử lý đúng nguyên nhân trước khi mở slice.
4. Không dùng reset/clean/xóa dữ liệu hoặc stage thay đổi của người khác để đạt worktree sạch.
5. Theo working-rules áp dụng cho công việc nhiều module này, planner phải tạo/cập nhật ledger `docs/plans` trước coding và chỉ mở slice hiện tại. Báo cáo này là đầu vào kiến trúc/thi công; ledger phải bám hợp đồng này, không tự thiết kế thêm scope.
6. Dấu `...` trong pattern Go là cú pháp đệ quy, không phải viết tắt danh sách file. Touch-map phải ghi đích danh từng file.
7. Cổng G dưới đây áp dụng sau từng bước 1B.1–1B.5; 1B.6 kiểm tra trạng thái cuối. Không chỉ test package vừa move.

```powershell
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '1'
go list ./...
if ($LASTEXITCODE -ne 0) { throw "Gate failed; stop this slice" }
go test -count=1 ./...
if ($LASTEXITCODE -ne 0) { throw "Gate failed; stop this slice" }
.\scripts\build_windows.ps1
if ($LASTEXITCODE -ne 0) { throw "Gate failed; stop this slice" }
git diff --check
if ($LASTEXITCODE -ne 0) { throw "Diff validation failed" }
```

Script build có vet/test; lệnh `-count=1` bảo đảm một lượt test không lấy cache. Sau build chạy native gate ở 5.2 và smoke binary ở 5.3. Không chạy build đồng thời native suite/app smoke vì build script có thể dừng executable đang chạy.

## 4.2. Bước 1B.0 — baseline

- Ghi HEAD, branch, `git status --porcelain`, `go version`, `go env GOOS GOARCH CGO_ENABLED CC`, compiler/resource tool và trạng thái app.
- Đối chiếu 104 paths bảng 2.1; syso có thể chưa có ở fresh clone và phải được sinh bởi build, không được yêu cầu tracked. Xác nhận ba output mới chưa bị chiếm dụng.
- Nếu source drift so với baseline R2, liệt kê file khác và cập nhật map trước move, không bỏ sót source mới.
- Chạy G, native suite và runtime smoke baseline; lưu exit code/test JSON/ảnh. Nếu gate bắt buộc fail hoặc môi trường không đáp ứng, chưa mở 1B.1.
- Đóng Auto Switcher bình thường trước build/native; không kill IDE Antigravity. Không chạy các test có side effect trên account/credential thật.
- Giữ nguyên thay đổi có sẵn của người khác, ghi vào baseline; nếu không thể cô lập thì dùng checkout riêng theo quy trình dự án. Không tự stage/commit chúng.

## 4.3. Bước 1B.1 — core và token adapter

**Move 12 file baseline, tạo thêm hai file raw core, cập nhật caller/build trong cùng bước.**

- Move wincred/version/i18n/preferences theo bảng 2.1; tách raw/token theo 1.3.
- Package core và adapter đều tên wincred; adapter import core bằng alias corewincred.
- Đổi package preferences và themes caller theo 1.4; cập nhật toàn bộ caller bảng 1B.1.
- Sửa version source path và hai linker flags ngay khi move version. Không đợi bước build cuối.
- Không tạo engine.go. Không giữ package bridge ở path cũ.

```powershell
go test -count=1 ./internal/core/...
go test -count=1 ./internal/engines/antigravity/wincred
```

Sau đó G. Acceptance riêng: raw contract và token tests PASS, sentinel identity giữ nguyên, preferences/i18n/version vẫn hoạt động; core không import domain cũ/engine. Ghi SKIP của live read riêng, không coi là tích hợp thật đã PASS.

## 4.4. Bước 1B.2 — scanner/reloader/telemetry/connector

**Move 26 file:** scanner 2, reloader 6, telemetry 8, connector 10.

- Move theo bảng 2.1 và cập nhật toàn bộ bảng caller 1B.2, gồm manager/automation/UI/entrypoint dù chúng chưa move.
- Nhóm này không độc lập hoàn toàn: telemetry còn dùng store, caller còn dùng package cũ khác. Các import model/store/manager/automation chưa move vẫn hợp lệ cho tới 1B.3.

```powershell
go test -count=1 ./internal/engines/antigravity/scanner ./internal/engines/antigravity/reloader ./internal/engines/antigravity/telemetry ./internal/engines/antigravity/connector
```

Sau đó G. Acceptance: toàn bộ graph compile/test, không còn import bốn path đã move; discovery/RPC/OAuth/quota/continuation không thay hành vi.

## 4.5. Bước 1B.3 — model/store/manager/automation

**Move 25 file:** model 2, store 5, manager 6, automation 12. Preferences đã tách, không move lại.

- Move theo bảng 2.1 và sửa toàn bộ bảng caller 1B.3, gồm model import trong token adapter mới.
- Chỉ xóa thư mục source cũ khi đã rỗng sau move và đã xác minh absolute path nằm trong repo; không xóa thư mục runtime trùng tên.

```powershell
go test -count=1 ./internal/engines/antigravity/...
```

Sau đó G. Acceptance: account/discovered stores, manager/timeline, switch/automation/retry/continuation giữ behavior; core không phụ thuộc engine trực tiếp hoặc gián tiếp.

## 4.6. Bước 1B.4 — audit UI và entrypoint

**Không move thêm file.** Caller đã sửa ngay ở bước phát sinh; đây là audit tích hợp, không phải lần đầu sửa import UI.

- Rà đủ 37 file UI và 4 entrypoint/resource, gồm `cmd/switcher/main_test.go` và ba test native.
- Kiểm alias preferences/token adapter, import cũ, startup wiring và asset/embed.

```powershell
go test -count=1 ./internal/ui ./cmd/switcher
```

Sau đó G. Giữ sidebar Antigravity, startup size/layout, label/icon, theme/locale, account actions, tray/single-instance. Không tạo tab Codex. Nếu audit không tạo diff source thì chỉ ghi evidence/ledger, không tạo sửa đổi giả hoặc empty commit.

## 4.7. Bước 1B.5 — placeholder Codex

- Tạo đúng `internal/engines/codex/.gitkeep` dài 0 byte.
- Chạy G và `Get-ChildItem -Force -Recurse internal/engines/codex`.
- Acceptance: chỉ một file gitkeep, không có thư mục con, source Go hoặc import Codex. Test count không giảm do placeholder.

## 4.8. Bước 1B.6 — nghiệm thu cuối

- Đối chiếu toàn bộ map, ba output mới, tổng file; không có engine.go và source ở đường dẫn cũ.
- Chạy G, dependency/resource checks, native và runtime smoke đầy đủ.
- QA ghi evidence từng invariant; Supervisor kiểm tra độc lập diff/source/log/ảnh/runtime. Không ghi PASS trước evidence.
- Commit từng slice đã được nghiệm thu; không gom toàn đợt chưa kiểm vào một commit.
- Sau commit chạy `git status --porcelain`: baseline sạch thì output phải rỗng. Baseline có thay đổi của người khác thì chỉ yêu cầu không còn thay đổi chưa commit thuộc phase 1B, phần còn lại phải khớp baseline.
- Không xóa ignored runtime data để đạt trạng thái sạch. Nếu smoke làm thay đổi tracked file, điều tra owner/path và xử lý đúng phạm vi.
- Gate fail thì dừng bước tiếp theo, sửa invariant bị fail và chạy lại kiểm tra liên quan.

---

# PHẦN 5: VERIFICATION VÀ EVIDENCE

## 5.1. Import, dependency và build resource

Quét source/build scripts, không quét báo cáo lịch sử làm gate vì chúng cố ý ghi path cũ.

```powershell
rg -n 'antigravity-switch/internal/(wincred|version|i18n|scanner|reloader|telemetry|connector|model|store|manager|automation)(/|")' cmd internal scripts -g '*.go' -g '*.ps1'
rg -n 'internal[\\/]+version|antigravity-switch/internal/version' scripts -g '*.ps1'
go list -deps -test ./internal/core/...
go version -m .\bin\antigravity-switch.exe
```

- Gate cuối không có match path cũ. `rg` exit 1 là không có match; lớn hơn 1 là lỗi scan. Tại bước trung gian chỉ cấm path đã move.
- Core dependency list không chứa engine/UI/cmd/domain cũ; kiểm cả test dependency, không chỉ import sản phẩm.
- Default go list không bao phủ tagged native files, cần gate 5.2.
- I18n JSON, UI assets và resource không thay nội dung ngoài version sync đã dự kiến; version ứng dụng không được tự tăng.
- Binary metadata phải chứa hai linker symbol mới và giá trị version/commit đúng lúc build. So với HEAD lúc build, không so HEAD của commit evidence tạo sau đó.
- Diff được phép: move, import/qualifier, package preferences, raw/token split, sentinel alias, tests cho boundary mới, bỏ token khỏi message lỗi test, build version path/symbol, gitkeep. Thay đổi thuật toán ngoài đây phải được xử lý riêng.

## 5.2. Native GUI gate

Ba file cần được chọn:

- `internal/ui/window_native_test.go`
- `internal/ui/window_resize_native_test.go`
- `internal/ui/two_store_native_test.go`

Dùng Windows desktop tương tác, CGO/compiler GUI. Đóng Auto Switcher trước test: TestMain từ chối chạy nếu cửa sổ thật đang tồn tại. Không đóng IDE Antigravity và không chạy test native đồng thời release smoke.

```powershell
go list -tags 'gui nativeui' -f '{{.TestGoFiles}}' ./internal/ui
go test -tags 'gui nativeui' -count=1 -run '^TestNative' -v ./internal/ui
```

Bốn test phải thực sự được chạy và PASS:

- `TestNativeWindowMaximizeRestoreAndGrid`
- `TestNativeWindowGeometryAndPointer`
- `TestNativeWindowManualResize`
- `TestNativeTwoStoreRefreshAndResize`

Không trộn toàn bộ test UI thường vào GUI harness khi chưa chứng minh hỗ trợ; test thường đã chạy riêng trong G.

Harness hỗ trợ `NATIVE_WINDOW_CAPTURE`, `NATIVE_MAXIMIZE_CAPTURE`, `TWO_STORE_CAPTURE`. Đặt biến này thành absolute path trong evidence của slice, tạo parent directory trước test. Mở ảnh để xác nhận native geometry/canvas, hit targets, resize/maximize/restore và ba trạng thái account/warning. Không dùng ảnh cũ chứng minh binary mới.

Không có desktop hoặc không chạy được native suite → ghi BLOCKED native gate; không thay bằng unit PASS.

## 5.3. Regression và release smoke

| Boundary | Kiểm chứng | Hành vi phải giữ |
|---|---|---|
| Raw credential | Test mới target riêng | Byte roundtrip, absent, validation, delete lặp |
| Token adapter | Test cũ và boundary assertions bổ sung | Token roundtrip, parse/validation, sentinel identity, default target/user |
| Preferences | Store tests và đổi setting/reopen trên app QA | JSON/path/default và save/load giữ nguyên |
| I18n | Package/UI tests, ảnh VI/EN | Embed và nội dung dịch không đổi |
| Version/build | Build script, metadata binary, UI | Version/commit đúng, icon còn |
| Account/discovered stores | Unit, manager two-store, native two-store | Active/ready/needs-OAuth và persistence không đổi |
| Connector/telemetry | Existing fixture/server tests | Discovery/RPC/OAuth/quota/cache giữ kết quả |
| Switch/automation/continuation | Manager, automation E2E/runtime regression, reloader tests | Thứ tự và side effect theo fixtures, không đổi policy |
| Native UI | Bốn native tests và ảnh | Geometry, pointer, pin, resize, maximize/restore, refresh |
| Release binary | Mở executable mới build | Startup/sidebar/theme/locale/footer/tray/hide-show/exit/single-instance |

**Chuẩn bị release smoke bắt buộc:** chỉ đổi cwd không cô lập được dữ liệu. `internal/store/sanitizer.go` ưu tiên tìm repo từ executable trước cwd; `internal/ui/app.go` lấy logs/preferences từ parent của account store và có thể tự enable automation theo preferences đã lưu.

1. Dùng Windows user/VM QA không chứa token hoặc Chrome profile cá nhân cho smoke cần cô lập hoàn toàn. Override account store không cô lập Windows Credential Manager hoặc profile IDE/Chrome.
2. Tạo một QA root mới cho lượt chạy tại `.tmp/phase-1b-smoke-<mã lượt chạy>`, resolve thành absolute path dưới repo; tạo `accounts_store` và `logs` bên trong. Đây là runtime artifact, không cộng vào inventory source.
3. Lưu giá trị hiện tại của biến môi trường `ANTIGRAVITY_STORE_DIR`; đặt biến của process launch thành absolute path `<QA root>/accounts_store` trước khi khởi chạy binary. Không thay environment toàn máy. Giữ override cho mọi lần reopen và kiểm single-instance của lượt smoke.
4. Trước lần mở đầu tiên, ghi `<QA root>/logs/ui_preferences.json` bằng UTF-8 với `{"theme":"dark","mask_accounts":true,"auto_switch":false,"always_on_top":true,"language":"vi"}`. Dùng root mới để không kế thừa preference hoặc token cache của lượt khác.
5. Chạy đúng absolute path binary vừa build, working directory là QA root. Xác nhận auto-switch OFF; xác nhận dữ liệu/log/preferences được sinh dưới QA root, không phải kho repo gốc. Account store override đồng thời đặt discovered store, quota log và timeline dưới cùng parent theo source baseline.
6. Không bật automation, switch, add/delete account thật trong smoke. Kiểm nghiệp vụ đó bằng fixture/hook; đổi theme/locale rồi reopen phải tiếp tục dùng chính QA store override và giữ auto_switch:false.
7. Sau khi đóng mọi process smoke, phục hồi giá trị environment ban đầu (hoặc bỏ biến nếu trước đó không có). Không xóa credential/default target hoặc account store thật để cleanup.

Live-read tests vẫn có thể đọc session thật nếu chạy trên Windows user hiện tại. Ghi rõ phạm vi read-only và SKIP; mọi log/ảnh phải được kiểm tra và che dữ liệu cá nhân trước lưu evidence. Đặc biệt `TestLiveLanguageServerDiscoveryAndStatus` trong `internal/connector/connector_test.go` có log ActiveEmail; không commit log thô có email. Việc biên tập evidence chỉ che dữ liệu nhạy cảm, không xóa failure, thay exit code hoặc đổi verdict.

Ghi path và SHA-256 executable đang mở. Kiểm theme dark/light/warm, VI/EN, lưu setting/mở lại và startup width/layout theo baseline đã nghiệm thu phase 1A. Native harness không thay cho việc mở release binary.

Integration cần IDE/OAuth thật nhưng thiếu môi trường phải ghi NOT RUN/SKIP riêng cùng giới hạn fixture. Nếu đó là acceptance bắt buộc thì gate BLOCKED; không suy ra “100% tích hợp thật” từ mocked tests.

## 5.4. Evidence, cleanup và rollback

- Evidence từng bước đặt tại `reports/QA/phase-1b/1b-0/`, `reports/QA/phase-1b/1b-1/`, `reports/QA/phase-1b/1b-2/`, `reports/QA/phase-1b/1b-3/`, `reports/QA/phase-1b/1b-4/`, `reports/QA/phase-1b/1b-5/`, `reports/QA/phase-1b/1b-6/`.
- Mỗi thư mục có `verification.md`, `verification.json`, `unit-tests.jsonl`, `native-tests.jsonl`, `build.log`, `dependencies.txt`; ảnh `native-window.png`, `native-maximize.png`, `native-two-store.png`, `runtime-dark-vi.png`, `runtime-light-en.png`, `runtime-warm-vi.png` khi gate đó chạy.
- Dùng `go test -json` cùng package/tags/filter thay cho lượt -v để thu JSONL; không phải chạy lại chỉ để đổi định dạng. Lưu exit code trước lệnh khác.
- MD/JSON ghi revision, dirty state, command/env, exit code, pass/fail/skip, binary hash, kết quả đọc ảnh, invariant và blocker. Không lưu token/blob/nội dung tài khoản cá nhân.
- Supervisor kiểm source/diff/evidence thật; PASS tài liệu không thay cho PASS implementation.
- Slice chưa commit bị lỗi: giữ evidence; nếu cần chỉ hoàn nguyên file do slice sở hữu, giữ thay đổi có sẵn. Slice đã commit: revert đúng commit theo quy trình, không reset lịch sử.
- Không có schema migration; không rollback bằng xóa account stores, credentials hoặc logs người dùng.
- Chỉ dọn artifact do đợt này sinh đã xác định superseded; giữ evidence hợp lệ và lịch sử báo cáo.

---

# PHẦN 6: ĐIỀU KIỆN KẾT THÚC VÀ BÀN GIAO

Phase 1B hoàn tất khi map khớp source, core độc lập, import/script paths sạch, hành vi được bảo toàn, mọi gate bắt buộc có evidence PASS, Codex chỉ có gitkeep, QA/Supervisor PASS và worktree đối chiếu đúng baseline sau commit. Owner kiểm tra cấu trúc thực tế trước mở phase 2.

Phase 2 mới thiết kế Codex, IDEEngine contract/lifecycle và tab Codex. Không tự triển khai chúng từ báo cáo 1B.

## Nhật ký R2 và review tài liệu

- Output cập nhật trực tiếp: chính báo cáo này; không tạo bản sao timestamp theo yêu cầu Owner.
- Đã điều tra: inventory tracked/generated, caller imports, raw/token source/tests, preferences/theme, build script/version test và native harness.
- Quyết định: tách raw/token, hoãn interface thiếu contract, sửa build paths, cập nhật caller nguyên tử, thêm native/runtime/evidence/rollback gates, sửa số lượng.
- Chưa thực hiện trong lượt cập nhật tài liệu: move source, build/test sản phẩm hoặc nghiệm thu implementation.
- Review độc lập Architect Review: **PASS tài liệu R2** sau khi sửa smoke isolation bằng absolute ANTIGRAVITY_STORE_DIR và seed auto_switch:false. Đã đối chiếu map/caller/raw-token/build/native gates; không còn blocker tài liệu được phát hiện. Đây không phải PASS implementation/runtime.
- Kiểm tra tài liệu: 104 map rows bao phủ đủ 103 tracked source/asset và một syso generated; không trùng source/target; 7 khối PowerShell parse thành công; git diff --check PASS.
- Commit tài liệu: tra lịch sử Git của file; commit baseline không phải commit triển khai.
