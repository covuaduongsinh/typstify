// ============================================================================
// TYPSTIFY CHESS ENGINE - MAGAZINE & ARTICLE MODULE
// Chuyên dụng cho Tạp chí Cờ Vua, Kỷ yếu Giải đấu, Bản tin CLB & Bài viết chuyên khảo.
// Phong cách New In Chess, Chess Life, British Chess Magazine.
// ============================================================================

#import "theme.typ": *
#import "symbols.typ": turn-indicator, nag, is-black-turn, chess-board

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
    fill: ds-brand-soft,
    stroke: 0.6pt + ds-line,
    radius: 4pt,
    inset: (x: 10pt, y: 8pt)
  )[
    // Hàng trên: Sự kiện, Địa điểm, Mã ECO, Kết quả. Trường để trống
    // (PGN thiếu thông tin) được bỏ qua thay vì in ra "( , )".
    #let place = (
      site,
      if round != "" { "V" + round } else { "" },
      date,
    ).filter(x => x != "").join(", ", default: "")
    #let opening-line = (eco, opening).filter(x => x != "").join(": ", default: "")
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      [
        #text(8pt, weight: "bold", fill: ds-muted)[#event#if place != "" [ (#place)]] \
        #if opening-line != "" [#text(7.5pt, fill: ds-muted)[#opening-line]]
      ],
      [
        #if result != "" [
          #box(
            fill: ds-brand,
            radius: 3pt,
            inset: (x: 6pt, y: 3pt)
          )[
            #text(8.5pt, weight: "bold", fill: ds-paper)[#result]
          ]
        ]
      ]
    )
    #v(4pt)
    #line(length: 100%, stroke: 0.4pt + ds-line)
    #v(4pt)
    // Hàng dưới: Kỳ thủ Trắng vs Đen
    #grid(
      columns: (1fr, auto, 1fr),
      align: (left + horizon, center + horizon, right + horizon),
      [
        #turn-indicator("w", size: 8pt) #h(3pt)
        #text(8.5pt, weight: "bold", fill: ds-ink)[#if white-title != "" [#white-title ]#white]
        #let w-info = (white-fed, white-elo).filter(x => x != "").join(", ", default: "")
        #if w-info != "" [#text(7.5pt, fill: ds-muted)[ (#w-info)]]
      ],
      [
        #text(8pt, weight: "bold", fill: ds-subtle)[VS]
      ],
      [
        #let b-info = (black-elo, black-fed).filter(x => x != "").join(", ", default: "")
        #if b-info != "" [#text(7.5pt, fill: ds-muted)[(#b-info) ]]
        #text(8.5pt, weight: "bold", fill: ds-ink)[#if black-title != "" [#black-title ]#black]
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
      #set par(justify: false)
      #if move-num != "" or caption != "" [
        #grid(
          columns: (1fr, auto),
          align: (left + horizon, right + horizon),
          [#text(7.5pt, weight: "bold", fill: ds-muted)[#move-num]],
          [#turn-indicator(turn, size: 7.5pt)]
        )
        #v(2pt)
      ]
      #chess-board(fen-str, size: size, reverse: is-black-turn(turn), arrows: arrows, frame: 0.8pt + ds-text)
      #if caption != "" [
        #v(2pt)
        #text(7pt, style: "italic", fill: ds-muted)[#caption]
      ]
    ]
  ]
}

// Hộp trích dẫn danh ngôn / bình luận nổi bật (Callout Box)
#let chess-quote(author: "", text-content) = {
  rect(
    width: 100%,
    stroke: (left: 3pt + ds-brand),
    fill: ds-brand-soft,
    radius: (right: 4pt),
    inset: (x: 10pt, y: 7pt)
  )[
    #text(8.5pt, style: "italic", fill: ds-brand-dark)[“#text-content”]
    #if author != "" [
      #v(2pt)
      #align(right)[
        #text(7.5pt, weight: "bold", fill: ds-brand-light)[— #author]
      ]
    ]
  ]
}
