// ============================================================================
// TYPSTIFY CHESS ENGINE - ECO & INFORMANT OPENINGS MODULE
// Chuyên dụng cho Bách khoa toàn thư Khai cuộc (ECO Encyclopedia) &
// Tạp chí Šahovski Informator (Chess Informant).
// ============================================================================

#import "theme.typ": *
#import "symbols.typ": note-num, nag, turn-indicator, is-black-turn, chess-board

// Hàm tạo Tiêu đề Khai cuộc ECO Chuẩn
#let eco-header(
  code: "C 58",
  name: "Phòng thủ Hai Mã (Two Knights Defense)",
  subname: "Biến thể Polerio - Bogoljubow (5...Na5)",
  intro-moves: "1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5"
) = {
  block(width: 100%, stroke: (bottom: 1pt + ds-text), inset: (bottom: 6pt))[
    #grid(
      columns: (auto, 1fr),
      gutter: 12pt,
      align: (left + horizon, left + horizon),
      box(
        fill: ds-text,
        radius: 3pt,
        inset: (x: 8pt, y: 4pt)
      )[
        #text(13pt, weight: "bold", fill: ds-paper)[#code]
      ],
      [
        #text(12pt, weight: "bold", fill: ds-ink)[#name] \
        #if subname != "" [
          #text(9pt, style: "italic", fill: ds-muted)[#subname] \
        ]
        #text(8.5pt, weight: "medium", fill: ds-muted)[#intro-moves]
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
    stroke: (x, y) => if y == 0 { (bottom: 0.8pt + ds-text) } else { 0.3pt + ds-line },
    fill: (col, row) => if row == 0 { ds-brand-soft } else if calc.odd(row) { ds-paper } else { ds-brand-soft },
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
  to-move: auto,
  eval-text: "",
  caption: "",
  size: 14pt,
  arrows: ()
) = {
  let side = if to-move != auto { to-move } else { turn }
  grid(
    columns: (auto, 1fr),
    gutter: 10pt,
    align: (top, top),
    chess-board(fen-str, size: size, reverse: is-black-turn(side), arrows: arrows, frame: 0.8pt + ds-muted),
    [
      #if title != "" [
        #turn-indicator(side, size: 8pt) #h(2pt)
        #text(9pt, weight: "bold", fill: ds-text)[#title] \
      ]
      #if eval-text != "" [
        #text(8.5pt, weight: "bold", fill: ds-brand)[Đánh giá: #eval-text] \
      ]
      #v(2pt)
      #text(8pt, fill: ds-muted)[#caption]
    ]
  )
}
