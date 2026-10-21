# Supervisor Report: Two-store và card ba trạng thái

Verdict: PASS

## Metadata

- Ngày review: 2026-10-21, Asia/Bangkok.
- Scope: diff triển khai so với `3016f44`, báo cáo R2 và yêu cầu Owner về Tray, không tạo thêm plan, build bằng script repo, kiểm tra resize sau code.
- Claim: triển khai hai kho, card ba trạng thái/cập nhật tại chỗ, bỏ lưu phiên riêng, giữ kho hiện có và bảo tồn resize.
- Không bao gồm claim đã chạy một lần chuyển tài khoản thật sau thay đổi.
- Evidence: `reports/qa/two-store/2026-10-21/`; coder report `rp_coder_261021_113100_by_gpt-6_two_store_three_state.md`.
- Git reference: cùng commit với thay đổi source; baseline `3016f44`.

## Kết luận

Source, suite, native runtime và ảnh đã xem phù hợp phạm vi được giao. Quyết định PASS áp dụng cho triển khai và kiểm chứng phạm vi này, không biến smoke executable hoặc native fixture thành bằng chứng live OAuth/account-switch.

## Source-Level Clearance Notes

| Nhóm | Kết quả |
|---|---|
| Store/catalog | Ghi schema hai trường, đọc lại, loại trùng, cập nhật timestamp khi quét; lỗi đọc vẫn trả phần dữ liệu hợp lệ. Replacement thất bại không xóa bản cũ. |
| Resolver/startup | Override không bị bỏ qua; repo data ngoài bin; không còn hardcode máy cá nhân. Lỗi startup có phản hồi Windows. |
| Manager | Hợp nhất cả danh bạ và credential, giữ active khi capture không lưu được, warning không biến lỗi phụ thành mất toàn danh sách. History dùng store đã chọn. |
| UI | Có điều kiện HasToken, cập nhật cả card đang tồn tại và đường single-refresh; stale callback không gọi switch sau khi không còn đủ điều kiện. |
| Quota UI | N/A chỉ ở tầng hiển thị, không ghi quota giả vào cache; thanh quota giữ fraction tương ứng khi layout lại. |
| Tray/callbacks | Không còn SaveSession/onSave trong production source; capture manager vẫn được gọi từ luồng lấy danh sách và trước switch. |
| Resize | Thuật toán grip/frameless/grid không bị sửa. Thay đổi window layout duy nhất là status text ellipsis, có native test thông báo dài. |
| Engine và sibling | Source automation/telemetry/reloader/connector/wincred/model không đổi; thân SwitchAccount giữ nguyên. Full suite pass. |
| Docs/tests | README bỏ hướng dẫn menu đã xóa. Fixtures quota/ready được cập nhật HasToken; có test mới cho trạng thái thiếu credential và chuyển trạng thái. |

## Evidence Checked

Passed:

- Script repo `scripts/build_windows.ps1 -SkipTests`: PASS, tạo đúng `bin/antigravity-switch.exe`, icon 32×32.
- Full suite `go test ./... -count=1 -timeout 120s`: PASS.
- Vet `go vet -unsafeptr=false ./cmd/switcher/... ./internal/...`: PASS.
- Race store + manager: PASS.
- Native GLFW/Win32: refresh → manager/store fixture → card; lỗi capture; xóa/thêm credential; lặp refresh: PASS.
- Native resize tám hướng, minimum clamp, maximize/restore hai vòng, canvas/hit map: PASS.
- Grid 3→5→3 cột; card 280×130. Ảnh `three-states.png` và `maximized-grid.png` đã được mở và xem trực tiếp.
- Executable production đã mở, process phản hồi. Danh bạ live 7/7 file đúng hai trường; không sinh kho dưới bin.
- Diff/nguồn được đọc lại; thông tin kết quả không chỉ dựa trên assertion test.

Failures đã xử lý:

- Targeted đầu tiên có ba assertion của hợp đồng UI cũ; cập nhật tên badge và cách tìm renderer, targeted/full suite sau đó pass.
- Computer Use chụp production executable timeout hai lần. Không tiếp tục bấm tọa độ khi không có ảnh mới; dùng native test để kiểm chứng giao diện với giới hạn nguồn ảnh ghi rõ.
- Anvien không đăng ký repo; thay bằng source/call-site/Git diff review, không tuyên bố graph check pass.

Not run:

- Live đổi tài khoản/OAuth/continuation sau thay đổi.
- Race toàn repo.
- Clean build xóa dữ liệu thật.
- Browser/Playwright/Docker: không phải runtime của ứng dụng desktop Go/Fyne này.

## Invariant Closure

- Danh bạ không cấp credential hoặc trở thành nguồn ứng viên automation.
- Card không cung cấp switch cho inactive thiếu credential; các đường cập nhật đồng nhất.
- Bỏ thao tác lưu riêng vẫn giữ capture theo quyết định Owner.
- Kho hiện tại và dữ liệu phụ không bị chuyển sang thư mục build.
- Resize/hit map/grid được kiểm chứng bằng native runtime.
- Giới hạn còn lại: chưa có ảnh Computer Use của production executable và chưa lặp một switch thật; hai nội dung này không được dùng làm bằng chứng PASS.
- Không có lỗi source hoặc native runtime chưa xử lý trong phạm vi thay đổi được kiểm tra.

## Commit và bàn giao

Chỉ commit source/test/docs thuộc phạm vi này. Không đưa credential, danh bạ live, binary, screenshot hoặc log chứa dữ liệu người dùng vào commit. Các ảnh fixture và log kiểm chứng lưu tại evidence directory đã nêu.
