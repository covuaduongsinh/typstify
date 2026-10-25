# Phân tích typst.app (2026-10-24)

Thuộc Giai đoạn 0 của [plan_typst_app_parity_2026-10.md](../plans/plan_typst_app_parity_2026-10.md).

## Phạm vi và phương pháp

- **Đối tượng:** typst.app, quan sát trong phiên Chrome đã đăng nhập của chủ dự án (tài khoản miễn phí, project `test typst`).
- **Công cụ:** extension Claude in Chrome. Mọi phân tích chạy tại chỗ trong trang: DOM, Performance Resource Timing, và đọc tĩnh bundle `index-*.js` (3,36 MB) cùng `worker-*.js`.
  - Không tải bundle về máy, không chạy lại request API, không thử thao tác thay đổi dữ liệu.
  - Các công cụ trình duyệt của `rea` (`inspect_web_page`, `analyze_web_bundle`, `observe_web_session`, ...) cần Chrome mở cổng CDP loopback (`--remote-debugging-port`). Máy chưa bật nên lượt này chưa dùng được. Xem mục "Việc còn mở".
- **Nguyên tắc:** tài liệu này chỉ mô tả **cơ chế và tính năng**, không chép mã của typst.app. Tên thao tác GraphQL và nhãn menu được ghi lại như bằng chứng hành vi.

## 1. Kiến trúc kỹ thuật quan sát được

| Thành phần | Quan sát | Bằng chứng |
|---|---|---|
| Frontend | SPA bundle bằng Rolldown/Vite. React (có `import_jsx_runtime`), state kiểu `useLive(...)`. Có PWA: `site.webmanifest`, service worker `serviceWorker-*.js` đang kiểm soát trang | `<script src=/assets/index-*.js>`, `navigator.serviceWorker.controller` |
| Editor | **CodeMirror 6**: lint, fold gutter, `cm-improvementsGutter` (gợi ý cải thiện), `cm-drag-gutter`, line wrapping. Có keymap Vim và Emacs | lớp `cm-*` trong DOM; chuỗi `vim`, `emacs` trong bundle |
| Lõi Typst | **Một module WASM Rust duy nhất** `core_bg-*.wasm` (~31 MB giải nén) gánh compile, IDE (autocomplete, tooltip, jump) và render. Worker JS chỉ là lớp mỏng (~12 KB, giao thức `init/request/accept`) | resource timing; nội dung `worker-*.js` |
| Đa phiên bản compiler | Bảng `URLS` có compiler riêng cho từng phiên bản Typst: 0.10.0 tới 0.15.1. Mỗi project chọn phiên bản ("Upgrade your projects") | bundle: các khoá phiên bản trỏ tới `wasmUrl` + `worker` riêng |
| Pool worker | Mở khoảng 9 instance worker cho một project (song song compile/IDE/render nhiều file) | 9 resource `worker-*.js` |
| Font | Font mặc định (Libertinus, New CM Math, DejaVu, Twitter Color Emoji) tải theo nhu cầu. Chỉ mục font `font-index-v2.mpk` (MessagePack, 1,3 MB) cùng sprite xem trước `font-previews.png/json` cho bộ chọn font | resource timing |
| Package | Đọc trực tiếp `packages.typst.org/preview/index.json`. Có namespace package riêng của người dùng/team qua `api.typst.app/v1/package/local/index.json` (private packages) | network |
| Preview | 1 `<canvas>` cùng nhiều `<svg>`. Có chế độ **Paged** và **HTML** (HTML preview/export ra từ 2026-01) | DOM; menu "Preview mode: Paged / HTML" |
| API | **GraphQL** `api.typst.app/v1/graphql` (125 operation), REST cho file/thumbnail | xem mục 3 |
| Cộng tác | **Yjs** + `y-codemirror`, awareness/peers, nội dung file tải qua `GET /v1/files/{project}/{file}/yjs` rồi đồng bộ qua **WebSocket** (có tuỳ chọn debug "Disable WebSocket" và "Use fetch cache") | network; chuỗi `awareness.peers`, `new WebSocket` |
| Phân tích sản phẩm | Mixpanel, Sentry, endpoint `misc-b.typst.app/events/track` | bundle/network |

**Kết luận kiến trúc:** typst.app biên dịch **hoàn toàn phía client** bằng WASM. Server chỉ lưu trữ Yjs, metadata, auth, package và thumbnail. Typstify hiện ngược lại: server compile, server chạy tinymist.

## 2. Danh mục tính năng (từ menu, cờ tính năng, GraphQL)

### Cờ tính năng (`FEATURES.*`)
`PRESENTATION_MODE`, `SPEAKER_NOTES`, `CREATE_COMMENTS`, `FOLDERS`, `PROJECT_EMAIL_INVITES`, `PUBLISH_PACKAGES`, `USE_VCS_SYNC`, `GITHUB_SYNC`, `GITLAB_SYNC`, `USE_REFERENCE_SYNC`, `ZOTERO_SYNC`, `MENDELEY_SYNC`, các theme (`CHRISTMAS_THEME`, `TUB_THEME`, `UBUNTU_THEME`) và các cờ on-prem/SSO (`LOCKED_*`, `ONPREM_LICENSE_EXPIRATION`). **typst.app có bản on-prem.**

### Menu editor
- **File:** New file/folder, Upload file, Rename, Preview this file / Set as preview (đổi file gốc preview), **Package project** (đóng gói project thành package), Export as (PDF/PNG/SVG/HTML), Download backup.
- **Edit:** Undo/Redo, Search and replace (và **tìm kiếm toàn project**), Go to line, toggle comment dòng/khối.
- **View:**
  - Bố cục: File panel, Show collaborator cursors, Show toolbar, Scroll on type, Wrap lines, Split views (dọc/ngang), Cursor size, Only editor / Only preview / Both / **Preview in popup**.
  - Preview: Zoom và Fit width/height/page, **Simulate color blindness** (Protanopia/Deuteranopia/Tritanopia), Invert preview.
  - Theme: Light/Dark/System và theme đặc biệt.
  - Line numbers: Normal/Relative/Disabled.
- **Trình chiếu:** Present, Present from current slide, **Speaker Mode**, Next/Previous slide, Blackout, **Pointer/Laser**, **vẽ mực** (Ink color, Clear slide), Show presentation in this window.
- **Debug:** Restart compiler, **Restart and start profiling**, Use fetch cache, Disable WebSocket.
- **Thanh công cụ định dạng:** Ag (font), B/I/U, Heading, list, enum, Σ (toán), `<>` (raw), `@` (tham chiếu), bình luận. Có nút "↷" (format/redo) và zoom preview.
- **Thanh bên:** Files, Search (toàn project), Outline, **Improve** (danh sách gợi ý cải thiện, có badge số lượng), Settings, Packages/Universe, Help.
- **Spellcheck:** Hunspell, có "Add to dictionary" và "Disable spellcheck". Hiển thị squiggle trong editor (trên tài liệu tiếng Việt thấy gạch dưới tên font).
- **Wizard tạo project từ template**:
  - Loại tài liệu: Book, Report, Paper, Thesis, Poster, Flyer, Presentation, Office.
  - Tuỳ chọn cấu hình: kiểu tiêu đề (Title page/row/none), khổ giấy (US Letter/Legal/A4), số cột (Single/Double), mật độ (Tight/Loose), font (Serif, ...), lĩnh vực (discipline), thông tin tác giả (Email/Affiliation/Postal address/Phone).
- **Dán file:** hộp xung đột "Pasting file / Conflict / Skip All / Overwrite All".
- **Lưu:** trạng thái "Saving changes / Saved changes / Failed to save changes", cảnh báo "File edit history too large" (Yjs history).

### Thao tác GraphQL (nhóm theo miền, 125 thao tác)
- **Project:** project files/refresh, file refresh, create/duplicate/delete/edit project, project join (qua link), report project, project file ids, dashboard content (folder).
- **Thành viên và chia sẻ:**
  - Mời thành viên vào project: invite, edit, delete project member, project invitation accept/decline/revoke.
  - Vai trò **Write / Review / Read**: review là quyền chỉ comment, ra mắt 2025-09.
  - Mời qua email.
- **Comment:** create, edit, resolve, delete comment; create, edit, delete comment reply; comment refresh.
- **File đặc biệt ("magic file"):** create magic file, force magic file sync, disenchant file, convert to editable text. Đây là file đồng bộ từ nguồn ngoài (Zotero/Mendeley `.bib`); "disenchant" chuyển nó thành file thường.
- **Thư mục tham khảo:** refresh zotero, refresh mendeley, reference manager meta, authorize/disconnect oauth.
- **VCS (GitHub/GitLab):** connect/reconnect/disconnect vcs, fetch vcs refs/dirs/emails, **vcs push / vcs pull**, refresh vcs, **new revisions** (lịch sử revision).
- **Package:** publish package, delete package (namespace private).
- **Team:**
  - Quản trị: create/edit/delete team; member invite/edit/delete; resend/revoke/accept/decline invitation.
  - Domain đã xác minh: add, verify, reroll, delete team domain.
  - SSO **OIDC** đầy đủ: discovery, claims, domains, test authentication, IdP access policy.
- **Tài khoản:** edit user, đa email (add/promote/delete), xoá tài khoản, **MFA TOTP** (enrollment, backup codes, đổi TOTP), add password/native auth, nguồn thông tin hồ sơ (user info source), storage quota (user storage).
- **Thương mại:** subscription, billing, invoice, cancellation survey, NPS, feedback. Typstify không cần nhóm này.

## 3. Đối chiếu với Typstify và đề xuất

Ưu tiên: **P1** là cốt lõi, làm trước; **P2** quan trọng; **P3** làm sau hoặc tuỳ chọn. "Thay thế mở" là công nghệ mã nguồn mở dùng để tái hiện.

| Tính năng typst.app | Typstify | Ưu tiên | Thay thế mở / hướng làm |
|---|---|---|---|
| Compile WASM phía client, preview tức thì, offline | Không có (server compile) | P1 | `typst.ts` (`@myriaddreamin/typst.ts`, Apache-2.0) trong Web Worker. Server compile giữ làm fallback |
| IDE trong WASM (autocomplete/tooltip không cần server) | Qua tinymist `/ws/lsp` | P2 | Giữ tinymist (đầy đủ hơn). Thêm `tinymist` WASM khi offline/mobile |
| Đa phiên bản compiler theo project | Một bản CLI | P3 | Ghi `typst-version` vào metadata project. Server tải nhiều bản CLI, client tải nhiều bản WASM |
| Font theo nhu cầu + bộ chọn font có xem trước | Font nhúng + upload | P2 | Chỉ mục font do server sinh (JSON/MessagePack) + sprite xem trước. Client tải lười |
| Cộng tác Yjs realtime + con trỏ | Không có | P1 | Yjs + `y-codemirror.next` + relay WebSocket bằng Go |
| Comment có reply/resolve, quyền Review | Không có | P1 | Comment neo `Y.RelativePosition`, lưu bbolt. Vai trò commenter |
| Chia sẻ, mời, vai trò Write/Review/Read, join qua link | Không có (workspace chung) | P1 | Mô hình Project + thành viên + share link (Giai đoạn 1) |
| Folder trên dashboard, sắp xếp, grid/list | Có cây file, không có dashboard project | P2 | Dashboard project web |
| Tìm kiếm toàn project | Chưa rõ trên web | P2 | API tìm kiếm server (ripgrep-like bằng Go) + panel |
| Panel Improve (gợi ý cải thiện) | Không | P3 | Lint tuỳ biến (tinymist lint + quy tắc chessbook: import dòng 1, escape `#`) |
| Spellcheck Hunspell + từ điển người dùng | Không | P2 | `nspell`/Hunspell WASM, từ điển `vi_VN` + `en_US` |
| Vim / Emacs keymap, relative line numbers | Không | P3 | `@replit/codemirror-vim`, `@replit/codemirror-emacs` |
| Thanh công cụ định dạng | Có toolbar cờ vua | P2 | Thêm B/I/U/heading/list/math/raw/ref/comment |
| Wizard template theo loại tài liệu và tuỳ chọn | 8 template cố định | P2 | Wizard tham số hoá + template Universe + chessbook |
| Upload, dán file, xử lý xung đột | Chỉ import chuyên dụng | P1 | Upload/drag-drop/paste vào cây file, hộp Skip/Overwrite |
| Set as preview (file gốc preview) | Có `main` cố định? | P2 | Metadata project `entry` |
| Preview HTML | Export HTML có, preview HTML chưa | P3 | `typst compile --format html` / typst.ts HTML |
| Trình chiếu: speaker notes, laser, vẽ mực, blackout | Không | P2 | Chế độ present trên preview (rất hợp dạy cờ) |
| Mô phỏng mù màu, invert preview | Không | P3 | CSS/SVG filter trên preview |
| Preview trong popup, split dọc/ngang | Có split | P3 | `window.open` đồng bộ qua BroadcastChannel |
| GitHub/GitLab sync push/pull | Không (chỉ git status desktop) | P2 | `go-git` + OAuth GitHub/GitLab |
| Lịch sử revision | Không | P1 | Snapshot Yjs + diff/khôi phục (Giai đoạn 4) |
| Zotero/Mendeley sync (`.bib` "magic file") | Không | P3 | Zotero Web API → file `.bib` chỉ đọc, có "convert to editable" |
| Đóng gói project thành package, publish private package | Có Tpix, `@local/chessbook` | P2 | Namespace `@local`/`@team` phục vụ từ server cho cả CLI và WASM |
| MFA TOTP, đa email, OIDC SSO | Mật khẩu SHA-256 | P1 (MFA, argon2id), P3 (OIDC) | argon2id + TOTP (`pquerna/otp`). OIDC (`coreos/go-oidc`) sau |
| PWA + service worker | Không | P1 (cho mobile) | `vite-plugin-pwa` |
| Debug: restart compiler, profiling | Không | P3 | Nút restart worker. Trace thời gian compile |

**Typstify có mà typst.app không có (giữ và đẩy mạnh):** AI agent ACP + MCP, chessbook và công cụ nhập cờ (PGN/FEN/puzzle), Markdown → Typst, đồng bộ Dropbox/VPS, tự host, desktop native.

## 4. Điều chỉnh thứ tự giai đoạn so với kế hoạch gốc

Kết quả phân tích **xác nhận** thứ tự trong kế hoạch, với ba điều chỉnh:

1. **Giai đoạn 1** thêm MFA TOTP và vai trò ba mức Write/Review/Read (Review = commenter). Thứ tự này khớp đúng typst.app.
2. **Giai đoạn 2** thêm việc phục vụ chỉ mục font cho client và cho phép mỗi project chọn phiên bản Typst (`typst-version`).
3. **Giai đoạn 6** bổ sung các mục sau, xếp theo mức độ hợp với người dùng dạy cờ:
   - Panel Improve và lint chessbook.
   - Trình chiếu kèm laser, vẽ mực và speaker notes.
   - Tìm kiếm toàn project.
   - Thanh công cụ định dạng.

Zotero/Mendeley, OIDC và theme đặc biệt để ở P3 (sau Giai đoạn 7).

## 5. Việc còn mở

- **Bật CDP để chạy các công cụ trình duyệt của `rea`.** Cần khởi động Chrome với `--remote-debugging-port=9222` trên một profile riêng (Chrome 136+ chặn CDP trên profile mặc định), đăng nhập typst.app trong profile đó, rồi dùng các công cụ:
  - `list_browser_targets`, `inspect_web_page`, `analyze_web_bundle` (chunk graph, endpoint, vendor fingerprint);
  - `observe_web_session` kèm `include_websocket_shapes` (hình dạng thông điệp WebSocket Yjs);
  - `inspect_web_event_listeners` (paste, drop).

  Bước này bổ sung bằng chứng về giao thức, nhưng **không chặn** Giai đoạn 1 và 2, vì cơ chế đã đủ rõ.
- **Phân tích tĩnh `core_bg.wasm` (`inspect_wasm_artifact`).** Bước này cần tải file 31 MB về máy và có WABT. Lợi ích thấp vì lõi là typst mã nguồn mở. Chỉ làm khi chủ dự án đồng ý tải.
