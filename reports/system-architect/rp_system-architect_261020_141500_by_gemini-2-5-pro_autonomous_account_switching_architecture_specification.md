# Báo Cáo Kiến Trúc Hệ Thống: Đặc Tả Toàn Diện Phương Án Tự Động Hóa Chuyển Đổi Tài Khoản (Autonomous Account Switching Architecture Specification)

**Mã tài liệu:** `rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification`  
**Ngày ban hành:** 2026-10-20  
**Làn thực hiện:** System Architect (Kiến trúc sư Hệ thống)  
**Mô hình thực hiện:** `gemini-2-5-pro`  
**Chỉ đạo tối cao:** Human Owner  
**Trạng thái:** HOÀN TẤT & SẴN SÀNG CHUYỂN GIAO (APPROVED ARCHITECTURAL AUTHORITY)  
**Điểm đối soát kế thừa (Traceability Anchors):**
- Mã nguồn hiện tại: `internal/manager/account_manager.go`, `internal/telemetry/`, `internal/reloader/`
- Quyết định kiến trúc ngày 18/10: `reports/architect/2026-10-18-coder-implementation-spec-4-step-hotswap.md`
- Kế hoạch cũ cần phân rã: `docs/plans/2026-10-14-antigravity-multi-account-switcher/` (Phase P7 cũ)

---

## 1. TỔNG QUAN & LÝ DO THAY THẾ TOÀN DIỆN PHASE 7 CŨ

### 1.1 Mục tiêu hệ thống
Thiết lập cơ chế điều phối tự động (**Autonomous Account Switching Controller**) trong Antigravity Multi-Account Switcher (Go Engine) nhằm:
1. Tự động luân chuyển tài khoản khi tài khoản hiện tại chạm ngưỡng cạn kiệt Quota hoặc gặp sự cố chặn hạn ngạch từ Google.
2. Bảo toàn 100% dòng suy nghĩ và công việc đang làm dở của lập trình viên: tự động đánh thức và tiếp tục các hội thoại trong vòng 2 giờ gần nhất mà không cần người dùng thao tác bấm nút thủ công.
3. Tuân thủ tuyệt đối triết lý thiết kế **KISS (Đơn giản là sức mạnh)** và **Burn-down Strategy (Dùng vét tài nguyên cũ trước, giữ tài nguyên mới sau)**.

### 1.2 Quyết định phân tách và bãi bỏ Phase 7 cũ (Deprecation Decision)
- **Quyết định:** **Xóa bỏ hoàn toàn Phase 7** trong bộ kế hoạch cũ (`docs/plans/2026-10-14-antigravity-multi-account-switcher/`). Tính năng Automation được tách thành **Một bộ kế hoạch độc lập mới** (`docs/plans/2026-10-20-autonomous-account-switching/`).
- **Lý do kỹ thuật:** Phase 7 cũ (lập ngày 14–15/10/2026) dựa trên các giả định đã bị thực nghiệm bác bỏ:
  1. *Giả định hoán đổi $< 1.0\text{s}$ bằng RPC ReAuth:* Thực tế Antigravity v2.24.0 không hỗ trợ ReAuth nóng trong RAM. Chuyển đổi chuẩn bắt buộc phải `taskkill /F /IM language_server.exe` và **chờ đủ 8 giây** để `monitorLsCrashInternal` tái sinh tiến trình an toàn.
  2. *Giả định bị động chỉ đọc file log:* Không tận dụng được gói `internal/telemetry` (Phase 6A) vốn đã có API gọi Google trực tiếp mỗi 60 giây và theo dõi phần trăm quota tường minh.
  3. *Giả định Round-Robin mù:* Dễ chuyển nhầm vào tài khoản đã cạn hạn mức, thiếu cơ chế liên kết với model đang chạy của phiên làm việc.
  4. *Xây dựng daemon Systray riêng:* Gây xung đột với khay hệ thống đã có sẵn trong Fyne GUI.

---

## 2. CÁC PHÁT HIỆN THỰC NGHIỆM ĐỘT PHÁ TRÊN ANTIGRAVITY v2.24.0

Trong quá trình đối soát trực tiếp trên môi trường Antigravity đang hoạt động, kiến trúc sư đã xác lập 4 phát hiện thực nghiệm mang tính cốt lõi:

1. **Cơ chế Retry nguyên bản của Antigravity (Nút "Try again"):**
   - Nút *Try again* trên giao diện IDE thực chất gửi một Connect-RPC `SendUserCascadeMessage` với **danh sách nội dung rỗng (`empty content list`)** tới đúng conversation ID đó.
   - Hành động này yêu cầu Language Server thử lại câu lệnh mà **KHÔNG CẦN restart tiến trình**. Đây là cơ sở để thực hiện khôi phục tối đa 3 lần siêu tốc ($< 500\text{ms}$) mà không làm gián đoạn IDE.
2. **Cơ chế giám sát trạng thái lượt chạy (`StreamAgentStateUpdates`):**
   - Stream này trả về trạng thái lượt chạy và thông tin lỗi có cấu trúc (`429`, `RESOURCE_EXHAUSTED`, `RATE_LIMIT_EXCEEDED`, `QUOTA_EXHAUSTED`) kèm metadata `retryAt`, `attempt`. Automation sử dụng stream này để nhận diện chính xác sự cố.
3. **Ánh xạ Model Placeholder (`GetCascadeModelConfigData`):**
   - Antigravity lưu mã placeholder nội bộ (ví dụ: `MODEL_PLACEHOLDER_M318`).
   - Hàm `DiscoverActiveAgentModel` kết hợp `GetCascadeModelConfigData` ánh xạ chính xác mã này thành tên model thực (**Gemini 3.8 Flash High**).
4. **Cờ vô hiệu hóa bucket 5h (`disabled: true`):**
   - Khi tài khoản hết hạn ngạch tuần ($0\%$), Google API vẫn trả về bucket 5h nhưng đặt cờ `disabled: true` (kèm thông điệp giải thích giới hạn 5h bị vô hiệu hóa vì tuần đã hết).

---

## 3. MÔ HÌNH DỮ LIỆU & LIÊN KẾT QUOTA THEO ACTIVE MODEL

### 3.1 Quy tắc liên kết Active Model (Active Model Binding)
Mọi quyết định kiểm tra ngưỡng và xếp hạng ứng viên đều dựa trên **nhóm model của phiên làm việc hiện tại**:

| Model đang hoạt động | Nhóm Quota được sử dụng để kiểm tra & xếp hạng |
|---|---|
| Dòng Gemini (`gemini-2.5`, `gemini-3.8-flash`, ...) | **Nhóm Gemini** (5h Gemini & Tuần Gemini) |
| Dòng Claude (`claude-4-6`, `opus`, `sonnet`, ...) | **Nhóm Claude/GPT** (5h Claude & Tuần Claude) |
| Dòng GPT-OSS | **Nhóm Claude/GPT** (theo phân loại của Antigravity) |

> 🔒 **Bất biến kiến trúc:** Phiên đang chạy Gemini thì việc Claude hết hạn ngạch tuần **tuyệt đối không phải là lý do để chuyển tài khoản**. Khi người dùng đổi model trong phiên, hệ thống lập tức tính toán lại theo nhóm quota mới.

### 3.2 Tam phân thái Quota (Xóa bỏ việc đoán mò qua `--%`)
Dữ liệu quota từ Google API `RetrieveUserQuotaSummary` được phân định chuẩn thành 3 trạng thái:
1. **`APPLICABLE_VALID` (Có áp dụng, dữ liệu hợp lệ):**
   - Có bucket 5h đang mở, fraction hợp lệ ($0.0 \le \text{fraction} \le 1.0$).
   - Áp dụng đầy đủ cả điều kiện $5\text{h} \le 2\%$ và $\text{Tuần} = 0\%$.
2. **`NOT_APPLICABLE` (Xác nhận không áp dụng chu kỳ 5h):**
   - Phản hồi quota thành công nhưng nhóm chỉ có bucket tuần, HOẶC bucket 5h có cờ `disabled: true`.
   - Bỏ qua điều kiện 5h. Chỉ theo dõi Quota Tuần và **dùng đến $0\%$ Tuần** mới kích hoạt chuyển.
3. **`UNKNOWN_UNCERTAIN` (Dữ liệu bị khuyết / Chưa xác định):**
   - Do mất mạng, timeout, hoặc cache đang làm mới.
   - **Tuyệt đối không suy diễn** thành hết hạn hay không giới hạn. Giữ nguyên tài khoản, yêu cầu quét lại, không kích hoạt chuyển.

---

## 4. CƠ CHẾ KÍCH HOẠT KÉP (DUAL-TRIGGER ARCHITECTURE)

Hệ thống vận hành đồng thời 2 van kích hoạt nhưng sử dụng chung một đường ống thực thi:

```text
                                [GIÁM SÁT LIÊN TỤC]
                                         │
                     ┌───────────────────┴───────────────────┐
                     ▼                                       ▼
       [VAN 1: CHỦ ĐỘNG (Proactive)]           [VAN 2: BỊ ĐỘNG (Reactive)]
       • Quét telemetry định kỳ 60s            • Bắt lỗi thực thi có cấu trúc
       • 5h <= 2% HOẶC Tuần = 0%               • Thử khôi phục tối đa 3 lần
       • Chuyển êm đềm trước khi lỗi           • Thất bại 3 lần -> Chuyển khẩn cấp
                     │                                       │
                     └───────────────────┬───────────────────┘
                                         ▼
                     [SINGLE-FLIGHT CONCURRENCY GATE]
                     (Hợp nhất thành 1 đợt chuyển duy nhất)
```

### Ma trận điều kiện kích hoạt:
| Loại tài khoản | Tình trạng thực tế | Hành động của Automation | Cơ chế Thử lại (Retry) |
|---|---|---|---|
| **Có 5h và Tuần** | **$5\text{h} \le 2\%$** | **Chủ động chuyển ngay** | Không retry (chuyển trực tiếp trước khi cạn). |
| **Có 5h và Tuần** | **$\text{Tuần} = 0\%$** *(dù 5h còn nhiều)* | **Chủ động chuyển ngay** | Không retry (vì tuần = 0% đã chặn cứng). |
| **Chỉ có Tuần** | $\text{Tuần} > 0\%$ | Tiếp tục sử dụng | Không chuyển. |
| **Chỉ có Tuần** | **$\text{Tuần} = 0\%$** | **Chủ động chuyển ngay** | Không retry (dùng đến giọt cuối cùng 0%). |
| **Bất kỳ loại nào** | Gặp lỗi `429` / `RESOURCE_EXHAUSTED` / `RATE_LIMIT_EXCEEDED` / `QUOTA_EXHAUSTED` | **Chuyển phòng vệ (Reactive)** | **Thử khôi phục tối đa 3 lần** qua cơ chế Native Retry. Nếu cả 3 lần đều hỏng $\rightarrow$ Kích hoạt chuyển. |
| **Chưa xác định** | Lỗi mạng / API trả về `--%` | Không chuyển, quét làm mới | Không kích hoạt chuyển. |

> 🔒 **Single-Flight Rule:** Nếu cả lỗi 429 và ngưỡng $\le 2\%$ xuất hiện đồng thời, chúng được hợp nhất thành **1 đợt chuyển duy nhất**.

---

## 5. QUY TRÌNH KHÔI PHỤC TỐI ĐA 3 LẦN (NATIVE IN-MEMORY RETRY)

Ở nhánh bị động khi gặp lỗi giới hạn:
1. **Kiểm tra trạng thái tự phục hồi:** Nếu Antigravity đang tự retry nội bộ (`retryAt`), automation tôn trọng và chờ đợi, không gửi lệnh đè.
2. **Kích hoạt Retry văn minh:** Automation gửi Connect-RPC `SendUserCascadeMessage` với **empty content list** tới conversation ID đang gặp lỗi.
3. **Theo dõi trạng thái lượt chạy:** Giám sát qua `StreamAgentStateUpdates`. Nếu thành công $\rightarrow$ Dừng khôi phục, tiếp tục làm việc bình thường.
4. **Giới hạn số lần:** Thử lại tối đa **3 lần** (mỗi lần cách nhau 2–3s). Nếu sau 3 lần vẫn lỗi $\rightarrow$ Xác nhận tài khoản đã cạn kiệt thật sự $\rightarrow$ Kích hoạt chuyển sang tài khoản khác.
5. **Hợp nhất tín hiệu:** Nếu trong lúc đang retry mà telemetry xác nhận $5\text{h} \le 2\%$ hoặc $\text{Tuần} = 0\%$, dừng retry và chuyển ngay lập tức.

---

## 6. BỘ LỌC ỨNG VIÊN & THUẬT TOÁN SẮP XẾP ƯU TIÊN (BURN-DOWN SORTER)

### 6.1 Bộ lọc sơ loại ứng viên (Pre-filtering)
Tài khoản đích bắt buộc phải vượt qua 5 tiêu chí loại trừ:
1. Khác tài khoản đang sử dụng.
2. File token trong `accounts_store/` hợp lệ, nạp được vào WinCred và có refresh token sống.
3. Chưa bị đánh dấu thất bại trong đợt xử lý hiện tại.
4. **Hạn mức sử dụng của nhóm model đang hoạt động:**
   - Với tài khoản có 5h: Bắt buộc **$5\text{h} > 2\%$** VÀ **$\text{Tuần} > 0\%$**.
   - Với tài khoản chỉ có tuần: Bắt buộc **$\text{Tuần} > 0\%$**.
5. Dữ liệu quota đủ tin cậy (kiểm tra tươi bằng `PollAccount`).

### 6.2 Thuật toán sắp xếp ưu tiên (Priority Sorter)
Áp dụng triết lý "Dùng vét đồ cũ trước, giữ đồ mới sau":

```text
[ỨNG VIÊN ĐỦ ĐIỀU KIỆN]
           │
           ▼
[ƯU TIÊN 1: QUOTA TUẦN CÒN LẠI THẤP NHẤT TRƯỚC]
(Tài khoản nào tuần sắp cạn thì mang ra xài cho hết: 3% -> 10% -> 50%...)
           │
           ▼ (Nếu Quota Tuần bằng nhau)
[ƯU TIÊN 2: ƯU TIÊN TÀI KHOẢN CÓ ÁP DỤNG QUOTA 5H]
(Tài khoản có 5h sẽ tự hồi sau 5 tiếng, giữ tài khoản chỉ có tuần làm dự trữ)
           │
           ▼ (Nếu cả hai đều có Quota 5h)
[ƯU TIÊN 3: QUOTA 5H CAO NHẤT TRƯỚC]
(Trong chu kỳ 5h đó, tài khoản nào còn nhiều 5h hơn sẽ dùng được lâu nhất)
           │
           ▼ (Nếu vẫn bằng nhau hoặc đều chỉ có Tuần)
[ƯU TIÊN 4: GIỮ NGUYÊN THỨ TỰ DANH BẠ CỐ ĐỊNH]
```

### Bảng ví dụ phân xử thực tế (Model: Gemini):
| Tài khoản | Quota Tuần | Quota 5h | Kết quả phân loại & Xếp hạng | Lý do giải thích |
|---|---:|---:|:---:|---|
| **A** | **3%** | **80%** | **Ưu tiên 1** | Tuần thấp nhất (3%), có 5h và 5h còn cao (80%). |
| **B** | **3%** | **40%** | **Ưu tiên 2** | Tuần bằng A (3%), nhưng 5h thấp hơn A (40% < 80%). |
| **C** | **3%** | *Không áp dụng* | **Ưu tiên 3** | Tuần bằng A và B (3%), nhưng không có 5h nên xếp sau tài khoản có 5h. |
| **D** | **20%** | **90%** | **Ưu tiên 4** | Tuần còn nhiều (20%), xếp sau các tài khoản tuần thấp. |
| **E** | **1%** | **1%** | ❌ **Loại bỏ** | Bị loại từ vòng lọc vì $5\text{h} \le 2\%$. |
| **F** | **0%** | **100%** | ❌ **Loại bỏ** | Bị loại từ vòng lọc vì $\text{Tuần} = 0\%$. |

---

## 7. QUY TRÌNH THỰC THI CHUYỂN ĐỔI & TIẾP TỤC PHIÊN 6 BƯỚC

```text
 ┌────────────────────────────────────────────────────────────────────────┐
 │ BƯỚC 1: RÀ SOÁT & XẾP HẠNG ỨNG VIÊN                                     │
 │ Lấy active model -> Lọc tài khoản theo quota của model đó -> Sắp xếp.  │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ BƯỚC 2: THEO DÕI SONG SONG HAI TÍN HIỆU                                 │
 │ Nhánh chủ động: Quota 5h <= 2% hoặc Tuần = 0% -> Kích hoạt ngay.       │
 │ Nhánh bị động: Lỗi giới hạn -> Retry tối đa 3 lần qua SendUserCascade. │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ BƯỚC 3: CHỌN & KIỂM TRA LẠI ỨNG VIÊN                                   │
 │ Lấy ứng viên #1 -> Kiểm tra nhanh PollAccount.                         │
 │ Nếu ứng viên #1 lỗi -> Bỏ qua, lấy ứng viên #2 trong danh sách.        │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ BƯỚC 4: THỰC THI CHUYỂN ĐỔI (Dùng SwitchAccount sẵn có)                │
 │ Snapshot SQLite 2h -> Ghi WinCred -> Đồng bộ ToS -> taskkill server    │
 │ -> CHỜ ĐỦ 8 GIÂY -> Khám phá lại Language Server & Xác nhận email mới. │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ BƯỚC 5: TỰ ĐỘNG TIẾP TỤC PHIÊN (Continuation 2 Giờ)                    │
 │ Đã nằm trọn vẹn trong SwitchAccount: Tự động gửi prompt tiếp tục       │
 │ tuần tự (cách nhau 3s) vào các thread trong 2h qua Connect-RPC.        │
 │ Controller tuyệt đối KHÔNG gọi thêm lượt thứ hai bên ngoài.            │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     │
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ BƯỚC 6: CẬP NHẬT TRẠNG THÁI & PHÒNG HỘ                                  │
 │ Cập nhật trạng thái trên Fyne GUI / Tray.                              │
 │ Nếu danh sách ứng viên = 0 -> Chuyển sang trạng thái chờ an toàn.      │
 └────────────────────────────────────────────────────────────────────────┘
```

---

## 8. BẢNG MA TRẬN XỬ LÝ 14 TÌNH HUỐNG BIÊN & AN TOÀN

| STT | Tình huống phát sinh | Hành vi xử lý chuẩn xác của Automation |
|---|---|---|
| **1** | Gemini $5\text{h} \le 2\%$, phiên đang dùng model Gemini. | Chủ động kích hoạt chuyển tài khoản ngay lập tức. |
| **2** | Claude hết quota tuần ($0\%$), nhưng phiên đang dùng Gemini và Gemini vẫn còn quota. | **Không chuyển**, tiếp tục sử dụng bình thường vì model đang chạy không bị ảnh hưởng. |
| **3** | Tài khoản chỉ áp dụng quota tuần, tuần còn $1\%$. | Tiếp tục sử dụng bình thường (vét đến giọt cuối cùng). |
| **4** | Tài khoản chỉ áp dụng quota tuần, tuần chạm $0\%$. | Chủ động kích hoạt chuyển tài khoản ngay lập tức. |
| **5** | Bucket 5h trả về thuộc tính `disabled: true`. | Bỏ qua điều kiện 5h trong lần đánh giá đó, chỉ xét quota tuần. |
| **6** | Khôi phục thành công ở lần thử thứ 1 hoặc thứ 2. | Dừng khôi phục ngay lập tức, tiếp tục phiên làm việc bình thường. |
| **7** | Cả 3 lần khôi phục đều thất bại. | Kích hoạt quy trình chuyển sang tài khoản khác. |
| **8** | Cả tín hiệu $\le 2\%$ và lỗi 429 cùng kích hoạt đồng thời. | **Single-flight:** Hợp nhất thành 1 đợt chuyển duy nhất, không tạo 2 lệnh chạy đè nhau. |
| **9** | Người dùng bấm nút chuyển tay trên GUI đúng lúc automation đang chuyển. | GUI báo trạng thái *"Đang xử lý chuyển đổi"*, khóa Mutex, không chạy chồng chéo. |
| **10**| Người dùng đổi model trong IDE ngay trước đợt chuyển. | Hệ thống tự động tính toán lại quyết định chuyển theo nhóm quota của model mới. |
| **11**| Ứng viên đầu bảng kiểm tra nhanh bị lỗi token/mạng. | Bỏ qua ứng viên đó trong đợt này, lập tức xét ứng viên kế tiếp trong danh sách. |
| **12**| Hàm `SwitchAccount` trả về kết quả thất bại. | Hiển thị thông báo lỗi trên UI; dừng đợt chuyển, không xoay vòng tài khoản mù quáng. |
| **13**| **Không còn tài khoản nào khả dụng trong kho.** | Hiển thị: **"Chưa có tài khoản phù hợp để chuyển"**. Ngừng chuyển, giữ nguyên phiên và chờ telemetry phục hồi. |
| **14**| Dòng log lỗi của phiên cũ dội về trễ sau khi đã chuyển sang tài khoản mới xong. | Kiểm tra đối soát PID/Email, bỏ qua các dòng lỗi thuộc về phiên cũ đã đóng. |

---

## 9. KIẾN TRÚC MÃ NGUỒN GO & GIAO DIỆN UI

Cấu trúc package Go chuẩn mực tại `internal/automation/`:
```text
internal/
├── automation/
│   ├── controller.go      # Vòng đời Automation, công tắc Bật/Tắt, khóa đơn luồng atomic.Bool
│   ├── model_quota.go     # Đọc active model, ánh xạ nhóm quota, phân định tam phân thái & disabled:true
│   ├── retry.go           # Bắt StreamAgentStateUpdates, điều khiển 3 lần Native Retry qua SendUserCascadeMsg
│   └── sorter.go          # Bộ lọc ứng viên & thuật toán Burn-down Sorter
├── manager/               # Giữ nguyên AccountManager.SwitchAccount()
├── reloader/              # Giữ nguyên active_threads.go và continuation.go
├── telemetry/             # Giữ nguyên client, cache và loop 60s
└── ui/                    # Bổ sung Toggle bật/tắt và nhãn hiển thị trạng thái Automation
```

---

## 10. CHỈ ĐẠO BÀN GIAO CHO LÀN LẬP KẾ HOẠCH (PLANNER HANDOFF DIRECTIVE)

1. **Dọn dẹp tài liệu cũ:** 
   - Xóa bỏ Phase 7 trong `docs/plans/2026-10-14-antigravity-multi-account-switcher/` (cập nhật lại plan, actual-status, evidence, benchmark).
2. **Khởi tạo Bộ Kế Hoạch Độc Lập:**
   - Tạo thư mục kế hoạch mới: `docs/plans/2026-10-20-autonomous-account-switching/`.
   - Lập bộ 5 file chuẩn: `*-plan.md`, `*-actual-status.md`, `*-evidence.md`, `*-benchmark.md`, `*-report.md`.
3. **Phân rã thành 3 Slices thi công:**
   - **Slice 1 (P1): Model Quota Resolver & Burn-down Sorter** (`internal/automation/model_quota.go`, `sorter.go`).
   - **Slice 2 (P2): Dual-Trigger Engine & 3-Retry Native Controller** (`internal/automation/retry.go`, `controller.go`).
   - **Slice 3 (P3): Automation Coordinator & GUI Toggle Integration** (`internal/ui/`, E2E test harness).

---
**Báo cáo được niêm phong bởi:** System Architect  
**Lưu trữ vĩnh viễn tại:** `reports/system-architect/rp_system-architect_261020_141500_by_gemini-2-5-pro_autonomous_account_switching_architecture_specification.md`