# HƯỚNG DẪN SỬ DỤNG TYPSTIFY CHESS PUBLISHING STUDIO
### Hệ Thống Soạn Thảo & Xuất Bản Sách Báo, Tạp Chí, Tài Liệu Cờ Vua Chuyên Nghiệp

---

## 📖 MỤC LỤC
1. [Tổng Quan Hệ Thống](#1-tổng-quan-hệ-thống)
2. [Hướng Dẫn 4 Dạng Ấn Phẩm Cờ Vua Chuẩn (A, B, C, D)](#2-hướng-dẫn-4-dạng-ấn-phẩm-cờ-vua-chuẩn)
   - [Dạng A: Sách Bài Tập Chiến Thuật (Tactics / Puzzles)](#dạng-a-sách-bài-tập-chiến-thuật)
   - [Dạng B: Bách Khoa Khai Cuộc ECO (Encyclopedia / Informant)](#dạng-b-bách-khoa-khai-cuộc-eco)
   - [Dạng C: Tạp Chí Cờ Vua 2 Cột (Magazine & Game Commentary)](#dạng-c-tạp-chí-cờ-vua-2-cột)
   - [Dạng D: Giáo Trình & Bài Giảng Huấn Luyện (Courseware)](#dạng-d-giáo-trình--bài-giảng-huấn-luyện)
3. [Hướng Dẫn Sử Dụng Bộ Công Cụ Web Editor](#3-hướng-dẫn-sử-dụng-bộ-công-cụ-web-editor)
   - [Thanh Công Cụ Ký Hiệu Cờ Vua (Chess Toolbar)](#thanh-công-cụ-ký-hiệu-cờ-vua)
   - [Bàn Cờ Trực Quan Xếp Thế Cờ (Visual Chessboard)](#bàn-cờ-trực-quan-xếp-thế-cờ)
   - [Trình Nhập & Chuyển Đổi Ván Cờ PGN (PGN Importer)](#trình-nhập--chuyển-đổi-ván-cờ-pgn)
4. [Hệ Thống Ký Hiệu Quân Cờ & Mã Đánh Giá NAG](#4-hệ-thống-ký-hiệu-quân-cờ--mã-đánh-giá-nag)
5. [Hướng Dẫn Chạy & Triển Khai Hệ Thống (Run & Deploy)](#5-hướng-dẫn-chạy--triển-khai-hệ-thống)

---

## 1. Tổng Quan Hệ Thống

**Typstify Chess Publishing Studio** là giải pháp xuất bản cờ vua toàn diện được xây dựng trên nền tảng **Typst** kết hợp với giao diện **Web Studio hiện đại**. Hệ thống mang đến trải nghiệm dàn trang cờ vua đỉnh cao:
- **Tốc độ xuất bản siêu tốc**: Biên dịch tài liệu hàng trăm trang với hàng trăm hình bàn cờ trong chưa đầy 1-2 giây.
- **Chất lượng in ấn chuẩn Vector**: Bàn cờ, quân cờ và ký hiệu quốc tế sắc nét tuyệt đối ở mọi độ phân giải (chuẩn in offset, nhà xuất bản, Amazon KDP).
- **Bộ công cụ trực quan**: Không cần nhớ mã FEN, có bàn cờ kéo thả xếp quân và trình nhập PGN tự động sinh mã.

---

## 2. Hướng Dẫn 4 Dạng Ấn Phẩm Cờ Vua Chuẩn

### Dạng A: Sách Bài Tập Chiến Thuật (Tactics & Puzzle Books)
Dành cho sách câu đố, cẩm nang luyện tập chiến thuật với bố cục lưới bàn cờ A5 cân đối.

**Template mẫu**: `chessbook/templates/puzzle_book.typ`

**Cách sử dụng:**
```typst
#import "lib/lib.typ": *

#show: doc => chess-book-init(
  title: "100 THẾ CỜ CHIẾN THUẬT KINH ĐIỂN",
  subtitle: "CHƯƠNG 1: ĐÒN ĐÁNH ĐÔI",
  paper-size: "a5",
  doc
)

// Dàn lưới 4 câu đố trên 1 trang A5 (2x2)
#grid(
  columns: (1fr, 1fr),
  row-gutter: 14pt,
  column-gutter: 10pt,
  [
    #puzzle-card(
      "r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
      number: 1,
      title: "Khai cuộc Ý",
      to-move: "w",            // "w" (Trắng) hoặc "b" (Đen)
      difficulty: 2,           // 1 đến 5 sao
      hint: "Gỡ ghim và phản công",
      solution: "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân."
    )
  ],
  // ... các câu đố tiếp theo
)

// In dải đáp án úp ngược 180 độ ở chân trang
#upside-down-solutions((
  "1": "1. dxe4 Bxc3+ 2. bxc3 Trắng hơn quân.",
  "2": "1. Qe2 Qxe2+ 2. Bxe2 Trắng phát triển tốt."
))

// Hoặc tự động gom toàn bộ đáp án về trang cuối sách
#render-puzzle-solutions()
```

---

### Dạng B: Bách Khoa Khai Cuộc ECO (Encyclopedia / Informant)
Tái hiện chuẩn bách khoa toàn thư của *Chess Informant (Šahovski Informator)* và *ECO*.

**Template mẫu**: `chessbook/templates/eco_encyclopedia.typ`

**Cách sử dụng:**
```typst
#import "lib/lib.typ": *

// 1. Tiêu đề Khai cuộc ECO lớn
#eco-header(
  code: "C 58",
  name: "Phòng thủ Hai Mã (Two Knights Defense)",
  subname: "Biến thể Polerio - Bogoljubow (5...Na5)",
  intro-moves: "1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5"
)

// 2. Diagram thế cờ then chốt
#opening-diagram-box(
  "r1bqkb1r/ppp2ppp/5n2/n2P4/2B5/5N2/PPPP1PPP/RNBQK2R w KQkq - 1 6",
  title: "Thế cờ sau 5...Na5",
  turn: "w",
  eval-text: "⩲ (Trắng hơi ưu thế)",
  caption: "Trắng đứng trước lựa chọn 6.Bb5+ hoặc 6.d3."
)

// 3. Bảng ma trận nước đi phân nhánh
#eco-table(
  columns-header: ("6", "7", "8", "9", "10", "Đánh giá"),
  rows: (
    ([1], [Bb5+ #note-num(1)], [c6], [dxc6], [bxc6], [#nag("14")]),
    ([2], [d3], [h6], [Nf3], [e4], [#nag("10")])
  )
)

// 4. Chú thích phân tích với số khoanh tròn
#note-num(1) *6. Bb5+ c6 7. dxc6 bxc6 8. Bd3:* Nước cờ phòng ngự chắc chắn do Steinitz đề xuất.
```

---

### Dạng C: Tạp Chí Cờ Vua 2 Cột (Magazine & Game Commentary)
Dàn trang tạp chí khổ A4 2 cột chuyên nghiệp phong cách *New In Chess*.

**Template mẫu**: `chessbook/templates/chess_magazine.typ`

**Cách sử dụng:**
```typst
#import "lib/lib.typ": *

#show: doc => chess-magazine-init(
  magazine-title: "TẠP CHÍ CỜ VUA VIỆT NAM",
  issue: "SỐ 09 - 2026",
  doc
)

#columns(2, gutter: 14pt)[
  #chess-quote(
    author: "Garry Kasparov",
    "Chiến thuật là biết phải làm gì khi có điều gì đó để làm..."
  )

  // Thẻ thông tin ván đấu cao cấp
  #game-header(
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
    opening: "Phòng thủ Hai Mã"
  )

  *1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5 6. Bb5+ c6 7. dxc6 bxc6 8. Bd3!*

  // Bàn cờ nhỏ gọn lồng trong 1 cột bài viết
  #column-diagram(
    "r1bqkb1r/p4ppp/2p2n2/n3p3/8/3B1N2/PPPP1PPP/RNBQK2R b KQkq - 1 8",
    move-num: "Thế cờ sau 8. Bd3!",
    turn: "b",
    caption: "Đen đứng trước nhiều lựa chọn phản công."
  )
]
```

---

### Dạng D: Giáo Trình & Bài Giảng Huấn Luyện (Courseware)
Chuyên dùng cho trung tâm, trường học cờ vua, câu lạc bộ và bài giảng HLV.

**Template mẫu**: `chessbook/templates/chess_courseware.typ`

**Cách sử dụng:**
```typst
#import "lib/lib.typ": *

// 1. Tiêu đề bài học có Mục tiêu, Cấp độ, Thời lượng
#lesson-header(
  lesson-num: 4,
  title: "CHIẾN THUẬT: ĐÒN GHIM QUÂN (THE PIN)",
  level: "Cấp độ: Sơ cấp nâng cao",
  duration: "Thời lượng: 60 phút",
  objective: "Học viên phân biệt được đòn ghim tuyệt đối và tương đối."
)

// 2. Hộp khái niệm lý thuyết nổi bật
#concept-box(title: "1. Khái niệm Đòn Ghim Tuyệt đối")[
  Đòn ghim tuyệt đối xảy ra khi một quân cờ đứng chắn giữa quân tấn công và quân *Vua*...
]

// 3. Diagram giảng dạy có tọa độ bàn cờ
#teaching-diagram(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5",
  title: "Ví dụ 1: Tượng Đen ghim Mã Trắng vào Vua",
  turn: "b",
  caption: "Tượng đen ở b4 ghim cứng Mã c3 vào Vua e1."
)

// 4. Ghi chú cho Huấn luyện viên
#instructor-note[
  Nhắc nhở học viên luôn kiểm tra Vua trước khi di chuyển quân bị ghim.
]

// 5. Câu hỏi trắc nghiệm trên lớp
#practice-question(
  number: 1,
  question: "Quân cờ nào có khả năng thực hiện đòn ghim?",
  choices: (
    "A. Xe, Tượng, Hậu",
    "B. Mã và Tốt",
    "C. Vua",
    "D. Tất cả các quân"
  ),
  answer: "A - Xe, Tượng và Hậu."
)
```

---

## 3. Hướng Dẫn Sử Dụng Bộ Công Cụ Web Editor

Giao diện Web của Typstify được tích hợp bộ công cụ cờ vua thông minh xuất hiện tự động trên đỉnh trình soạn thảo khi bạn mở file `.typ`:

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│ [♟️ Xếp Bàn Cờ] [📜 Nhập PGN] | ♔ ♕ ♖ ♗ ♘ ♙ ♚ ♛ ♜ ♝ ♞ ♟ | ! ? !! ?? !? ?! ± ∓ ⩲ ⩱ = ∞ □ | [📐 Chèn Mẫu ▾] │
└─────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Thanh Công Cụ Ký Hiệu Cờ Vua (Chess Toolbar)
- **Palette Quân cờ**: Bấm vào biểu tượng quân cờ để chèn trực tiếp mã ký hiệu hình tượng (`#wK`, `#wQ`, `#bN`, `#bP`...) vào vị trí con trỏ.
- **Palette NAG**: Bấm vào các ký hiệu `!`, `?`, `±`, `∓`, `⩲`, `⩱`, `=`, `∞`, `□` để chèn đánh giá thế trận chuẩn quốc tế.
- **Menu Chèn Mẫu (📐)**: Chèn nhanh khung mẫu hoàn chỉnh của 4 dạng A, B, C, D chỉ với 1 click chuột.

### Bàn Cờ Trực Quan Xếp Thế Cờ (Visual Chessboard)
1. Bấm nút **"♟️ Xếp Bàn Cờ"** trên thanh công cụ.
2. Cửa sổ bàn cờ 8x8 trực quan xuất hiện:
   - Chọn quân cờ trên thanh công cụ palette (Vua, Hậu, Xe, Tượng, Mã, Tốt).
   - Click vào ô cờ để đặt quân; chọn 🧹 để xóa quân cờ.
   - Bấm **"🔄 Đảo góc nhìn"** để xoay bàn cờ theo hướng quân Đen.
   - Bấm **"🏁 Ván cờ đầu"** để xếp lại vị trí xuất phát hoặc **"🗑️ Xóa trắng"** để xóa toàn bộ bàn cờ.
3. Chọn định dạng xuất bản mong muốn (**Dạng A, B, C hoặc D**), nhập tiêu đề, độ khó, lời giải.
4. Bấm **"✨ Chèn vào Tài Liệu"**: Hệ thống sẽ tự động sinh mã Typst và chèn thẳng vào vị trí con trỏ trong Editor!

### Trình Nhập & Chuyển Đổi Ván Cờ PGN (PGN Importer)
1. Bấm nút **"📜 Nhập PGN"** trên thanh công cụ.
2. Dán nội dung ván cờ PGN từ Chess.com / Lichess hoặc bấm **"📂 Tải file PGN"** từ máy tính.
3. Bấm **"✨ Chuyển đổi & Chèn vào Tài Liệu"**: Hệ thống tự động bóc tách tên kỳ thủ, Elo, ngày đấu, mã ECO và định dạng danh sách nước đi thành code Typst hoàn chỉnh.

---

## 4. Hệ Thống Ký Hiệu Quân Cờ & Mã Đánh Giá NAG

| Mã NAG | Ký Hiệu | Ý Nghĩa Chuyên Môn | Code Typst |
|:---:|:---:|---|:---:|
| `$1` | **!** | Nước cờ hay | `!` hoặc `#nag("1")` |
| `$2` | **?** | Nước cờ yếu / sai sót | `?` hoặc `#nag("2")` |
| `$3` | **!!** | Nước cờ xuất sắc / thiên tài | `!!` hoặc `#nag("3")` |
| `$4` | **??** | Đại sai lầm | `??` hoặc `#nag("4")` |
| `$5` | **!?** | Nước cờ đáng chú ý / sắc bén | `!?` hoặc `#nag("5")` |
| `$6` | **?!** | Nước cờ đáng ngờ | `?!` hoặc `#nag("6")` |
| `$7` | **□** | Nước cờ duy nhất | `#nag("7")` |
| `$10` | **=** | Thế cờ cân bằng | `#nag("10")` |
| `$13` | **∞** | Thế cờ không rõ ràng / phức tạp | `#nag("13")` |
| `$14` | **⩲** | Trắng hơi ưu thế | `#nag("14")` |
| `$15` | **⩱** | Đen hơi ưu thế | `#nag("15")` |
| `$16` | **±** | Trắng ưu thế rõ rệt | `#nag("16")` |
| `$17` | **∓** | Đen ưu thế rõ rệt | `#nag("17")` |
| `$18` | **+-** | Trắng thắng chắc | `#nag("18")` |
| `$19` | **-+** | Đen thắng chắc | `#nag("19")` |
| `$44` | **⯹** | Có sự bù đắp thế trận | `#nag("44")` |

---

## 5. Hướng Dẫn Chạy & Triển Khai Hệ Thống

### 1. Biên Dịch Trực Tiếp File PDF Qua Dòng Lệnh
```bash
# Biên dịch Sách bài tập A5
typst compile --root chessbook chessbook/templates/puzzle_book.typ chessbook/output/puzzle_book.pdf

# Biên dịch Bách khoa ECO C58
typst compile --root chessbook chessbook/templates/eco_encyclopedia.typ chessbook/output/eco_encyclopedia.pdf

# Biên dịch Tạp chí Cờ Vua A4
typst compile --root chessbook chessbook/templates/chess_magazine.typ chessbook/output/chess_magazine.pdf

# Biên dịch Giáo trình bài giảng Huấn luyện
typst compile --root chessbook chessbook/templates/chess_courseware.typ chessbook/output/chess_courseware.pdf
```

### 2. Chạy Web Studio Cục Bộ (Local Development)
```bash
# Khởi động Backend Server
go run ./cmd/typstify-server

# Khởi động Frontend Web App (trong thư mục web/)
cd web
npm install
npm run dev
```
Truy cập trình duyệt tại: `http://localhost:5173`

### 3. Build & Triển Khai Web App Cho Production
```bash
# Build mã nguồn Web thành static assets
cd web
npm run build

# Chạy server với Docker
docker-compose up -d --build
```
Hệ thống sẽ chạy container backend kèm Caddy reverse proxy sẵn sàng cho môi trường production / VPS Dokploy.
