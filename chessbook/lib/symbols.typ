// ============================================================================
// TYPSTIFY CHESS ENGINE - SYMBOLS & NOTATIONS MODULE
// Cung cấp hệ thống ký hiệu quân cờ hình tượng (Figurines) và bộ ký hiệu
// đánh giá quốc tế (NAG - Numeric Annotation Glyphs) chuẩn FIDE & Informant.
// ============================================================================

#import "@preview/board-n-pieces:0.9.0": chess-sym, board, fen
#import "theme.typ": *

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
#let nag-symbols = (
  "1": nag-good,
  "2": nag-mistake,
  "3": nag-brilliant,
  "4": nag-blunder,
  "5": nag-interesting,
  "6": nag-dubious,
  "7": nag-only-move,
  "10": nag-equal,
  "13": nag-unclear,
  "14": nag-white-slight,
  "15": nag-black-slight,
  "16": nag-white-advantage,
  "17": nag-black-advantage,
  "18": nag-white-winning,
  "19": nag-black-winning,
  "22": nag-zugzwang,
  "36": nag-initiative,
  "40": nag-attack,
  "44": nag-compensation,
  "138": nag-time-trouble,
)

// Không thêm khoảng trắng quanh ký hiệu: "Nf3!" phải dính liền nước đi.
#let nag(code) = {
  let c = str(code).trim("$")
  nag-symbols.at(c, default: [\$#c])
}

// --- 4. Lượt đi ---
// Một cách hiểu duy nhất cho tham số `turn` của mọi hàm: "b", "black",
// "Đen", "den" (không phân biệt hoa thường) là Đen; còn lại là Trắng.
#let is-black-turn(turn) = {
  if turn == none or turn == auto { return false }
  lower(str(turn)) in ("b", "black", "đen", "den")
}

// Lượt đi đọc từ trường thứ 2 của FEN ("w"/"b"); mặc định Trắng.
#let fen-turn(fen-str) = {
  let fields = str(fen-str).trim().split(" ")
  if fields.len() > 1 and fields.at(1) == "b" { "b" } else { "w" }
}

// Ký hiệu Lượt đi (Turn Indicator Box)
#let turn-indicator(turn, size: 8pt) = {
  let is-black = is-black-turn(turn)
  if is-black {
    box(
      width: size,
      height: size,
      fill: ds-ink,
      radius: 1pt,
      baseline: 10%
    )
  } else {
    box(
      width: size,
      height: size,
      stroke: 0.9pt + ds-ink,
      fill: ds-paper,
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
  stroke: 0.5pt + ds-text,
  radius: 50%,
  fill: ds-brand-soft
)[
  #align(center + horizon)[
    #text(6.5pt, weight: "bold", fill: ds-text)[#n]
  ]
]

// --- 6. Kiểm tra FEN & vẽ bàn cờ dùng chung ---
// Trả về none nếu phần xếp quân của FEN hợp lệ, ngược lại là câu mô tả lỗi.
#let fen-error(fen-str) = {
  let placement = str(fen-str).trim().split(" ").first()
  let ranks = placement.split("/")
  if ranks.len() != 8 {
    return "cần 8 hàng, có " + str(ranks.len())
  }
  for (i, rank) in ranks.enumerate() {
    let n = 0
    for c in rank.clusters() {
      if c in ("1", "2", "3", "4", "5", "6", "7", "8") {
        n += int(c)
      } else if c in ("p", "n", "b", "r", "q", "k", "P", "N", "B", "R", "Q", "K") {
        n += 1
      } else {
        return "ký tự không hợp lệ \"" + c + "\" ở hàng " + str(8 - i)
      }
    }
    if n != 8 {
      return "hàng " + str(8 - i) + " có " + str(n) + " ô (cần 8)"
    }
  }
  none
}

// Bàn cờ in sách. FEN sai hiện khung báo lỗi thay vì làm hỏng cả tài liệu.
#let chess-board(
  fen-str,
  size: 16pt,
  reverse: false,
  numbers: false,
  arrows: (),
  marked: (:),
  dark-fill: ds-board-dark,
  frame: 0.9pt + ds-text,
) = {
  let err = fen-error(fen-str)
  if err != none {
    return box(
      width: size * 8,
      height: size * 8,
      fill: rgb("#FFF5F5"),
      stroke: 0.8pt + rgb("#C62828"),
      inset: 6pt,
    )[
      #align(center + horizon)[
        #text(7.5pt, fill: rgb("#C62828"))[*FEN không hợp lệ:* #err]
        #v(3pt)
        #text(6.5pt, fill: ds-muted)[#raw(str(fen-str))]
      ]
    ]
  }
  box(stroke: frame, fill: ds-paper, inset: 0pt)[
    #board(
      fen(fen-str),
      square-size: size,
      reverse: reverse,
      display-numbers: numbers,
      white-square-fill: ds-board-light,
      black-square-fill: dark-fill,
      arrows: arrows,
      marked-squares: marked,
    )
  ]
}
