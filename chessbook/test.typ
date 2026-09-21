#set page(
  width: 16cm,
  height: 24cm,
  margin: 2cm,
  header: align(right)[
    #text(9pt, fill: luma(120))[Tài liệu Cờ vua • FEN & PGN • 2026]
  ],
  footer: [
    #line(length: 100%, stroke: 0.5pt + luma(180))
    #align(center)[#context counter(page).display("1 / 1", both: true)]
  ]
)

#set text(
  font: ("Arial", "Segoe UI Symbol"),
  lang: "vi",
  size: 12pt
)

#set par(
  justify: true,
  leading: 0.7em,
  first-line-indent: 0em
)

// --- Hàm vẽ bàn cờ FEN dùng package board-n-pieces (quân cờ vector Cburnett) ---
#import "@preview/board-n-pieces:0.9.0": board, fen

#let render-board(fen-str, caption: none, size: 12.5pt, flipped: false) = {
  align(center)[
    #box(
      stroke: 1.2pt + rgb("#593a1c"),
      radius: 3pt,
      fill: rgb("#faf6f0"),
      inset: 2.5pt
    )[
      #board(
        fen(fen-str),
        reverse: flipped,
        square-size: size,
        display-numbers: true,
        white-square-fill: rgb("#f0d9b5"),
        black-square-fill: rgb("#b58863"),
      )
    ]
    #if caption != none [
      #v(2pt)
      #text(9.5pt, style: "italic", fill: rgb("#4a5568"))[#caption]
    ]
  ]
}

// --- Tiêu đề tài liệu ---
#align(center)[
  #text(18pt, weight: "bold", fill: rgb("#1a365d"))[Tài Liệu Cờ Vua]
  
  #v(2pt)
  #text(12pt, style: "italic", fill: rgb("#4a5568"))[Trình bày thế cờ FEN, ván cờ PGN và các ký hiệu bình luận chuẩn quốc tế]
  
  #v(4pt)
  #text(10pt)[*Tác giả:* duongsinh | *Ngày tạo:* #datetime.today().display("[day]/[month]/[year]")]
]

#v(0.4em)
#line(length: 100%, stroke: 1.2pt + rgb("#2b6cb0"))
#v(0.5em)

// --- Mục lục ---
#outline(title: "Mục Lục", depth: 2, indent: auto)

#v(0.8em)

// --- Bố cục 2 cột cho khổ 16x24 cm ---
#columns(2, gutter: 10pt)[

= 1. Thế cờ FEN (Forsyth–Edwards Notation)

*FEN* là chuỗi ký tự tiêu chuẩn mô tả chính xác trạng thái bàn cờ tại một thời điểm:
+ *Vị trí quân cờ*: Hàng 8 đến 1 (`r, n, b, q, k, p` = Đen, `R, N, B, Q, K, P` = Trắng, `1-8` = ô trống).
+ *Lượt đi*: `w` (Trắng) hoặc `b` (Đen).
+ *Nhập thành*: `KQkq` hoặc `-`.
+ *Bắt tốt qua đường*: ví dụ `e3` hoặc `-`.
+ *Halfmove clock*: Luật 50 nước.
+ *Fullmove number*: Số thứ tự nước đi.

#v(0.3em)

#render-board(
  "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4",
  caption: [Hình 1: Two Knights Defence (C57)],
  size: 12.5pt
)

#v(2pt)

*Chuỗi FEN tương ứng:*
#rect(
  width: 100%,
  fill: rgb("#f7fafc"),
  stroke: 0.5pt + rgb("#cbd5e0"),
  radius: 3pt,
  inset: 5pt
)[
  #text(9pt, font: "Consolas")[
    `r1bqkb1r/pppp1ppp/2n2n2/`\
    `4p3/2B1P3/5N2/`\
    `PPPP1PPP/RNBQK2R w KQkq - 4 4`
  ]
]

#v(2pt)
*Thông tin thế cờ:*
- *Lượt đi:* Trắng (`w`)
- *Kế hoạch:* `4. Ng5` hoặc `4. d4`
- *Đánh giá:* Thế trận mở, sắc bén.

#colbreak()

= 2. Ván cờ PGN (Portable Game Notation)

*PGN* là định dạng chuẩn để lưu trữ dữ liệu ván cờ, gồm Tags và biên bản nước đi.

#v(0.3em)

#rect(
  width: 100%,
  fill: rgb("#fffaf0"),
  stroke: (left: 3.5pt + rgb("#dd6b20")),
  inset: 8pt,
  radius: (right: 3pt)
)[
  #text(11pt, weight: "bold", fill: rgb("#9c4221"))[The Evergreen Game (1852)]
  
  #v(3pt)
  #text(9pt, font: "Consolas", fill: rgb("#744210"))[
    [Event "Casual Game"] [Site "Berlin"] \
    [White "Adolf Anderssen"] \
    [Black "Jean Dufresne"] \
    [Result "1-0"] [ECO "C52"]
  ]
  #line(length: 100%, stroke: 0.5pt + rgb("#ed8936"))
  #v(3pt)
  *Biên bản nước đi:*
  
  #text(10pt)[
    *1.* e4 e5 *2.* Nf3 Nc6 *3.* Bc4 Bc5 *4.* b4 Bxb4 *5.* c3 Ba5 *6.* d4 exd4 *7.* O-O d3 *8.* Qb3 Qf6 *9.* e5 Qg6 *10.* Re1 Nge7 *11.* Ba3 b5 *12.* Qxb5 Rb8 *13.* Qa4 Bb6 *14.* Nbd2 Bb7 *15.* Ne4 Qf5 *16.* Bxd3 Qh5 *17.* Nf6+! gxf6 *18.* exf6 Rg8 *19.* Rad1! Qxf3 *20.* Rxe7+! Nxe7 *21.* Qxd7+!! Kxd7 *22.* Bf5++ Ke8 *23.* Bd7+ Kf8 *24.* Bxe7\# *1-0*
  ]
]

#v(0.4em)

= 3. Bảng ký hiệu chú giải cờ vua

#v(0.2em)

#align(center)[
  #table(
    columns: (auto, 1fr, auto, 1fr),
    fill: (col, row) => if row == 0 { rgb("#fefcbf") } else { none },
    stroke: 0.4pt + luma(160),
    inset: (x: 4pt, y: 4.5pt),
    align: (center + horizon, left + horizon, center + horizon, left + horizon),
    [*Ký hiệu*], [*Ý nghĩa*], [*Ký hiệu*], [*Ý nghĩa*],
    [`!`], [Nước hay], [`?`], [Sai lầm],
    [`!!`], [Thiên tài], [`??`], [Đại sai lầm],
    [`!?`], [Thú vị], [`?!`], [Đáng ngờ],
    [`+-`], [Trắng thắng thế], [`-+`], [Đen thắng thế],
    [`+=`], [Trắng hơi ưu], [`=`], [Cân bằng],
  )
]

]
