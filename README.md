[![license](https://img.shields.io/badge/license-Apache%20V2-green)](https://github.com/typstify/typstify/blob/main/LICENSE)

<p align="center"><img src="version/appicon.png" width="100" /></p>

# Typstify

The cross-platform desktop editor for Typst. Unlock the power of Typst with Typstify. Get the professional power of LaTeX with a modern, intuitive editor designed for seamless typesetting and development.


## Tính năng chính

- **Soạn thảo Typst** với LSP (`tinymist`): gợi ý, chẩn đoán, hover, outline; autosave có kiểm tra hash chống ghi đè; git gutter.
- **Live preview** (desktop: webview tinymist + click-to-source; web: SVG từ nội dung chưa lưu, PDF, hoặc iframe tinymist) với đồng bộ cuộn hai chiều.
- **Xuất bản** PDF / PNG / SVG / HTML, chọn phiên bản và chuẩn PDF.
- **Trợ lý AI** qua ACP + MCP nhúng (biên dịch, chẩn đoán, font, gói Typst), lịch sử phiên, hỏi quyền từng tool call.
- **Quản lý gói Typst** (Tpix): tìm, chọn phiên bản, tải, kéo phụ thuộc, đóng gói/publish.
- **Xuất bản cờ vua** `@local/chessbook`: puzzle, ECO, tạp chí, giáo án, ký hiệu; nhập PGN/CSV/JSON/FEN/Markdown.
- **Đồng bộ desktop ↔ web**: settings theo section, phiên agent từ xa, dự án trên server; Dropbox sync.
- **Web tự host**: đăng nhập nhiều người dùng, API token, quản lý font, giao diện tiếng Việt.

## Tài liệu

Mục lục đầy đủ: [docs/README.md](docs/README.md). Nhanh: kiến trúc [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md),
module [docs/MODULES.md](docs/MODULES.md), thuật toán [docs/ALGORITHMS.md](docs/ALGORITHMS.md),
lịch sử [docs/HISTORY.md](docs/HISTORY.md), hướng đi [docs/ROADMAP.md](docs/ROADMAP.md),
nhân bản/mở rộng [docs/REPLICATION.md](docs/REPLICATION.md), hướng dẫn cho AI [CLAUDE.md](CLAUDE.md).

## Run

```sh
git clone https://github.com/typstify/typstify.git
cd typstify
go run .
```

To run the app locally, you must 
* Place the executables `typst` and `tinymist` (or `typst.exe` and `tinymist.exe`) in the root folder, 
* Or set custom executable paths for Typst and Tinymist in the setting page.


## Build

This project uses [Gio](https://gioui.org/) to build the UI. To build a binary release, you have to install and use the gogio tool, please 
refer to [gio-cmd](https://git.sr.ht/~eliasnaur/gio-cmd) to learn more. Also CGO must be enabled to build it.

On Windows, `scripts/package-desktop.ps1 -Version 0.1.0` builds the app with gogio, bundles pinned `typst`/`tinymist`, the chess library and `board-n-pieces`, and produces `dist/Typstify-Setup-<version>.exe` (needs [Inno Setup](https://jrsoftware.org/isinfo.php); otherwise a portable zip with `install.ps1`).

**Important:** The typstify project is distributed as source code only. For pre-compiled binary releases, please download from the [official website](https://typstify.com/download)

## Web version (self-hosted)

`cmd/typstify-server` runs the same backend headless and serves a browser editor (`web/`,
React + CodeMirror) with live preview, export, an AI assistant and the chess publishing tools.

```sh
cp .env.example .env            # set TYPSTIFY_SERVER_PASSWORD and TYPSTIFY_DOMAIN
docker compose up -d --build    # server + Caddy with automatic TLS
```

The image bundles typst, tinymist, the chess library as the local package
`@local/chessbook:0.1.0`, its `@preview/board-n-pieces` dependency and the print fonts, so
documents compile offline. Configuration, security notes and Dokploy deployment:
[docs/web-server.md](docs/web-server.md).

For development: `go run ./cmd/typstify-server` and, in `web/`, `npm install && npm run dev`.

## Chess publishing library

`chessbook/lib` provides puzzle cards, ECO headers, magazine game cards, courseware blocks and
figurine/NAG symbols in the Dương Sinh brand style. Use it with
`#import "@local/chessbook:0.1.0": *` (pre-installed in the Docker image; on a desktop run
`scripts/install-chessbook.sh` or `scripts/install-chessbook.ps1`). Guide (Vietnamese):
[docs/CHESS_STUDIO_GUIDE.md](docs/CHESS_STUDIO_GUIDE.md).

## Tests

```sh
go test ./...                    # desktop UI packages need X11/Wayland dev headers
cd web && npm test               # web unit tests (Vitest)
scripts/check-chessbook.sh       # compiles every chess demo/template
```

CI (`.github/workflows/ci.yml`) runs these plus a Docker build on every pull request.

## Contribute

Please feel free to contribute by filing issues or creating pull requests. 

## Explore Further

-	[Official Website](https://typstify.com)

## License

This project is distributed under the [Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0), see [LICENSE](./LICENSE) for more information.