# Kế Hoạch & Tài Liệu Thiết Kế: Bộ Mẫu Chèn Thế Cờ (FEN) Sau Diễn Giải Lý Thuyết

**Mã Kế Hoạch**: `PLAN-2026-09-FEN-THEORY-TEMPLATES`  
**Trạng Thái**: `COMPLETED & DEPLOYED`  
**Vị Trí**: `docs/plans/plan_chess_theory_fen_templates.md`

---

## 1. Mục Tiêu & Bối Cảnh (Goal Description)

Trong quá trình soạn thảo giáo trình cờ vua, bài giảng huấn luyện viên, kỷ yếu giải đấu hoặc tạp chí chuyên đề cờ vua, người dùng thường xuyên có nhu cầu:
> **"Sau khi diễn giải lý thuyết hoặc kế hoạch chiến lược, chèn ngay 1 thế cờ minh họa mà chỉ cần điền chuỗi FEN và ghi chú."**

Kế hoạch này chuẩn hóa bộ mẫu (templates), tối ưu hóa cú pháp Typst, và tích hợp trực tiếp vào thanh công cụ Web Studio (`ChessToolbar.tsx`), giúp việc soạn thảo trở nên nhanh chóng, trực quan và không lo lỗi biên dịch.

---

## 2. Danh Sách Các Mẫu Thiết Kế (Ready-to-Use FEN Templates)

### 🌟 Mẫu 1: Bàn cờ Giảng dạy Căn giữa (Standard Centered Diagram)
* **Mục đích**: Bàn cờ khổ lớn rõ nét đặt giữa trang sau khi phân tích xong 1 biến thể hoặc 1 đòn chiến thuật.
* **Cú pháp Typst**:
```typst
#teaching-diagram(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <- THAY FEN VÀO ĐÂY
  title: "Ví dụ 1: Tượng Đen ghim Mã Trắng",
  turn: "b", // "w" (Trắng đi) hoặc "b" (Đen đi)
  size: 16pt,
  caption: "Tượng đen ở b4 ghim cứng Mã c3 vào Vua e1."
)
```

---

### 🌟 Mẫu 2A: Bố cục Song song (Lý thuyết bên Trái – Bàn cờ bên Phải)
* **Mục đích**: Tận dụng chiều ngang trang giấy, người đọc vừa nhìn phân tích vừa đối chiếu hình cờ bên cạnh.
* **Cú pháp Typst**:
```typst
#grid(
  columns: (1fr, auto),
  gutter: 14pt,
  align: (left + top, center + top),
  [
    *Phân tích lý thuyết & Kế hoạch:*
    - Sau khi Đen chơi `5... Bb4`, quân Mã tại c3 rơi vào thế ghim tuyệt đối.
    - *Kế hoạch của Trắng:* Nhập thành `0-0` để giải phóng ghim, hoặc chơi `a3` để chất vấn Tượng Đen.
  ],
  [
    #teaching-diagram(
      "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <- THAY FEN VÀO ĐÂY
      title: "Thế trận thực chiến",
      turn: "b",
      size: 13.5pt,
      caption: "Đen đang tạo sức ép lên c3"
    )
  ]
)
```

---

### 🌟 Mẫu 2B: Bố cục Song song (Bàn cờ bên Trái – Lý thuyết bên Phải)
* **Mục đích**: Ưu tiên người đọc quan sát thế cờ trước từ góc nhìn trực quan bên trái rồi đọc phân tích chi tiết bên phải.
* **Cú pháp Typst**:
```typst
#grid(
  columns: (auto, 1fr),
  gutter: 14pt,
  align: (center + top, left + top),
  [
    #teaching-diagram(
      "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <- THAY FEN VÀO ĐÂY
      title: "Thế trận thực chiến",
      turn: "b",
      size: 13.5pt,
      caption: "Đen đang tạo sức ép lên c3"
    )
  ],
  [
    *Phân tích lý thuyết & Kế hoạch:*
    - Sau khi Đen chơi `5... Bb4`, quân Mã tại c3 rơi vào thế ghim tuyệt đối.
    - *Kế hoạch của Trắng:* Nhập thành `0-0` để giải phóng ghim, hoặc chơi `a3` để chất vấn Tượng Đen.
  ]
)
```

---

### 🌟 Mẫu 3: Khung Khái niệm Lý thuyết kèm Bàn cờ Nổi bật (`#concept-box`)
* **Mục đích**: Đóng khung nổi bật các nguyên lý then chốt, bí quyết đại kiện tướng kèm hình cờ minh họa cô đọng.
* **Cú pháp Typst**:
```typst
#concept-box(title: "Nguyên lý Khai thác Quân bị Ghim")[
  #grid(
    columns: (1fr, auto),
    gutter: 12pt,
    align: (left + horizon, center + horizon),
    [
      *Nguyên tắc vàng:* Tăng thêm sức ép tấn công vào quân đang bị ghim!
      
      Hãy nhanh chóng huy động thêm Tốt hoặc quân nhẹ để tấn công vào quân bị bất động.
    ],
    [
      #chess-board(
        "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <- THAY FEN VÀO ĐÂY
        size: 12.5pt,
        numbers: true
      )
    ]
  )
]
```

---

### 🌟 Mẫu 4: Bàn cờ Phân tích Chiến thuật có Mũi tên (Arrows)
* **Mục đích**: Chỉ rõ đòn phối hợp, đường cơ động của quân bằng các mũi tên chiến thuật.
* **Cú pháp Typst**:
```typst
#teaching-diagram(
  "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", // <- THAY FEN VÀO ĐÂY
  title: "Đòn phối hợp tấn công điểm yếu f7",
  turn: "w",
  size: 16pt,
  arrows: ("c4-f7", "d1-h5"), // <- Hỗ trợ "c4-f7", "c4->f7", "c4f7"
  caption: "Tượng c4 nhắm vào điểm f7, Hậu sẵn sàng tiến vào h5."
)
```

---

### 🌟 Mẫu 5: Bàn cờ Góc nhìn bên Đen (Tự động lật bàn cờ)
* **Mục đích**: Giảng dạy thế trận hoặc phòng thủ từ góc nhìn người cầm quân Đen.
* **Cú pháp Typst**:
```typst
#teaching-diagram(
  "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10", // <- THAY FEN VÀO ĐÂY
  title: "Góc nhìn Đen: Phản công cánh Vua",
  turn: "b", // Khi turn: "b", bàn cờ sẽ tự động đảo chiều
  size: 16pt,
  caption: "Đen chuẩn bị dâng cao phản công cánh Vua."
)
```

---

## 3. Các Thành Phần Đã Triển Khai (Implemented Changes)

1. **[`web/src/components/ChessToolbar.tsx`](file:///D:/code/typstify/web/src/components/ChessToolbar.tsx)**:
   - Thêm `Thế cờ giảng dạy`
   - Thêm `Lý thuyết trái – Bàn cờ phải`
   - Thêm `Bàn cờ trái – Lý thuyết phải`
   - Thêm `Khái niệm kèm Bàn cờ`
   - Thêm `Thế cờ có Mũi tên (Arrows)`
2. **[`chessbook/lib/symbols.typ`](file:///D:/code/typstify/chessbook/lib/symbols.typ)**:
   - Thêm hàm `normalize-arrows` tự động chuẩn hóa chuỗi mũi tên không đồng nhất (`"c4-f7"`, `"c4->f7"`, `"c4f7"`), triệt tiêu hoàn toàn lỗi runtime khi biên dịch.
3. **[`chessbook/templates/chess_theory_fen_samples.typ`](file:///D:/code/typstify/chessbook/templates/chess_theory_fen_samples.typ)**:
   - Template mẫu tham khảo đầy đủ đã biên dịch kiểm tra thành công ra `chessbook/output/chess_theory_fen_samples.pdf`.

---

## 4. Kết Quả Kiểm Thử (Verification)
- `vitest run` trong `web/`: **46/46 unit tests passed (100%)**.
- `typst compile`: **Thành công không có cảnh báo/lỗi**.
