# Giai đoạn 7c: ứng dụng mobile Tauri 2 (Android)

> Thuộc `plan_typst_app_parity_2026-10.md`, Giai đoạn 7. Chốt với chủ dự án ngày 2026-10-26:
> giao diện **đóng gói trong app** (không tải từ server); cache build để trên ổ D:; chạy thử
> trên máy ảo x86_64. iOS không làm được trên Windows (cần macOS + Xcode), để sau.

## Bài toán

Giao diện web (`web/`) gọi server bằng hơn 100 URL tương đối (`/api/...`, `/ws/...`, iframe
`/preview/`, link tải PDF) và xác thực bằng cookie cùng origin. Nếu đóng gói giao diện vào app
rồi gọi thẳng server từ xa thì phải đổi sang token + CORS + WebSocket chéo origin: nhiều chỗ
sửa, dễ hở bảo mật (token trong URL, cookie bên thứ ba bị Android WebView chặn).

## Thiết kế: proxy cục bộ trong vỏ Rust

```
WebView ──http://127.0.0.1:<cổng>──► proxy (Rust, trong app)
                                      ├─ /api, /ws, /preview ──https──► server tự host
                                      └─ mọi đường dẫn khác ──► giao diện đóng gói (web/dist)
```

- Giao diện nằm trong file APK (`frontendDist` của bản mobile = `web/dist`, đọc qua
  `asset_resolver`), mở được ngay cả khi chưa có mạng; trình biên dịch WASM và IndexedDB chạy
  như bản web.
- WebView thấy một origin duy nhất nên code web, cookie phiên, WebSocket, iframe preview giữ
  nguyên. Server không cần CORS.
- Proxy viết lại `Host`, `Origin`, `Referer` theo server đích (để kiểm tra Origin của server
  vẫn đúng), và bỏ thuộc tính `Domain`/`Secure` trong `Set-Cookie` để cookie gắn với
  `127.0.0.1:<cổng>`. Cookie nằm trong kho của WebView, proxy không giữ thông tin đăng nhập
  nào: app khác gọi vào proxy cũng chỉ như truy cập server từ Internet mà không có phiên.
- Cổng được nhớ trong thư mục cấu hình app (localStorage, IndexedDB gắn theo origin, có cả cổng).
- Không có đường dẫn HTTP nào để đổi server (nếu có, app khác trên máy đổi được server đích sang
  máy của kẻ tấn công và nhận cookie). Màn hình chọn server là trang cục bộ của Tauri
  (`tauri://`), gọi lệnh IPC `connect`; trang của proxy không có quyền IPC.
- Địa chỉ server: bắt buộc `https://`; `http://` chỉ chấp nhận cho địa chỉ loopback/mạng
  nội bộ (LAN, máy ảo `10.0.2.2`). Trước khi lưu, kiểm tra `GET /api/auth/status` trả JSON.
- Đổi server: giao diện mở `/__app/server`; `on_navigation` của Rust chặn URL này và đưa cửa
  sổ về trang chọn server (proxy trả 404 cho đường dẫn đó nếu gọi qua HTTP).
- TLS: `hyper` + `hyper-rustls` với `rustls` dùng `ring` và `webpki-roots` (Android không cho
  đọc kho chứng chỉ hệ thống từ Rust một cách ổn định).

## Cấu trúc code

- `apps/tauri/src-tauri/src/lib.rs`: `run()` chung, `#[cfg_attr(mobile, tauri::mobile_entry_point)]`.
- `desktop.rs`: phần sidecar hiện có (chỉ `cfg(desktop)`), `main.rs` chỉ gọi `run()`.
- `mobile.rs`: màn hình chọn server, lệnh `connect`, khởi động proxy (`cfg(mobile)`).
- `proxy.rs`: proxy độc lập nền tảng, có unit test chạy trên Windows (`cargo test`).
- `tauri.android.conf.json`: ghi đè `frontendDist`, bỏ `externalBin`/`resources`.
- `mobile/index.html`: trang chọn server.
- `src-tauri/gen/android`: sinh bằng `tauri android init`, có commit (theo khuyến nghị của Tauri).

## Web

- Thêm mục "Đổi server" (chỉ hiện khi `AuthStatus`/môi trường cho biết đang chạy trong app
  mobile) trỏ tới `/__app/server`.
- Bố cục điện thoại (**đã làm 26/10/2026**, `Workspace.tsx`, `TypstKeyBar.tsx`,
  `lib/useMediaQuery.ts`): màn hình ≤ 720px chỉ hiện một khung (Tệp / Soạn thảo / Xem trước /
  Công cụ) chuyển bằng thanh tab ở đáy; các khung vẫn mounted (giữ trạng thái editor, phòng cộng
  tác, preview), CSS chỉ ẩn/hiện. Mở tệp từ cây thư mục tự chuyển sang Soạn thảo. Khung Công cụ
  có hàng tab riêng cho Trợ lý AI, bình luận, lịch sử, dàn ý, gói, cài đặt. Thanh ký hiệu Typst
  (`# $ [ ] { } * _ = < > @ \`…) phía trên bàn phím ảo, bấm không làm mất focus editor (giữ bàn
  phím mở). Đã kiểm thử trong Chrome với khung rộng 412px; còn phải thử trên máy ảo (bàn phím
  ảo thật, cảm ứng).

## Môi trường build (đã có)

`ANDROID_HOME=D:\Android\Sdk`, NDK 27, JDK 21, `GRADLE_USER_HOME=D:\gradle`,
`ANDROID_AVD_HOME=D:\Android\avd` (AVD `rn_api36_x86_64`). Cần thêm `NDK_HOME` và Rust target
`x86_64-linux-android` (máy ảo) + `aarch64-linux-android` (máy thật). Rustup nằm trên C:
(khoảng 0,5 GB cho hai target).

## Kiểm chứng

1. `cargo test` cho `proxy.rs`: chuyển tiếp HTTP, viết lại header và cookie, WebSocket upgrade,
   404 cho `/__app/*`.
2. `cargo check` bản desktop vẫn sạch; build desktop không đổi hành vi.
3. Server Go chạy trên máy host (`0.0.0.0`), máy ảo trỏ tới `http://10.0.2.2:<cổng>`: đăng
   nhập, mở project, gõ, preview WASM, cộng tác với một tab trình duyệt trên host.
4. Rà soát bảo mật riêng cho proxy và IPC. **Đã làm (26/10/2026)**: proxy từng tin header
   `Host` để quyết định viết lại `Origin`, nên một trang lạ trong trình duyệt khác trên máy
   (DNS rebinding về `127.0.0.1`) có `Host` = `Origin` và được proxy đổi cả hai thành của
   server — vượt qua lớp kiểm tra Host/Origin của server, đọc được các trang không cần đăng
   nhập (kể cả server chỉ có trong LAN). Đã sửa: `Proxy::handle` chỉ nhận `Host` đúng bằng
   địa chỉ đang lắng nghe (`127.0.0.1:<port>`), còn lại trả 421; test
   `refuses_rebound_host_names`. IPC (`connect`, `current_server`): trang của proxy là
   origin ngoài (`http://127.0.0.1:<port>`), Tauri không cấp IPC cho origin ngoài khi không
   có capability `remote`, nên trang web/preview của server không gọi được `connect` — cần xác
   nhận lại khi chạy thật trên máy ảo (bước 3).

## Còn mở

iOS; ký APK bản phát hành (hiện chỉ debug/APK chưa ký). CI build Android đã viết (`.github/workflows/tauri.yml`, job `android`: APK debug aarch64 + x86_64) nhưng chưa chạy lần nào trên GitHub.
