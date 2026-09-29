# Nhân bản phần mềm và thêm module mới

Hướng dẫn thực hành để (1) thêm module/tính năng vào Typstify và (2) nhân bản Typstify thành sản phẩm khác. Các đường dẫn lấy từ khảo sát 2026-09-29; kiểm lại bằng grep khi làm.

## 1. Mô hình để bắt chước

Một tính năng đầy đủ đi qua các lớp sau, từ dưới lên:

```
typst/ | lsp/ | agent/         logic thuần, không phụ thuộc UI
        ↓
service/<tên>/                 dịch vụ + settings Model + topic bus
        ↓  (ServiceFacade nối vào, không import Gio)
server/<tên>_api.go            route HTTP/WS  ──→ web/src/api + components
ui/<tên>/                      view Gio       ──→ đăng ký trong ui/ui.go, navpanel, palette
        ↓
service/mcp/                   (tuỳ chọn) công cụ MCP cho AI agent
docs/  +  docs/HISTORY.md      bắt buộc
```

Mẫu tham chiếu nên đọc trước:
- Dịch vụ + settings + sync: `service/dropbox/` (syncer), `service/fonts/fonts.go` (dùng chung desktop/web).
- Route + UI web: `server/export_api.go` ↔ `web/src/components/ExportButton.tsx`.
- Công cụ MCP: `service/mcp/editor_service.go`, `package_service.go`.
- Giao thức WebSocket: `server/agent_ws.go` ↔ `web/src/lib/agentClient.ts`.

## 2. Checklist thêm một tính năng

1. **Logic**: đặt vào package không phụ thuộc Gio (`typst/`, `service/<tên>/`). Viết test cạnh mã.
2. **Facade**: thêm phương thức vào `service.ServiceFacade` (`service/service.go`). Không import `gioui.org` trong `service/`.
3. **Settings** (nếu có cấu hình): thêm `Model` với `Save/Load/Validate` trong `service/settings/`. Nếu cần đồng bộ desktop↔web, cài `RemoteApplier.ApplyRemote(raw, remoteUpdatedAt)` và thêm section vào danh sách sync ở `ui/settings/sync.go`. Section chứa dữ liệu cục bộ máy (như `agent`) hoặc bí mật (như `tpix`) thì không sync và không expose.
4. **Sự kiện**: nếu module cần báo cho UI, thêm topic ở `service/bus/topics.go`; nhớ khoá đăng ký `%p:name` phải duy nhất.
5. **Server**: thêm handler trong `server/`, đăng ký trong `routes()` của `server/server.go`, bọc `auth.require`.
   - File/thư mục: qua `resolveInRoot`.
   - Compile nặng: `acquireCompile`.
   - Body lớn: kiểm `withBodyLimit`.
   - Không trả secret.
6. **Web**: kiểu dữ liệu trong `web/src/api/types.ts`, gọi qua `api/client.ts`, thành phần trong `web/src/components/`; panel nặng dùng lazy import như trong `Workspace.tsx`; chuỗi giao diện đưa vào `lib/vi.ts`/`lib/i18n.ts`.
7. **Desktop**: view trong `ui/<tên>/`, đăng ký ở `ui/ui.go`/`home.go`; chuỗi vào `i18n/translations/locales/*/messages.gotext.json`; callback từ goroutine nền phải `RefreshWindow()`.
8. **MCP** (nếu AI cần dùng): `AddMcpTool[In,Out]` trong `service/mcp/`, đăng ký qua `RegisterToolProvider`.
9. **Tài liệu**: cập nhật `docs/MODULES.md`, `docs/API.md` (nếu có route), `docs/ALGORITHMS.md` (nếu có thuật toán), thêm dòng `docs/HISTORY.md`, gạch mục ở `docs/ROADMAP.md`.
10. **Kiểm**: `go build ./... && go vet ./...`, `go test ./...`, `cd web && npm run lint && npm test && npm run build`, và `scripts/check-chessbook.sh` nếu đụng chessbook.

## 3. Thêm module vào thư viện chessbook

1. Tạo `chessbook/lib/<tên>.typ`, export từ `lib.typ` (kiểm cách `lib.typ` re-export các module hiện có).
2. Dùng lại hàm bàn cờ trong `symbols.typ` (`chess-board`, `nag`, `turn-indicator`); không chép mã vẽ bàn cờ lần thứ bảy.
3. Chấp nhận cả `turn` và `to-move` cho tham số lượt đi, như các hàm hiện có.
4. Thêm demo/template vào `chessbook/templates/` hoặc `demo_collection/` để `scripts/check-chessbook.sh` biên dịch.
5. Cập nhật danh sách hàm trong `CLAUDE.md` **và** `AGENTS.md` (giữ giống nhau), rồi `docs/CHESSBOOK.md`.
6. Cài lại bản cục bộ: `scripts/install-chessbook.sh` hoặc `.ps1`; Docker image đã gồm sẵn thư viện dưới tên `@local/chessbook:0.1.0`. Đổi số phiên bản package thì phải đổi ở mọi nơi nhắc `0.1.0` (Dockerfile, scripts, tài liệu, template, snippet).
7. Luật cho AI phải giữ: import ở dòng 1, không `#let` mock, escape `\#`.

## 4. Nhân bản thành sản phẩm khác (fork/rebrand)

| Việc | Ở đâu (kiểm lại bằng grep) |
|---|---|
| Đổi tên module Go | `go.mod` (`looz.ws/typstify`) và mọi import; dùng công cụ đổi hàng loạt rồi `go build ./...` |
| Tên/biểu tượng ứng dụng | `version/appicon.png`, README, tiêu đề web (`web/index.html`, `BrandMark.tsx`, `LoginPage.tsx`) |
| Bảng màu, giao diện | design tokens/theme trong `web/src/lib/theme.ts`, `editorTheme.ts` và CSS |
| Ngôn ngữ giao diện | `web/src/lib/vi.ts`, `i18n.ts`; desktop `i18n/translations/` |
| Dịch vụ ngoài | cấu hình Tpix, Dropbox app key/secret, endpoint đăng ký thiết bị/kiểm tra cập nhật (`service/net/`) — thay bằng của bạn hoặc tắt |
| Tên thư viện Typst | `@local/chessbook` trong `chessbook/`, `scripts/install-chessbook.*`, `Dockerfile`, `CLAUDE.md`/`AGENTS.md` |
| Triển khai | `Dockerfile`, `docker-compose.yml`, `Caddyfile`, `.env.example` (`TYPSTIFY_SERVER_PASSWORD`, `TYPSTIFY_DOMAIN`), `docs/web-server.md` |
| Phiên bản công cụ | typst v0.15.1, tinymist v0.15.8 được ghim trong `Dockerfile`; nâng cấp phải chạy lại `check-chessbook.sh` |
| Giấy phép | Apache-2.0 (`LICENSE`): giữ thông báo bản quyền gốc khi phân phối lại |

Thứ tự khuyến nghị: (1) sao chép repo, giữ lịch sử git; (2) đổi module Go và build; (3) đổi thương hiệu web; (4) thay dịch vụ ngoài; (5) bỏ/giữ chessbook; (6) chạy toàn bộ kiểm thử; (7) sửa `CLAUDE.md`, `README.md`, `docs/`.

## 5. Nhân bản chỉ bản server (không desktop)

Server và web không cần CGO và không import Gio: `go build ./cmd/typstify-server` (Docker build với CGO tắt). Có thể bỏ `ui/`, `editor/`, `widgets/`, `internal/gvcode` khỏi bản sao **chỉ khi** không package nào trong đường phụ thuộc của `cmd/typstify-server` import chúng; kiểm bằng `go list -deps ./cmd/typstify-server`.
