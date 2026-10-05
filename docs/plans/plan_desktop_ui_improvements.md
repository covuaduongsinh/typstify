# Kế hoạch cải thiện giao diện Typstify Desktop

> Bản kế hoạch đã được chủ dự án duyệt ngày 2026-10-05. Người triển khai: Antigravity.
> Rà soát và lập kế hoạch: Claude Code. Quy trình kết thúc hạng mục: xem `docs/ROADMAP.md` mục "Quy trình đưa một mục vào làm".


## Bối cảnh

Bản desktop (Gio, code trong `ui/`, `widgets/`, `editor/`) đã đủ tính năng nhưng phần giao diện tích tụ nhiều khiếm khuyết: giao diện lẫn hai ngôn ngữ, không có locale tiếng Việt dù người dùng là người Việt, một shortcut được quảng cáo ngay màn hình chào nhưng không hoạt động, nút đóng tab gần như vô hình, danh sách không dùng được bằng bàn phím, nhiều lỗi chỉ ghi log nên người dùng không thấy gì. Tài liệu đã ghi nợ sẵn: `docs/ROADMAP.md:11` (mục A3 — i18n desktop thiếu tiếng Việt) và `docs/HISTORY.md:59` (chuỗi VPS Sync chưa dịch).

Chủ dự án yêu cầu cải thiện cả bốn mảng — Việt hoá, hiệu suất làm việc, thẩm mỹ/nhất quán, cấu trúc layout — theo cách **vá dần, an toàn**: mỗi đợt độc lập, sau mỗi đợt phần mềm vẫn chạy, dễ hoàn tác. Chủ dự án đã chốt thêm: **có** nối bộ gõ Telex (đợt riêng, sau cùng) và **có** thêm theme thương hiệu Dương Sinh (sáng + tối).

## Đánh giá hiện trạng (có bằng chứng)

**Mức CAO**

1. **Giao diện lẫn hai ngôn ngữ.** Nhiều chuỗi tiếng Việt được bọc trong `i18n.Translate()` nhưng không có trong catalog, nên mọi locale (kể cả English) đều hiện tiếng Việt: `ui/settings/sync.go:57,67,95,102,106,108,118,134,136,142,153,205,253`, `ui/remoteproject/view.go:41,154,168,172,181,264,278,400`. *(Kiểm chứng: grep "Đồng bộ", "Test Connection", "Enable VPS sync" trong `i18n/` = 0 kết quả.)*
2. **Không có locale tiếng Việt.** `i18n/localizer.go:22-43` chỉ có `en-us`, `zh-cn`, `de`; directive gotext `i18n/translations/translations.go:14` cũng chỉ `en-US,zh-CN,de-DE`.
3. **Mã locale mặc định không khớp → Settings không chọn ngôn ngữ nào.** Mặc định `Language: "en-US"` (`service/settings/models.go:418`) nhưng ID trong `i18n.Locales` là chữ thường `"en-us"` → `Get` không khớp, trả localizer rỗng, lỗi bị bỏ (`ui/ui.go:172,185`), radio ở `ui/settings/subviews.go:178-182` không khớp option nào. *(Tự kiểm chứng.)*
4. **Ctrl+L không mở được AI Assistant khi đang soạn thảo.** Global đăng ký Ctrl+L mở chat (`ui/home.go:108,128`), nhưng editor đăng ký cùng tổ hợp có `Focus` để bật/tắt read-only (`editor/editor.go:325,344-346`); filter có focus thắng. Màn hình chào vẫn quảng cáo tổ hợp này (`ui/welcome.go:297`). *(Tự kiểm chứng.)*
5. **Tìm kiếm không dùng được bằng bàn phím.** Search bar chỉ nghe Escape (`editor/search.go:85-103`); nhảy match trước/sau chỉ bấm chuột (`:108-113`) dù đã có sẵn `moveToPrevMatch`/`moveToNextMatch`. *(Tự kiểm chứng.)*
6. **Nút đóng tab gần như vô hình**: `iconAlpha = uint8(1)` khi không hover, không tooltip (`ui/navpanel/tabbar.go:222-232`); màu chữ tab đang chọn bị comment (`:201-203`) nên tab active trông y như tab thường. *(Tự kiểm chứng.)*
7. **Danh sách không điều hướng được bằng bàn phím, không có focus ring.** `widgets/label.go:20-94` (`InteractiveLabel`) chỉ nghe chuột — mà đây là widget dùng cho file tree, outline, dropdown, popup, danh sách phiên chat, danh sách nhánh git. File tree có `key.FocusFilter` nhưng chỉ nghe Ctrl+C/V/X (`widgets/filetree/tree.go:338-341`).
8. **Lỗi bị nuốt, chỉ `log.Println`**: `ui/navpanel/filetree.go:103,112,137,314,329`; `ui/editors/typst_view.go:197,482,514`; `ui/assistant/chat.go:83,149,178,194,207,308`; `ui/settings/fonts.go:159,164,174,179,191`; `ui/dialog/create_project.go:137`; `ui/navpanel/menu_panel.go:187-189,218-221`; `ui/welcome.go:236-238`.
9. **Font UI không phủ dấu tiếng Việt.** `fonts/fonts.go` nhúng Hack, NotoSansMath, NotoEmoji, RobotoMono; trong đó **chỉ RobotoMono** phủ Latin Extended Additional (ạ ả ấ ầ ộ ữ…). Face mặc định của material là Go font, typeface mặc định của editor bắt đầu bằng Hack (`service/settings/models.go:427`) — cả hai đều không phủ. `theme.NewTheme("", fonts.Embedded, false)` (`ui/ui.go:194`) để `noSystemFonts=false` nên hiện dấu đang **dựa vào font hệ thống**; trên Linux/container tối giản có nguy cơ tofu hoặc lệch baseline.
10. **Bộ gõ Telex đã viết xong nhưng chưa nối.** `utils/vietnamese/telex.go` (`TransformWordTelex`, kèm `telex_test.go`) **không có caller nào trong code chạy**. *(Tự kiểm chứng: chỉ chính nó và test tham chiếu.)*

**Mức TRUNG**

11. **Thông báo không phân biệt mức độ**: icon luôn tô `th.Fg` bất kể info/warn/error (`ui/statusbar/statusbar.go:94-106`).
12. **Thông báo thứ hai bị cắt ngắn hoặc mất**: `lastUpdateTime` không reset khi gán message mới (`:215` so với `:78-86`). *(Tôi đã loại bỏ nghi vấn "thông báo dính vĩnh viễn" — dòng `:212-214` đã ép `Duration<=0` thành 5 giây.)*
13. **Tooltip có sẵn nhưng gần như không dùng.** `widgets/tipbutton.go:159` chỉ được dùng trong `ui/navpanel/menu_panel.go`. Header editor còn có field `Name` khai báo mà không dùng và một dòng `//tip wg.TipArea` bị comment (`ui/editors/editor_header.go:28-31,127-163`).
14. **Chuỗi tiếng Anh không qua i18n**: nhãn `material.Switch` (`ui/settings/lsp.go:104,117,130,143`; `ui/settings/subviews.go:291,646,659,672,685`; `ui/settings/agent.go:229`), tiêu đề view/tab (`ui/assistant/chat.go:69`, `ui/pkgmgmt/manage.go:59`, `ui/editors/typst_view.go:97`, `ui/editors/generic_text_view.go:38`, `ui/viewer/image_view.go:36`), `ui/crash_report.go:75,98`, `ui/pkgmgmt/card.go:250,257,277`, `ui/assistant/sessions.go:136`.
15. **Không có token spacing/màu.** ~390 literal `unit.Dp(n)` rải rác; riêng "khoảng cách giữa section" trong Settings dùng 12/16/20/24/32 (`ui/settings/sync.go:77,146`, `vpssync.go:162`, `tpix.go:147,156`). Màu lỗi có 3 biến thể hardcode, màu success 2 biến thể — không khớp 9 palette (có palette tối). Palette của gioview chỉ có 6 màu + 2 alpha (`Fg/Bg/ContrastFg/ContrastBg/Bg2/HoverAlpha/SelectedAlpha`), **không có token semantic**, nên màu semantic phải tự suy ra trong `ui/palette`.
16. **Thiếu shortcut thông dụng**: đóng tab (Ctrl+W đang dùng cho wrap-line, `editor/editor.go:326`), chuyển tab (`ui/navpanel/tabbar.go` không có `key.Filter` nào), command palette, Ctrl+G, Ctrl+N, Ctrl+O, Ctrl+,, phím tăng/giảm cỡ chữ UI.
17. **`"tabIdx": 4` hardcode** ở `ui/home.go:138` để mở tab TPIX, nhưng theo thứ tự thực tế (`ui/settings/view_setting.go:76-87`: General 0, Editor 1, Typst 2, LSP 3, Fonts 4, Sync 5, TPIX 6, Agent 7, VPS Sync 8, Help 9) thì vị trí 4 là **Fonts** → bấm vùng tài khoản mở sai tab. *(Tự kiểm chứng.)*
18. **Code chết lệch chuẩn, panic tiềm ẩn**: `ui/windowview.go:81-98` (`LoadTheme`) là bản sao song song của `ui/ui.go:192-212` nhưng **thiếu `th.Face`** và **thiếu `Register("codeColorScheme")`** → nếu hồi sinh `WidgetView` để render editor thì `editor/editor.go:229` (`th.Get("codeColorScheme")`) sẽ panic. Hiện không có caller.
19. **Thiếu control nhỏ**: `WrapLine` có trong settings (`service/settings/models.go:78`) nhưng không có UI bật/tắt; `eye.svg`/`view.svg` có file trong `widgets/icons/lucide/` nhưng không được embed; titlebar bị comment (`ui/home.go:151-157`, `ui/ui.go:120`) vì đã dùng native decoration.

## Các đợt triển khai

Mỗi đợt là một commit riêng, build và chạy được độc lập. Antigravity làm theo thứ tự, báo lại sau mỗi đợt, **không gộp đợt**.

### Đợt 1 — Sửa những thứ đang hoạt động sai (rủi ro thấp nhất, giá trị cao nhất)

1. **Giải xung đột Ctrl+L.** Giữ Ctrl+L = mở AI Assistant (đúng như màn hình chào quảng cáo), chuyển toggle read-only sang `Ctrl+Shift+R`: sửa `editor/editor.go:325` (key filter) và `:344-346` (xử lý); cập nhật phần liệt kê shortcut `ui/welcome.go:287-298`.
2. **Bàn phím cho tìm kiếm.** `editor/search.go:85-113`: Enter → `moveToNextMatch()`, Shift+Enter → `moveToPrevMatch()`, F3/Shift+F3 tương tự (hai hàm đã có sẵn, chỉ nối dây). Bỏ comment hai filter Escape có `Focus` ở `:88-89` và bỏ filter Escape global `:90` để không tranh Escape với modal dialog.
3. **Sửa tab Settings mở sai.** Tạo `ui/settings/tabs.go` khai báo hằng số cho cả 10 tab (chuyển `VPSSyncTabIdx` từ `ui/settings/vpssync.go` về đây), rồi dùng hằng số ở `ui/home.go:138` và `ui/navpanel/menu_panel.go:243`.
4. **Tab bar nhìn được.** `ui/navpanel/tabbar.go`: nâng `iconAlpha` lúc không hover từ `1` lên ~`0x60` (`:222`), thêm tooltip "Close tab" bằng `wg.TipIconButton` + `TipArea` (`widgets/tipbutton.go:159`), bỏ comment màu chữ tab đang chọn (`:201-203`).
5. **Thông báo thứ hai không bị mất.** `ui/statusbar/statusbar.go:215`: reset `lastUpdateTime = time.Time{}` khi gán `lastMessage` mới.

Kiểm chứng: build; chạy app, thử Ctrl+L khi con trỏ trong editor, Enter/Shift+Enter trong search, bấm vùng tài khoản phải mở tab TPIX, thấy nút X trên tab khi không hover, kích hoạt liên tiếp 2 thông báo và xác nhận cả hai hiện đủ.

### Đợt 2 — Việt hoá (i18n) — đóng mục A3 của ROADMAP

1. **Sửa lệch mã locale** (phát hiện 3): thống nhất một dạng ID. Đề xuất giữ ID chữ thường trong `i18n.Locales` và chuẩn hoá đầu vào bằng `strings.ToLower` trong `i18n.Get` (bản web đã phải tự bù việc này ở `server/i18n_api.go:37-39`), đồng thời sửa mặc định `service/settings/models.go:418` cho khớp. `ui/ui.go:172,185` phải **xử lý** lỗi `SetLocale` (fallback + log) thay vì bỏ.
2. **Chuẩn hoá chuỗi nguồn về tiếng Anh.** gotext dùng `-srclang=en-US` nên chuỗi trong code phải là tiếng Anh, bản tiếng Việt nằm ở catalog. Đổi chuỗi tiếng Việt hardcode trong `ui/settings/sync.go` (13 chỗ), `ui/remoteproject/view.go` (8 chỗ), `ui/settings/vpssync.go` sang tiếng Anh. Bỏ nối chuỗi quanh `Translate` (`ui/settings/sync.go:118,253`, `ui/dialog/export.go:109`) → dùng tham số `%s`.
3. **Bọc các chuỗi còn sót vào `i18n.Translate`** theo danh sách ở phát hiện 14.
4. **Thêm locale tiếng Việt.** Thêm `vi-VN` vào directive `i18n/translations/translations.go:14`; thêm entry vào `i18n.Locales` (`ID: "vi-vn"`, `Name: "Tiếng Việt"` — radio trong Settings tự sinh từ danh sách này). Chạy đúng quy trình 5 bước ghi trong `i18n/translations/translations.go` (cần `go install golang.org/x/text/cmd/gotext@latest`), dịch `messages.gotext.json` của `vi-VN`, generate lại `catalog.go`.
5. **Font có dấu cho UI** (phát hiện 9): nhúng thêm một sans phủ đủ tiếng Việt (ví dụ Noto Sans hoặc Roboto) vào `fonts/` + `fonts/fonts.go`, và đặt nó trước Hack trong `TypeFace` mặc định (`service/settings/models.go:427`) / `th.Face`. Kiểm bằng mắt một chuỗi đủ dấu: "Dương Sinh — tệp đã lưu, ộ ữ ẩ ị ợ".
6. *(tuỳ chọn, sau khi có catalog)* `web/src/lib/vi.ts` hiện chỉ là stub 4 chuỗi và tự ghi "The desktop app's Go catalog has no Vietnamese"; khi đã có `vi-VN` thì bản web có thể lấy tiếng Việt từ `/api/i18n` thay vì hardcode client-side.

Kiểm chứng: chạy app, đổi ngôn ngữ sang Tiếng Việt trong Settings → General, xác nhận giao diện đổi và radio hiển thị đúng lựa chọn ngay từ lần mở đầu; đổi sang English và xác nhận **không còn** chuỗi tiếng Việt lọt ra; chữ có dấu hiển thị đủ nét, không tofu.

### Đợt 3 — Token thị giác + theme thương hiệu

1. **Token.** Tạo một nơi duy nhất (đề xuất `ui/uitokens/tokens.go`, hoặc mở rộng `ui/palette` để gom cùng màu): spacing 4/8/12/16/24/32 dp, radius 2/4/8 dp, và màu semantic `Error`/`Warning`/`Success`/`Info`. Lưu ý palette của gioview chỉ có 6 màu + 2 alpha nên **màu semantic phải khai báo theo từng palette** trong `ui/palette/palette.go` (mỗi entry `UIPalette` thêm các màu semantic) để theme tối vẫn đọc được.
2. **Theme thương hiệu Dương Sinh** (chủ dự án đã chốt): thêm 2 entry `Dương Sinh Light` và `Dương Sinh Dark` vào `themeMap` (`ui/palette/palette.go:17-137`). Lấy đúng mã màu đã định nghĩa ở `chessbook/lib/theme.typ:8-14` để desktop, web và tài liệu cờ cùng một nhận diện: navy `#2B3990` (chủ đạo), `#1F2A6E` (navy đậm, tiêu đề), `#5A67A8` (navy nhạt, chữ phụ), gold `#C9A227` (nhấn), `#FBF6E6` (nền gold nhạt). Đặt bản sáng làm mặc định (`service/settings/models.go:420`). `ThemeNames()` (`:139-161`) đang ép `"Default Light"` lên đầu — cập nhật cho theme mặc định mới lên đầu.
3. **Áp dụng có giới hạn** (không sửa cả ~390 chỗ): thay màu lỗi/success hardcode tại `ui/dialog/dialog.go:176`, `ui/dialog/bibliography.go:205,318`, `ui/dialog/publish_pkg.go:182`, `ui/settings/update_check.go:134`, `editor/statusbar.go:89`, `ui/settings/agent.go:184,370,374,406,410`, `ui/navpanel/menu_panel.go:42-43`; tô màu thông báo theo mức độ (`ui/statusbar/statusbar.go:94-106`); đồng nhất khoảng cách trong `ui/settings/*`.

Kiểm chứng: chạy app, lần lượt chọn theme sáng, theme tối và theme Dương Sinh; xác nhận chữ lỗi/thành công đọc được ở cả ba; các tab Settings cùng một nhịp khoảng cách.

### Đợt 4 — Tooltip và lỗi nhìn thấy được

1. **Tooltip cho nút icon.** `ui/editors/editor_header.go`: bỏ comment `//tip wg.TipArea` (`:30`), dùng `action.Name` (đang khai báo mà không dùng) qua `i18n.Translate` với `wg.TipIconButton` trong `layoutActions` (`:127-163`) — áp cho 4 action ở `ui/editors/typst_view.go:211-252`. Thêm tooltip cho nút console/chat (`ui/statusbar/statusbar.go:170-186`) và nút refresh preview (`ui/viewer/preview_op.go:113,142`).
2. **Lỗi phải hiện cho người dùng.** Bổ sung emit `statusbar.Notification{Level: 2, Content: ...}` qua `bus.TopicStatusbarNotifyEvent` (mẫu sẵn ở `ui/navpanel/history.go:27`) tại toàn bộ vị trí ở phát hiện 8. Giữ nguyên `log` để còn dấu vết điều tra.
3. **Dọn phần lỗi preview chết**: `ui/preview/previewer.go:17` khai báo `err` nhưng không bao giờ gán/hiển thị — gán và render, hoặc xoá field.
4. **Empty state cho outline** (`ui/navpanel/outline.go:80-91`) thay vì khung trắng.
5. **Dọn code chết, rủi ro panic** (phát hiện 18): xoá `LoadTheme`/`WidgetView` không dùng trong `ui/windowview.go`, hoặc nếu giữ thì bổ sung `th.Face` và `Register("codeColorScheme", ...)` cho khớp `ui/ui.go:192-212`.

Kiểm chứng: chạy app, cố tình gây lỗi (mở file đã xoá, cấu hình sai đường dẫn tinymist, bấm Sync Now khi chưa cấu hình) và xác nhận statusbar hiện lỗi; hover từng nút icon thấy tooltip.

### Đợt 5 — Bàn phím cho danh sách (rủi ro cao hơn, làm riêng)

`InteractiveLabel` (`widgets/label.go:20-94`) dùng ở rất nhiều nơi nên sửa nó ảnh hưởng toàn cục. Cách an toàn: thêm focus + xử lý ↑/↓/Enter + vẽ focus ring, nhưng **mặc định tắt**, bật qua một trường tuỳ chọn; bật trước ở outline (`ui/navpanel/outline.go`), kiểm thử, rồi mở rộng sang session list, dropdown, popup, danh sách nhánh git. Sau đó mới thêm điều hướng cây cho file tree (↑/↓/←/→/Enter/Delete tại `widgets/filetree/tree.go:338-341`). Mẫu điều hướng bàn phím đã có ở `widgets/menu/context_menu.go:260-288` — bám theo cho nhất quán.

Kiểm chứng: dùng Tab/↑/↓/Enter đi trong outline và file tree không cần chuột; xác nhận chuột vẫn hoạt động y như trước ở **mọi** nơi dùng `InteractiveLabel`.

### Đợt 6 — Hiệu suất làm việc và dọn layout

1. **Shortcut còn thiếu**: Ctrl+W đóng tab (chuyển wrap-line ở `editor/editor.go:326` sang `Ctrl+Alt+W`), Ctrl+Tab và Ctrl+1..9 chuyển tab (đăng ký trong `ui/navpanel/tabbar.go`, dùng `vm.SwitchTab`), Ctrl+, mở Settings, Ctrl+G nhảy tới dòng, Ctrl+`+`/`-` đổi cỡ chữ UI.
2. **Command palette**: dựng bằng `widgets/popup.go` + `InteractiveLabel` (sau đợt 5 đã có bàn phím), mở bằng Ctrl+Shift+P, gom các hành động đã tồn tại (open folder, new project, settings, export, toggle preview/chat/console, package center). Lưu ý `ui/palette/` là **color theme**, không phải command palette — đặt tên package mới cho rõ, ví dụ `ui/commandbar`.
3. **Bổ sung control thiếu**: công tắc `WrapLine` trong tab Editor (`ui/settings/subviews.go`).
4. **Dọn layout**: xoá hoặc bật lại code titlebar đang comment (`ui/home.go:151-157`, `ui/ui.go:120`) — chọn một; embed hoặc xoá `eye.svg`/`view.svg`; đưa `ui/remoteproject/view.go` và `ui/pkgmgmt` về đúng convention chung (`page.PageStyle` + dialog framework).

Kiểm chứng: thử từng shortcut mới; mở command palette chạy vài lệnh; xác nhận các view đã chỉnh vẫn hoạt động.

### Đợt 7 — Nối bộ gõ Telex (chủ dự án đã chốt làm, để sau cùng)

`utils/vietnamese/telex.go` đã có `TransformWordTelex(word, keyChar) (string, bool)` kèm test nhưng chưa có caller. Nối vào luồng gõ của editor desktop (`editor/editor.go`, lớp gvcode): bắt ký tự vừa nhập, lấy từ đang gõ, gọi `TransformWordTelex`, nếu `ok` thì thay từ đó. Bắt buộc có **công tắc bật/tắt trong Settings → Editor** (mặc định tắt để không làm người dùng bất ngờ) và không được can thiệp khi đang ở chế độ read-only hoặc khi đang gõ trong ô tìm kiếm/dialog.

Rủi ro: can thiệp trực tiếp vào luồng nhập liệu, dễ gây lỗi khó chịu (nuốt phím, sai vị trí con trỏ, xung đột với IME hệ thống của Windows). Vì vậy để sau cùng, làm commit riêng, và kiểm thử kỹ: gõ `tieengs Vieejt` → "tiếng Việt", gõ tiếng Anh bình thường không bị biến dạng, undo/redo đúng, bật/tắt công tắc có hiệu lực ngay.

## Cạm bẫy bắt buộc tuân thủ (từ `docs/MEMORY.md`, `docs/ARCHITECTURE.md`)

- `bus.Subscribe` **panic nếu trùng khoá `%p:name`** — thêm subscriber mới (theme, settings, notification) phải đặt tên khoá riêng. Theme đang dùng `"ui.onSettingsChanged"` (`ui/ui.go:183`).
- Callback chạy trên goroutine LSP/agent, đụng UI thì **phải gọi `RefreshWindow()`** nếu không màn hình không vẽ lại.
- Webview native phải huỷ trễ một frame (`destroyPending`) vì chạy trong frame Gio.
- **Không kéo phụ thuộc Gio vào `service/`**; giá trị cấu hình đặt ở `service/settings`, phần vẽ đặt ở `ui/`/`widgets/`.
- Icon phải khởi tạo ở **package level** (`var xIcon = icons.NewSvgIcon(icons.X)`), không gọi `NewSvgIcon` trong hàm `Layout` (parse lỗi sẽ panic và parse lại mỗi frame).

## Kiểm chứng chung

- Build: `go build $(go list ./... | grep -v /scratch)` — `go build ./...` **lỗi sẵn** vì thư mục `scratch/` có nhiều `func main`, không liên quan thay đổi UI.
- Test: `go test ./editor/... ./service/... ./server/... ./utils/...`.
- Chạy thật: `go run .` (cần `typst` và `tinymist`), kiểm tra bằng mắt theo checklist từng đợt.
- Mỗi đợt một commit, message tiếng Anh dạng `fix(ui): ...` / `feat(ui): ...`, kèm dòng `Co-Authored-By`.

## Tuân thủ quy trình repo

Theo `docs/ROADMAP.md:34-38`: viết plan ngắn vào `docs/plans/plan_desktop_ui_improvements.md`, cập nhật tài liệu liên quan, thêm mục vào `docs/HISTORY.md`, và **gạch mục A3** (cùng A4 — sửa `docs/architecture_and_guide.md:28` đang nói sai rằng i18n đã có tiếng Việt) khỏi `docs/ROADMAP.md` khi Đợt 2 xong. Bổ sung `ui/palette/palette.go` vào bảng rebrand ở `docs/REPLICATION.md:60-61` (hiện chỉ liệt kê theme của bản web).

## Phân công

- **Claude Code (tôi)**: đã rà soát, lập kế hoạch; review từng đợt Antigravity báo về.
- **Antigravity**: triển khai Đợt 1 → 7 theo thứ tự, mỗi đợt một commit, báo kết quả kèm bằng chứng build/chạy sau mỗi đợt.
- **Chủ dự án**: duyệt bản dịch tiếng Việt (Đợt 2), duyệt cảm quan theme Dương Sinh (Đợt 3), nghiệm thu bộ gõ Telex (Đợt 7).

