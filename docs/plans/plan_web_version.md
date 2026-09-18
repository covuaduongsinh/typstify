# Kế hoạch: Phiên bản Web cho Typstify

## Bối cảnh

Typstify hiện là ứng dụng desktop viết bằng Go + [Gio](https://gioui.org/) (`app.go`, `ui/`), chạy trên Windows/macOS/Linux. Người dùng muốn có thêm một **phiên bản web** để dùng Typstify từ trình duyệt.

Sau khi khảo sát codebase (3 agent song song đọc `ui/`, `lsp/`, `agent/`, `service/`, `typst/`, `utils/`), kết luận khả thi/hiệu quả như sau:

- **Khả thi: có**, nhưng **không** theo cách "biên dịch lại UI Gio hiện tại sang WASM". Ba tính năng lõi (biên dịch Typst, LSP `tinymist`, AI Agent) đều spawn tiến trình native qua `os/exec` — thứ không tồn tại trong sandbox WASM của trình duyệt. Ngoài ra, preview hiện tại nhúng **webview OS gốc** (`ui/preview/webview.go`) trỏ vào `http://127.0.0.1:<port>` do `tinymist` tự chạy — không có tương đương trong trình duyệt. File chooser (`explorer.FileChooser` ở `ui/ui.go:61-70`, `ui/welcome.go:225`...) cũng gọi dialog OS gốc. Do đó **phải tách kiến trúc client-server thật sự**: một backend Go chạy trên máy chủ (giữ nguyên toàn bộ logic nghiệp vụ hiện có), và một **frontend web hoàn toàn mới** nói chuyện với backend qua HTTP/WebSocket.
- **Hiệu quả: cao**, nếu chọn đúng mô hình triển khai. Người dùng đã chốt mô hình **self-hosted, một người dùng/instance** (giống `code-server`/Jupyter): backend chạy trên máy cá nhân, VPS hoặc container riêng của người dùng, truy cập từ bất kỳ thiết bị nào qua trình duyệt. Mô hình này **không cần** giải quyết bài toán cách ly multi-tenant (nhiều người dùng lạ chia sẻ hạ tầng), nên tái dùng được gần như nguyên vẹn `service/`, `lsp/`, `typst/`, `agent/` — chỉ cần thêm một tầng API mới, không phải viết lại backend.
- Phạm vi v1 đã chốt: **đầy đủ tính năng** (editor + biên dịch/preview + LSP + AI Agent chat), không cắt bớt.

Phát hiện quan trọng làm giảm rủi ro của kế hoạch:
1. `agent/session.go` (`ACPSession.SubscribeUpdates`/`PublishUpdate`, dòng 394-473) đã có sẵn mô hình channel/subscriber cho mọi sự kiện của AI Agent (tin nhắn, tool call, permission request, usage...) — cắm thẳng một WebSocket writer goroutine vào đây là đủ, không cần viết lại logic ACP.
2. `lsp/client.go` (`Complete`, `Hover`, dòng 483-553) đã là các lời gọi `jsonrpc2.Call(...).Await(...)` đồng bộ, dễ ánh xạ 1-1 sang RPC-over-WebSocket.
3. Preview hiện tại (`lsp/previewer.go:52-185`) hoạt động bằng cách gọi lệnh `tinymist.startDefaultPreview` qua LSP, nhận về `staticServerPort`, rồi desktop app **mở một webview trỏ vào `127.0.0.1:<port>` đó**. Tinymist tự phục vụ toàn bộ trang preview (HTML/JS/SVG + WebSocket đồng bộ cuộn) — nghĩa là preview *vốn đã là một trang web*. Với bản web, ta chỉ cần **reverse-proxy** cổng đó ra `/api/preview/...` trên chính backend, rồi nhúng bằng `<iframe>` — không cần tự vẽ lại trang preview. Đây là pattern đã có người triển khai thành công trong thực tế cho các web IDE khác (ví dụ dự án `dsh-typst-preview` proxy preview server của tinymist qua origin của app; xem thêm [Websocket connection for preview on code-server · Issue #625 · Myriad-Dreamin/tinymist](https://github.com/Myriad-Dreamin/tinymist/issues/625)). Lưu ý đã biết: tinymist cần được cấu hình đường dẫn preview base URL/WS path phù hợp khi đứng sau reverse proxy, nếu không nó sẽ cố kết nối thẳng `localhost` từ phía trình duyệt.
4. Theo agent khảo sát build/container: các package nghiệp vụ (`service/`, `agent/`, `typst/`, `lsp/`) **không có cgo**, file cgo duy nhất là `widgets/filetree/clipboard_darwin.go` (chỉ macOS). Nghĩa là build một binary "headless" cho Linux (không import Gio) là khả thi về mặt kỹ thuật mà không cần CGO/GPU.

---

## Kiến trúc đề xuất

```
Trình duyệt (bất kỳ thiết bị nào)
        │  HTTPS/WSS
        ▼
┌─────────────────────────────────────────────┐
│  cmd/typstify-server (binary mới, headless)  │
│                                               │
│  server/  (package mới)                      │
│   ├─ auth       - token/password, session    │
│   ├─ files_api  - REST: workspace, project,  │
│   │                open/create/list-recent   │
│   ├─ settings_api - REST: cấu hình           │
│   ├─ pkg_api    - REST: tìm/cài Typst package │
│   ├─ lsp_ws     - WS: completion/hover/       │
│   │                diagnostics + didOpen/     │
│   │                didChange                  │
│   ├─ agent_ws   - WS: chat AI Agent (bọc      │
│   │                ACPSession.SubscribeUpdates)│
│   └─ preview_proxy - reverse-proxy HTTP+WS    │
│       sang tinymist preview server nội bộ     │
│                                               │
│  (tái dùng nguyên vẹn, không sửa nhiều)       │
│   service/  lsp/  typst/  agent/              │
└─────────────────────────────────────────────┘
        │ os/exec (không đổi)
        ▼
   typst CLI, tinymist, AI agent CLI (npx/uvx/binary)
```

Frontend: một SPA web **mới** (không tái dùng code Gio), vì lý do đã nêu ở Bối cảnh (webview native, file dialog native, `ServiceFacade` gọi đồng bộ trong `Layout()` mỗi frame — không có ranh giới async sẵn có để "nối mạng"). Đề xuất stack: **Vite + TypeScript + React**, dùng **CodeMirror 6** làm editor (nhẹ, chạy tốt trên mobile, có kiến trúc LSP-client extension rõ ràng). Rủi ro lớn nhất của toàn kế hoạch nằm ở đây — xem mục Rủi ro.

---

## Phân kỳ triển khai

### Giai đoạn 0 — Tách backend khỏi kiểu dữ liệu Gio (nền tảng) **[Đã triển khai]**

`ServiceFacade` (`service/service.go:37-57`) hiện có field `vm view.ViewManager` và `fileChooserBuilder func() *explorer.FileChooser` — đều là kiểu của `gioui-plugins`/`gioview`, chỉ dùng cho desktop UI. Cần:
- Thêm một entrypoint headless mới `cmd/typstify-server/main.go`, mô phỏng `app.go` nhưng **không** import `gioui.org/app`, không gọi `ui.NewUI`/`app.Main()`. Khởi tạo `service.NewService(ctx)` như cũ, `SetProjectDir` như cũ.
- Các method dùng `vm`/`fileChooserBuilder` (`RequestSwitch`, `RefreshWindow`, `InitFileChooser`, `FileChooser`) chỉ đơn giản không được gọi trong đường chạy headless — không bắt buộc phải refactor sâu ngay ở v1, miễn là các API mới ở Giai đoạn 1 không đụng tới chúng. Refactor tách hẳn "core service" khỏi các field UI-only có thể để ở giai đoạn dọn dẹp sau.
- Vì mô hình đã chốt là **một người dùng/instance**, không cần thêm khái niệm multi-user vào `service/settings/store.go`, `service/workspace.go` (bbolt `recent.db`, `.typstify/settings.json` mỗi project) — dùng nguyên vẹn, chỉ cần đảm bảo `configRoot()`/`os.UserConfigDir()` trỏ đúng thư mục ghi được khi chạy trong container (set qua biến môi trường `HOME`/`XDG_CONFIG_HOME`).

### Giai đoạn 1 — Backend API (REST + WebSocket) **[Đã triển khai và xác minh end-to-end]**

Đã build package `server/` với đầy đủ các thành phần bên dưới, và xác minh trực tiếp (không chỉ đọc code): chạy `cmd/typstify-server` thật với một project mẫu, gọi REST qua `curl` (auth/login, workspace tree, đọc/ghi/đổi tên/xoá file, chặn path traversal, settings), và nối `/ws/lsp`, `/ws/agent` bằng một script Node.js — nhận được **diagnostics/completion/hover thật từ tinymist** và **một phiên ACP Agent thật khởi tạo thành công** (agent CLI qua `npx` chạy được). `pkg_api` (quản lý package Typst) chưa làm, để lại Giai đoạn 5 như kế hoạch cho phép.

Package mới `server/`:
- **Auth**: middleware token đơn giản (biến môi trường `TYPSTIFY_SERVER_TOKEN` hoặc form đăng nhập bằng password), áp dụng cho mọi route. Bắt buộc vì service này có quyền đọc/ghi file, chạy compiler và **chạy AI agent CLI có thể tốn phí API** — không được để mở công khai không xác thực.
- **`files_api`**: REST cho duyệt cây thư mục project, đọc/ghi nội dung file, tạo/xoá/đổi tên, tạo project mới, danh sách "recent projects" — bọc quanh `service/workspace.go` (`WorkspaceService`) và `service/filewatcher.go` (đẩy sự kiện thay đổi file qua WS thay vì Gio `EventBus` nội bộ).
- **`settings_api`**: REST get/set cho `service/settings` (General, Typst, Lsp, AcpAgent, Tpix).
- **`pkg_api`**: REST bọc `typst/pkg` (tìm kiếm/cài đặt package Typst qua `tpix-cli`) — có thể để cuối Giai đoạn 5 nếu cần rút gọn phạm vi.
- **`lsp_ws`**: một kết nối WebSocket mỗi phiên editor, forward các lời gọi `Complete`/`Hover`/`DocumentSymbols` (`lsp/client.go`) và nhận diagnostics push (`lsp/client.go:316-325` `updateDiagnostics`), cùng `didOpen`/`didChange`/`didSave` từ trình soạn thảo trên trình duyệt — dùng lại nguyên `documentCache` (`lsp/document.go`) ở phía server.
- **`agent_ws`**: một kết nối WebSocket mỗi phiên chat, gọi `SessionManager.NewSession`/`StartACPSession` (`service/service.go:333-361`), rồi `ACPSession.SubscribeUpdates` (`agent/session.go:404-473`) để bơm mọi update (tin nhắn, tool call, plan, permission request) thẳng vào WS; chiều ngược lại nhận `Prompt`, phản hồi permission (`RequestPermission`), `Cancel`.
- **`preview_proxy`**: dùng `net/http/httputil.ReverseProxy` trỏ tới `s.PreviewService().Address()` (`lsp/previewer.go`), hỗ trợ cả HTTP lẫn nâng cấp WebSocket (Go's `ReverseProxy` hỗ trợ sẵn từ Go 1.12). Cần khảo sát/khớp cấu hình base-path của tinymist preview server để nó không cố redirect thẳng về `localhost` khi đứng sau proxy (vấn đề đã ghi nhận công khai ở dự án tinymist khi dùng với code-server).

### Giai đoạn 2 — Frontend web MVP **[Đã triển khai và xác minh bằng trình duyệt thật]**

Thư mục mới `web/` (Vite + TypeScript + React). Rủi ro #1 (editor/syntax highlighting) đã được giải quyết gọn hơn dự kiến: tìm thấy package cộng đồng **`codemirror-lang-typst`** (bản `typst_lezer()`, không cần WASM, có sẵn autocomplete tĩnh) — không cần tự viết grammar hay dựa vào semantic tokens của LSP như phương án dự phòng đã tính.

- Màn đăng nhập bằng password (`LoginPage.tsx`).
- File tree lazy-load + mở/tạo project (`FileTree.tsx`, `ProjectPicker.tsx`).
- Trình soạn thảo CodeMirror 6 (`Editor.tsx`): syntax highlighting qua `codemirror-lang-typst`, completion/hover qua `lib/lspClient.ts` (RPC request/response tự tương quan bằng `id` trên cùng một WebSocket `/ws/lsp`), diagnostics đẩy về qua `@codemirror/lint` (`setDiagnostics`), debounce 250ms cho `didChange`, `Ctrl+S` lưu file + đồng bộ.
- Preview pane: `<iframe src="/preview/">` (`PreviewPane.tsx`), poll `/api/preview/status` tới khi sẵn sàng.
- Trang cài đặt (General/Typst/Lsp/AcpAgent/Tpix) bind trực tiếp vào `settings_api` (`SettingsPanel.tsx`).
- Đã build (`npm run build`) và chạy thử **end-to-end thật trong Chrome** (qua công cụ browser automation) với một project mẫu và `tinymist`/`typst` thật cài trên máy: đăng nhập, mở project, gõ code sai cú pháp thấy diagnostics thật từ tinymist, gõ `#` thấy completion, preview hiển thị đúng tài liệu đã biên dịch. Trong lúc test đã phát hiện và sửa 2 lỗi thật ở tầng preview proxy (xem mục Rủi ro #2).

### Giai đoạn 3 — AI Agent chat trên web **[Đã triển khai; xác minh một phần]**

- Panel chat mô phỏng lại UX của `agent/view/chat.go`, `message.go` (native) nhưng viết mới bằng React (`AgentChat.tsx`), nối `agent_ws`: hiển thị stream tin nhắn/tool-call, khung xin phép (permission), hiển thị usage. Kiểu dữ liệu ACP (`ContentBlock`, `ToolCall`, `PermissionOption`...) đối chiếu trực tiếp với JSON tag trong `github.com/coder/acp-go-sdk@v0.13.5` để đảm bảo đúng hình dạng wire, không đoán.
- Đã xác minh: kết nối `/ws/agent` thành công, phiên ACP thật được tạo (`SessionManager.Start` → `NewSession`), nhận đúng `ready` event kèm session ID — cả từ script Node.js lẫn từ trình duyệt thật.
- **Chưa xác minh trọn vẹn round-trip prompt → phản hồi trong trình duyệt**: agent đã cấu hình sẵn trên máy test (`antigravity-acp`, một agent thử nghiệm riêng của người dùng, không phải `Claude Code` mặc định của Typstify) không phản hồi trong phiên browser-automation này; đồng thời quan sát thấy cả các request tĩnh không liên quan (`GET /`) cũng có độ trễ ~15s bất thường trong riêng sandbox trình duyệt tự động đó — nhiều khả năng là đặc thù môi trường test, không phải lỗi code. Không sửa `settings.json` thật của người dùng (dùng chung với bản desktop) để đổi sang agent khác trong lúc test. **Cần làm lại bước xác minh này với agent mặc định (`npx @agentclientprotocol/claude-agent-acp`) trong môi trường không bị giới hạn mạng trước khi coi Giai đoạn 3 là hoàn tất.**
- Về agent runtime: agent mặc định (`defaultAgentConfig` ở `service/service.go:379-383`) chạy qua `npx` → image Docker cần Node.js. Ở v1 chỉ hỗ trợ chính thức các agent phân phối qua `npx` (đã đủ dùng Claude Code, và phần lớn agent phổ biến trong registry); agent dạng `uvx` (Python) hoặc `binary` riêng nền tảng (ví dụ `antigravity_bridge.exe` — hiện chỉ có bản Windows) để lại giai đoạn sau.

### Giai đoạn 4 — Đóng gói & triển khai self-host **[Đã triển khai; build Docker thật chưa chạy được trong sandbox này]**

- `Dockerfile` multi-stage (4 stage: `frontend` build Vite, `backend` build Go, `tools` tải `typst`/`tinymist` binary chính chủ từ GitHub Releases, image cuối `node:22-slim` để có sẵn `npx` cho agent mặc định). Trước khi viết Dockerfile đã **tự sửa xong rủi ro #5** (xem ở trên) nên build Linux không cần cgo/X11 — đúng như giả định ban đầu của kế hoạch, chỉ là phải tự tay đi sửa code trước.
- `docker-compose.yml` + `.env.example` + `.dockerignore` — mặc định chỉ bind `127.0.0.1:8080`, bắt buộc set `TYPSTIFY_SERVER_PASSWORD`, khuyến nghị rõ trong docs là đặt sau reverse proxy TLS nếu public ra Internet.
- `docs/web-server.md` — hướng dẫn tự host, không đụng tới `README.md` gốc.
- **Đã xác minh riêng lẻ**: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/typstify-server` thành công (xem rủi ro #5); `npm run build` cho frontend thành công; URL tải `typst`/`tinymist` trong Dockerfile trả về đúng (kiểm bằng `curl -I`, theo redirect 302 bình thường của GitHub Releases); cấu trúc archive `typst-*.tar.xz` đã kiểm tra thực tế khớp với đường dẫn dùng trong Dockerfile.
- **Chưa xác minh được**: lệnh `docker build`/`docker compose up` thật, vì Docker daemon không chạy trong sandbox này (`docker build` báo lỗi không kết nối được `dockerDesktopLinuxEngine`) — chỉ có Docker CLI, không có daemon. **Cần build thử thật trên máy có Docker daemon trước khi coi Giai đoạn 4 là hoàn tất.**

### Giai đoạn 5 — Hoàn thiện **[3/4 mục đã triển khai và xác minh; 1 mục để lại có chủ đích]**

- ~~Xuất file (PDF/PNG/SVG) tải về từ trình duyệt~~ **[Xong]**: `GET /api/export` (`server/export_api.go`) bọc nguyên `typst/export.CompileHelper` — cùng helper mà dialog export của bản desktop dùng, không viết lại logic biên dịch. Nhiều trang (PNG/SVG) tự động nén thành `.zip`. Đã test thật qua UI: tải PDF thật (1 trang, hợp lệ) và PNG thật (magic bytes đúng) từ trình duyệt.
- ~~Quản lý package Typst qua UI web~~ **[Xong]**: `server/pkg_api.go` bọc `typst/pkg.TypstPkgService` (search/cached/detail/download/pull-deps), panel `PackageManager.tsx` mới. Đã test thật: tìm kiếm "cetz" trả về kết quả thật từ Tpix, tải một package thật, trạng thái "Cached" cập nhật đúng sau khi tải xong.
- ~~i18n~~ **[Xong, phạm vi có chủ đích thu hẹp]**: `POST /api/i18n` (`server/i18n_api.go`) dịch một danh sách key bằng đúng catalog `i18n/translations/` (chỉ hỗ trợ `en-US`/`zh-CN`/`de` như bản desktop — **không có tiếng Việt**, đây là giới hạn của catalog gốc, không phải web thêm vào). Do catalog được đánh key theo đúng câu tiếng Anh gốc trong UI desktop, và UI web mới viết chữ khác desktop ở phần lớn chỗ, chỉ áp dụng `t()` cho các nhãn **trùng khớp chính xác** với key có sẵn (`Sign In`, `Settings`, `Export`, `AI Assistant`, `Cancel`) — đã xác minh thật: gọi API trả về đúng `"KI-Assistent"/"Einstellungen"/"Exportieren"/"Anmelden"` (de) và `"AI助手"/"设置"/"导出"/"登陆"` (zh-CN). Phủ toàn bộ UI (mọi nhãn, mọi component) không nằm trong phạm vi đã làm — cần một đợt riêng để hoặc viết lại UI theo đúng từ vựng desktop, hoặc bổ sung key mới vào catalog gốc.
- **Để lại có chủ đích, chưa làm**: dialog bibliography, indentation, diff view. Đây là các tính năng đặc thù desktop, phục vụ nhu cầu nâng cao (quản lý `.bib`, đổi kiểu thụt lề nhanh theo file, xem diff khi agent sửa file) — không nằm trên đường đi chính "soạn thảo + biên dịch + xem trước + chat AI" của MVP web, và mỗi cái cần thiết kế API mới riêng (đặc biệt diff view — cần expose `editor/diff.go` qua một endpoint chưa tồn tại). Ưu tiên thấp hơn 3 mục trên trong ngân sách thời gian của đợt triển khai này.

---

## Rủi ro & điểm cần xử lý sớm

1. ~~**Editor + syntax highlighting Typst cho web**~~ **[Đã giải quyết ở Giai đoạn 2]**: spike đầu Giai đoạn 2 tìm thấy `codemirror-lang-typst` (npm, ~15k lượt tải/tháng, cập nhật gần đây) — bản `typst_lezer()` dùng Lezer parser thuần, không cần WASM, kèm sẵn autocomplete tĩnh cho hàm/symbol built-in của Typst. Không cần tới phương án dự phòng (dựa vào semantic tokens của LSP).
2. **Preview qua reverse proxy — đã tự sửa 2 lỗi thật nhờ test bằng trình duyệt thật (Giai đoạn 2), không chỉ `curl`.** Test tĩnh bằng `curl` ban đầu (Giai đoạn 1) cho kết quả khả quan giả (trang HTML tải được 200 OK) nhưng **sai** khi kiểm tra thật trong Chrome:
   - **Lỗi 1 — WebSocket data-plane của tinymist luôn kết nối tới gốc origin (`ws://<host>/`), không theo path `/preview/...` trang được tải.** Không suy ra được từ `location.href` như đoán ban đầu. Phát hiện bằng cách patch `window.WebSocket` trong iframe để log URL thực tế. **Đã sửa**: `server/preview_proxy.go` (`handlePreviewRootWebSocket`) đặc cách nhận diện request nâng cấp WebSocket ngay tại `/` và proxy thẳng tới tinymist không qua strip-prefix, cắm vào `staticHandler` ở `server/server.go`.
   - **Lỗi 2 — `tinymist.startDefaultPreview`/`RestartPreviewWithEntry` yêu cầu `entryFile` là đường dẫn tuyệt đối**, nhưng frontend gửi path tương đối theo project (đúng chuẩn REST của mọi endpoint khác) → lỗi ngầm `entry file must be absolute path`, preview không bao giờ "pin" đúng file đang mở. **Đã sửa**: `handlePreviewRestart` (`server/preview_proxy.go`) tự resolve qua `resolveInRoot` trước khi gọi `RestartPreviewWithEntry`.
   - Sau khi sửa cả hai: xác nhận **preview hiển thị đúng nội dung Typst đã biên dịch thật** (không phải giả lập) ngay trong `<iframe>`, và cập nhật đúng khi mở file khác hoặc lưu file (Ctrl+S → frontend gọi `POST /api/preview/restart` rồi remount `<iframe>`).
   - **Giới hạn còn lại, chưa giải quyết**: preview **không** tự cập nhật theo từng phím gõ chưa lưu (khác với hành vi "live" của bản desktop) — gõ xong phải Ctrl+S mới thấy thay đổi trong preview, dù LSP `didChange`/diagnostics vẫn phản ứng theo từng phím gõ bình thường (đã xác nhận diagnostics/completion không bị ảnh hưởng). Nguyên nhân gốc (tinymist có theo dõi buffer chưa lưu qua LSP hay không trong chế độ `startDefaultPreview`) chưa được xác minh — cần điều tra thêm ở một phiên sau nếu muốn khôi phục đúng trải nghiệm "gõ là thấy" của bản desktop.
3. **Bảo mật khi expose ra VPS**: bắt buộc có auth token/password ngay từ Giai đoạn 1; khuyến nghị rõ trong docs là chạy sau TLS reverse proxy, không public thẳng cổng backend.
4. **Runtime dependency của AI Agent** (Node/npx, Python/uv, hoặc binary riêng nền tảng) thay đổi tuỳ agent người dùng chọn — tài liệu triển khai cần liệt kê rõ, v1 chỉ cam kết hỗ trợ agent dạng `npx`.
5. ~~**`ServiceFacade` vẫn kéo theo Gio một cách gián tiếp**~~ **[Đã giải quyết triệt để ở Giai đoạn 4]**. Test `GOOS=linux GOARCH=amd64 go build ./cmd/typstify-server` xác nhận điều lo ngại là có thật: build **thất bại** hẳn (không chỉ "nặng thêm") vì `gioui.org/internal/vk` không có file nào thoả build constraint khi cross-compile không cgo. Đã truy ra và cắt đứt toàn bộ 3 đường kéo Gio vào package `service`:
   - `service/window.go`: `WindowService.NewWindow`/`LoadTheme`/`Window`/`WindowView`/`WidgetView` (import `gioui.org/app` trực tiếp) — xác nhận **không có nơi nào khác trong repo gọi tới** (dead code), chuyển nguyên vẹn sang `ui/windowview.go` (nơi đúng về mặt kiến trúc). `WindowService` chỉ còn giữ `Shutdown`/`Wait`/`Context`/`Settings()`.
   - `service/service.go`: các field `vm view.ViewManager`, `fileChooserBuilder func() *explorer.FileChooser` (từ `github.com/oligo/gioview`) đổi thành hook dạng `func(any)`/`func() any` thuần Go, do `ui/ui.go` truyền vào qua `SetViewManager(requestSwitch, invalidateWindow, onUIClose, currentView)` — 3 điểm gọi `srv.FileChooser()` trong `ui/` (`welcome.go`, `dialog/create_project.go`, `navpanel/menu_panel.go`) thêm type-assertion `.(*explorer.FileChooser)`, hành vi không đổi.
   - `service/workspace.go`: `WorkspaceState.TreeState *filetree.TreeState` — struct `TreeState` (thuần dữ liệu: `Path`, `ExpandedNodes`, không có field Gio nào) được tách ra package mới `widgets/filetree/treestate`, `widgets/filetree.TreeState` giữ nguyên như **type alias** trỏ tới đó → dữ liệu đã lưu (bbolt) không bị ảnh hưởng vì vẫn là cùng một type.
   - **Xác minh sau khi sửa**: `go build .` (desktop) vẫn sạch, `go vet` sạch, chạy lại smoke test REST đầy đủ vẫn đúng, và **`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/typstify-server` build thành công ra binary Linux tĩnh** (~24MB, không cần cgo/X11/Wayland) — sẵn sàng cho Dockerfile ở phần đóng gói.

---

## Xác minh

**Đã làm** (trong phiên triển khai này, bằng `curl`, một script Node.js, và trình duyệt Chrome thật qua công cụ browser automation):
- Đăng nhập bằng password, chặn truy cập khi chưa đăng nhập, chặn path traversal trên các endpoint file.
- CRUD file/thư mục qua REST (`tree`/`file` GET-PUT-POST-DELETE/`rename`), mở/tạo project.
- Editor CodeMirror thật: cú pháp Typst tô màu đúng, gõ code sai cú pháp thấy diagnostics thật từ tinymist, completion trả kết quả thật.
- Preview: hiển thị đúng tài liệu Typst đã biên dịch thật trong `<iframe>` (sau khi sửa 2 lỗi mô tả ở rủi ro #2), cập nhật đúng khi mở file khác hoặc `Ctrl+S`.
- AI Agent: phiên ACP thật khởi tạo thành công qua `/ws/agent` (cả từ script Node.js lẫn trình duyệt) — xem giới hạn còn lại ở Giai đoạn 3.
- `go build`/`go vet` sạch cho desktop lẫn `cmd/typstify-server`; cross-compile `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` cho `cmd/typstify-server` thành công.

**Chưa làm được trong phiên này** (cần làm ở lần sau trước khi coi là sẵn sàng dùng thật):
- `docker build`/`docker compose up` thật — Docker daemon không có sẵn trong sandbox này.
- Round-trip prompt → phản hồi của AI Agent trong trình duyệt với agent mặc định (`npx @agentclientprotocol/claude-agent-acp`) — phiên này chỉ thử được với agent thử nghiệm khác đã cấu hình sẵn trên máy test, không phản hồi kịp trong thời gian test.
- So sánh song song hành vi completion/hover/diagnostics với bản desktop trên cùng một project.
- Toàn bộ Giai đoạn 5 (xuất file, quản lý package Typst qua web, i18n, các dialog còn thiếu).
