// ============================================================================
// TYPSTIFY CHESS ENGINE - SYMBOLS & NOTATIONS MODULE
// Cung cấp hệ thống ký hiệu quân cờ hình tượng (Figurines) và bộ ký hiệu
// đánh giá quốc tế (NAG - Numeric Annotation Glyphs) chuẩn FIDE & Informant.
// ============================================================================

#import "@preview/board-n-pieces:0.9.0": chess-sym

// --- 1. Ký hiệu Hình tượng Quân cờ (Figurine Pieces) ---
// Định nghĩa sẵn các biến quân cờ để chèn trực tiếp vào văn bản
#let wK = chess-sym.king.white
#let wQ = chess-sym.queen.white
#let wR = chess-sym.rook.white
#let wB = chess-sym.bishop.white
#let wN = chess-sym.knight.white
#let wP = chess-sym.pawn.white

#let bK = chess-sym.king.black
#let bQ = chess-sym.queen.black
#let bR = chess-sym.rook.black
#let bB = chess-sym.bishop.black
#let bN = chess-sym.knight.black
#let bP = chess-sym.pawn.black

// --- 2. Bộ Ký hiệu Đánh giá Quốc tế (NAG & Informant Symbols) ---
#let nag-good = text(weight: "bold")[!]          // $1: Nước cờ hay
#let nag-mistake = text(weight: "bold")[?]       // $2: Nước cờ yếu / sai lầm
#let nag-brilliant = text(weight: "bold")[!!]    // $3: Nước cờ xuất sắc / thiên tài
#let nag-blunder = text(weight: "bold")[??]      // $4: Đại sai lầm
#let nag-interesting = text(weight: "bold")[!?]  // $5: Nước cờ đáng chú ý / sắc bén
#let nag-dubious = text(weight: "bold")[?!]      // $6: Nước cờ đáng ngờ

// Ký hiệu thế trận (Position Evaluation)
#let nag-white-winning = [+-]                   // $18: Trắng thắng chắc
#let nag-black-winning = [-+]                   // $19: Đen thắng chắc
#let nag-white-advantage = [±]                  // $16: Trắng ưu thế rõ rệt
#let nag-black-advantage = [∓]                  // $17: Đen ưu thế rõ rệt
#let nag-white-slight = [⩲]                     // $14: Trắng hơi ưu thế
#let nag-black-slight = [⩱]                     // $15: Đen hơi ưu thế
#let nag-equal = [=]                            // $10: Thế cờ cân bằng
#let nag-unclear = [∞]                          // $13: Thế cờ không rõ ràng / phức tạp
#let nag-compensation = [⯹]                     // $44: Có sự bù đắp thế trận
#let nag-initiative = [⯺]                       // $36: Có quyền chủ động
#let nag-attack = [→]                           // $40: Đang tấn công
#let nag-only-move = [□]                        // $7:  Nước cờ duy nhất
#let nag-zugzwang = [⊙]                         // $22: Rơi vào thế Zugzwang
#let nag-time-trouble = [⨁]                     // $138: Đang thiếu thời gian
#let nag-novelty = [N]                          // Nước cờ mới (Theoretic Novelty)

// --- 3. Hàm chuyển đổi mã NAG ($1 -> Symbol) ---
#let nag(code) = {
  let c = str(code).trim("$")
  if c == "1" [ #nag-good ]
  else if c == "2" [ #nag-mistake ]
  else if c == "3" [ #nag-brilliant ]
  else if c == "4" [ #nag-blunder ]
  else if c == "5" [ #nag-interesting ]
  else if c == "6" [ #nag-dubious ]
  else if c == "7" [ #nag-only-move ]
  else if c == "10" [ #nag-equal ]
  else if c == "13" [ #nag-unclear ]
  else if c == "14" [ #nag-white-slight ]
  else if c == "15" [ #nag-black-slight ]
  else if c == "16" [ #nag-white-advantage ]
  else if c == "17" [ #nag-black-advantage ]
  else if c == "18" [ #nag-white-winning ]
  else if c == "19" [ #nag-black-winning ]
  else if c == "22" [ #nag-zugzwang ]
  else if c == "36" [ #nag-initiative ]
  else if c == "40" [ #nag-attack ]
  else if c == "44" [ #nag-compensation ]
  else if c == "138" [ #nag-time-trouble ]
  else [ \$#c ]
}

// --- 4. Ký hiệu Lượt đi (Turn Indicator Box) ---
#let turn-indicator(turn, size: 8pt) = {
  let is-black = (turn == "b" or turn == "black" or turn == "Đen" or turn == "den")
  if is-black {
    box(
      width: size,
      height: size,
      fill: rgb("#1a202c"),
      radius: 1pt,
      baseline: 10%
    )
  } else {
    box(
      width: size,
      height: size,
      stroke: 0.9pt + rgb("#1a202c"),
      fill: rgb("#ffffff"),
      radius: 1pt,
      baseline: 10%
    )
  }
}

// --- 5. Số mục chú thích dạng khoanh tròn (Informant Circle Numbers) ---
#let note-num(n) = box(
  width: 12pt,
  height: 12pt,
  baseline: 1.5pt,
  stroke: 0.5pt + rgb("#2d3748"),
  radius: 50%,
  fill: rgb("#edf2f7")
)[
  #align(center + horizon)[
    #text(6.5pt, weight: "bold", fill: rgb("#2d3748"))[#n]
  ]
]
