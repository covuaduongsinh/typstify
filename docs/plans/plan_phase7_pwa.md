# Giai đoạn 7a: PWA (web cài được, chạy khi mạng chập chờn)

> Thuộc `docs/plans/plan_typst_app_parity_2026-10.md`. Phần Tauri (desktop, mobile) làm sau, khi web đã ổn định. Ngày lập: 2026-10-25.

## Mục tiêu

- Cài Typstify như một ứng dụng (Chrome/Edge trên máy tính, Android; "Thêm vào màn hình chính" trên iOS).
- Giao diện mở được ngay cả khi máy chủ không liên lạc được, và báo rõ đang ngoại tuyến.
- Phiên bản mới không phá phiên đang mở: người dùng chủ động tải lại.

## Hạng mục

| # | Hạng mục | Cách làm |
|---|---|---|
| 1 | Manifest | `public/manifest.webmanifest`: tên, màu `#2B3990`, `display: standalone`, icon PNG 192/512 và maskable (vẽ từ `favicon.svg`), `apple-touch-icon` |
| 2 | Service worker | `sw.js` tự viết, sinh lúc build bằng plugin Vite nhỏ (chèn danh sách tệp cần cache sẵn và mã phiên bản). Không thêm `vite-plugin-pwa`/Workbox |
| 3 | Chiến lược cache | Trang (`navigate`): mạng trước, mất mạng thì dùng bản đã cache. `/assets/*` (tên có hash): cache trước. Tệp `.wasm` (compiler ~30 MB) không cache sẵn khi cài, chỉ cache lần đầu dùng. Google Fonts: dùng bản cache rồi cập nhật nền. Không bao giờ cache `/api`, `/ws`, `/preview` |
| 4 | Cập nhật | Bản mới chờ (`waiting`), giao diện hiện "Có phiên bản mới, Tải lại"; bấm thì `SKIP_WAITING` rồi tải lại. Cache tài nguyên cũ được giữ tới khi bản mới kích hoạt, để trang cũ còn tải được các chunk lười |
| 5 | Ngoại tuyến | Thanh báo "Đang ngoại tuyến" theo `navigator.onLine`; việc soạn thảo ngoại tuyến đã có từ Giai đoạn 3 (y-indexeddb) |

## Kiểm chứng

- Build sinh `dist/sw.js` có danh sách tệp, `manifest.webmanifest` hợp lệ (Chrome DevTools: Application → Manifest không lỗi, có nút cài).
- Chạy thật: lần tải thứ hai trang và chunk lấy từ service worker; tắt máy chủ rồi tải lại vẫn ra giao diện kèm báo ngoại tuyến; build mới thì hiện nút tải lại.
- vitest cho phần chọn chiến lược cache theo URL.
