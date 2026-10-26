# Kế hoạch chi tiết — Giai đoạn 6: Hoàn thiện editor

> Thuộc [plan_typst_app_parity_2026-10.md](plan_typst_app_parity_2026-10.md). Ngày lập: 2026-10-25.

## Mục tiêu

Bù những tiện ích soạn thảo hằng ngày mà typst.app có: định dạng code, chèn ký hiệu, dán ảnh và tải tệp lên, kiểm tra chính tả tiếng Việt, chế độ Vim, trình chiếu. Mọi tính năng chạy được cả khi đang cộng tác (Giai đoạn 3), vì chúng chỉ tạo ra các thay đổi CodeMirror bình thường.

## Hạng mục (theo thứ tự làm)

| # | Tính năng | Cách làm |
|---|---|---|
| 1 | **Định dạng** (Shift+Alt+F, nút trên thanh công cụ) | tinymist với `formatterMode: "typstyle"`; `Client.Formatting` (Go) → tin `format` qua `/ws/lsp` → các `TextEdit` được áp vào editor |
| 2 | **Dán ảnh, kéo thả tệp** | `POST /api/workspace/upload?dir=&name=` (editor; body nhị phân; tên được làm sạch và tự đánh số khi trùng; giới hạn 25 MB). Dán hoặc thả ảnh vào editor thì lưu vào `assets/` rồi chèn `#image("assets/…")`. Thả tệp vào cây thư mục thì tải lên thư mục đó |
| 3 | **Chế độ Vim** | `@replit/codemirror-vim`, bật/tắt trong Cài đặt (lưu cục bộ trên trình duyệt) |
| 4 | **Bảng ký hiệu** | Danh sách `sym.*` lấy từ tinymist (`tinymist.getResources` `/symbols`), tìm theo tên hoặc ký tự, bấm để chèn `#sym.x` (hoặc `sym.x` trong code) |
| 5 | **Kiểm tra chính tả tiếng Việt** | `nspell` + `dictionary-vi` trong Web Worker; chỉ kiểm chữ thuần (bỏ qua code, math, chuỗi, nhãn, comment theo cây cú pháp Lezer); gạch chân và gợi ý sửa; bật/tắt trong Cài đặt |
| 6 | **Trình chiếu** | Từ preview "Tức thì": xem toàn màn hình theo trang, chuyển trang bằng phím mũi tên hoặc PageUp/PageDown |

## Kiểm chứng

- Go: `Client.Formatting` với tinymist thật (bỏ qua nếu không có tinymist); upload (quyền, tên xấu, trùng tên, vượt giới hạn).
- Web (vitest): áp TextEdit, đặt tên ảnh dán, lọc từ cần kiểm chính tả theo cây cú pháp, tìm ký hiệu.
- Chạy thật: định dạng một tài liệu chessbook, dán ảnh chụp màn hình, bật Vim, chèn ký hiệu, thấy gạch chân từ sai chính tả, trình chiếu.

## Kết quả (2026-10-25)

Đã làm đủ 6 hạng mục, riêng hạng mục 5 đổi cách làm: `dictionary-vi` dùng giấy phép GPL (tệp LICENSE ghi GPLv3, `package.json` ghi GPL-2.0), không đóng gói được vào Typstify (Apache 2.0). Thay vào đó dùng spellcheck của trình duyệt, giới hạn vào các nút `Text` của cây cú pháp (`web/src/lib/spellcheck.ts`). Nếu chủ dự án chấp nhận phần GPL (ví dụ tải từ điển riêng, không gộp vào bundle), có thể quay lại phương án `nspell`. Chi tiết và việc còn mở: `docs/HISTORY.md`, mục Giai đoạn 6.
