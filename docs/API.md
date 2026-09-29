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

### `/ws/lsp`
Cầu nối LSP tới tinymist (giới hạn đọc 20 MiB, `server/lsp_ws.go`). Client `web/src/lib/lspClient.ts` tự nối lại với backoff 1–15 giây.

## Xác thực và giới hạn

- Người dùng lưu ở `auth_users.json`, phiên ở `auth_sessions.json`; username khớp `^[a-zA-Z0-9_.-]{3,32}$`.
- Phiên: TTL trượt 30 ngày, trần 90 ngày (`sessionTTL`, `sessionMaxAge`, `server/auth.go`).
- Đăng nhập sai bị trễ (`failedLoginDelay`); giới hạn 5 lần / 15 phút / IP, khoá 15 phút, trả 429 (`loginLimiter`, `server/security.go`).
- Body: 2 MiB mặc định, 64 MiB cho PUT file (`withBodyLimit`).
- Biên dịch đồng thời tối đa 2, mỗi lần 90 giây (`maxConcurrentCompiles`, `compileTimeout`, `server/export_api.go`).
- WebSocket kiểm tra Origin cùng nguồn (`sameOrigin`); chỉ tin `X-Forwarded-*` từ peer loopback/riêng tư (`trustedPeer`).
