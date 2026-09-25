// ============================================================================
// TYPSTIFY CHESS ENGINE - COURSEWARE & LESSON PLANS MODULE (DẠNG D)
// Chuyên dụng cho Giáo trình Trung tâm Cờ vua, Bài giảng HLV & Tài liệu Tập huấn.
// Hỗ trợ khung mục tiêu bài giảng, diagram mũi tên chiến thuật, câu hỏi trắc nghiệm.
// ============================================================================

#import "theme.typ": *
#import "symbols.typ": turn-indicator, nag, is-black-turn, chess-board

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
    fill: ds-brand-soft,
    stroke: (left: 4pt + ds-brand, rest: 0.5pt + ds-brand-border),
    radius: (right: 4pt),
    inset: (x: 12pt, y: 10pt)
  )[
    #grid(
      columns: (1fr, auto),
      align: (left + horizon, right + horizon),
      [
        #text(9pt, weight: "bold", fill: ds-brand)[BÀI HỌC SỐ #lesson-num] \
        #v(1pt)
        #text(13pt, weight: "bold", fill: ds-brand-dark)[#title]
      ],
      [
        #box(fill: ds-brand, radius: 3pt, inset: (x: 6pt, y: 3pt))[
          #text(8pt, weight: "bold", fill: ds-paper)[#level]
        ] \
        #v(2pt)
        #text(7.5pt, fill: ds-muted)[#duration]
      ]
    )
    #if objective != "" [
      #v(6pt)
      #line(length: 100%, stroke: 0.4pt + ds-line)
      #v(4pt)
      #text(8pt, weight: "bold", fill: ds-brand-dark)[◎ Mục tiêu bài học: ]
      #text(8pt, fill: ds-text)[#objective]
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
    fill: ds-brand-soft,
    stroke: 0.8pt + ds-line,
    radius: 4pt,
    inset: (x: 10pt, y: 8pt)
  )[
    #text(9pt, weight: "bold", fill: ds-brand)[✦ #title]
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
      #set par(justify: false)
      #if title != "" [
        #grid(
          columns: (1fr, auto),
          align: (left + horizon, right + horizon),
          [#text(8.5pt, weight: "bold", fill: ds-text)[#title]],
          [#turn-indicator(turn, size: 8.5pt)]
        )
        #v(2pt)
      ]
      #chess-board(fen-str, size: size, reverse: is-black-turn(turn), numbers: true, arrows: arrows, frame: 1pt + ds-text)
      #if caption != "" [
        #v(3pt)
        #text(7.5pt, style: "italic", fill: ds-muted)[#caption]
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
    fill: ds-paper,
    stroke: 0.5pt + ds-line,
    radius: 3pt,
    inset: (x: 8pt, y: 6pt)
  )[
    #text(8.5pt, weight: "bold", fill: ds-brand)[Câu hỏi #number: ]
    #text(8.5pt, fill: ds-ink)[#question]
    #if choices.len() > 0 [
      #v(3pt)
      #grid(
        columns: (1fr, 1fr),
        gutter: 4pt,
        ..choices.map(c => text(8pt, fill: ds-muted)[#c])
      )
    ]
    #if answer != none [
      #v(2pt)
      #text(7.5pt, style: "italic", fill: ds-muted)[Đáp án: #answer]
    ]
  ]
}

// Ghi chú Dành riêng cho Huấn luyện viên (Teacher / Instructor Note)
#let instructor-note(note) = {
  rect(
    width: 100%,
    stroke: (left: 3pt + ds-warn),
    fill: ds-warn-soft,
    radius: (right: 3pt),
    inset: (x: 8pt, y: 5pt)
  )[
    #text(8pt, weight: "bold", fill: ds-warn)[✎ Lưu ý cho HLV: ]
    #text(8pt, fill: ds-text)[#note]
  ]
}
