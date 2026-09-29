# Tối ưu phần mềm web Typstify (bám sau đợt rà soát 25/09)

## Context

Đợt rà soát toàn diện `docs/plans/plan_codebase_review_2026-09.md` (25/09/2026) đã xử lý xong 6
giai đoạn: lỗ hổng bảo mật công khai, mất dữ liệu, thư viện chessbook chạy trên server, ổn định
agent/dịch vụ, test + CI, hiệu năng/UX web (code-split bằng `React.lazy`, bundle đầu 922→256 kB),
desktop/Docker/dọn dẹp. Đã đối chiếu lại code hiện tại (29/09/2026, sau các đợt preview tức thời,
đồng bộ heading thật, đồng bộ desktop↔web) — các mục đó vẫn còn hiệu lực, không hồi quy.

Yêu cầu lần này ("rà soát lại tổng thể để tối ưu phần mềm web") tập trung vào **phần mềm web**
(server tự host `cmd/typstify-server` + `server/` + frontend `web/`, không đụng desktop Gio) và
nhằm **tối ưu** — hiệu năng tải/chạy + khả năng bảo trì — trên nền đã ổn định về bảo mật/đúng đắn,
tránh lặp lại việc đã Xong.

## Phát hiện (đã kiểm chứng, chưa có trong đợt review trước)

1. **Không nén, không cache HTTP cho tài nguyên tĩnh** — `Caddyfile:12-14` chỉ có
   `reverse_proxy`, không có `encode gzip zstd`; `server/server.go:236-252` (`staticHandler`)
   dùng `http.FileServer` trần, không set `Cache-Control` cho asset đã hash tên bởi Vite.
2. **LSP mở lại WebSocket + phiên tinymist mỗi khi đổi file** — `Editor.tsx:334-336` tạo
   `new LspClient()` (handshake `/ws/lsp` + tinymist init lại từ đầu) trong effect khoá theo
   `path`, dù vẫn trong cùng phiên làm việc. Đã kiểm `server/lsp_ws.go`: 1 kết nối WS đã hỗ trợ
   nhiều `didOpen`/`didClose` nối tiếp (theo dõi qua `watchedPaths`, không giả định 1 doc/kết
   nối) — an toàn để chuyển sang 1 kết nối cho cả phiên.
3. **Chưa tách vendor chunk** — `vite.config.ts` không có `build.rollupOptions.output.manualChunks`.
   CodeMirror + `codemirror-lang-typst` + `react-markdown`/remark/rehype nằm chung 1 chunk lazy
   theo từng component dùng chúng, không tách vendor riêng để cache trình duyệt bền qua các lần
   deploy nhỏ không liên quan.
4. **Đã xác nhận KHÔNG còn là vấn đề** (không làm lại): escape chuỗi Typst dùng chung
   `typstString`; reset `content`/dirty khi chuyển file, `beforeunload`, dialog xác nhận chưa lưu
   còn nguyên trong `Workspace.tsx`; preview WS proxy đã qua `auth.require`.

## Đã triển khai

### GĐ A — Tải trang qua mạng
- `Caddyfile`: thêm `encode zstd gzip`.
- `server/server.go` (`staticHandler`): `Cache-Control: public, max-age=31536000, immutable` cho
  asset tĩnh thực có trên đĩa (tên đã hash bởi Vite), `no-cache` cho `index.html`/fallback SPA.
- `web/vite.config.ts`: `manualChunks` tách `codemirror-vendor` (CodeMirror, Lezer,
  `codemirror-lang-typst`) và `markdown-vendor` (`react-markdown` + remark/rehype/mdast/micromark)
  khỏi chunk theo từng component.

### GĐ B — LSP theo phiên làm việc, không theo từng file
- `web/src/components/Workspace.tsx`: tạo 1 `LspClient` cho cả phiên (khoá theo lần mount
  Workspace = theo `projectPath`), đóng khi Workspace unmount; truyền xuống `Editor` qua prop.
- `web/src/components/Editor.tsx`: nhận `lsp` qua prop thay vì tự tạo/tự `close()`; chỉ còn gọi
  `didOpen`/`didChange`/`didClose` theo `path` khi mount/đổi nội dung/unmount.
- Không làm phần "giữ undo/redo khi chuyển file" (vẫn `key={activePath}` remount `Editor`) — giữ
  nguyên quyết định hoãn từ đợt review trước, vì cần tách hẳn vòng đời `EditorState` khỏi
  component, rủi ro/công sức không tương xứng với phạm vi "tối ưu" lần này.

## Không làm trong đợt này
- Không lặp lại các mục bảo mật/mất dữ liệu/chessbook/test-CI đã "Xong" ở
  `plan_codebase_review_2026-09.md`.
- Không đụng desktop (Gio).
- Tách `App.css` theo domain và bật React Compiler: rủi ro/lợi ích chưa rõ bằng GĐ A/B, để làm
  đợt riêng nếu cần.

## Kiểm chứng
- `cd web && npm run build && npm test && npm run lint`.
- `CGO_ENABLED=0 go build ./cmd/typstify-server && go vet ./server/... && go test -race ./server/...`.
- Thủ công (sau khi deploy qua Docker/Caddy): DevTools Network xác nhận `Content-Encoding` nén và
  `Cache-Control` đúng cho asset/`index.html`; mở file A → gõ → chuyển file B → gõ: completion/hover
  ở B sẵn sàng ngay, không còn khoảng chờ khởi động lại LSP; `/preview/`, `/ws/lsp`, `/ws/agent`
  vẫn hoạt động bình thường qua Caddy.

## Kết quả kiểm chứng (29/09/2026)
- `go build ./cmd/typstify-server`, `go vet ./server/...`: sạch. `go test -race ./server/...`: PASS.
- `npm run lint`: không có warning mới ở các file đã sửa. `npm test`: 88/88 PASS.
- `npm run build`: `manualChunks` tách đúng — `codemirror-vendor-*.js` (510 kB, gzip 156 kB) và
  `markdown-vendor-*.js` (124 kB, gzip 38 kB) tách khỏi chunk `Editor`/`AgentChat`/
  `MarkdownImportModal` (giờ chỉ còn 8–26 kB mỗi chunk, là code glue thật của từng component).
  Đã kiểm ổn định cache: sửa 1 dòng comment trong `AgentChat.tsx`, build lại — hash
  `codemirror-vendor-*` và `markdown-vendor-*` **không đổi** (đã revert dòng sửa tạm sau khi kiểm).
- Chưa kiểm thủ công qua Docker/Caddy thật (môi trường dev không chạy container) — để người vận
  hành xác nhận `Content-Encoding`/`Cache-Control` và LSP theo phiên khi deploy lần tới.
