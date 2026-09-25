# Rà soát tổng thể codebase & kế hoạch cải thiện (09/2026)

## Context
Sau đợt nâng cấp giao diện web (PR covuaduongsinh/typstify#1, đã merge), Thầy Tường yêu cầu review toàn bộ codebase: phân tích, đánh giá, lập kế hoạch cải thiện. Phạm vi: **cả bản web (server Go + React) lẫn desktop (Gio)**, thư viện Typst `chessbook/`, Docker/deploy. Bối cảnh vận hành: **chỉ Thầy đăng nhập** typst.dsc.edu.vn (Dokploy trên VPS). Rà soát bằng 3 luồng song song + tự kiểm chứng các phát hiện nghiêm trọng trong mã nguồn.

## 1. Bức tranh tổng thể (~69k dòng Go, 4.2k dòng TS/CSS, 32 file test)

| Mảng | Điểm | Nhận xét ngắn |
|---|---|---|
| Kiến trúc | 8/10 | Service layer dùng chung desktop ↔ server (compile/export/settings/agent). Server build CGO=0, `go vet` sạch. |
| Bảo mật server | 4/10 | 1 lỗ hổng không cần đăng nhập, không timeout/giới hạn body, không chống dò mật khẩu, cookie thiếu Secure sau proxy. |
| Tính đúng / ổn định | 5/10 | Deadlock + data race trong agent session, rò session, 2 lỗi mất dữ liệu ở web editor. |
| Tính năng cờ vua trên server | 3/10 | **Thư viện `chessbook` và font không có trong image** → mọi nút chèn cờ báo "unknown variable". PGN import sinh Typst sai. |
| Kiểm thử & CI | 2/10 | 0 test cho `server/`, `agent/`, `service/`, `typst/`, web; không có CI. |
| Deploy | 7/10 | Multi-stage, non-root, healthcheck, pin phiên bản; nhưng chỉ amd64, không checksum, ship binary antigravity có thể SIGILL. |
| Tài liệu / vệ sinh repo | 5/10 | README chỉ nói desktop; ~26 MB font không dùng trong git; file trùng lặp, code chết. |

## 2. Phát hiện chính (đã kiểm chứng file:line)

### Nghiêm trọng — cần xử lý ngay
1. **WebSocket xem trước ở `/` không qua xác thực, không kiểm Origin** — `server/server.go:124` mount static handler không `auth.require`; `staticHandler` → `handlePreviewRootWebSocket` (`server/preview_proxy.go:77`) proxy thẳng tới tinymist. Người ngoài đọc được nội dung tài liệu đang mở.
2. **Không timeout HTTP, không giới hạn body ở endpoint công khai** — `cmd/typstify-server/main.go` (http.Server không ReadHeaderTimeout); `server/auth.go:97` login, `server/i18n_api.go:29` decode không giới hạn → slowloris/DoS.
3. **Không chống dò mật khẩu** — `auth.go:90-105`.
4. **Cookie phiên không bao giờ `Secure` sau Traefik/Dokploy** — `auth.go:123` dùng `r.TLS != nil`.
5. **Mất dữ liệu: chuyển file có thể ghi nội dung file cũ đè file mới** — `web/src/components/Workspace.tsx:108-123`: không reset `content` trước khi fetch; `<Editor key={activePath} initialContent={content}>` mount ngay với nội dung cũ → Ctrl+S ghi sai file.
6. **Mất dữ liệu: đổi file / đóng dự án / rời trang không cảnh báo khi chưa lưu; lưu lỗi vẫn báo "Đã lưu"** — `Editor.tsx:84-93` (không await `onSave`), không có `beforeunload`.
7. **Thư viện cờ vua không chạy trên server** — `Dockerfile:104-108` không chép `chessbook/`, không có font; `NEW_DOC_TEMPLATE` (`Workspace.tsx:54`) không `#import`; import tương đối `../lib/lib.typ` bị `--root` chặn.

### Cao
8. **Deadlock** — `agent/session.go:411-413` `SubscribeUpdates` return khi còn giữ `sn.mu`.
9. **Rò session agent + treo kết nối** — `agent/manager.go:450-453` bỏ qua dọn dẹp khi agent không hỗ trợ `session/close`; `RequestPermission` chờ vô hạn trên channel không buffer (`session.go:390`, `acp_client.go:175,184`).
10. **Data race** — `CurrentTurn` (`session.go:319,332,368,433,514`), `ServiceFacade.currentProjectDir/previewSrv` (`service/service.go:234,296,300`), `SessionManager.AgentConn()` (`manager.go:109`); `Close` đóng `updateChan` → panic khi publish sau đó.
11. **File .typ > 32 KiB làm rớt kết nối LSP** — `server/lsp_ws.go:89` không `SetReadLimit` (agent_ws có 20 MiB). File > 64 MiB bị cắt âm thầm (`files_api.go:138`).
12. **Biên dịch PDF/Export không timeout, không giới hạn song song** — `typst/export/helper.go` dùng `context.Background()`.
13. **PGN import sinh Typst sai** — `PgnImportModal.tsx:16-91`: không escape (`$`, `*`, `"`), gộp nhiều ván, **bịa dữ liệu** (Elo 2700, GM, "1 - 0", Hà Nội) khi thiếu tag. `ChessBoardModal.tsx:80-111` không escape chuỗi; `column-diagram` nhét tiêu đề vào `move-num`.
14. **Tiến trình con (agent/terminal) thừa kế `TYPSTIFY_SERVER_PASSWORD`** — `agent/terminal.go:59`, `manager.go:130`.

### Trung bình
15. Symlink thoát khỏi thư mục dự án — `server/paths.go:14-54` (agent/path.go đã làm đúng, tái dùng).
16. Cấu hình lệnh agent / đường dẫn typst không kiểm tra (`service/settings/models.go:114,155-170`) — chấp nhận được vì chỉ Thầy dùng, nhưng phải được bảo vệ bởi các mục 1–4.
17. API thư viện cờ không nhất quán (`to-move` vs `turn`, `opening-diagram-box` bỏ qua `turn`), code bàn cờ lặp 6 nơi, FEN sai làm sập cả lần biên dịch.
18. LSP/agent client web không tự kết nối lại, promise bị từ chối không xử lý (`lib/lspClient.ts:118-131`, `lib/agentClient.ts` báo ngắt kết nối 2 lần).
19. Bundle web 922 kB một chunk, không code-split.
20. Panic trong code agent dùng chung server (`agent/mcp_server.go:88`, `session.go:125`); dialog desktop chặn UI (`ui/dialog/bibliography.go:127`, `publish_pkg.go:95`).
21. Docker: `GOARCH=amd64` cứng nhưng stage antigravity rẽ nhánh arm64; không checksum; `.dockerignore` thiếu `.env`, `fonts/`.
22. Không CI (`.github/` không tồn tại); `go test ./...` hỏng trên máy headless vì package Gio.

### Thấp
Secrets trả về trình duyệt (`/api/settings/tpix` ApiKey, `/api/console`); rename ghi đè đích; phiên 30 ngày trượt không có hạn tuyệt đối; link trong chat AI mở cùng tab; i18n `detectLocale` không bao giờ trả en-US; bàn cờ không thao tác được bằng bàn phím; code chết `cmd/agy_acp_bridge`; file trùng `chessbook/Extracted pages C58 …typ`; 26 MB font không dùng (`NotoColorEmoji`, `NotoSansSC`); README thiếu phần web, docs cũ.

### Điểm mạnh cần giữ
So sánh mật khẩu constant-time, token 32 byte crypto/rand, cookie HttpOnly+SameSite; `/ws/*` kiểm Origin; `resolveInRoot` chặn `..`; O_EXCL khi tạo file; Docker non-root + healthcheck; comment giải thích quyết định rất kỹ; test tốt cho gvcode & LSP protocol.

## 3. Kế hoạch cải thiện (6 giai đoạn, mỗi giai đoạn = 1 PR, merge & deploy độc lập)

### GĐ1 — Vá khẩn: bảo mật công khai + mất dữ liệu (ưu tiên số 1)
- `server/server.go` / `preview_proxy.go`: yêu cầu `s.auth.validRequest(r)` + kiểm Origin trước `handlePreviewRootWebSocket`.
- `cmd/typstify-server/main.go`: `ReadHeaderTimeout 10s`, `ReadTimeout`/`IdleTimeout`; `http.MaxBytesReader` 1 MiB cho login/i18n/settings (helper chung trong `server/helpers.go`).
- `server/auth.go`: rate limit theo IP (token bucket, trễ sau lần sai), cookie `Secure` khi `X-Forwarded-Proto=https` từ proxy tin cậy hoặc cờ `-secure-cookie`; hạn phiên tuyệt đối.
- `agent/terminal.go`, `agent/manager.go`: lọc `TYPSTIFY_*` khỏi env tiến trình con.
- `web/src/components/Workspace.tsx`: `setContent(null)` trước fetch + bỏ phản hồi muộn (AbortController); hỏi xác nhận khi đổi file/đóng dự án lúc `dirty`; `beforeunload`.
- `web/src/components/Editor.tsx`: `save` await `onSave`, chỉ clear dirty khi thành công, báo lỗi lưu trên StatusBar.
- `server/lsp_ws.go`: `conn.SetReadLimit(20 << 20)`; `files_api.go`: trả 413 thay vì cắt.

### GĐ2 — Thư viện cờ vua chạy thật trên server
- Đóng gói `chessbook/lib` thành gói Typst local `@local/chessbook:0.1.0` (thêm `typst.toml`); Dockerfile chép vào `$HOME/.local/share/typst/packages/local/chessbook/0.1.0/` (HOME=/data hoặc `TYPST_PACKAGE_PATH`).
- Font: thêm vào image bộ font tiếng Việt (Roboto/Noto Serif/Noto Sans) + font quân cờ figurine; truyền `--font-path` qua `settings.Typst.extraFontPath`. Thay "Arial/Times New Roman/Segoe UI Symbol" trong lib bằng font có sẵn.
- `Workspace.tsx` `NEW_DOC_TEMPLATE`: `#import "@local/chessbook:0.1.0": *` + `#show: chess-book-init.with(...)`; QuickActions nhắc agent thêm import.
- `PgnImportModal.tsx`: viết lại parser (tách nhiều ván, bỏ `{}`/`()`/`$n`→NAG, giữ kết quả thật, **không bịa tag**), escape chuỗi Typst dùng chung với `ChessBoardModal.tsx` (tạo `web/src/lib/typstEscape.ts`); sửa `column-diagram move-num`.
- Thư viện: thống nhất `turn` (giữ `to-move` làm alias), `opening-diagram-box` tôn trọng `turn`, gom hàm vẽ bàn cờ về `symbols.typ`, kiểm FEN và báo lỗi rõ thay vì sập. Xóa file trùng C58 và `output/_bnp-test.typ`.
- Bản desktop: cùng gói `@local/chessbook` (hướng dẫn cài 1 lần).

### GĐ3 — Ổn định lõi agent & dịch vụ
- `agent/session.go`: sửa unlock trong `SubscribeUpdates`; `select`+context cho `PublishUpdate`/`RequestPermission`; không close channel khi còn publisher; mutex/atomic cho `CurrentTurn`.
- `agent/manager.go`: luôn xóa + `Close()` session cục bộ; khóa `sm.conn`.
- `service/service.go`: `sync.RWMutex`/`atomic.Pointer` cho `currentProjectDir`, `previewSrv`.
- `typst/export/helper.go` + `server/export_api.go`: truyền `r.Context()` với timeout 60s, semaphore 2 lần biên dịch đồng thời.
- `server/paths.go`: resolve symlink rồi kiểm lại (tái dùng logic `agent/path.go:12-27`); rename thất bại nếu đích tồn tại; chặn xóa `.git`/`.typstify`.
- Đổi `panic` trong `agent/mcp_server.go:88`, `session.go:125` thành trả lỗi; ẩn ApiKey trong `/api/settings/tpix`.

### GĐ4 — Kiểm thử & CI
- Go: test cho `server` (resolveInRoot traversal/symlink, auth middleware + cookie flags, preview WS cần auth, rate limit, LSP read limit), `agent` session với `-race`, `typst/export` (timeout).
- Web: thêm Vitest; test thuần cho `parsePgn`, `boardToFen`, `typstEscape`, sinh code template; Playwright smoke (mock API như script đã dùng) cho đăng nhập → mở file → chèn bài tập.
- Typst: script biên dịch `chessbook/demo_collection/*.typ` + template mới trong Docker để bắt lỗi thư viện.
- `.github/workflows/ci.yml`: `go vet` + `go test -race` (CGO=0 cho gói không-UI), `npm ci && lint && test && build`, `docker build`. Thêm build tag/loại trừ để `go test ./...` chạy được headless.

### GĐ5 — Hiệu năng & trải nghiệm web
- Code-split: `lazy()` cho AgentChat, ChessBoardModal, PgnImportModal, SettingsPanel, PackageManager; `manualChunks` tách CodeMirror.
- `lib/lspClient.ts`, `lib/agentClient.ts`: tự kết nối lại có backoff, timeout request, xử lý promise bị từ chối, sửa báo ngắt 2 lần; giữ 1 kết nối LSP cho cả phiên thay vì mỗi file.
- Giữ undo/scroll khi chuyển file (cache EditorState theo path); link chat mở tab mới.
- Bàn cờ thao tác được bằng bàn phím; i18n cho người dùng tiếng Anh.

### GĐ6 — Desktop, Docker, dọn dẹp, tài liệu
- Desktop: chuyển tác vụ chặn UI sang goroutine (`ui/dialog/bibliography.go:127`, `publish_pkg.go:95`), thay `panic` ở `ui/dialog/export.go:66`, `ui/navpanel/filetree.go:69` bằng thông báo lỗi; dọn ảnh đã xóa (`ui/viewer/image_list.go:30`); dùng chung gói `@local/chessbook`.
- Docker: dùng `TARGETARCH` cho Go/typst/tinymist (multi-arch), kiểm checksum tải về, antigravity thành build-arg tùy chọn; `.dockerignore` thêm `.env`, `fonts/`, `web/node_modules`.
- Repo: xóa `cmd/agy_acp_bridge` (code chết), `NotoColorEmoji`/`NotoSansSC`/`RobotoMono-Italic` không dùng (−26 MB), `internal/gvcode/screenshot.png`; thay `github.com/pkg/errors`.
- Tài liệu: README thêm phần Web/Docker/Dokploy; cập nhật `docs/web-server.md`, `docs/architecture_and_guide.md`; hướng dẫn dùng thư viện cờ `docs/CHESS_STUDIO_GUIDE.md`.

## 4. Thứ tự & ước lượng
| GĐ | Nội dung | Công sức | Rủi ro nếu hoãn |
|---|---|---|---|
| 1 | Bảo mật công khai + mất dữ liệu | 1–1,5 ngày | Rất cao |
| 2 | Thư viện cờ chạy trên server | 2 ngày | Cao (tính năng cốt lõi không dùng được) |
| 3 | Ổn định agent/dịch vụ | 1,5 ngày | Trung bình–cao |
| 4 | Test + CI | 1,5 ngày | Trung bình |
| 5 | Hiệu năng & UX web | 1,5 ngày | Thấp–trung bình |
| 6 | Desktop, Docker, dọn dẹp, docs | 1,5 ngày | Thấp |

## 5. Kiểm chứng (mỗi giai đoạn)
- `CGO_ENABLED=0 go vet ./server/... ./service/... ./agent/... ./typst/...` và `go test -race` các gói đó; `cd web && npm run lint && npm test && npm run build`.
- Chạy thật server trong container: `docker build` → chạy với mật khẩu → Playwright: đăng nhập, mở/tạo tài liệu, chèn `#puzzle-card` và biên dịch ra PDF thành công, nhập PGN nhiều ván, chuyển file khi chưa lưu (phải hỏi), mở file .typ > 32 KiB (LSP không rớt).
- Kiểm bảo mật: `curl` WebSocket upgrade tới `/` không cookie → 401; 20 lần login sai → bị chặn; cookie có `Secure` qua HTTPS; body 2 MiB tới `/api/auth/login` → 413.
- Desktop: `go build .` trên máy có Gio, mở dialog export/bibliography không treo.
- Sau mỗi PR: merge vào `main`, deploy qua Dokploy (Thầy bấm Deploy hoặc mở whitelist `dokploy.dsc.edu.vn` cho môi trường này), kiểm tra typst.dsc.edu.vn.

## 6. Kết quả triển khai (25/09/2026)

Cả 6 giai đoạn đã triển khai trên nhánh `claude/magical-rubin-ks9zy5`, mỗi giai đoạn một commit.

| GĐ | Trạng thái | Kiểm chứng |
|---|---|---|
| 1. Vá khẩn | Xong | Test Go cho auth/giới hạn/cookie/preview WS; Playwright: chuyển file khi mạng chậm, hộp thoại chưa lưu, lưu lỗi |
| 2. Thư viện cờ trên server | Xong | Biên dịch mọi demo/template + tài liệu mới sinh từ code web bằng typst 0.15.1 và font thật: 0 lỗi, 0 cảnh báo |
| 3. Ổn định lõi | Xong | Test `-race` vòng đời phiên agent; test symlink/traversal |
| 4. Test & CI | Xong | `.github/workflows/ci.yml`; 18 test Vitest; `scripts/check-chessbook.sh` |
| 5. Web | Xong | Bundle đầu 922 kB → 256 kB; Playwright hồi quy toàn bộ |
| 6. Desktop, Docker, docs | Xong | `go build .`, `go vet ./...`, `go test -race ./...` toàn repo |

Hoãn có chủ đích:
- **Thay `github.com/pkg/errors`**: `errors.Wrapf(nil, …)` trả `nil` còn `fmt.Errorf` thì không; hai chỗ trong `lsp/client.go` gán vô điều kiện nên thay máy móc sẽ đổi hành vi. Rủi ro lớn hơn lợi ích.
- **Giao diện tiếng Anh**: bản web ưu tiên tiếng Việt cho Dương Sinh; chuỗi UI viết trực tiếp tiếng Việt.
- **Giữ lịch sử hoàn tác khi chuyển file**: cần tách vòng đời LSP khỏi EditorView; để đợt sau.
- **Checksum `board-n-pieces`**: môi trường phát triển không truy cập được packages.typst.org để tính; phiên bản đã được ghim.
- **`docker build` chưa chạy thử cục bộ** (môi trường không có Docker); job `docker` trong CI sẽ kiểm tra.
