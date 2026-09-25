# Chạy Typstify Web (self-host)

Đây là hướng dẫn triển khai **phiên bản web** của Typstify theo mô hình self-hosted, một
người dùng/instance — xem bối cảnh và kiến trúc đầy đủ ở
[`docs/plans/plan_web_version.md`](plans/plan_web_version.md). Bản thân server không public
đa người dùng: mỗi instance chỉ phục vụ một chủ sở hữu, giống mô hình `code-server`/Jupyter.

Tài liệu này **không** thay thế hướng dẫn chạy bản desktop ở [`README.md`](../README.md) gốc.

## Chạy nhanh bằng Docker Compose (khuyến nghị)

```sh
git clone https://github.com/typstify/typstify.git
cd typstify
cp .env.example .env
# sửa .env, đặt TYPSTIFY_SERVER_PASSWORD thành một mật khẩu mạnh
docker compose up -d --build
```

`docker-compose.yml` đã kèm **Caddy** làm reverse proxy có TLS tự động (Let's Encrypt, đặt
`TYPSTIFY_DOMAIN` trong `.env`); container `typstify` chỉ bind vào `127.0.0.1:8080`. Khi deploy
bằng **Dokploy**, Traefik của Dokploy đóng vai trò này. Không public thẳng cổng 8080 ra Internet
mà không có TLS, vì mật khẩu đăng nhập và session cookie sẽ đi ở dạng plaintext.

### Bảo mật có sẵn

- Mọi API và WebSocket (kể cả kênh preview ở `/`) đều yêu cầu đăng nhập; WebSocket kiểm tra
  `Origin` cùng tên miền.
- Sai mật khẩu 5 lần trong 15 phút từ cùng một IP thì IP đó bị khóa 15 phút (HTTP 429).
- Cookie phiên `HttpOnly`, `SameSite=Lax`, bật `Secure` khi truy cập qua HTTPS (kể cả sau proxy
  báo `X-Forwarded-Proto: https`); phiên hết hạn sau 30 ngày không dùng, tối đa 90 ngày.
- Header `X-Forwarded-For/Proto/Host` chỉ được tin khi kết nối đến từ địa chỉ loopback/mạng
  riêng (proxy trong Docker), nên client ngoài Internet không giả mạo được.
- Giới hạn kích thước body (2 MiB; lưu file 64 MiB, vượt quá trả 413 thay vì cắt file), timeout
  đọc header, tối đa 2 lượt biên dịch PDF cùng lúc, mỗi lượt tối đa 90 giây.
- Đường dẫn file bị chặn khi thoát khỏi thư mục dự án, kể cả qua symlink; không xóa/đổi tên
  được `.git`, `.typstify`; đổi tên không ghi đè file đã có.
- `TYPSTIFY_SERVER_PASSWORD` bị xóa khỏi môi trường của các tiến trình con (AI agent, terminal).
- Người đăng nhập được coi là quản trị viên: trang Cài đặt cho phép cấu hình lệnh chạy AI agent
  và đường dẫn `typst`/`tinymist`, tức là chạy lệnh trên máy chủ. Chỉ cấp mật khẩu cho người
  được phép quản trị máy chủ.

Dữ liệu (project files, cấu hình, cache package Typst) được lưu trong Docker volume
`typstify-data`, mount vào `/data` trong container. Xoá volume này sẽ mất toàn bộ project và
cấu hình.

## Chạy trực tiếp (không Docker)

Cần cài sẵn trên máy:
- `typst` và `tinymist` executable, có trong `PATH` (hoặc đặt cạnh binary `typstify-server`,
  hoặc chỉ định qua setting `externalTypst`/`externalTinymist` — xem `GET/PUT
  /api/settings/general`).
- Node.js + `npx` nếu dùng agent AI mặc định (Claude Code) hoặc bất kỳ agent nào phân phối
  qua `npx` trong [registry ACP](https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json).
  v1 chỉ cam kết hỗ trợ agent dạng `npx` — xem rủi ro #4 trong kế hoạch.

```sh
# build backend
go build -o bin/typstify-server ./cmd/typstify-server

# build frontend
cd web && npm ci && npm run build && cd ..

# chạy
TYPSTIFY_SERVER_PASSWORD=<mật khẩu> \
TYPSTIFY_PROJECT_DIR=/path/to/your/project \
TYPSTIFY_STATIC_DIR=web/dist \
./bin/typstify-server
```

Mặc định lắng nghe ở `:8080` (đổi qua `TYPSTIFY_SERVER_ADDR`, ví dụ `127.0.0.1:9000`).

## Biến môi trường / cờ dòng lệnh

`cmd/typstify-server` nhận cả biến môi trường lẫn cờ tương ứng (cờ ưu tiên hơn nếu đặt cả
hai, trừ `-password` không có biến môi trường đọc mặc định qua flag mà đọc trực tiếp từ
`TYPSTIFY_SERVER_PASSWORD` làm giá trị mặc định của flag):

| Biến môi trường | Cờ | Mặc định | Ý nghĩa |
| --- | --- | --- | --- |
| `TYPSTIFY_SERVER_PASSWORD` | `-password` | *(rỗng — tắt xác thực)* | Mật khẩu đăng nhập server. **Bắt buộc đặt** nếu chạy trên bất kỳ máy nào không phải `localhost` thuần. |
| `TYPSTIFY_SERVER_ADDR` | `-addr` | `:8080` | Địa chỉ/cổng lắng nghe. |
| `TYPSTIFY_PROJECT_DIR` | `-project` | thư mục hiện tại | Project Typst mở khi khởi động. |
| `TYPSTIFY_STATIC_DIR` | `-static-dir` | `web/dist` | Thư mục frontend đã build; nếu không tồn tại, server chỉ phục vụ API (không có UI). |
| `XDG_CONFIG_HOME` | — | theo OS | Nơi lưu `settings.json` (dùng `os.UserConfigDir()`), nên trỏ vào một thư mục ghi được/bền vững trong container. |
| `XDG_CACHE_HOME` | — | theo OS | Nơi cache package Typst đã tải (`os.UserCacheDir()`). |
| `TYPST_PACKAGE_PATH` | — | image: `/opt/typst-packages` | Thư mục gói Typst cục bộ dùng khi Cài đặt → "Thư mục gói cục bộ" để trống. Image đặt sẵn `@local/chessbook` và `@preview/board-n-pieces` ở đây. |
| `TYPSTIFY_PROJECT_ROOT` | `-project-root` | image: `/data` | Chỉ cho mở/tạo dự án bên trong thư mục này (volume bền vững). |

## AI Agent: chọn qua registry + đăng nhập gói thuê bao

Trang Settings của bản web có mục "Agent Registry" (gọi API bên dưới) để chọn agent AI theo
gói thuê bao hàng tháng của chính nhà cung cấp (Claude Pro/Max, ChatGPT Plus/Pro qua Codex...)
thay vì trả tiền theo API. Nguyên tắc cốt lõi: server chỉ **spawn thẳng binary CLI chính chủ** của agent
(qua `npx`/binary như registry ACP mô tả) và không tự đọc/giải mã token OAuth của nó.

**Đăng nhập kiểu "Log in with Google" (OAuth loopback — Antigravity, Gemini CLI…):** agent
chạy trong container trên máy chủ và chờ Google chuyển hướng về `http://127.0.0.1:<cổng>` của
**container**. Trình duyệt lại mở `127.0.0.1` trên **máy người dùng** nên hiện trang lỗi
"không kết nối được" — đó là bình thường. Thẻ đăng nhập trong khung Trợ lý AI hướng dẫn sao chép
toàn bộ địa chỉ trang lỗi đó, dán vào ô **Hoàn tất đăng nhập**; server gửi đúng request ấy tới
agent qua `POST /api/agent/auth/callback`. Endpoint chỉ nhận địa chỉ loopback `http`, đúng cổng
`redirect_uri` trong liên kết đăng nhập agent vừa in ra, có tham số `code`, và chỉ khi đang có
lượt đăng nhập chờ — không thể dùng để gọi dịch vụ nội bộ khác.

| Endpoint | Ý nghĩa |
| --- | --- |
| `GET /api/agent/registry` | Danh sách đầy đủ agent từ [registry ACP chính thức](https://cdn.agentclientprotocol.com/registry/v1/latest/registry.json) (cache — xem `settings.FetchAgentRegistry`). |
| `POST /api/agent/select {"agentId": "..."}` | Chọn một agent từ registry, tự resolve ra `cmd`/`args`/`env` đúng nền tảng server đang chạy (`service/settings.ResolveAgentCommand`) và lưu làm agent hiện tại. |
| `POST /api/agent/auth/{methodId}` | Chạy bước đăng nhập của agent hiện tại (khớp `AuthMethod.id` mà agent tự báo qua ACP `authMethods`) — block tới khi CLI tự hoàn tất OAuth/login của nó. |
| `GET /api/console` | Đọc thô log stderr của agent hiện tại (cùng luồng Console panel của bản desktop) — frontend dùng để tự tìm link đăng nhập agent tự in ra, theo đúng khuôn "pipe + regex". |

Khi phiên `/ws/agent` cần đăng nhập, server gửi `{"type":"authRequired","data":{...}}`; sau
khi người dùng bấm "Sign in" trên `AuthCard` (frontend tự gọi `/api/agent/auth/{methodId}` rồi
poll `/api/console` tìm URL), frontend gửi lại `{"type":"retryAuth"}` để server thử khởi động
phiên lần nữa.

Agent được đóng gói sẵn trong image Docker (không cần cài thêm gì):
- **Claude Code** — phân phối qua `npx`, dùng chung Node.js đã có sẵn trong `node:22-slim`.
  Đã xác nhận chạy được thật (bản desktop, cùng cơ chế `npx`).
- **Codex** (ChatGPT) — cũng phân phối qua `npx` (`@agentclientprotocol/codex-acp`). **Đã xác
  nhận chạy được thật** trong container `node:22-slim` (18/9/2026): cài đặt thành công
  (`npm install`, 20 package, ~23s khi cache CDN đã ấm), binary `codex-acp` khởi động không
  lỗi và chờ JSON-RPC trên stdio đúng như một ACP server bình thường (cùng hành vi với Claude
  Code). Lần thử đầu tiên trông như "treo": gói phụ thuộc optional `@openai/codex-linux-x64`
  (binary CLI Codex nhúng sẵn) mất **~126 giây** để tải lần đầu (cold cache) — dài hơn mọi
  timeout thử nghiệm ban đầu (60-150s), và ở log level mặc định npm không in gì trong lúc tải
  nên trông giống bị treo dù thực chất vẫn đang chạy bình thường. Không phải lỗi mạng (`npm
  view` luôn phản hồi tức thời) cũng không phải lỗi cài đặt.
- **Google Antigravity** (`antigravity-acp`) — binary Linux gốc của Google
  (`agy_acp_server.par`, ~vài MB, không cần `localharness.exe`) được tải và xác minh ngay lúc
  build image (xem stage `antigravity` trong `Dockerfile`). Build sẽ đỏ nếu tải/giải nén lỗi;
  riêng bước "chạy thử xem có khởi động được không" thì **không** làm đỏ build khi binary bị
  `SIGILL` do thiếu tập lệnh CPU AVX — chỉ in cảnh báo to. Đã tái hiện SIGILL này thật (exit
  132) cả lúc build lẫn lúc chạy container thật trên máy dev (VM của Docker Desktop ở đó không
  lộ AVX ra) — AVX phổ biến trên phần cứng x86_64 thật từ 2011 nên nhiều khả năng đây chỉ là
  giới hạn của máy dev, nhưng **chưa xác minh được trên một host Linux/CI thật có AVX**, nên
  coi Antigravity trong Docker là "cài được nhưng chưa chắc chạy được" cho tới khi kiểm tra
  lại trên phần cứng thật.

Agent khác trong registry (dạng `uvx`, hoặc binary nền tảng khác Antigravity) chưa được đóng
gói sẵn — `POST /api/agent/select` vẫn cho chọn nhưng sẽ báo lỗi rõ ràng khi không tìm thấy
lệnh tương ứng trên server.

### Agent mặc định & mức độ ổn định (khuyến nghị)

**Claude Code nên là lựa chọn mặc định** cho bản Web self-host — đã chứng minh ổn định qua toàn
bộ quá trình phát triển tính năng này: không cần AVX, hỗ trợ đóng phiên (`session/close`) qua
ACP, và tự echo lại tin nhắn người dùng đúng chuẩn ACP.

**Google Antigravity dùng được, nhưng ở mức "best-effort", không nên đặt làm mặc định**, vì
nhiều giới hạn đã xác nhận thật (không phải suy đoán) trong quá trình phát triển:
- Bản Linux (dùng trong Docker) yêu cầu CPU có AVX — không chạy được trên CPU không hỗ trợ (ví
  dụ dòng Pentium Gold/Celeron của Intel, vốn bị khoá cứng AVX ở phần cứng). Xem mục AI Agent ở
  trên.
- **Không hỗ trợ `session/close` qua ACP** — Typstify không có cách nào báo cho tiến trình
  Antigravity biết một phiên đã kết thúc (tab đóng, mất mạng, reload) ngoài `session/cancel`
  (đã vá ở `server/agent_ws.go`: luôn gửi cancel khi phiên đang dở dang bị ngắt kết nối, dù agent
  có hỗ trợ đóng phiên hay không) — nhưng bản thân agent phía Google vẫn có thể tích tụ trạng
  thái từ nhiều phiên bị bỏ rơi nếu người dùng ngắt kết nối liên tục trong thời gian ngắn.
- **Không echo lại tin nhắn của người dùng** qua `session/update` (khác Claude Code) — đã vá ở
  client (`AgentChat.tsx` tự vẽ bong bóng tin nhắn ngay khi gửi, không đợi agent xác nhận).
- **Đã từng bị dính lỗi cấu hình nghiêm trọng, đã sửa xong** (19/9/2026): cơ chế lưu settings có
  lỗi logic khiến field `Args` để rỗng có chủ đích (đúng cho Antigravity, không cần tham số) bị
  tự động ghi đè bằng tham số mặc định của Claude Code mỗi lần đọc lại settings — nghĩa là
  Antigravity từng bị khởi động kèm tham số hoàn toàn sai (`-y @agentclientprotocol/claude-agent-acp@...`)
  suốt thời gian dài mà không ai biết. Xem `service/settings/base.go`/`mergeModel` và
  `TestModelSave_PreservesIntentionallyEmptyField` (test tái hiện đúng lỗi này).
- **Tự ý "tìm kiếm file trên toàn bộ ổ đĩa" thay vì dùng thư mục project (`cwd`) đã cho qua
  ACP** — quan sát thật nhiều lần (19/9/2026), kể cả khi prompt nêu rõ đường dẫn tuyệt đối: agent
  tường thuật "đang tìm kiếm... trên các ổ đĩa", đôi khi kết thúc lượt (`end_turn`) mà chưa thực
  sự đọc/sửa file nào. Đây là hành vi nội bộ (đóng mã nguồn) của chính agent Google, không phải
  lỗi trong code Typstify — không có cách sửa từ phía này. Với **câu hỏi đơn giản không cần đọc
  file**, Antigravity phản hồi tốt trong 10-15 giây; với **tác vụ cần sửa file thật**, kết quả
  không ổn định (dao động từ 30 giây tới thất bại hoàn toàn tuỳ lần).

Đây không phải khuyến nghị "đừng dùng Antigravity" — chỉ là đặt đúng kỳ vọng: agent còn khá mới
(beta) từ phía Google, có thể mất 10-15 giây (hoặc hơn, với tác vụ sửa file thật — và đôi khi
không hoàn thành được, xem điểm trên) cho phản hồi đầu tiên. **Cho việc sửa file thật, khuyến
nghị dùng Claude Code.** Giao diện web đã có chỉ báo "Agent is thinking…" và nút "Cancel" để chủ
động huỷ nếu chờ quá lâu (quá 30 giây sẽ tự hiện cảnh báo).

**Cảnh báo điều khoản dịch vụ**: gói thuê bao (Claude Pro/Max, ChatGPT Plus/Pro, Google AI...)
chỉ dành cho dùng cá nhân thông thường qua CLI chính chủ, không phải cho việc chạy nền liên
tục hay dùng chung nhiều người trên một tài khoản — kể cả khi server chạy trên VPS riêng của
bạn. Vì bản web tự host này chỉ có một người dùng và agent chỉ chạy khi bạn chủ động mở chat
(không có cron/queue nào tự gọi agent), rủi ro vi phạm điều khoản thấp, nhưng vẫn nên biết
trước khi để server chạy 24/7 trên máy chủ công cộng.

## Giới hạn đã biết của v1

Xem đầy đủ ở mục "Rủi ro & điểm cần xử lý sớm" trong
[`docs/plans/plan_web_version.md`](plans/plan_web_version.md). Tóm tắt các điểm ảnh hưởng
trực tiếp tới vận hành:

- **Preview không tự cập nhật theo từng phím gõ chưa lưu** — cần `Ctrl+S` để thấy thay đổi.
- **Chỉ đóng gói sẵn trong Docker image: agent phân phối qua `npx`** (Claude Code, Codex — cả
  hai đã xác nhận chạy được thật) và **Google Antigravity** (binary Linux riêng, đã xác nhận
  cài đúng nhưng cần CPU có AVX để chạy, xem mục AI Agent phía trên). Agent dạng `uvx` (Python)
  hoặc binary nền tảng khác chưa được đóng gói sẵn.
- **Lần đầu chọn một agent phân phối qua `npx` (Claude Code, Codex) có thể mất 1-2 phút** để
  `npx` tải gói lần đầu (một số gói kèm binary CLI nhúng sẵn, quan sát thực tế ~126s cho gói
  của Codex) — các lần sau nhanh hơn nhiều nhờ cache npm của container/máy chủ. Đây là hành vi
  bình thường của `npx`, không phải lỗi.
- **`grok-build` (xAI) tồn tại trong registry ACP nhưng chưa xác nhận có đường dùng qua gói
  SuperGrok** — thử nghiệm thực tế cho thấy package hiện không khởi động ổn định qua ACP; đừng
  giả định nó dùng được gói thuê bao cho tới khi tự kiểm chứng lại.
- **Chưa có quản lý package Typst qua giao diện web** (`pkg_api`) — để lại cho giai đoạn hoàn
  thiện tiếp theo.
- Server dùng **một mật khẩu chung** cho toàn instance (không có khái niệm nhiều tài khoản) —
  đúng với mô hình self-hosted một người dùng đã chốt, không phù hợp để chia sẻ cho nhiều
  người dùng không tin cậy lẫn nhau.

## Build image

```sh
docker build -t typstify .                                   # amd64, kèm Antigravity
docker buildx build --platform linux/arm64 -t typstify .     # arm64
docker build --build-arg WITH_ANTIGRAVITY=false -t typstify . # bỏ agent Antigravity (image nhỏ hơn)
```

`typst` và `tinymist` được tải đúng kiến trúc và kiểm tra SHA-256; khi nâng phiên bản
(`TYPST_VERSION`, `TINYMIST_VERSION`) phải cập nhật checksum tương ứng trong `Dockerfile`.
