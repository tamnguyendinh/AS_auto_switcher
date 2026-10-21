# Coder Report: Hai kho dữ liệu, card ba trạng thái và bỏ lưu phiên trong Tray

- Ngày: 2026-10-21, Asia/Bangkok.
- Baseline: `3016f44`.
- Authority: báo cáo R2 `SPEC-ARCH-20261021-TWO-STORE-AND-UI-3-STATE` và yêu cầu triển khai của Owner.
- Điều chỉnh workflow từ Owner: không lập thêm bộ plan; dùng trực tiếp R2. Kiểm tra co giãn UI sau khi code xong; build bản giao bằng script repo.
- Trạng thái: code đã triển khai; build, toàn bộ test, vet, race store/manager và native UI pass. Bản executable đã được mở.
- Git reference: báo cáo này nằm trong cùng commit triển khai với source; baseline nêu trên là mốc đối chiếu.

## Invariant Family Map

| Nhóm | Trước | Sau | Bề mặt liên quan đã kiểm tra |
|---|---|---|---|
| Danh bạ và credential | Chỉ quét Chrome + store | Ghi/đọc danh bạ hai trường, hợp nhất theo email, nhớ tài khoản từng thấy | store, manager, scanner, nguồn tài khoản telemetry/automation |
| Trạng thái card | Mọi inactive đều có nút chuyển; nhánh update bỏ qua token | Active / có credential / cần OAuth; đổi nhãn và quota ngay trên card hiện có | widgets, app.updateUI, refresh từng tài khoản, theme rebuild |
| Capture không có thao tác lưu riêng | Có SaveSession qua Tray | Bỏ menu, hàm GUI và callback; giữ capture khi lấy danh sách/trước khi chuyển | tray, app, manager, callback wrapper và test |
| Đường dẫn | Có hardcode máy cá nhân và fallback CWD | Giữ override/kho repo, dữ liệu ngoài bin; portable dùng vị trí xác định | resolver, khởi tạo store, cache/preferences, history output |
| Bảo tồn hình học | Resize phụ thuộc MinSize của nội dung | Giữ resize; thông báo dài dùng ellipsis để không đẩy MinSize ngang | window layout, native grips, maximize, canvas, grid |
| Engine chuyển tài khoản | Chuỗi restart/xác minh/continuation hiện có | Giữ logic; history output bám store được chọn | manager.SwitchAccount, reloader, gate, telemetry, retry |

Không thêm daemon, database, watcher OAuth hoặc bộ xếp hạng automation mới.

## Các thay đổi

- Thêm `DiscoveredStore`: chuẩn hóa email, lưu email/scanned_at, thay thế file qua temp + sync + rename; lỗi replacement không xóa bản cũ.
- Danh bạ cho phép đọc phần dữ liệu còn hợp lệ đồng thời trả lỗi bản ghi hỏng; đọc không đổi scanned_at.
- Manager khởi tạo danh bạ cạnh kho token, hợp nhất scanner + catalog + credential + danh tính active. Không mất kết quả live khi danh bạ không ghi được.
- Thêm warnings vào overview. Phiên không tồn tại là bình thường; lỗi capture/lưu được phản hồi qua status bar khi lấy danh sách.
- Bấm refresh khi đang bận vẫn bị bỏ qua; lượt sau cập nhật cùng tài khoản. Có integration test với reader bị chặn để kiểm tra không chạy chồng.
- Card lưu HasToken, cập nhật action slot, nền/viền active và quota tại chỗ. Callback cũ được giữ tham chiếu cũng không thể chuyển sau khi trạng thái không còn cho phép.
- Tài khoản inactive thiếu credential hiện N/A, không có nút chuyển; icon quota giữ vị trí và được vô hiệu hóa vì chưa có nguồn credential. Tài khoản active vẫn có đường đọc quota hiện có.
- Bỏ toàn bộ SaveSession/onSave trong code và menu Tray. README cập nhật quy trình đăng nhập → lấy danh sách.
- Bỏ fallback workspace hardcode. Override không bị bỏ qua khi thư mục chưa tồn tại hoặc không hợp lệ; lỗi khởi tạo có hộp thoại Windows vì binary GUI không có console.
- Kho repo vẫn ở gốc, ngoài bin. History được gắn theo store đã chọn; cache và preferences tiếp tục dùng cùng store.
- Giữ các file resize/frameless và thuật toán grid. Chỉ thêm ellipsis cho status bar để lỗi dài không làm cửa sổ không thu nhỏ được.
- Sửa layout.fraction của quota cell cùng lúc cập nhật fill, tránh lần layout sau phục hồi độ dài thanh cũ.

## Kiểm chứng

Evidence: [results.json](G:/antigravity_auto_switch/reports/qa/two-store/2026-10-21/results.json).

| Kiểm tra | Lệnh/phạm vi | Kết quả |
|---|---|---|
| Build chính thức | `scripts/build_windows.ps1 -SkipTests` | PASS; 7.310 giây compile; exe 29,070,848 bytes; PE icon 32×32 |
| Full suite | `go test ./... -count=1 -timeout 120s` | PASS tất cả package có test |
| Vet | `go vet -unsafeptr=false ./cmd/switcher/... ./internal/...` | PASS |
| Race | `go test -race ./internal/store ./internal/manager -count=1 -timeout 120s` | PASS; store 5.743s, manager 5.704s |
| Native UI | `go test -tags "gui nativeui" ./internal/ui -run "^TestNative(TwoStore\|Window)" -count=1 -timeout 120s -v` | PASS; 6.535s |
| Production smoke | Mở `bin/antigravity-switch.exe` bằng Computer Use | Cửa sổ xuất hiện, process Responding=True; tạo 7/7 bản ghi danh bạ đúng schema |
| Vị trí live | Kiểm tra thư mục sau launch | Có kho ở gốc; không có bin/accounts_store hoặc bin/discovered_accounts |
| Diff | `git diff --check`; kiểm tra phạm vi | PASS |

Lần targeted đầu tiên có ba assertion cũ không còn phù hợp: tên badge active/ready và kiểu đối tượng vẽ nhãn. Đã cập nhật theo hành vi mới rồi chạy lại targeted và toàn bộ suite thành công. Không ghi đè lịch sử lỗi trong evidence.

Bản giao được build bằng script repo như Owner yêu cầu. `-SkipTests` chỉ tách bước build khỏi bước test; full suite và vet đã chạy riêng sau build. Không chạy clean xóa thư mục của người dùng.

### Test mới

| File | Test | Nội dung |
|---|---|---|
| internal/store/discovered_store_test.go | TestDiscoveredStoreRepeatReloadAndPartialRead | Lặp/concurrent save, schema hai trường, reopen, không đổi timestamp khi đọc, partial read |
| cùng file | TestDiscoveredStoreUnavailableAndFailedReplacement | Lỗi vị trí lưu, replacement thất bại giữ bản cũ, không rò temp |
| cùng file | TestResolveStoreLocationsAndInvalidOverride | Repo từ bin/CWD khác, go run, portable mới/cũ, override mới và lỗi |
| internal/manager/two_store_test.go | TestTwoStoreCaptureMergeRepeatAndRemember | Capture, loại trùng, credential-only, nhớ email không còn trong Chrome |
| cùng file | TestTwoStorePartialFailuresAndMissingSession | Không có phiên bình thường, lỗi capture/catalog không mất phần danh sách còn lại |
| cùng file | TestTwoStoreCaptureWriteFailureKeepsLiveIdentity | Lưu credential lỗi vẫn có active identity và warning |
| internal/ui/two_store_test.go | TestThreeStateCardUpdatesInPlace | Ba trạng thái, callback, root/cell identity, masking, N/A/unknown |
| cùng file | TestUpdateUIObservesCredentialChangesWithSameEmail | Đổi token trên cùng email và card |
| cùng file | TestRefreshRepeatedCaptureAndFailureFeedback | Refresh thật qua manager/store fixture, chống chạy chồng, capture, thông báo lỗi |
| internal/ui/two_store_native_test.go | TestNativeTwoStoreRefreshAndResize | Bấm native → file store → card; lặp refresh, xóa/thêm credential fixture, warning, resize |

Các test cũ về quota, layout và callbacks được cập nhật fixture để tài khoản sẵn sàng có HasToken=true. Không bỏ kiểm tra nút chuyển để né lỗi. Thao tác SaveSession bị bỏ được thay bằng kiểm tra capture qua Refresh.

### Co giãn UI tổng

- Kéo trái, phải, trên, dưới và bốn góc: PASS.
- Kích thước client Windows khớp canvas; hit map sau resize vẫn đúng.
- Thu nhỏ tới mức MinSize: 978×468 pixel ở DPI đang chạy; góc đối diện được giữ.
- Maximize/restore hai vòng: PASS; grid 3 → 5 → 3 cột.
- Cửa sổ maximize: 1920×1042 pixel theo work area.
- Card luôn giữ 280×130 đơn vị Fyne.
- Thông báo dài không tăng MinSize ngang.
- Đã mở và xem trực tiếp [ảnh ba trạng thái](G:/antigravity_auto_switch/reports/qa/two-store/2026-10-21/three-states.png) và [ảnh maximize](G:/antigravity_auto_switch/reports/qa/two-store/2026-10-21/maximized-grid.png).

## Bảo tồn và giới hạn bằng chứng

- Không sửa source các package automation, telemetry, reloader, connector, wincred, model.
- Không thay logic thân SwitchAccount; cấu hình history output theo store ở bước khởi tạo manager.
- Native UI test chạy cửa sổ GLFW/Win32 thật với manager/file store và credential fixture; không phải screenshot trình duyệt mô phỏng.
- Không thực hiện một lần chuyển tài khoản thật/OAuth/continuation mới trong đợt này. Bảo tồn các luồng này được kiểm tra bằng source diff và suite hiện có.
- Computer Use không chụp được cửa sổ executable sau hai lần thử: `FrameArrived timed out`, `window capture timed out`. Không dùng lỗi công cụ này làm bằng chứng UI app lỗi; ảnh đã xem là ảnh native test, không gắn nhãn thành ảnh production binary.
- Anvien detect_changes không chạy được vì repo chưa đăng ký. Đã đối chiếu source, call sites và Git diff trực tiếp.
- Race chỉ chạy store/manager, không phải toàn repo.
- Không tạo thêm bộ plan theo yêu cầu Owner; R2 là checklist thực hiện.
- Credential thật không được đưa vào fixture, log hoặc commit. Catalog live mới tạo được ignore.
- Lệnh dọn hai tệp trung gian `.tmp/update_card_tests.go` và `.tmp/two-store-switcher.exe` bị bộ duyệt tự động chặn với lý do `blocked by policy`; giữ nguyên các tệp ignored này, không đưa vào commit hoặc bản giao.

## Trạng thái bàn giao

Đã triển khai và kiểm chứng phạm vi hai kho/card/Tray/đường dẫn; co giãn UI tổng pass trên native Windows. Bản build mới đang mở cho Owner sử dụng. Những giới hạn bằng chứng nêu trên không được diễn giải thành một lần E2E chuyển tài khoản thật đã chạy.
