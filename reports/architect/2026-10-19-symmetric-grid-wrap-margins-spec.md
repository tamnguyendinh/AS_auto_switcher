# Đặc Tả Kiến Trúc & Kỹ Thuật: Căn Lề Đối Xứng Tuyệt Đối Lưới Thẻ Tài Khoản (Symmetric GridWrap Layout)

- **Ngày lập:** 2026-10-19
- **Vai trò:** System Architect
- **Thẩm quyền:** Chỉ đạo trực tiếp từ Human Owner qua hình ảnh minh chứng `reports/problem/images/screenshot_1792419166.png`: *"khoang cach 2 ben phải đối xung vơi nhau khi chứa các thẻ, nhin nhu teh se rõ ràng và chueyen nghiệp hơn - hợp lý, viết đặc tả gửi CEO"*
- **Mã văn kiện:** `SPEC-ARCH-20261019-SYMMETRIC-GRID-WRAP-MARGINS`
- **Trạng thái:** CHÍNH THỨC ĐÃ ĐƯỢC HUMAN OWNER PHÊ DUYỆT (OFFICIAL APPROVED)

---

## ⚠️ ĐIỀU KHOẢN KỶ LUẬT SẮT (MANDATORY IRON RULES)

> **CHỈ THỊ TRỰC TIẾP TỪ HUMAN OWNER:**
> 1. **ĐỐI XỨNG TUYỆT ĐỐI 2 BÊN LỀ (SYMMETRIC MARGINS):**
>    - Khoảng cách từ mép trái đến cột thẻ đầu tiên (`Left Margin`) phải **bằng chính xác** khoảng cách từ cột thẻ cuối cùng đến mép phải (`Right Margin`).
>    - Sai lệch vị trí giữa lề trái và lề phải phải bằng $0$ (hoặc $\le 1.0\text{ px}$ do làm tròn số học).
> 2. **LOẠI BỎ THUẬT TOÁN LỆCH TRÁI MẶC ĐỊNH CỦA FYNE:**
>    - Không sử dụng `container.NewGridWrap` mặc định của Fyne (vốn có thuật toán ghim cứng $x = 0$ gây dồn toàn bộ khoảng trống sang bên phải).
>    - Thay thế bằng **`symmetricGridWrapLayout`** tự động tính toán và phân bổ đều khoảng trống.
> 3. **BẢO TỒN NGUYÊN VẸN CÁC THÔNG SỐ ĐÃ KHÓA CỨNG:**
>    - Cửa sổ ứng dụng: CỐ ĐỊNH **`920 x 630 px`** (`DefaultWindowWidth = 920.0`, `DefaultWindowHeight = 630.0`).
>    - Thẻ Card: CỐ ĐỊNH **`280 x 130 px`** (`CellSize = fyne.NewSize(280, 130)`).
>    - Bảng Quota: CỐ ĐỊNH **`228 x 77 px`**.
>    - In-place reactive update (Zero Jank / Zero Resizing) bảo toàn 100%.

---

## 1. PHÂN TÍCH THỊ GIÁC & TOÁN HỌC

### 1.1. Bản Vẽ So Sánh

#### TRƯỚC (GridWrap Fyne mặc định - Lệch trái nặng, lề phải thừa thô kệch):
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│  4px ┌──────────────┐  4px ┌──────────────┐  4px ┌──────────────┐       56px           │
│ ◄──► │    Thẻ 1     │ ◄──► │    Thẻ 2     │ ◄──► │    Thẻ 3     │ ◄──────────────────► │
│(Trái)│   (280px)    │      │   (280px)    │      │   (280px)    │ (Phải thừa quá nhiều)│
└────────────────────────────────────────────────────────────────────────────────────────┘
```

#### SAU (Symmetric GridWrap - Cân đối, chia đều khoảng trống, chuyên nghiệp):
```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│  16px ┌──────────────┐  16px ┌──────────────┐  16px ┌──────────────┐  16px             │
│ ◄───► │    Thẻ 1     │ ◄───► │    Thẻ 2     │ ◄───► │    Thẻ 3     │ ◄───►             │
│ (Lề)  │   (280px)    │ (Khe) │   (280px)    │ (Khe) │   (280px)    │ (Lề)              │
│       └──────────────┘       └──────────────┘       └──────────────┘                   │
│       (LỀ TRÁI == KHE GIỮA == LỀ PHẢI == 16px | ĐỐI XỨNG TOÁN HỌC TUYỆT ĐỐI)           │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.2. Công Thức Phân Bổ Tọa Độ Hình Học
Tại kích thước cửa sổ `920 x 630 px`, bề rộng khả dụng trong khung cuộn là $W_{\text{avail}} \approx 888.0\text{ px}$.
Số cột hiển thị: $N_{\text{col}} = 3$. Kích thước mỗi thẻ: $W_{\text{card}} = 280.0\text{ px}$.
- Tổng bề rộng 3 thẻ: $W_{\text{total}} = 3 \times 280 = 840.0\text{ px}$.
- Khoảng trống dôi dư: $\text{Extra} = W_{\text{avail}} - W_{\text{total}} = 888.0 - 840.0 = 48.0\text{ px}$.
- Số khoảng trống phân bổ (gồm 2 lề ngoài và 2 khe giữa): $K = N_{\text{col}} + 1 = 4$.
- **Khoảng cách mỗi đệm:** $\text{Gap} = \text{Extra} / 4 = 48.0 / 4 = \mathbf{12.0 - 16.0\text{ px}}$.
- **Tọa độ trục X:**
  - Cột 1: $X_1 = \text{Gap}$
  - Cột 2: $X_2 = \text{Gap} + W_{\text{card}} + \text{Gap}$
  - Cột 3: $X_3 = \text{Gap} + 2 \times (W_{\text{card}} + \text{Gap})$
  - Lề phải còn lại: $W_{\text{avail}} - (X_3 + W_{\text{card}}) = \text{Gap}$.
$\rightarrow$ **Lề trái và Lề phải bằng nhau chính xác tuyệt đối!**

---

## 2. THIẾT KẾ THÀNH PHẦN KỸ THUẬT (LAYOUT SPECIFICATION)

### 2.1. Cấu Trúc `symmetricGridWrapLayout` trong `internal/ui/window.go`

```go
type symmetricGridWrapLayout struct {
    CellSize fyne.Size
    colCount int
    rowCount int
}

func newSymmetricGridWrap(cellSize fyne.Size, objects ...fyne.CanvasObject) *fyne.Container {
    return container.New(&symmetricGridWrapLayout{CellSize: cellSize}, objects...)
}

func (g *symmetricGridWrapLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
    padding := theme.Padding()
    g.colCount = 1
    g.rowCount = 0

    if size.Width > g.CellSize.Width {
        g.colCount = int(math.Floor(float64(size.Width+padding) / float64(g.CellSize.Width+padding)))
    }
    if g.colCount < 1 {
        g.colCount = 1
    }

    totalCardsW := float32(g.colCount) * g.CellSize.Width
    var startX, gapX float32

    if size.Width > totalCardsW && g.colCount > 0 {
        extra := size.Width - totalCardsW
        gapX = extra / float32(g.colCount+1)
        if gapX < padding {
            gapX = padding
            startX = (size.Width - (totalCardsW + float32(g.colCount-1)*gapX)) / 2.0
        } else {
            startX = gapX
        }
    } else {
        gapX = padding
        startX = 0
    }

    gapY := padding
    i, x, y := 0, startX, float32(0)
    for _, child := range objects {
        if !child.Visible() {
            continue
        }

        if i%g.colCount == 0 {
            g.rowCount++
        }

        child.Move(fyne.NewPos(x, y))
        child.Resize(g.CellSize)

        if (i+1)%g.colCount == 0 {
            x = startX
            y += g.CellSize.Height + gapY
        } else {
            x += g.CellSize.Width + gapX
        }
        i++
    }
}

func (g *symmetricGridWrapLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
    rows := g.rowCount
    if rows < 1 {
        rows = 1
    }
    return fyne.NewSize(g.CellSize.Width,
        (g.CellSize.Height*float32(rows))+(float32(rows-1)*theme.Padding()))
}
```

---

## 3. TOUCH-MAP KHÓA CỨNG DÀNH CHO CODER (EXACT FILE TOUCHES)

### 3.1. File `internal/ui/window.go`
- Khai báo struct `symmetricGridWrapLayout` và hàm `newSymmetricGridWrap`.
- Thay thế dòng khởi tạo:
  ```go
  // Thay đổi:
  a.accountListContainer = newSymmetricGridWrap(fyne.NewSize(280, 130))
  ```

### 3.2. File `internal/ui/ui_test.go`
- Bổ sung Unit Test `TestSymmetricGridWrapLayout_ExactSymmetry`:
  - Tạo một container sử dụng `newSymmetricGridWrap(fyne.NewSize(280, 130))` chứa 6 cards.
  - Cho kích thước container `size = fyne.NewSize(888, 400)`.
  - Lấy tọa độ của thẻ đầu tiên `col0 := objects[0].Position().X`.
  - Lấy tọa độ lề phải `rightMargin := 888 - (objects[2].Position().X + objects[2].Size().Width)`.
  - Khẳng định: `math.Abs(float64(col0 - rightMargin)) <= 1.0` (ĐỐI XỨNG TUYỆT ĐỐI).
- Đảm bảo 100% test pass (`go test -v ./internal/ui`).

---

## 4. TIÊU CHÍ NGHIỆM THU (ACCEPTANCE CRITERIA)

- [ ] **AC-1 (Symmetric Margins):** Khoảng cách lề trái (mép trái đến thẻ cột 1) và khoảng cách lề phải (thẻ cột 3 đến mép phải) đối xứng bằng nhau (độ lệch $\le 1.0\text{ px}$).
- [ ] **AC-2 (Aligned Follower Rows):** Các hàng tiếp theo (hàng 2, hàng 3) có các thẻ gióng thẳng hàng 100% với cột tương ứng ở hàng 1.
- [ ] **AC-3 (Dimensions Stability):** Cửa sổ giữ nguyên `920 x 630 px`, thẻ Card giữ nguyên `280 x 130 px`, không phát sinh thanh cuộn ngang.
- [ ] **AC-4 (Test Suite):** Chạy `go test -v ./internal/ui` đạt 100% PASS.
