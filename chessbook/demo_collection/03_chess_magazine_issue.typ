// ============================================================================
// DEMO 3: TẠP CHÍ CỜ VUA 2 CỘT (CHESS MAGAZINE ISSUE) - DẠNG C
// ============================================================================

#import "../lib/lib.typ": *

#show: doc => chess-magazine-init(
  magazine-title: "TẠP CHÍ CỜ VUA VIỆT NAM",
  issue: "SỐ ĐẶC BIỆT - 2026",
  doc
)

#align(center)[
  #text(15pt, weight: "bold", fill: rgb("#1a365d"))[ĐẠI CHIẾN TẠI GIẢI ỨNG CỬ VIÊN 2026] \
  #v(2pt)
  #text(9.5pt, style: "italic", fill: rgb("#4a5568"))[Bình luận bởi Ban Chuyên Môn]
]

#v(8pt)

#columns(2, gutter: 14pt)[
  #chess-quote(
    author: "Garry Kasparov",
    "Chiến thuật là biết phải làm gì khi có điều gì đó để làm, còn chiến lược là biết phải làm gì khi không có gì để làm."
  )

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
    event: "FIDE Candidates",
    site: "Toronto",
    date: "2024.04.15",
    round: "10",
    result: "1 - 0",
    eco: "C58",
    opening: "Phòng thủ Hai Mã"
  )

  #v(4pt)

  *1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5 6. Bb5+ c6 7. dxc6 bxc6 8. Bd3!*

  Nước cờ kinh điển của Wilhelm Steinitz đưa Tượng về vị trí d3 kiểm soát trung tâm.

  #column-diagram(
    "r1bqkb1r/p4ppp/2p2n2/n3p3/8/3B1N2/PPPP1PPP/RNBQK2R b KQkq - 1 8",
    move-num: "Thế cờ sau 8. Bd3!",
    turn: "b",
    caption: "Đen đứng trước nhiều lựa chọn."
  )

  *8... h6 9. Ne4 Nd5 10. 0-0 Nf4 11. Be2 Bf5 12. d3! Nxe2+ 13. Qxe2 Be7 14. Nbc3 0-0 15. f4!* #nag("16")

  Trắng mở toang cánh Vua với nước đẩy Tốt f4 đầy quyết đoán, chuẩn bị đưa Xe vào vòng chiến.
]
