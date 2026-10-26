# Bộ nhớ dự án (ngữ cảnh bền vững cho AI)

File này gom những điều không suy ra được từ đọc mã một lần: quyết định thiết kế, cạm bẫy đã trả giá, điều cấm. Cập nhật khi phát hiện điều mới. Mọi mục đều đã đối chiếu với mã hoặc tài liệu tại 2026-09-29; kiểm lại trước khi dựa vào.

## Bối cảnh vận hành
- Chủ dự án: công ty Cờ vua Dương Sinh. Bản web triển khai tự host (Dokploy/VPS, Docker Compose + Caddy TLS), giao diện tiếng Việt. Xem `docs/web-server.md`.
- Ngôn ngữ làm việc: tiếng Việt cho giải thích và tài liệu; code/định danh/đường dẫn tiếng Anh.
- Commit theo kiểu `feat(scope): …`, `fix(scope): …`.

## Quyết định kiến trúc
- `service.ServiceFacade` không phụ thuộc Gio → dùng chung desktop và server. UI gắn vào qua hook (`SetViewManager`). Giữ nguyên tính chất này khi thêm dịch vụ.
- `server/` chỉ là transport; logic ở `service/`, `agent/`, `lsp/`, `typst/`.
- Settings lưu JSON có timestamp từng section; đồng bộ theo last-write-wins per section (`ApplyRemote`). Section `agent` và `RemoteSettings` không đồng bộ; `tpix` (API key) không expose qua HTTP.
- Server compile giới hạn 2 slot song song, 90 giây mỗi lần (`compileSlots`).
- Web dùng `useState`/refs thuần, không router, không state library; các panel nặng lazy-load.
- Preview web có 3 chế độ: SVG (từ nội dung chưa lưu, qua file bóng `.live_preview_*.typ` cùng thư mục), PDF, iframe tinymist.
- Đồng bộ cuộn web chỉ tin anchors từ server khi số heading client (regex `^=+\s+\S`) khớp số heading server.

## Cạm bẫy đã biết
**Desktop / LSP**
- Kết nối tinymist qua stdio chỉ có một lần: `initialize` quá hạn thì pipe bị đóng, các lần thử lại trong `Client.Connect` không thể thành công. Lần khởi động đầu có thể mất hơn 15 giây (quét font, antivirus quét exe mới), nên `InitializeTimeout` để 45 giây.
- `tinymist.doKillPreview` phải truyền id `"default_preview"`; gọi không tham số thì lỗi.
- Callback `window/showDocument` chạy trên goroutine LSP: sau `NavigateToLine` phải `RefreshWindow()`; và luôn trả `ShowDocumentResult` vì đó là request.
- `bus.Subscribe` panic nếu đăng ký trùng khoá `%p:name`.
- Chỉ nhận `TextEdit` trong completion; `InsertReplaceEdit` bị bỏ qua.
- Webview native phải huỷ trễ một frame (`destroyPending`) vì chạy trong frame Gio.
- Autosave không ghi đè nếu file trên đĩa đã đổi so với `originalHash` (báo "edited elsewhere").

**Server / web**
- Preview WASM: `getBoundingClientRect()` của `g.typst-page` chỉ bao phần có mực. Kích thước trang lấy từ `data-page-width/height`, vị trí từ `transform` (`pageBoxes` trong `WasmPreview.tsx`).
- `requestFullscreen` phải gọi ngay trong trình xử lý click, gọi trong `useEffect` sẽ bị từ chối ("not granted"). Chrome điều khiển qua extension từ chối luôn, nên trình chiếu có dự phòng phủ kín cửa sổ.
- Trong Chrome tự động hoá (Claude in Chrome), trình duyệt cắt WebSocket sau khoảng 15 giây kể cả với server tối giản; đừng nhầm là lỗi của `/ws/lsp` hay `/ws/collab`.
- Worker WASM tải font từ CDN phải dùng `credentials: 'same-origin'`: CDN trả `Access-Control-Allow-Origin: *`, yêu cầu kèm credentials bị chặn và font bị bỏ qua mà không báo lỗi.
- typst.ts nhớ câu trả lời "không có tệp": phải có danh sách tệp trước lần biên dịch đầu, và `compiler.reset()` khi tệp thêm/bớt (`setManifest` trả `true`).
- Desktop Tauri (`apps/tauri`): server chạy kèm trên `127.0.0.1` với cổng được nhớ trong `<app config>/port.txt` (đổi cổng là mất localStorage/IndexedDB/Cache Storage của giao diện, vì chúng theo origin). Token khởi động đi qua biến môi trường, không qua tham số. Job Object (`KILL_ON_JOB_CLOSE`) bảo đảm server, tinymist, typst dừng theo app kể cả khi app bị giết. Nhật ký server: `%LOCALAPPDATA%\vn.duongsinh.chessstudio\logs\typstify-server.log`. Không chạy cùng lúc với bản Gio hay một `typstify-server` khác dùng chung thư mục cấu hình (bbolt chỉ mở được một lần).
- Máy chủ chỉ trả `index.html` cho đường dẫn kiểu route (không đuôi tệp); tệp không tồn tại (`/assets/*`, `*.js`, …) là 404. Đừng bỏ quy tắc này: service worker và trình duyệt sẽ lưu HTML dưới tên script.
- Service worker (`web/src/sw/sw.ts` → `dist/sw.js`): không bao giờ cache `/api`, `/ws`, `/preview`; `.wasm` không cache sẵn khi cài (khoảng 30 MB) mà cache lần đầu dùng. Bản mới chỉ kích hoạt khi người dùng bấm "Tải lại" (`SKIP_WAITING`). Thử trên Chrome của người dùng xong thì gỡ đăng ký (`navigator.serviceWorker.getRegistrations()` → `unregister`) để không ảnh hưởng cổng đó về sau.
- Chính tả: CodeMirror đặt `spellcheck=false` cho nội dung; chỉ các span văn bản được đánh dấu `spellcheck="true" lang="vi"`. Không đóng gói `dictionary-vi` (GPL).
- Preview "Tức thì" (WASM, typst.ts 0.8.0-rc3 = Typst 0.15): **không dùng `renderToSvg`** của renderer. Nó gọi `window.typstProcessSvg` rồi `unwrap()` (panic khi hàm không tồn tại, làm hỏng cả module WASM) và bỏ qua vẽ lại khi bề rộng không đổi (`data-applied-width`). Dùng `renderSvgDiff` + `patchRoot` như typst-preview (`WasmPreview.tsx`).
- `renderSvgDiff` không kèm stylesheet: lớp chữ để bôi đen (`.tsel`) phải được CSS của app làm trong suốt, nếu không nó vẽ chồng lên chữ thật (`App.css`, `.wasm-preview-doc .tsel`).
- Phiên bản Go: Docker build dùng `golang:1.25`, `go.mod` giữ `go 1.25.5`. Khi thêm hoặc nâng thư viện, kiểm tra dòng `go` trong `.mod` của nó (`go-git` 5.19+ và `x/crypto` 0.56+ đòi 1.26). `go get` tự nâng `go.mod` mà không báo.
- go-git (`service/gitsync`):
  - **Không dùng `HardReset` cho pull:** nó xoá mọi tệp chưa theo dõi, kể cả tệp bị ignore. Thay vào đó áp `DiffTree` của hai commit (hàm `applyTree`) rồi `MixedReset` để cập nhật index.
  - `AddOptions{All: true}` chỉ dùng `Worktree.Excludes`, **không** đọc `.gitignore` lẫn `.git/info/exclude`. Phải nạp mẫu vào `Excludes` (hàm `worktree()`), nếu không `.typstify/` bị commit.
  - Push bị từ chối trả `fmt.Errorf("non-fast-forward update: …")` không bọc lỗi gốc, nên chỉ nhận ra được theo chuỗi.
  - Slice Go rỗng (`nil`) thành `null` trong JSON và làm vỡ giao diện (`.length`); trả `[]` cho mọi danh sách.
  - Transport https được cài toàn cục (`client.InstallProtocol`); bộ chặn IP nội bộ nằm ở `Dialer.Control` nên DNS rebinding không vượt qua được.
- Cộng tác (`service/collab`, `lib/collab`):
  - **Không bao giờ ép văn bản chung về nội dung trên đĩa.** Sửa từ bên ngoài được gửi dạng `{base, text}` và client chỉ áp phần `base → text` (`rebaseChange`); chồng lấn thì hỏi người dùng. Cách "ép theo đĩa" cũ đã xoá mất sửa đổi ngoại tuyến.
  - Mọi lần leader lưu (kể cả khi đĩa đã trùng) phải cập nhật `base.txt`/`DiskHash`, nếu không lần khởi động lại sau sẽ báo nhầm "sửa ngoài".
  - Lần lưu nào cũng kèm hash của bản đĩa mà leader đã gộp. Máy chủ chỉ ghi khi tệp vẫn là bản đó (hoặc bản đã đồng bộ), nên autosave đến trước lần kiểm tra đĩa 2 giây không thể đè sửa đổi bên ngoài. Khi đang có xung đột thì client không lưu.
  - Phòng khoá theo (thư mục project, đường dẫn tương đối); client gửi `project` và bị từ chối (409) nếu phiên của nó đã đổi sang project khác (cùng đường dẫn tương đối có thể là tệp khác).
  - Bản IndexedDB được đặt tên theo epoch; đổi epoch thì dùng cơ sở dữ liệu mới thay vì xoá rồi mở lại trùng tên (`clearData` là bất đồng bộ).
  - Kiểm thử hai tài khoản trong cùng Chrome: dùng `127.0.0.1` và `localhost` (cookie tách biệt).
- Sau `patchRoot`, thứ tự `.typst-page` trong DOM **không** theo thứ tự trang; tính vị trí trang phải sắp theo toạ độ.
- `compiler.query()` của typst.ts parse JSON hai lần (lỗi với kết quả object); dùng `runWithWorld` + `world.query`. Range của diagnostics đánh số từ 0.
- Mỗi lần reset tài liệu (đổi file, nạp lại, mount lại) phải dùng session renderer mới và đổ toàn bộ SVG; diff chỉ hợp lệ với DOM do chính session đó vẽ.
- Cookie phiên `Secure` phụ thuộc `X-Forwarded-Proto` từ peer tin cậy (loopback/private).
- Đường dẫn file phải qua `resolveInRoot` (chặn `..` và symlink); `.git`, `.typstify` được bảo vệ.
- Lỗi compile preview trả HTTP 200 với `ok:false`.
- WebSocket agent đọc tối đa 20 MiB (ảnh dán); `/ws/lsp` từng thiếu `SetReadLimit` (xem plan rà soát, đối chiếu lại trước khi tin).
- Rò tiến trình con: không để `TYPSTIFY_SERVER_PASSWORD` lọt vào env agent/terminal.
- **Mỗi phiên đăng nhập một project** (Giai đoạn 3b). Trong `server/`, lấy thư mục project bằng `s.projectRoot(r)` / `s.projectDir(r)` (`server/session_project.go`), **không** dùng `appSrv.CurrentProjectDir()` (đó chỉ là project mặc định cho request không có phiên) và **không** gọi `SetProjectDir` (khởi động tinymist mặc định, đổi project cho mọi người); mở project thì dùng `s.openProjectFor(r, dir)`. tinymist/preview/MCP của server lấy qua `appSrv.Runtimes()` (`Acquire` giữ runtime trong thời gian kết nối, nhớ `release`), không dùng `lsp.GetLspClient`. Phân quyền (`handleRole`) kiểm user của request với project của phiên đó. Route mới đụng file project phải đăng ký bằng `handleRole` (đọc ≥ viewer, ghi ≥ editor); route đổi cấu hình máy chủ dùng `handleAdmin`.
- Test server nhiều người dùng: mỗi `apiClient` phải tự `POST /api/projects/{id}/open` (không còn "người này mở thì người kia dùng chung").
- Test trong `server/` phải dùng `testService()` (dùng chung một `ServiceFacade`): mỗi `service.NewService` mở cùng file bbolt thật, lần mở thứ hai trong cùng tiến trình hết thời gian chờ và `log.Fatal("timeout")` làm chết cả gói test mà không có `--- FAIL`. Truyền `Options.StorageDir` là thư mục tạm để không ghi vào tài khoản thật.
- Mở project trong test khởi động tinymist và file watcher, giữ thư mục trên Windows; dùng `os.MkdirTemp` + `RemoveAll` bỏ qua lỗi thay cho `t.TempDir()`.
- Chạy thử server mà không đụng dữ liệu thật: đặt `APPDATA` sang thư mục tạm (settings và tài khoản nằm dưới `os.UserConfigDir()`). Gửi JSON tiếng Việt từ Git Bash bằng `--data-binary @file`, vì tham số dòng lệnh bị đổi codepage.

**Typst / chessbook** (AI hay mắc)
- Không tự khai báo `#let` mock cho hàm chessbook; `#import "@local/chessbook:0.1.0": *` phải ở **dòng 1**.
- `#` trong `[...]` phải escape `\#`. `=` đầu content block → heading rỗng, viết `[#"="]`.
- Dòng markup bắt đầu `N.` thành danh sách đánh số; đưa movetext vào chuỗi.
- Không đặt font "chess" toàn cục (chiếm ∓/±); dùng show-rule theo dải mã quân cờ.
- `notation()` của staunton không xác thực luật cờ.
- Import tương đối `../lib/lib.typ` bị `--root` chặn trên server; dùng package `@local`.
- API không nhất quán lịch sử: `turn` và `to-move` đều được chấp nhận; theo dõi `opening-diagram-box`.

## Điều cấm / cần hỏi trước
- Không bịa dữ liệu ván cờ (Elo, danh hiệu, kết quả, địa điểm) khi PGN thiếu tag.
- Không commit `.env`, token, `*.exe`.
- Không đổi `chessbook` theo cách phá `CLAUDE.md` mà không cập nhật tài liệu và chạy `scripts/check-chessbook.sh`.

## Điểm mạnh cần giữ (theo rà soát 09/2026)
Service layer dùng chung; so sánh mật khẩu constant-time; token 32 byte crypto/rand; cookie HttpOnly + SameSite; `/ws/*` kiểm Origin; O_EXCL khi tạo file; Docker non-root + healthcheck; comment giải thích quyết định.

## Liên kết
[HISTORY.md](HISTORY.md) · [ROADMAP.md](ROADMAP.md) · [SKILLS.md](SKILLS.md) · [plans/plan_codebase_review_2026-09.md](plans/plan_codebase_review_2026-09.md)
