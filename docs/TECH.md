# Công nghệ và vận hành

## Phần mềm

| Lớp | Công nghệ |
|---|---|
| Desktop | Go 1.25 (`go.mod`: module `looz.ws/typstify`), Gio (`gioui.org`, `gioview`), gvcode (fork ở `internal/gvcode`), Chroma |
| Agent AI | `coder/acp-go-sdk` (ACP), `modelcontextprotocol/go-sdk` (MCP nhúng) |
| Lưu trữ / hệ thống | `bbolt`, `fsnotify`, `mustafaturan/bus/v3`, `sahilm/fuzzy`, `typstify/tpix-cli` |
| LSP / JSON-RPC | `golang.org/x/exp/jsonrpc2`, tinymist |
| Server web | Go `net/http` (mux có method), `coder/websocket` |
| Frontend | React 19, Vite 8, TypeScript 6, CodeMirror 6 + `codemirror-lang-typst`, `react-markdown`, vitest 3, oxlint |
| Công cụ ngoài | `typst` v0.15.1, `tinymist` v0.15.8 (ghim trong `Dockerfile`) |

Frontend không dùng router hay thư viện state (chỉ `useState` và ref); `App.tsx` là máy trạng thái loading → login → pickProject → workspace.

## Lệnh build / chạy

- Desktop: `go run .` (cần `typst`, `tinymist` cạnh app hoặc chỉ đường dẫn trong Settings). Bản phát hành dùng `gogio`, cần CGO; bộ cài Windows: `scripts/package-desktop.ps1`.
- Server: `go build -o bin/typstify-server ./cmd/typstify-server`.
- Frontend: `cd web && npm ci && npm run build` (`tsc -b && vite build`). Dev: `npm run dev` (proxy `/api`, `/preview`, `/ws` tới `TYPSTIFY_BACKEND`, mặc định `127.0.0.1:8080`). Kiểm thử: `npm test`. Lint: `npm run lint`.
- Kiểm thử Go: `go test ./...`; kiểm tra chessbook: `scripts/check-chessbook.sh`.
- Không có Justfile/Makefile.

## Cờ CLI và biến môi trường của server (`cmd/typstify-server/main.go`)

| Cờ | Biến môi trường | Mặc định / ý nghĩa |
|---|---|---|
| `-addr` | `TYPSTIFY_SERVER_ADDR` | `:8080` |
| `-password` | `TYPSTIFY_SERVER_PASSWORD` | Mật khẩu quản trị đơn |
| `-project` | `TYPSTIFY_PROJECT_DIR` | Project mở khi khởi động |
| `-static-dir` | `TYPSTIFY_STATIC_DIR` | `web/dist` |
| `-project-root` | `TYPSTIFY_PROJECT_ROOT` | Giới hạn open/create project trong thư mục này |

Server bỏ biến mật khẩu khỏi môi trường tiến trình con, tắt êm khi nhận SIGTERM, `ReadHeaderTimeout` 10 giây, không đặt timeout đọc/ghi tổng vì WebSocket và biên dịch chạy lâu.

## Docker / triển khai

- `Dockerfile` nhiều giai đoạn: `node:22-slim` (frontend) → `golang:1.25-bookworm` (backend, `CGO_ENABLED=0`) → tải `typst` + `tinymist` ghim phiên bản → `chesslib` (`board-n-pieces` 0.9.0 và font cờ) → tuỳ chọn Antigravity agent (`WITH_ANTIGRAVITY=true`) → image chạy `node:22-slim`.
- Biến đặt sẵn trong image: `TYPSTIFY_STATIC_DIR=/app/web/dist`, `TYPSTIFY_PROJECT_DIR=/data/project`, `TYPSTIFY_PROJECT_ROOT=/data`, `TYPSTIFY_SERVER_ADDR=:8080`. `TYPSTIFY_SERVER_PASSWORD` cố ý không có mặc định. Dữ liệu ở volume `/data`.
- `docker-compose.yml`: dịch vụ `typstify` chỉ bind `127.0.0.1:8080`; `caddy` (`Caddyfile`) chấm dứt TLS tự động và là dịch vụ duy nhất mở 80/443. Không mở 8080 trực tiếp vì server không tự làm TLS.
- Tài liệu liên quan: `docs/web-server.md`, `docs/USER_AUTH_GUIDE.md`.
