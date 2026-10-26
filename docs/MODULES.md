# Danh mục module

Tài liệu này liệt kê từng module theo thư mục. Phần **Desktop / lõi dùng chung** bên dưới; các phần server, web, chessbook do tài liệu khác bổ sung.

## Desktop và lõi dùng chung

### Gốc repo
| File | Vai trò |
|---|---|
| `app.go` | `main()`: `service.NewService`, `logger.InitLogger(RootDir/application.log)`, `ui.NewUI`, chạy `ui.Loop` trong goroutine rồi `app.Main()` |
| `go.mod` | Module `looz.ws/typstify`, Go 1.25; gioui.org, gvcode (fork `internal/gvcode`), acp-go-sdk, MCP go-sdk, bbolt, fsnotify, fuzzy |
| `utils/` | git, lru_cache, debouncer, scrollbar, `vietnamese/` |
| `widgets/`, `version/`, `scripts/`, `scratch/` | Widget dùng chung, phiên bản, script, chương trình thử nghiệm |

### `agent/` — client ACP và phiên chat
| File | Nội dung |
|---|---|
| `manager.go` | `SessionManager`: spawn agent (stdio), `Authenticate`, `NewSession`, `ListSessions`, `LoadSession`, `ResumeSession`, `CloseSession` |
| `acp_client.go` | `ACPClient` cài `acp.Client`: đọc/ghi file (chống thoát cwd), terminal, `SessionUpdate`, `RequestPermission`, `RegisterExtension` |
| `session.go` | `ACPSession` cài `ChatSession`: `Prompt` (buffer + CAS), `Cancel`, `RequestPermission` |
| `turn.go`, `broker.go` | `PromptTurn` theo dõi tool call; `Broker[ToolCallId]` pub/sub không chặn |
| `terminal.go`, `path.go` | Terminal có bộ đệm giới hạn; `resolvePath` |
| `remote_session.go` | `RemoteChatSession` — cùng interface nhưng đi qua WebSocket `/ws/agent` |
| `mcp_server.go` | `McpServer` (streamable HTTP), `AddMcpTool[In,Out]` |
| `extensions/compiler.go` | Extension ACP biên dịch qua `CompileHelper` |
| `view/` | Giao diện chat: message, markdown, toolcall, permission, terminal, inputbox, auth, config |

### `editor/` — trình soạn thảo
| File | Nội dung |
|---|---|
| `editor.go` | `TextEditor` bọc gvcode; hook LSP, `saveFunc` |
| `autosaver.go` | `AutoSaver` (`Update`, `doSave`, `SaveNow`, `Stop`) |
| `diff.go` | `GitDiff` — git gutter (`Trigger`, `ParseDiff`, `parseDiffOutput`) |
| `typst.go` | Ghép cặp `[]()"`, `mathDelimiters` cho `$…$` |
| `highlighter.go`, `ruler.go`, `search.go`, `hovertips.go`, `menu.go`, `statusbar.go`, `extension.go` | Tô sáng Chroma, thước, tìm kiếm, tooltip, menu, thanh trạng thái |

### `lsp/` — tinymist
| File | Nội dung |
|---|---|
| `server.go`, `lsp.go` | `Server.Start` spawn tinymist (stdio, chạy song song được); `NewClient(workspace, settings)` cho từng project; `GetLspClient` là client mặc định của desktop |
| `client.go` | JSON-RPC client: diagnostics, `didOpen/didChange/didSave/didClose`, hover, symbols, completion, `RegisterShowDocumentHandler`, `handleShowDocument` |
| `completion.go` | `LspAutoCompletor` (`Suggest`, `FilterAndRank`) |
| `previewer.go` | `PreviewService`: `Start`, `ScrollOnSelectionChange`, chế độ `document`/`slide` |
| `document.go`, `protocol/` | Bộ đệm tài liệu; kiểu LSP sinh tự động (`ts*.go`) |

### `apps/tauri/` — ứng dụng Tauri 2 (desktop và Android)
| Tệp | Nội dung |
|---|---|
| `src-tauri/src/lib.rs` | `run()` chung (`mobile_entry_point`), nhớ cổng (`bind_remembered_port`), mở link ngoài bằng trình duyệt hệ thống; `main.rs` chỉ gọi `run()` |
| `src-tauri/src/proxy.rs` | Proxy cục bộ của bản mobile: phục vụ giao diện đóng gói, chuyển `/api`, `/ws`, `/preview` tới server (viết lại Host/Origin/Referer, cookie bỏ Domain/Secure, WebSocket), kiểm địa chỉ server (`Upstream::parse`); có unit test |
| `src-tauri/src/mobile.rs` | Bản Android: màn hình chọn server (`app-setup.html`, lệnh `connect`/`current_server`), xoá dữ liệu duyệt web khi đổi server, chặn `/__app/server` để quay lại màn hình chọn |
| `src-tauri/tauri.android.conf.json` | Ghi đè cho Android: `frontendDist` = `resources/mobile-dist`, không sidecar |
| `src-tauri/gen/android/` | Dự án Gradle (sinh bằng `tauri android init`), thêm `network_security_config.xml` chỉ cho http tới `127.0.0.1` |
| `scripts/prepare-mobile.ps1` | Build web, gộp `web/dist` với `mobile/app-setup.html` vào `src-tauri/resources/mobile-dist` |
| `src-tauri/src/desktop.rs` | Chọn cổng (nhớ qua các lần mở), tạo token khởi động, chạy `typstify-server` kèm theo (ẩn cửa sổ console, ghi nhật ký), chờ cổng mở rồi mở cửa sổ chính tới `/api/auth/launch`, một phiên bản duy nhất, Job Object dừng cả cây tiến trình khi thoát |
| `src-tauri/tauri.conf.json` | `externalBin` (typstify-server, typst, tinymist), `resources` (`web/`, `packages/`), bộ cài NSIS |
| `scripts/prepare.ps1` | Build server, tải typst/tinymist (kiểm SHA-256), build web, chép gói chessbook + board-n-pieces vào `src-tauri/{binaries,resources}` |
| `splash/index.html` | Màn hình chờ trong lúc server khởi động; báo lỗi kèm đường dẫn nhật ký |

### `service/`
| Gói / file | Nội dung |
|---|---|
| `service.go` | `ServiceFacade`, `SetProjectDir` (desktop), `SetDefaultProjectDir` (server), `initMcpServer`, `RuntimeMcpServers`, `SetMcpResolver`, `StartACPSession`, `ListACPSessions`, `LoadOrResumeACPSession`, `CloseACPSession` |
| `runtimes.go` | `Runtimes`/`ProjectRuntime`: tinymist, preview, MCP theo từng project mở trên server; đếm tham chiếu, giới hạn `MaxOpenProjects`, dọn nhàn rỗi |
| `workspace.go` | `WorkspaceService` (bbolt, git watch, `syncManagedBibliography`) |
| `filewatcher.go` | fsnotify → bus |
| `bus/` | `bus.go`, `topics.go` |
| `settings/` | Model: General, Editor, Typst, Lsp, Tpix, AcpAgent, Dropbox, Remote; `agent_registry.go`; nâng cấp `loadLegacy*` từ bbolt cũ |
| `mcp/` | Tool editor: `typstCompiler`, `typst-previewer`, `getDocumentOutline`, `queryDiagnostics`, `discoverFonts`, `getActiveDocument`. Tool package: `listLocalPackages`, `downloadPackage`, `queryPackageDetail`, `searchPackages`, `publishPackage`, `getUserInfo`, `readPackageIndex`. Resource: `user-profile`, `active-document` |
| `dropbox/` | `Syncer.SyncProject`, `restartAutoSync`, `isIgnored` |
| `remote/` | Client REST tới server tự host (`Login`, `GetSettingsMeta`, `GetSettings`, `PutSettings`, `ListAgentSessions`, thao tác file) |
| `projectstore/` | `ProjectStore`: `LocalStore`, `RemoteStore` |
| `history/` | Lịch sử phiên bản: kho nội dung theo hash (`objects/`), manifest mỗi phiên bản, `Take`/`Compare`/`Read`/`Restore`, dọn theo lịch (giữ 24 giờ, mỗi giờ 7 ngày, mỗi ngày sau đó; phiên bản có tên giữ mãi) |
| `gitsync/` | Đồng bộ Git bằng go-git: cấu hình (token 0600 trên máy chủ), `Status`/`Commit`/`Push`/`Pull`, chặn IP nội bộ khi kết nối |
| `collab/` | Cộng tác thời gian thực: `Hub`/`Room`/`Peer` chuyển tiếp và lưu update Yjs theo tệp, phân quyền theo kênh, chọn leader, nén log, phát hiện sửa ngoài (`base.txt`), `EncodeInsertText` (khởi tạo phòng từ tệp) |
| `wasmassets/` | Font và package `.tar.gz` cho compiler trong trình duyệt: `FontIndex.Scan/Lookup`, `Packages.Archive`/`Packages.Dir` (tìm thư mục package, tải `@preview` vào cache, giải nén an toàn) |
| `universe/` | Typst Universe: `Index` (chỉ mục `packages.typst.org`, bản mới nhất mỗi gói, cache đĩa, làm mới nền), `Thumbnails` (ảnh xem trước, cache đĩa), `CopyTemplate` (như `typst init`), `ReadReadme` |
| `fonts/` | Quản lý thư mục font thêm (`SanitizeFilename`, giới hạn 30 MiB, đuôi `.ttf/.otf/.ttc`) |
| `net/` | HTTP invoker tới typstify.com (đăng ký thiết bị, kiểm tra cập nhật) |
| `tpix.go` | Cấp API key cho SDK tpix |

### `typst/`
| File | Nội dung |
|---|---|
| `compiler.go` | `Compiler`: `Compile`, `Watch` |
| `cmd.go` | `SetupCmdBuilder`, `InitCmd`, `FontsCmd`, `VersionCmd`, `QueryEvalCmd`, `QueryHeadingPages` |
| `options.go`, `inputs.go`, `pdf.go`, `version.go` | Tuỳ chọn compile, `--input`, bảng chuẩn/phiên bản PDF |
| `export/helper.go` | `CompileHelper.BuildParams` — dùng chung cho export, MCP, extension ACP, server |
| `pkg/` | `TypstPkgService`: `SearchPkgs`, `GetPkgDetail`, `Download`, `DownloadWithSpec`, `PullDependencies`, đóng gói/publish, `PkgIndexForLLM` |

### `ui/`
| Thư mục | Nội dung |
|---|---|
| `ui.go`, `home.go`, `welcome.go`, `windowview.go`, `crash_report.go` | Khởi tạo, bố cục chính, màn chọn project, báo lỗi |
| `editors/` | `typst_view.go` (`TypstEditor`: editor + preview + outline; `togglePreview`, `setupLsp`), `editor_header.go`, `generic_text_view.go` |
| `preview/` | `previewer.go`, `webview.go`, `webview_linux.go` — webview native |
| `navpanel/` | filetree, tabbar, outline, history, navdrawer, menu_panel |
| `pkgmgmt/` | `card.go` (chọn phiên bản), `list.go`, `manage.go` |
| `assistant/` | `chat.go` (local/remote), `sessions.go` (lịch sử) |
| `settings/` | `view_setting.go`, `agent.go`, `lsp.go`, `fonts.go`, `sync.go`, `tpix.go`, `update_check.go`, `subviews.go`, `form/binder.go` |
| `dialog/` | export, bibliography, create_project, delete_file, publish_pkg, indentation, open_external, drag-drop |
| `palette/`, `statusbar/`, `viewer/`, `remoteproject/` | Command palette, thanh trạng thái, xem ảnh (cache LRU), duyệt project trên server |

### `i18n/` và `fonts/`
- `i18n/`: `Localizer` dùng `golang.org/x/text/message`; locale `en-US`, `zh-CN`, `de-DE` (`translations/locales/`). **Chưa có tiếng Việt.** `Get(id)` quay về `en-US` nếu thiếu.
- `fonts/`: nhúng Hack, RobotoMono, NotoSansMath, NotoEmoji — font của **giao diện**. Font cho tài liệu Typst do `service/fonts` quản lý.

## Web và server

### `cmd/typstify-server/`
Entry server headless (`main.go`). Đọc cờ/biến môi trường (`-addr`, `-password`, `-project`, `-static-dir`, `-project-root`), dựng `service.ServiceFacade` rồi `server.New`. Tắt êm khi SIGTERM. Chi tiết cờ: `docs/TECH.md`.

### `server/` — tầng HTTP/WebSocket
Chỉ làm transport, ủy quyền cho `ServiceFacade`. Bảng route đầy đủ: `docs/API.md`.

| File | Vai trò |
|---|---|
| `server.go` | `Server`, `routes()`, `compileSlots`, static handler + SPA fallback |
| `auth.go`, `api_tokens.go`, `password.go` | Người dùng (mật khẩu argon2id, tự nâng cấp hash SHA-256 cũ), phiên cookie, Bearer token (băm SHA-256) |
| `session_project.go` | Project của phiên (`projectDir`, `projectRoot`, `openProjectFor`), header `X-Typstify-Project` / `?project=` → 409 `project_changed`, theo dõi project đang hoạt động |
| `security.go` | `loginLimiter`, `withBodyLimit`, `sameOrigin`, `trustedPeer` |
| `paths.go`, `files_api.go` | `resolveInRoot` (chặn symlink), `isProtectedPath`, CRUD file/workspace |
| `export_api.go` | `/api/export`, `acquireCompile` (2 slot, 90 giây) |
| `preview_render.go` | SVG live từ nội dung chưa lưu (shadow file `.live_preview_*.typ`) |
| `preview_anchors.go` | Số trang thật của heading (`typst.QueryHeadingPages`) |
| `preview_proxy.go` | Reverse proxy tới preview của tinymist |
| `pkg_api.go`, `fonts_api.go`, `settings_api.go` | Package Tpix, font, settings + `meta` |
| `history_api.go`, `git_api.go` | `/api/history*` (tự chụp mỗi 5 phút), `/api/git*` |
| `collab_ws.go` | WebSocket `/ws/collab` cho `service/collab`; kiểm tra tệp trên đĩa mỗi 2 giây |
| `wasm_api.go` | Font và package cho compiler trong trình duyệt (`/api/wasm/*`) |
| `dropbox_api.go` | Auth/sync/import Dropbox |
| `agent_api.go`, `agent_auth_callback.go`, `agent_ws.go`, `console_api.go` | Registry, đăng nhập, WebSocket agent, console |
| `lsp_ws.go` | Cầu nối LSP qua WebSocket |
| `typst_repair.go` | Sửa import chessbook trong nguồn Typst |
| `i18n_api.go`, `helpers.go` | Dịch chuỗi, tiện ích JSON/lỗi |

### `web/src/` — frontend React 19
Không router, không thư viện state. `App.tsx` là máy trạng thái loading → login → pickProject → workspace (gọi `/api/auth/status`, `/api/workspace/current`).

- `api/`: `client.ts` (`api.get/post/put/putJson/postBinary/del`, `wsUrl`, `ApiError`, `credentials: 'include'`), `types.ts`, `dropboxTypes.ts`, `pkgTypes.ts`, `agentRegistry.ts`.
- `components/`:
  - Khung: `Workspace.tsx` (shell: cây file, editor, preview, panel; màn hình ≤ 720px (`lib/useMediaQuery.ts`) hiện một khung mỗi lúc, chuyển bằng thanh tab ở đáy), `FileTree`, `StatusBar`, `Resizer`, `Modal`, `PromptDialog`, `QuickActions`, `ProjectPicker`, `NewDocModal`, `TemplateGallery` (dự án mới từ mẫu cờ vua hoặc Universe).
  - Lịch sử: `HistoryPanel.tsx` (thẻ Phiên bản / Git), `HistoryDiff.tsx` (`@codemirror/merge`), `GitPanel.tsx`.
  - Soạn thảo/preview: `Editor.tsx` (CodeMirror), `OutlinePanel`, `PreviewPane.tsx` (4 chế độ: wasm "Tức thì" mặc định / svg / pdf / tinymist), `WasmPreview.tsx` (renderer typst.ts, vá SVG tăng dần, click-to-source, cuộn đồng bộ, trình chiếu), `ExportButton`, `SymbolPicker.tsx` (bảng `sym.*`), `TypstKeyBar.tsx` (hàng phím ký tự Typst trên bàn phím ảo, chỉ ở bố cục điện thoại).
  - Agent: `AgentChat`, `AgentSessionHistory`, `AgentRegistryPicker`, `AuthCard`.
  - Cờ vua / nhập liệu: `ChessToolbar`, `ChessBoardModal`, `PgnImportModal`, `DataImportModal`, `MarkdownImportModal`.
  - Dropbox: `DropboxSyncModal`, `DropboxImportModal`. Cài đặt: `SettingsPanel`, `PackageManager` (tìm qua chỉ mục Universe, chèn `#import`, `PackageDocs` hiển thị README). Đăng nhập: `LoginPage`, `BrandMark`.
  - PWA: `PwaBanners` (ngoại tuyến, có bản mới). Service worker ở `src/sw/sw.ts` (tsconfig riêng `tsconfig.sw.json`), plugin trong `vite.config.ts` build ra `dist/sw.js` kèm danh sách cache sẵn; `lib/swStrategy.ts` chọn cách trả lời theo URL, `lib/pwa.ts` đăng ký và theo dõi bản mới.
  - Tải lười (lazy chunk): Editor, AgentChat, PackageManager, SettingsPanel, ChessBoardModal, các modal import, DropboxSyncModal.
- `lib/`:
  - Mạng: `agentClient.ts` (`/ws/agent`, tự tạo message `disconnected`), `lspClient.ts` (`/ws/lsp`, nối lại 1–15 giây), `acpTypes.ts`.
  - Preview: `scrollSync.ts`, `svgHelper.ts` (`scopeSvgIds`, `extractSvgDimensions`).
  - `collab/`: cộng tác. `provider.ts` (`CollabProvider`: WebSocket, hai Y.Doc, awareness, IndexedDB theo epoch, nhiệm vụ leader, gộp ba chiều sửa ngoài), `protocol.ts`, `textDiff.ts` (`minimalChange`, `rebaseChange`), `comments.ts`, `commentHighlight.ts`, `useCollab.ts` (`useCollab`, `useThreads`).
  - `wasm/`: compiler Typst trong Web Worker. `compiler.worker.ts` (typst.ts, IncrementalServer, truy vấn trang heading), `client.ts` (`WasmCompilerClient` singleton, font mặc định), `accessModel.ts` (file project đọc lười, cache theo hash), `packageRegistry.ts`, `idb.ts` (IndexedDB), `diagnostics.ts`, `sourceMap.ts` (`findSourceLine`).
  - Typst/cờ vua: `typst.ts` (sửa import chessbook, directive cỡ chữ/cột), `chess.ts`, `pgn.ts`.
  - Chuyển đổi: `markdownToTypst.ts`, `dataImport.ts`.
  - Giao diện: `i18n.ts`, `vi.ts`, `theme.ts`, `editorTheme.ts`, `shortcuts.ts`, `useDismiss.ts`.
  - Mẫu và gói: `chessTemplates.ts` (8 mẫu cờ vua), `universe.ts` (kiểu dữ liệu Universe, `importInsertion`).
  - Editor: `editorPrefs.ts` (Vim, chính tả; localStorage `typstify_editor_prefs`), `lspEdits.ts` (TextEdit LSP → thay đổi CodeMirror), `upload.ts` (dán/kéo thả ảnh, `POST /api/workspace/upload`), `symbols.ts` (lọc ký hiệu), `spellcheck.ts` (đánh dấu `spellcheck` chỉ cho văn bản).
- Kiểm thử: vitest (`*.test.ts` cạnh nguồn cho `scrollSync`, `svgHelper`, `chess`, `pgn`, `typst`, `dataImport`, `markdownToTypst`, `wasm/`).
