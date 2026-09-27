// ============================================================================
// TEMPLATE: BỘ MẪU CHÈN THẾ CỜ (FEN) SAU KHI DIỄN GIẢI LÝ THUYẾT
// Dành cho: Bài giảng, Giáo trình huấn luyện, Chuyên khảo chiến thuật & Khai cuộc.
// Bạn chỉ việc sao chép mẫu tương ứng và thay thế chuỗi FEN.
// ============================================================================

#import "../lib/lib.typ": *

#set page(
  paper: "a4",
  margin: (x: 1.8cm, top: 1.8cm, bottom: 1.8cm),
  header: context [
    #grid(
      columns: (1fr, auto, 1fr),
      align: (left + horizon, center + horizon, right + horizon),
      [#text(8pt, weight: "bold", fill: rgb("#2b6cb0"))[TYPSTIFY CHESS ENGINE - BỘ MẪU CHÈN FEN LÝ THUYẾT]],
      [#text(7.5pt, fill: rgb("#a0aec0"))[| GIÁO TRÌNH |]],
      [#text(8pt, weight: "bold", fill: rgb("#2d3748"))[Trang #counter(page).display()]]
    )
    #v(2pt)
    #line(length: 100%, stroke: 0.5pt + rgb("#cbd5e0"))
  ]
)

#set text(font: font-sans, size: 9pt, lang: "vi")
#set par(justify: true, leading: 0.55em)

#lesson-header(
  lesson-num: 1,
  title: "CÁC MẪU CHÈN THẾ CỜ FEN VÀO BÀI GIẢNG",
  level: "Dành cho Tác giả & Huấn luyện viên",
  duration: "Tra cứu nhanh",
  objective: "Tổng hợp các mẫu chèn bàn cờ chuẩn mực, tiện lợi sau khi giải thích lý thuyết cờ vua."
)

#v(8pt)

// ----------------------------------------------------------------------------
// MẪU 1: BÀN CỜ GIẢNG DẠY CĂN GIỮA (CHUẨN BÀI GIẢNG)
// ----------------------------------------------------------------------------
#text(10pt, weight: "bold", fill: rgb("#2b6cb0"))[Mẫu 1: Bàn cờ Giảng dạy Căn giữa (Teaching Diagram)]
#v(2pt)
Sau khi giảng giải một phần lý thuyết hoặc phân tích một biến thể, bạn muốn chèn một bàn cờ khổ lớn rõ ràng ở giữa trang để học viên quan sát tổng thể.

#teaching-diagram(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <--- THAY FEN Ở ĐÂY
  title: "Ví dụ 1: Tượng Đen ghim Mã Trắng vào Vua",
  turn: "b", // "w" (Trắng đi) hoặc "b" (Đen đi)
  size: 16pt,
  caption: "Tượng đen ở b4 ghim cứng Mã c3 vào Vua e1. Trắng không thể di chuyển Mã c3."
)

#v(10pt)

// ----------------------------------------------------------------------------
// MẪU 2A: BỐ CỤC SONG SONG (LÝ THUYẾT TRÁI - BÀN CỜ PHẢI)
// ----------------------------------------------------------------------------
#text(10pt, weight: "bold", fill: rgb("#2b6cb0"))[Mẫu 2A: Song song (Lý thuyết bên Trái – Bàn cờ bên Phải)]
#v(2pt)

#grid(
  columns: (1fr, auto),
  gutter: 14pt,
  align: (left + top, center + top),
  [
    *Phân tích Kế hoạch & Ý đồ chiến thuật:*
    - Sau khi Đen phát triển Tượng lên b4, quân Mã tại c3 rơi vào thế ghim tuyệt đối.
    - *Kế hoạch của Trắng:*
      + Chơi `0-0` (Nhập thành) để đưa Vua thoát khỏi đường chiếu của Tượng.
      + Hoặc chơi `a3` để lập tức chất vấn quân Tượng đối phương.
    - *Cảnh báo:* Tránh vội vàng di chuyển Hậu khi chưa tháo ghim an toàn cho Vua!
  ],
  [
    #teaching-diagram(
      "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <--- THAY FEN Ở ĐÂY
      title: "Thế trận thực chiến",
      turn: "b",
      size: 13.5pt,
      caption: "Đen đang tạo sức ép lên c3"
    )
  ]
)

#v(10pt)

// ----------------------------------------------------------------------------
// MẪU 2B: BỐ CỤC SONG SONG (BÀN CỜ TRÁI - LÝ THUYẾT PHẢI)
// ----------------------------------------------------------------------------
#text(10pt, weight: "bold", fill: rgb("#2b6cb0"))[Mẫu 2B: Song song (Bàn cờ bên Trái – Lý thuyết bên Phải)]
#v(2pt)

#grid(
  columns: (auto, 1fr),
  gutter: 14pt,
  align: (center + top, left + top),
  [
    #teaching-diagram(
      "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <--- THAY FEN Ở ĐÂY
      title: "Thế trận thực chiến",
      turn: "b",
      size: 13.5pt,
      caption: "Đen đang tạo sức ép lên c3"
    )
  ],
  [
    *Phân tích Kế hoạch & Ý đồ chiến thuật:*
    - Sau khi Đen phát triển Tượng lên b4, quân Mã tại c3 rơi vào thế ghim tuyệt đối.
    - *Kế hoạch của Trắng:*
      + Chơi `0-0` (Nhập thành) để đưa Vua thoát khỏi đường chiếu của Tượng.
      + Hoặc chơi `a3` để lập tức chất vấn quân Tượng đối phương.
    - *Cảnh báo:* Tránh vội vàng di chuyển Hậu khi chưa tháo ghim an toàn cho Vua!
  ]
)

#v(10pt)

// ----------------------------------------------------------------------------
// MẪU 3: KHUNG KHÁI NIỆM LÝ THUYẾT KÈM BÀN CỜ NỔI BẬT
// ----------------------------------------------------------------------------
#text(10pt, weight: "bold", fill: rgb("#2b6cb0"))[Mẫu 3: Khung Khái niệm Lý thuyết kèm Bàn cờ]
#v(2pt)

#concept-box(title: "Nguyên lý Vàng: Khai thác Quân bị Ghim")[
  #grid(
    columns: (1fr, auto),
    gutter: 12pt,
    align: (left + horizon, center + horizon),
    [
      *Nguyên tắc Đại Kiện Tướng:*
      *"Gia tăng áp lực tấn công trực tiếp vào quân cờ đang bị ghim!"*
      
      Khi đối phương có quân bị ghim, hãy lập tức huy động Tốt hoặc quân nhẹ để tăng cường sức ép trước khi đối phương kịp gỡ ghim.
    ],
    [
      #chess-board(
        "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5", // <--- THAY FEN Ở ĐÂY
        size: 12.5pt,
        numbers: true
      )
    ]
  )
]

#v(10pt)

// ----------------------------------------------------------------------------
// MẪU 4: BÀN CỜ CÓ MŨI TÊN CHỈ HƯỚNG TẤN CÔNG (ARROWS)
// ----------------------------------------------------------------------------
#text(10pt, weight: "bold", fill: rgb("#2b6cb0"))[Mẫu 4: Bàn cờ có Mũi tên Chiến thuật (Tactical Arrows)]
#v(2pt)
Mũi tên chiến thuật giúp học viên nhìn thấy ngay đòn phối hợp hoặc đường đi của quân cờ:

#teaching-diagram(
  "r1bqkb1r/pppp1ppp/2n5/4p3/2B1n3/5N2/PPPP1PPP/RNBQK2R w KQkq - 0 4", // <--- THAY FEN Ở ĐÂY
  title: "Đòn phối hợp tấn công điểm yếu f7",
  turn: "w",
  size: 15pt,
  arrows: ("c4-f7", "d1-h5"), // <--- MŨI TÊN: ("ô_gốc-ô_đích", ...)
  caption: "Tượng c4 ngắm thẳng vào f7, Hậu d1 sẵn sàng tiến công lên h5."
)

#v(10pt)

// ----------------------------------------------------------------------------
// MẪU 5: BÀN CỜ GÓC NHÌN BÊN ĐEN (TỰ ĐỘNG LẬT BÀN CỜ)
// ----------------------------------------------------------------------------
#text(10pt, weight: "bold", fill: rgb("#2b6cb0"))[Mẫu 5: Bàn cờ Góc nhìn bên Đen (Tự động đảo chiều)]
#v(2pt)

#teaching-diagram(
  "r1b1k2r/ppppqppp/2n5/4P3/2B2Bn1/2P2N2/P4PPP/R2Q1RK1 b kq - 0 10", // <--- THAY FEN Ở ĐÂY
  title: "Góc nhìn Đen: Kế hoạch phản công cánh Vua",
  turn: "b", // Khi turn: "b", bàn cờ sẽ tự động lật theo góc nhìn Đen
  size: 15pt,
  caption: "Từ góc nhìn của Đen, phối hợp Hậu e7 và Mã g4 tạo thế công nguy hiểm."
)
