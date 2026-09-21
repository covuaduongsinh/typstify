// ============================================================================
// DEMO 1: SÁCH BÀI TẬP CHIẾN THUẬT CỜ VUA (TACTICAL PUZZLES) - DẠNG A
// ============================================================================

#import "../lib/lib.typ": *

#show: doc => chess-book-init(
  title: "TUYỂN TẬP BÀI TẬP CHIẾN THUẬT CỜ VUA",
  subtitle: "CHƯƠNG 1: ĐÒN GHIM & ĐÒN BẮT ĐÔI",
  author: "Chess Publishing Studio",
  paper-size: "a5",
  doc
)

#align(center)[
  #text(13pt, weight: "bold", fill: rgb("#2b6cb0"))[BÀI TẬP CHIẾN THUẬT: ĐÒN GHIM & BẮT ĐÔI] \
  #v(2pt)
  #text(8pt, fill: rgb("#718096"))[Quan sát kỹ các thế cờ và tìm nước đi tối ưu nhất cho bên đi trước]
]

#v(6pt)

#grid(
  columns: (1fr, 1fr),
  row-gutter: 12pt,
  column-gutter: 8pt,
  align: center,
  [
    #puzzle-card(
      "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
      number: 1,
      title: "Đòn gỡ ghim phản công",
      to-move: "w",
      difficulty: 1,
      hint: "Tốt ăn Mã e4",
      solution: "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân nhẹ."
    )
  ],
  [
    #puzzle-card(
      "r1b1kb1r/pppp1ppp/8/4q3/8/8/PPPP1PPP/RNBQKB1R w KQkq - 0 6",
      number: 2,
      title: "Hóa giải đòn chiếu",
      to-move: "w",
      difficulty: 2,
      hint: "Đưa Hậu ra cản chiếu",
      solution: "1. Qe2 Qxe2+ 2. Bxe2 Trắng phát triển an toàn."
    )
  ],
  [
    #puzzle-card(
      "r1bqkb1r/pp3ppp/2n5/3np3/8/1N1B4/PPP2PPP/RNBQK2R w KQkq - 0 8",
      number: 3,
      title: "Hy sinh Tượng h7",
      to-move: "w",
      difficulty: 3,
      hint: "Tượng ăn h7 chiếu",
      solution: "1. Bxh7+ Kxh7 2. Qxd5 Trắng hơn Tốt."
    )
  ],
  [
    #puzzle-card(
      "r2q1rk1/1pp2ppp/p1np1n2/4p3/1bB1P1b1/2NP1N2/PPP1QPPP/R1B2RK1 w - - 2 9",
      number: 4,
      title: "Ghim quân trung tâm",
      to-move: "w",
      difficulty: 2,
      hint: "Mã d5",
      solution: "1. Nd5 Nxd5 2. Bxd5 Nd4 3. Qd1 thế trận cân bằng."
    )
  ]
)

#upside-down-solutions((
  "1": "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân.",
  "2": "1. Qe2 Qxe2+ 2. Bxe2 Trắng an toàn.",
  "3": "1. Bxh7+ Kxh7 2. Qxd5 Trắng hơn Tốt.",
  "4": "1. Nd5 Nxd5 2. Bxd5 Cân bằng."
))
