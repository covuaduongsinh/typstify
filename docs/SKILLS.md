# Skill của dự án

Skill là gói hướng dẫn cho AI agent, đặt ở `.claude/skills/<tên>/SKILL.md` (frontmatter `name`, `description`, rồi thân bằng Markdown).

> Lưu ý: `.gitignore` đang loại `.claude/`, nên skill dự án **không được theo dõi bởi git** trừ khi bị ép thêm (`git add -f`) hoặc sửa `.gitignore`. Muốn nhân bản dự án kèm skill, phải sao chép thư mục này tay hoặc bỏ luật loại trừ.

## Danh mục

| Skill | File | Khi dùng |
|---|---|---|
| `chess-pdf-to-typst` | `.claude/skills/chess-pdf-to-typst/SKILL.md` | Người dùng đưa PDF/ảnh scan trang sách cờ vua (kiểu ECO / Chess Informator) và yêu cầu tái tạo thành Typst biên dịch được |

## `chess-pdf-to-typst` — tóm tắt quy trình

Đã kiểm chứng trên trang C58 (`chessbook/C58_414-415.typ`, 2 trang, 43 mục chú thích + 6 biến thể chính).

- **Hai package cốt lõi**: `@preview/board-n-pieces:0.9.0` (bàn cờ từ FEN, `chess-sym` cho ký hiệu quân) và `@preview/staunton:2.0.0` (`game` + `notation` dàn movetext PGN).
- **Các bước**: đọc ảnh bằng vision → dựng khung trang trước (`#counter(page).update(N)` đặt **trước** `#set page`) → kiểm font quân cờ → chép chú thích qua staunton → compile liên tục → render PNG đối chiếu.
- **Chuyển đổi ký hiệu**: SAN chữ cái thường; biến thể dùng ngoặc đơn PGN; đánh giá thế cờ bằng NAG (`$10` =, `$13` ∞, `$14` ⩲, `$15` ⩱, `$16` ±, `$17` ∓, `$18` +−, `$19` −+, `$36` →), luôn có khoảng trắng trước; trích dẫn ván đấu thành comment `{...}`.
- **Thẻ `[SetUp]/[FEN]`**: bắt buộc để `notation()` đánh số nước đúng; FEN không cần đúng thế cờ thật, chỉ cần lượt đi và số nước.
- **Bảng ECO dựng tay** (`#table`): không đi qua staunton nên màu quân `wN/bN...` gõ tay, phải đối chiếu từng ô; dòng trên của cặp là Trắng, dòng dưới là Đen.

### Cạm bẫy quan trọng của skill
- **Không** đưa font kiểu "Chess Merida Unicode" vào `#set text(font: ...)` toàn cục: chúng chiếm mã ∓/± và chỉ lỗi trong PDF thật. Dùng show-rule theo dải mã quân cờ:
  `#show regex("[♔♕♖♗♘♙♚♛♜♝♞♟]"): set text(font: (...))`.
- Kiểm chứng bằng cách rasterize chính file PDF bằng engine độc lập (PyMuPDF), không chỉ `typst compile --format png`.
- `notation()` **không xác thực luật cờ**; không báo "engine đã xác nhận" nếu không gọi `legal-moves`/`apply`.
- Dòng markup bắt đầu bằng `N.` bị hiểu là danh sách đánh số → đưa movetext vào chuỗi `game("...")`.
- `[= ]` trần trong content block render rỗng (thành heading) → dùng `[#"="]`.
- Ký tự nghi vấn do ảnh độ phân giải thấp: giữ bản đọc được và hỏi người dùng, không tự "sửa cho hợp lý".

## Thêm skill mới

1. Tạo `.claude/skills/<tên-kebab>/SKILL.md` với frontmatter:
   ```yaml
   ---
   name: <tên-kebab>
   description: <khi nào dùng; nêu cả các cụm người dùng hay nói>
   ---
   ```
2. Thân skill: quy trình từng bước, bảng chuyển đổi, cạm bẫy, giới hạn phải báo người dùng, ví dụ có thật trong repo.
3. Nếu skill sinh mã Typst cờ vua: nhắc luật ở `CLAUDE.md` (import ở dòng 1, không `#let` mock, escape `\#`).
4. Ghi skill vào bảng danh mục trên và một dòng vào [HISTORY.md](HISTORY.md).
5. Quyết định có theo dõi trong git không (xem lưu ý về `.gitignore` ở đầu file).

## Ý tưởng skill kế tiếp
- Nhập PGN/CSV vào `render-puzzle-collection` (quy trình `web/src/lib/pgn.ts`, `dataImport.ts`).
- Kiểm chessbook sau khi sửa thư viện (bọc `scripts/check-chessbook.sh`).
- Thêm route/setting mới vào server + desktop + web theo mẫu trong [REPLICATION.md](REPLICATION.md).
