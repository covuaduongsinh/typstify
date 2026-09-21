// ============================================================================
// TYPSTIFY CHESS ENGINE - COURSEWARE & LESSON PLANS MODULE (DẠNG D)
// Chuyên dụng cho Giáo trình Trung tâm Cờ vua, Bài giảng HLV & Tài liệu Tập huấn.
// Hỗ trợ khung mục tiêu bài giảng, diagram mũi tên chiến thuật, câu hỏi trắc nghiệm.
// ============================================================================

#import "@preview/board-n-pieces:0.9.0": board, fen
#import "symbols.typ": turn-indicator, nag

// Tiêu đề Bài học Giáo trình (Lesson Header)
#let lesson-header(
  lesson-num: 1,
  title: "ĐÒN GHIM QUÂN TRONG CHIẾN THUẬT CỜ VUA",
  level: "Trình độ: Nhập môn & Cơ bản",
  duration: "Thời lượng: 90 phút",
  objective: "Học viên hiểu rõ khái niệm đòn ghim, phân biệt ghim tuyệt đối và ghim tương đối, nhận biết cơ hội thực hiện đòn ghim trong thực chiến."
) = {
  block(
    width: 100%,
    fill: rgb("#ebf8ff"),
    stroke: (left: 4pt + rgb("#3182ce"), rest: 0.5pt + rgb("#bee3f8")),
    radius: (right: 4pt),
    inset: (x: 12pt, y: 10pt)
  )[
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      [
        #text(9pt, weight: "bold", fill: rgb("#2b6cb0"))[BÀI HỌC SỐ #lesson-num] \
        #v(1pt)
        #text(13pt, weight: "bold", fill: rgb("#1a365d"))[#title]
      ],
      [
        #box(fill: rgb("#3182ce"), radius: 3pt, inset: (x: 6pt, y: 3pt))[
          #text(8pt, weight: "bold", fill: rgb("#ffffff"))[#level]
        ] \
        #v(2pt)
        #text(7.5pt, fill: rgb("#718096"))[#duration]
      ]
    )
    #if objective != "" [
      #v(6pt)
      #line(length: 100%, stroke: 0.4pt + rgb("#cbd5e0"))
      #v(4pt)
      #text(8pt, weight: "bold", fill: rgb("#2c5282"))[🎯 Mục tiêu bài học: ]
      #text(8pt, fill: rgb("#2d3748"))[#objective]
    ]
  ]
}

// Khung Khái niệm Lý thuyết (Key Concept Box)
#let concept-box(
  title: "Khái niệm Then chốt",
  content
) = {
  rect(
    width: 100%,
    fill: rgb("#f7fafc"),
    stroke: 0.8pt + rgb("#cbd5e0"),
    radius: 4pt,
    inset: (x: 10pt, y: 8pt)
  )[
    #text(9pt, weight: "bold", fill: rgb("#2b6cb0"))[💡 #title]
    #v(4pt)
    #content
  ]
}

// Diagram Giảng dạy Khổ lớn với Mũi tên & Ô Highlight (Teaching Diagram)
#let teaching-diagram(
  fen-str,
  title: "",
  turn: "w",
  size: 18pt,
  arrows: (),
  caption: ""
) = {
  align(center)[
    #block(width: size * 8, breakable: false)[
      #if title != "" [
        #grid(
          columns: (1fr, auto),
          align: (left + horizon, right + horizon),
          [#text(8.5pt, weight: "bold", fill: rgb("#2d3748"))[#title]],
          [#turn-indicator(turn, size: 8.5pt)]
        )
        #v(2pt)
      ]
      #box(stroke: 1pt + rgb("#2d3748"), fill: rgb("#ffffff"), inset: 0pt)[
        #board(
          fen(fen-str),
          square-size: size,
          reverse: (turn == "b" or turn == "black"),
          display-numbers: true,
          white-square-fill: rgb("#ffffff"),
          black-square-fill: rgb("#cbd5e0"),
          arrows: arrows
        )
      ]
      #if caption != "" [
        #v(3pt)
        #text(7.5pt, style: "italic", fill: rgb("#4a5568"))[#caption]
      ]
    ]
  ]
}

// Câu hỏi Trắc nghiệm & Bài tập Tình huống trên Lớp (Practice Question)
#let practice-question(
  number: 1,
  question: "",
  choices: (),
  answer: none
) = {
  block(
    width: 100%,
    fill: rgb("#ffffff"),
    stroke: 0.5pt + rgb("#e2e8f0"),
    radius: 3pt,
    inset: (x: 8pt, y: 6pt)
  )[
    #text(8.5pt, weight: "bold", fill: rgb("#2b6cb0"))[Câu hỏi #number: ]
    #text(8.5pt, fill: rgb("#1a202c"))[#question]
    #if choices.len() > 0 [
      #v(3pt)
      #grid(
        columns: (1fr, 1fr),
        gutter: 4pt,
        ..choices.map(c => text(8pt, fill: rgb("#4a5568"))[#c])
      )
    ]
    #if answer != none [
      #v(2pt)
      #text(7.5pt, style: "italic", fill: rgb("#718096"))[Đáp án: #answer]
    ]
  ]
}

// Ghi chú Dành riêng cho Huấn luyện viên (Teacher / Instructor Note)
#let instructor-note(note) = {
  rect(
    width: 100%,
    stroke: (left: 3pt + rgb("#dd6b20")),
    fill: rgb("#fffaf0"),
    radius: (right: 3pt),
    inset: (x: 8pt, y: 5pt)
  )[
    #text(8pt, weight: "bold", fill: rgb("#c05621"))[📌 Lưu ý cho HLV: ]
    #text(8pt, fill: rgb("#7b341e"))[#note]
  ]
}
