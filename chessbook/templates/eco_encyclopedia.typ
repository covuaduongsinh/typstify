// ============================================================================
// TEMPLATE: BÁCH KHOA TOÀN THƯ KHAI CUỘC (ECO ENCYCLOPEDIA / INFORMANT)
// ============================================================================

#import "../lib/lib.typ": *

#set page(
  paper: "a4",
  margin: (x: 1.5cm, top: 1.5cm, bottom: 1.5cm),
  header: context [
    #grid(
      columns: (1fr, 1fr),
      align: (left + horizon, right + horizon),
      [#text(9pt, weight: "bold")[C 58 - PHÒNG THỦ HAI MÃ]],
      [#text(9pt, weight: "bold")[Trang #counter(page).display()]]
    )
    #v(2pt)
    #line(length: 100%, stroke: 0.5pt + black)
  ]
)

#set text(font: ("Times New Roman", "Segoe UI Symbol"), size: 8.5pt, lang: "vi")
#set par(justify: true, leading: 0.5em)

// Tiêu đề Khai cuộc
#eco-header(
  code: "C 58",
  name: "Phòng thủ Hai Mã (Two Knights Defense)",
  subname: "Biến thể Polerio - Bogoljubow (5...Na5)",
  intro-moves: "1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5"
)

#v(8pt)

// Diagram thế cờ chính
#opening-diagram-box(
  "r1bqkb1r/ppp2ppp/5n2/n2P4/2B5/5N2/PPPP1PPP/RNBQK2R w KQkq - 1 6",
  title: "Thế cờ then chốt sau 5...Na5",
  turn: "w",
  eval-text: "⩲ (Trắng hơi ưu thế)",
  caption: "Trắng đứng trước lựa chọn 6.Bb5+ (Polerio) hoặc 6.d3 (Chigorin) hoặc 6.Qe2+."
)

#v(8pt)

#text(9pt, weight: "bold")[BẢNG BIẾN THỂ CHÍNH (MAIN VARIATIONS TABLE)]

#v(4pt)

// Ma trận bảng khai cuộc
#eco-table(
  columns-header: ("6", "7", "8", "9", "10", "11", "Đánh giá"),
  rows: (
    (
      [1],
      [Bb5+ #note-num(1)],
      [c6],
      [dxc6],
      [bxc6],
      [Be2 #note-num(2)],
      [h6],
      [#nag("14")]
    ),
    (
      [2],
      [Bb5+],
      [c6],
      [dxc6],
      [bxc6],
      [Qf3 #note-num(3)],
      [Rb8],
      [#nag("13")]
    ),
    (
      [3],
      [d3],
      [h6],
      [Nf3],
      [e4],
      [Qe2],
      [Nxc4],
      [#nag("10")]
    ),
    (
      [4],
      [Qe2+],
      [Be7],
      [Bb5+],
      [c6],
      [dxc6],
      [0-0],
      [#nag("15")]
    )
  )
)

#v(10pt)

#text(9pt, weight: "bold")[CHÚ THÍCH PHÂN TÍCH (ANALYSIS & NOTES)]

#v(4pt)

#note-num(1) *6. Bb5+ c6 7. dxc6 bxc6 8. Bd3:* Nước cờ phòng ngự chắc chắn do Steinitz đề xuất. Tiếp theo 8...Nd5 9. Nf3 Bd6 10. 0-0 0-0 11. Re1 f5 #nag("14")

#note-num(2) *8. Be2 h6 9. Nf3 e4 10. Ne5 Bd6 11. d4 exd3 12. Nxd3 Qc7:* Thế cờ cực kỳ năng động của Đen để đổi lấy 1 Tốt hy sinh #nag("44").

#note-num(3) *8. Qf3 Rb8:* Đòn ghim phản công. 9. Bd3 h6 10. Ne4 Nd5 11. Nbc3 Nf4 12. Be2 Ne6 với thế chủ động tấn công cho Đen #nag("36").
