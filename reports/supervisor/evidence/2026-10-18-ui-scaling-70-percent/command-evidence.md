# Bằng chứng nghiệm thu GUI 70%

Target: `92b324542302f2fe311b886650d89ffbfef78d1b`. Ngày chạy: 2026-10-18, Asia/Bangkok.

| ID | Artifact | Nội dung |
|---|---|---|
| E00 | `E00-reviewed-diff.log` | Nguyên diff target trên ba file Touch-Map |
| E01 | `E01-source-evidence.json`, `E01-toolchain.json` | Geometry/wiring, F1 priority, F2 consumer search, F4 helper declarations, theme constants và toolchain |
| E02 | `E02-workspace-build.log` | Workspace build exit 0 |
| E03 | `E03-native-gui-build.log` | Native GUI build exit 0; binary riêng được dọn sau gate |
| E04 | `E04-go-test-verbose.log`, `E04-go-test-summary.json` | Full suite 10/10 package, 165 PASS, 0 FAIL, 0 SKIP |
| E05 | `E05-staticcheck.log` | Staticcheck exit 0, 0 diagnostic |
| E06 | `E06-runtime-probe.json`, `E06-runtime-summary.json`, `E06-runtime-command.json`, `E06-runtime-stderr.log` | 1075/1075 assertion, exit 0, stderr rỗng; geometry, quota/caches, refresh state và theme |
| E06 | `E06-headless-dark-1008x630.png`, `E06-headless-light-1008x630.png` | Canvas default Dark/Light, đã xem trực tiếp |
| E07 | `E07-headless-dark-840x525.png` | Canvas minimum, đã xem trực tiếp |

`command-results.json` giữ command, reviewed commit, thời điểm UTC+07, exit code và thời gian từng gate. Log build/staticcheck rỗng là bình thường khi lệnh thành công.

Chạy lại từ repo root khi cần một lần nghiệm thu mới:

```powershell
& 'reports/supervisor/evidence/2026-10-18-ui-scaling-70-percent/run_gates.ps1'
& 'reports/supervisor/evidence/2026-10-18-ui-scaling-70-percent/run_runtime_probe.ps1'
```

Probe import production UI và dùng Fyne `test.NewApp` software canvas scale 1, không phải native OS screenshot. Accounts/token/storage synthetic; storage tạm tạo trong `.tmp` và dọn khi kết thúc. `runtime_probe.go` có build constraint `ignore`, chỉ chạy khi được gọi tường minh.

Các hằng expected về geometry, màu theme, tuple quota và text 5h/week độc lập với report Coder. Source sản phẩm không bị sửa trong quá trình nghiệm thu. Full report: `reports/supervisor/2026-10-18-ui-scaling-70-percent-supervisor-report.md`.
