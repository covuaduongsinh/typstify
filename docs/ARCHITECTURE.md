# Kiến trúc Typstify

Typstify là trình soạn thảo Typst có ba "vỏ" dùng chung một lõi dịch vụ:

- **Desktop** (Go + Gio, `app.go`, `ui/`): ứng dụng chính.
- **Server** (Go, `cmd/typstify-server`, `server/`): API HTTP/WebSocket không giao diện.
- **Web** (`web/`, React 19 + Vite + CodeMirror 6): giao diện trình duyệt gọi vào server.

Thư viện cờ vua `chessbook/` (`@local/chessbook:0.1.0`) là nội dung Typst, không phải mã Go.

## 1. Sơ đồ lớp

```
 ┌────────── Desktop (Gio) ──────────┐   ┌──────── Web (React) ────────┐
 │ ui/  editor/  agent/view/  widgets│   │ web/src (CodeMirror, SVG)   │
 └───────────────┬───────────────────┘   └──────────────┬──────────────┘
                 │ gọi trực tiếp                         │ HTTP + WS
                 │                          ┌────────────▼─────────────┐
                 │                          │ server/  (transport)     │
                 │                          └────────────┬─────────────┘
                 └───────────────┬───────────────────────┘
                        ┌────────▼─────────┐
                        │ service.ServiceFacade (service/service.go)
                        └─┬──┬──┬──┬──┬──┬─┘
   bus/ settings/ Workspace  │  │  │  │  └─ agent.SessionManager (ACP)
        eventbus             │  │  │  └──── MCP server nhúng (service/mcp)
                  typst/pkg ─┘  │  └─────── lsp.PreviewService / lsp.Client (tinymist)
                       typst/export.CompileHelper ── CLI `typst`
```

`ServiceFacade` **không phụ thuộc Gio**: UI gắn vào qua các hook như `SetViewManager`. Nhờ đó `cmd/typstify-server` dùng lại nguyên lõi mà không kéo theo giao diện.

## 2. Các thành phần lõi

| Thành phần | File | Vai trò |
|---|---|---|
| `ServiceFacade` | `service/service.go` | Giữ eventbus, settings, `WorkspaceService`, `pkgService`, `previewSrv`, Dropbox client/syncer, `agent.SessionManager`, MCP server, `consoleState` |
| `WorkspaceService` | `service/workspace.go` | Trạng thái project lưu bbolt (lịch sử project, cây file/tab, `WorkspaceSettings`), theo dõi git, `syncManagedBibliography` |
| Event bus | `service/bus/bus.go`, `topics.go` | Pub/sub trong tiến trình (bọc `mustafaturan/bus/v3`) |
| File watcher | `service/filewatcher.go` | fsnotify → topic bus |
| Settings | `service/settings/` | Một file JSON, có timestamp theo section |
| Project store | `service/projectstore/store.go` | Trừu tượng hoá local / remote |
| Compile | `typst/compiler.go`, `typst/export/helper.go` | Chạy CLI `typst` |
| LSP + preview | `lsp/` | Điều khiển `tinymist` |
| Agent | `agent/` | Client ACP, phiên chat, quyền, MCP |

### Event bus

`Subscribe(instance, name, pattern, fn)` khoá theo `%p:name` và **panic nếu đăng ký trùng**. Có chế độ async qua channel. Các topic (`service/bus/topics.go`):

| Topic | Phát khi |
|---|---|
| `settings.updated` | Settings đổi |
| `statusbar.notification` | Cần hiện thông báo ở thanh trạng thái |
| `project.switched` / `project.create` | Đổi / tạo project |
| `workspace.file.changed` | fsnotify báo file ngoài `.git` đổi |
| `git.branch.changed` / `git.file.staged` | Sự kiện trong `.git` |
| `preview.toggle` | MCP tool `typst-previewer` yêu cầu bật/tắt preview |

## 3. Vòng đời project

`SetProjectDir` (`service/service.go`):
1. `Workspace.SwitchWorkspace`.
2. `lsp.GetLspClient(dir)` — singleton theo project, khởi chạy sớm; đổi workspace thì dừng client cũ.
3. Tạo `lsp.PreviewService` và `Start` trong goroutine (mặc định chế độ `document`).
4. Dừng `SessionManager` ACP cũ.

## 4. Agent và MCP

- `agent.SessionManager` spawn tiến trình agent qua stdio và nói chuyện ACP (`coder/acp-go-sdk`).
- MCP server nhúng (`service/mcp`, `agent/mcp_server.go`) mở cổng ngẫu nhiên hoặc tĩnh; được quảng bá cho agent qua `acp.McpServer{Http}`. Lỗi bind thì tắt MCP, không làm sập app.
- `agent/view/*` chỉ phụ thuộc interface `ChatSession`, nên dùng chung cho phiên local (`ACPSession`) và phiên remote (`RemoteChatSession`).

## 5. Đồng bộ desktop ↔ server

Ba đường độc lập, đều qua `service/remote/client.go` (Bearer token):

1. **Settings** — hợp nhất theo section bằng timestamp (`/api/settings/meta`).
2. **Remote agent** — desktop nối `/ws/agent` của server.
3. **Remote project** — duyệt/sửa file trên server qua `ProjectStore` (`RemoteStore`).

Dropbox là đường thứ tư, không đi qua server: `service/dropbox/syncer.go`. Chi tiết thuật toán trong [ALGORITHMS.md](ALGORITHMS.md).

## 6. Build và cấu hình

- Không có Justfile/Makefile. Chạy: `go run .`; test: `go test ./...`; `scripts/check-chessbook.sh` kiểm thư viện chessbook.
- Bản phát hành desktop build bằng `gogio`, cần CGO. Đóng gói bộ cài Windows: `scripts/package-desktop.ps1` (xem `docs/plans/plan_desktop_installer.md`).
- Cần `typst` và `tinymist`: đặt cạnh app hoặc chỉ đường dẫn trong Settings.
- Module Go: `looz.ws/typstify`, Go 1.25.
- Trên macOS/Linux, `init()` trong `app.go` lấy `PATH` từ login shell (`utils.LoginShellEnv`) để tìm `npx`.

## 7. Điểm cần nhớ khi mở rộng

- Thêm tính năng dùng chung cho cả desktop lẫn server → đặt trong `service/`, không import Gio.
- Thêm loại sự kiện → khai báo trong `service/bus/topics.go` và thêm vào `allTopics`.
- Thêm section settings → cài `Model` (`Save/Load/Validate`); muốn sync thì cài thêm `RemoteApplier.ApplyRemote`. `RemoteSettings` và section `agent` **không** được sync.
- Callback từ goroutine LSP/agent phải gọi `RefreshWindow()` để Gio vẽ lại.
