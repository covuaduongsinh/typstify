# API của Typstify Server

Nguồn sự thật: `server/server.go` (`routes()`). Server chỉ là lớp vận chuyển, ủy quyền cho `service.ServiceFacade`.
Mọi route yêu cầu đăng nhập (`s.handle` → `auth.require`) trừ nhóm "Không cần xác thực". Xác thực bằng cookie `typstify_session` hoặc header `Authorization: Bearer <token>`.

Lỗi trả về JSON `{"error": "..."}`. Riêng render preview trả HTTP 200 với `ok:false` khi Typst lỗi biên dịch.

## Không cần xác thực

| Method | Path | Mục đích |
|---|---|---|
| GET | `/api/health` | Kiểm tra sống, trả `{ok:true}` |
| POST | `/api/auth/register` | Đăng ký |
| POST | `/api/auth/login` | Đăng nhập, đặt cookie |
| POST | `/api/auth/logout` | Đăng xuất |
| GET | `/api/auth/status` | Trạng thái xác thực |
| POST | `/api/auth/token` | Cấp Bearer token dài hạn (username, password, label) |
| POST | `/api/i18n` | Dịch chuỗi |

## Xác thực và token

| Method | Path | Mục đích |
|---|---|---|
| POST | `/api/auth/change-password` | Đổi mật khẩu |
| GET | `/api/auth/tokens` | Liệt kê token |
| DELETE | `/api/auth/tokens/{hash}` | Thu hồi token (lưu dạng SHA-256) |
| POST | `/api/auth/mfa/setup` | Sinh khoá TOTP tạm, trả `{secret, otpauthUri}` |
| POST | `/api/auth/mfa/enable` | `{code, password}`: bật MFA, trả 10 `recoveryCodes` (chỉ hiện một lần), đăng xuất các phiên khác |
| POST | `/api/auth/mfa/disable` | `{password, code}`: tắt MFA (có giới hạn tần suất), đăng xuất các phiên khác |

`POST /api/auth/login` và `POST /api/auth/token` nhận thêm `code` (mã TOTP hoặc mã khôi phục). Tài khoản đã bật MFA mà thiếu hoặc sai mã thì nhận 401 kèm `{"mfaRequired": true}`. `GET /api/auth/status` có thêm `admin`, `mfaEnabled`.

## Project, thành viên, chia sẻ (`server/projects_api.go`)

Vai trò: `owner` > `editor` (sửa) > `commenter` (góp ý) > `viewer` (chỉ xem). Admin được coi là owner của mọi project. Logic nằm ở `service/projects`, dữ liệu lưu trong `projects.json` cạnh `auth_users.json`.

| Method | Path | Quyền | Mục đích |
|---|---|---|---|
| GET | `/api/projects` | đăng nhập | Project mình có quyền, kèm `role` |
| POST | `/api/projects` | đăng nhập | `{name}`: tạo thư mục dưới `-project-root` (hoặc `<RootDir>/projects`), tạo `main.typ`, người tạo là owner |
| POST | `/api/projects/join` | đăng nhập | `{token}`: tham gia bằng link chia sẻ; không hạ vai trò đang có |
| POST | `/api/projects/{id}/open` | ≥ viewer | Đặt làm project hiện hành |
| DELETE | `/api/projects/{id}` | owner | Bỏ đăng ký (giữ nguyên file) |
| GET | `/api/projects/{id}/members` | owner | Danh sách thành viên |
| PUT / DELETE | `/api/projects/{id}/members/{user}` | owner | `{role}` đặt vai trò / xoá thành viên (không áp dụng cho owner) |
| GET / POST | `/api/projects/{id}/shares` | owner | Liệt kê / tạo link `{role, expiresInDays}` (0 = không hết hạn); token chỉ trả một lần, server chỉ lưu SHA-256 |
| DELETE | `/api/projects/{id}/shares/{shareId}` | owner | Thu hồi link |
| GET | `/api/workspace/role` | đăng nhập | Vai trò trên project hiện hành (`registered: false` nếu thư mục chưa đăng ký) |

Link chia sẻ có dạng `https://<máy chủ>/?join=<token>`; web tự tham gia sau khi đăng nhập.

## Workspace / file

| Method | Path | Mục đích |
|---|---|---|
| GET | `/api/workspace/current` | Project hiện tại |
| GET | `/api/workspace/recent` | Project gần đây |
| POST | `/api/workspace/open` | Mở project |
| POST | `/api/workspace/create` | Tạo project |
| GET | `/api/workspace/tree` | Cây file |
| GET / PUT / POST / DELETE | `/api/workspace/file` | Đọc / ghi / tạo (`{path,isDir}`) / xoá |
| POST | `/api/workspace/rename` | Đổi tên |

Đường dẫn được chặn thoát khỏi root bởi `resolveInRoot` (`server/paths.go`); `.git` và `.typstify` được bảo vệ (`isProtectedPath`, `server/files_api.go`).

**Phân quyền theo project hiện hành** (`handleRole`): khi thư mục đang mở đã đăng ký trong `projects.json`, thì:
- đọc (`tree`, `GET file`, export, preview, `GET /api/sync/*`, `/ws/lsp`) cần ≥ viewer;
- ghi (`PUT/POST/DELETE file`, `rename`, `POST /api/sync/*`, `POST /api/dropbox/sync`, `/ws/agent`) cần ≥ editor;
- không đủ quyền thì trả 403.

Các quy tắc kèm theo:
- Thư mục con của project đã đăng ký vẫn tính là project đó (`Covering`).
- Mở hoặc tạo theo đường dẫn một thư mục **chứa** project của người khác: chỉ admin được làm (`ContainsAny`).
- Preview tinymist (`/preview/` và WebSocket ở `/`) cần ≥ viewer.
- Trên `/ws/lsp`, người dưới editor không gửi được `didChange`/`didSave`; `didOpen` chỉ nạp nội dung trên đĩa. Quyền được kiểm lại ở mỗi tin nhắn.
- Phản hồi về project không chứa đường dẫn máy chủ hay danh sách thành viên.

Thư mục chưa đăng ký giữ hành vi cũ (mọi người đã đăng nhập). `POST /api/workspace/create` chỉ đăng ký thư mục do chính request đó tạo ra (không chiếm thư mục có sẵn); tự đăng ký thư mục mới với owner là người tạo.

**Chỉ admin** (`handleAdmin`): `PUT /api/settings/*`, upload/xoá font, `POST /api/dropbox/auth/*`, `POST /api/agent/select|auth/*|preferred-config`, `GET /api/console`.

## Export và preview

| Method | Path | Mục đích |
|---|---|---|
| GET | `/api/export` | Biên dịch và tải về. Query: `path`, `format=pdf\|png\|svg\|html`, `pages`, `ppi` (mặc định 144), `filename`, `pdfVersion`, `pdfStandard`, `noPdfTags=1`. Nhiều trang PNG/SVG trả zip |
| GET | `/api/preview/pdf` | PDF inline cho iframe/embed |
| POST | `/api/preview/render` | SVG từng trang: body `{path, content, format}` → `{ok, pages[], pageCount, error}` |
| POST | `/api/preview/anchors` | Số trang vật lý của từng heading, dùng cho scroll sync |
| GET | `/api/preview/status` | tinymist sẵn sàng chưa |
| POST | `/api/preview/restart` | Khởi động lại preview, body `{entryFile}` |
| POST | `/api/preview/cursor` | Chuyển vị trí con trỏ cho tinymist |
| ANY | `/preview/` (prefix) | Reverse proxy tới preview server của tinymist (chèn `<base href>`, hỗ trợ nâng cấp WebSocket) |

## Trình biên dịch trong trình duyệt (`server/wasm_api.go`)

Cần đăng nhập, không phụ thuộc project. Logic ở `service/wasmassets`.

| Method | Path | Mục đích |
|---|---|---|
| GET | `/api/wasm/fonts` | Danh sách font cho compiler WASM: `[{id, name, size, version, url}]`. Nguồn: thư mục font tải lên, `TYPSTIFY_WASM_FONT_DIRS`, và font hệ thống khớp `TYPSTIFY_WASM_SYSTEM_FONTS` (mặc định `Roboto-*,NotoSerif-*,NotoSans-*,NotoSansSymbols2-*`; bỏ qua khi Settings tắt font hệ thống). Không lộ đường dẫn: `id` là hash |
| GET | `/api/wasm/fonts/{id}` | Bytes của font; cache `immutable` (URL có `?v=` theo size+mtime) |
| GET | `/api/wasm/packages/{namespace}/{name}/{version}` | Package dạng `.tar.gz`. Tìm ở thư mục package (Settings/`TYPST_PACKAGE_PATH`), thư mục data của typst, cache package; `@preview` thiếu thì tải từ packages.typst.org vào cache. `@preview` cache vĩnh viễn, `@local` dùng ETag. Tên sai định dạng → 400, không có → 404 |

Worker còn đọc file project qua `/api/sync/manifest` và `/api/sync/pull` (vai trò viewer). `/api/sync/pull` trả kèm header `X-Content-Sha256` (cùng thuật toán với manifest) để client chỉ cache khi nội dung khớp.

## Package (Tpix)

| Method | Path | Mục đích |
|---|---|---|
| GET | `/api/packages/search` | Tìm package |
| GET | `/api/packages/cached` | Package đã cache |
| GET | `/api/packages/detail` | Chi tiết, danh sách phiên bản |
| POST | `/api/packages/download` | Tải một phiên bản |
| POST | `/api/packages/pull-deps` | Kéo phụ thuộc của project |

## Settings và font

| Method | Path | Mục đích |
|---|---|---|
| GET / PUT | `/api/settings/{general,editor,typst,lsp,agent}` | Đọc/ghi từng section |
| GET | `/api/settings/meta` | Timestamp ghi gần nhất từng section (phục vụ sync) |
| GET | `/api/settings/fonts` | Liệt kê font |
| POST | `/api/settings/fonts?filename=` | Upload font (body nhị phân) |
| DELETE | `/api/settings/fonts/{name}` | Xoá font |

`/api/settings/tpix` cố ý **không** mở ra (chứa API key registry). Section `agent` có route nhưng bị loại khỏi vòng sync vì gắn với máy.

Font: chỉ `.ttf`, `.otf`, `.ttc`; tối đa 30 MiB; tên khớp `^[\p{L}\p{N}_.\- ]+$` (`service/fonts/fonts.go`).

## Dropbox

| Method | Path | Mục đích |
|---|---|---|
| GET | `/api/dropbox/status` | Trạng thái kết nối |
| GET | `/api/dropbox/projects` | Danh sách project trên Dropbox |
| POST | `/api/dropbox/auth/url`, `/auth/callback`, `/auth/token`, `/auth/disconnect` | Luồng xác thực |
| POST | `/api/dropbox/sync` | Đồng bộ (`two-way`, `push`, `pull`) |
| POST | `/api/dropbox/import` | Nhập project |

## Agent AI (ACP)

| Method | Path | Mục đích |
|---|---|---|
| GET | `/api/agent/registry` | Registry agent (cache) |
| POST | `/api/agent/select` | Chọn agent |
| POST | `/api/agent/auth/{methodId}` | Đăng nhập agent (chặn tới khi xong) |
| POST | `/api/agent/auth/callback` | Chuyển tiếp OAuth callback |
| POST | `/api/agent/preferred-config` | Lưu lựa chọn ưa thích (model, mode) |
| GET | `/api/agent/sessions` | Lịch sử phiên `{sessionId,title,updatedAt}` |
| GET | `/api/console` | Stderr/console của agent |

## WebSocket

### `/ws/agent?sessionId=&resume=1`
Giới hạn đọc 20 MiB (ảnh dán). Không có `sessionId` → phiên mới; có `sessionId` → load (phát lại lịch sử) hoặc `resume=1` (không phát lại).

- Client → server: `prompt{text,images}`, `setConfigOption`, `cancel`, `permissionResponse{optionId}`, `retryAuth`.
- Server → client: `ready{sessionId}`, `configOptions`, `userMessage`, `agentMessage`, `agentThought`, `toolCall`, `toolCallUpdate`, `plan`, `permission` (tối đa một yêu cầu treo mỗi kết nối), `turnEnd`, `error`, `authRequired{agentName, authMethods}`.
- Payload `data` là kiểu ACP thô (`web/src/lib/acpTypes.ts`).

### `/ws/collab?path=&project=`
Cộng tác thời gian thực trên một tệp (`server/collab_ws.go`, `service/collab`). Cần vai trò viewer. `project` phải trùng project hiện hành của máy chủ, nếu không thì trả 409. Tin nhắn nhị phân `[loại][kênh][payload]`:

| Loại | Hướng | Nội dung |
|---|---|---|
| 0 update | hai chiều | Update Yjs. Kênh 0 là văn bản (cần editor), kênh 1 là bình luận (cần commenter) |
| 1 awareness | hai chiều | Con trỏ, tên. Chỉ chuyển tiếp |
| 2 control | server → client | JSON `hello` / `peers` / `leader` / `compact` / `external {base, text}` / `error` |
| 3 synced | server → client | Đã phát lại hết log |
| 4 compact | leader → server | Trạng thái đầy đủ thay cho log |
| 5 save | leader → server | 64 ký tự hash của bản đĩa leader đã thấy, rồi văn bản (cùng quy tắc với `PUT /api/workspace/file`). Máy chủ không ghi nếu tệp đã đổi sau bản đó, mà báo `external` |

Vai trò được kiểm tra lại ở mỗi tin nhắn; người bị thu hồi quyền bị ngắt kết nối (mã 1008, không nối lại). Mã 4000 là ngắt tạm thời (client chậm), client tự nối lại. Phần phát lại log khi vào phòng được ghi thẳng ra socket, không qua hàng đợi. Trong lúc tệp có phòng mở, `PUT /api/workspace/file` cho tệp đó trả 409.

### `/ws/lsp`
Cầu nối LSP tới tinymist (giới hạn đọc 20 MiB, `server/lsp_ws.go`). Client `web/src/lib/lspClient.ts` tự nối lại với backoff 1–15 giây.

## Xác thực và giới hạn

- Người dùng lưu ở `auth_users.json`, phiên ở `auth_sessions.json`; username khớp `^[a-zA-Z0-9_.-]{3,32}$`.
- Mật khẩu băm argon2id dạng PHC (`server/password.go`); hash SHA-256 cũ tự nâng cấp khi đăng nhập. Người đăng ký đầu tiên là admin; file cũ chưa có admin thì tài khoản tạo sớm nhất thành admin.
- Mật khẩu máy chủ (`-password`) chỉ dùng để đăng nhập khi **chưa có tài khoản nào** (tài khoản `admin` khởi tạo); khi đã có tài khoản, nó chỉ còn là mã cho phép đăng ký.
- Argon2 chạy ngoài khoá `authManager.mu`. Tên tài khoản không tồn tại vẫn tốn cùng thời gian băm như tài khoản thật (`server/credentials.go`).
- TOTP theo RFC 6238 (SHA-1, 30 s, 6 số, lệch ±1 bước, không cho dùng lại bước đã đăng nhập; `server/totp.go`).
- Mã khôi phục (`xxxxx-xxxxx`) lưu dạng argon2id, mỗi mã dùng một lần.
- Sai mã MFA 5 lần thì khoá MFA **theo tài khoản** trong 15 phút, độc lập với giới hạn theo IP.
- Phiên: TTL trượt 30 ngày, trần 90 ngày (`sessionTTL`, `sessionMaxAge`, `server/auth.go`).
- Đăng nhập sai bị trễ (`failedLoginDelay`); giới hạn 5 lần / 15 phút / IP, khoá 15 phút, trả 429 (`loginLimiter`, `server/security.go`).
- Body: 2 MiB mặc định, 64 MiB cho PUT file (`withBodyLimit`).
- Biên dịch đồng thời tối đa 2, mỗi lần 90 giây (`maxConcurrentCompiles`, `compileTimeout`, `server/export_api.go`).
- WebSocket kiểm tra Origin cùng nguồn (`sameOrigin`); chỉ tin `X-Forwarded-*` từ peer loopback/riêng tư (`trustedPeer`).
