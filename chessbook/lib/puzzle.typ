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

// Hàm chuẩn hóa dữ liệu thô từ CSV thành danh sách dictionary bài tập
#let csv-to-puzzles(csv-data) = {
  if type(csv-data) != array or csv-data.len() == 0 {
    return ()
  }
  // Nếu là array of dictionaries (khi load csv(..., row-type: dictionary))
  if type(csv-data.first()) == dictionary {
    return csv-data.map(row => {
      let diff = row.at("difficulty", default: "1")
      let diff-int = if type(diff) == int { diff } else { int(diff) }
      (
        fen: row.at("fen", default: ""),
        title: row.at("title", default: ""),
        turn: row.at("turn", default: auto),
        difficulty: diff-int,
        hint: row.at("hint", default: none),
        solution: row.at("solution", default: none),
      )
    })
  }
  // Nếu là array of arrays (dạng bảng ma trận thông thường)
  let start-idx = 0
  let headers = csv-data.first()
  if headers.contains("fen") or headers.contains("FEN") {
    start-idx = 1
  }
  let result = ()
  for row in csv-data.slice(start-idx) {
    if row.len() > 0 and row.first().trim() != "" {
      let diff-val = if row.len() > 3 and row.at(3) != "" {
        let raw = row.at(3)
        if type(raw) == int { raw } else { int(raw) }
      } else { 1 }

      result.push((
        fen: row.at(0, default: ""),
        title: row.at(1, default: ""),
        turn: row.at(2, default: auto),
        difficulty: diff-val,
        hint: if row.len() > 4 and row.at(4) != "" { row.at(4) } else { none },
        solution: if row.len() > 5 and row.at(5) != "" { row.at(5) } else { none },
      ))
    }
  }
  result
}

// Alias nội bộ để không bị trùng tên với tham số
#let _render-upside-down-solutions = upside-down-solutions

// Tự động phân trang và hiển thị tuyển tập bài tập từ cơ sở dữ liệu (JSON / CSV / Array)
#let render-puzzle-collection(
  data,
  layout: "a4-3x4",
  per-page: auto,
  start-number: 1,
  show-upside-down: true,
  upside-down: auto,
  upside-down-solutions: auto,
  render-appendix-at-end: true,
  page-title: none,
) = {
  let do-upside-down = if upside-down-solutions != auto {
    upside-down-solutions
  } else if upside-down != auto {
    upside-down
  } else {
    show-upside-down
  }
  let items = if type(data) == array {
    if data.len() > 0 and type(data.first()) == array {
      csv-to-puzzles(data)
    } else {
      data
    }
  } else {
    ()
  }

  let count-per-page = if per-page != auto {
    per-page
  } else if layout == "16x24-2x3" or layout == "16x24" {
    6
  } else if layout == "a5-2x2" or layout == "a5" {
    4
  } else {
    12
  }

  let total-items = items.len()
  let num-pages = calc.ceil(total-items / count-per-page)
  let cur-num = start-number

  for p in range(num-pages) {
    let start-idx = p * count-per-page
    let end-idx = calc.min(total-items, (p + 1) * count-per-page)
    let page-items = items.slice(start-idx, end-idx)

    // Gán số thứ tự bài tập tăng dần liên tục và trích xuất đáp án
    let numbered-items = ()
    let sol-dict = (:)
    for item in page-items {
      let num = cur-num
      cur-num += 1

      let p-dict = if type(item) == dictionary {
        item + (number: num)
      } else if type(item) == str {
        (fen: item, number: num)
      } else {
        item
      }
      numbered-items.push(p-dict)

      let sol = if type(p-dict) == dictionary { p-dict.at("solution", default: none) } else { none }
      if sol != none and sol != "" {
        sol-dict.insert(str(num), sol)
      }
    }

    if page-title != none [
      #heading(level: 2)[#page-title #(if num-pages > 1 [ (Trang #(p + 1))] else [])]
      #v(4pt)
    ]

    // Render lưới bài tập tương ứng
    if layout == "16x24-2x3" or layout == "16x24" {
      puzzle-grid-16x24(puzzles: numbered-items)
    } else {
      puzzle-grid-a4(puzzles: numbered-items)
    }

    // In dải đáp án úp ngược ở chân trang
    if do-upside-down and sol-dict.len() > 0 {
      _render-upside-down-solutions(sol-dict)
    }

    // Ngắt trang nếu chưa phải trang cuối hoặc có phụ lục đáp án
    if p < num-pages - 1 or render-appendix-at-end {
      pagebreak()
    }
  }

  if render-appendix-at-end {
    render-puzzle-solutions()
  }
}


