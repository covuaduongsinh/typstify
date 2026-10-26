# Kế hoạch: Đạt và vượt typst.app — Typstify đồng bộ Web / Desktop / Mobile

> Sau khi được duyệt, bản này được chép nguyên vào `docs/plans/plan_typst_app_parity_2026-10.md` (quy ước: mọi kế hoạch nằm ở `docs/plans/`). Mỗi giai đoạn lớn bên dưới sẽ có kế hoạch chi tiết riêng `docs/plans/plan_<giai-doan>.md` trước khi code.

## Tiến độ

- [x] Giai đoạn 0: phân tích xong, xem [docs/research/typst-app-analysis.md](../research/typst-app-analysis.md). Còn mở: chạy công cụ trình duyệt của `rea` khi Chrome bật CDP.
- [x] Giai đoạn 1 (chi tiết: [plan_phase1_multiuser.md](plan_phase1_multiuser.md))
  - [x] Băm mật khẩu argon2id, tự nâng cấp hash cũ (`server/password.go`)
  - [x] Mô hình Project/thành viên/vai trò Write·Review·Read (`service/projects`, `server/projects_api.go`)
  - [x] Share link (`?join=`), hộp thoại Chia sẻ trên web
  - [x] MFA TOTP và mã khôi phục (`server/totp.go`, `SecuritySection.tsx`)
  - Chuyển sang Giai đoạn 3: phiên tách theo user, bình luận
- [x] Giai đoạn 2 (chi tiết: [plan_phase2_wasm.md](plan_phase2_wasm.md))
  - [x] Compiler typst.ts trong Web Worker, biên dịch tăng dần, chế độ preview "Tức thì"
  - [x] Font và package qua server (`/api/wasm/*`), cache IndexedDB/Cache Storage
  - [x] Click-to-source (khớp văn bản) và cuộn đồng bộ bằng trang heading
  - Để Giai đoạn 7 (PWA): service worker cho offline hoàn toàn
- [x] Giai đoạn 3a (chi tiết: [plan_phase3_collab.md](plan_phase3_collab.md))
  - [x] Yjs + relay Go, presence, undo riêng, leader ghi đĩa
  - [x] Bình luận theo luồng, vai trò Góp ý có hiệu lực
  - [x] Ngoại tuyến (IndexedDB), gộp ba chiều sửa từ bên ngoài
- [x] Giai đoạn 4 (chi tiết: [plan_phase4_history.md](plan_phase4_history.md))
  - [x] Lịch sử phiên bản: tự chụp 5 phút, phiên bản có tên, xem khác biệt, khôi phục tệp/cả phiên bản
  - [x] Đồng bộ Git (go-git): commit, push, pull, chọn bên khi hai bên cùng đổi
- [x] Giai đoạn 6 (2026-10-25; spellcheck dùng từ điển trình duyệt vì giấy phép, chưa có vẽ tay tìm ký hiệu)
- [x] Giai đoạn 5 (2026-10-25; thư viện mẫu cờ vua + Universe, tạo dự án từ template, tài liệu gói và chèn `#import`)
- [x] Giai đoạn 7 (PWA trước, Tauri sau)
  - [x] 7a PWA (2026-10-25; chi tiết: [plan_phase7_pwa.md](plan_phase7_pwa.md)): manifest, service worker tự viết, cập nhật có nút tải lại, màn hình mất kết nối
  - [x] 7b Tauri 2 desktop (2026-10-25; chi tiết: [plan_phase7b_tauri_desktop.md](plan_phase7b_tauri_desktop.md)): `apps/tauri`, sidecar `typstify-server` + typst + tinymist, đăng nhập bằng token khởi động, bộ cài NSIS
  - [x] 7c Tauri 2 mobile (2026-10-26 → 2026-10-27; chi tiết: [plan_phase7c_tauri_mobile.md](plan_phase7c_tauri_mobile.md)): vỏ Android, proxy cục bộ tới server tự host, bố cục điện thoại thanh ký hiệu; build thành công APK debug universal (x86_64 + aarch64) và bộ cài desktop NSIS mới qua CI; iOS để sau
- [x] Giai đoạn 3b: mỗi người một project cùng lúc (tách `ServiceFacade` theo project). Kế hoạch đã chốt 2026-10-26: [plan_phase3b_per_user_projects.md](plan_phase3b_per_user_projects.md)

### Thứ tự thực hiện (chốt 2026-10-25)

Chủ dự án yêu cầu "chạy theo thứ tự tối ưu". Thứ tự đã chọn: **4 → 6 → 5 → 7 → 3b**.

1. **Giai đoạn 4 (lịch sử phiên bản) trước.** Từ khi có cộng tác, nhiều người và cả agent AI cùng ghi vào tệp. Lịch sử là lưới an toàn để xem lại và khôi phục. Nó cũng dựa trực tiếp trên luồng lưu của Giai đoạn 3.
2. **Giai đoạn 6 (hoàn thiện editor) tiếp theo.** Đây là phần dùng hằng ngày, rủi ro thấp, thấy lợi ích ngay.
3. **Giai đoạn 5 (template, Universe).** Giá trị vừa phải và độc lập với các phần khác.
4. **Giai đoạn 7 (đa nền tảng).** PWA rẻ nên làm trước; Tauri làm sau, khi web đã ổn định.
5. **Giai đoạn 3b cuối cùng.** Phải tách `ServiceFacade`, tinymist và file watcher theo từng project, là thay đổi kiến trúc lớn và rủi ro nhất. Cách dùng hiện tại (một máy chủ, một nhóm) chưa cần ngay.

## Bối cảnh

Chủ dự án muốn dùng bộ công cụ `rea` (D:\code\rea, MCP `mcp__rea__*`) để phân tích ứng dụng web typst.app, chọn ra những gì Typstify chưa có, đưa vào Typstify, và xây dựng Typstify trên ba nền tảng web, desktop, mobile đồng bộ với nhau bằng công nghệ hiện đại.

Khảo sát repo (chỉ đọc) cho thấy hiện trạng:

| Hạng mục | Typstify hiện tại | typst.app |
|---|---|---|
| Biên dịch | Chỉ phía server (typst CLI, `typst/compiler.go`, tối đa 2 slot, timeout 90 s) | WASM trong trình duyệt, preview tức thì, chạy offline |
| Preview | SVG theo trang, PDF, iframe tinymist; render lại cả trang; click-to-source chỉ trong chế độ tinymist | Render tăng dần, click hai chiều |
| Cộng tác | **Không có**: mọi user dùng chung một workspace (`server/files_api.go`) | CRDT realtime, con trỏ, comment, chia sẻ link có quyền |
| Lịch sử phiên bản | **Không có** (chỉ git status trên desktop) | Lịch sử, khôi phục, đồng bộ GitHub |
| Template | 8 template cờ vua cố định (`web/src/components/NewDocModal.tsx`) | Thư viện template từ Universe |
| Package | Có: Tpix/Universe (`service/tpix.go`, `web/src/components/PackageManager.tsx`) | Có |
| Editor | Autocomplete, diagnostics, hover, outline. Thiếu format, spellcheck, symbol picker, vim, dán ảnh/upload | Có đủ |
| Mobile | **Không có** (không có PWA, manifest hay app) | Web responsive |
| Auth | Băm salted SHA-256 (`server/auth.go`), không có vai trò | Tài khoản, team, quyền |

Typstify đã có những thứ typst.app không có, phải giữ nguyên: AI agent ACP + MCP (`agent/`), chessbook, đồng bộ Dropbox/VPS, tự host.

## Nguyên tắc pháp lý và bảo mật (bắt buộc)

- **Mật khẩu đã dán vào chat**: không dùng nó. Tôi chỉ làm việc trong phiên Chrome bạn đã đăng nhập sẵn. Vì mật khẩu đã lộ trong transcript, **nên đổi mật khẩu typst.app** sau buổi này.
- **Phân tích ở cả hai mức** (theo lựa chọn của bạn): (a) UI, luồng và tài liệu công khai; (b) bundle JS, network/WebSocket, WASM, thông qua `rea`.
  - Kết quả được ghi thành **mô tả cơ chế**, không dán mã của typst.app vào repo.
  - Phần tái hiện dùng mã tự viết cộng thư viện mã nguồn mở (typst, typst.ts, Yjs, ... đều Apache/MIT).
- Chỉ quan sát thụ động trên tài khoản của bạn: không tải lượng lớn, không thử vượt quyền, không đụng dữ liệu của người khác.

## Giai đoạn 0: Phân tích typst.app bằng rea (sau khi duyệt)

1. Mở tab mới trong Chrome đang đăng nhập (`mcp__claude-in-chrome__tabs_create_mcp`). Ghi lại các luồng: dashboard, tạo project, editor, preview, chia sẻ, comment, lịch sử, template, package, export, cài đặt.
2. Phân tích bằng `rea`:
   - `inspect_web_page` và `capture_browser_scenario`: các kịch bản gõ, cộng tác, chia sẻ.
   - `inspect_web_network_capture` và `observe_web_session`: giao thức WebSocket, sync, API REST.
   - `analyze_web_bundle`, `trace_web_module_imports`, `inspect_wasm_artifact`: kiến trúc compiler WASM, worker, cách render và cache font/package.
   - `inspect_web_event_listeners`: phím tắt, kéo thả, dán ảnh.
   - Cuối cùng `export_evidence_bundle` rồi `close_binary`.
3. Sản phẩm: `docs/research/typst-app-analysis.md`. Với mỗi tính năng ghi: cơ chế, bằng chứng, có/không trong Typstify, độ ưu tiên, công nghệ thay thế mở. Sau bước này cập nhật lại thứ tự các giai đoạn 1 đến 7 cho khớp.

## Giai đoạn 1: Nền tảng đa người dùng (điều kiện cho cộng tác và chia sẻ)

- Đổi cách băm mật khẩu sang argon2id (`golang.org/x/crypto/argon2`) trong `server/auth.go`. Hash cũ tự nâng cấp ở lần đăng nhập kế tiếp.
- Mô hình **Project** có chủ sở hữu, thành viên và vai trò owner/editor/commenter/viewer, lưu trong bbolt. Logic đặt ở `service/projectstore`, `server/` chỉ làm transport.
- Mỗi project là một thư mục riêng dưới `-project-root`. Mọi đường dẫn vẫn đi qua `resolveInRoot` (`server/paths.go`).
- Link chia sẻ có token, vai trò và hạn dùng. Dashboard web liệt kê project (thay cho workspace chung).

## Giai đoạn 2: Biên dịch WASM phía client và preview thế hệ mới

- Dùng `@myriaddreamin/typst.ts` (web compiler + renderer) chạy trong Web Worker. Mỗi phím gõ biên dịch tăng dần, renderer vẽ SVG/canvas tăng dần, không render lại cả trang.
- Font và package nạp qua proxy của server: `/api/packages/*` có sẵn, thêm phục vụ `@local/chessbook` từ `chessbook/lib` để WASM import được. Cache bằng IndexedDB để chạy offline.
- Click-to-source hai chiều trên preview web (giải quyết ROADMAP A1) bằng source-mapping của typst.ts. Scroll sync dùng lại `web/src/lib/scrollSync.ts`.
- Biên dịch server (`typst/compiler.go`) giữ lại làm fallback và cho export chất lượng cao, PDF/A (`server/export_api.go`).

## Giai đoạn 3: Cộng tác thời gian thực

- Client dùng **Yjs** + `y-codemirror.next` (gắn vào editor CM6 trong `web/src/components/Editor.tsx`): presence, con trỏ có màu và tên, theo dõi người khác.
- Server: relay `y-protocols` viết bằng Go tại `/ws/collab/:project/:file`. Update append-only lưu trong bbolt. Client gộp update bằng `Y.mergeUpdates` để nén định kỳ. Quyền kiểm theo vai trò ở giai đoạn 1.
- Ghi file `.typ` xuống đĩa (debounced, ghi nguyên tử) để CLI, tinymist, Dropbox và desktop vẫn thấy file thật.
- Comment theo luồng, neo bằng `Y.RelativePosition`, có resolve và nhắc tên.
- Offline: `y-indexeddb` gộp lại khi có mạng. Đây cũng là cơ sở đồng bộ cho desktop và mobile.

## Giai đoạn 4: Lịch sử phiên bản và Git

- Snapshot tự động (theo thời gian và phiên làm việc) và snapshot có tên. Xem diff, khôi phục toàn project hoặc từng file.
- Đồng bộ Git/GitHub hai chiều bằng `go-git`: commit, push, pull, giải quyết xung đột qua giao diện. Mở rộng `GitRepoState` có sẵn trong `service/workspace.go`.

## Giai đoạn 5: Template và Universe

- Thư viện template: template từ Typst Universe (package có `[template]`, tận dụng `typst/pkg`) cùng template cờ vua có sẵn. Mỗi template có ảnh thumbnail.
- Wizard tạo project từ template, thay cho `STARTER_TEMPLATES` cố định.
- Duyệt Universe ngay trong editor: chèn `#import` và xem tài liệu package.

## Giai đoạn 6: Hoàn thiện editor cho ngang typst.app

- Format bằng typstyle (formatter của tinymist, thêm request `textDocument/formatting` vào `web/src/lib/lspClient.ts`).
- Spellcheck có hỗ trợ tiếng Việt (Harper/Hunspell WASM).
- Symbol picker (bảng `sym.*` và vẽ tay để tìm ký hiệu), vim mode (`@replit/codemirror-vim`).
- Upload file, kéo thả vào cây thư mục, dán ảnh vào editor (tự lưu vào `assets/` và chèn `#image(...)`).
- Chế độ trình chiếu và tối ưu giao diện responsive.

## Giai đoạn 7: Ba nền tảng dùng chung một codebase React

- **Web**: React app hiện tại, thêm **PWA** (manifest, service worker, offline nhờ WASM + IndexedDB).
- **Desktop mới**: **Tauri 2** bọc `web/`, `typstify-server` chạy cục bộ dưới dạng sidecar, nên agent ACP, tinymist và chessbook vẫn đủ. Bản Gio (`app.go`, `ui/`) giữ song song trong giai đoạn chuyển tiếp, quyết định ngừng sau.
- **Mobile**: **Tauri 2 iOS/Android**, cùng UI React. Biên dịch bằng WASM trên máy, đồng bộ và cộng tác qua server tự host (Yjs + y-indexeddb). Layout touch riêng: chuyển qua lại editor/preview, thanh ký hiệu.
- Settings đồng bộ qua `RemoteApplier` có sẵn. Section `agent` vẫn là cấu hình cục bộ.
- CI (`.github/workflows/ci.yml`): build web, Tauri desktop (Windows/macOS/Linux), APK/IPA.

## Tệp và chỗ chính sẽ đụng tới

`server/auth.go`, `server/paths.go`, `server/files_api.go`, `server/export_api.go`, `server/` (thêm `collab_ws.go`, `projects_api.go`, `history_api.go`), `service/projectstore/`, `service/workspace.go`, `typst/pkg/`, `web/src/components/{Editor,PreviewPane,NewDocModal,PackageManager}.tsx`, `web/src/lib/{lspClient,scrollSync}.ts`, `web/package.json`, thư mục mới `apps/tauri/`, cùng `docs/` (`ARCHITECTURE`, `API`, `ROADMAP`, `HISTORY`, `MEMORY`) và `CLAUDE.md`/`AGENTS.md` (sửa cả hai).

## Kiểm chứng

- Mỗi giai đoạn: `go test ./...`, `cd web && npm test && npm run lint && npm run build`, `scripts/check-chessbook.sh`.
- Giai đoạn 2: tài liệu chessbook demo biên dịch được trong WASM. Độ trễ preview mỗi phím gõ dưới 100 ms trên tài liệu mẫu. Click-to-source chạy được trên SVG.
- Giai đoạn 3: E2E Playwright với 2 trình duyệt sửa cùng file: hội tụ, presence, offline rồi online lại. Kiểm quyền viewer không ghi được.
- Giai đoạn 4: snapshot, khôi phục, push/pull với repo GitHub thử.
- Giai đoạn 7: chạy `tauri dev` trên Windows, emulator Android; mở cùng project trên web, desktop và mobile, sửa đồng thời và thấy cập nhật realtime.
- Bảo mật: review riêng (agent `security-reviewer`) cho auth, chia sẻ link, WebSocket collab và path traversal.
- Mỗi giai đoạn cập nhật `docs/HISTORY.md` và commit kiểu `feat(scope): ...`.
