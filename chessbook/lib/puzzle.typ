// ============================================================================
// TYPSTIFY CHESS ENGINE - PUZZLE & TACTICS MODULE
// Chuyên dụng cho sách bài tập chiến thuật, câu đố, cẩm nang luyện cờ.
// Hỗ trợ lưới thế cờ (2x2, 2x3, 3x3, 3x4), đáp án lật ngược và gom đáp án cuối sách.
// ============================================================================

#import "@preview/board-n-pieces:0.9.0": board, fen
#import "symbols.typ": turn-indicator

// State toàn cục để tự động thu thập đáp án câu đố
#let puzzle-solutions-state = state("typstify-puzzle-solutions", ())

// Biểu tượng sao độ khó
#let difficulty-stars(level) = {
  let count = calc.min(5, calc.max(1, level))
  text(fill: rgb("#d69e2e"), size: 8pt)[
    #for i in range(count) [★]
    #for i in range(5 - count) [#text(fill: rgb("#cbd5e0"))[★]]
  ]
}

// Hàm vẽ 1 câu đố / bài tập cờ vua hoàn chỉnh (Puzzle Card)
#let puzzle-card(
  fen-str,
  number: 1,
  title: "",
  to-move: auto,
  difficulty: 1,
  hint: none,
  size: 16pt,
  solution: none,
  arrows: ()
) = {
  // Xác định lượt đi
  let is-black = if to-move == auto {
    fen-str.contains(" b ") or fen-str.ends-with(" b")
  } else {
    to-move == "b" or to-move == "black" or to-move == "Đen"
  }
  let turn-label = if is-black { "Đen đi trước" } else { "Trắng đi trước" }
  let board-width = size * 8

  // Nếu có solution, tự động ghi nhận vào state
  if solution != none {
    puzzle-solutions-state.update(curr => {
      curr.push((
        number: number,
        title: title,
        to-move: if is-black { "b" } else { "w" },
        solution: solution
      ))
      curr
    })
  }

  block(width: board-width, breakable: false)[
    // Header bài tập: Số thứ tự + Lượt đi
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      [
        #box(
          fill: rgb("#2b6cb0"),
          radius: 3pt,
          inset: (x: 5pt, y: 2pt)
        )[
          #text(8pt, weight: "bold", fill: rgb("#ffffff"))[#number]
        ]
        #h(4pt)
        #if title != "" [
          #text(8.5pt, weight: "bold", fill: rgb("#2d3748"))[#title]
        ]
      ],
      [
        #turn-indicator(if is-black { "b" } else { "w" }, size: 9pt)
        #h(3pt)
        #text(7.5pt, fill: rgb("#718096"), weight: "medium")[#turn-label]
      ]
    )

    #v(3pt)

    // Khung bàn cờ
    #box(
      stroke: 0.9pt + rgb("#2d3748"),
      radius: 1pt,
      fill: rgb("#ffffff"),
      inset: 0pt
    )[
      #board(
        fen(fen-str),
        square-size: size,
        reverse: is-black,
        display-numbers: false,
        white-square-fill: rgb("#ffffff"),
        black-square-fill: rgb("#e2e8f0"),
        arrows: arrows
      )
    ]

    // Footer phụ: Độ khó và Gợi ý (nếu có)
    #if difficulty > 0 or hint != none [
      #v(2pt)
      #grid(
        columns: (1fr, auto),
        align: (left + horizon, right + horizon),
        [
          #if hint != none [
            #text(7pt, style: "italic", fill: rgb("#a0aec0"))[Gợi ý: #hint]
          ]
        ],
        [
          #if difficulty > 0 [
            #difficulty-stars(difficulty)
          ]
        ]
      )
    ]
  ]
}

// In dải đáp án úp ngược 180 độ ở chân trang (Upside Down Solutions)
#let upside-down-solutions(solutions-dict) = {
  v(1fr)
  line(length: 100%, stroke: (dash: "densely-dashed", thickness: 0.5pt, paint: rgb("#a0aec0")))
  v(2pt)
  rotate(180deg)[
    #box(width: 100%, fill: rgb("#f7fafc"), inset: 6pt, radius: 4pt)[
      #text(7pt, weight: "bold", fill: rgb("#718096"))[ĐÁP ÁN (LẬT NGƯỢC)]
      #v(2pt)
      #for (k, v) in solutions-dict [
        #text(7pt, weight: "bold")[#k.] #text(7pt)[#v] #h(8pt)
      ]
    ]
  ]
}

// Trang xuất toàn bộ đáp án cuối sách / cuối chương
#let render-puzzle-solutions() = context {
  let sol-list = puzzle-solutions-state.final()
  if sol-list.len() > 0 [
    #heading(level: 2)[Đáp án & Lời giải Chi tiết]
    #v(8pt)
    #grid(
      columns: (1fr, 1fr),
      gutter: 12pt,
      ..sol-list.map(item => box(
        stroke: 0.5pt + rgb("#e2e8f0"),
        inset: 6pt,
        radius: 3pt,
        width: 100%
      )[
        #text(8.5pt, weight: "bold", fill: rgb("#2b6cb0"))[Bài #item.number: #item.title]
        #h(4pt)
        #turn-indicator(item.to-move, size: 7pt)
        #v(3pt)
        #text(8pt)[#item.solution]
      ])
    )
  ]
}
