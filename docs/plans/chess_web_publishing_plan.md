# KẾ HOẠCH TOÀN DIỆN: NỀN TẢNG WEB TYPSTIFY CHESS PUBLISHING STUDIO

Tài liệu này xác lập kiến trúc chi tiết, thiết kế UI/UX và lộ trình triển khai trên nền tảng **Web (`web/` + `server/`)** cho hệ thống soạn thảo và xuất bản sách báo cờ vua chuyên nghiệp, bao quát đầy đủ 4 dạng ấn phẩm chuẩn FIDE & Quốc tế:

---

## 1. 4 Dạng Ấn phẩm Mục tiêu (A, B, C, D)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                 TYPSTIFY CHESS PUBLISHING SUITE (4 ĐỊNH DẠNG CHUẨN)             │
├─────────────────────────┬─────────────────────────┬─────────────────────────────┤
│ A. SÁCH BÀI TẬP (PUZZLE)│ B. BÁCH KHOA ECO        │ C. TẠP CHÍ & BÌNH LUẬN VÁN  │
│ - Lưới 2x2, 2x3, 3x4    │ - Bảng nước đi Informant│ - 2 cột magazine phong cách │
│ - Đảo ngược 180° đáp án │ - Ký hiệu thế trận NAG  │ - Game Header Card danh thủ │
│ - Tự gom trang giải cuối│ - Chú thích số khoanh   │ - Biểu đồ Eval Bar / Curve  │
│ - Mã QR quét Lichess    │ - Diagram thế cờ then ch│ - Trích dẫn Callout Quote   │
├─────────────────────────┴─────────────────────────┴─────────────────────────────┤
│ D. GIÁO TRÌNH & BÀI GIẢNG HUẤN LUYỆN (COURSEWARE & LESSON PLANS)                │
│ - Khung mục tiêu bài giảng (Objectives), Thời lượng & Trình độ                  │
│ - Hộp lý thuyết trọng tâm (Key Concept Boxes)                                   │
│ - Diagram khổ lớn với Mũi tên chiến thuật (Arrows) & Highlight ô kiểm soát      │
│ - Câu hỏi trắc nghiệm & Tình huống thực chiến trên lớp                          │
│ - Hộp ghi chú nghiệp vụ dành riêng cho Huấn luyện viên (Instructor Notes)       │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Thiết kế Module Web Platform (`web/src/`)

### 2.1. Thanh Công cụ Soạn thảo Cờ Vua (`ChessToolbar.tsx`)
Đặt ngay phía trên vùng soạn thảo mã `Editor.tsx`:
- **Figurines Palette**: Nút chèn nhanh quân cờ Trắng `♔ ♕ ♖ ♗ ♘ ♙` và Đen `♚ ♛ ♜ ♝ ♞ ♟` (tương ứng `#wK`, `#wQ`, `#wR`, `#wB`, `#wN`, `#wP`...).
- **NAG Evaluation Palette**: Nút chèn ký hiệu đánh giá `!` (hay), `?` (lỗi), `!!` (xuất sắc), `??` (đại sai lầm), `±` (Trắng ưu), `∓` (Đen ưu), `⩲` (Trắng hơi ưu), `⩱` (Đen hơi ưu), `=` (cân bằng), `∞` (phức tạp), `□` (duy nhất).
- **Template Insert Dropdown**: Nút chèn nhanh cấu trúc:
  - Chèn Bài tập (`#puzzle-card(...)`)
  - Chèn Bảng ECO (`#eco-table(...)`)
  - Chèn Thẻ ván đấu (`#game-header(...)`)
  - Chèn Bài giảng (`#lesson-header(...)`)

### 2.2. Bàn cờ Tương tác Trực quan (`ChessBoardModal.tsx`)
- Giao diện 8x8 trực quan tương tác hoàn toàn bằng chuột / cảm ứng.
- Tính năng:
  - Kéo thả / Click để di chuyển quân cờ.
  - Setup FEN tùy ý hoặc bấm "Xếp cờ mới", "Xóa trắng bàn cờ".
  - Chọn bên đi trước: Trắng / Đen.
  - Vẽ mũi tên tấn công (Click chuột phải hoặc giữ phím vẽ).
  - Chọn kiểu chèn mã: **Bài tập A5**, **Diagram cột Báo**, **Diagram Khai cuộc**, hoặc **Diagram Bài giảng**.
  - Bấm **"Chèn vào tài liệu"** $\rightarrow$ Mã Typst được tự động tạo và đưa thẳng vào con trỏ của CodeMirror Editor!

### 2.3. Công cụ Nhập ván cờ PGN Thông minh (`PgnImportModal.tsx`)
- Cho phép dán văn bản PGN hoặc upload file `.pgn`.
- Trình bóc tách (Parser) tự động nhận diện:
  - Tên hai kỳ thủ, Elo, Giải đấu, Ngày đấu, Kết quả, ECO.
  - Toàn bộ danh sách nước đi, phân nhánh biến thể phụ và bình luận dạng text.
- Tự động sinh ra khối `#game-header(...)` kèm mã Typst có định dạng chuẩn đẹp.

### 2.4. Tích hợp AI Chess Assistant (ACP Co-author)
- Cập nhật System Prompt và context của AgentChat để AI nhận biết và hỗ trợ soạn thảo cờ vua theo cú pháp của `typstify-chess-lib`.

---

## 3. Lộ trình Triển khai Code (Execution Steps)

1. **Bước 1**: Nâng cấp [`Editor.tsx`](file:///D:/code/typstify/web/src/components/Editor.tsx) bổ sung hàm `insertText(text)` vào `EditorHandle`.
2. **Bước 2**: Xây dựng component [`ChessToolbar.tsx`](file:///D:/code/typstify/web/src/components/ChessToolbar.tsx) tích hợp vào giao diện soạn thảo.
3. **Bước 3**: Xây dựng modal [`ChessBoardModal.tsx`](file:///D:/code/typstify/web/src/components/ChessBoardModal.tsx) xếp cờ trực quan sinh mã Typst.
4. **Bước 4**: Xây dựng modal [`PgnImportModal.tsx`](file:///D:/code/typstify/web/src/components/PgnImportModal.tsx) chuyển đổi ván cờ PGN sang Typst.
5. **Bước 5**: Tích hợp các công cụ vào [`Workspace.tsx`](file:///D:/code/typstify/web/src/components/Workspace.tsx) và cập nhật CSS cho giao diện chuyên nghiệp.
6. **Bước 6**: Kiểm thử Web Build (`npm run build` hoặc `vite build`) để đảm bảo hệ thống build mượt mà không có lỗi TypeScript / JSX.
