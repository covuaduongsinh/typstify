# Thư viện xuất bản cờ vua `@local/chessbook:0.1.0`

Thư viện Typst nằm ở `chessbook/lib/`, phụ thuộc `@preview/board-n-pieces:0.9.0` (vẽ bàn cờ, đọc FEN, ký hiệu quân).
Nhận diện thương hiệu Dương Sinh (navy `#2B3990` + gold). Mọi tài liệu phải mở đầu bằng:

```typst
#import "@local/chessbook:0.1.0": *
```

Dòng này phải ở **dòng 1**. Không tự viết `#let` giả lập các hàm của thư viện. Ký tự `#` trong content block phải viết `\#`.
Trong repo, có thể import trực tiếp bằng `#import "lib/lib.typ": *` (từ thư mục `chessbook/`).

## 1. Cấu trúc file

| File | Vai trò |
|---|---|
| `lib/lib.typ` | Điểm vào: re-export mọi module; ba hàm khởi tạo tài liệu |
| `lib/theme.typ` | Bảng màu `ds-*` và font (`font-serif` = Noto Serif, `font-sans` = Roboto). Đổi bộ nhận diện chỉ cần sửa file này |
| `lib/symbols.typ` | Quân cờ hình tượng, NAG, lượt đi, kiểm tra FEN, `chess-board` |
| `lib/puzzle.typ` | Thẻ bài tập, lưới, đáp án, tuyển tập tự phân trang, CSV→bài tập |
| `lib/eco.typ` | Khai cuộc ECO / Informator |
| `lib/magazine.typ` | Tạp chí, bài viết, ván đấu |
| `lib/courseware.typ` | Giáo trình, bài giảng HLV |
| `templates/` | Mẫu hoàn chỉnh: `puzzle_book`, `chess_magazine`, `chess_courseware`, `eco_encyclopedia`, `chess_theory_fen_samples` |
| `demo_collection/` | 4 bản demo đầy đủ (tactics, ECO, magazine, courseware) |
| `data/` | `sample_puzzles_24.csv`, `sample_puzzles_24.json` |
| `demobook.typ`, `test.typ`, `C58_414-415.typ` | Demo/thử nghiệm và trang C58 dựng lại từ PDF Informator |

Kiểm tra biên dịch mọi demo/template: `scripts/check-chessbook.sh`. Cài gói vào máy: `scripts/install-chessbook.sh` / `.ps1`.

## 2. Khởi tạo tài liệu (`lib.typ`)

| Hàm | Tham số chính | Ghi chú |
|---|---|---|
| `chess-book-init(title, subtitle, author, paper-size, font, body)` | `paper-size`: `"a5"` (mặc định), `"a4"`, `"16x24"`/`"16x24cm"`, hoặc bất kỳ khổ Typst / `(width:, height:)` | Chữ 9pt, `lang: "vi"`, căn đều. Header chẵn/lẻ: trang lẻ hiện tiêu đề bên trái + số trang bên phải; trang chẵn ngược lại (phụ đề bên phải). Khổ 16x24 dùng lề trong/ngoài 1.8/1.3cm |
| `chess-worksheet-init(title, subtitle, author, date, paper-size, font, body)` | mặc định A4, `font-sans` | Header có tên phiếu + ngày + "Trang N", chữ 8.5pt |
| `chess-magazine-init(magazine-title, issue, font, body)` | A4 | Header 3 cột (tên tạp chí / số / trang) |

Dùng dạng `#show: chess-book-init.with(title: "...", paper-size: "16x24")`.

## 3. Ký hiệu và bàn cờ (`symbols.typ`)

- **Quân cờ:** `wK wQ wR wB wN wP bK bQ bR bB bN bP` (lấy từ `chess-sym` của board-n-pieces).
- **NAG:** `nag("1")` → `!`, `"2"` `?`, `"3"` `!!`, `"4"` `??`, `"5"` `!?`, `"6"` `?!`, `"7"` □, `"10"` =, `"13"` ∞, `"14"` ⩲, `"15"` ⩱, `"16"` ±, `"17"` ∓, `"18"` +-, `"19"` -+, `"22"` ⊙, `"36"` ⯺, `"40"` →, `"44"` ⯹, `"138"` ⨁. Mã lạ trả về `$<mã>`. Chấp nhận tiền tố `$` (`nag("$1")`). Không thêm khoảng trắng quanh ký hiệu để `Nf3!` dính liền.
- **Lượt đi:**
  - `is-black-turn(turn)`: `"b"`, `"black"`, `"Đen"`, `"den"` (không phân biệt hoa thường) là Đen; `none`/`auto`/giá trị khác là Trắng.
  - `fen-turn(fen)`: đọc trường thứ 2 của FEN, mặc định `"w"`.
  - `turn-indicator(turn, size: 8pt)`: ô vuông đặc (Đen) hoặc rỗng (Trắng).
- `note-num(n)`: số chú thích khoanh tròn kiểu Informator.
- `fen-error(fen)`: trả `none` nếu hợp lệ, ngược lại chuỗi mô tả lỗi.
- `normalize-arrows(arrs)`: chuẩn hoá mũi tên.
- `chess-board(fen, size: 16pt, reverse: false, numbers: false, arrows: (), marked: (:), dark-fill, frame)`.

### Thuật toán
- **Kiểm FEN (`fen-error`):** tách phần xếp quân theo `/`; phải đúng 8 hàng; mỗi hàng cộng chữ số (1–8) và ký tự quân (`pnbrqkPNBRQK`), phải đúng 8 ô. Báo lỗi có số hàng (đánh số 8→1). Chỉ kiểm phần xếp quân, không kiểm quyền nhập thành/en passant/số nước.
- **Chống vỡ tài liệu:** `chess-board` gặp FEN sai sẽ vẽ khung đỏ "FEN không hợp lệ: …" đúng kích thước `size*8` thay vì làm lỗi biên dịch.
- **Chuẩn hoá mũi tên:** chuỗi dạng `"c4-f7"`, `"c4->f7"`, `"c4 f7"`, `"c4f7"` được bỏ `->`, `-`, khoảng trắng; nếu còn đúng 4 ký tự thì dùng (`"c4f7"`), ngược lại giữ nguyên. Phần tử không phải chuỗi giữ nguyên; tham số không phải mảng thành `()`.
- **Hai kiểu viền:** `numbers: false` → khung đơn sắc ôm 8x8 ô (chuẩn in sách); `numbers: true` → bàn cờ kèm toạ độ.
- Màu ô: sáng `#FFFFFF`, tối `#D9DDEF` để in đơn sắc vẫn rõ.

## 4. Bài tập và chiến thuật (`puzzle.typ`)

| Hàm | Mô tả |
|---|---|
| `puzzle-card(fen, number, title, turn, to-move, difficulty, hint, size, compact, solution, arrows)` | Một bài tập. Lượt đi: `turn` > `to-move` (tên cũ, vẫn nhận) > `fen-turn(fen)`. Lượt Đen thì bàn cờ tự lật (`reverse`). Có `solution` thì tự ghi vào state |
| `difficulty-stars(level)` | 1–5 sao (kẹp giá trị) |
| `puzzle-grid-a4(puzzles:, gutter:, size: 13.5pt, compact: true)` | Lưới 3x4 = 12 bài |
| `puzzle-grid-16x24(puzzles:, gutter:, size: 15pt, compact: false)` | Lưới 2x3 = 6 bài |
| `upside-down-solutions(dict/array)` | Dải đáp án xoay 180° ở chân trang, đường kẻ đứt phía trên, đẩy xuống đáy bằng `v(1fr)` |
| `render-puzzle-solutions()` | Trang "Đáp án & Lời giải Chi tiết" 2 cột từ state |
| `csv-to-puzzles(csv-data)` | CSV → mảng bài tập |
| `render-puzzle-collection(data, layout, per-page, start-number, show-upside-down, upside-down, upside-down-solutions, render-appendix-at-end, page-title)` | Tự phân trang và xuất tuyển tập |

> **Lưu ý về `CLAUDE.md`:** bản hướng dẫn cũ ghi `title-prefix`, `show-page-solutions`, `show-end-appendix`. Tên tham số **thật** trong mã là `page-title`, `show-upside-down` (hoặc `upside-down` / `upside-down-solutions`), `render-appendix-at-end`. Dùng theo tên trong bảng trên.

### Thuật toán
- **Thu thập đáp án toàn cục:** `state("typstify-puzzle-solutions")`. Mỗi `puzzle-card` có `solution` thì `update` thêm `(number, title, to-move, solution)`. `render-puzzle-solutions` đọc `state.final()` trong `context`, nên đáp án tự gom về cuối dù thẻ nằm ở đâu.
- **Phân trang `render-puzzle-collection`:**
  1. Chuẩn hoá `data`: mảng mảng → `csv-to-puzzles`; mảng dictionary giữ nguyên.
  2. Số bài mỗi trang: `per-page` nếu có; `16x24`/`16x24-2x3` → 6; `a5`/`a5-2x2` → 4; còn lại → 12.
  3. `num-pages = ceil(tổng / số-bài-mỗi-trang)`. Với mỗi trang, cắt lát, gán số thứ tự tăng liên tục từ `start-number` (chuỗi thuần được coi là FEN), gom đáp án của trang thành dict.
  4. Lưới: layout 16x24 dùng `puzzle-grid-16x24`, còn lại (kể cả a5) dùng `puzzle-grid-a4`. Lưu ý layout `a5` chỉ ảnh hưởng số bài/trang, không đổi lưới.
  5. In dải đáp án úp ngược nếu bật và trang có đáp án. `pagebreak()` giữa các trang, và thêm một `pagebreak()` cuối nếu có phụ lục.
  6. Cuối cùng gọi `render-puzzle-solutions()` nếu `render-appendix-at-end`.
- **`csv-to-puzzles`:**
  - Dạng dictionary (`csv(..., row-type: dictionary)`): lấy các cột `fen, title, turn, difficulty, hint, solution`; `difficulty` ép sang int.
  - Dạng mảng mảng: nếu dòng đầu chứa `fen`/`FEN` thì coi là header và bỏ qua; thứ tự cột `fen, title, turn, difficulty, hint, solution`; bỏ dòng có FEN rỗng; độ khó mặc định 1.

Ví dụ:

```typst
#import "@local/chessbook:0.1.0": *
#let puzzles = csv-to-puzzles(csv("data/sample_puzzles_24.csv"))
#render-puzzle-collection(puzzles, layout: "a4-3x4", page-title: "Chiến thuật")
```

## 5. Khai cuộc ECO / Informator (`eco.typ`)

- `eco-header(code, name, subname, intro-moves)`: khối mã ECO nền đậm + tên khai cuộc + biến thể + nước dẫn nhập.
- `eco-table(columns-header:, rows:)`: bảng nước đi kiểu Informator. Cột đầu "No." rộng 22pt, các cột còn lại `1fr`; hàng lẻ nền trắng, hàng chẵn nền nhạt; `rows` được `flatten()` nên truyền từng ô nối liền.
- `opening-diagram-box(fen, title, turn, to-move, eval-text, caption, size: 14pt, arrows)`: diagram cạnh khối chú thích, có "Đánh giá: …".

## 6. Tạp chí (`magazine.typ`)

- `game-header(white, white-title, white-elo, white-fed, black, …, event, site, date, round, result, eco, opening)`: thẻ ván đấu. Trường rỗng (PGN thiếu) bị lọc bỏ, không in `( , )`; `round` rỗng thì không in "V…".
- `column-diagram(fen, move-num, caption, turn, to-move, size: 13.5pt, arrows)`: bàn cờ nhỏ trong một cột.
- `chess-quote(author:)[…]`: hộp trích dẫn, nhận nội dung qua `content:` hoặc tham số vị trí.

## 7. Giáo trình (`courseware.typ`)

- `lesson-header(lesson-num, title, level, duration, objective)`.
- `concept-box(title:)[…]`.
- `teaching-diagram(fen, title, turn, to-move, size: 18pt, numbers, arrows, caption)`: bàn cờ lớn; khi `numbers: true` khối rộng `size*10` để chừa toạ độ.
- `practice-question(number, question, choices, answer)`: lưới 2 cột cho lựa chọn.
- `instructor-note[…]`: khung cam cho HLV (`ds-warn`).

## 8. Quy ước chung của tham số

- `turn` và `to-move` cùng nghĩa; `to-move` ưu tiên khi được truyền cho `opening-diagram-box`, `column-diagram`, `teaching-diagram` (còn `puzzle-card` ưu tiên `turn`).
- Lượt Đen luôn lật bàn cờ (`reverse: is-black-turn(side)`).
- `arrows` nhận mảng chuỗi (`("e2e4", "g1f3")`) hoặc các dạng ở mục 3.
- Font: `Noto Serif` (sách) và `Roboto` (tạp chí/phiếu); thiếu font thì Typst dùng font dự phòng. Trên server web, image Docker đã cài các font này cùng `board-n-pieces`.

## 9. Mở rộng thư viện

1. Thêm hàm vào module phù hợp (hoặc module mới `lib/<ten>.typ`), import `theme.typ` và `symbols.typ` thay vì tự định nghĩa màu.
2. Nếu là module mới, thêm `#import "<ten>.typ": *` vào `lib/lib.typ`.
3. Thêm ví dụ vào `demo_collection/` hoặc `templates/` và chạy `scripts/check-chessbook.sh`.
4. Cập nhật file này và `CLAUDE.md`/`AGENTS.md` (hai file phải giống nhau).
5. Bump phiên bản trong `chessbook/lib/typst.toml` nếu thay đổi API.
