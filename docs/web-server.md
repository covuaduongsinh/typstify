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

Mặc định container chỉ bind vào `127.0.0.1:8080` trên máy chủ (xem `docker-compose.yml`) —
**không** tự có TLS. Nếu muốn truy cập từ Internet, đặt một reverse proxy có TLS (Caddy,
nginx, Traefik...) phía trước, trỏ vào cổng đó. Không public thẳng cổng 8080 ra Internet mà
không có TLS, vì mật khẩu đăng nhập và session cookie sẽ đi ở dạng plaintext.

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

## Giới hạn đã biết của v1

Xem đầy đủ ở mục "Rủi ro & điểm cần xử lý sớm" trong
[`docs/plans/plan_web_version.md`](plans/plan_web_version.md). Tóm tắt các điểm ảnh hưởng
trực tiếp tới vận hành:

- **Preview không tự cập nhật theo từng phím gõ chưa lưu** — cần `Ctrl+S` để thấy thay đổi.
- **Chỉ hỗ trợ chính thức agent AI phân phối qua `npx`** (ví dụ Claude Code mặc định). Agent
  dạng `uvx` (Python) hoặc binary riêng nền tảng chưa được đóng gói sẵn trong image Docker.
- **Chưa có quản lý package Typst qua giao diện web** (`pkg_api`) — để lại cho giai đoạn hoàn
  thiện tiếp theo.
- Server dùng **một mật khẩu chung** cho toàn instance (không có khái niệm nhiều tài khoản) —
  đúng với mô hình self-hosted một người dùng đã chốt, không phù hợp để chia sẻ cho nhiều
  người dùng không tin cậy lẫn nhau.
