// ============================================================================
// TYPSTIFY CHESS ENGINE - MAGAZINE & ARTICLE MODULE
// Chuyên dụng cho Tạp chí Cờ Vua, Kỷ yếu Giải đấu, Bản tin CLB & Bài viết chuyên khảo.
// Phong cách New In Chess, Chess Life, British Chess Magazine.
// ============================================================================

#import "@preview/board-n-pieces:0.9.0": board, fen
#import "symbols.typ": turn-indicator, nag

// Thẻ Thông tin Ván đấu Đỉnh cao (Game Header Card)
#let game-header(
  white: "Magnus Carlsen",
  white-title: "GM",
  white-elo: "2835",
  white-fed: "NOR",
  black: "Hikaru Nakamura",
  black-title: "GM",
  black-elo: "2802",
  black-fed: "USA",
  event: "FIDE Candidates Tournament",
  site: "Toronto",
  date: "2024.04.15",
  round: "10",
  result: "1 - 0",
  eco: "C58",
  opening: "Two Knights Defense"
) = {
  box(
    width: 100%,
    fill: rgb("#f8fafc"),
    stroke: 0.6pt + rgb("#cbd5e0"),
    radius: 4pt,
    inset: (x: 10pt, y: 8pt)
  )[
    // Hàng trên: Sự kiện, Địa điểm, Mã ECO, Kết quả
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      [
        #text(8pt, weight: "bold", fill: rgb("#4a5568"))[#event (#site, R#round - #date)] \
        #text(7.5pt, fill: rgb("#718096"))[#eco: #opening]
      ],
      [
        #box(
          fill: rgb("#2b6cb0"),
          radius: 3pt,
          inset: (x: 6pt, y: 3pt)
        )[
          #text(8.5pt, weight: "bold", fill: rgb("#ffffff"))[#result]
        ]
      ]
    )
    #v(4pt)
    #line(length: 100%, stroke: 0.4pt + rgb("#e2e8f0"))
    #v(4pt)
    // Hàng dưới: Kỳ thủ Trắng vs Đen
    #grid(
      columns: (1fr, auto, 1fr),
      align: (left + horizon, center + horizon, right + horizon),
      [
        #turn-indicator("w", size: 8pt) #h(3pt)
        #text(8.5pt, weight: "bold", fill: rgb("#1a202c"))[#if white-title != "" [#white-title ]#white]
        #text(7.5pt, fill: rgb("#718096"))[ (#white-fed, #white-elo)]
      ],
      [
        #text(8pt, weight: "bold", fill: rgb("#a0aec0"))[VS]
      ],
      [
        #text(7.5pt, fill: rgb("#718096"))[(#black-elo, #black-fed) ]
        #text(8.5pt, weight: "bold", fill: rgb("#1a202c"))[#if black-title != "" [#black-title ]#black]
        #h(3pt) #turn-indicator("b", size: 8pt)
      ]
    )
  ]
}

// Bàn cờ nhỏ gọn lồng trong 1 cột văn bản (Column Diagram)
#let column-diagram(
  fen-str,
  move-num: "",
  caption: "",
  turn: "w",
  size: 13.5pt,
  arrows: ()
) = {
  align(center)[
    #block(width: size * 8, breakable: false)[
      #if move-num != "" or caption != "" [
        #grid(
          columns: (1fr, auto),
          align: (left + horizon, right + horizon),
          [#text(7.5pt, weight: "bold", fill: rgb("#4a5568"))[#move-num]],
          [#turn-indicator(turn, size: 7.5pt)]
        )
        #v(2pt)
      ]
      #box(stroke: 0.8pt + rgb("#2d3748"), fill: rgb("#ffffff"), inset: 0pt)[
        #board(
          fen(fen-str),
          square-size: size,
          reverse: (turn == "b" or turn == "black"),
          display-numbers: false,
          white-square-fill: rgb("#ffffff"),
          black-square-fill: rgb("#e2e8f0"),
          arrows: arrows
        )
      ]
      #if caption != "" [
        #v(2pt)
        #text(7pt, style: "italic", fill: rgb("#718096"))[#caption]
      ]
    ]
  ]
}

// Hộp trích dẫn danh ngôn / bình luận nổi bật (Callout Box)
#let chess-quote(author: "", text-content) = {
  rect(
    width: 100%,
    stroke: (left: 3pt + rgb("#3182ce")),
    fill: rgb("#ebf8ff"),
    radius: (right: 4pt),
    inset: (x: 10pt, y: 7pt)
  )[
    #text(8.5pt, style: "italic", fill: rgb("#2c5282"))[“#text-content”]
    #if author != "" [
      #v(2pt)
      #align(right)[
        #text(7.5pt, weight: "bold", fill: rgb("#4299e1"))[— #author]
      ]
    ]
  ]
}
