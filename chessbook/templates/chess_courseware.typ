// ============================================================================
// TEMPLATE: GIÁO TRÌNH & BÀI GIẢNG CỜ VUA (CHESS COURSEWARE & LESSON PLANS)
// Khổ giấy: A4, Khung mục tiêu bài giảng, Khái niệm lý thuyết, Diagram minh họa,
// Bài tập tình huống trên lớp & Lưu ý cho Huấn luyện viên.
// ============================================================================

#import "../lib/lib.typ": *

#set page(
  paper: "a4",
  margin: (x: 1.8cm, top: 1.8cm, bottom: 1.8cm),
  header: context [
    #grid(
      columns: (1fr, auto, 1fr),
      align: (left + horizon, center + horizon, right + horizon),
      [#text(8pt, weight: "bold", fill: rgb("#2b6cb0"))[CLB CỜ VUA VIỆT NAM - GIÁO TRÌNH HUẤN LUYỆN]],
      [#text(7.5pt, fill: rgb("#a0aec0"))[| CẤP ĐỘ 2 |]],
      [#text(8pt, weight: "bold", fill: rgb("#2d3748"))[Trang #counter(page).display()]]
    )
    #v(2pt)
    #line(length: 100%, stroke: 0.5pt + rgb("#cbd5e0"))
  ]
)

#set text(font: font-sans, size: 9pt, lang: "vi")
#set par(justify: true, leading: 0.55em)

// Tiêu đề bài học
#lesson-header(
  lesson-num: 4,
  title: "CHIẾN THUẬT: ĐÒN GHIM QUÂN (THE PIN)",
  level: "Cấp độ: Sơ cấp nâng cao",
  duration: "Thời lượng: 60 phút",
  objective: "Học viên nhận biết được sự khác nhau giữa Đòn ghim Tuyệt đối (quân bị ghim vào Vua) và Đòn ghim Tương đối (quân bị ghim vào Hậu/Xe), từ đó áp dụng đòn ghim để đoạt quân trong thực chiến."
)

#v(8pt)

#concept-box(title: "1. Khái niệm Đòn Ghim Tuyệt đối (Absolute Pin)")[
  Đòn ghim tuyệt đối xảy ra khi một quân cờ đứng chắn giữa quân tấn công của đối phương và quân *Vua*. Quân cờ bị ghim *tuyệt đối không được phép di chuyển* vì nếu di chuyển sẽ để Vua bị chiếu (vi phạm luật cờ vua).
]

#v(8pt)

// Diagram giảng dạy
#teaching-diagram(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5",
  title: "Ví dụ 1: Tượng Đen ghim Mã Trắng vào Vua",
  turn: "b",
  size: 16pt,
  caption: "Tượng đen ở b4 ghim cứng Mã c3 vào Vua e1. Trắng không thể di chuyển Mã c3."
)

#v(8pt)

#concept-box(title: "2. Cách thức Khai thác Quân bị Ghim")[
  Nguyên tắc vàng của Đại Kiện Tướng: *"Tăng thêm sức ép tấn công vào quân đang bị ghim!"*. Khi đối phương có quân bị ghim, hãy dùng Tốt hoặc quân nhẹ khác để tấn công vào quân đó.
]

#v(8pt)

#instructor-note[
  Nhắc nhở học viên kiểm tra kỹ xem quân phía sau là Vua (ghim tuyệt đối) hay Hậu/Xe (ghim tương đối) trước khi tính toán nước đi.
]

#v(8pt)

#text(9.5pt, weight: "bold", fill: rgb("#2b6cb0"))[3. Bài tập Thực hành trên Lớp]

#v(4pt)

#practice-question(
  number: 1,
  question: "Trong thế cờ trên, nếu Đen chơi 5... d5 để tấn công Tượng c4, Trắng nên xử lý thế nào?",
  choices: (
    "A. 6. exd5 mở toang trung tâm",
    "B. 6. Bxd5 giữ cấu trúc Tốt",
    "C. 6. 0-0 phát triển Vua an toàn",
    "D. 6. a3 đuổi Tượng đen"
  ),
  answer: "A hoặc B - Trắng xử lý linh hoạt giữ vững trung tâm."
)

#v(4pt)

#practice-question(
  number: 2,
  question: "Quân cờ nào có khả năng thực hiện đòn ghim?",
  choices: (
    "A. Xe, Tượng, Hậu (quân đi thẳng/chéo xa)",
    "B. Mã và Tốt",
    "C. Vua",
    "D. Tất cả các quân"
  ),
  answer: "A - Xe, Tượng và Hậu là các quân tầm xa có khả năng ghim."
)
