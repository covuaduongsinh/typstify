// ============================================================================
// TYPSTIFY CHESS PUBLISHING ENGINE - MAIN ENTRYPOINT
// Import toàn bộ công cụ cờ vua vào 1 điểm duy nhất.
// Cách dùng (gói cài sẵn trên máy chủ / máy cá nhân):
//   #import "@local/chessbook:0.1.0": *
// hoặc trong thư mục chessbook/: #import "lib/lib.typ": *
// ============================================================================

#import "theme.typ": *
#import "symbols.typ": *
#import "puzzle.typ": *
#import "eco.typ": *
#import "magazine.typ": *
#import "courseware.typ": *

// Hàm khởi tạo định dạng Sách Cờ Vua Chuẩn (khổ "a5", "a4", "16x24", hoặc mọi khổ
// giấy Typst hỗ trợ, ví dụ "b5" hoặc (width: 16cm, height: 24cm))
#let chess-book-init(
  title: "CẨM NANG CỜ VUA",
  subtitle: "",
  author: "",
  paper-size: "a5",
  font: font-serif,
  body
) = {
  let actual-paper = if paper-size == "16x24" or paper-size == "16x24cm" {
    (width: 16cm, height: 24cm)
  } else {
    paper-size
  }

  let is-16x24 = paper-size == "16x24" or paper-size == "16x24cm"
  let page-margin = if is-16x24 {
    (inside: 1.8cm, outside: 1.3cm, top: 1.5cm, bottom: 1.4cm)
  } else {
    (x: 1.5cm, top: 1.6cm, bottom: 1.6cm)
  }

  set document(title: title, author: author)
  set page(
    paper: if type(actual-paper) == str { actual-paper } else { "a4" },
    width: if type(actual-paper) == dictionary { actual-paper.at("width", default: auto) } else { auto },
    height: if type(actual-paper) == dictionary { actual-paper.at("height", default: auto) } else { auto },
    margin: page-margin,
    header: context [
      #let p = counter(page).get().first()
      #let is-odd = calc.odd(p)
      #grid(
        columns: (1fr, 1fr),
        align: (left + horizon, right + horizon),
        if is-odd [
          #text(8pt, weight: "bold", fill: ds-muted)[#title]
        ] else [
          #text(8pt, weight: "bold", fill: ds-text)[#p]
        ],
        if is-odd [
          #text(8pt, weight: "bold", fill: ds-text)[#p]
        ] else [
          #text(8pt, weight: "medium", fill: ds-muted)[#subtitle]
        ]
      )
      #v(2pt)
      #line(length: 100%, stroke: 0.4pt + ds-line)
    ]
  )

  set text(
    font: font,
    size: 9pt,
    lang: "vi"
  )

  set par(
    justify: true,
    leading: 0.55em,
    first-line-indent: 0em
  )

  body
}

// Hàm khởi tạo Phiếu Bài Tập / Worksheet (Khổ A4 hoặc 16x24cm)
#let chess-worksheet-init(
  title: "PHIẾU BÀI TẬP CỜ VUA",
  subtitle: "",
  author: "CLB Cờ vua Dương Sinh",
  date: "",
  paper-size: "a4",
  font: font-sans,
  body
) = {
  let actual-paper = if paper-size == "16x24" or paper-size == "16x24cm" {
    (width: 16cm, height: 24cm)
  } else {
    paper-size
  }

  set page(
    paper: if type(actual-paper) == str { actual-paper } else { "a4" },
    width: if type(actual-paper) == dictionary { actual-paper.at("width", default: auto) } else { auto },
    height: if type(actual-paper) == dictionary { actual-paper.at("height", default: auto) } else { auto },
    margin: (x: 1.2cm, top: 1.3cm, bottom: 1.2cm),
    header: context [
      #let p = counter(page).get().first()
      #grid(
        columns: (1fr, auto),
        align: (left + horizon, right + horizon),
        [
          #text(8pt, weight: "bold", fill: ds-brand)[#title]
          #if subtitle != "" [
            #text(7.5pt, fill: ds-muted)[ — #subtitle]
          ]
        ],
        [
          #if date != "" [
            #text(7.5pt, fill: ds-muted)[#date | ]
          ]
          #text(7.5pt, weight: "bold", fill: ds-text)[Trang #p]
        ]
      )
      #v(2pt)
      #line(length: 100%, stroke: 0.5pt + ds-brand)
    ]
  )

  set text(
    font: font,
    size: 8.5pt,
    lang: "vi"
  )

  set par(
    justify: true,
    leading: 0.52em
  )

  body
}

// Hàm khởi tạo định dạng Tạp Chí Cờ Vua Chuẩn (2 cột, Khổ A4)
#let chess-magazine-init(
  magazine-title: "TẠP CHÍ CỜ VUA VIỆT NAM",
  issue: "SỐ 09 - 2026",
  font: font-sans,
  body
) = {
  set page(
    paper: "a4",
    margin: (x: 1.5cm, top: 1.8cm, bottom: 1.6cm),
    header: context [
      #let p = counter(page).get().first()
      #grid(
        columns: (1fr, auto, 1fr),
        align: (left + horizon, center + horizon, right + horizon),
        [#text(8pt, weight: "bold", fill: ds-brand)[#magazine-title]],
        [#text(7.5pt, fill: ds-subtle)[| #issue |]],
        [#text(8pt, weight: "bold", fill: ds-text)[Trang #p]]
      )
      #v(2pt)
      #line(length: 100%, stroke: 0.6pt + ds-brand)
    ]
  )

  set text(
    font: font,
    size: 8.5pt,
    lang: "vi"
  )

  set par(
    justify: true,
    leading: 0.52em
  )

  body
}
