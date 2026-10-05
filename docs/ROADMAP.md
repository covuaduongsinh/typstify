# Lộ trình và khoảng trống

Nguồn: khảo sát mã nguồn 2026-09-29 và `docs/plans/plan_codebase_review_2026-09.md`. Mức ưu tiên là đề xuất, chờ chủ dự án chốt. Mỗi mục có "cách kiểm" để biết đã xong.

## A. Khoảng trống tính năng đã xác nhận trong mã

| # | Khoảng trống | Vị trí | Đề xuất | Cách kiểm |
|---|---|---|---|---|
| A1 | Preview SVG trên web chưa có click-to-source (chỉ iframe tinymist có) | `web/src/components/PreviewPane.tsx`, `server/preview_render.go` | Thêm bản đồ nguồn (typst `query`/span) hoặc chuyển chế độ tinymist làm mặc định khi cần nhảy về nguồn | Click vào chữ trong preview → editor nhảy đúng dòng |
| A2 | Desktop chưa dùng anchors heading thật (chỉ web) | `typst.QueryHeadingPages` chỉ được `server/preview_anchors.go` gọi | Cân nhắc dùng cho desktop nếu tinymist scroll chưa đủ chính xác | So vị trí cuộn với tài liệu nhiều trang |
| A3 | (Đã hoàn thành) i18n desktop tiếng Việt | `i18n/translations/locales/vi-VN/` | Thêm `vi-VN/messages.gotext.json`, chuẩn hoá chuỗi nguồn & catalog | Đổi ngôn ngữ trong Settings hiện tiếng Việt đầy đủ |
| A4 | (Đã hoàn thành) Tài liệu i18n tiếng Việt | `i18n/`, `docs/architecture_and_guide.md` | Đã hỗ trợ tiếng Việt đầy đủ và khớp với mã nguồn | Tài liệu khớp mã |
| A5 | Agent từ xa: title, usage, available commands không đi qua WebSocket | `agent/remote_session.go` | Mở rộng giao thức `/ws/agent` | Phiên remote hiện đủ thông tin như local |
| A6 | Không có Justfile/Makefile | gốc repo | Thêm target build/test/run thống nhất | `just test` chạy đủ Go + web |
| A7 | (Đã hoàn thành) `docs/CHESSBOOK.md` | `docs/` | Đã viết đầy đủ tài liệu API, tham số, thuật toán và ví dụ | Mục lục `docs/README.md` đã có |

## B. Vệ sinh repo
- `.claude/` (kể cả skill dự án) và `demo/` bị `.gitignore` loại → skill không được theo dõi. Quyết định: bỏ luật loại cho `.claude/skills/` hay chấp nhận sao chép tay (xem [SKILLS.md](SKILLS.md)).
- `*.exe` ở gốc đã bị `.gitignore` loại (`typstify.exe`, `typstify-server.exe`, `agy_acp_server.exe`, `antigravity_bridge.exe`): không cần thêm luật, chỉ cần không commit ép.
- `chessbook/` chứa PDF/ảnh đầu ra; `chessbook/*.pdf` bị loại, kiểm lại file `C58_414-415.pdf` và file "Extracted pages…pdf" có nằm trong git không.
- Font lớn không dùng (rà soát 09/2026 ghi ~26 MB: `NotoColorEmoji`, `NotoSansSC`), code chết `cmd/agy_acp_bridge`, file trùng `chessbook/Extracted pages C58 …typ` — **đối chiếu hiện trạng trước khi xoá**, vì kế hoạch rà soát có thể đã được xử lý một phần.

## C. Đối chiếu kế hoạch rà soát 09/2026
Kế hoạch chấm: kiến trúc 8/10, bảo mật server 4/10, đúng/ổn định 5/10, cờ vua trên server 3/10, kiểm thử/CI 2/10, deploy 7/10, tài liệu 5/10, với 22 phát hiện chia 6 giai đoạn. Theo git log, nhiều mục đã được xử lý sau đó:
- Đã thấy commit tương ứng: đóng WebSocket preview không xác thực và làm cứng auth `8ac9b5e`; deadlock/leak/race agent, compile có giới hạn, symlink `ddf0508`; CI + test `7856648`; chessbook trên server `26f0d13`; code-split, LSP/agent socket tự nối lại, a11y `1d6b2f5`; multi-arch Docker `5fd7fd6`.
- **Chưa xác minh** từng mục: cần một lượt rà lại theo danh sách 22 phát hiện (ví dụ `/ws/lsp` `SetReadLimit`, PGN import bịa dữ liệu, cookie `Secure` sau proxy) rồi đánh dấu xong/còn.

## D. Ý tưởng mở rộng (chưa quyết)
- Thư viện chessbook: thêm module mới (ví dụ bài tập theo chủ đề, bảng giải đấu), thống nhất tham số `turn`/`to-move`.
- Nhập dữ liệu: thêm định dạng nguồn khác cho `dataImport.ts`.
- Trợ lý AI: skill sinh Typst chessbook có kiểm tra tự động (import dòng 1, không mock).
- Đồng bộ: mở rộng sang section settings còn lại nếu cần; xử lý xung đột Dropbox (`conflicts` hiện chỉ đếm).

## Quy trình đưa một mục vào làm
1. Chốt mục và cách kiểm với chủ dự án.
2. Viết plan ngắn vào `docs/plans/` (tên `plan_<chủ-đề>.md`).
3. Làm theo mẫu trong [REPLICATION.md](REPLICATION.md), cập nhật tài liệu liên quan.
4. Thêm dòng vào [HISTORY.md](HISTORY.md), gạch mục khỏi bảng này.
