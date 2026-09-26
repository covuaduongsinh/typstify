// ============================================================================
// TYPSTIFY CHESS ENGINE - PUZZLE & TACTICS MODULE
// Chuyên dụng cho sách bài tập chiến thuật, câu đố, cẩm nang luyện cờ.
// Hỗ trợ lưới thế cờ (2x2, 2x3, 3x3, 3x4), đáp án lật ngược và gom đáp án cuối sách.
// ============================================================================

#import "theme.typ": *
#import "symbols.typ": turn-indicator, is-black-turn, fen-turn, chess-board

// State toàn cục để tự động thu thập đáp án câu đố
#let puzzle-solutions-state = state("typstify-puzzle-solutions", ())

// Biểu tượng sao độ khó
#let difficulty-stars(level) = {
  let count = calc.min(5, calc.max(1, level))
  text(fill: ds-gold, size: 8pt)[
    #for i in range(count) [★]
    #for i in range(5 - count) [#text(fill: ds-line)[★]]
  ]
}

// Hàm vẽ 1 câu đố / bài tập cờ vua hoàn chỉnh (Puzzle Card)
#let puzzle-card(
  fen-str,
  number: 1,
  title: "",
  turn: auto,
  to-move: auto, // tên cũ của `turn`, vẫn nhận để không vỡ tài liệu cũ
  difficulty: 1,
  hint: none,
  size: 16pt,
  compact: false,
  solution: none,
  arrows: ()
) = {
  // Xác định lượt đi
  let side = if turn != auto { turn } else if to-move != auto { to-move } else { fen-turn(fen-str) }
  let is-black = is-black-turn(side)
  let turn-label = if is-black { "Đen đi" } else { "Trắng đi" }
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

  let num-font-size = if compact { 7.5pt } else { 8pt }
  let title-font-size = if compact { 7.5pt } else { 8.5pt }
  let turn-font-size = if compact { 7pt } else { 7.5pt }
  let indicator-size = if compact { 7.5pt } else { 9pt }

  block(width: board-width, breakable: false)[
    // Ô hẹp: căn đều hai bên (mặc định của sách) làm giãn chữ tiêu đề.
    #set par(justify: false)
    // Header bài tập: Số thứ tự + Lượt đi
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      [
        #box(
          fill: ds-brand,
          radius: 2.5pt,
          inset: (x: if compact { 4pt } else { 5pt }, y: if compact { 1.5pt } else { 2pt })
        )[
          #text(num-font-size, weight: "bold", fill: ds-paper)[#number]
        ]
        #if title != "" [
          #h(3pt)
          #text(title-font-size, weight: "bold", fill: ds-text)[#title]
        ]
      ],
      [
        #turn-indicator(if is-black { "b" } else { "w" }, size: indicator-size)
        #h(2pt)
        #text(turn-font-size, fill: ds-muted, weight: "medium")[#turn-label]
      ]
    )

    #v(if compact { 2pt } else { 3pt })

    // Khung bàn cờ
    #chess-board(fen-str, size: size, reverse: is-black, arrows: arrows)

    // Footer phụ: Độ khó và Gợi ý (nếu có)
    #if difficulty > 0 or hint != none [
      #v(if compact { 1.5pt } else { 2pt })
      #grid(
        columns: (1fr, auto),
        align: (left + horizon, right + horizon),
        [
          #if hint != none [
            #text(6.5pt, style: "italic", fill: ds-subtle)[Gợi ý: #hint]
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

// Lưới bài tập A4 chuẩn: 3 cột x 4 hàng (12 bài tập)
#let puzzle-grid-a4(
  puzzles: (),
  gutter: (x: 10pt, y: 8pt),
  size: 13.5pt,
  compact: true,
  ..args
) = {
  let items = if puzzles.len() > 0 { puzzles } else { args.pos() }
  align(center)[
    #grid(
      columns: (1fr, 1fr, 1fr),
      row-gutter: gutter.at("y", default: 8pt),
      column-gutter: gutter.at("x", default: 10pt),
      align: center + top,
      ..items.map(p => {
        if type(p) == dictionary {
          puzzle-card(
            p.fen,
            number: p.at("number", default: 1),
            title: p.at("title", default: ""),
            turn: p.at("turn", default: auto),
            difficulty: p.at("difficulty", default: 0),
            hint: p.at("hint", default: none),
            size: p.at("size", default: size),
            compact: p.at("compact", default: compact),
            solution: p.at("solution", default: none),
            arrows: p.at("arrows", default: ()),
          )
        } else {
          p
        }
      })
    )
  ]
}

// Lưới bài tập 16x24cm chuẩn: 2 cột x 3 hàng (6 bài tập)
#let puzzle-grid-16x24(
  puzzles: (),
  gutter: (x: 12pt, y: 10pt),
  size: 15pt,
  compact: false,
  ..args
) = {
  let items = if puzzles.len() > 0 { puzzles } else { args.pos() }
  align(center)[
    #grid(
      columns: (1fr, 1fr),
      row-gutter: gutter.at("y", default: 10pt),
      column-gutter: gutter.at("x", default: 12pt),
      align: center + top,
      ..items.map(p => {
        if type(p) == dictionary {
          puzzle-card(
            p.fen,
            number: p.at("number", default: 1),
            title: p.at("title", default: ""),
            turn: p.at("turn", default: auto),
            difficulty: p.at("difficulty", default: 0),
            hint: p.at("hint", default: none),
            size: p.at("size", default: size),
            compact: p.at("compact", default: compact),
            solution: p.at("solution", default: none),
            arrows: p.at("arrows", default: ()),
          )
        } else {
          p
        }
      })
    )
  ]
}

// In dải đáp án úp ngược 180 độ ở chân trang (Upside Down Solutions)
#let upside-down-solutions(solutions-dict) = {
  v(1fr)
  line(length: 100%, stroke: (dash: "densely-dashed", thickness: 0.5pt, paint: ds-subtle))
  v(2pt)
  rotate(180deg)[
    #box(width: 100%, fill: ds-brand-soft, inset: (x: 6pt, y: 4pt), radius: 3pt)[
      #text(6.5pt, weight: "bold", fill: ds-muted)[ĐÁP ÁN (LẬT NGƯỢC)]
      #v(1.5pt)
      #let entries = if type(solutions-dict) == dictionary {
        solutions-dict.pairs()
      } else if type(solutions-dict) == array {
        solutions-dict
      } else {
        ()
      }
      #for item in entries [
        #if type(item) == array [
          #text(6.5pt, weight: "bold")[#item.at(0).] #text(6.5pt)[#item.at(1)] #h(6pt)
        ] else [
          #text(6.5pt)[#item] #h(6pt)
        ]
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
        stroke: 0.5pt + ds-line,
        inset: 6pt,
        radius: 3pt,
        width: 100%
      )[
        #text(8.5pt, weight: "bold", fill: ds-brand)[Bài #item.number: #item.title]
        #h(4pt)
        #turn-indicator(item.to-move, size: 7pt)
        #v(3pt)
        #text(8pt)[#item.solution]
      ])
    )
  ]
}

