# Lịch sử phát triển Typstify

Nguồn: `git log` (238 commit tại 2026-09-29) và `docs/plans/*`. Ngày theo commit. Mục "Phiên làm việc" ghi lại các lần cộng tác với AI để làm cơ sở tiếp nối.

## Các mốc chính

| Giai đoạn | Ngày | Nội dung tiêu biểu (commit) |
|---|---|---|
| Khởi đầu mã nguồn mở | 2026-03-03 → 03-06 | Initial commit / open source release `69a8a95`; cải thiện highlight, hiệu năng editor, refactor previewer |
| Bản web đầu tiên | 2026-09-18 → 09-21 | Kế hoạch web + VPS; thay preview WebSocket bằng preview PDF tĩnh `899adc8`; lưu lựa chọn model/mode agent `9ff1ccc`; giới hạn Open/Create Project trong root cố định trên Docker `7246fe2` |
| Biến thành "Chess Publishing Studio" | 2026-09-21 | `be8e258` thư viện chessbook + công cụ cờ vua; splitter kéo được, tạo file/thư mục trong FileTree, welcome card cờ vua, quick actions cho AI |
| Thiết kế thương hiệu Dương Sinh | 2026-09-24 | Design tokens sáng/tối, giao diện tiếng Việt, toolbar cờ vua, màn hình vào có thương hiệu (PR #1) |
| Rà soát và làm cứng | 2026-09-25 | Kế hoạch rà soát `7141973`; đóng WebSocket preview không xác thực, vá deadlock/race agent, giới hạn compile song song, đường dẫn an toàn symlink `ddf0508`; CI + test `7856648`; code-split web `1d6b2f5`; OAuth loopback cho agent `67aa8f3`, `19a2664`; bundle Antigravity `4eb1822`; chessbook chạy được trên server `26f0d13` |
| Nhập liệu cờ vua | 2026-09-25 → 09-27 | Cải thiện nhập PGN, thư viện template `c008d26`; escape `#` trong `table.header` `51b73bf`; sửa tương thích tham số, tự phát hiện import `c5c6088`; gỡ `#let` mock do AI sinh `2a7d32a`; hoist `#import` lên dòng 1 `e0c496d`; snippet VS Code `15da32e` |
| Mẫu puzzle in ấn | 2026-09-26 | Template A4 (12 bài, 3x4) và 16x24cm (6 bài, 2x3) `fea2a0a`; nhập JSON/CSV/FEN với tự chia trang `9f747df`; Dropbox sync + import/export dự án `c028966` |
| Markdown → Typst và tài khoản | 2026-09-27 | Bộ chuyển Markdown → Typst `84657aa`; sửa FEN (`fen:` prefix, code fence) `259d6d4`; đăng ký, đổi mật khẩu, phiên bền `f2fa7dc`; nâng cấp giao diện nhập cờ `7748b75`; cỡ chữ tuỳ ý + 1–2 cột `ea883f8`, áp dụng toàn tài liệu `232b24e` |
| Preview tức thời | 2026-09-27 | Instant live preview + đồng bộ cuộn theo con trỏ `4169ee9` |
| Chỉnh preview cho chính xác | 2026-09-28 | Namespace id SVG theo trang, ảo hoá render `571e44c`; hiển thị đủ chiều cao trang `1f4c80b`; sửa url(#...) `8aea7ca`; phát hiện `#pagebreak` `c5c54e3`; chế độ tinymist-exact `29c170e`; sửa Host/Origin khi proxy tinymist `0cc731d`, `e0c56db` |
| Đồng bộ vị trí heading thật | 2026-09-28 | `a14f3bd` dùng `QueryHeadingPages` + `/api/preview/anchors` |
| Khoảng cách tính năng desktop ↔ web | 2026-09-28 | Server: lịch sử phiên agent, quản lý font, tuỳ chọn export `d939ea7`; web: bù tính năng editor/preview/export/agent/settings `7266554`; desktop: click-to-source, chọn phiên bản gói, cài đặt font `7a82251` |
| Đồng bộ desktop ↔ web | 2026-09-29 | `c3cd2ef` sync settings, phiên AI agent từ xa, dự án remote (Bearer token) |

## Kế hoạch trong `docs/plans/`

Đây là nhật ký ý định lúc viết. Đối chiếu hiện trạng bằng git log ở trên và [ROADMAP.md](ROADMAP.md).

| File | Ngày | Chủ đề |
|---|---|---|
| `plan_exploration_and_setup.md` | 2026-09-18 | Tìm hiểu, giới thiệu, vận hành Typstify |
| `plan_fix_preview.md` | 2026-09-18 | Sửa split-view/live preview desktop (StartTimeout tinymist 25s, `togglePreview` im lặng khi lỗi, vị trí webview) |
| `plan_web_version.md` | 2026-09-20 | Phiên bản web |
| `plan_vps_dokploy_deployment.md` | 2026-09-20 | Docker Compose + Caddy TLS + Claude Code/Antigravity qua ACP trên VPS |
| `chess_web_publishing_plan.md` | 2026-09-21 | Nền tảng web Chess Publishing Studio |
| `plan_codebase_review_2026-09.md` | 2026-09-25 | Rà soát toàn diện: điểm số từng mảng, 22 phát hiện, kế hoạch 6 giai đoạn |
| `plan_chess_improvements.md` | 2026-09-25 | Nhập PGN, xếp bàn cờ, template, cơ chế biên dịch |
| `plan_chess_theory_fen_templates.md` | 2026-09-27 | Mẫu chèn thế cờ (FEN) sau diễn giải lý thuyết |
| `plan_global_font_size_and_columns.md` | 2026-09-27 | Cỡ chữ và số cột toàn tài liệu |
| `plan_instant_preview_and_cursor_sync.md` | 2026-09-27 | Preview tức thời (debounce 300ms) + cuộn theo con trỏ |
| `plan_markdown_to_typst_chess.md` | 2026-09-27 | Markdown → Typst chuyên cờ vua + VPS Dokploy |
| `plan_ui_data_import_and_markdown_enhancements.md` | 2026-09-27 | Giao diện nhập liệu cờ, Markdown, xuất bản |
| `plan_desktop_installer.md` | 2026-09-29 | Đóng gói desktop Windows: `gogio` + typst/tinymist/chessbook → bộ cài Inno Setup hoặc zip portable |
| `plan_user_auth_and_persistence.md` | 2026-09-27 | Tài khoản, đổi mật khẩu, duy trì đăng nhập |
| `plan_desktop_file_chooser.md` | 2026-10-05 | Cải tiến toàn diện File Chooser & Quản lý File/Thư mục Desktop |

## Phiên làm việc

### 2026-10-05 — Đợt 7: Cải tiến toàn diện File Chooser & Quản lý File/Thư mục Desktop (thay thế gioview/explorer)
- **Yêu cầu**: Xây dựng mới hoàn toàn component `ui/filechooser` thay thế cho `github.com/oligo/gioview/explorer`. Sửa lỗi ổ đĩa Windows hiển thị thành dấu `\`, thiếu Volume Label (`Local Disk (C:)`, `Data (D:)`, `Google Drive (G:)`); thêm Breadcrumbs có thể click từng cấp và ô nhập trực tiếp đường dẫn (Address Bar) hỗ trợ paste/enter; bổ sung danh mục Favorites đầy đủ (Home, Desktop, Documents, Downloads); sắp xếp danh sách file luôn gom Thư mục lên trước Files, hỗ trợ sort đa cột (Name, Date Modified, Size) kèm icon nhận diện từng loại file (.typ, .pdf, image, font, code, binary); hỗ trợ tạo New Folder inline và phím tắt (Enter, Esc, F5).
- **Quyết định & Thực hiện**:
  1. `ui/filechooser/locations_windows.go` & `locations_unix.go`: Tự động quét và phát hiện các ổ đĩa logic trên Windows (`GetLogicalDrives`, `GetVolumeInformation`, `GetDriveType`), format tên hiển thị chuẩn mực `Tên Ổ Đĩa (C:)` / `Local Disk (C:)`. Trên Unix/macOS quét mount points từ `/`, `/Volumes`, `/media`.
  2. `ui/filechooser/locations.go`: Tự động lấy các thư mục Favorites khả dụng (Home, Desktop, Documents, Downloads).
  3. `ui/filechooser/breadcrumbs.go` & `navigation.go`: Tách đường dẫn thành từng cấp phân cấp (Breadcrumb segments) cho phép nhảy nhanh; hỗ trợ Direct Address Bar mode để nhập/dán đường dẫn trực tiếp và tìm kiếm file tức thời; nút điều hướng Back, Forward, Up, Refresh.
  4. `ui/filechooser/entry_list.go`: Đọc thư mục và áp dụng quy tắc sắp xếp cốt lõi: Thư mục (Folders) luôn luôn đứng trước Files; hỗ trợ sort 2 chiều theo Name, Date Modified, Size; hiển thị icon nhận diện theo loại file (màu vàng cho thư mục, cyan cho .typ, đỏ cho .pdf, xanh cho ảnh, tím cho font, xanh dương cho code).
  5. `ui/filechooser/bottom_bar.go`: Thêm form inline tạo "New Folder", vùng hiển thị file đang chọn, ô nhập tên file cho Save As, và các nút thao tác Cancel / Open / Select Folder / Save.
  6. `ui/filechooser/dialog.go` & `file_chooser.go`: Cài đặt view modal dialog tương thích 100% với API `ChooseFolder()`, `ChooseFile()`, `ChooseFiles()`, `CreateFile()`.
  7. Chuyển đổi toàn bộ 7 callsites trong `ui/ui.go`, `ui/welcome.go`, `ui/home.go`, `ui/commandbar/commandbar.go`, `ui/dialog/create_project.go`, `ui/navpanel/menu_panel.go`, `ui/settings/fonts.go`.
- **Kiểm chứng**:
  - Unit test `ui/filechooser`: `TestDetectVolumes`, `TestGetFavorites`, `TestParseBreadcrumbs`, `TestHistoryStack`, `TestReadDirectoryAndSorting` (Sort Name, Size DESC, Name DESC, Folders first), `TestExtensionFilter` PASS 100%.
  - `go test ./ui/...` PASS.
  - `go build .` và `go build ./cmd/typstify-server` build thành công hoàn hảo.
- **File đã đổi**: `ui/filechooser/*`, `ui/ui.go`, `ui/welcome.go`, `ui/home.go`, `ui/commandbar/commandbar.go`, `ui/dialog/create_project.go`, `ui/navpanel/menu_panel.go`, `ui/settings/fonts.go`, `ui/uitokens/tokens.go`, `docs/HISTORY.md`.

### 2026-10-05 — Đợt 6: Cải thiện UI desktop — Shortcut còn thiếu, Command Palette & công tắc WrapLine
- **Yêu cầu**: Bổ sung các phím tắt còn thiếu (Ctrl+W đóng tab, Ctrl+Tab/Ctrl+Shift+Tab/Ctrl+1..9 chuyển tab, Ctrl+, mở Cài đặt, Ctrl+N tạo dự án, Ctrl+O mở thư mục); chuyển phím tắt wrap-line trong editor sang Ctrl+Alt+W; thêm công tắc `WrapLine` vào Settings tab Editor; xây dựng Command Palette (`ui/commandbar`) mở bằng Ctrl+Shift+P.
- **Quyết định & Thực hiện**:
  1. `editor/editor.go`: Chuyển tổ hợp phím toggle WrapLine sang `Ctrl+Alt+W` (`key.ModShortcut | key.ModAlt`), giải phóng `Ctrl+W` cho việc đóng tab.
  2. `ui/home.go`: Bổ sung lắng nghe và xử lý phím tắt toàn cục: `Ctrl+Shift+P` (mở/đóng Command Palette), `Ctrl+W` (đóng tab hiện tại), `Ctrl+Tab` / `Ctrl+Shift+Tab` (chuyển tab tới/lui), `Ctrl+1..9` (chuyển trực tiếp tới tab thứ n), `Ctrl+,` (mở Settings), `Ctrl+N` (mở modal tạo dự án), `Ctrl+O` (mở thư mục dự án).
  3. `ui/settings/subviews.go`: Thêm switch `WrapLine` vào giao diện Cài đặt tab Editor để người dùng có thể bật/tắt tính năng tự ngắt dòng.
  4. `ui/commandbar/commandbar.go`: Tạo package Command Palette dạng modal popup ở giữa phía trên màn hình, hỗ trợ tìm kiếm nhanh và thực thi các lệnh phổ biến (Tạo dự án mới, Mở thư mục, Cài đặt, Bật/tắt AI Assistant, Bật/tắt Console, Bật/tắt Drawer, Quản lý gói, Đồng bộ VPS).
- **File đã đổi**: `editor/editor.go`, `ui/home.go`, `ui/settings/subviews.go`, `ui/commandbar/commandbar.go`, `docs/HISTORY.md`.

### 2026-10-05 — Đợt 5: Cải thiện UI desktop — Bàn phím & Focus ring cho InteractiveLabel và FileTree
- **Yêu cầu**: Thêm hỗ trợ điều hướng bàn phím và focus ring cho `InteractiveLabel` (mặc định tắt, bật qua cờ `Focusable`), áp dụng trước cho outline (`ui/navpanel/outline.go`); bổ sung điều hướng phím (Mũi tên lên/xuống/trái/phải, Enter, Delete) cho `TreeView` (`widgets/filetree/tree.go`).
- **Quyết định & Thực hiện**:
  1. `widgets/label.go`: Thêm trường `Focusable` và `isFocused` vào `InteractiveLabel`. Khi `Focusable == true`, lắng nghe `key.FocusFilter`, các phím Enter / Return / Space để kích hoạt chọn; vẽ focus ring đường viền 1dp `th.ContrastBg` khi widget có focus; tự động focus khi click chuột. Mặc định `Focusable: false` giữ trọn vẹn tương thích cho toàn bộ các nơi khác dùng `InteractiveLabel`.
  2. `ui/navpanel/outline.go`: Bật `Focusable: true` khi khởi tạo các `InteractiveLabel` trong outline items.
  3. `widgets/filetree/tree.go`: Bổ sung filter và xử lý phím điều hướng cho cây thư mục: Mũi tên Lên/Xuống chuyển chọn node liền kề; Mũi tên Trái đóng thư mục hoặc nhảy về thư mục cha; Mũi tên Phải mở thư mục hoặc mở file; Enter/Return mở file hoặc toggle thư mục; Delete xoá node đang chọn qua `OnFileRemoveFunc`.
- **File đã đổi**: `widgets/label.go`, `ui/navpanel/outline.go`, `widgets/filetree/tree.go`, `docs/HISTORY.md`.

### 2026-10-05 — Đợt 4: Cải thiện UI desktop — Tooltip cho nút icon, hiển thị lỗi đầy đủ & dọn layout
- **Yêu cầu**: Thêm tooltip cho toàn bộ nút icon trong header editor (`ui/editors/editor_header.go`), thanh trạng thái (`ui/statusbar/statusbar.go`) và thanh preview (`ui/viewer/preview_op.go`). Chuyển toàn bộ lỗi bị nuốt (chỉ log) sang phát notification `statusbar.Notification` qua `bus.TopicStatusbarNotifyEvent` với duration 15s. Dọn dead field trong `ui/preview/previewer.go`, thêm empty state cho `ui/navpanel/outline.go`, và sửa `LoadTheme` trong `ui/windowview.go` tránh panic.
- **Quyết định & Thực hiện**:
  1. `ui/editors/editor_header.go`: Khôi phục `ViewActionState.tip wg.TipArea`, bọc các action header qua `wg.TipIconButton` và dịch tooltip qua `i18n.Translate(action.Name)`.
  2. `ui/statusbar/statusbar.go`: Thêm tooltip `"AI Assistant"` và `"Console"` cho hai nút góc phải statusbar.
  3. `ui/viewer/preview_op.go`: Dịch tooltip "Refresh preview" qua `i18n.Translate`.
  4. Emit notification lỗi (Level 2/1, Duration 15s) thay vì chỉ log: `ui/navpanel/filetree.go` (lỗi restore tree, open explorer, open file, delete file), `ui/editors/typst_view.go` (lỗi start ACP session, list remote sessions), `ui/assistant/chat.go` (lỗi ACP session, load session, preview server address), `ui/settings/fonts.go` (lỗi file chooser, upload font, delete font), `ui/dialog/create_project.go` (trả lỗi đúng từ `createPackageProject`), `ui/navpanel/menu_panel.go` (lỗi VPS sync, choose folder), `ui/welcome.go` (lỗi choose folder).
  5. `ui/preview/previewer.go`: Xóa field chết `err error`.
  6. `ui/navpanel/outline.go`: Thêm `layoutEmptyState` hiển thị nhãn "No outline available" khi tài liệu chưa có heading thay vì để trống khung.
  7. `ui/windowview.go`: Cập nhật `LoadTheme` bổ sung `th.Face`, đăng ký `"codeColorScheme"`, `"semanticPalette"` và fallback `"Dương Sinh Light"`.
- **File đã đổi**: `ui/editors/editor_header.go`, `ui/statusbar/statusbar.go`, `ui/viewer/preview_op.go`, `ui/navpanel/filetree.go`, `ui/editors/typst_view.go`, `ui/assistant/chat.go`, `ui/settings/fonts.go`, `ui/dialog/create_project.go`, `ui/navpanel/menu_panel.go`, `ui/welcome.go`, `ui/preview/previewer.go`, `ui/navpanel/outline.go`, `ui/windowview.go`, `docs/HISTORY.md`.

### 2026-10-05 — Đợt 3: Cải thiện UI desktop — Token spacing/màu semantic & Theme thương hiệu Dương Sinh
- **Yêu cầu**: Xây dựng hệ thống design tokens cho desktop (`ui/uitokens/`), cung cấp bảng màu semantic (lỗi, cảnh báo, thành công, thông tin), thêm theme thương hiệu Dương Sinh (sáng/tối) tương thích với chessbook/web và đặt làm mặc định.
- **Quyết định & Thực hiện**:
  1. `ui/uitokens/tokens.go`: Khai báo hằng số khoảng cách chuẩn `SpacingXXS` (2dp) -> `SpacingXXL` (32dp), `RadiusSmall`/`Medium`/`Large`, và helper hàm `ErrorColor`, `WarningColor`, `SuccessColor`, `InfoColor` tra cứu theo theme.
  2. `ui/palette/palette.go`: Thêm struct `SemanticPalette`, đăng ký hai theme thương hiệu `"Dương Sinh Light"` (Navy `#2B3990`, `#1F2A6E`, Accent Gold `#C9A227`, Background `#F8F9FA`) và `"Dương Sinh Dark"` (Navy `#1C2140`, Accent Gold `#E5B83A`, Background `#0F141C`) đồng nhất với bảng màu trong `chessbook/lib/theme.typ` và `web/src/lib/theme.ts`. Cập nhật `ThemeNames()` đưa 2 theme Dương Sinh lên đầu danh sách.
  3. `ui/ui.go`: Khởi tạo semantic palette và đặt theme mặc định là `"Dương Sinh Light"`.
  4. `service/settings/models.go`: Đặt giá trị mặc định `Theme: "Dương Sinh Light"`.
  5. Chuyển đổi toàn bộ màu thông báo/lỗi/thành công hardcode sang `uitokens` trong `ui/dialog/dialog.go`, `ui/dialog/bibliography.go`, `ui/dialog/publish_pkg.go`, `ui/settings/update_check.go`, `editor/statusbar.go`, `ui/settings/agent.go`, `ui/navpanel/menu_panel.go`, `ui/statusbar/statusbar.go`.
- **File đã đổi**: `ui/uitokens/tokens.go`, `ui/palette/palette.go`, `ui/ui.go`, `service/settings/models.go`, `ui/dialog/dialog.go`, `ui/dialog/bibliography.go`, `ui/dialog/publish_pkg.go`, `ui/settings/update_check.go`, `editor/statusbar.go`, `ui/settings/agent.go`, `ui/navpanel/menu_panel.go`, `ui/statusbar/statusbar.go`, `docs/REPLICATION.md`, `docs/HISTORY.md`.

### 2026-10-05 — Đợt 2: Cải thiện UI desktop — Việt hoá i18n & chuẩn hoá chuỗi nguồn (đóng ROADMAP A3, A4)
- **Yêu cầu**: Thêm locale tiếng Việt `vi-VN` vào desktop app, sửa lỗi lệch mã locale (`en-US` vs `en-us`), chuẩn hoá chuỗi nguồn về tiếng Anh, bọc toàn bộ chuỗi còn sót trong `i18n.Translate()`, cập nhật catalog đa ngôn ngữ.
- **Quyết định & Thực hiện**:
  1. `i18n/localizer.go`: Thêm `vi-vn` vào `Locales`, chuẩn hoá tìm kiếm mã locale qua `strings.ToLower(strings.TrimSpace(id))` để tương thích cả `en-US` lẫn `en-us`, fallback an toàn `en-us` khi gặp locale lạ.
  2. Chuẩn hoá chuỗi nguồn trong `ui/settings/sync.go`, `ui/remoteproject/view.go`, `ui/dialog/export.go` sang tiếng Anh chuẩn (`-srclang=en-US`), dùng placeholder format (`%s`, `{ServerURL}`, `{Format150405}`) thay vì nối chuỗi thủ công.
  3. Bọc các chuỗi switch và tiêu đề còn sót trong `ui/settings/lsp.go`, `ui/settings/subviews.go`, `ui/settings/agent.go`, `ui/assistant/chat.go`, `ui/pkgmgmt/manage.go`, `ui/pkgmgmt/card.go`, `ui/editors/typst_view.go`, `ui/editors/generic_text_view.go`, `ui/viewer/image_view.go`, `ui/crash_report.go`, `ui/assistant/sessions.go` qua `i18n.Translate()`.
  4. Tạo catalog `vi-VN/messages.gotext.json` với bản dịch tiếng Việt đầy đủ và chính xác cho toàn bộ giao diện; cập nhật `catalog.go` sinh tự động qua `gotext`.
  5. Cập nhật `service/settings/models.go`: mặc định `Language: "en-us"`, ưu tiên `Roboto Mono` trong `TypeFace` để hỗ trợ dấu tiếng Việt đầy đủ.
- **Kiểm chứng**: Unit test `TestVietnameseLocale` & `TestEnglishLocale` PASS; toàn bộ test `service/settings`, `editor`, `utils` PASS; `go build` sinh `typstify.exe` thành công.
- **File đã đổi**: `i18n/localizer.go`, `i18n/localizer_test.go`, `i18n/translations/translations.go`, `i18n/translations/catalog.go`, `i18n/translations/locales/vi-VN/messages.gotext.json`, `service/settings/models.go`, `ui/ui.go`, `ui/settings/sync.go`, `ui/remoteproject/view.go`, `ui/dialog/export.go`, `ui/settings/lsp.go`, `ui/settings/subviews.go`, `ui/settings/agent.go`, `ui/assistant/chat.go`, `ui/pkgmgmt/manage.go`, `ui/pkgmgmt/card.go`, `ui/editors/typst_view.go`, `ui/editors/generic_text_view.go`, `ui/viewer/image_view.go`, `ui/crash_report.go`, `ui/assistant/sessions.go`, `docs/ROADMAP.md`, `docs/HISTORY.md`.

### 2026-10-05 — Cải tiến UI Desktop Đợt 7: Tích hợp bộ gõ tiếng Việt Telex cho Editor
- **Nội dung thực hiện**:
  - Tích hợp bộ gõ Telex `utils/vietnamese/telex.go` vào luồng nhập liệu của editor (`editor/editor.go`, `internal/gvcode`) qua cơ chế `gvcode.AddTextInputHook`.
  - Tự động phát hiện và biến đổi các từ tiếng Việt gõ theo quy tắc Telex (`tieengs Vieejt` -> `tiếng Việt`, `ddoongf booj` -> `đồng bộ`, `tawng kichs thuwocs` -> `tăng kích thước`).
  - Thêm thuộc tính `EnableTelex` vào `service/settings/models.go` (`defaultEditorSettings`: mặc định `"false"` để không làm phiền người dùng sử dụng bộ gõ ngoài).
  - Thêm công tắc bật/tắt bộ gõ Telex trong tab Settings -> Editor (`ui/settings/subviews.go`), đồng bộ realtime qua EventBus `bus.TopicSettingsUpdated`.
  - Giữ nguyên trạng thái khi ở chế độ ReadOnly hoặc khi gõ trong thanh tìm kiếm/dialog.
  - Viết unit test tích hợp `editor/telex_integration_test.go` kiểm thử toàn diện các quy tắc gõ và chuyển đổi chế độ.
- **File đã cập nhật**: `editor/editor.go`, `service/settings/models.go`, `ui/settings/subviews.go`, `editor/telex_integration_test.go`, `docs/HISTORY.md`.

### 2026-10-05 — Cải tiến UI Desktop Đợt 6: Phím tắt còn thiếu, Command Palette, công tắc WrapLine
- **Nội dung thực hiện**:
  - Chuyển phím tắt WrapLine trong editor từ `Alt+Z` sang `Ctrl+Alt+W` theo kế hoạch.
  - Thêm công tắc cấu hình WrapLine vào Settings -> Editor (`ui/settings/subviews.go`).
  - Xây dựng component Command Palette `ui/commandbar/commandbar.go` (`Ctrl+Shift+P`), cho phép tìm kiếm nhanh hành động, chuyển đổi khung nhìn, kích hoạt AI Assistant (`Ctrl+L`), Console (`Ctrl+K`), Sidebar (`Ctrl+D`), New Project (`Ctrl+N`), Open Folder (`Ctrl+O`), Settings (`Ctrl+,`), Package Management, VPS Sync.
  - Tích hợp phím tắt toàn cục trong `ui/home.go` (`Ctrl+W`, `Ctrl+Tab`, `Ctrl+Shift+Tab`, `Ctrl+1..9`, `Ctrl+Shift+P`).
- **File đã cập nhật**: `editor/editor.go`, `ui/settings/subviews.go`, `ui/commandbar/commandbar.go`, `ui/home.go`, `service/bus/topics.go`, `docs/HISTORY.md`.

### 2026-10-05 — Cải thiện giao diện Desktop (7 đợt)
- **Yêu cầu**: rà soát và cải thiện giao diện Desktop theo cách vá dần, an toàn; ưu tiên Việt hoá, hiệu suất làm việc, thẩm mỹ/nhất quán, cấu trúc layout. Kế hoạch: `docs/plans/plan_desktop_ui_improvements.md`.
- **Thực hiện** (Antigravity triển khai, Claude Code rà soát và kiểm chứng):
  1. `39848df` — Ctrl+L mở AI Assistant (toggle read-only chuyển sang Ctrl+Shift+R); phím điều hướng search (Enter/Shift+Enter/F3); tab Settings đúng qua `ui/settings/tabs.go`; nút đóng tab nhìn thấy được; thông báo thứ hai không bị cắt.
  2. `c23a651` — locale vi-VN (gotext, sinh lại `catalog.go`); chuẩn hoá chuỗi nguồn tiếng Anh; sửa lệch mã `en-us`/`en-US`. Đóng ROADMAP A3, A4. **Lưu ý**: kế hoạch ban đầu đề xuất bảng Go thay vì gotext để tránh diff lớn; commit này đã sinh lại `catalog.go` (~2089 dòng).
  3. `ad1e80d` — token giao diện trong `ui/uitokens` (kế hoạch ghi `ui/palette`); theme Dương Sinh sáng (mặc định) và tối, lấy từ `chessbook/lib/theme.typ`.
  4. `bf5bc80` — tooltip cho nút icon; lỗi hiện trên statusbar 15 giây thay vì chỉ log; empty state outline.
  5. `b7e21bc` — bàn phím và focus ring cho `InteractiveLabel` (bật ở outline); file tree điều hướng bằng mũi tên/Enter/Delete.
  6. `3e25805` — phím tắt Ctrl+W, Ctrl+Tab, Ctrl+Shift+Tab, Ctrl+1..9; command palette Ctrl+Shift+P; wrap line chuyển sang Ctrl+Alt+W, có công tắc trong Settings.
  7. `be1060b` — bộ gõ Telex nối vào editor, công tắc `EnableTelex` trong Settings → Editor, mặc định tắt.
- **Kiểm chứng**: `go test` pass cho `editor`, `service`, `server`, `utils`; `go build` root và `cmd/typstify-server` thành công. **Chưa** kiểm tra giao diện bằng mắt trên màn hình.
- **Việc còn lại**: chủ dự án duyệt bản dịch tiếng Việt, cảm quan theme Dương Sinh, nghiệm thu Telex; deploy bản mới lên VPS.
### 2026-10-05 — Đồng bộ file một chiều Local -> VPS (typstify-server)
- **Yêu cầu**: thay Dropbox (hay hết hạn access token) bằng đồng bộ file trực tiếp từ desktop lên VPS chạy `typstify-server`, xác thực bằng Bearer token.
- **Quyết định**:
  - Chỉ hướng Local -> VPS. Trong `PerformSync` không xoá file chỉ có trên VPS; xoá từ xa chỉ xảy ra qua `DeleteFile` khi file đang mở bị xoá local.
  - So khớp bằng sha256 nội dung. Server lưu nguyên bytes nhận được, **không** qua `repairTypstContent`, để hash hai phía luôn khớp (nếu biến đổi thì mỗi lần sync sẽ đẩy lại mãi).
  - `.git`, `node_modules`, `dist`, `.tmp` và `.typstify` (chứa token máy) luôn bị bỏ qua ở cả client lẫn server.
  - Cấu hình `VPSSyncSettings` là credential riêng của máy nên không đưa vào đồng bộ settings desktop<->web.
- **Thành phần**:
  - Server: `server/sync_handler.go` với `GET /api/sync/manifest`, `GET /api/sync/pull`, `POST /api/sync/push` (body thô, ghi qua file tạm rồi rename), `POST /api/sync/delete`. Mọi đường dẫn đi qua `resolveInRoot`; route bọc `authMiddleware`.
  - Client: `service/vpssync/` (`SyncEngine`: `TestConnection`, `PushFile`, `DeleteFile`, `PerformSync`, `OnFileChanged`, `Status`), nối vào `ServiceFacade` qua `VPSSync()` và bus `TopicWorkspaceFileChanged`.
  - Settings: `service/settings/vpssync.go` (section `vpsSync`) và `Settings.VPSSync()`.
  - UI: tab "VPS Sync" trong Settings (đặt sau Agent, `settings.VPSSyncTabIdx = 8`); nút trạng thái cloud + Sync Now trên thanh menu (`ui/navpanel/menu_panel.go`).
- **Kiểm chứng**: unit test `service/vpssync` và `server/sync_handler_test.go`; chạy `typstify-server` thật bằng curl (401 khi thiếu token, chặn `.git`/`.typstify`, `../` bị neo trong root) và client `PerformSync` thật (lần 2 không đẩy file nào).
- **Việc còn lại**: `IntervalSec` đã lưu nhưng chưa có vòng lặp định kỳ; chuỗi i18n mới chưa thêm vào `i18n/`.
- **File đã đổi**: `server/server.go`, `server/security.go`, `server/sync_handler.go` (+test), `service/service.go`, `service/settings/db.go`, `service/settings/vpssync.go`, `service/vpssync/` (+test), `ui/navpanel/menu_panel.go`, `ui/settings/vpssync.go`, `ui/settings/view_setting.go`, `widgets/icons/icons.go` và 4 file `widgets/icons/lucide/cloud*.svg`, `docs/HISTORY.md`.

### 2026-09-30 — Rà soát, chuẩn hoá và đồng bộ toàn diện hệ thống tài liệu
- **Yêu cầu**: Rà soát, cập nhật và hoàn thiện toàn bộ các file tài liệu `.md` cốt lõi (`AGENTS.md`, `CLAUDE.md`, `README.md`, `docs/README.md`, `docs/TECH.md`, `docs/ARCHITECTURE.md`, `docs/MODULES.md`, `docs/ALGORITHMS.md`, `docs/API.md`, `docs/CHESSBOOK.md`, `docs/MEMORY.md`, `docs/SKILLS.md`, `docs/HISTORY.md`, `docs/ROADMAP.md`, `docs/REPLICATION.md`...) làm tài liệu đặc tả chuẩn xác phục vụ việc bảo trì, phát triển tính năng mới và nhân bản phần mềm.
- **Quyết định & Thực hiện**:
  1. Kiểm tra và đồng bộ 100% nội dung giữa `AGENTS.md` và `CLAUDE.md`.
  2. Chuẩn hoá chữ ký tham số thực tế của hàm `render-puzzle-collection` trong thư viện `puzzle.typ` (`page-title`, `show-upside-down`, `render-appendix-at-end`, `per-page`) trên `CLAUDE.md`, `AGENTS.md` và `CHESSBOOK.md`.
  3. Cập nhật `ROADMAP.md` ghi nhận hoàn thành `docs/CHESSBOOK.md`.
  4. Xác nhận tính đầy đủ của các phân tích thuật toán sâu trong `ALGORITHMS.md` (biên dịch Typst, preview SVG web, shadow file `.live_preview_*.typ`, scroll sync 2 chiều theo heading anchors, click-to-source, autosave digest an toàn, ACP turn buffer, sync settings đa diện, cơ chế bảo mật server `resolveInRoot`).
- **File đã cập nhật**: `AGENTS.md`, `CLAUDE.md`, `docs/ROADMAP.md`, `docs/HISTORY.md`.

### 2026-09-29 — Lập bộ tài liệu nền tảng
- **Yêu cầu**: tạo/cập nhật các file .md (agents, claude, readme, skills, tech, memory…) để thống kê module, mô tả thuật toán, lưu lịch sử, làm cơ sở rà soát, thêm module mới và nhân bản phần mềm.
- **Cách làm**: khảo sát chỉ-đọc bằng ba agent (desktop Go; web + server; chessbook + tài liệu), viết kế hoạch, rồi ghi tài liệu.
- **Kết quả phần này**: `README.md` (thêm tính năng + mục lục), `CLAUDE.md`/`AGENTS.md` (thêm bản đồ repo, lệnh, quy ước, cạm bẫy), và `docs/` gồm `README`, `SKILLS`, `MEMORY`, `HISTORY`, `ROADMAP`, `REPLICATION`.
- **Phát hiện đáng nhớ**: web SVG preview chưa có click-to-source; desktop chưa dùng anchors heading thật; desktop i18n không có tiếng Việt; `.claude/` và `demo/` bị `.gitignore` loại.

## Cách ghi tiếp
Thêm mục mới trên cùng phần "Phiên làm việc" với: ngày, yêu cầu, quyết định, file đã đổi, việc dang dở. Với mốc tính năng thì thêm một dòng vào bảng "Các mốc chính".
