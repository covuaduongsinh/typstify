# Bộ nhớ dự án (ngữ cảnh bền vững cho AI)

File này gom những điều không suy ra được từ đọc mã một lần: quyết định thiết kế, cạm bẫy đã trả giá, điều cấm. Cập nhật khi phát hiện điều mới. Mọi mục đều đã đối chiếu với mã hoặc tài liệu tại 2026-09-29; kiểm lại trước khi dựa vào.

## Bối cảnh vận hành
- Chủ dự án: công ty Cờ vua Dương Sinh. Bản web triển khai tự host (Dokploy/VPS, Docker Compose + Caddy TLS), giao diện tiếng Việt. Xem `docs/web-server.md`.
- Ngôn ngữ làm việc: tiếng Việt cho giải thích và tài liệu; code/định danh/đường dẫn tiếng Anh.
- Commit theo kiểu `feat(scope): …`, `fix(scope): …`.

## Quyết định kiến trúc
- `service.ServiceFacade` không phụ thuộc Gio → dùng chung desktop và server. UI gắn vào qua hook (`SetViewManager`). Giữ nguyên tính chất này khi thêm dịch vụ.
- `server/` chỉ là transport; logic ở `service/`, `agent/`, `lsp/`, `typst/`.
- Settings lưu JSON có timestamp từng section; đồng bộ theo last-write-wins per section (`ApplyRemote`). Section `agent` và `RemoteSettings` không đồng bộ; `tpix` (API key) không expose qua HTTP.
- Server compile giới hạn 2 slot song song, 90 giây mỗi lần (`compileSlots`).
- Web dùng `useState`/refs thuần, không router, không state library; các panel nặng lazy-load.
- Preview web có 3 chế độ: SVG (từ nội dung chưa lưu, qua file bóng `.live_preview_*.typ` cùng thư mục), PDF, iframe tinymist.
- Đồng bộ cuộn web chỉ tin anchors từ server khi số heading client (regex `^=+\s+\S`) khớp số heading server.

## Cạm bẫy đã biết
**Desktop / LSP**
- `tinymist.doKillPreview` phải truyền id `"default_preview"`; gọi không tham số thì lỗi.
- Callback `window/showDocument` chạy trên goroutine LSP: sau `NavigateToLine` phải `RefreshWindow()`; và luôn trả `ShowDocumentResult` vì đó là request.
- `bus.Subscribe` panic nếu đăng ký trùng khoá `%p:name`.
- Chỉ nhận `TextEdit` trong completion; `InsertReplaceEdit` bị bỏ qua.
- Webview native phải huỷ trễ một frame (`destroyPending`) vì chạy trong frame Gio.
- Autosave không ghi đè nếu file trên đĩa đã đổi so với `originalHash` (báo "edited elsewhere").

**Server / web**
- Preview "Tức thì" (WASM, typst.ts 0.8.0-rc3 = Typst 0.15): **không dùng `renderToSvg`** của renderer. Nó gọi `window.typstProcessSvg` rồi `unwrap()` (panic khi hàm không tồn tại, làm hỏng cả module WASM) và bỏ qua vẽ lại khi bề rộng không đổi (`data-applied-width`). Dùng `renderSvgDiff` + `patchRoot` như typst-preview (`WasmPreview.tsx`).
- Sau `patchRoot`, thứ tự `.typst-page` trong DOM **không** theo thứ tự trang; tính vị trí trang phải sắp theo toạ độ.
- `compiler.query()` của typst.ts parse JSON hai lần (lỗi với kết quả object); dùng `runWithWorld` + `world.query`. Range của diagnostics đánh số từ 0.
- Mỗi lần reset tài liệu (đổi file, nạp lại, mount lại) phải dùng session renderer mới và đổ toàn bộ SVG; diff chỉ hợp lệ với DOM do chính session đó vẽ.
- Cookie phiên `Secure` phụ thuộc `X-Forwarded-Proto` từ peer tin cậy (loopback/private).
- Đường dẫn file phải qua `resolveInRoot` (chặn `..` và symlink); `.git`, `.typstify` được bảo vệ.
- Lỗi compile preview trả HTTP 200 với `ok:false`.
- WebSocket agent đọc tối đa 20 MiB (ảnh dán); `/ws/lsp` từng thiếu `SetReadLimit` (xem plan rà soát, đối chiếu lại trước khi tin).
- Rò tiến trình con: không để `TYPSTIFY_SERVER_PASSWORD` lọt vào env agent/terminal.
- Server chỉ có **một project hiện hành toàn cục**; phân quyền (`handleRole`) kiểm user của request với project đó. Route mới đụng file project phải đăng ký bằng `handleRole` (đọc ≥ viewer, ghi ≥ editor); route đổi cấu hình máy chủ dùng `handleAdmin`.
- Test trong `server/` phải dùng `testService()` (dùng chung một `ServiceFacade`): mỗi `service.NewService` mở cùng file bbolt thật, lần mở thứ hai trong cùng tiến trình hết thời gian chờ và `log.Fatal("timeout")` làm chết cả gói test mà không có `--- FAIL`. Truyền `Options.StorageDir` là thư mục tạm để không ghi vào tài khoản thật.
- Mở project trong test khởi động tinymist và file watcher, giữ thư mục trên Windows; dùng `os.MkdirTemp` + `RemoveAll` bỏ qua lỗi thay cho `t.TempDir()`.
- Chạy thử server mà không đụng dữ liệu thật: đặt `APPDATA` sang thư mục tạm (settings và tài khoản nằm dưới `os.UserConfigDir()`). Gửi JSON tiếng Việt từ Git Bash bằng `--data-binary @file`, vì tham số dòng lệnh bị đổi codepage.

**Typst / chessbook** (AI hay mắc)
- Không tự khai báo `#let` mock cho hàm chessbook; `#import "@local/chessbook:0.1.0": *` phải ở **dòng 1**.
- `#` trong `[...]` phải escape `\#`. `=` đầu content block → heading rỗng, viết `[#"="]`.
- Dòng markup bắt đầu `N.` thành danh sách đánh số; đưa movetext vào chuỗi.
- Không đặt font "chess" toàn cục (chiếm ∓/±); dùng show-rule theo dải mã quân cờ.
- `notation()` của staunton không xác thực luật cờ.
- Import tương đối `../lib/lib.typ` bị `--root` chặn trên server; dùng package `@local`.
- API không nhất quán lịch sử: `turn` và `to-move` đều được chấp nhận; theo dõi `opening-diagram-box`.

## Điều cấm / cần hỏi trước
- Không bịa dữ liệu ván cờ (Elo, danh hiệu, kết quả, địa điểm) khi PGN thiếu tag.
- Không commit `.env`, token, `*.exe`.
- Không đổi `chessbook` theo cách phá `CLAUDE.md` mà không cập nhật tài liệu và chạy `scripts/check-chessbook.sh`.

## Điểm mạnh cần giữ (theo rà soát 09/2026)
Service layer dùng chung; so sánh mật khẩu constant-time; token 32 byte crypto/rand; cookie HttpOnly + SameSite; `/ws/*` kiểm Origin; O_EXCL khi tạo file; Docker non-root + healthcheck; comment giải thích quyết định.

## Liên kết
[HISTORY.md](HISTORY.md) · [ROADMAP.md](ROADMAP.md) · [SKILLS.md](SKILLS.md) · [plans/plan_codebase_review_2026-09.md](plans/plan_codebase_review_2026-09.md)
