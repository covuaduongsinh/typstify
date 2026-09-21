// ============================================================================
// TEMPLATE: TẠP CHÍ CỜ VUA CHUYÊN NGHIỆP (CHESS MAGAZINE)
// Khổ giấy: A4, Bố cục: 2 Cột với thẻ ván cờ, Diagram minh họa, Callout quote
// ============================================================================

#import "../lib/lib.typ": *

#show: doc => chess-magazine-init(
  magazine-title: "TẠP CHÍ CỜ VUA VIỆT NAM",
  issue: "SỐ 09 - THÁNG 9/2026",
  doc
)

#align(center)[
  #text(16pt, weight: "bold", fill: rgb("#1a365d"))[ĐẠI CHIẾN TẠI GIẢI ỨNG CỬ VIÊN 2026] \
  #v(2pt)
  #text(10pt, style: "italic", fill: rgb("#4a5568"))[Bình luận & Phân tích chuyên sâu bởi Ban Biên Tập]
]

#v(8pt)

#columns(2, gutter: 14pt)[
  #chess-quote(
    author: "Garry Kasparov",
    "Cờ vua là một cuộc đấu trí mà ở đó chiến thuật là biết phải làm gì khi có điều gì đó để làm, còn chiến lược là biết phải làm gì khi không có gì để làm."
  )

  #v(6pt)

  Vòng thi đấu thứ 10 chứng kiến màn chạm trán đỉnh cao giữa hai kỳ thủ hàng đầu thế giới. Trắng đã sử dụng biến thể tấn công sắc bén để ép đối phương rơi vào tình thế phức tạp.

  #v(4pt)

  #game-header(
    white: "Magnus Carlsen",
    white-title: "GM",
    white-elo: "2835",
    white-fed: "NOR",
    black: "Hikaru Nakamura",
    black-title: "GM",
    black-elo: "2802",
    black-fed: "USA",
    event: "FIDE Candidates Tournament",
    site: "Toronto",
    date: "2024.04.15",
    round: "10",
    result: "1 - 0",
    eco: "C58",
    opening: "Phòng thủ Hai Mã"
  )

  #v(6pt)

  *1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5 6. Bb5+ c6 7. dxc6 bxc6 8. Bd3!*

  Nước cờ kinh điển của Wilhelm Steinitz, đưa Tượng về vị trí d3 nhằm bảo vệ Tốt trung tâm và kiểm soát đường chéo then chốt.

  #column-diagram(
    "r1bqkb1r/p4ppp/2p2n2/n3p3/8/3B1N2/PPPP1PPP/RNBQK2R b KQkq - 1 8",
    move-num: "Thế cờ sau 8. Bd3!",
    turn: "b",
    caption: "Đen đứng trước nhiều lựa chọn phản công."
  )

  *8... h6 9. Ne4 Nd5 10. 0-0 Nf4 11. Be2 Bf5 12. d3! Nxe2+ 13. Qxe2 Be7 14. Nbc3 0-0 15. f4!* #nag("16")

  Trắng mở toang cánh Vua với nước đẩy Tốt f4 đầy quyết đoán, chuẩn bị đưa Xe vào vòng chiến.

  #column-diagram(
    "r2q1rk1/p3bpp1/2p4p/n3pb2/4NP2/2NP4/PPP1Q1PP/R1B2RK1 b - - 0 15",
    move-num: "Thế cờ sau 15. f4!",
    turn: "b",
    caption: "Trắng nắm hoàn toàn quyền chủ động tấn công."
  )

  *15... exf4 16. Bxf4 Bg6 17. Rae1 Nb7 18. Qf3 Nc5 19. Kh1 Ne6 20. Be3 f5 21. Qh3!* #nag("18")

  Đòn ghim tinh tế khiến Đen không thể tiếp tục đẩy Tốt f4. Sau hàng loạt nước phối hợp chính xác, Trắng giành thắng lợi thuyết phục sau 42 nước đi.
]
