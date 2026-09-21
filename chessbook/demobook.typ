#set page(
  width: 18cm,
  height: 24cm,
  margin: 2cm,
  header: context [
    #grid(
      columns: (auto, 1fr),
      align: (left + horizon, right + horizon),
      [
        #box(
          stroke: 0.8pt + rgb("#718096"),
          radius: 8pt,
          inset: (x: 7pt, y: 2.5pt),
          fill: rgb("#f7fafc")
        )[
          #text(8.5pt, weight: "bold", fill: rgb("#2d3748"))[#counter(page).display("1")]
        ]
      ],
      [
        #text(7.5pt, tracking: 0.5pt, weight: "medium", fill: rgb("#718096"))[
          CHIẾN THUẬT CỜ VUA TỪ CON SỐ 0 - TẬP 4
        ]
      ]
    )
    #v(-2pt)
    #line(length: 100%, stroke: 0.4pt + rgb("#cbd5e0"))
  ]
)

#set text(
  font: ("Arial", "Segoe UI Symbol"),
  lang: "vi",
  size: 9.5pt
)

#set par(
  justify: true,
  leading: 0.55em,
  first-line-indent: 0em
)

#import "@preview/board-n-pieces:0.9.0": board, fen

// --- Icon mục tiêu & Icon mảnh ghép ---
#let target-icon = {
  box(baseline: 15%, width: 14pt, height: 14pt)[
    #align(center + horizon)[
      #circle(radius: 6pt, stroke: 1.2pt + rgb("#4a5568"))[
        #place(center + horizon)[
          #circle(radius: 3.5pt, stroke: 0.8pt + rgb("#4a5568"), fill: none)[
            #place(center + horizon)[#circle(radius: 1.5pt, fill: rgb("#4a5568"))]
          ]
        ]
      ]
    ]
  ]
}

#let puzzle-icon = {
  box(baseline: 15%, width: 15pt, height: 15pt)[
    #align(center + horizon)[#text(13pt)[🧩]]
  ]
}

// --- Chỉ báo lượt đi (Trắng □ / Đen ■) ---
#let turn-box(turn) = {
  if turn == "b" or turn == "black" [
    #box(width: 8pt, height: 8pt, fill: black, radius: 0.5pt)
  ] else [
    #box(width: 8pt, height: 8pt, stroke: 0.9pt + black, fill: white, radius: 0.5pt)
  ]
}

// --- Hàm vẽ khung bàn cờ chuẩn sách in ---
#let book-diagram(
  fen-str,
  title: "",
  turn: auto,
  size: 18pt,
  arrows: ()
) = {
  let is-black = if turn == auto {
    fen-str.contains(" b ") or fen-str.ends-with(" b")
  } else {
    turn == "b" or turn == "black"
  }
  let board-width = size * 8
  block(width: board-width)[
    #grid(
      columns: (1fr, auto),
      align: (left + bottom, right + bottom),
      [#text(9pt, weight: "bold", fill: rgb("#2d3748"))[#title]],
      [#turn-box(if is-black { "b" } else { "w" })]
    )
    #v(2.5pt)
    #box(
      stroke: 0.9pt + rgb("#2d3748"),
      fill: white,
      inset: 0pt
    )[
      #board(
        fen(fen-str),
        square-size: size,
        display-numbers: false,
        white-square-fill: rgb("#ffffff"),
        black-square-fill: rgb("#d8dbe0"),
        arrows: arrows,
        arrow-fill: rgb("#3182ceb8"),
        arrow-thickness: 10%,
        arrow-base-offset: 20%,
      )
    ]
  ]
}

// =========================================================================
// TRANG 62: BÀI HỌC LÝ THUYẾT & VÍ DỤ
// =========================================================================
#counter(page).update(62)

#v(0.2em)

// --- PHẦN 1: BẢO VỆ QUÂN MỞ ĐƯỜNG ---
#text(12.5pt, weight: "bold", fill: rgb("#1a202c"))[
  #target-icon #h(4pt) Bảo vệ Quân mở đường
]

#v(0.4em)

#grid(
  columns: (auto, 1fr),
  gutter: 14pt,
  align: top,
  [
    #book-diagram(
      "5rk1/pb3ppp/2n5/2p5/8/3P1R2/PP1B1PPP/3R2K1 b - - 0 1",
      title: "Hình 45.1",
      turn: "b",
      size: 17.5pt,
      arrows: ("c6d4", "b7f3")
    )
  ],
  [
    #v(18pt)
    #text(10pt)[
      *Ví dụ hình bên:* \
      #v(4pt)
      *1... Md4* \
      #v(4pt)
      #text(fill: rgb("#2d3748"))[
        *♞* mở đường cho *♝* tấn công *♖*. \
        #v(3pt)
        *♞* khi tới *d4* được *♟ c5* bảo vệ.
      ]
    ]
  ]
)

#v(1em)

// --- PHẦN 2: BẢO VỆ QUÂN TẤN CÔNG CHÍNH ---
#text(12.5pt, weight: "bold", fill: rgb("#1a202c"))[
  #target-icon #h(4pt) Bảo vệ Quân tấn công chính
]

#v(0.4em)

#grid(
  columns: (1fr, 1fr),
  gutter: 10pt,
  align: top + left,
  [
    #book-diagram(
      "2r2rk1/pp1b1ppp/1q6/8/8/5P2/PPP2BPP/2KR3R w - - 0 1",
      title: "Hình 45.2",
      turn: "w",
      size: 17.5pt,
      arrows: ("f3f4", "f2b6")
    )
    #v(4pt)
    #align(left)[
      #text(9pt)[
        *1. f4*, *♙* mở đường cho *♗* tấn công *♛*. Quân tấn công chính *♗* đang được *♔* bảo vệ.
      ]
    ]
  ],
  [
    #book-diagram(
      "3q1rk1/pp3ppp/8/4n3/8/5B2/PPP1N1PP/3Q1RK1 b - - 0 1",
      title: "Hình 45.3",
      turn: "b",
      size: 17.5pt,
      arrows: ("e5f3", "d8d1")
    )
    #v(4pt)
    #align(left)[
      #text(9pt)[
        *1... Mxf3*, *♞* mở đường cho *♛* tấn công *♕*, đồng thời *♞* bảo vệ cho *♛* - Quân tấn công chính.
      ]
    ]
  ]
)

#pagebreak()

// =========================================================================
// TRANG 64: BÀI TẬP THỰC HÀNH 6 THẾ CỜ (TRANG 1)
// =========================================================================
#counter(page).update(64)

#v(0.1em)

#grid(
  columns: (1fr, auto),
  align: (left + horizon, right + horizon),
  [
    #text(12.5pt, weight: "bold", fill: rgb("#1a202c"))[
      #puzzle-icon #h(3pt) Tấn công mở có sự trợ giúp của đồng đội
    ]
  ],
  [
    #text(12.5pt, fill: rgb("#2d3748"))[#text(11pt)[$arrow.r$] ♖]
  ]
)

#v(0.3em)

#grid(
  columns: (1fr, 1fr),
  row-gutter: 8pt,
  column-gutter: 10pt,
  align: top + center,
  [
    #book-diagram(
      "r1bkb2r/p4pp1/2p2n1p/n3N3/3qp3/8/PPPPBPPP/RNBQK2R w - - 0 11",
      title: "TH45-1.1 - bài mẫu",
      size: 18pt,
      arrows: ("e5f7",)
    )
  ],
  [
    #book-diagram(
      "r1bk3r/pn3pp1/1bpq1n1p/4N3/1P2pP2/2P5/P3B1PP/RNBQKR2 w - - 0 15",
      title: "TH45-1.2",
      size: 18pt
    )
  ],
  [
    #book-diagram(
      "r4rk1/1nq2pp1/2p4p/3nN3/PP2pP2/B5Pb/3BB3/1R1QKR2 w - - 0 23",
      title: "TH45-1.3",
      size: 18pt
    )
  ],
  [
    #book-diagram(
      "r2r2k1/2P2qp1/5pNp/n2n4/P3pP2/B5P1/2QBB3/2R2K2 b - - 0 29",
      title: "TH45-1.4",
      size: 18pt
    )
  ],
  [
    #book-diagram(
      "1R6/6pk/3np1qp/Q7/P7/B2rp1P1/4B2P/5K2 w - - 0 36",
      title: "TH45-1.5",
      size: 18pt
    )
  ],
  [
    #book-diagram(
      "5r1k/3Q2p1/6qp/P4p2/8/3B2P1/4p1KP/4B3 w - - 0 44",
      title: "TH45-1.6",
      size: 18pt
    )
  ]
)

#pagebreak()

// =========================================================================
// TRANG 65: BÀI TẬP THỰC HÀNH (TIẾP THEO)
// =========================================================================

#v(0.1em)

#grid(
  columns: (1fr, auto),
  align: (left + horizon, right + horizon),
  [
    #text(12.5pt, weight: "bold", fill: rgb("#1a202c"))[
      #puzzle-icon #h(3pt) Tấn công mở có sự trợ giúp của đồng đội (tiếp theo)
    ]
  ],
  [
    #text(12.5pt, fill: rgb("#2d3748"))[#text(11pt)[$arrow.r$] ♖]
  ]
)

#v(0.3em)

#grid(
  columns: (1fr, 1fr),
  row-gutter: 8pt,
  column-gutter: 10pt,
  align: top + center,
  [
    #book-diagram(
      "r1b1k2r/p1q2pp1/2pbn2p/n7/8/3N3P/PPP1BPP1/RNBQK2R b - - 0 13",
      title: "TH45-1.7",
      size: 18pt
    )
  ],
  [
    #book-diagram(
      "r1b2rk1/p1q2pp1/3bn2p/n1p5/2P5/3N3P/PP2BPP1/RNBQ1RK1 b - - 0 15",
      title: "TH45-1.8",
      size: 18pt
    )
  ],
  [
    #book-diagram(
      "1r3rk1/p1q2pp1/3bn2p/n4b2/Q1p5/2N1BN1P/PP2BPP1/3R1RK1 b - - 0 22",
      title: "TH45-1.9",
      size: 18pt
    )
  ],
  []
)
