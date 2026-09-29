# Thuật toán chính

Mỗi mục kèm file nguồn để đối chiếu. Phần này gồm các thuật toán phía **desktop và lõi dùng chung**: mục 1, 2, 5–11. Các mục 3, 4, 12, 13 (preview SVG web, scroll sync web, bảo mật server, chessbook) do tài liệu khác bổ sung.

## 1. Biên dịch Typst
Nguồn: `typst/compiler.go`, `typst/cmd.go`, `typst/export/helper.go`.

- `CompileHelper.BuildParams(target, outName)` dựng root, `--font-path` (thư mục project cộng `ExtraFontPath`), package-path và features. Đây là điểm vào duy nhất, dùng chung cho export UI, MCP `typstCompiler`, extension ACP và server.
- `Compiler.Compile` chạy `typst compile`, sau đó liệt kê file trong `OutDir` có tiền tố `OutFilename` và gọi callback cho từng file. Đầu vào `-` nghĩa là đọc stdin từ `InputReader`.
- `Compiler.Watch` chạy `typst watch`, dùng fsnotify theo dõi thư mục tạm để báo file đầu ra đổi; chặn goroutine tới khi lệnh kết thúc.
- `QueryEvalCmd` chạy `typst eval <expr> --in file --format json` với đủ root/font-path/package-path/`--input`. `QueryHeadingPages` dùng nó để chạy `query(heading).map(h => h.location().page())` và trả số trang vật lý của mỗi heading. Hiện chỉ server web gọi hàm này.

## 2. Preview trên desktop
Nguồn: `lsp/previewer.go`, `ui/preview/`.

`PreviewService.Start`:
1. Chờ LSP client sẵn sàng (tối đa 25 lần × 0,5 giây).
2. Kill preview cũ bằng `tinymist.doKillPreview ["default_preview"]`. **Phải truyền id**; gọi không tham số sẽ lỗi.
3. Gửi `workspace/didChangeConfiguration` với `preview.browsing.args` (data-plane-host, preview-mode, invert-colors, partial-rendering, no-open, entry file).
4. `tinymist.pinMain`, rồi `tinymist.startDefaultPreview`; nếu lỗi thì thử lại **không** có entry.
5. Lấy `staticServerPort` từ kết quả, lưu `http://127.0.0.1:port` để webview `Navigate`.

Đổi chế độ `document` ↔ `slide`: `Previewer.ToggleMode` lưu vào `WorkspaceSettings` rồi `RestartPreview`. Webview bị huỷ trễ một frame (`destroyPending`) vì phải thực hiện trong frame Gio; `Layout` chừa lề trái 6dp để không che tay kéo chia đôi.

## 5. Click-to-source (desktop)
Nguồn: `lsp/client.go`, `ui/editors/typst_view.go`.

1. `setupLsp` gọi `Client.RegisterShowDocumentHandler(path, fn)`; handler lưu theo `filepath.Clean(path)`.
2. Người dùng bấm trong preview → tinymist gửi request `window/showDocument`.
3. `handleShowDocument` phân tích URI, tìm handler theo đường dẫn, gọi `handler(line, col)` (chỉ số từ 0).
4. Handler gọi `srcEditor.NavigateToLine` rồi `RefreshWindow()` — bắt buộc vì callback chạy trên goroutine LSP, không phải goroutine UI.
5. Luôn trả `ShowDocumentResult` vì đây là *request*, không phải notification.

Cuộn theo con trỏ (chiều ngược lại): `OnSelectChange` → `PreviewService.ScrollOnSelectionChange` → `tinymist.scrollPreview`; tinymist tự suy vị trí con trỏ.

## 6. Autosave an toàn và git gutter
Nguồn: `editor/autosaver.go`, `editor/editor.go`, `editor/diff.go`.

**Autosave**
- `AutoSaver.Update()` tăng bộ đếm mỗi lần sửa; timer trần 10 giây gọi `doSave`, chỉ lưu khi bộ đếm > 0. `SaveNow(cb, async)` lưu ngay; `Stop` lưu lần cuối.
- `saveFunc` đọc file trên đĩa, tính digest, so với `originalHash`. **Khác** → báo "edited elsewhere" và *không ghi đè*. **Khớp** → Truncate, ghi `PrepareForSave(text)`, cập nhật hash, cuối cùng `lspClient.OnEditorSaved` (gửi `didSave`).

**Git gutter (`GitDiff`)**
- Baseline chọn theo `utils.GitFileStatus` lần đầu: bản HEAD, hoặc bản đã stage. File chưa được git theo dõi → baseline là nội dung hiện tại.
- `Trigger` (debounce) → `ParseDiff` so buffer với baseline bằng `diff.Diff` thuần Go → `parseDiffOutput` sinh `providers.DiffHunk` (add/modify/delete), đánh dấu `Staged` nếu baseline là bản stage.
- Kết quả để trong `atomic pendingHunks`; UI lấy bằng `PendingHunks()`. `nil` = không đổi; slice rỗng = xoá hết dấu.
- `BindWorkspaceWatcher` gắn với event `workspace.file.changed`.

## 7. Gợi ý code (LSP completion)
Nguồn: `lsp/completion.go`.

- Trigger: các ký tự server khai báo, cộng Ctrl+P.
- `Suggest` gọi `OnEditorUpdated` (đẩy nội dung mới lên server) rồi `Complete` tại (line, col).
- Chỉ nhận `TextEdit`; `InsertReplaceEdit` bị bỏ qua.
- Sắp xếp ổn định theo `SortText`, rồi `Label`.
- `FilterAndRank` lọc mờ bằng `sahilm/fuzzy` trên `FilterText` (hoặc `Label` nếu trống).

## 8. Agent ACP
Nguồn: `agent/session.go`, `agent/acp_client.go`, `agent/turn.go`, `agent/broker.go`.

**Prompt có buffer** (`ACPSession.Prompt`)
- Kiểm content block theo capabilities của agent.
- Nếu đang có turn (`hasOngoingTurn`, CAS): đẩy vào `contentBuf`, trả `ErrPromptBuffered`.
- Ngược lại tạo `PromptTurn`, gọi `Conn.Prompt`. Xong → prompt trong buffer được gửi thành turn mới với `context.WithoutCancel`.
- `Cancel`: gửi `session/cancel`, `turn.Cancel()`, xoá buffer.

**Luồng xin quyền**
1. Agent gọi `ACPClient.RequestPermission`.
2. Tạo channel phản hồi buffer 1, gọi `session.RequestPermission` → đẩy `PermissionGrantRequest` qua `grantChan`; `agent/view/permission.go` hiện thẻ.
3. `select` giữa: người dùng chọn option / `turn.CancelChan(toolCallId)` đóng / `session.Done()` / `ctx.Done()`. Ba trường hợp sau trả outcome `Cancelled`.

**Broker và huỷ turn**
- Mỗi tool call Pending/InProgress đăng ký một channel qua `Broker[ToolCallId]`. Publish không chặn; subscriber chậm bị bỏ qua.
- `PromptTurn.Cancel` (CAS `canceled`) publish id mọi tool call đang chạy để các `RequestPermission` đang chờ thoát.

**Đường dẫn file**: `resolvePath` (`agent/path.go`) chặn ghi/đọc ngoài cwd và thư mục phụ được cho phép.

**Phiên remote** (`agent/remote_session.go`): cùng interface `ChatSession`, nói giao thức `/ws/agent`. Không nhận title, usage, available commands.

## 9. Đồng bộ (phần desktop, remote, Dropbox)

**Settings — last-write-wins theo section** (`ui/settings/sync.go`, `service/remote/client.go`)
- Kết nối: `Login` (`POST /api/auth/token`) lưu Bearer token vào `RemoteSettings`.
- `syncNow`: `GET /api/settings/meta`, rồi với từng section general/editor/typst/lsp so timestamp: remote mới hơn → GET + `ApplyRemote`; local mới hơn → PUT; bằng nhau → bỏ qua.
- Section `agent` và `RemoteSettings` **không sync** (thuộc máy cục bộ).
- `ApplyRemote(raw, remoteUpdatedAt)` chỉ áp dụng khi bản remote mới hơn.

**Remote agent**: `GET /api/agent/sessions` liệt kê, desktop nối lại phiên có `UpdatedAt` mới nhất qua `RemoteChatSession`.

**Remote project**: `ProjectStore` có `LocalStore` (resolve chống thoát root) và `RemoteStore` (bọc `remote.Client`: tree, đọc/ghi/tạo/xoá/đổi tên file). `ui/remoteproject/view.go` dùng interface này.

**Dropbox** (`service/dropbox/syncer.go`)
`SyncProject(ctx, dir, mode)` với mode `two-way`/`push`/`pull`:
1. Quét file local (bỏ qua theo `isIgnored`).
2. Lấy danh sách remote tại `<SyncFolder>/<projectName>`.
3. Với file có ở local: so mtime với ngưỡng 2 giây; two-way → bên mới hơn thắng; push/pull ép một chiều.
4. Với file chỉ có ở remote: tải về (trừ chế độ push).

Single-flight (mutex + cờ `syncing`); `restartAutoSync` chạy định kỳ. Kết quả `{uploaded, downloaded, conflicts, errors, duration_ms}`.

## 10. Export
Nguồn: `ui/dialog/export.go`, `typst/export/helper.go`, `typst/pdf.go`.

- Hộp thoại chọn định dạng PDF/PNG/SVG, phiên bản và chuẩn PDF, PPI, trang.
- `typst/pdf.go` có bảng chuẩn/phiên bản với `Compatible` và `MinVersion`, để UI chỉ cho tổ hợp hợp lệ.
- Gọi `CompileHelper` (một lần `typst compile`), báo kết quả qua statusbar (`statusbar.notification`).

## 11. Trình quản lý package
Nguồn: `typst/pkg/`, `ui/pkgmgmt/`.

- `SearchPkgs(ns, kind, category, query)` cache theo khoá `ns:kind:query` và cờ `loading` để không gọi trùng.
- **Chọn phiên bản** (`ui/pkgmgmt/card.go`): chi tiết (danh sách phiên bản) tải **một lần**, bất đồng bộ, chỉ khi người dùng bấm nút chọn. Sau đó tạo `widgets.Dropdown` một lần, mặc định `LatestVersion`; `onDownloadClicked(pkg, version)` tải đúng phiên bản đã chọn.
- `Download(ns, name, version)`, `DownloadWithSpec(spec)`, `PullDependencies(projectDir)` (nút "Pull all dependencies").
- Đóng gói: `CreatePkg`/`Bundle` (tar.gz) rồi `Push` lên tpix.
- `PkgIndexForLLM` tạo chỉ mục văn bản cho agent; `scanPackages` quét thư mục cache.

---

# Thuật toán phía web và server

> Các mục 3, 4, 9 (phần web) và 12 dưới đây do nhánh web/server viết, đã đối chiếu với mã nguồn. Mục 9 dưới đây bổ sung cho mục 9 (desktop) ở trên.

## 3. Preview SVG trên web

Nguồn: `server/preview_render.go`, `web/src/components/PreviewPane.tsx`, `web/src/lib/svgHelper.ts`.

Client (`PreviewPane`): nội dung gửi đi là `liveContent ?? content` (bản đang gõ, chưa lưu). Effect chạy lại khi đổi `[path, version, effectiveContent, nonce, viewMode]`; mỗi lần tạo `AbortController` mới để huỷ yêu cầu cũ.

Server `POST /api/preview/render {path, content, format}`:
1. Phân giải `path` trong project root (`resolveInRoot`).
2. Nếu có `content`: ghi vào **file bóng** `.live_preview_*.typ` **cùng thư mục** với file thật, để `#import`/đường dẫn tương đối vẫn đúng; xoá file sau khi xong. Không có `content` thì biên dịch thẳng file trên đĩa (báo `file not found` nếu thiếu).
3. Tạo thư mục ra tạm `typstify-preview-live-*`, giành một slot biên dịch (`acquireCompile`).
4. `export.NewCompileHelper(root, settings.Typst())`, định dạng SVG (hoặc PDF khi `format=pdf`), PPI 144; `BuildParams` rồi `Compile`.
5. Đọc các file `.svg`, **sắp xếp theo số trang** (`extractPageNum`, để trang 10 đứng sau trang 9), trả `{ok, pages[], pageCount}`.
6. Mọi lỗi (kể cả lỗi biên dịch) trả HTTP 200 với `ok:false` và `error`. Không sinh được trang nào cũng là lỗi.

Client hiển thị lười: `IntersectionObserver` với `rootMargin` 1000px chỉ dựng các trang gần khung nhìn. `scopeSvgIds` thêm tiền tố id theo trang để các SVG không đè `id` của nhau; `extractSvgDimensions` đọc kích thước.

Ba chế độ xem: `svg` (luồng trên), PDF (`GET /api/preview/pdf`, hiển thị bằng blob URL), `tinymist` (iframe `/preview/`; Workspace gọi `POST /api/preview/restart {entryFile}`, `PreviewPane` hỏi `/api/preview/status` mỗi 500 ms).

## 4. Đồng bộ cuộn editor ↔ preview (web)

Nguồn: `web/src/lib/scrollSync.ts`, `PreviewPane.tsx`, `server/preview_anchors.go`.

**Bản đồ trang → dòng** — `buildPageLineMap(content, pageCount, anchors?)`, phần tử `i` là dòng 0-based bắt đầu trang `i+1`:
- Có anchors: dùng `buildPageLineMapFromAnchors`, ưu tiên tuyệt đối.
- Không có: dò `#pagebreak(...)` (trang sau bắt đầu ở dòng kế tiếp), thiếu thì chia đều.

**Anchors thật:** `POST /api/preview/anchors` chạy `typst.QueryHeadingPages` (`query(heading)`, lấy trang vật lý của mỗi heading theo thứ tự tài liệu). Đây là endpoint riêng và được debounce vì eval + query tốn gần bằng một lần biên dịch. Client chỉ tin anchors khi `findHeadingLines` (regex `^=+\s+\S`) tìm ra **cùng số heading** với server. Nếu tài liệu sinh heading bằng code (ví dụ `render-puzzle-collection` gọi `#heading(...)`) thì số lượng lệch và client bỏ anchors. "Hình dạng" heading được cache nên cấu trúc không đổi thì không gọi lại.

**`buildPageLineMapFromAnchors`:** lọc anchor hợp lệ, sắp theo dòng, thêm hai anchor ảo `(dòng 0, trang 1)` và `(tổng dòng, trang pageCount+1)`. Với mỗi cặp liên tiếp cần đặt `pagesToPlace = toPage - fromPage` ranh giới trang; đoạn được chia thành `pagesToPlace + 1` phần đều. Không chia `pagesToPlace` phần vì hai đầu chỉ là dòng nằm trong trang đó chứ không phải ranh giới trang; chia sai sẽ đẩy ranh giới cuối về sát `to.line`, quá muộn. `lastPage` giúp chịu nhiễu khi số trang không tăng.

**Editor → preview:** dòng con trỏ → tỉ lệ trang (`getLineRatio`). Cuộn được debounce 600 ms sau khi con trỏ đứng yên; chỉ cuộn nếu đích lệch hơn nửa khung nhìn; dùng `scrollTo` mượt và đánh dấu "cuộn do chương trình" bằng timer để sự kiện scroll phát sinh không bật ngược lại.

**Preview → editor:** handler cuộn tay được debounce; `scrollTopToPageRatio(scrollTop, pageLayouts, containerHeight)` rồi `pageRatioToLine(ratio, pageLineMap, totalLines)` → `onScrollToLine` → `editorRef.scrollToLine`. Bỏ qua khi đang có cuộn do chương trình hoặc do editor vừa gây ra.

**Chế độ tinymist:** `POST /api/preview/cursor` (debounce) → `PreviewService.ScrollOnSelectionChange`; chỉ cuộn khi gõ phím (giới hạn của tinymist, giao diện có ghi chú).

**Click-to-source:** preview SVG trên web **chưa có** (không có handler click/dblclick trong `web/src`). Chỉ iframe tinymist có cơ chế riêng. Desktop có, xem mục 5.

## 9 (bổ sung). Đồng bộ phía web/server

Nguồn: `server/settings_api.go`, `server/agent_ws.go`, `server/dropbox_api.go`, `service/dropbox/syncer.go`, `service/remote/client.go`.

**Settings desktop ↔ server (last-write-wins theo section):**
- Desktop đăng nhập `POST /api/auth/token` và lưu Bearer token trong `RemoteSettings` (`{ServerURL, Token, Enabled, RemoteAgentEnabled}`, **không** được sync).
- `syncNow` gọi `GET /api/settings/meta` lấy timestamp từng section, xét `general`, `editor`, `typst`, `lsp`: bên nào mới hơn thắng. Remote mới hơn thì GET rồi `ApplyRemote`; local mới hơn thì PUT; bằng nhau thì bỏ qua.
- `agent` bị loại khỏi vòng sync (Cmd/Args/Env/AgentID là của từng máy) dù route GET/PUT vẫn có cho web.
- Timestamp lưu trong khoá `__meta__` của file settings JSON. Chỉ General, Editor, Typst, Lsp cài `RemoteApplier`.

**Agent từ xa (`/ws/agent`):** `startSessionOrRequireAuth` lặp: `LoadOrResumeACPSession`, hoặc `StartACPSession` khi chưa có `sessionId`. Load phát lại lịch sử dưới dạng `session/update`; resume thì không. Gặp `AuthRequiredErr` thì gửi `authRequired` rồi chờ `retryAuth`; gặp `ErrSessionManagerClosed` thì đóng với lý do "project changed". Khi phiên đã chạy: đặt `wsSubscriber`, `applyPreferredConfig`, gửi `ready` và `configOptions`. `runPrompt` chạy trong goroutine; ngắt kết nối thì `session.Cancel` (nếu đang có lượt) rồi `CloseACPSession`. Desktop dùng `RemoteChatSession` nói cùng giao thức và tiếp tục phiên mới nhất theo `UpdatedAt` từ `GET /api/agent/sessions`; title, usage, lệnh khả dụng không đi qua dây.

**Project từ xa:** desktop dùng `ProjectStore` (`RemoteStore` bọc `remote.Client`) gọi các route workspace bằng Bearer token.

**Dropbox:** xem mô tả `SyncProject` ở mục 9 (desktop); phía server chỉ bọc bằng các route `/api/dropbox/*` (`server/dropbox_api.go`), chạy đơn luồng nhờ mutex + cờ `syncing`.

## 12. Bảo mật server

Nguồn: `server/security.go`, `server/auth.go`, `server/api_tokens.go`, `server/paths.go`.

- **Chặn thoát root (`resolveInRoot`):** bỏ `/` đầu, `filepath.Clean("/"+rel)` để gộp `..` trên "root giả", nối vào root thật, kiểm tra tiền tố từ vựng (`within`); rồi so lại bằng `realPath` (giải symlink; với đường dẫn chưa tồn tại thì giải tổ tiên gần nhất còn tồn tại rồi nối phần còn lại) để symlink do agent tạo không thoát được ra ngoài. `isUnderRoot` kiểm tương tự cho đường dẫn tuyệt đối khi mở/tạo project theo `TYPSTIFY_PROJECT_ROOT`. `isProtectedPath` bảo vệ `.git`, `.typstify`.
- **Giới hạn đăng nhập (`loginLimiter`):** theo IP; `maxLoginFailures = 5` trong `failureWindow = 15 phút` thì khoá `lockoutDuration = 15 phút` (HTTP 429); thành công thì xoá bộ đếm; `sweep` dọn định kỳ. Đăng nhập sai còn bị trễ `failedLoginDelay`.
- **Phiên:** cookie `typstify_session`, hết hạn trượt `sessionTTL` 30 ngày, trần tuyệt đối `sessionMaxAge` 90 ngày. Token API lưu dạng băm SHA-256, liệt kê và thu hồi được.
- **Body:** `withBodyLimit` 2 MiB mặc định, 64 MiB cho `PUT /api/workspace/file`; vượt giới hạn trả lỗi (`helpers.go`).
- **WebSocket:** `sameOrigin` kiểm Origin; `trustedPeer` chỉ tin `X-Forwarded-*` từ loopback/mạng riêng.
- **Tài nguyên:** `compileSlots` = 2 biên dịch đồng thời, mỗi lần tối đa 90 giây (`maxConcurrentCompiles`, `compileTimeout`); chờ slot có thể bị client huỷ.
- **Không lộ bí mật:** `/api/settings/tpix` không đăng ký route; server bỏ `TYPSTIFY_SERVER_PASSWORD` khỏi môi trường tiến trình con.
