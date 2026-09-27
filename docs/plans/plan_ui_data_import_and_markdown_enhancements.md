# Kế hoạch Nâng cấp & Hoàn thiện Giao diện Nhập liệu Cờ Vua, Markdown & Xuất bản Typstify

## 1. Mục tiêu & Bối cảnh

Typstify cung cấp các công cụ mạnh mẽ để chuyển đổi Markdown và tập dữ liệu cờ vua (Excel, CSV, JSON, FEN, PGN) sang Typst thông qua package `@local/chessbook:0.1.0`. Tuy nhiên, giao diện hiện tại của 2 Modal nhập liệu chính:
1. **Chuyển đổi Markdown (.md) sang Typst (.typ) Cờ Vua** (`MarkdownImportModal.tsx`)
2. **Nhập Dữ Liệu Bài Tập (Excel / CSV / JSON / FEN)** (`DataImportModal.tsx`)

đang gặp các vấn đề nghiêm trọng về giao diện người dùng:
- **Thiếu stylesheet chuẩn**: Nhiều class CSS (`.data-import-container`, `.layout-picker-grid`, `.layout-card`, v.v.) chưa được định nghĩa trong `App.css` hoặc bị dùng lẫn các class Tailwind không tồn tại do project dùng Vanilla CSS tokens.
- **Bố cục chật chội**: Vùng nhập văn bản (textarea) quá nhỏ, bị scroll ngang dọc không thoải mái.
- **Thao tác chọn mẫu khó nhìn**: Các nút chọn layout bị dính chữ ("Sách 16x24cmSách xuất bản..."), không có phân cấp trực quan rõ ràng.
- **Thiếu dữ liệu mẫu đa dạng**: Người dùng chưa có các nút bấm nạp nhanh nhiều bộ dữ liệu mẫu thực tế (Khai cuộc, Bài tập, Giáo trình, Bình luận ván đấu).
- **Xem trước dữ liệu đơn điệu**: Bảng xem trước dữ liệu bài tập chưa có định dạng bảng chuẩn, chưa hiển thị rõ lượt đi Trắng/Đen, độ khó sao và lời giải.

## 2. Các thay đổi chi tiết

### Giai đoạn 1: Xây dựng hệ thống CSS cho Modal & Form Nhập liệu (`web/src/App.css`)
- Định nghĩa bộ CSS hoàn chỉnh cho modal dạng rộng (`.modal-wide`, `.data-import-modal`, `.chess-modal-container`):
  - Kích thước linh hoạt (92vw, max-width 960px, max-height 88vh).
  - Phân chia 2 cột responsive (bên trái là nguồn nhập/xem trước, bên phải là cấu hình xuất bản).
  - Thiết kế thẻ chọn giao diện (`.layout-picker-grid`, `.layout-card`): hỗ trợ hover, active, icon minh họa, tiêu đề đậm, mô tả phụ rõ nét, viền sáng màu thương hiệu (`--brand` & `--gold`).
  - Thanh tab chuyển đổi (`.tab-buttons`, `.tab-btn`): giao diện hiện đại dạng pill, có hiệu ứng chuyển trạng thái mượt mà.
  - Form nhập liệu (`.control-group`, `.control-label`, `.control-input`): viền chuẩn, hỗ trợ focus ring màu vàng `--gold`, typography chuẩn `Roboto`.
  - Bảng xem trước dữ liệu bài tập (`.puzzle-preview-table`): header cố định, hàng xen kẽ, huy hiệu lượt đi Trắng/Đen, sao vàng độ khó, lời giải rõ ràng.
  - Thanh thống kê (`.footer-stats`, `.badge`, `.badge-accent`): hiển thị số thế cờ, ván PGN, bảng biểu, hộp ghi chú với icon SVG tương ứng.

### Giai đoạn 2: Tối ưu & Bổ sung tính năng cho `MarkdownImportModal.tsx`
- **Bộ mẫu đa dạng (Markdown Presets)**:
  - Mẫu 1: *Giáo trình Huấn luyện & Đòn chiến thuật* (Đòn ghim, thế cờ FEN, ván cờ PGN, callout định nghĩa).
  - Mẫu 2: *Tuyển tập Khai cuộc Cờ Vua* (Biến thể Sicilian, bảng ECO, thế cờ trọng tâm).
  - Mẫu 3: *Tạp chí & Bình luận ván đấu đỉnh cao* (Ván đấu kinh điển Kasparov vs Topalov, trích dẫn, ảnh thế cờ).
  - Mẫu 4: *Phiếu bài tập chiến thuật cho học sinh* (Worksheet đòn đánh đôi, chiếu hết).
- **Khu vực soạn thảo nâng cao**: Textarea rộng rãi, font monospace, thanh công cụ xóa nhanh / tải file .md / nạp mẫu nhanh.
- **Bộ chọn khổ in trực quan**: Thẻ chọn 4 khổ: Sách 16x24cm, Worksheet A4, Tạp chí A4, Đoạn trích thuần.
- **Tùy chọn Cờ vua**: Bật/tắt tự động chuyển đổi FEN, PGN, NAG glyphs, và tùy chỉnh tiêu đề / phụ đề / tác giả / tên tệp xuất ra.

### Giai đoạn 3: Tối ưu & Bổ sung tính năng cho `DataImportModal.tsx`
- **Chuyển đổi hoàn toàn sang hệ CSS chuẩn**: Xóa bỏ các class Tailwind không hoạt động, thay bằng CSS semantic classes chuẩn `App.css`.
- **Hỗ trợ định dạng nhập liệu linh hoạt**: Tự động nhận diện CSV, TSV, JSON (mảng thế cờ Lichess/Chess.com format), hoặc danh sách FEN thuần.
- **Dữ liệu mẫu nạp nhanh (Sample Presets)**:
  - Mẫu A: CSV bài tập phối hợp chiến thuật (Đòn đôi, mở chiếu, gỡ ghim, bắt Hậu).
  - Mẫu B: JSON bài tập cờ tàn chuẩn.
  - Mẫu C: Danh sách FEN cờ thế kinh điển.
- **Bảng xem trước thông minh**: Hiển thị bảng danh sách các bài tập đã phân tích kèm huy hiệu lượt đi (⚪ Trắng / ⚫ Đen), số sao (⭐), nút ẩn/hiện lời giải.
- **Xem trước mã Typst tức thì**: Tab hoặc toggle xem trước mã Typst sinh ra với mã màu rõ ràng và nút sao chép nhanh.

### Giai đoạn 4: Đồng bộ & Kiểm tra các Modal Cờ Vua khác (`PgnImportModal.tsx`, `ChessBoardModal.tsx`)
- Đảm bảo các modal khác tuân thủ đúng theme màu (Dark/Light mode), padding, nút bấm đồng nhất.

### Giai đoạn 5: Viết tài liệu Hướng dẫn Sử dụng (`docs/CHESS_IMPORT_USER_GUIDE.md`)
- Hướng dẫn chi tiết cách:
  - Nhập Markdown có nhúng FEN/PGN để tạo sách bài giảng.
  - Nhập Excel / CSV / JSON / FEN để tự động chia trang và xuất sách bài tập.
  - Cấu hình in ấn khổ Sách 16x24cm vs A4 Worksheet.
  - Xuất bản tài liệu với đáp án lật ngược chân trang hoặc phụ lục cuối sách.

### Giai đoạn 6: Kiểm thử, Build, Git Commit, Push & Deploy lên VPS Dokploy
1. Chạy `npm run test` và `npm run build` trong `web/` để đảm bảo không có lỗi biên dịch TypeScript.
2. Kiểm tra backend Go `go test ./...`.
3. Commit toàn bộ thay đổi với thông điệp rõ ràng và Push lên GitHub `covuaduongsinh/typstify`.
4. Kích hoạt Dokploy webhook / API hoặc deploy qua SSH trên VPS `217.15.160.118` (domain `typst.dsc.edu.vn`).
5. Xác minh hệ thống hoạt động ổn định trên môi trường production.

---

## 3. Kế hoạch xác minh (Verification Plan)

### Kiểm thử tự động
- `npm run test` trong `web/` (kiểm tra các parser `dataImport.test.ts`, `markdownToTypst.test.ts`, `pgn.test.ts`).
- `npm run build` trong `web/` (biên dịch Vite + TypeScript sang bundle tĩnh).
- `go test ./...` tại thư mục gốc.

### Kiểm thử thủ công trên giao diện
- Mở Modal Markdown Import: kiểm tra giao diện 2 cột, thử chuyển đổi tab, thử tải file `.md`, chọn từng mẫu tài liệu, kiểm tra mã Typst sinh ra.
- Mở Modal Data Import: thử nạp các preset CSV, JSON, FEN; kiểm tra bảng xem trước thế cờ, kiểm tra nút chọn khổ giấy A4 vs 16x24cm.
- Đổi Dark/Light mode trên header để xác nhận giao diện modal hiển thị hoàn hảo ở cả hai chế độ.
