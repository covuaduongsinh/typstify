# Giai đoạn 3b: mỗi người một project cùng lúc

> Thuộc `plan_typst_app_parity_2026-10.md`. Viết và **chốt với chủ dự án 2026-10-26**: tối đa 3
> project mở cùng lúc (mặc định, quản trị viên đổi được), một project mỗi phiên trình duyệt (tab
> cũ được báo tải lại). Đây là thay đổi kiến trúc lớn nhất của kế hoạch (tách `ServiceFacade`
> theo project).

## Bài toán

Server hiện có **một project hiện hành dùng chung** (`ServiceFacade.currentProjectDir`). Khi một
người mở project khác (`POST /api/projects/{id}/open` → `SetProjectDir`), mọi người khác đang
dùng server bị chuyển theo:

- Mọi route project (`server/files_api.go`, `history_api.go`, `export_api.go`, `sync_handler.go`,
  `upload_api.go`, `lsp_ws.go`, `agent_ws.go`, preview… khoảng 20 chỗ) lấy thư mục qua
  `s.projectRoot()` = project hiện hành chung.
- **Lỗi đang có**: quyền (`currentProjectRole`, `roleGate`) được tính trên project người khác vừa
  mở, không phải project người gọi đang xem. Người là editor của cả X và Y, đang mở X, có thể ghi
  nhầm vào Y sau khi người khác chuyển server sang Y. Cộng tác (`/ws/collab`) đã tự chặn trường
  hợp này (so `?project=`), các route REST thì chưa.
- Một tiến trình `tinymist` duy nhất (`lsp.GetLspClient` là singleton, đổi workspace thì giết
  client cũ), một `PreviewService`, một file watcher, một MCP server gắn với project hiện hành.

Đã sẵn sàng cho nhiều project: phòng cộng tác khoá theo (thư mục project, đường dẫn tương đối);
phiên ACP tạo theo `cwd`; kho project/quyền (`service/projectstore`) độc lập với project đang mở.

## Thiết kế

### 1. Project gắn với phiên đăng nhập

- `session` (`server/auth.go`) thêm trường `Project` (id project trong kho). `handleProjectOpen`
  ghi vào phiên thay vì gọi `SetProjectDir`. Mỗi trình duyệt/thiết bị đăng nhập có project riêng.
- `s.projectRoot()` → `s.projectFor(r)`: đọc phiên → project → runtime (mục 2). Quyền
  (`currentProjectRole`, `roleGate`, `hasProjectRole`, `collabRole`) tính trên project của chính
  phiên đó, sửa luôn lỗi ở trên.
- Token Bearer (đồng bộ desktop ↔ web, `RemoteChatSession`) không có phiên: request phải nêu
  project (`?project=<id>`), server kiểm quyền như route `/api/projects/{id}/...`.
- **Chế độ không xác thực** (`-password` rỗng, bản desktop Tauri chạy sidecar) và **bản desktop
  Gio**: chỉ có một "phiên mặc định", hành vi giữ nguyên như hiện nay.
- Nhiều tab cùng trình duyệt chung cookie nên chung project. Để tab cũ không ghi nhầm khi tab mới
  mở project khác: `api/client.ts` gửi header `X-Typstify-Project: <id>` (một chỗ duy nhất cho
  mọi `fetch`; WebSocket/iframe dùng `?project=`); server trả 409 nếu không khớp project của phiên,
  giao diện báo "Dự án đã đổi ở tab khác" kèm nút tải lại.

### 2. `ProjectRuntime`: tài nguyên theo từng project

Gói mới trong `service/` (không phụ thuộc Gio):

```go
type ProjectRuntime struct {
    Dir     string
    LSP     *lsp.Client          // tinymist riêng
    Preview *lsp.PreviewService
    Watcher *WorkspaceFileWatcher
    MCP     *agent.McpServer     // công cụ MCP trỏ đúng project
    // + cài đặt workspace (.typstify), trạng thái git
}
```

- `ServiceFacade.Runtimes().Acquire(dir) (*ProjectRuntime, release func())`: tạo khi cần, đếm số
  phiên/WebSocket đang dùng. Phiên đổi project thì nhả runtime cũ.
- **Giới hạn RAM**: mỗi `tinymist` tốn khoảng 150–300 MB. Số runtime sống tối đa là một cài đặt
  máy chủ (mặc định **3**). Vượt giới hạn thì đóng runtime nhàn rỗi dùng lâu nhất; không còn cái nào
  nhàn rỗi thì từ chối mở với thông báo rõ ("Máy chủ đang mở tối đa N dự án"). Runtime không ai
  dùng quá 10 phút thì tự đóng.
- `lsp`: bỏ singleton `lspClient`, thêm `lsp.NewClient(workspace)`; `GetLspClient` còn lại như lớp
  mỏng cho bản desktop Gio (trỏ runtime mặc định) để không phải sửa `ui/`.
- `WorkspaceService` tách phần "project hiện hành" (watcher, settings cache, git) vào runtime;
  phần dùng chung (`recent.db`, danh sách gần đây) giữ nguyên.

### 3. Trợ lý AI và MCP

- Một tiến trình agent ACP dùng chung (giữ như hiện nay, gói thuê bao một người); mỗi phiên ACP
  tạo với `cwd` = project của phiên web, và danh sách MCP server truyền vào `session/new` là MCP
  của đúng runtime đó (sửa `listMcpServer` nhận runtime thay vì đọc project chung).
- `getActiveDocument` của MCP hiện đọc view Gio: bản web không có, giữ nguyên.

## Thứ tự làm (mỗi bước build + test xanh, commit riêng)

1. `lsp`: client theo workspace, bỏ singleton; desktop Gio vẫn chạy như cũ.
2. `ProjectRuntime` + registry (đếm tham chiếu, giới hạn, dọn nhàn rỗi), runtime mặc định cho
   desktop/không xác thực. Unit test cho registry.
3. Server: project theo phiên, thay `projectRoot()`/quyền theo project của phiên, header
   `X-Typstify-Project`. Test Go: hai phiên, hai project, sửa song song; quyền tính đúng project;
   phiên viewer của X không ghi được vào Y.
4. Agent/MCP theo runtime.
5. Frontend: header + 409 "đổi ở tab khác"; cài đặt "số dự án mở tối đa" trong trang quản trị.
6. Tài liệu (`docs/MODULES.md`, `docs/MEMORY.md`, `HISTORY.md`).

**Tiến độ (2026-10-26)**: xong cả 6 bước (`84b3814`, `d5861d7`, `9c2b91c`, và commit frontend + tài liệu). Phần kiểm chứng còn thiếu ghi ở mục 3b của `docs/HISTORY.md`.

## Kiểm chứng

- `go test ./...` (phần không cần X11), `go vet`, `npm test`, build Docker.
- Hai trình duyệt (hai tài khoản) mở hai project khác nhau, cùng gõ và xem preview: không ai bị
  chuyển project; tinymist đúng hai tiến trình; mở project thứ 4 khi giới hạn 3 thì đóng runtime
  nhàn rỗi hoặc báo lỗi rõ.
- Hai tab cùng trình duyệt: tab cũ nhận 409 và lời nhắc tải lại, không ghi nhầm.
- Bản desktop Gio và bản Tauri (sidecar, không xác thực) hoạt động như trước.
- Đo RAM server với 1, 2, 3 project mở.

## Đã chốt (2026-10-26)

1. Mặc định tối đa **3** project mở cùng lúc; quản trị viên đổi trong cài đặt máy chủ.
2. **Một project mỗi phiên trình duyệt**: tab cũ nhận 409 và lời nhắc tải lại, không hỗ trợ mỗi
   tab một project.
