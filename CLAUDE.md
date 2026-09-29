# Typstify — Hướng dẫn cho AI Agent

> `CLAUDE.md` và `AGENTS.md` có nội dung giống hệt nhau. Sửa một file thì sửa cả hai.
> Tài liệu chi tiết nằm trong [`docs/`](docs/README.md) (mục lục).

## 1. Dự án là gì

Typstify là trình soạn thảo **Typst** gồm ba bề mặt dùng chung một backend Go:

- **Desktop**: Go 1.25 + Gio (`gioui.org`), soạn thảo bằng fork `internal/gvcode`, preview qua `tinymist`.
- **Server web tự host**: `cmd/typstify-server` + `server/` (HTTP/WebSocket), giao diện React 19 + Vite + CodeMirror 6 trong `web/`.
- **Thư viện xuất bản cờ vua** `@local/chessbook:0.1.0` (Typst) trong `chessbook/lib`.

Desktop và server dùng chung `service.ServiceFacade` (không phụ thuộc Gio), `agent/`, `lsp/`, `typst/`.

## 2. Bản đồ repo

| Đường dẫn | Vai trò |
|---|---|
| `app.go` | Entry desktop: `service.NewService` → `ui.NewUI` → `ui.Loop` |
| `cmd/typstify-server/` | Entry server headless (cờ `-addr`, `-password`, `-project`, `-static-dir`, `-project-root`) |
| `server/` | Tầng vận chuyển HTTP/WS: auth, workspace, export, preview, packages, settings, fonts, dropbox, agent, lsp |
| `service/` | Facade + bus, settings, workspace (bbolt), filewatcher, mcp, dropbox, remote, projectstore, fonts |
| `agent/` | Client ACP, `SessionManager`, phiên chat, `Broker`, MCP server nhúng, `RemoteChatSession` |
| `lsp/` | Client tinymist (JSON-RPC), completion, `PreviewService`, click-to-source |
| `typst/` | Bọc CLI `typst` (compile/watch/eval/query), `export.CompileHelper`, `pkg/` (Tpix) |
| `editor/` | Editor lõi: autosave, git gutter diff, highlight, search |
| `ui/` | Giao diện Gio: `editors`, `preview`, `navpanel`, `pkgmgmt`, `assistant`, `settings`, `dialog`, `remoteproject` |
| `web/` | Frontend React (`src/components`, `src/lib`, `src/api`) |
| `chessbook/` | Thư viện Typst cờ vua: `lib/`, `templates/`, `data/`, `demo_collection/` |
| `i18n/`, `fonts/` | i18n desktop (en-US, zh-CN, de-DE); font nhúng của UI desktop |
| `scripts/` | `check-chessbook.sh`, `install-chessbook.sh/.ps1` |
| `docs/` | Tài liệu (xem `docs/README.md`), `docs/plans/` là kế hoạch lịch sử |
| `.claude/skills/` | Skill dự án (`chess-pdf-to-typst`) |

## 3. Lệnh build / chạy / kiểm thử

```sh
go run .                                   # desktop (cần typst + tinymist cạnh app hoặc đặt trong Settings)
go run ./cmd/typstify-server               # server headless
go build -o bin/typstify-server ./cmd/typstify-server
cd web && npm ci && npm run build          # frontend (tsc -b && vite build)
cd web && npm run dev                      # Vite dev, proxy /api /preview /ws tới TYPSTIFY_BACKEND
cd web && npm test                         # vitest
cd web && npm run lint                     # oxlint
go test ./...                              # package UI desktop cần header X11/Wayland
scripts/check-chessbook.sh                 # biên dịch mọi demo/template chessbook
powershell -File scripts/package-desktop.ps1 -Version 0.1.0   # bộ cài Windows -> dist/
docker compose up -d --build               # server + Caddy TLS
```

Không có Justfile/Makefile. Bản desktop phát hành cần `gogio` và CGO. CI: `.github/workflows/ci.yml`.

## 4. Quy ước làm việc

- Giao tiếp và tài liệu bằng **tiếng Việt**; code, đường dẫn, định danh giữ tiếng Anh.
- Commit message theo kiểu `feat(scope): ...` / `fix(scope): ...` (tiếng Anh), kết thúc bằng dòng `Co-Authored-By` khi có yêu cầu.
- Logic dùng chung đặt ở `service/`, `agent/`, `lsp/`, `typst/`; `server/` chỉ làm transport. Không kéo phụ thuộc Gio vào `service/`.
- Server: mọi đường dẫn file phải qua `resolveInRoot` (`server/paths.go`); không lộ secret (`/api/settings/tpix` cố ý không expose).
- Settings mới: thêm `Model` (`Save/Load/Validate`), và `RemoteApplier` nếu cần đồng bộ desktop↔web. Section `agent` là máy cục bộ, không sync.
- Thay đổi hành vi thì cập nhật `docs/` tương ứng và thêm mục vào `docs/HISTORY.md`.
- `.gitignore` đang loại `.claude/`, `demo/`, `scratch/`, `*.exe`, `.env` — đừng giả định các thư mục này được theo dõi.

## 5. Cạm bẫy đã biết (tóm tắt — chi tiết ở `docs/MEMORY.md`)

- Preview SVG trên web **chưa có** click-to-source; chỉ desktop và iframe tinymist có.
- Scroll sync bằng vị trí heading thật chỉ có ở web (`typst.QueryHeadingPages`).
- `tinymist.doKillPreview` phải truyền id `"default_preview"`.
- Callback LSP chạy trên goroutine riêng: đụng UI Gio phải gọi `RefreshWindow()`.
- `bus.Subscribe` panic nếu đăng ký trùng khoá `%p:name`.
- Typst: `#` trong `[...]` phải escape `\#`; `=` đầu content block bị hiểu là heading (viết `[#"="]`).

---

# Typstify Chessbook Engine Guide for AI Agents

Typstify includes the built-in chess package `@local/chessbook:0.1.0`.

## ⚠️ Important Rules for AI Assistants

1. **NEVER manually declare `#let` mock definitions** for chessbook functions (e.g. `#let lesson-header`, `#let chess-quote`, `#let instructor-note`, `#let practice-question`, `#let eco-header`, `#let game-header`, `#let puzzle-card`).
2. **ALWAYS include `#import "@local/chessbook:0.1.0": *` at LINE 1 (the very top of the file)** before any chess notation, diagrams, templates, or helpers. Typst executes sequentially top-to-bottom; calling a function before its `#import` causes `unknown variable` compilation errors.
3. In Typst content blocks `[...]`, `#` unconditionally starts a code expression. Literal hash `#` MUST be escaped as `\#` (for example in table headers: `table.header([*\#*], [*Trắng*], [*Đen*])`).

## Available Functions in `@local/chessbook:0.1.0`

### 1. Document Initialization
- `#show: chess-book-init.with(title: "...", subtitle: "...", author: "...", paper-size: "16x24" | "a5" | "a4")`
- `#show: chess-worksheet-init.with(title: "...", subtitle: "...", author: "...", date: "...", paper-size: "a4")`
- `#show: chess-magazine-init.with(magazine-title: "...", issue: "...")`

### 2. Puzzle & Tactics Module (`puzzle.typ`)
- `#render-puzzle-collection(puzzle-data, layout: "16x24-2x3" | "a4-3x4", title-prefix: "...", start-number: 1, show-page-solutions: true, show-end-appendix: false)` (Tự động chia trang và xuất bài tập từ mảng JSON/CSV)
- `#csv-to-puzzles(csv-data)` (Chuyển đổi dữ liệu từ hàm `csv("...")` chuẩn Typst sang mảng đối tượng bài tập)
- `#puzzle-grid-a4(puzzles: (...))` (Lưới 12 bài tập A4: 3 cột x 4 hàng, tự căn chỉnh `size: 13.5pt` vừa khít 1 trang)
- `#puzzle-grid-16x24(puzzles: (...))` (Lưới 6 bài tập 16x24cm: 2 cột x 3 hàng, `size: 15pt` chuẩn in sách)
- `#puzzle-card(fen, number: 1, title: "...", turn: "w" | "b", difficulty: 1..5, hint: "...", size: 16pt, compact: false, solution: "...", arrows: ())` (Note: both `turn` and `to-move` are accepted)
- `#difficulty-stars(level)`
- `#upside-down-solutions(( "1": "1. e4", "2": "1. d4" ))`
- `#render-puzzle-solutions()`

### 3. ECO & Openings Module (`eco.typ`)
- `#eco-header(code: "C 58", name: "...", subname: "...", intro-moves: "...")`
- `#eco-table(columns-header: ("Trắng", "Đen", "Đánh giá"), rows: (...))`
- `#opening-diagram-box(fen, title: "...", turn: "w", eval-text: "+=", caption: "...", arrows: ())`

### 4. Magazine & Articles Module (`magazine.typ`)
- `#game-header(white: "...", black: "...", event: "...", site: "...", date: "...", result: "1 - 0", eco: "...")`
- `#column-diagram(fen, move-num: "12...", caption: "...", turn: "w", arrows: ())`
- `#chess-quote(author: "Garry Kasparov")[Nội dung trích dẫn...]`

### 5. Courseware & Lesson Plans Module (`courseware.typ`)
- `#lesson-header(lesson-num: 1, title: "...", level: "...", duration: "...", objective: "...")`
- `#concept-box(title: "Khái niệm Then chốt")[Nội dung...]`
- `#teaching-diagram(fen, title: "...", turn: "w", size: 18pt, arrows: (), caption: "...")`
- `#practice-question(number: 1, question: "...", choices: ("A...", "B..."), answer: "...")`
- `#instructor-note[Lưu ý cho HLV...]`

### 6. Figurines & Symbols Module (`symbols.typ`)
- Pieces: `#wK`, `#wQ`, `#wR`, `#wB`, `#wN`, `#wP`, `#bK`, `#bQ`, `#bR`, `#bB`, `#bN`, `#bP`
- NAG Glyphs: `#nag("1")` (!), `#nag("2")` (?), `#nag("3")` (!!), `#nag("4")` (??), `#nag("14")` (⩲), `#nag("16")` (±), `#nag("10")` (=), `#nag("7")` (□)
- Indicators: `#turn-indicator("w")`, `#turn-indicator("b")`, `#note-num(1)`
- Board: `#chess-board(fen, size: 16pt, reverse: false, arrows: ())`
