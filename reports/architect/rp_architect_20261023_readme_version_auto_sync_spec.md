# BÁO CÁO ĐẶC TẢ KIẾN TRÚC & THIẾT KẾ TRIỂN KHAI CHI TIẾT
## GIẢI PHÁP TỰ ĐỘNG HÓA ĐỒNG BỘ PHIÊN BẢN (VERSION) TRONG README.MD VÀ HỆ SINH THÁI BUILD
### SINGLE SOURCE OF TRUTH (SSOT) & ZERO-DRIFT SYNCHRONIZATION ARCHITECTURE

- **Mã tài liệu:** `rp_architect_20261023_readme_version_auto_sync_spec`
- **Tác giả:** System Architect Lane
- **Ngày lập:** 2026-10-23
- **Thẩm quyền:** Human Owner Directive ("version trong readme ko tu dong cap nhat theo version cua app. goi lane ra xu ly van de nay") & CEO Agent Session
- **Đối tượng thụ hưởng:** Planner Lane, Coder Lane, QA Lane, Supervisor Lane
- **Trạng thái:** **`VERDICT: PASS / READY_FOR_CODER`**

---

## 1. KHẢO SÁT HIỆN TRẠNG & NGUYÊN NHÂN GỐC RỄ (ROOT CAUSE ANALYSIS)

### 1.1. Hiện trạng phân tán thông tin phiên bản
Qua khảo sát toàn bộ codebase của ứng dụng `antigravity-switch`, thông tin phiên bản hiện đang tồn tại tại 4 vị trí độc lập:

| STT | Vị trí tệp | Giá trị hiện tại | Bản chất / Vai trò |
|---|---|---|---|
| 1 | `internal/version/version.go` | `Version = "1.0.2.0"` | **Single Source of Truth (SSOT)** chính thức của ứng dụng Go, được chú thích rõ là nguồn chân lý duy nhất. |
| 2 | `scripts/build_windows.ps1` | `$AppVersion = "1.0.2.0"` (dòng 121) | **Bị hardcode cục bộ** trong hàm `Build-GUI`, inject qua `-ldflags` vào binary. |
| 3 | `cmd/switcher/app.rc` | `FILEVERSION 1,0,2,0`<br>`PRODUCTVERSION 1,0,2,0`<br>`VALUE "FileVersion", "1.0.2.0"`<br>`VALUE "ProductVersion", "1.0.2.0"` | **Bị hardcode cục bộ** trong file tài nguyên Windows PE header. |
| 4 | `README.md` | `# ⚡ Antigravity Multi-Account Switcher - **Version 1.0.1.9**` (dòng 1) | **Bị lệch phiên bản (Drift)**: vẫn còn lưu phiên bản cũ `1.0.1.9` do người phát triển nâng cấp app lên `1.0.2.0` nhưng quên cập nhật thủ công tài liệu. |

### 1.2. Nguyên nhân gốc rễ (Root Cause)
1. **Thiếu cơ chế tự động đồng bộ (No Automated Sync Hook):** Quy trình build và release qua `scripts/build_windows.ps1` chưa có bước quét và đồng bộ tệp `README.md` theo SSOT.
2. **Phân mảnh nguồn chân lý (Fragmented SSOT):** Script `scripts/build_windows.ps1` tự khai báo biến cứng `$AppVersion = "1.0.2.0"` thay vì đọc trực tiếp từ `internal/version/version.go`. Nếu lập trình viên tăng version trong `version.go` nhưng không sửa `build_windows.ps1`, cờ `-ldflags` sẽ ghi đè ngược lại giá trị cũ.
3. **Thiếu chốt chặn kiểm thử (Missing Test Gate):** Bộ test của ứng dụng (`go test ./...`) không có bài kiểm tra (contract test) xác nhận xem phiên bản trong `README.md` có khớp với `version.GetVersion()` hay không. Do đó, mã nguồn có thể được commit và release mà không phát hiện tình trạng lệch phiên bản.

---

## 2. NGUYÊN TẮC BẤT BIẾN KIẾN TRÚC (ARCHITECTURAL INVARIANTS)

Để triệt tiêu hoàn toàn sự sai lệch phiên bản trong tương lai, kiến trúc mới thiết lập 6 nguyên tắc bất biến bắt buộc:

1. **`Invariant-SSOT-Version` (Nguồn chân lý duy nhất tuyệt đối):**
   - Tệp [`internal/version/version.go`](file:///g:/antigravity_auto_switch/internal/version/version.go) là nguồn chân lý duy nhất (Single Source of Truth) cho phiên bản ứng dụng.
   - Mọi tệp tài liệu (`README.md`), tệp tài nguyên Windows (`cmd/switcher/app.rc`), và script tự động (`scripts/build_windows.ps1`) đều là các dẫn xuất (derived artifacts), **bắt buộc phải đồng bộ tự động theo `version.go`**.

2. **`Invariant-Automated-Readme-Sync` (Tự động đồng bộ README khi build):**
   - Mỗi khi script build [`scripts/build_windows.ps1`](file:///g:/antigravity_auto_switch/scripts/build_windows.ps1) được kích hoạt (bao gồm cả build đầy đủ lẫn build với `-SkipTests`), hệ thống tự động kiểm tra và cập nhật dòng tiêu đề của `README.md` nếu phát hiện có sự sai khác.
   - Định dạng dòng 1 tiêu đề `README.md` được chuẩn hóa:
     `# ⚡ Antigravity Multi-Account Switcher - **Version <X.Y.Z.W>**`

3. **`Invariant-Preflight-Sync-Gate` (Đồng bộ trước khi kiểm thử và biên dịch):**
   - Quá trình đồng bộ phiên bản diễn ra tại **Giai đoạn 0 (Stage 0: Pre-Flight Version Synchronization)**, ngay trước khi chạy `go vet`, `go test` hoặc `go build`.
   - Điều này bảo đảm mọi bài kiểm tra sau đó luôn vận hành trên trạng thái codebase đã đồng bộ 100%.

4. **`Invariant-CI-Contract-Enforcement` (Khóa chặn lệch phiên bản bằng Go Test):**
   - Bổ sung kiểm thử `TestVersion_ReadmeSync` trong [`internal/version/version_test.go`](file:///g:/antigravity_auto_switch/internal/version/version_test.go).
   - Kiểm thử này đọc tệp `README.md`, bóc tách số phiên bản trên đầu tệp và so sánh với `version.GetVersion()`.
   - Nếu phát hiện lệch phiên bản, test **FAIL NGAY LẬP TỨC** với thông báo hướng dẫn rõ ràng, ngăn chặn việc commit mã nguồn lệch phiên bản vào repository.

5. **`Invariant-Encoding-Safety` (Bảo toàn chuẩn mã hóa UTF-8 Không BOM):**
   - Tệp `README.md` chứa các ký tự đặc biệt (emoji ⚡, dấu tiếng Việt có dấu).
   - Quá trình ghi đè phiên bản trong PowerShell **TUYỆT ĐỐI KHÔNG DÙNG `Set-Content` mặc định** (tránh bị chèn UTF-8 BOM hoặc làm hỏng encoding trên PowerShell 5.1).
   - Bắt buộc sử dụng .NET API: `[System.IO.File]::WriteAllText($readmePath, $newContent, [System.Text.UTF8Encoding]::new($false))` để bảo toàn 100% định dạng UTF-8 No BOM và giữ nguyên xuống dòng.

6. **`Invariant-Zero-Overhead-SyncOnly` (Hỗ trợ chế độ đồng bộ độc lập không build):**
   - Script `build_windows.ps1` hỗ trợ tham số `-SyncOnly`.
   - Khi nhà phát triển chỉ muốn đồng bộ version giữa các file mà không cần biên dịch lại binary CGo (vốn mất thời gian), họ chỉ cần chạy `.\scripts\build_windows.ps1 -SyncOnly` với thời gian thực thi < 100ms.

---

## 3. THIẾT KẾ GIẢI PHÁP KỸ THUẬT CHI TIẾT (TECHNICAL SPECIFICATION)

### 3.1. Sơ đồ Luồng Hoạt động (Workflow Diagram)

```text
                  ┌──────────────────────────────────────────────┐
                  │          internal/version/version.go         │
                  │   var Version = "1.0.2.0" (Chân lý duy nhất) │
                  └──────────────────────┬───────────────────────┘
                                         │
                   [scripts/build_windows.ps1 kích hoạt]
                                         │
                                         ▼
                  ┌──────────────────────────────────────────────┐
                  │       Giai đoạn 0: Parse-AppVersion          │
                  │   Đọc chuỗi Version từ internal/version.go   │
                  └──────────────────────┬───────────────────────┘
                                         │
                     ┌───────────────────┴───────────────────┐
                     ▼                                       ▼
        ┌─────────────────────────┐             ┌─────────────────────────┐
        │   Sync-ReadmeVersion    │             │   Sync-AppRcVersion     │
        │ Đọc dòng 1 README.md:   │             │ Cập nhật FILEVERSION    │
        │ Regex so khớp & thay thế│             │ và StringFileInfo       │
        │ Ghi UTF-8 No-BOM        │             │ trong cmd/switcher/     │
        └────────────┬────────────┘             └────────────┬────────────┘
                     │                                       │
                     └───────────────────┬───────────────────┘
                                         │
                          (Nếu có cờ -SyncOnly: Thoát)
                                         │
                                         ▼
                  ┌──────────────────────────────────────────────┐
                  │ Giai đoạn 2: Pre-Flight Tests (go test)      │
                  │ TestVersion_ReadmeSync xác nhận README == Go │
                  └──────────────────────┬───────────────────────┘
                                         │ (PASS)
                                         ▼
                  ┌──────────────────────────────────────────────┐
                  │ Giai đoạn 3: Biên dịch Windows PE & Go GUI    │
                  │ windres biên dịch app.rc đã đồng bộ           │
                  │ go build nhúng -ldflags từ $AppVersion động  │
                  └──────────────────────────────────────────────┘
```

---

### 3.2. Chi tiết cơ chế trong `scripts/build_windows.ps1`

#### A. Trích xuất Phiên bản Động từ SSOT (Loại bỏ Hardcode)
Hàm `Get-AppVersionFromSource` đọc tệp `internal/version/version.go` bằng Regex chuẩn:
```powershell
function Get-AppVersionFromSource {
    param([string]$SourcePath = "internal\version\version.go")
    if (-not (Test-Path $SourcePath)) {
        Write-Error "Version source of truth not found: $SourcePath"
    }
    $content = Get-Content $SourcePath -Raw -Encoding UTF8
    if ($content -match 'Version\s*=\s*"([^"]+)"') {
        $v = $Matches[1].Trim()
        Write-Host "  -> Single Source of Truth detected: Version $v (from $SourcePath)" -ForegroundColor DarkGreen
        return $v
    }
    Write-Error "Failed to parse Version from $SourcePath"
}
```

#### B. Tự động Đồng bộ `README.md`
Hàm `Sync-ReadmeVersion` thực hiện:
1. Đọc nội dung `README.md`.
2. Dùng biểu thức chính quy (Regex) quét dòng 1:
   `^(#\s*⚡?\s*Antigravity Multi-Account Switcher\s*-\s*\*\*Version\s*)([0-9]+(?:\.[0-9]+){2,3})(\*\*.*)$`
3. So sánh phiên bản hiện tại trong file với `$TargetVersion`:
   - Nếu khác nhau: thay thế bằng `$TargetVersion`, ghi lại tệp bằng `[System.IO.File]::WriteAllText` với `[System.Text.UTF8Encoding]::new($false)`.
   - Ghi log thông báo: `[SYNC] README.md version updated from <old> to <target>`.
   - Nếu đã khớp: ghi log `[OK] README.md is already up to date (<target>)`.

#### C. Tự động Đồng bộ `cmd/switcher/app.rc`
Để đảm bảo toàn bộ tệp tài nguyên PE của Windows cũng tự động ăn theo mà không cần sửa tay:
1. Phân giải `$TargetVersion` sang định dạng dấu phẩy cho PE header (ví dụ `1.0.2.0` -> `1,0,2,0`).
2. Thay thế các trường:
   - `FILEVERSION .*` -> `FILEVERSION 1,0,2,0`
   - `PRODUCTVERSION .*` -> `PRODUCTVERSION 1,0,2,0`
   - `VALUE "FileVersion", ".*"` -> `VALUE "FileVersion", "1.0.2.0"`
   - `VALUE "ProductVersion", ".*"` -> `VALUE "ProductVersion", "1.0.2.0"`
3. Ghi lại tệp nếu có sự thay đổi.

#### D. Tham số `-SyncOnly`
Bổ sung tham số vào đầu script:
```powershell
param(
    [switch]$SkipTests,
    [switch]$Clean,
    [switch]$SyncOnly,
    [string]$OutDir = "bin"
)
```
Nếu `$SyncOnly` được truyền vào, script chạy xong Stage 0 đồng bộ phiên bản thì in thông báo thành công và dừng lại ngay, không chạy test hay build.

---

### 3.3. Chi tiết Chốt chặn Kiểm thử trong `internal/version/version_test.go`

Thêm bài kiểm thử hợp đồng `TestVersion_ReadmeSync`:
1. Xác định thư mục gốc của repository (bằng cách duyệt ngược tìm tệp `go.mod`).
2. Đọc tệp `README.md`.
3. Bóc tách số phiên bản trong dòng tiêu đề đầu tiên qua Regex: `\*\*Version\s+([0-9]+(?:\.[0-9]+){2,3})\*\*`.
4. So sánh với `GetVersion()`:
   ```go
   if readmeVersion != GetVersion() {
       t.Fatalf("README.md header version (%s) is out of sync with internal/version.Version (%s). Run '.\\scripts\\build_windows.ps1' or '.\\scripts\\build_windows.ps1 -SyncOnly' to synchronize.", readmeVersion, GetVersion())
   }
   ```
5. Nâng cấp các test case hiện hữu (`TestVersion_Default`) để kiểm tra tính hợp lệ về mặt định dạng (Quad-part Semantic Version) thay vì hardcode chuỗi cố định, giúp việc bump version trong tương lai không làm vỡ các bài unit test nền tảng.

---

## 4. BẢN ĐỒ CAN THIỆP MÃ NGUỒN KHÓA CỨNG (TOUCH-MAP)

Coder Lane chỉ được phép chỉnh sửa đúng các tệp sau theo phạm vi quy định:

| STT | Tệp tin | Phạm vi can thiệp | Mục đích |
|---|---|---|---|
| 1 | `scripts/build_windows.ps1` | Đầu file (param), thêm hàm `Get-AppVersionFromSource`, `Sync-ReadmeVersion`, `Sync-AppRcVersion` tại Stage 0, và dùng `$AppVersion` động trong `Build-GUI`. | Tự động hóa đồng bộ `README.md` và `app.rc` từ SSOT khi build hoặc khi chạy `-SyncOnly`. |
| 2 | `internal/version/version_test.go` | Bổ sung hàm `TestVersion_ReadmeSync` và tinh chỉnh `TestVersion_Default`. | Đóng vai trò Verification Gate (chốt chặn CI/CD), bảo đảm `README.md` không bao giờ bị lệch. |
| 3 | `README.md` | Dòng 1: đổi từ `**Version 1.0.1.9**` thành `**Version 1.0.2.0**`. | Đưa tài liệu về đúng trạng thái đồng bộ với phiên bản hiện tại của ứng dụng. |

> [!CAUTION]
> **Các tệp TUYỆT ĐỐI CẤM SỬA trong đợt này:**
> - `internal/version/version.go`: Giữ nguyên `Version = "1.0.2.0"` (đây là SSOT, không được thay đổi giá trị).
> - Mã nguồn nghiệp vụ của ứng dụng (`internal/manager/*`, `internal/ui/*`, `internal/connector/*`, `cmd/switcher/main.go`).

---

## 5. TIÊU CHÍ NGHIỆM THU KIẾN TRÚC (ACCEPTANCE CRITERIA)

| Mã AC | Tiêu chí | Điều kiện nghiệm thu thành công (PASS Condition) |
|---|---|---|
| **AC-01** | **Automated README Sync** | Chạy `.\scripts\build_windows.ps1` sẽ tự động phát hiện phiên bản từ `internal/version/version.go` và đồng bộ dòng 1 của `README.md` thành `# ⚡ Antigravity Multi-Account Switcher - **Version 1.0.2.0**`. |
| **AC-02** | **SyncOnly Mode** | Chạy `.\scripts\build_windows.ps1 -SyncOnly` thực thi thành công trong < 1 giây, hoàn tất đồng bộ `README.md` và `app.rc` mà không kích hoạt compiler CGo. |
| **AC-03** | **Dynamic Build Invariant** | Trong `scripts/build_windows.ps1`, biến `$AppVersion` không còn bị gán cứng `1.0.2.0`. Giá trị này luôn được phân giải động từ tệp `internal/version/version.go`. |
| **AC-04** | **CI Gate / Test Protection** | Chạy `go test ./internal/version/...` có test `TestVersion_ReadmeSync` khẳng định PASS. Nếu cố tình sửa `README.md` thành phiên bản khác, test phải FAIL với thông báo hướng dẫn rõ ràng. |
| **AC-05** | **Encoding & Formatting Integrity** | Tệp `README.md` sau khi đồng bộ giữ nguyên chuẩn UTF-8 No BOM, không bị lỗi icon ⚡ và các ký tự tiếng Việt có dấu. |
| **AC-06** | **PE Resource Synchronization** | Tệp `cmd/switcher/app.rc` luôn giữ được sự đồng bộ tuyệt đối về `FILEVERSION`, `PRODUCTVERSION` và `StringFileInfo` tương ứng với số phiên bản của ứng dụng. |

---

## 6. HƯỚNG DẪN CHI TIẾT DÀNH CHO CODER LANE (EXACT CODE IMPLEMENTATION BLUEPRINT)

### 6.1. Chi tiết chỉnh sửa `scripts/build_windows.ps1`

```powershell
# Thêm param -SyncOnly vào đầu file:
param(
    [switch]$SkipTests,
    [switch]$Clean,
    [switch]$SyncOnly,
    [string]$OutDir = "bin"
)

# Thêm Stage 0: Version Synchronization ngay sau phần kiểm tra $Clean:
Write-Host "[0/4] Checking and synchronizing application version..." -ForegroundColor Yellow

function Get-AppVersionFromSource {
    param([string]$SourcePath = "internal\version\version.go")
    if (-not (Test-Path $SourcePath)) {
        Write-Error "Version source of truth not found: $SourcePath"
    }
    $content = Get-Content $SourcePath -Raw -Encoding UTF8
    if ($content -match 'Version\s*=\s*"([^"]+)"') {
        return $Matches[1].Trim()
    }
    Write-Error "Failed to parse Version from $SourcePath"
}

function Sync-ReadmeVersion {
    param(
        [string]$ReadmePath = "README.md",
        [string]$TargetVersion
    )
    if (-not (Test-Path $ReadmePath)) {
        Write-Warning "README file not found: $ReadmePath"
        return
    }

    $content = Get-Content $ReadmePath -Raw -Encoding UTF8
    $pattern = '^(#\s*⚡?\s*Antigravity Multi-Account Switcher\s*-\s*\*\*Version\s*)([0-9]+(?:\.[0-9]+){2,3})(\*\*.*)$'

    if ($content -match $pattern) {
        $currentVersion = $Matches[2]
        if ($currentVersion -ne $TargetVersion) {
            Write-Host "  -> Synchronizing README.md version: $currentVersion -> $TargetVersion" -ForegroundColor Cyan
            # Replace only the version token in the header line
            $newContent = [regex]::Replace($content, $pattern, "${1}$TargetVersion${3}", [System.Text.RegularExpressions.RegexOptions]::Multiline)
            [System.IO.File]::WriteAllText((Get-Item $ReadmePath).FullName, $newContent, [System.Text.UTF8Encoding]::new($false))
            Write-Host "  [SYNC] README.md synchronized to Version $TargetVersion" -ForegroundColor Green
        } else {
            Write-Host "  [OK] README.md version is already in sync ($TargetVersion)" -ForegroundColor DarkGray
        }
    } else {
        Write-Warning "Could not match version header pattern in $ReadmePath"
    }
}

function Sync-AppRcVersion {
    param(
        [string]$RcPath = "cmd\switcher\app.rc",
        [string]$TargetVersion
    )
    if (-not (Test-Path $RcPath)) {
        return
    }

    $parts = $TargetVersion.Split('.')
    while ($parts.Count -lt 4) { $parts += "0" }
    $commaVersion = ($parts[0..3]) -join ','

    $content = Get-Content $RcPath -Raw -Encoding UTF8
    $newContent = $content
    $newContent = [regex]::Replace($newContent, 'FILEVERSION\s+[0-9,]+', "FILEVERSION $commaVersion")
    $newContent = [regex]::Replace($newContent, 'PRODUCTVERSION\s+[0-9,]+', "PRODUCTVERSION $commaVersion")
    $newContent = [regex]::Replace($newContent, 'VALUE\s+"FileVersion",\s*"[^"]*"', "VALUE `"FileVersion`", `"$TargetVersion`"")
    $newContent = [regex]::Replace($newContent, 'VALUE\s+"ProductVersion",\s*"[^"]*"', "VALUE `"ProductVersion`", `"$TargetVersion`"")

    if ($newContent -ne $content) {
        Write-Host "  -> Synchronizing $RcPath to $TargetVersion ($commaVersion)" -ForegroundColor Cyan
        [System.IO.File]::WriteAllText((Get-Item $RcPath).FullName, $newContent, [System.Text.UTF8Encoding]::new($false))
        Write-Host "  [SYNC] $RcPath synchronized." -ForegroundColor Green
    }
}

# Thực thi đồng bộ SSOT
$AppVersion = Get-AppVersionFromSource
Sync-ReadmeVersion -TargetVersion $AppVersion
Sync-AppRcVersion -TargetVersion $AppVersion

if ($SyncOnly) {
    Write-Host "================================================================================" -ForegroundColor Cyan
    Write-Host "⚡ VERSION SYNCHRONIZATION COMPLETE (-SyncOnly)" -ForegroundColor Green
    Write-Host "================================================================================" -ForegroundColor Cyan
    exit 0
}

# Trong Build-GUI, sử dụng biến $AppVersion đã trích xuất từ SSOT, xóa dòng $AppVersion = "1.0.2.0" hardcode cũ:
# function Build-GUI {
#     ...
#     # Sử dụng biến $AppVersion từ outer scope hoặc truyền param
#     ...
# }
```

### 6.2. Chi tiết chỉnh sửa `internal/version/version_test.go`

```go
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func TestVersion_ReadmeSync(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to locate repo root: %v", err)
	}

	readmePath := filepath.Join(root, "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	re := regexp.MustCompile(`\*\*Version\s+([0-9]+(?:\.[0-9]+){2,3})\*\*`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) < 2 {
		t.Fatalf("failed to find version pattern in README.md header")
	}

	readmeVersion := matches[1]
	appVersion := GetVersion()

	if readmeVersion != appVersion {
		t.Fatalf("README.md header version (%s) is out of sync with internal/version.Version (%s). Run '.\\scripts\\build_windows.ps1' or '.\\scripts\\build_windows.ps1 -SyncOnly' to synchronize.", readmeVersion, appVersion)
	}
}
```

---

## 7. KẾT LUẬN & BÀN GIAO (HANDOFF)

Bản đặc tả này đã xác lập hoàn chỉnh mô hình kiến trúc đồng bộ tự động cho phiên bản ứng dụng, giải quyết triệt để vấn đề lệch version trong `README.md` theo đúng chỉ đạo của Human Owner. 

- **Trạng thái:** Sẵn sàng chuyển giao cho **Planner Lane** lập kế hoạch thực thi và **Coder Lane** triển khai mã nguồn.
- **Commit mục tiêu:** Commit báo cáo kiến trúc độc lập này lên Git repository.
- **Handoff Target:** CEO Agent Session (`ebad9771-35b4-4afb-955f-1b75612f7630`).
