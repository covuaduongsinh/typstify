// ============================================================================
// TYPSTIFY CHESS PUBLISHING ENGINE - MAIN ENTRYPOINT
// Import toàn bộ công cụ cờ vua vào 1 điểm duy nhất.
// Cách dùng: #import "lib/lib.typ": *
// ============================================================================

#import "symbols.typ": *
#import "puzzle.typ": *
#import "eco.typ": *
#import "magazine.typ": *
#import "courseware.typ": *

// Hàm khởi tạo định dạng Sách Cờ Vua Chuẩn (Khổ sách Crown Quarto 18x24cm hoặc A5)
#let chess-book-init(
  title: "CẨM NANG CỜ VUA",
  subtitle: "",
  author: "",
  paper-size: "a5",
  body
) = {
  let is-a5 = (paper-size == "a5")
  set page(
    paper: if is-a5 { "a5" } else { "a4" },
    margin: (x: 1.5cm, top: 1.6cm, bottom: 1.6cm),
    header: context [
      #let p = counter(page).get().first()
      #let is-odd = calc.odd(p)
      #grid(
        columns: (1fr, 1fr),
        align: (left + horizon, right + horizon),
        if is-odd [
          #text(8pt, weight: "bold", fill: rgb("#718096"))[#title]
        ] else [
          #text(8pt, weight: "bold", fill: rgb("#2d3748"))[#p]
        ],
        if is-odd [
          #text(8pt, weight: "bold", fill: rgb("#2d3748"))[#p]
        ] else [
          #text(8pt, weight: "medium", fill: rgb("#718096"))[#subtitle]
        ]
      )
      #v(2pt)
      #line(length: 100%, stroke: 0.4pt + rgb("#cbd5e0"))
    ]
  )

  set text(
    font: ("Times New Roman", "Segoe UI Symbol"),
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

// Hàm khởi tạo định dạng Tạp Chí Cờ Vua Chuẩn (2 cột, Khổ A4)
#let chess-magazine-init(
  magazine-title: "TẠP CHÍ CỜ VUA VIỆT NAM",
  issue: "SỐ 09 - 2026",
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
        [#text(8pt, weight: "bold", fill: rgb("#2b6cb0"))[#magazine-title]],
        [#text(7.5pt, fill: rgb("#a0aec0"))[| #issue |]],
        [#text(8pt, weight: "bold", fill: rgb("#2d3748"))[Trang #p]]
      )
      #v(2pt)
      #line(length: 100%, stroke: 0.6pt + rgb("#2b6cb0"))
    ]
  )

  set text(
    font: ("Arial", "Segoe UI Symbol"),
    size: 8.5pt,
    lang: "vi"
  )

  set par(
    justify: true,
    leading: 0.52em
  )

  body
}
