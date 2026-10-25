# Giai đoạn 7b: Ứng dụng desktop Tauri 2

> Thuộc `docs/plans/plan_typst_app_parity_2026-10.md`. Ngày lập: 2026-10-25. Chủ dự án chọn build với cấu hình Cargo mặc định (bộ nhớ đệm Cargo trên C:).

## Kiến trúc

- `apps/tauri/`: vỏ Tauri 2 (Rust). Không có giao diện riêng: cửa sổ chính mở chính giao diện web React do `typstify-server` phục vụ, nên web và desktop dùng một mã nguồn.
- `typstify-server` chạy kèm (sidecar) trên `127.0.0.1:<cổng trống>`, cùng `typst.exe` và `tinymist.exe` đặt cạnh tệp chạy (server tự tìm ở thư mục của mình, `utils.LookupExecutable`).
- Tài nguyên đóng gói: `web/dist` (giao diện) và thư mục gói Typst (`local/chessbook/0.1.0`, `preview/board-n-pieces/0.9.0`), trỏ bằng `TYPST_PACKAGE_PATH` như bản Docker.
- Đăng nhập: mỗi lần khởi động, vỏ tạo token ngẫu nhiên, truyền cho server qua biến môi trường `TYPSTIFY_LAUNCH_TOKEN` (không qua tham số dòng lệnh) và mở `/api/auth/launch?token=…`. Server luôn bật xác thực khi có token, nên trang web khác trên máy không gọi được vào cổng cục bộ. Phiên thuộc tài khoản `desktop` (tạo ở lần đầu, mật khẩu ngẫu nhiên) hoặc tài khoản quản trị sẵn có.
- Một phiên bản duy nhất (`tauri-plugin-single-instance`): cơ sở dữ liệu bbolt của server chỉ mở được một lần.
- Cửa sổ chờ (splash, trang cục bộ) hiện trong lúc server khởi động; server lỗi thì splash báo lỗi. Đóng app thì dừng server.

## Hạng mục

1. Server: `Options.LaunchToken`, `GET /api/auth/launch` (`server/launch.go`), đọc `TYPSTIFY_LAUNCH_TOKEN` trong `cmd/typstify-server` rồi xoá khỏi môi trường của tiến trình con.
2. `apps/tauri/scripts/prepare.ps1`: build server, tải typst/tinymist (cùng phiên bản và SHA-256 với `scripts/package-desktop.ps1`), build web, chép gói chessbook và board-n-pieces.
3. `apps/tauri/src-tauri`: `main.rs` (cổng trống, token, chạy sidecar, chờ cổng mở, mở cửa sổ, dừng server khi thoát), `tauri.conf.json` (externalBin, resources, icon).
4. CI và bộ cài: để sau khi chạy ổn trên Windows.

## Kiểm chứng

- Go: test đăng nhập khởi động (token sai bị từ chối, xác thực luôn bật, tạo/dùng tài khoản đúng).
- `cargo check`/`tauri build` trên Windows; chạy bản build: splash → cửa sổ chính đã đăng nhập, mở dự án, preview biên dịch được (typst/tinymist/chessbook đi kèm), đóng app thì `typstify-server.exe` không còn chạy.
