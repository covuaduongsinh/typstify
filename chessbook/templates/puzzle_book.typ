// ============================================================================
// TEMPLATE: SÁCH BÀI TẬP CHIẾN THUẬT CỜ VUA (TACTICAL PUZZLE BOOK)
// Khổ sách: A5 (14.8 x 21.0 cm) chuẩn in ấn
// ============================================================================

#import "../lib/lib.typ": *

#show: doc => chess-book-init(
  title: "100 THẾ CỜ CHIẾN THUẬT KINH ĐIỂN",
  subtitle: "CHƯƠNG 1: ĐÒN ĐÁNH ĐÔI & TẤN CÔNG MỞ",
  author: "Đại Kiện Tướng Chess Publishing",
  paper-size: "a5",
  doc
)

#align(center)[
  #text(14pt, weight: "bold", fill: rgb("#2b6cb0"))[CHƯƠNG 1: ĐÒN GHIM & ĐÒN ĐÁNH ĐÔI] \
  #v(2pt)
  #text(8.5pt, fill: rgb("#718096"))[Hãy tìm nước đi chiến thuật tốt nhất cho mỗi thế cờ dưới đây]
]

#v(8pt)

// Lưới 4 bài tập trên trang A5 (2 cột x 2 hàng)
#grid(
  columns: (1fr, 1fr),
  row-gutter: 14pt,
  column-gutter: 10pt,
  align: center,
  [
    #puzzle-card(
      "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
      number: 1,
      title: "Khai cuộc Ý",
      to-move: "w",
      difficulty: 1,
      hint: "Tấn công đôi và gỡ ghim",
      solution: "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân nhẹ."
    )
  ],
  [
    #puzzle-card(
      "r1b1kb1r/pppp1ppp/8/4q3/8/8/PPPP1PPP/RNBQKB1R w KQkq - 0 6",
      number: 2,
      title: "Chiếu bắt Hậu",
      to-move: "w",
      difficulty: 2,
      hint: "Chặn chiếu hiệu quả",
      solution: "1. Qe2 Qxe2+ 2. Bxe2 Trắng phát triển an toàn."
    )
  ],
  [
    #puzzle-card(
      "r1bqkb1r/pp3ppp/2n5/3np3/8/1N1B4/PPP2PPP/RNBQK2R w KQkq - 0 8",
      number: 3,
      title: "Đòn hy sinh Tượng",
      to-move: "w",
      difficulty: 3,
      hint: "Hy sinh tại h7",
      solution: "1. Bxh7+ Kxh7 2. Qxd5 Trắng bắt lại Mã với lợi thế tốt."
    )
  ],
  [
    #puzzle-card(
      "r2q1rk1/1pp2ppp/p1np1n2/4p3/1bB1P1b1/2NP1N2/PPP1QPPP/R1B2RK1 w - - 2 9",
      number: 4,
      title: "Ghim quân trung tâm",
      to-move: "w",
      difficulty: 2,
      hint: "Mã nhảy lên d5",
      solution: "1. Nd5 Nxd5 2. Bxd5 Nd4 3. Qd1 thế trận cân bằng."
    )
  ]
)

// Dải đáp án úp ngược ở chân trang
#upside-down-solutions((
  "1": "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân.",
  "2": "1. Qe2 Qxe2+ 2. Bxe2 Trắng phát triển tốt.",
  "3": "1. Bxh7+ Kxh7 2. Qxd5 Trắng hơn Tốt.",
  "4": "1. Nd5 Nxd5 2. Bxd5 Thế cờ cân bằng."
))
