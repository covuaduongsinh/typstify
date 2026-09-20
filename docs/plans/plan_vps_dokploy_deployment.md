# Kế hoạch triển khai Typstify Web lên VPS (Docker Compose + Caddy TLS + Claude Code/Antigravity qua ACP)

## Bối cảnh

Bạn đang tự host **Typstify Web** (repo `looz.ws/typstify`, thư mục `D:\code\typstify`) lên một
**VPS Linux đã có sẵn**, có **domain riêng** và muốn **TLS thật** (Let's Encrypt), để dùng từ xa
qua trình duyệt giống mô hình `code-server`. Bạn vừa gặp lỗi "Connection to the agent was lost"
trên máy Windows dev do agent AI đang chọn (Google Antigravity) bị cấu hình `cmd` là đường dẫn
tương đối, không resolve được. Bạn muốn dùng AI agent qua **ACP** (không phải MCP — MCP không thay
thế được ACP cho mục đích này, xem giải thích đã trao đổi ở lượt trước) với **cả Claude Code**
(ưu tiên/mặc định, đăng nhập Claude Pro/Max) **và Google Antigravity CLI** (best-effort, đăng nhập
gói Google AI) — cả hai xác thực qua **CLI chính chủ dùng gói thuê bao tháng**, không dùng API key
tính theo usage.

Kiến trúc self-host (Dockerfile multi-stage + `docker-compose.yml`) đã có sẵn trong repo và đã được
verify chạy thật (theo `docs/plans/plan_web_version.md` giai đoạn 4 và `docs/web-server.md`):
image cuối dựa trên `node:22-slim` (giữ `npx` để chạy Claude Code/Codex), cài sẵn binary
`typst`/`tinymist`/`agy_acp_server` (Antigravity, Linux, cần CPU hỗ trợ **AVX**), chạy non-root,
persist dữ liệu qua volume `typstify-data`. `docker-compose.yml` hiện tại **chỉ bind
`127.0.0.1:8080`** — đúng ý đồ, cần một reverse proxy TLS phía trước để truy cập từ Internet, và
đây là phần **chưa có sẵn**, cần thêm mới.

Ngoài ra, research phát hiện 2 gap nhỏ ở tầng code đáng vá trước khi coi cấu hình agent là "tối
ưu" (không bắt buộc để chạy được, nhưng ngăn đúng loại lỗi bạn vừa gặp xảy ra lại khi đổi agent qua
UI trên VPS):
- `service/service.go:438-441` pin `@agentclientprotocol/claude-agent-acp@0.50.0` làm default,
  trong khi `service/settings/models.go:357-363` pin `@0.35.0` cho cùng agent — hai default lệch
  version.
- `server/agent_api.go` (`handleAgentSelect`) chỉ kiểm tra `resolved.Cmd != ""` trước khi lưu
  settings, không gọi `utils.LookupExecutable` (đã có sẵn, dùng ở bước spawn) để xác nhận binary
  thực sự tồn tại — nên chọn sai agent qua UI vẫn lưu được, lỗi chỉ lộ ra lúc agent thật sự được
  spawn.

## Kế hoạch

### Giai đoạn 0 — Vá 2 gap nhỏ (khuyến nghị làm trước khi deploy)

1. Thống nhất version pin: sửa `service/settings/models.go:357-363`
   (`defaultAcpAgentSettings.Args`) từ `@0.35.0` lên `@0.50.0` để khớp
   `service/service.go:438-441`.
2. Thêm validate ở `server/agent_api.go` `handleAgentSelect`: sau khi `ResolveAgentCommand`, gọi
   `utils.LookupExecutable(resolved.Cmd)` (package `utils`, đã dùng ở `agent/manager.go` lúc
   spawn) — nếu không resolve được, trả lỗi HTTP rõ ràng ngay tại bước chọn thay vì chỉ phát hiện
   khi WebSocket agent spawn thất bại.

Đây là thay đổi code nhỏ, không ảnh hưởng kiến trúc; có thể bỏ qua nếu muốn deploy ngay và chấp
nhận rủi ro gặp lại lỗi tương tự nếu tự tay nhập sai `cmd` qua UI sau này.

### Giai đoạn 1 — Chuẩn bị VPS

- Cài Docker Engine + Docker Compose plugin trên VPS (theo hướng dẫn chính thức Docker cho
  distro đang dùng).
- Kiểm tra CPU có AVX không: `grep avx /proc/cpuinfo`. Nếu **không** có output, binary Linux của
  Antigravity (`agy_acp_server`) sẽ SIGILL (exit 132) trong container — set kỳ vọng đúng: dùng
  Claude Code làm chính, coi Antigravity là "thử, có thể không chạy được" thay vì mặc định phải
  hoạt động.
- Mở firewall: chỉ **80/443** (cho Caddy) ra ngoài. **Không** mở 8080 — service `typstify` trong
  `docker-compose.yml` hiện tại đã chỉ bind `127.0.0.1:8080`, đúng thiết kế, giữ nguyên.
- Trỏ A record của domain về IP VPS **trước** khi chạy Caddy (cần cho bước xin chứng chỉ Let's
  Encrypt tự động qua HTTP-01 challenge ở bước 2).

### Giai đoạn 2 — Thêm reverse proxy Caddy (TLS thật) vào `docker-compose.yml`

File **mới**: `Caddyfile` ở root repo trên VPS:

```
your-domain.com {
    reverse_proxy typstify:8080
}
```

Caddy v2 tự động forward header `Upgrade`/`Connection` cho WebSocket qua `reverse_proxy` — không
cần cấu hình riêng cho `/ws/agent`, `/ws/lsp`, `/preview` (đã xác nhận các route WS này tồn tại
qua `server/server.go` routes).

**Sửa** `docker-compose.yml`: thêm service `caddy` (image `caddy:2-alpine`), mount
`./Caddyfile:/etc/caddy/Caddyfile:ro`, volumes `caddy_data`/`caddy_config` (lưu chứng chỉ, tồn tại
qua các lần restart), map port `80:80` và `443:443` ra host, cùng network mặc định của compose với
service `typstify` (Caddy gọi `typstify:8080` qua tên service nội bộ Docker, không cần expose
8080 ra host).

### Giai đoạn 3 — Cấu hình `.env`

- Copy `.env.example` → `.env` trên VPS.
- Generate password mạnh, ví dụ `openssl rand -base64 24`, gán vào `TYPSTIFY_SERVER_PASSWORD`.
- Xác nhận `.env` nằm trong `.gitignore` (không commit).

### Giai đoạn 4 — Deploy lần đầu

- `docker compose up -d --build`
- `docker compose logs -f typstify` — không có lỗi khởi động (đối chiếu format log đã thấy khi
  chạy thử local: `main.go:80: listening on 127.0.0.1:8080`).
- `docker compose logs -f caddy` — thấy dòng xác nhận xin chứng chỉ Let's Encrypt thành công.
- Truy cập `https://your-domain.com` từ trình duyệt — phải thấy trang login Typstify Web, khoá
  ổ khoá TLS hợp lệ (không cảnh báo chứng chỉ).

### Giai đoạn 5 — Cấu hình AI agent lần đầu (qua UI web, sau khi container đã chạy)

1. Đăng nhập bằng `TYPSTIFY_SERVER_PASSWORD`.
2. **Settings → Agent Registry** (`AgentRegistryPicker.tsx`) → chọn **Claude Code** → Use. Hệ
   thống lưu `cmd: npx`, `args: -y @agentclientprotocol/claude-agent-acp@<version đã thống nhất
   ở Giai đoạn 0>` — chạy được ngay vì image nền `node:22-slim` đã có `npx`.
3. Mở **AI Assistant** chat → server báo `authRequired` → `AuthCard` hiện → bấm "Sign in" →
   backend gọi `/api/agent/auth/{methodId}`, agent tự in URL OAuth ra stderr, frontend poll
   `/api/console` mỗi 1.5s tìm URL bằng regex, hiện nút "Open login link" → đăng nhập bằng tài
   khoản **Claude Pro/Max** ở tab mới → quay lại, phiên chuyển sang "connected".
4. (Tuỳ chọn) Lặp lại, chọn **Google Antigravity** trong Agent Registry để có phương án agent thứ
   hai. Nếu log container báo exit 132 (SIGILL) — nghĩa là CPU VPS không có AVX như kiểm tra ở
   Giai đoạn 1 — quay lại dùng Claude Code làm mặc định duy nhất, Antigravity không khả dụng trên
   VPS này.

### Giai đoạn 6 — Kiểm chứng end-to-end

- Gửi 1 prompt thật trong chat AI (Claude Code) yêu cầu sửa nội dung một file `.typ` trong
  project, xác nhận nhận phản hồi và agent đọc/sửa file được qua ACP.
- Nhấn `Ctrl+S`, xác nhận preview qua `/preview/` (kết nối WSS qua Caddy) cập nhật đúng.
- Nếu đã bật Antigravity: lặp lại test tương tự, chấp nhận độ trễ/độ ổn định kém hơn Claude Code
  theo đúng cảnh báo đã ghi trong `docs/web-server.md`.

### Giai đoạn 7 — Vận hành lâu dài

- **Backup** định kỳ volume `typstify-data` (chứa project + settings + cache):
  `docker run --rm -v typstify-data:/data -v $(pwd):/backup alpine tar czf /backup/typstify-$(date +%F).tar.gz /data`
- **Update**: `git pull && docker compose up -d --build` (rebuild từ source) — không cần xoá
  volume, dữ liệu persist qua các lần rebuild.
- **Xoay password**: đổi `TYPSTIFY_SERVER_PASSWORD` trong `.env`, `docker compose up -d` để áp
  dụng lại.
- **Nhắc ToS**: gói thuê bao (Claude Pro/Max, Google AI) chỉ dành cho dùng cá nhân qua CLI chính
  chủ; vì server chỉ spawn agent khi bạn chủ động mở chat (không cron/queue nào tự gọi), rủi ro vi
  phạm điều khoản thấp — nhưng tránh chia sẻ URL/password server này cho người khác dùng chung một
  tài khoản subscription.

## Files cần tạo/sửa

- **MỚI**: `Caddyfile` (root repo, deploy cùng VPS)
- **SỬA**: `docker-compose.yml` — thêm service `caddy`, volumes `caddy_data`/`caddy_config`, map
  port 80/443
- **MỚI** (trên VPS, không commit): `.env` từ `.env.example`
- **SỬA** (Giai đoạn 0, tuỳ chọn): `service/settings/models.go` (thống nhất version pin),
  `server/agent_api.go` (`handleAgentSelect` — thêm validate qua `utils.LookupExecutable`)

## Verification

- `docker compose ps` — cả `typstify` và `caddy` đều `Up`.
- Từ máy khác (ngoài VPS): `curl -I https://your-domain.com/api/health` → `200`, chứng chỉ hợp lệ.
- Đăng nhập UI qua domain HTTPS, chọn Claude Code, hoàn tất OAuth qua `AuthCard`, gửi prompt thật,
  nhận phản hồi và xác nhận agent sửa được file.
- Nếu bật Antigravity: log container không có exit 132; gửi thử 1 prompt xác nhận phản hồi được.
- `docker compose restart typstify` → xác nhận project/settings/cache vẫn còn nguyên (volume
  persist đúng), không phải đăng nhập lại agent từ đầu.
