# BÁO CÁO ĐẶC TẢ KIẾN TRÚC CHI TIẾT (SYSTEM ARCHITECTURE SPECIFICATION)
## Thiết Kế Mở Rộng Dashboard UI: Đổi Tên Thành "Auto Switcher" & Dựng Khung Sidebar Bên Trái — Giai Đoạn 1: Tab Antigravity Độc Lập

- **Mã tài liệu:** `rp_system-architect_261028_194500_by_antigravity_auto_switcher_dashboard_sidebar_phase1_antigravity_tab`
- **Tác giả:** System Architect (Strategic Lane)
- **Ngày lập:** 2026-10-28
- **Thẩm quyền phê duyệt:** Human Owner (Chủ tịch) phê duyệt trực tiếp qua trao đổi kỹ thuật
- **Trạng thái:** PRODUCTION-READY BLUEPRINT (Khóa cứng Touch-Map, bàn giao chìa khóa trao tay cho Coder triển khai Giai đoạn 1)

---

# PHẦN 1: TỔNG QUAN CHIẾN LƯỢC & YÊU CẦU CỐT TỬ CỦA CHỦ TỊCH

### 1.1. Chuyển dịch tầm nhìn sản phẩm
Ứng dụng chuyển dịch từ một công cụ đơn lẻ (*Antigravity Multi-Account Switcher*) thành một **Platform Suite đa nền tảng mang tên "Auto Switcher"**, có khả năng quản lý và chuyển đổi tài khoản song song cho nhiều IDE/Agent.

### 1.2. Chỉ thị tối cao của Chủ tịch về lộ trình thực hiện
Chủ tịch đã chỉ đạo rõ ràng nguyên tắc thi công:
> *"Đặc tả phải triển khai theo từng bước: Code dashboard sidebar có tab Antigravity trước, PASS mới đến code thêm tab Codex xuất hiện trên sidebar. Cứ tuần tự từng chút một, không nên nóng vội, code đến đâu build đến đó để kiểm tra từng chức năng thì mới an toàn."*

Do đó, toàn bộ tài liệu kiến trúc này tập trung giải quyết **GIAI ĐOẠN 1 (PHASE 1)** một cách dứt điểm, an toàn tuyệt đối, không để sót bất kỳ điểm mù nào cho đội ngũ Coder.

### 1.3. Mục tiêu cụ thể của Giai đoạn 1
1. **Đổi tên ứng dụng trên Top Bar**: Thay thế tên cũ thành **`Auto Switcher`** (giữ nguyên logo Monogram **`[AS]`** vì AS là viết tắt hoàn hảo của Auto Switcher).
2. **Đồng bộ hóa tên thương hiệu toàn diện**: Cập nhật từ điển `internal/i18n` (cả bản `vi.json` và `en.json`), tiêu đề cửa sổ hệ điều hành Windows (`windowTitle`), và định danh Single-Instance Mutex.
3. **Mở rộng kích thước cửa sổ**: Nâng chiều rộng mặc định từ `920px` lên `980px` để bù đắp không gian cho thanh Sidebar 170px, đảm bảo không làm co hẹp lưới card tài khoản.
4. **Tái cấu trúc bố cục Dashboard (Layout Restructuring)**:
   - Phân chia khu vực thân app thành 2 khối độc lập: **Sidebar bên trái (170px)** và **Content Area bên phải**.
5. **Dựng Sidebar với duy nhất Tab Antigravity**:
   - Sử dụng **Logo thật, chính thức của Google Antigravity IDE** (tuyệt đối không dùng emoji `⚡` hay icon tự vẽ).
   - Nhãn hiển thị: `Antigravity`.
   - Trạng thái: Mặc định được chọn (Active).
   - Tuyệt đối **chưa đưa tab Codex lên giao diện ở Giai đoạn 1**.
6. **Bảo toàn 100% nghiệp vụ Antigravity (Zero-Regression)**:
   - Toàn bộ Toolbar và Lưới Card tài khoản Antigravity được chuyển sang Content Area bên phải, hoạt động ổn định và chính xác như trước.
   - Footer Bar giữ nguyên hiển thị trạng thái daemon/liveness của Antigravity và version `[v1.0.3.0]`.

---

# PHẦN 2: BỐ CỤC GIAO DIỆN GIAI ĐOẠN 1 (TARGET LAYOUT FOR PHASE 1)

```text
┌──────────────────────────────────────────────────────────────────────────────────┐
│ TOP BAR: [AS] Auto Switcher ────── (Kéo cửa sổ) ────── [VN|EN] [—] [□] [✕]        │
├─────────────────┬────────────────────────────────────────────────────────────────┤
│ SIDEBAR (170px) │ CONTENT AREA (Khung làm việc Antigravity)                      │
│                 │                                                                │
│ [Logo] Antigr.. │ ┌────────────────────────────────────────────────────────────┐ │
│ (Active Tab)    │ │ Toolbar: [+ Thêm] [F5] [Quota] [Ghim] │ [Ẩn] │ [Tự động]   │ │
│                 │ └────────────────────────────────────────────────────────────┘ │
│                 │ ┌────────────────────────────────────────────────────────────┐ │
│ (Chưa có Codex) │ │                                                            │ │
│                 │ │ Lưới Card Tài Khoản Antigravity (280x130 px)               │ │
│                 │ │ (Cuộn dọc độc lập trong vertical scroller)                 │ │
│                 │ │                                                            │ │
│                 │ └────────────────────────────────────────────────────────────┘ │
├─────────────────┴────────────────────────────────────────────────────────────────┤
│ FOOTER BAR: [Trạng thái Daemon / Liveness] ─────────────────────── [v1.0.3.0]    │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

# PHẦN 3: NGUYÊN TẮC BẤT BIẾN KIẾN TRÚC (ARCHITECTURAL INVARIANTS)

Coder bắt buộc phải tuân thủ nghiêm ngặt 6 bất biến kiến trúc sau trong quá trình lập trình:

1. **Official Brand Logo Invariant (Bất biến Logo thật):**
   - Nghiêm cấm dùng icon emoji (`⚡`, `🤖`) hoặc ký tự unicode làm biểu tượng trên Sidebar.
   - Phải dùng Logo chính thức của Google Antigravity IDE, định dạng file ảnh transparent PNG hoặc vector SVG nhúng trực tiếp qua cơ chế `_ "embed"` và `fyne.NewStaticResource`.
2. **Phased Isolation Invariant (Bất biến cô lập giai đoạn):**
   - Trong Giai đoạn 1, giao diện Sidebar chỉ hiển thị **duy nhất 1 Tab là Antigravity**. Tuyệt đối không xuất hiện chữ "Codex" hay nút tab Codex trên màn hình người dùng.
3. **Zero-Regression Invariant (Bất biến không hồi quy tính năng):**
   - Tất cả các nút Toolbar (`Add`, `Refresh`, `Refresh Quota`, `Pin`, `Mask Accounts`, `Auto Switch`, `Theme Select`) và các cơ chế xử lý sự kiện trong `internal/ui/app.go` phải hoạt động 100% nguyên vẹn.
   - Không được phép làm đứt gãy luồng dữ liệu của `accountListContainer` và `accountScroll`.
4. **Theme Parity Invariant (Bất biến đồng bộ Theme):**
   - Thanh Sidebar, đường viền phân cách, icon, nhãn tab và background của tab active phải phản ứng tức thì khi người dùng thay đổi 3 theme hệ thống (`ThemeDark`, `ThemeLight`, `ThemeWarm`).
5. **i18n Single-Source-of-Truth Invariant (Bất biến Đa ngữ):**
   - Tiêu đề trên Top Bar không được hardcode chuỗi tĩnh mà phải đọc động từ `i18n.Current().AppTitle`. Cả 2 file từ điển `vi.json` và `en.json` đều phải được cập nhật đồng bộ.
6. **Clean Worktree & Standard Build Invariant:**
   - Coder chỉ được chỉnh sửa các tệp thuộc Touch-Map được phân công.
   - Phải chạy script chuẩn `.\scripts\build_windows.ps1` để xác thực biên dịch và kiểm thử thành công trước khi bàn giao.

---

# PHẦN 4: HƯỚNG DẪN THI CÔNG CHI TIẾT DÀNH CHO CODER (TOUCH-MAP)

### 4.1. Tệp 1: Cập nhật Từ điển Đa ngữ
- Đường dẫn:
  - [`internal/i18n/dictionary.go`](file:///g:/antigravity_auto_switch/internal/i18n/dictionary.go)
  - [`internal/i18n/locales/vi.json`](file:///g:/antigravity_auto_switch/internal/i18n/locales/vi.json)
  - [`internal/i18n/locales/en.json`](file:///g:/antigravity_auto_switch/internal/i18n/locales/en.json)

#### A. Bổ sung trường trong `internal/i18n/dictionary.go`:
```go
type Dictionary struct {
    AppTitle          string `json:"app_title"`
    NavAntigravity    string `json:"nav_antigravity"`
    NavCodex          string `json:"nav_codex"`
    // ... các trường hiện có giữ nguyên 100%
}
```

#### B. Cập nhật `internal/i18n/locales/vi.json`:
```json
{
  "app_title": "Auto Switcher",
  "nav_antigravity": "Antigravity",
  "nav_codex": "Codex"
}
```

#### C. Cập nhật `internal/i18n/locales/en.json`:
```json
{
  "app_title": "Auto Switcher",
  "nav_antigravity": "Antigravity",
  "nav_codex": "Codex"
}
```

---

### 4.2. Tệp 2: Nhúng Tài nguyên Logo Antigravity IDE Thật
- Thư mục tài nguyên: `internal/ui/assets/`
- Tệp định nghĩa: [`internal/ui/assets.go`](file:///g:/antigravity_auto_switch/internal/ui/assets.go)

#### A. Trích xuất / Bổ sung file ảnh logo Antigravity:
Đặt tệp `antigravity_logo.png` (kích thước chuẩn 32x32 hoặc 64x64 px với nền trong suốt) vào `internal/ui/assets/antigravity_logo.png`.
*(Lưu ý: Có thể trích xuất icon chính thức từ ứng dụng Antigravity IDE cài đặt trên máy `C:\Users\TAM NGUYEN\AppData\Local\Programs\antigravity\Antigravity.exe` hoặc sử dụng asset chuẩn của Google Antigravity).*

#### B. Bổ sung Resource trong `internal/ui/assets.go`:
```go
//go:embed assets/antigravity_logo.png
var antigravityLogoBytes []byte

// AntigravityLogoResource cung cấp logo chính thức của Google Antigravity IDE cho thanh Sidebar.
var AntigravityLogoResource = fyne.NewStaticResource("antigravity_logo.png", antigravityLogoBytes)
```

---

### 4.3. Tệp 3: Xây dựng Module Sidebar Mới
- Đường dẫn tạo mới: `internal/ui/sidebar.go`

Tạo mới tệp `sidebar.go` để đóng gói toàn bộ logic hiển thị và giao diện của thanh Sidebar bên trái:

```go
package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"antigravity-switch/internal/i18n"
)

const SidebarWidth float32 = 170.0

// NavTab định danh các tab điều hướng trên Sidebar
type NavTab int

const (
	NavTabAntigravity NavTab = iota
	NavTabCodex
)

// sidebarItem đại diện cho một mục tab trên Sidebar
type sidebarItem struct {
	widget.BaseWidget
	tabID        NavTab
	iconResource fyne.Resource
	label        string
	isActive     bool
	onSelected   func(NavTab)

	// Các thành phần canvas đồ họa
	bg           *canvas.Rectangle
	activeBar    *canvas.Rectangle
	iconImg      *canvas.Image
	textLabel    *canvas.Text
	palette      ThemePalette
	isHovered    bool
}

func newSidebarItem(tabID NavTab, icon fyne.Resource, label string, isActive bool, palette ThemePalette, onSelected func(NavTab)) *sidebarItem {
	item := &sidebarItem{
		tabID:        tabID,
		iconResource: icon,
		label:        label,
		isActive:     isActive,
		palette:      palette,
		onSelected:   onSelected,
	}
	item.initCanvas()
	item.ExtendBaseWidget(item)
	return item
}

func (s *sidebarItem) initCanvas() {
	s.bg = canvas.NewRectangle(color.Transparent)
	s.bg.CornerRadius = 6

	s.activeBar = canvas.NewRectangle(color.Transparent)
	s.activeBar.CornerRadius = 2
	s.activeBar.SetMinSize(fyne.NewSize(3, 20))

	s.iconImg = canvas.NewImageFromResource(s.iconResource)
	s.iconImg.FillMode = canvas.ImageFillContain
	s.iconImg.SetMinSize(fyne.NewSize(20, 20))

	s.textLabel = canvas.NewText(s.label, s.palette.TextPrimary)
	s.textLabel.TextSize = 13
	s.textLabel.TextStyle = fyne.TextStyle{Bold: s.isActive}

	s.applyTheme(s.palette)
}

func (s *sidebarItem) applyTheme(palette ThemePalette) {
	s.palette = palette
	if s.isActive {
		// Active: Nền highlight nhẹ, vạch mép màu Primary, chữ đậm màu chính
		s.bg.FillColor = palette.SurfaceHighlight
		s.activeBar.FillColor = palette.Primary
		s.textLabel.Color = palette.TextPrimary
		s.textLabel.TextStyle = fyne.TextStyle{Bold: true}
	} else if s.isHovered {
		// Hover: Nền mờ nhẹ, vạch trong suốt, chữ màu chính
		s.bg.FillColor = color.NRGBA{R: 0x45, G: 0x47, B: 0x5a, A: 0x55}
		s.activeBar.FillColor = color.Transparent
		s.textLabel.Color = palette.TextPrimary
		s.textLabel.TextStyle = fyne.TextStyle{Bold: false}
	} else {
		// Normal: Nền trong suốt, chữ màu phụ (muted)
		s.bg.FillColor = color.Transparent
		s.activeBar.FillColor = color.Transparent
		s.textLabel.Color = palette.TextMuted
		s.textLabel.TextStyle = fyne.TextStyle{Bold: false}
	}
	s.bg.Refresh()
	s.activeBar.Refresh()
	s.textLabel.Refresh()
}

func (s *sidebarItem) SetActive(active bool) {
	s.isActive = active
	s.applyTheme(s.palette)
}

func (s *sidebarItem) CreateRenderer() fyne.WidgetRenderer {
	iconWrap := container.NewGridWrap(fyne.NewSize(20, 20), s.iconImg)
	contentBox := container.NewHBox(
		container.NewGridWrap(fyne.NewSize(3, 20), s.activeBar),
		canvas.NewRectangle(color.Transparent), // Khoảng cách nhỏ 4px
		iconWrap,
		container.NewCenter(s.textLabel),
	)
	paddedContent := container.NewPadded(contentBox)
	stack := container.NewStack(s.bg, paddedContent)
	return widget.NewSimpleRenderer(stack)
}

func (s *sidebarItem) MouseIn(*desktop.MouseEvent) {
	s.isHovered = true
	if !s.isActive {
		s.applyTheme(s.palette)
	}
}

func (s *sidebarItem) MouseOut() {
	s.isHovered = false
	if !s.isActive {
		s.applyTheme(s.palette)
	}
}

func (s *sidebarItem) MouseMoved(*desktop.MouseEvent) {}

func (s *sidebarItem) Tapped(*fyne.PointEvent) {
	if s.onSelected != nil {
		s.onSelected(s.tabID)
	}
}

// SidebarContainer chứa cấu trúc sidebar hoàn chỉnh
type SidebarContainer struct {
	widget.BaseWidget
	antigravityItem *sidebarItem
	divider         *canvas.Rectangle
	palette         ThemePalette
	container       *fyne.Container
}

func newSidebarContainer(palette ThemePalette, onTabSelected func(NavTab)) *SidebarContainer {
	sb := &SidebarContainer{palette: palette}
	d := i18n.Current()

	// Khởi tạo tab Antigravity (Active mặc định)
	sb.antigravityItem = newSidebarItem(
		NavTabAntigravity,
		AntigravityLogoResource,
		d.NavAntigravity,
		true, // Active
		palette,
		onTabSelected,
	)

	// Đường viền ngăn cách bên phải (1px)
	sb.divider = canvas.NewRectangle(palette.BorderSubtle)
	sb.divider.SetMinSize(fyne.NewSize(1, 0))

	// Danh sách các tab (Giai đoạn 1 chỉ có Antigravity)
	itemsBox := container.NewVBox(
		container.NewPadded(sb.antigravityItem),
	)

	sb.container = container.NewBorder(
		nil, nil, nil, sb.divider,
		itemsBox,
	)

	sb.ExtendBaseWidget(sb)
	return sb
}

func (sb *SidebarContainer) ApplyTheme(palette ThemePalette) {
	sb.palette = palette
	sb.divider.FillColor = palette.BorderSubtle
	sb.divider.Refresh()
	if sb.antigravityItem != nil {
		sb.antigravityItem.applyTheme(palette)
	}
}

func (sb *SidebarContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(sb.container)
}

func (sb *SidebarContainer) MinSize() fyne.Size {
	return fyne.NewSize(SidebarWidth, 0)
}
```

---

### 4.4. Tệp 4: Cập nhật Cửa sổ Chính & Tái cấu trúc Layout
- Đường dẫn: [`internal/ui/window.go`](file:///g:/antigravity_auto_switch/internal/ui/window.go)

#### A. Cập nhật Hằng số Tiêu đề & Kích thước:
```go
const windowTitle = "⚡ Auto Switcher"

const (
	DefaultWindowWidth  float32 = 980.0 // Nâng từ 920.0 lên 980.0
	DefaultWindowHeight float32 = 630.0
	MinWindowWidth      float32 = 640.0 // Nâng từ 480.0 lên 640.0
	MinWindowHeight     float32 = 360.0
)
```

#### B. Thêm trường trong struct `App` tại `internal/ui/app.go`:
```go
type App struct {
    // ... các trường hiện có
    sidebar     *SidebarContainer
    titleLabel  *widget.RichText
    // ...
}
```

#### C. Cập nhật `setupLayout()` trong `internal/ui/window.go`:
1. **Top Bar (Row 1 Header)**:
   Thay thế `titleLabel` thành biến quản lý trong `a.titleLabel`:
   ```go
   a.titleLabel = widget.NewRichText(&widget.TextSegment{
       Text: i18n.Current().AppTitle, // Lấy động từ i18n ("Auto Switcher")
       Style: widget.RichTextStyle{
           SizeName:  titleTextSizeName,
           TextStyle: fyne.TextStyle{Bold: true},
       },
   })
   titleLeft := container.NewHBox(monogramWrap, a.titleLabel)
   ```

2. **Khởi tạo Sidebar**:
   ```go
   a.sidebar = newSidebarContainer(palette, func(selectedTab NavTab) {
       // Xử lý chuyển tab khi người dùng bấm (Giai đoạn 1 giữ vững Antigravity)
   })
   ```

3. **Cấu trúc Khung Content Antigravity**:
   ```go
   // Toàn bộ giao diện Antigravity hiện hữu (Toolbar hàng 2 + Lưới Card cuộn dọc):
   antigravityContent := container.NewBorder(
       row2Combined, // Toolbar Antigravity (Thêm, F5, Quota, Ghim, Ẩn, Tự động, Theme)
       nil,
       nil,
       nil,
       scrollList,   // Lưới Card tài khoản Antigravity
   )
   ```

4. **Tổ chức Khung Thân App (Body Layout)**:
   ```go
   // Ghép Sidebar (170px) ở mép trái và Content Area ở bên phải:
   bodyLayout := container.NewBorder(
       nil,
       nil,
       a.sidebar,
       nil,
       antigravityContent,
   )
   ```

5. **Hoàn thiện Cấu trúc Tổng thể Cửa sổ (Main Window Layout)**:
   ```go
   mainLayout := container.NewBorder(
       container.NewVBox(row1Header, sep), // Top Bar + Đường phân cách
       bottomSection,                      // Footer Bar giữ nguyên 100%
       nil,
       nil,
       bodyLayout,                         // Thân app
   )

   addWindowResizeFrame(a.win, mainLayout)
   a.win.SetContent(mainLayout)
   ```

---

### 4.5. Tệp 5: Đồng bộ Theme & Ngôn ngữ Động
- Đường dẫn: [`internal/ui/app.go`](file:///g:/antigravity_auto_switch/internal/ui/app.go)

#### A. Cập nhật trong `SetTheme(themeName string)`:
Khi người dùng đổi theme (`Dark`, `Light`, `Warm`), gọi làm mới màu sắc của Sidebar:
```go
if a.sidebar != nil {
    a.sidebar.ApplyTheme(palette)
}
```

#### B. Cập nhật trong `SetLanguage(targetLocale string)`:
Khi chuyển đổi giữa VN và EN, cập nhật lại nhãn tiêu đề Top Bar và nhãn Tab:
```go
d := i18n.Current()
if a.titleLabel != nil {
    a.titleLabel.Segments = []widget.RichTextSegment{
        &widget.TextSegment{
            Text: d.AppTitle,
            Style: widget.RichTextStyle{
                SizeName:  titleTextSizeName,
                TextStyle: fyne.TextStyle{Bold: true},
            },
        },
    }
    a.titleLabel.Refresh()
}
// Cập nhật lại nhãn của Tab Antigravity
if a.sidebar != nil && a.sidebar.antigravityItem != nil {
    a.sidebar.antigravityItem.textLabel.Text = d.NavAntigravity
    a.sidebar.antigravityItem.textLabel.Refresh()
}
```

---

### 4.6. Tệp 6: Cập nhật Mutex Single-Instance
- Đường dẫn: [`cmd/switcher/main.go`](file:///g:/antigravity_auto_switch/cmd/switcher/main.go)

Cập nhật chuỗi Mutex để đồng bộ với tên mới của ứng dụng:
```go
lock, isFirst := ui.TryAcquireSingleInstance("⚡ Auto Switcher")
```

---

# PHẦN 5: CỔNG KIỂM TRA CHẤT LƯỢNG & NGHIỆM THU (SAFETY GATE 1)

Coder trước khi hoàn thành và tạo commit phải vượt qua toàn bộ các bước kiểm tra sau:

1. **Kiểm tra Unit Test i18n & UI:**
   ```powershell
   go test ./internal/i18n/... -v
   go test ./internal/ui/... -v
   ```
   *Yêu cầu:* 100% test case đạt trạng thái `PASS`.

2. **Kiểm tra Biên dịch Toàn bộ Hệ thống:**
   ```powershell
   .\scripts\build_windows.ps1
   ```
   *Yêu cầu:* Biên dịch nhị phân thành công không có warning hay lỗi link tài nguyên.

3. **Kiểm tra Trực quan Giao diện Thực tế:**
   - Chạy ứng dụng vừa build.
   - Top Bar hiển thị: `[AS] Auto Switcher`.
   - Sidebar bên trái hiển thị gọn gàng (rộng 170px) với Tab **Antigravity** (có logo thật của Antigravity, được highlight active).
   - Chưa xuất hiện bất kỳ thành phần nào của Codex.
   - Thao tác thêm tài khoản, refresh quota, bật/tắt tự động, ẩn tài khoản, đổi 3 theme (`🌙`, `☀️`, `☕`), đổi ngôn ngữ (`VN`, `EN`) hoạt động trơn tru.
   - Footer Bar hiển thị chính xác trạng thái kết nối daemon của Antigravity và nhãn phiên bản `[v1.0.3.0]`.

---

# PHẦN 6: KẾ HOẠCH BÀN GIAO CHO GIAI ĐOẠN 2

Sau khi Coder hoàn tất triển khai Giai đoạn 1 và được kiểm thử nghiệm thu (QA/Supervisor) xác nhận **PASS tuyệt đối**:
- System Architect và Chủ tịch sẽ tiếp tục phiên thảo luận chi tiết để thiết kế **GIAI ĐOẠN 2 (PHASE 2)**:
  - Tích hợp Logo thật của OpenAI Codex / Codex IDE.
  - Thêm Tab `Codex` vào Sidebar.
  - Hoàn thiện cơ chế chuyển đổi tab động (Tab Switching Mechanism) và giao diện UI-only placeholder cho Codex.
  - Xử lý làm trống/ẩn nhãn liveness tại Footer khi người dùng chuyển sang xem Tab Codex.
