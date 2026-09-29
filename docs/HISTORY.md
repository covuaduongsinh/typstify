# Lịch sử phát triển Typstify

Nguồn: `git log` (238 commit tại 2026-09-29) và `docs/plans/*`. Ngày theo commit. Mục "Phiên làm việc" ghi lại các lần cộng tác với AI để làm cơ sở tiếp nối.

## Các mốc chính

| Giai đoạn | Ngày | Nội dung tiêu biểu (commit) |
|---|---|---|
| Khởi đầu mã nguồn mở | 2026-03-03 → 03-06 | Initial commit / open source release `69a8a95`; cải thiện highlight, hiệu năng editor, refactor previewer |
| Bản web đầu tiên | 2026-09-18 → 09-21 | Kế hoạch web + VPS; thay preview WebSocket bằng preview PDF tĩnh `899adc8`; lưu lựa chọn model/mode agent `9ff1ccc`; giới hạn Open/Create Project trong root cố định trên Docker `7246fe2` |
| Biến thành "Chess Publishing Studio" | 2026-09-21 | `be8e258` thư viện chessbook + công cụ cờ vua; splitter kéo được, tạo file/thư mục trong FileTree, welcome card cờ vua, quick actions cho AI |
| Thiết kế thương hiệu Dương Sinh | 2026-09-24 | Design tokens sáng/tối, giao diện tiếng Việt, toolbar cờ vua, màn hình vào có thương hiệu (PR #1) |
| Rà soát và làm cứng | 2026-09-25 | Kế hoạch rà soát `7141973`; đóng WebSocket preview không xác thực, vá deadlock/race agent, giới hạn compile song song, đường dẫn an toàn symlink `ddf0508`; CI + test `7856648`; code-split web `1d6b2f5`; OAuth loopback cho agent `67aa8f3`, `19a2664`; bundle Antigravity `4eb1822`; chessbook chạy được trên server `26f0d13` |
| Nhập liệu cờ vua | 2026-09-25 → 09-27 | Cải thiện nhập PGN, thư viện template `c008d26`; escape `#` trong `table.header` `51b73bf`; sửa tương thích tham số, tự phát hiện import `c5c6088`; gỡ `#let` mock do AI sinh `2a7d32a`; hoist `#import` lên dòng 1 `e0c496d`; snippet VS Code `15da32e` |
| Mẫu puzzle in ấn | 2026-09-26 | Template A4 (12 bài, 3x4) và 16x24cm (6 bài, 2x3) `fea2a0a`; nhập JSON/CSV/FEN với tự chia trang `9f747df`; Dropbox sync + import/export dự án `c028966` |
| Markdown → Typst và tài khoản | 2026-09-27 | Bộ chuyển Markdown → Typst `84657aa`; sửa FEN (`fen:` prefix, code fence) `259d6d4`; đăng ký, đổi mật khẩu, phiên bền `f2fa7dc`; nâng cấp giao diện nhập cờ `7748b75`; cỡ chữ tuỳ ý + 1–2 cột `ea883f8`, áp dụng toàn tài liệu `232b24e` |
| Preview tức thời | 2026-09-27 | Instant live preview + đồng bộ cuộn theo con trỏ `4169ee9` |
| Chỉnh preview cho chính xác | 2026-09-28 | Namespace id SVG theo trang, ảo hoá render `571e44c`; hiển thị đủ chiều cao trang `1f4c80b`; sửa url(#...) `8aea7ca`; phát hiện `#pagebreak` `c5c54e3`; chế độ tinymist-exact `29c170e`; sửa Host/Origin khi proxy tinymist `0cc731d`, `e0c56db` |
| Đồng bộ vị trí heading thật | 2026-09-28 | `a14f3bd` dùng `QueryHeadingPages` + `/api/preview/anchors` |
| Khoảng cách tính năng desktop ↔ web | 2026-09-28 | Server: lịch sử phiên agent, quản lý font, tuỳ chọn export `d939ea7`; web: bù tính năng editor/preview/export/agent/settings `7266554`; desktop: click-to-source, chọn phiên bản gói, cài đặt font `7a82251` |
| Đồng bộ desktop ↔ web | 2026-09-29 | `c3cd2ef` sync settings, phiên AI agent từ xa, dự án remote (Bearer token) |

## Kế hoạch trong `docs/plans/`

Đây là nhật ký ý định lúc viết. Đối chiếu hiện trạng bằng git log ở trên và [ROADMAP.md](ROADMAP.md).

| File | Ngày | Chủ đề |
|---|---|---|
| `plan_exploration_and_setup.md` | 2026-09-18 | Tìm hiểu, giới thiệu, vận hành Typstify |
| `plan_fix_preview.md` | 2026-09-18 | Sửa split-view/live preview desktop (StartTimeout tinymist 25s, `togglePreview` im lặng khi lỗi, vị trí webview) |
| `plan_web_version.md` | 2026-09-20 | Phiên bản web |
| `plan_vps_dokploy_deployment.md` | 2026-09-20 | Docker Compose + Caddy TLS + Claude Code/Antigravity qua ACP trên VPS |
| `chess_web_publishing_plan.md` | 2026-09-21 | Nền tảng web Chess Publishing Studio |
| `plan_codebase_review_2026-09.md` | 2026-09-25 | Rà soát toàn diện: điểm số từng mảng, 22 phát hiện, kế hoạch 6 giai đoạn |
| `plan_chess_improvements.md` | 2026-09-25 | Nhập PGN, xếp bàn cờ, template, cơ chế biên dịch |
| `plan_chess_theory_fen_templates.md` | 2026-09-27 | Mẫu chèn thế cờ (FEN) sau diễn giải lý thuyết |
| `plan_global_font_size_and_columns.md` | 2026-09-27 | Cỡ chữ và số cột toàn tài liệu |
| `plan_instant_preview_and_cursor_sync.md` | 2026-09-27 | Preview tức thời (debounce 300ms) + cuộn theo con trỏ |
| `plan_markdown_to_typst_chess.md` | 2026-09-27 | Markdown → Typst chuyên cờ vua + VPS Dokploy |
| `plan_ui_data_import_and_markdown_enhancements.md` | 2026-09-27 | Giao diện nhập liệu cờ, Markdown, xuất bản |
| `plan_desktop_installer.md` | 2026-09-29 | Đóng gói desktop Windows: `gogio` + typst/tinymist/chessbook → bộ cài Inno Setup hoặc zip portable |
| `plan_user_auth_and_persistence.md` | 2026-09-27 | Tài khoản, đổi mật khẩu, duy trì đăng nhập |

## Phiên làm việc

### 2026-09-29 — Lập bộ tài liệu nền tảng
- **Yêu cầu**: tạo/cập nhật các file .md (agents, claude, readme, skills, tech, memory…) để thống kê module, mô tả thuật toán, lưu lịch sử, làm cơ sở rà soát, thêm module mới và nhân bản phần mềm.
- **Cách làm**: khảo sát chỉ-đọc bằng ba agent (desktop Go; web + server; chessbook + tài liệu), viết kế hoạch, rồi ghi tài liệu.
- **Kết quả phần này**: `README.md` (thêm tính năng + mục lục), `CLAUDE.md`/`AGENTS.md` (thêm bản đồ repo, lệnh, quy ước, cạm bẫy), và `docs/` gồm `README`, `SKILLS`, `MEMORY`, `HISTORY`, `ROADMAP`, `REPLICATION`.
- **Phát hiện đáng nhớ**: web SVG preview chưa có click-to-source; desktop chưa dùng anchors heading thật; desktop i18n không có tiếng Việt; `.claude/` và `demo/` bị `.gitignore` loại.

## Cách ghi tiếp
Thêm mục mới trên cùng phần "Phiên làm việc" với: ngày, yêu cầu, quyết định, file đã đổi, việc dang dở. Với mốc tính năng thì thêm một dòng vào bảng "Các mốc chính".
