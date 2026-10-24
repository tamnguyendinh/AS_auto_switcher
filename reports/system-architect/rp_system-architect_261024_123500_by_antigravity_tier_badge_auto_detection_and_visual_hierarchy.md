# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Tính năng: Tự động phát hiện & Hiển thị phân cấp Tên Gói (Tier Badge) trên Thẻ Tài khoản

- **Mã tài liệu:** `rp_system-architect_261024_123500_by_antigravity_tier_badge_auto_detection_and_visual_hierarchy`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-24
- **Trạng thái:** PRODUCTION-READY BLUEPRINT (Turnkey Specification for Coder)
- **Mục tiêu:** Cung cấp bản đặc tả chi tiết 100% đến từng cấu trúc dữ liệu, thuật toán chuẩn hóa, API contract, bảng màu và giải pháp cập nhật giao diện không giật lag. Coder chỉ cần thực thi theo đúng tài liệu mà không cần phải tự suy đoán.

---

## 1. TỔNG QUAN YÊU CẦU & BỐI CẢNH (EXECUTIVE SUMMARY)

### 1.1. Hiện trạng
- Trên mỗi thẻ tài khoản (`AccountCardWidget`), bảng thông số Quota là một ma trận 3x3 (`internal/ui/quota_badge.go`):
  - Ô `[Row 0, Col 0]` hiển thị chữ cứng `"Quota"` màu `palette.TextMuted`.
  - Cột 1 có độ rộng cố định `col1W = float32(48.0)`.
- Nhược điểm: Chữ "Quota" lặp lại trên mọi thẻ không mang lại giá trị phân loại tài khoản, người dùng không phân biệt được đâu là tài khoản PRO, ULTRA, hay FREE.

### 1.2. Mục tiêu sau khi nâng cấp
- Thay thế chữ `"Quota"` bằng **Tên gói dịch vụ** (`PRO`, `ULTRA`, `FREE`, `BIZ`, `--`).
- **Tự động nhận diện 100%**: Không bắt người dùng nhập tay.
- **Phân cấp thị giác (Visual Hierarchy)**: Mỗi gói có màu sắc nhận diện đặc trưng (ULTRA: Vàng Amber nổi bật; PRO: Xanh Primary; FREE: Xám tinh tế).
- **Độ mượt tuyệt đối (Zero Flicker)**: Cập nhật in-place vào đối tượng `canvas.Text`, không re-layout hay rebuild container.

---

## 2. NGUYÊN TẮC BẤT BIẾN (INVARIANTS & CONSTRAINTS)

1. **Spatial Invariant (Bất biến không gian):**
   - Độ rộng Cột 1 (`col1W = 48.0px`) trong `quota_badge.go` là CỐ ĐỊNH TUYỆT ĐỐI.
   - Chuỗi tên gói hiển thị tối đa 5 ký tự (`PRO`, `ULTRA`, `FREE`, `BIZ`, `--`). Không được vượt quá 5 ký tự để tránh tràn cột.
2. **Dual-Source Invariant (Bất biến 2 nguồn dữ liệu):**
   - Tài khoản **ACTIVE**: Thu thập từ Antigravity Language Server loopback RPC (`/GetUserStatus`).
   - Tài khoản **INACTIVE / STORED**: Thu thập từ Google Cloud Code API (`/v1internal:loadCodeAssist`) thông qua OAuth refresh token.
3. **No-Flicker Invariant (Bất biến chống giật UI):**
   - Cấm hủy widget card hoặc rebuild `matrixTable` khi cập nhật tier. Phải gọi hàm cập nhật in-place: `w.tierText.SetText(...)` hoặc gán text và `.Refresh()`.
4. **Backward Compatibility Invariant:**
   - Dữ liệu lưu trong `accounts_store/<email>.json` và `quota_log.json` dùng tag `omitempty` để tương thích hoàn toàn với các file JSON cũ.
   - Parser của `GetUserStatus` phải hỗ trợ cả dạng `string` cũ lẫn dạng `map[string]interface{}` mới.

---

## 3. BẢNG MÃ CHUẨN HÓA DỮ LIỆU (TIER NORMALIZATION MATRIX)

Hàm chuẩn hóa toàn cục: `NormalizeTier(raw string) string`
Vị trí khuyến nghị: `internal/telemetry/tier.go` hoặc `internal/model/types.go`.

| Giá trị thô đầu vào (`raw`) | Kết quả chuẩn hóa (`string`) | Mô tả ý nghĩa |
|---|:---:|---|
| `TEAMS_TIER_PRO_ULTIMATE`, `ultra-tier`, `g1-ultra-tier`, `Ultra`, `ultra` | **`ULTRA`** | Google AI Ultra (Hạn mức cao nhất) |
| `TEAMS_TIER_PRO`, `pro-tier`, `g1-pro-tier`, `Pro`, `pro`, `PLAN_PRO`, `Google AI Pro` | **`PRO`** | Google AI Pro / Google One AI Premium |
| `free-tier`, `Free`, `free`, `PLAN_FREE` | **`FREE`** | Gói cá nhân miễn phí |
| `TEAMS_TIER_ENTERPRISE_...`, `standard-tier`, `Enterprise`, `Teams` | **`BIZ`** | Tài khoản Workspace / Doanh nghiệp |
| Chuỗi rỗng `""`, `unknown`, `[Cần OAuth]` | **`--`** | Chưa có token hoặc chưa tải xong |

```go
// Thuật toán chuẩn hóa:
func NormalizeTier(raw string) string {
    trimmed := strings.ToLower(strings.TrimSpace(raw))
    switch {
    case strings.Contains(trimmed, "ultra") || strings.Contains(trimmed, "ultimate"):
        return "ULTRA"
    case strings.Contains(trimmed, "pro"):
        return "PRO"
    case strings.Contains(trimmed, "enterprise") || strings.Contains(trimmed, "standard") || strings.Contains(trimmed, "team"):
        return "BIZ"
    case strings.Contains(trimmed, "free"):
        return "FREE"
    default:
        return "--"
    }
}
```

---

## 4. BẢNG MÀU PHÂN CẤP THỊ GIÁC (COLOR HIERARCHY SPECIFICATION)

Vị trí cài đặt: `internal/ui/quota_badge.go`

```go
var (
    // ColorTierUltra: Vàng Amber nổi bật rực rỡ (#f59e0b)
    ColorTierUltra = color.NRGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}
    
    // ColorTierBiz: Xanh lục doanh nghiệp (#10b981)
    ColorTierBiz = color.NRGBA{R: 0x10, G: 0xb9, B: 0x81, A: 0xff}
)

func TierColor(tier string, palette ThemePalette) color.Color {
    switch strings.ToUpper(strings.TrimSpace(tier)) {
    case "ULTRA":
        return ColorTierUltra
    case "PRO":
        return palette.Primary // Xanh Cyan / DodgerBlue theo theme đang chọn
    case "BIZ":
        return ColorTierBiz
    case "FREE":
        return palette.TextMuted // Xám mờ tinh tế
    default:
        return palette.TextMuted
    }
}
```

---

## 5. ĐẶC TẢ CHI TIẾT TỪNG TẦNG (LAYER-BY-LAYER SPECIFICATION)

### 5.1. Tầng Connector: `internal/connector/api_client.go`
**Mục tiêu:** Chữa lành `GetUserStatus` để bóc tách đúng cấu trúc JSON Map mới của Antigravity mà không làm hỏng string cũ.

1. **Cập nhật `UserStatus` struct (`internal/connector/endpoints.go`):**
   ```go
   type UserStatus struct {
       Email       string                 `json:"email"`
       Name        string                 `json:"name,omitempty"`
       DisplayName string                 `json:"display_name,omitempty"`
       Status      string                 `json:"status,omitempty"`
       PlanName    string                 `json:"plan_name,omitempty"`  // "Pro", "Ultra"...
       TeamsTier   string                 `json:"teams_tier,omitempty"` // "TEAMS_TIER_PRO"...
       Raw         map[string]interface{} `json:"raw,omitempty"`
   }
   ```
2. **Cập nhật parser trong `GetUserStatus` (`internal/connector/api_client.go`):**
   ```go
   var planName string
   var teamsTier string

   // 1. Phân tích nếu planStatus là Map lồng nhau (Antigravity 2.x chuẩn mới)
   if planMap, ok := userObj["planStatus"].(map[string]interface{}); ok {
       if info, ok := planMap["planInfo"].(map[string]interface{}); ok {
           if pn, ok := info["planName"].(string); ok {
               planName = pn
           }
           if tt, ok := info["teamsTier"].(string); ok {
               teamsTier = tt
           }
       }
   } else if planStr, ok := userObj["planStatus"].(string); ok {
       // 2. Tương thích ngược nếu là string đơn giản ("PLAN_PRO")
       planName = planStr
       teamsTier = planStr
   }

   // Fallback status text
   status := planName
   if status == "" {
       status, _ = userObj["status"].(string)
   }
   if status == "" {
       status = "ACTIVE"
   }

   return &UserStatus{
       Email:       email,
       Name:        name,
       DisplayName: name,
       Status:      status,
       PlanName:    planName,
       TeamsTier:   teamsTier,
       Raw:         rawData,
   }, nil
   ```

---

### 5.2. Tầng Telemetry: `internal/telemetry`
**Mục tiêu:** Gọi `loadCodeAssist` cho mọi tài khoản để lấy tier, và cache vào snapshot.

1. **Định nghĩa endpoint và struct (`internal/telemetry/client.go`):**
   ```go
   const DefaultGoogleCodeAssistURL = "https://daily-cloudcode-pa.googleapis.com/v1internal:loadCodeAssist"

   type rawCodeAssistResponse struct {
       CurrentTier struct {
           ID   string `json:"id"`
           Name string `json:"name"`
       } `json:"currentTier"`
       PaidTier struct {
           ID   string `json:"id"`
           Name string `json:"name"`
       } `json:"paidTier"`
   }
   ```
2. **Bổ sung trường `Tier` vào `AccountQuotaData` (`client.go`):**
   ```go
   type AccountQuotaData struct {
       Email        string             `json:"email"`
       Tier         string             `json:"tier,omitempty"` // "PRO", "ULTRA", "FREE", "--"
       Claude       ModelQuotaBucket   `json:"claude"`
       Gemini       ModelQuotaBucket   `json:"gemini"`
       RawBuckets   []ModelQuotaBucket `json:"raw_buckets,omitempty"`
       ResetTime    time.Time          `json:"reset_time"`
       Status       string             `json:"status"`
       ErrorMessage string             `json:"error_message,omitempty"`
       LastUpdated  time.Time          `json:"last_updated"`
   }
   ```
3. **Logic bóc tách Tier trong `RetrieveUserQuota` (`client.go`):**
   - Sau khi có `tokenResult.AccessToken`, gửi request song song hoặc tuần tự tới `DefaultGoogleCodeAssistURL` (Timeout: 3 giây, payload `"{}"`).
   - Nếu gọi thành công:
     ```go
     var caResp rawCodeAssistResponse
     if json.Unmarshal(caBytes, &caResp) == nil {
         rawTier := caResp.CurrentTier.ID
         if rawTier == "free-tier" && caResp.PaidTier.ID != "" {
             // Trường hợp có gói trả phí
             rawTier = caResp.PaidTier.ID
         }
         data.Tier = NormalizeTier(rawTier)
     }
     ```
   - Nếu gọi thất bại: fallback `data.Tier = "--"`.
4. **Logic trong `pollFromLanguageServer` (`internal/telemetry/pool.go`):**
   - Khi query từ Language Server của tài khoản active:
     ```go
     userStatus, err := client.GetUserStatus(ctx)
     if err == nil && userStatus != nil {
         raw := userStatus.PlanName
         if raw == "" { raw = userStatus.TeamsTier }
         data.Tier = NormalizeTier(raw)
     }
     ```
5. **Đồng bộ hóa trong `QuotaCache` (`cache.go`):**
   - File JSON snapshot tự động lưu trường `tier` nhờ tag `json:"tier,omitempty"`. Khi nạp từ đĩa (`LoadFromDisk`), `Tier` được phục hồi nguyên vẹn.

---

### 5.3. Tầng Store: `internal/store/account_store.go`
**Mục tiêu:** Lưu trường `Tier` trực tiếp vào file tài khoản `accounts_store/<email>.json`.

1. **Cập nhật `StoredAccountEnvelope`:**
   ```go
   type StoredAccountEnvelope struct {
       Email     string           `json:"email"`
       Tier      string           `json:"tier,omitempty"`   // Mới bổ sung
       Status    string           `json:"status,omitempty"`
       SavedAt   float64          `json:"saved_at"`
       TokenData *model.TokenData `json:"token_data"`
   }
   ```
2. **Cập nhật `SaveAccount` / `UpdateAccountTier`:**
   - Cho phép cập nhật trường `Tier` khi telemetry có dữ liệu mới.

---

### 5.4. Tầng Manager: `internal/manager/account_manager.go`
**Mục tiêu:** Đưa trường `Tier` lên `AccountDisplayItem` để cung cấp cho UI.

1. **Cập nhật `AccountDisplayItem`:**
   ```go
   type AccountDisplayItem struct {
       Email        string             `json:"email"`
       Name         string             `json:"name,omitempty"`
       AvatarURL    string             `json:"avatar_url,omitempty"`
       ProfileDir   string             `json:"profile_dir,omitempty"`
       State        model.AccountState `json:"state"`
       Status       string             `json:"status"`
       IsActive     bool               `json:"is_active"`
       HasToken     bool               `json:"has_token"`
       DisplayBadge string             `json:"display_badge"`
       Tier         string             `json:"tier,omitempty"` // "PRO", "ULTRA", "FREE", "--"
   }
   ```
2. **Trong `GetAccountStates`:**
   - Gán `item.Tier = storeEnvelope.Tier` hoặc lấy từ `quotaCache.Get(item.Email).Tier`.

---

### 5.5. Tầng UI: `internal/ui/quota_badge.go` & `internal/ui/widgets.go`
**Mục tiêu:** Hiển thị và cập nhật reactive in-place nhãn tên gói.

1. **Sửa `createMatrixQuotaTableWithCells` (`quota_badge.go`):**
   ```go
   func createMatrixQuotaTableWithCells(
       dq DualCycleQuota, 
       tier string, 
       palette ThemePalette,
   ) (fyne.CanvasObject, *canvas.Text, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell, *matrixQuotaCell) {
       displayTier := NormalizeTier(tier)
       txtTier := canvas.NewText(displayTier, TierColor(displayTier, palette))
       txtTier.TextSize = BodyTextSize
       txtTier.TextStyle = fyne.TextStyle{Bold: true}

       // Cột 1: Thay txtQuota bằng txtTier
       colModel := container.New(&matrixColumnLayout{}, txtTier, cellGeminiModel, cellClaudeModel)
       // ... giữ nguyên toàn bộ layout bên dưới ...
       return table, txtTier, cellGemini5h, cellGeminiWeek, cellClaude5h, cellClaudeWeek
   }
   ```
2. **Cập nhật `AccountCardWidget` (`widgets.go`):**
   ```go
   type AccountCardWidget struct {
       // ... các trường cũ ...
       tierText *canvas.Text // Thêm trường tham chiếu này
   }

   func (w *AccountCardWidget) UpdateTier(tier string, palette ThemePalette) {
       if w == nil || w.tierText == nil {
           return
       }
       norm := NormalizeTier(tier)
       if w.tierText.Text == norm {
           return
       }
       w.tierText.Text = norm
       w.tierText.Color = TierColor(norm, palette)
       w.tierText.Refresh()
   }
   ```
3. **Móc nối vào luồng cập nhật:**
   - Trong `w.UpdateQuota(quota, palette)`:
     ```go
     if quota != nil && quota.Tier != "" {
         w.UpdateTier(quota.Tier, palette)
     }
     ```
   - Trong `w.UpdateState(item, palette)`:
     ```go
     if item.Tier != "" {
         w.UpdateTier(item.Tier, palette)
     }
     ```

---

## 6. DANH MỤC THAY ĐỔI KIỂM THỬ (TEST SUITE IMPACT & REGRESSION PLAN)

1. **`internal/connector/connector_test.go`:**
   - Bổ sung `TestGetUserStatus_NestedPlanInfo`: Mock response trả về `planStatus.planInfo.planName = "Pro"` và xác nhận `status.PlanName == "Pro"`.
   - Giữ nguyên `TestGetUserStatus` cũ (trả về `"planStatus": "PLAN_PRO"`) để xác nhận tương thích ngược.
2. **`internal/telemetry/telemetry_test.go`:**
   - Bổ sung `TestNormalizeTier`: Bảng kiểm thử các chuỗi `ultra-tier`, `TEAMS_TIER_PRO`, `free-tier`, `""`...
   - Bổ sung `TestLoadCodeAssist_Parser`: Mock payload Google Cloud Code và kiểm tra trích xuất Tier.
3. **`internal/ui/ui_test.go`:**
   - Cập nhật test case `TestCreateMatrixQuotaTable_Labels`:
     - Trước đây kiểm tra `requiredLabels := []string{"Quota", "Gemini", "Claude", "5h", "Week"}`.
     - Sửa assertion để chấp nhận hoặc `"Quota"`, hoặc tên gói chuẩn hóa (`"PRO"`, `"ULTRA"`, `"FREE"`, `"--"`).

---

## 7. BIÊN BẢN BÀN GIAO THỰC THI (CODER HANDOFF CHECKLIST)

Coder nhận bàn giao thực hiện lần lượt theo đúng 5 bước sau:
- [ ] **Bước 1:** Tạo file `internal/telemetry/tier.go` chứa `NormalizeTier(raw string) string`.
- [ ] **Bước 2:** Cập nhật `internal/connector/api_client.go` hỗ trợ parse map `planInfo` trong `GetUserStatus`.
- [ ] **Bước 3:** Bổ sung `Tier` vào `AccountQuotaData` (`client.go`) và gọi `loadCodeAssist`.
- [ ] **Bước 4:** Cập nhật `quota_badge.go` (`TierColor`, `txtTier`) và `widgets.go` (`UpdateTier`).
- [ ] **Bước 5:** Chạy `go test ./internal/...` để đảm bảo 100% test pass.

---
*(Bản đặc tả đã được hoàn thiện sản xuất và lưu vết tại `reports/system-architect/`).*
