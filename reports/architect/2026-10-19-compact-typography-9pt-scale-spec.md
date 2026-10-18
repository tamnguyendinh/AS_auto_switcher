# Báo Cáo Đặc Tả Kỹ Thuật: Hệ Thống Typography Tối Giản Chuẩn 9pt – 9.5pt (High-Density Swiss Typography)

- **Tài liệu đặc tả:** `reports/architect/2026-10-19-compact-typography-9pt-scale-spec.md`
- **Ngày lập:** 2026-10-19
- **Làn thực hiện:** Senior System Architect
- **Cơ quan chỉ đạo:** Human Owner trực tiếp thảo luận và phê duyệt thang đo
- **Mục tiêu:** Giảm toàn diện kích thước chữ trên toàn bộ giao diện xuống chuẩn **9pt – 9.5pt** để thẻ tài khoản và thanh điều khiển thanh mảnh, sang trọng, đẳng cấp như các devtool cao cấp (Linear, Raycast, VS Code).
- **Trạng thái:** SPECIFICATION APPROVED BY OWNER — RETAINED LOCALLY (CHƯA GỬI CEO THEO ĐÚNG LỆNH OWNER)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT BẮT BUỘC DÀNH CHO CODER (MANDATORY IRON RULE)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**  
> **"KHI CODER CODE XONG, CODER PHẢI CHẠY FULL BUILD BẰNG SCRIPT CỦA REPO, CODER KHÔNG ĐƯỢC SỬA SCRIPT FULL BUILD."**  
>
> - Script full build bắt buộc: `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
> - Coder **TUYỆT ĐỐI KHÔNG ĐƯỢC PHÉP CHỈNH SỬA BẤT KỲ DÒNG NÀO** trong tệp script `scripts/build_windows.ps1`.
> - Quá trình build phải biên dịch sạch cả CLI lẫn GUI, 100% test suite PASS, kết thúc với Exit Code 0.

---

## 1. Bản Đồ Thang Đo Typography Đã Khóa (Approved Typography Scale)

```text
┌────────────────────────────────────────────────────────────────────────┐
│ [AS] Antigravity Multi-Account Switcher (12pt Bold)            _ |_| X │
│ Hoạt động | PID 21484 | Port 65144 (9pt)                   [ ☀️ ▼ ] │
│ [ Danh sách tài khoản ]  [ Làm mới Quota ]  [ 📌 Ghim ] (Nút 9pt)   │
├────────────────────────────────────────────────────────────────────────┤
│ ┌────────────────────────────────────────────────────────────────────┐ │
│ │ ngau2hung@gmail.com (9pt)                      [✔ Dùng]  [🔄]   │ │
│ │ ────────────────────────────────────────────────────────────────── │ │
│ │ Quota (9pt) │       5h (9pt)       │         Week (9pt)            │ │
│ │ Gemini (9.5)│ [██████████]  100%   │   [████░░░░░░]  43.8%         │ │
│ │ Claude (9.5)│ [██████████]  100%   │   [██████████]  100%          │ │
│ └────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────┘
```

### Bảng Định Nghĩa Cỡ Chữ Chi Tiết Từng Phần Tử:

| Phần tử giao diện | Cỡ chữ chuẩn (pt) | Độ đậm (Weight) | Màu sắc ngữ nghĩa |
|---|---|---|---|
| **Tiêu đề App Header** | **12.0 pt** | Bold | `palette.Text` |
| **Dòng Trạng Thái Server** | **9.0 pt** | Regular / Medium | Chữ "Hoạt động": `Success`; PID/Port: `TextMuted` |
| **Các nút bấm Toolbar TopSection** | **9.0 pt** | Medium (Padding 4px) | Nút phẳng, thanh thoát, chiều cao ~24-26px |
| **Dropdown Theme mini** | **11.0 pt (Icon)** | Regular | Icon `🌙`, `☀️`, `☕` |
| **Địa chỉ Email trên thẻ** | **9.0 pt** | Medium / Normal | `palette.Text` (Hiển thị trọn vẹn 28 ký tự) |
| **Nút `[Chuyển]` / `[✔ Dùng]`** | **9.0 pt** | Bold | Co gọn ~22px cao, vừa vặn bên cạnh email |
| **Tiêu đề bảng Quota (`Quota`, `5h`, `Week`)**| **9.0 pt** | Bold | `palette.TextMuted` (Màu xám kỹ thuật) |
| **Tên Model (`Gemini`, `Claude`)** | **9.5 pt** | Bold / Medium | `palette.Text` |
| **Số phần trăm Quota (`100%`, `43.8%`)** | **9.5 pt** | Bold | Đổi màu động theo Tier: Xanh/Vàng/Đỏ |
| **Thanh đo Mini Progress Bar** | Dài **38 px**, Cao **5 px** | CornerRadius: 2.5px | Bo góc tròn mượt mà dạng viên thuốc |

---

## 2. Hướng Dẫn Thi Công Chi Tiết Từng Tệp (Code Cụ Thể 100%)

### TỆP 1: `internal/ui/themes.go`
*Ghi đè phương thức `Size(name fyne.ThemeSizeName)` của cả 3 Theme (DarkTheme, LightTheme, WarmTheme) để thu nhỏ kích thước toàn cục từ 14pt $\rightarrow$ 9.5pt:*

```go
// Cập nhật hàm Size trong DarkTheme, LightTheme và WarmTheme:
func (t *DarkTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 9.5 // Giảm từ 14 xuống 9.5
	case theme.SizeNameCaptionText:
		return 8.5 // Giảm từ 11 xuống 8.5
	case theme.SizeNameInlineIcon:
		return 14  // Giảm từ 20 xuống 14
	case theme.SizeNameInnerPadding:
		return 4   // Giảm từ 8 xuống 4 để nút bấm thanh mảnh
	case theme.SizeNamePadding:
		return 4
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func (t *LightTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 9.5
	case theme.SizeNameCaptionText:
		return 8.5
	case theme.SizeNameInlineIcon:
		return 14
	case theme.SizeNameInnerPadding:
		return 4
	case theme.SizeNamePadding:
		return 4
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func (t *WarmTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 9.5
	case theme.SizeNameCaptionText:
		return 8.5
	case theme.SizeNameInlineIcon:
		return 14
	case theme.SizeNameInnerPadding:
		return 4
	case theme.SizeNamePadding:
		return 4
	default:
		return theme.DefaultTheme().Size(name)
	}
}
```

---

### TỆP 2: `internal/ui/quota_badge.go`
*Cập nhật toàn bộ TextSize và kích thước MiniBar trong bảng ma trận Quota:*

```go
// 1. Tiêu đề bảng: Quota, 5h, Week -> 9.0pt
txtQuota := canvas.NewText("Quota", palette.TextMuted)
txtQuota.TextSize = 9.0
txtQuota.TextStyle = fyne.TextStyle{Bold: true}

txt5h := canvas.NewText("5h", palette.TextMuted)
txt5h.TextSize = 9.0
txt5h.TextStyle = fyne.TextStyle{Bold: true}

txtWeek := canvas.NewText("Week", palette.TextMuted)
txtWeek.TextSize = 9.0
txtWeek.TextStyle = fyne.TextStyle{Bold: true}

// 2. Tên mô hình: Gemini, Claude -> 9.5pt
txtGemini := canvas.NewText("Gemini", palette.Text)
txtGemini.TextSize = 9.5
txtGemini.TextStyle = fyne.TextStyle{Bold: true}

txtClaude := canvas.NewText("Claude", palette.Text)
txtClaude.TextSize = 9.5
txtClaude.TextStyle = fyne.TextStyle{Bold: true}

// 3. Số phần trăm: 100%, 43.8% -> 9.5pt Bold
txtPct := canvas.NewText(formatFraction(frac), tierColor)
txtPct.TextSize = 9.5
txtPct.TextStyle = fyne.TextStyle{Bold: true}

// 4. Kích thước thanh Mini Progress Bar: Rộng 38px, Cao 5px, Bo góc 2.5px
const (
	miniBarWidth  = 38.0
	miniBarHeight = 5.0
	miniBarRadius = 2.5
)
```

---

### TỆP 3: `internal/ui/widgets.go`
*Cập nhật kích thước Email và nút bấm trên thẻ tài khoản:*

```go
// 1. Nhãn Email: 9.0pt, TextStyle Medium (Bold = false để thanh thoát)
func newEmailLabelWithTooltip(email string, bold bool) *emailLabelWithTooltip {
	l := &emailLabelWithTooltip{fullText: email}
	l.Text = email
	l.Truncation = fyne.TextTruncateEllipsis
	l.Wrapping = fyne.TextWrapOff
	// Đặt TextSize 9.0pt cho email
	l.ExtendBaseWidget(l)
	return l
}

// 2. Cụm nút thao tác bên phải thẻ:
// Nút [Chuyển] và badge [✔ Dùng] co gọn, font 9.0pt
activeBadge := widget.NewLabelWithStyle("[✔ Dùng]", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
// activeBadge tự động thừa hưởng TextSize 9.5pt từ Theme
```

---

### TỆP 4: `internal/ui/window.go`
*Cập nhật kích thước Header và Toolbar:*

```go
// 1. Tiêu đề App Header: 12.0pt Bold (Thay vì 14pt)
monogramText := canvas.NewText("AS", color.White)
monogramText.TextSize = 11.0
monogramText.TextStyle = fyne.TextStyle{Bold: true}

titleLabel := widget.NewLabelWithStyle("Antigravity Multi-Account Switcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
// TitleLabel dùng TextStyle{Bold: true}, font size 12.0pt

// 2. Trạng thái Server: 9.0pt
a.serverStatusLabel = widget.NewLabelWithStyle("Hoạt động  | Đang kết nối...", fyne.TextAlignLeading, fyne.TextStyle{Bold: false})

// 3. Các nút Toolbar tự động thu gọn về 9.5pt và innerPadding 4px từ Theme
```

---

## 3. Tiêu Chuẩn Nghiệm Thu Kỹ Thuật (Acceptance Criteria)

1. **Hiển Thị Cực Kỳ Tinh Gọn (High-Density):**
   - Chiều cao mỗi thẻ tài khoản rút gọn về khoảng **~70px – 74px**.
   - Bảng Quota hiển thị sắc sảo, số % và tên mô hình 9.5pt nổi bật hơn tiêu đề 9pt.
   - Email 9pt hiển thị dài, rõ nét, không bị thô.
2. **Nút Bấm Thanh Mảnh:**
   - Các nút Toolbar cao gọn gàng ~24–26px, không còn bị dày cộp như trước.
3. **Kỷ Luật Full Build 100% PASS:**
   - Chạy thành công lệnh:  
     `powershell -ExecutionPolicy Bypass -File .\scripts\build_windows.ps1 -Mode all`
   - Exit code = 0, tuyệt đối không chỉnh sửa script build.

---

## 4. Trạng Thái Bàn Giao

Tài liệu đã được lập hoàn chỉnh và commit vào Git:  
`reports/architect/2026-10-19-compact-typography-9pt-scale-spec.md`  
**TRẠNG THÁI: GIỮ NGUYÊN TẠI ĐÂY, CHƯA GỬI CHO CEO THEO ĐÚNG CHỈ ĐẠO CỦA HUMAN OWNER.**
