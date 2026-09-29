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
| `server.go`, `lsp.go` | `Server.Start` spawn tinymist; `GetLspClient(workspace, settings)` singleton theo project |
| `client.go` | JSON-RPC client: diagnostics, `didOpen/didChange/didSave/didClose`, hover, symbols, completion, `RegisterShowDocumentHandler`, `handleShowDocument` |
| `completion.go` | `LspAutoCompletor` (`Suggest`, `FilterAndRank`) |
| `previewer.go` | `PreviewService`: `Start`, `ScrollOnSelectionChange`, chế độ `document`/`slide` |
| `document.go`, `protocol/` | Bộ đệm tài liệu; kiểu LSP sinh tự động (`ts*.go`) |

### `service/`
| Gói / file | Nội dung |
|---|---|
| `service.go` | `ServiceFacade`, `SetProjectDir`, `initMcpServer`, `StartACPSession`, `ListACPSessions`, `LoadOrResumeACPSession`, `CloseACPSession` |
| `workspace.go` | `WorkspaceService` (bbolt, git watch, `syncManagedBibliography`) |
| `filewatcher.go` | fsnotify → bus |
| `bus/` | `bus.go`, `topics.go` |
| `settings/` | Model: General, Editor, Typst, Lsp, Tpix, AcpAgent, Dropbox, Remote; `agent_registry.go`; nâng cấp `loadLegacy*` từ bbolt cũ |
| `mcp/` | Tool editor: `typstCompiler`, `typst-previewer`, `getDocumentOutline`, `queryDiagnostics`, `discoverFonts`, `getActiveDocument`. Tool package: `listLocalPackages`, `downloadPackage`, `queryPackageDetail`, `searchPackages`, `publishPackage`, `getUserInfo`, `readPackageIndex`. Resource: `user-profile`, `active-document` |
| `dropbox/` | `Syncer.SyncProject`, `restartAutoSync`, `isIgnored` |
| `remote/` | Client REST tới server tự host (`Login`, `GetSettingsMeta`, `GetSettings`, `PutSettings`, `ListAgentSessions`, thao tác file) |
| `projectstore/` | `ProjectStore`: `LocalStore`, `RemoteStore` |
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
| `auth.go`, `api_tokens.go` | Người dùng, phiên cookie, Bearer token (băm SHA-256) |
| `security.go` | `loginLimiter`, `withBodyLimit`, `sameOrigin`, `trustedPeer` |
| `paths.go`, `files_api.go` | `resolveInRoot` (chặn symlink), `isProtectedPath`, CRUD file/workspace |
| `export_api.go` | `/api/export`, `acquireCompile` (2 slot, 90 giây) |
| `preview_render.go` | SVG live từ nội dung chưa lưu (shadow file `.live_preview_*.typ`) |
| `preview_anchors.go` | Số trang thật của heading (`typst.QueryHeadingPages`) |
| `preview_proxy.go` | Reverse proxy tới preview của tinymist |
| `pkg_api.go`, `fonts_api.go`, `settings_api.go` | Package Tpix, font, settings + `meta` |
| `dropbox_api.go` | Auth/sync/import Dropbox |
| `agent_api.go`, `agent_auth_callback.go`, `agent_ws.go`, `console_api.go` | Registry, đăng nhập, WebSocket agent, console |
| `lsp_ws.go` | Cầu nối LSP qua WebSocket |
| `typst_repair.go` | Sửa import chessbook trong nguồn Typst |
| `i18n_api.go`, `helpers.go` | Dịch chuỗi, tiện ích JSON/lỗi |

### `web/src/` — frontend React 19
Không router, không thư viện state. `App.tsx` là máy trạng thái loading → login → pickProject → workspace (gọi `/api/auth/status`, `/api/workspace/current`).

- `api/`: `client.ts` (`api.get/post/put/putJson/postBinary/del`, `wsUrl`, `ApiError`, `credentials: 'include'`), `types.ts`, `dropboxTypes.ts`, `pkgTypes.ts`, `agentRegistry.ts`.
- `components/`:
  - Khung: `Workspace.tsx` (shell: cây file, editor, preview, panel), `FileTree`, `StatusBar`, `Resizer`, `Modal`, `PromptDialog`, `QuickActions`, `ProjectPicker`, `NewDocModal`.
  - Soạn thảo/preview: `Editor.tsx` (CodeMirror), `OutlinePanel`, `PreviewPane.tsx` (3 chế độ svg / pdf / tinymist), `ExportButton`.
  - Agent: `AgentChat`, `AgentSessionHistory`, `AgentRegistryPicker`, `AuthCard`.
  - Cờ vua / nhập liệu: `ChessToolbar`, `ChessBoardModal`, `PgnImportModal`, `DataImportModal`, `MarkdownImportModal`.
  - Dropbox: `DropboxSyncModal`, `DropboxImportModal`. Cài đặt: `SettingsPanel`, `PackageManager`. Đăng nhập: `LoginPage`, `BrandMark`.
  - Tải lười (lazy chunk): Editor, AgentChat, PackageManager, SettingsPanel, ChessBoardModal, các modal import, DropboxSyncModal.
- `lib/`:
  - Mạng: `agentClient.ts` (`/ws/agent`, tự tạo message `disconnected`), `lspClient.ts` (`/ws/lsp`, nối lại 1–15 giây), `acpTypes.ts`.
  - Preview: `scrollSync.ts`, `svgHelper.ts` (`scopeSvgIds`, `extractSvgDimensions`).
  - Typst/cờ vua: `typst.ts` (sửa import chessbook, directive cỡ chữ/cột), `chess.ts`, `pgn.ts`.
  - Chuyển đổi: `markdownToTypst.ts`, `dataImport.ts`.
  - Giao diện: `i18n.ts`, `vi.ts`, `theme.ts`, `editorTheme.ts`, `shortcuts.ts`, `useDismiss.ts`.
- Kiểm thử: vitest (`*.test.ts` cạnh nguồn cho `scrollSync`, `svgHelper`, `chess`, `pgn`, `typst`, `dataImport`, `markdownToTypst`).
