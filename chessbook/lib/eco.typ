// ============================================================================
// TYPSTIFY CHESS ENGINE - ECO & INFORMANT OPENINGS MODULE
// Chuyên dụng cho Bách khoa toàn thư Khai cuộc (ECO Encyclopedia) &
// Tạp chí Šahovski Informator (Chess Informant).
// ============================================================================

#import "@preview/board-n-pieces:0.9.0": board, fen
#import "symbols.typ": note-num, nag, turn-indicator

// Hàm tạo Tiêu đề Khai cuộc ECO Chuẩn
#let eco-header(
  code: "C 58",
  name: "Phòng thủ Hai Mã (Two Knights Defense)",
  subname: "Biến thể Polerio - Bogoljubow (5...Na5)",
  intro-moves: "1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5"
) = {
  block(width: 100%, stroke: (bottom: 1pt + rgb("#2d3748")), inset: (bottom: 6pt))[
    #grid(
      columns: (auto, 1fr),
      gutter: 12pt,
      align: (left + horizon, left + horizon),
      box(
        fill: rgb("#2d3748"),
        radius: 3pt,
        inset: (x: 8pt, y: 4pt)
      )[
        #text(13pt, weight: "bold", fill: rgb("#ffffff"))[#code]
      ],
      [
        #text(12pt, weight: "bold", fill: rgb("#1a202c"))[#name] \
        #if subname != "" [
          #text(9pt, style: "italic", fill: rgb("#4a5568"))[#subname] \
        ]
        #text(8.5pt, weight: "medium", fill: rgb("#718096"))[#intro-moves]
      ]
    )
  ]
}

// Bảng ma trận nước đi Informant Style (ECO Table)
#let eco-table(
  columns-header: (),
  rows: ()
) = {
  set text(size: 8pt)
  table(
    columns: (22pt, ..range(columns-header.len()).map(_ => 1fr)),
    stroke: (x, y) => if y == 0 { (bottom: 0.8pt + rgb("#2d3748")) } else { 0.3pt + rgb("#e2e8f0") },
    fill: (col, row) => if row == 0 { rgb("#f7fafc") } else if calc.odd(row) { rgb("#ffffff") } else { rgb("#fafafa") },
    align: (col, row) => if col == 0 { center + horizon } else { left + horizon },
    table.header([*No.*], ..columns-header.map(h => [*#h*])),
    ..rows.flatten()
  )
}

// Khung diagram khai cuộc kèm chú thích bên cạnh
#let opening-diagram-box(
  fen-str,
  title: "",
  turn: "w",
  eval-text: "",
  caption: ""
) = {
  grid(
    columns: (auto, 1fr),
    gutter: 10pt,
    align: (top, top),
    box(stroke: 0.8pt + rgb("#4a5568"), inset: 0pt)[
      #board(
        fen(fen-str),
        square-size: 14pt,
        display-numbers: false,
        white-square-fill: rgb("#ffffff"),
        black-square-fill: rgb("#e2e8f0")
      )
    ],
    [
      #if title != "" [
        #text(9pt, weight: "bold", fill: rgb("#2d3748"))[#title] \
      ]
      #if eval-text != "" [
        #text(8.5pt, weight: "bold", fill: rgb("#2b6cb0"))[Đánh giá: #eval-text] \
      ]
      #v(2pt)
      #text(8pt, fill: rgb("#4a5568"))[#caption]
    ]
  )
}
