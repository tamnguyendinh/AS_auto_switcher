# BÁO CÁO ĐỐI SOÁT ZERO-TRUST TOÀN DIỆN PHASE 6: PYTHON VS GO
**Dự án**: Antigravity Multi-Account Switcher  
**Ngày thực hiện**: 2026-10-16  
**Đơn vị thực hiện**: Supervisor Reviewer Lane (Zero-Trust Special Agent)  
**Ủy quyền tối cao**: Human Owner Mandate & CEO Agent Directive  
**Tiêu chuẩn áp dụng**: `supervisor` + `system-architect` + `data-integrity-review`  
**Nguyên tắc chỉ đạo số 1**: "GO KHÔNG ĐƯỢC PHÉP SUY ĐOÁN XEM PYTHON TẠI SAO LÀM THẾ NÀY THẾ KIA. CONVERT SANG GO LÀ PHẢI GIỐNG 100% TÍNH NĂNG CHỨ KHÔNG ĐƯỢC SÁNG TÁC!"  
**Kết quả tổng thể**: **AUDIT_COMPLETE / REJECT FOR CURRENT GO IMPLEMENTATION** (Phát hiện 4 lỗi nghiêm trọng P0, 2 lỗi P1, 2 điểm lệch P2 cần Coder khắc phục dứt điểm).

---

## 1. MỤC TIÊU & PHƯƠNG PHÁP KIỂM CHỨNG ZERO-TRUST

Thực hiện kiểm chứng độc lập "Code First, Test Second", đối soát trực tiếp dòng-đối-dòng (line-by-line) và tính-năng-đối-tính-năng (feature-by-feature) giữa **100% mã nguồn Python gốc** (`core/*`, `scanner/*`, `ui/*`, `main.py`, `tests/*`) và **mã nguồn Go hiện tại** (`internal/*`, `cmd/*`).

Tuyệt đối không tin tưởng vào các kết quả `go test` có sẵn vì test có thể được viết sai để khẳng định hành vi lệch lạc. Toàn bộ nhận định trong báo cáo này đều dựa trên bằng chứng mã nguồn tĩnh và hành vi runtime thực tế.

---

## 2. MA TRẬN ĐỐI SOÁT TOÀN DIỆN 5 PHÂN HỆ (PARITY MATRIX)

| Phân hệ | Thành phần Python | Thành phần Go tương đương | Mức độ Parity | Đánh giá Zero-Trust |
| :--- | :--- | :--- | :--- | :--- |
| **Phân hệ 1: Chrome Discovery Scanner** | `scanner/chrome_scanner.py` (268 dòng) | `internal/scanner/chrome_scanner.go` (340 dòng) | **100% PARITY PASS** | Quét chuẩn `Local State`, `Preferences`, lọc phần mở rộng rác, gom đủ tài khoản, 0% cơ chế lọc blacklist nhân tạo. |
| **Phân hệ 2: Accounts Store & Cấu Trúc Dữ Liệu** | `core/account_store.py` (150 dòng)<br>`core/jwt_helper.py` (53 dòng) | `internal/store/account_store.go` (220 dòng)<br>`internal/store/sanitizer.go` (110 dòng)<br>`internal/model/types.go` (355 dòng) | **DISCREPANCY (P0, P1)** | **P0-01**: Go phân giải đường dẫn qua `os.Getwd()` thay vì neo thư mục gốc repo (`G:\antigravity_auto_switch\accounts_store`).<br>**P1-01**: Payload TokenData trong Go bị ô nhiễm schema khi marshal. |
| **Phân hệ 3: Windows Credential Manager** | `core/credential_manager.py` (145 dòng) | `internal/wincred/cred.go` (180 dòng)<br>`internal/wincred/types.go` (45 dòng) | **100% PARITY PASS** | Win32 API `CredReadW`, `CredWriteW`, `CredDeleteW`, `CredFree` qua Advapi32.dll; Target `LegacyGeneric:target=gemini:antigravity`; Type `CRED_TYPE_GENERIC`; Persist `CRED_PERSIST_LOCAL_MACHINE`. |
| **Phân hệ 4: Language Server Discovery & Connect-RPC** | `core/api_connector.py` (643 dòng) | `internal/connector/discovery.go` (305 dòng)<br>`internal/connector/api_client.go` (439 dòng)<br>`internal/connector/endpoints.go` (65 dòng) | **PARITY PASS (Bộ Tool Primitives)** | Dò PID, Netstat/ExtendedTcpTable, parse `--csrf_token`, headers `X-Csrf-Token` + `x-codeium-csrf-token`, các endpoints Connect-RPC đầy đủ. |
| **Phân hệ 5: Chu Trình Chuyển Đổi Hot-Swap & Session Reloader** | `core/state_resolver.py` (720 dòng)<br>`core/session_reloader.py` (993 dòng) | `internal/manager/account_manager.go` (441 dòng)<br>`internal/reloader/session_reloader.go` (245 dòng) | **CRITICAL REGRESSIONS (P0, P1)** | **P0-02**: Go tự ý gọi `AuthLogout` gây văng IDE và pop Chrome OAuth.<br>**P0-03**: Go gửi phím `Ctrl+R` thay vì kill `language_server.exe` để respawn nạp RAM.<br>**P0-04**: Go bỏ sót 100% chu trình Continuation (Active Threads Snapshot -> SmartFocus -> LoadTrajectory -> SendUserCascadeMessage).<br>**P1-02**: Tự ý sáng tác Auto-Failover/AccountPool, bỏ Always-on-top. |

---

## 3. CHI TIẾT CÁC SAI LỆCH VÀ KHUYẾT TẬT NGHIÊM TRỌNG

### 3.1. [P0-02] Tự Ý Gọi `AuthLogout` Khiến Trình Duyệt Bị Văng (Chrome OAuth Popup)
- **Mã nguồn Python chuẩn (`core/state_resolver.py` dòng 380-425, 470-485)**:
  ```python
  # Default is False (smooth zero-chrome hot-swap without OAuth popups)
  if skip_logout is not None:
      should_logout = not bool(skip_logout)
  elif force_logout is not None:
      should_logout = bool(force_logout)
  else:
      should_logout = bool(getattr(self, "force_logout", False))

  # Step 2 (AuthLogout): Only invoked if should_logout is True
  if should_logout and info is not None:
      logout = logout_fn or self.logout_fn or auth_logout
      try:
          logout(info)
      except Exception:
          pass
  ```
  *Python ghi chú rõ*: Mặc định **TUYỆT ĐỐI KHÔNG GỌI `auth_logout()`**. Bỏ qua `AuthLogout()` là chìa khóa sống còn để không kích hoạt thông báo đăng xuất lên Electron IDE, triệt tiêu 100% cửa sổ trình duyệt Chrome OAuth nhảy ra.
- **Mã nguồn Go vi phạm (`internal/manager/account_manager.go` dòng 398-403, `cmd/switcher/main.go` dòng 80, `internal/ui/app.go` dòng 165)**:
  ```go
  // internal/manager/account_manager.go:
  func (m *AccountManager) SwitchAccount(ctx context.Context, targetEmail string, skipLogout bool) (*SwitchResult, error) {
      ...
      // 4. Clear RAM via AuthLogout if not skipLogout
      if !skipLogout && client != nil {
          _ = client.AuthLogout(ctx)
      }
  ```
  Và tại `cmd/switcher/main.go`:
  ```go
  res, err := mgr.SwitchAccount(ctx, switchTarget, false) // skipLogout = false -> !skipLogout = true!
  ```
  Và tại `internal/ui/app.go`:
  ```go
  res, err := a.mgr.SwitchAccount(ctx, targetEmail, false) // skipLogout = false -> !skipLogout = true!
  ```
- **Hậu quả thực tế**: Go đã đảo ngược logic mặc định của Python! Go luôn luôn gửi request `AuthLogout` đến Language Server. Ngay khi nhận `AuthLogout`, Language Server báo cho IDE Antigravity là user đã đăng xuất, khiến Antigravity lập tức mở trình duyệt Chrome yêu cầu đăng nhập lại, phá hỏng hoàn toàn tính năng hot-swap ngầm!

---

### 3.2. [P0-03] Tự Ý Gửi Phím `Ctrl+R` Thay Vì Recycle `language_server.exe`
- **Mã nguồn Python chuẩn (`core/session_reloader.py` dòng 304-335, `core/state_resolver.py` dòng 525-545)**:
  ```python
  def recycle_language_server(wait_seconds: float = 5.0, ...):
      subprocess.run(
          ["taskkill", "/F", "/IM", "language_server.exe"],
          capture_output=True,
          timeout=timeout,
          creationflags=creationflags,
          check=False,
      )
      if wait_seconds > 0:
          sleeper(wait_seconds)
      return True
  ```
  *Cơ chế thực tế của Antigravity*: IDE cha (Electron) có tiến trình giám sát nội bộ `monitorLsCrashInternal`. Khi `language_server.exe` bị chấm dứt (`taskkill /F`), Electron cha sẽ tự động khởi sinh một instance `language_server.exe` hoàn toàn mới trong vòng ~2000ms. Tiến trình mới khởi động sẽ tự động gọi Win32 API `CredReadW` để nạp token mới từ Windows Credential Manager vào RAM, không cần bất kỳ thao tác UI nào.
- **Mã nguồn Go vi phạm (`internal/reloader/session_reloader.go` dòng 170-245)**:
  Go hoàn toàn không có hàm `RecycleLanguageServer`! Thay vào đó, Go tự ý sáng tác ra hàm gửi phím:
  ```go
  func ReloadAntigravityWindow(hwnds []windows.HWND) error {
      ...
      // Key down: Ctrl + R
      procKeybdEvent.Call(uintptr(VK_CONTROL), 0, 0, 0)
      procKeybdEvent.Call(uintptr(VK_R), 0, 0, 0)
      time.Sleep(50 * time.Millisecond)
      // Key up: R, then Ctrl
      procKeybdEvent.Call(uintptr(VK_R), 0, uintptr(KEYEVENTF_KEYUP), 0)
      procKeybdEvent.Call(uintptr(VK_CONTROL), 0, uintptr(KEYEVENTF_KEYUP), 0)
  }
  ```
- **Hậu quả thực tế**:
  1. Gửi `Ctrl+R` vào cửa sổ Antigravity chỉ đơn thuần là reload trang web Electron (renderer process), làm chớp giật giao diện người dùng, mất tab đang mở, nhưng **TIẾN TRÌNH `language_server.exe` VẪN ĐANG CHẠY TRONG RAM VỚI TOKEN CŨ**!
  2. Token mới ghi vào Credential Manager không hề được nạp vào Language Server vì tiến trình này không hề được restart!

---

### 3.3. [P0-04] Bỏ Sót Toàn Bộ Chu Trình Phục Hồi Hội Thoại Đang Dở Dang (In-Flight Continuation)
- **Mã nguồn Python chuẩn (`core/session_reloader.py` dòng 818-990, `core/state_resolver.py` dòng 560-640)**:
  Python thực hiện quy trình phục hồi 3 giai đoạn chặt chẽ:
  1. **Giai đoạn 1 (Snapshot trước khi switch)**:
     Hàm `capture_active_threads_snapshot()` đọc `app_storage.json`, bóc tách các pane hội thoại active (`aux-pane-v2-session.conversationPanes`), đối chiếu thư mục `~/.gemini/antigravity/brain/{thread_id}`, lưu snapshot JSON vào `accounts_store/active_threads_history/<timestamp>_active_threads.json`.
  2. **Giai đoạn 2 (Recycle Language Server)**:
     Kill `language_server.exe` và chờ 5 giây để respawn.
  3. **Giai đoạn 3 (Pure Native RPC Continuation)**:
     - Dò tìm lại tiến trình `find_language_server_process()` để lấy **PID mới, Port mới, CSRF token mới**.
     - Duyệt qua từng `thread_id` trong danh sách active threads:
       + Gọi Connect-RPC `SmartFocusConversation(new_info, thread_id)`
       + Gọi Connect-RPC `LoadTrajectory(new_info, thread_id)`
       + Gọi Connect-RPC `SendUserCascadeMessage(new_info, thread_id, prompt_text="Continue if you still have unfinished work in progress...", model_name=..., cascade_config=...)`
     - Không sử dụng bất kỳ giả lập bàn phím nào!
- **Mã nguồn Go vi phạm**:
  1. Trong Go, `internal/connector/api_client.go` đã viết sẵn các hàm `SmartFocusConversation`, `LoadTrajectory`, `SendUserCascadeMessage`, `DiscoverActiveAgentModel`.
  2. Nhưng tại `internal/manager/account_manager.go`, Go **KHÔNG HỀ GỌI** các hàm này! Chúng biến thành **DEAD CODE**!
  3. Go hoàn toàn không có hàm snapshot `capture_active_threads_snapshot`.
  4. Khi chuyển tài khoản, các tác vụ Cascade/Agent đang làm dở trên Antigravity bị bỏ rơi hoàn toàn, không tự động tiếp tục công việc như Python.

---

### 3.4. [P0-01] Phân Giải Sai Đường Dẫn Thư Mục `accounts_store/`
- **Mã nguồn Python chuẩn (`core/account_store.py` dòng 16-22)**:
  ```python
  def get_default_store_dir() -> str:
      base_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
      store_dir = os.path.join(base_dir, "accounts_store")
      os.makedirs(store_dir, exist_ok=True)
      return store_dir
  ```
  Python luôn luôn neo vào thư mục gốc của repository (`G:\antigravity_auto_switch\accounts_store`).
- **Mã nguồn Go vi phạm (`internal/store/sanitizer.go` dòng 88-96)**:
  ```go
  func GetDefaultStoreDir() string {
      cwd, err := os.Getwd()
      if err == nil {
          dir := filepath.Join(cwd, "accounts_store")
          return dir
      }
      return "accounts_store"
  }
  ```
- **Hậu quả thực tế**:
  Khi Go biên dịch thành `bin/switcher.exe` hoặc khi người dùng chạy từ shortcut, terminal với working directory khác, `os.Getwd()` sẽ tạo ra `bin\accounts_store` hoặc tìm kiếm sai chỗ, không đọc được 7 tài khoản JSON đã có sẵn trong `G:\antigravity_auto_switch\accounts_store\`. Vi phạm trực tiếp lệnh của Human Owner về việc dùng chung 1 kho dữ liệu.

---

### 3.5. [P1-01] Ô Nhiễm Cấu Trúc JSON TokenData (Schema Pollution)
- **Mã nguồn Python chuẩn (`core/account_store.py`, `core/credential_manager.py`)**:
  Python lưu và đọc `token_data` dưới dạng dictionary nguyên bản từ WinCred. File JSON trên đĩa có định dạng chuẩn:
  ```json
  {
    "email": "ngau2hung@gmail.com",
    "saved_at": 1792065252.052,
    "token_data": {
      "token": {
        "access_token": "ya29...",
        "token_type": "Bearer",
        "refresh_token": "1//...",
        "expiry": "2026-10-15T19:53:38.0169854+07:00"
      },
      "auth_method": "consumer",
      "id_token": "eyJhbGci..."
    }
  }
  ```
- **Mã nguồn Go vi phạm (`internal/model/types.go` dòng 130-165)**:
  Khi marshal mà không có `RawPayload`, Go tạo ra payload chứa cả các trường phẳng ở root:
  ```go
  output := map[string]interface{}{
      "access_token":  t.AccessToken,
      "refresh_token": t.RefreshToken,
      "token_type":    tokenType,
      "id_token":      t.IDToken,
      "auth_method":   authMethod,
      "token": map[string]interface{}{...},
  }
  ```
  Việc đưa `access_token` và `refresh_token` ra ngoài root làm sai lệch schema gốc của Antigravity Language Server. Cần bảo tồn nguyên vẹn `json.RawMessage` hoặc map nguyên bản 100% như Python.

---

### 3.6. [P1-02] Tự Ý Sáng Tác Tính Năng "Auto-Failover / AccountPool" Và Bỏ Quên "Always-on-top"
- **Mã nguồn Python (`ui/app_window.py` dòng 92-105)**:
  Có nút checkbox: `Ghim trên cùng (Always on Top)` bằng thuộc tính `-topmost`.
- **Mã nguồn Go (`internal/ui/window.go`, `internal/model/types.go`)**:
  - Go bỏ mất tính năng "Ghim trên cùng (Always on Top)".
  - Thay vào đó, Go tự ý đưa vào nút `⚡ Auto-Failover: BẬT`, tạo ra các khái niệm `AccountPool`, `FailoverConfig`, `MarkExhausted`, `ConsecutiveExhaustions` - những thứ không hề tồn tại trong mã nguồn Python gốc. Đây là hành vi vi phạm điều răn: "Go không được phép suy đoán hay tự ý sáng tác".

---

## 4. MA TRẬN ĐỐI SOÁT TỪNG DÒNG CODE VÀ HÀM TƯƠNG ĐƯƠNG

| Chức năng | File & Dòng Code Python Gốc | File & Dòng Code Go Hiện Tại | Điểm Lệch Cụ Thể | Phân Loại |
| :--- | :--- | :--- | :--- | :--- |
| **Phân giải kho JSON** | `core/account_store.py:16-22` (`base_dir = os.path.dirname(...)`) | `internal/store/sanitizer.go:88-96` (`cwd, _ := os.Getwd()`) | Go dùng `os.Getwd()` tạo nguy cơ lệch thư mục sang `bin\accounts_store`. Phải dùng logic dò tìm thư mục repo chứa `accounts_store`. | **P0-01** |
| **Bỏ qua AuthLogout** | `core/state_resolver.py:380-425` (`force_logout=False`, `should_logout=False`) | `internal/manager/account_manager.go:400` (`!skipLogout && client != nil`) | Go truyền `skipLogout=false`, luôn gọi `client.AuthLogout(ctx)`. Python cấm gọi mặc định. | **P0-02** |
| **Recycle Language Server** | `core/session_reloader.py:304-335` (`taskkill /F /IM language_server.exe`) | `internal/reloader/session_reloader.go:170-192` (`keybd_event VK_CONTROL + VK_R`) | Go không hề kill process, mà gửi phím `Ctrl+R`. Cần thay bằng `exec.Command("taskkill", "/F", "/IM", "language_server.exe")`. | **P0-03** |
| **Active Threads Snapshot** | `core/session_reloader.py:818-990` (`capture_active_threads_snapshot`) | *Hoàn toàn chưa có trong Go* | Go thiếu toàn bộ logic đọc `aux-pane-v2-session.conversationPanes` và ghi snapshot lịch sử. | **P0-04** |
| **Native RPC Continuation** | `core/state_resolver.py:560-640` (Dò lại LS mới -> SmartFocus -> LoadTrajectory -> SendUserCascadeMessage) | `internal/manager/account_manager.go:415-435` | Các hàm RPC đã có trong `connector`, nhưng bị bỏ quên hoàn toàn trong `SwitchAccount`. | **P0-04** |
| **Pre-patch ToS GCP** | `core/session_reloader.py:228-275` (`jetski.onboarding.lastLoginIsGcpTos = "false"`) | `internal/reloader/session_reloader.go:200-245` (`jetski.onboarding.lastLoginIsGcpTos = "false"`) | Đã làm đúng cấu trúc file và giá trị chuỗi literal `"false"`. | **PASS** |
| **Win32 Credential Write** | `core/credential_manager.py:80-110` (`Advapi32.dll CredWriteW`) | `internal/wincred/cred.go:68-115` (`procCredWriteW.Call`) | Đúng API, đúng hằng số `CredTypeGeneric=1`, `CredPersistLocalMachine=2`. | **PASS** |
| **Chrome Profile Scanner** | `scanner/chrome_scanner.py:40-260` | `internal/scanner/chrome_scanner.go:40-340` | Đúng thuật toán Local State, Preferences, regex fallback, 0% blacklist. | **PASS** |

---

## 5. MÃ GIẢ ĐỀ XUẤT CHO CODER ĐỂ SỬA CHỮA DỨT ĐIỂM (EXACT PSEUDOCODE FIXES)

### 5.1. Sửa `GetDefaultStoreDir` trong `internal/store/sanitizer.go`:
```go
func GetDefaultStoreDir() string {
    // 1. Kiểm tra biến môi trường hoặc thư mục hiện tại nếu có accounts_store
    if cwd, err := os.Getwd(); err == nil {
        dir := filepath.Join(cwd, "accounts_store")
        if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
            return dir
        }
    }
    // 2. Tìm ngược lên từ đường dẫn file thực thi (Executable)
    if exePath, err := os.Executable(); err == nil {
        exeDir := filepath.Dir(exePath)
        // Kiểm tra exeDir/accounts_store
        dir := filepath.Join(exeDir, "accounts_store")
        if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
            return dir
        }
        // Kiểm tra exeDir/../accounts_store (trường hợp binary nằm trong bin/)
        parentDir := filepath.Join(filepath.Dir(exeDir), "accounts_store")
        if stat, err := os.Stat(parentDir); err == nil && stat.IsDir() {
            return parentDir
        }
    }
    // 3. Fallback cố định vào workspace gốc
    return `G:\antigravity_auto_switch\accounts_store`
}
```

### 5.2. Sửa `RecycleLanguageServer` trong `internal/reloader/session_reloader.go`:
```go
// RecycleLanguageServer terminates language_server.exe cleanly and waits waitSeconds (default 5s).
// Antigravity's monitorLsCrashInternal automatically respawns a fresh instance with new credentials.
func RecycleLanguageServer(waitSeconds float64) error {
    cmd := exec.Command("taskkill", "/F", "/IM", "language_server.exe")
    // Hide console window on Windows
    cmd.SysProcAttr = &windows.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
    _ = cmd.Run() // Ignore exit code if process was already dead

    if waitSeconds > 0 {
        time.Sleep(time.Duration(waitSeconds * float64(time.Second)))
    }
    return nil
}
```

### 5.3. Sửa `SwitchAccount` trong `internal/manager/account_manager.go`:
```go
// SwitchAccount executes seamless hot-swap matching Python state_resolver.py 100%:
func (m *AccountManager) SwitchAccount(ctx context.Context, targetEmail string, forceLogout bool) (*SwitchResult, error) {
    normTarget := strings.ToLower(strings.TrimSpace(targetEmail))
    ...
    // BƯỚC 1: Probe live RAM identity
    currentEmail, _ := m.GetCurrentActiveEmail(ctx)
    if currentEmail != "" && currentEmail == normTarget {
        return &SwitchResult{Success: true, Message: "Tài khoản hiện đang hoạt động"}, nil
    }

    // BƯỚC 2: Snapshot active threads TRƯỚC KHI RECYCLE
    snapshotPath, activeThreads := reloader.CaptureActiveThreadsSnapshot("", "", "")

    // BƯỚC 3: Auto-capture active session vào store
    _, _ = m.CaptureCurrentSession()

    // BƯỚC 4: Load target token từ accounts_store/<email>.json
    targetToken, err := m.store.GetAccount(normTarget)
    if err != nil {
        return &SwitchResult{Success: false, Message: "Chưa có file JSON"}, err
    }

    // BƯỚC 5: Tuyệt đối KHÔNG gọi AuthLogout trừ khi forceLogout == true!
    if forceLogout && client != nil {
        _ = client.AuthLogout(ctx)
    }

    // BƯỚC 6: Ghi token vào Windows Credential Manager
    if err := m.wincredWriter(targetToken); err != nil {
        return nil, err
    }

    // BƯỚC 7: Pre-patch ToS vào app_storage.json (lastLoginIsGcpTos = "false")
    _ = reloader.PrePatchToSAcceptance(normTarget, "")

    // BƯỚC 8: Bring Antigravity window to foreground
    _ = reloader.EnsureAntigravityForeground()

    // BƯỚC 9: Recycle Language Server (Taskkill /F /IM language_server.exe và chờ 5.0s)
    _ = reloader.RecycleLanguageServer(5.0)

    // BƯỚC 10: In-Flight Session Continuation (Re-probe new LS & Dispatch Cascade RPC)
    // 1. Dò tìm lại language server mới (PID mới, Port mới, CSRF token mới)
    newServerInfo, err := connector.DiscoverLanguageServer()
    if err == nil && newServerInfo != nil {
        newClient := connector.NewClient(newServerInfo)
        // 2. Dispatch continuation cho từng active thread từ snapshot
        for idx, thread := range activeThreads {
            if idx > 0 {
                time.Sleep(3 * time.Second) // Delay tránh nghẽn server
            }
            _ = newClient.SmartFocusConversation(ctx, thread.ThreadID)
            _ = newClient.LoadTrajectory(ctx, thread.ThreadID)
            _ = newClient.SendUserCascadeMessage(ctx, thread.ThreadID, "Continue if you still have unfinished work in progress. If all tasks are already completed, reply with: Currently there is no work to do.", "", nil)
        }
    }

    return &SwitchResult{
        Success: true,
        Message: fmt.Sprintf("Đã chuyển đổi thành công sang: %s", normTarget),
    }, nil
}
```

---

## 6. KẾT LUẬN & CHỈ ĐẠO TIẾP THEO

- **Kết luận của Supervisor**: Bản chuyển đổi Go hiện tại chưa đạt tính toàn vẹn 100% so với Python gốc. Sự xuất hiện của việc tự ý gọi `AuthLogout` và tự ý gửi phím `Ctrl+R` là nguyên nhân trực tiếp dẫn tới hiện tượng "Python chạy đúng mà Go chạy sai".
- **Handoff cho Coder**: Báo cáo này cung cấp đầy đủ căn cứ mã nguồn, vị trí dòng code, nguyên nhân cốt lõi và mã giả chính xác để Coder thực hiện sửa chữa dứt điểm, đưa Go về trạng thái parity 100% không tì vết.

---
*Báo cáo được lập và phê duyệt bởi Supervisor Zero-Trust Lane theo đúng quy trình ủy quyền của Human Owner.*