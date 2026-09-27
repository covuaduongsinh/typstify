# Hướng Dẫn Tính Năng: Live Instant Preview & Đồng Bộ Cuộn Theo Con Trỏ

## 1. Giới thiệu tổng quan
Phiên bản mới của **Typstify** mang đến trải nghiệm soạn thảo thời gian thực vượt trội với hai tính năng cốt lõi:
1. **Live Instant Preview (Xem trước tức thì):** Tự động biên dịch ngầm và cập nhật bản xem trước ngay khi bạn gõ phím hoặc chèn bài tập, không cần phải bấm `Ctrl+S` (Lưu).
2. **Cursor & Scroll Sync (Đồng bộ cuộn theo điểm chỉnh sửa):** Khung xem trước tự động cuộn mượt mà đến đúng trang và vị trí của đoạn văn bản hoặc bài tập mà con trỏ soạn thảo đang đứng.

---

## 2. Các Tính Năng Chi Tiết

### ⚡ 2.1. Cập nhật Tức thì (Live Instant Preview)
- **Tự động biên dịch ngầm:** Khi bạn gõ văn bản, thêm công thức, chèn bàn cờ, hệ thống áp dụng cơ chế *debounce thông minh (300ms)* để biên dịch và cập nhật bản xem trước mượt mà.
- **Chèn mẫu cờ vua 1-chạm:** Khi bạn bấm chèn bài tập từ *Chess Toolbar*, *Bàn cờ*, *PGN* hay *Markdown Modal*, bản xem trước cập nhật ngay lập tức (0ms).
- **Chế độ Trang In SVG (Paper View):**
  - Hiển thị từng trang tài liệu như trang giấy in thực tế với hiệu ứng đổ bóng thanh lịch.
  - **Không chớp nháy (Zero-flicker):** Khắc phục triệt để hiện tượng trắng màn hình của các plugin PDF nhúng truyền thống.
  - Hiển thị nhãn số trang rõ ràng: `Trang 1 / 3`, `Trang 2 / 3`...

### 🎯 2.2. Đồng bộ Cuộn theo Điểm Chỉnh sửa (Cursor Sync)
- **Tự động theo dõi vị trí soạn thảo:** Khi bạn di chuyển con trỏ chuột, gõ phím hoặc chuyển sang trang tiếp theo trong trình soạn thảo, khung Preview sẽ tự động cuộn mượt mà (**Smooth Scroll**) đưa đoạn nội dung đang sửa vào vùng nhìn thuận mắt nhất.
- **Nút Bật / Tắt Đồng bộ cuộn (`🔗 Đồng bộ cuộn`):**
  - **Bật (Mặc định):** Giữ khung xem trước luôn khóa theo vị trí con trỏ.
  - **Tắt (Nút chuyển thành `unlink`):** Cho phép bạn tự do cuộn xem toàn bộ tài liệu độc lập mà không bị nhảy lại theo con trỏ soạn thảo.
- **Tự động nhường quyền khi lăn chuột:** Khi bạn chủ động lăn chuột trong khung Preview, hệ thống sẽ tạm dừng đồng bộ trong 1.5 giây để bạn đọc nội dung thoải mái.

### 🔍 2.3. Công cụ Thu Phóng & Chế độ Xem (Zoom & Mode Controls)
- **Bộ nút Zoom:**
  - `Thu nhỏ (-)` / `Phóng to (+)`: Thay đổi tỷ lệ hiển thị từ `50%` đến `200%`.
  - `100%`: Trở về kích thước chuẩn một chạm.
  - `Vừa khung (Fit Width)`: Tự động co giãn trang vừa khít chiều rộng của khung Preview.
- **Chuyển đổi Chế độ Xem:**
  - `Trang in (SVG)`: Chế độ mặc định, siêu mượt, hỗ trợ cập nhật tức thì và cuộn theo con trỏ.
  - `PDF`: Chế độ xem qua trình đọc PDF gốc của trình duyệt.
  - `Mở PDF trong tab mới`: Nút mở file PDF riêng biệt để in ấn hoặc tải về máy.

### 🛡️ 2.4. Báo lỗi Thông minh Không Chặn Tương tác
- Khi bạn đang gõ dở một câu lệnh Typst chưa hoàn chỉnh (ví dụ gõ `#chess-board(` chưa đóng ngoặc):
  - Khung Preview **vẫn giữ nguyên trang đã render thành công trước đó** (không bị mất trắng nội dung).
  - Một banner cảnh báo lỗi cú pháp nhỏ xuất hiện ở góc dưới để bạn dễ dàng nhận biết và sửa lỗi.

---

## 3. Bảng Phím tắt & Thao tác Tiện lợi

| Thao tác | Phím tắt / Vị trí | Công dụng |
| :--- | :--- | :--- |
| **Soạn thảo trực tiếp** | Bàn phím | Bản xem trước cập nhật sau 300ms dừng gõ |
| **Chèn mẫu cờ vua** | Chess Toolbar | Chèn bàn cờ, bài tập, nước cờ, cập nhật preview ngay |
| **Lưu vĩnh viễn** | `Ctrl + S` / `Cmd + S` | Lưu file vào hệ thống và đồng bộ preview |
| **Bật/Tắt cuộn đồng bộ** | Nút `🔗 Đồng bộ cuộn` | Chuyển đổi giữa cuộn theo con trỏ hoặc cuộn tự do |
| **Vừa chiều rộng** | Nút `Maximize` | Tự căn chỉnh trang vừa khít khung nhìn |
| **Mở bản in PDF** | Nút `External link` | Mở trực tiếp PDF trong tab mới |
