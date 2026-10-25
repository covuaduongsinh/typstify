# Giai đoạn 5: Template và Typst Universe

> Thuộc `docs/plans/plan_typst_app_parity_2026-10.md`. Ngày lập: 2026-10-25.

## Hiện trạng

- 8 mẫu cờ vua viết cứng trong `web/src/components/NewDocModal.tsx`, mỗi mẫu một tệp, chỉ tạo tệp trong dự án đang mở.
- Bảng Gói (`PackageManager.tsx`) tìm qua máy chủ Tpix (`tpix.typstify.com`), không có ảnh xem trước, không có nút chèn `#import`, không xem được tài liệu gói.
- `POST /api/projects` chỉ tạo dự án rỗng (`main.typ` có một tiêu đề).
- Preview "Tức thì" đã tự tải gói `@preview` từ `packages.typst.org` (`service/wasmassets`).

## Dữ liệu Universe (đã kiểm chứng 2026-10-25)

- `https://packages.typst.org/preview/index.json`: khoảng 2,4 MB, 5037 bản ghi (mỗi phiên bản một bản ghi). 870 gói có `template: {path, entrypoint, thumbnail}`.
- Ảnh xem trước: `https://packages.typst.org/preview/thumbnails/{name}-{version}-small.webp` (khoảng 3 KB) và `{name}-{version}.webp`.
- Gói: `https://packages.typst.org/preview/{name}-{version}.tar.gz`.

## Hạng mục

| # | Hạng mục | Cách làm |
|---|---|---|
| 1 | Chỉ mục Universe phía máy chủ | `service/universe`: tải `index.json`, giữ bản mới nhất của mỗi gói, cache trên đĩa, làm mới sau 12 giờ, mất mạng thì dùng bản cũ. `GET /api/universe/packages?kind=template\|package&q=&category=&offset=&limit=` |
| 2 | Ảnh xem trước | `GET /api/universe/thumbnail/{name}/{version}?size=small`: máy chủ tải hộ và cache trên đĩa (trình duyệt không gọi thẳng bên thứ ba) |
| 3 | Tài liệu gói | `GET /api/universe/readme/{name}/{version}`: README.md trong gói (gói được tải vào cache như preview) |
| 4 | Dự án từ template | `POST /api/projects` nhận thêm `template: {name, version}` (chép thư mục `[template].path` theo `typst.toml` của gói, như `typst init`) hoặc `mainContent` (mẫu cờ vua). Thất bại thì xoá thư mục vừa tạo. Trả về tệp chính để mở |
| 5 | Thư viện mẫu trên web | Modal "Dự án mới từ mẫu": thẻ "Cờ vua" (8 mẫu hiện có, chuyển sang `lib/chessTemplates.ts`) và thẻ "Typst Universe" (tìm kiếm, lọc theo nhóm, lưới ảnh xem trước, xem chi tiết, đặt tên dự án) |
| 6 | Universe trong editor | Bảng Gói: nút "Chèn #import" (thêm `#import "@preview/tên:phiên-bản": *` lên đầu tệp nếu chưa có) và "Tài liệu" (README hiển thị bằng `react-markdown`, không cho HTML thô) |

## Bảo mật

- Chỉ gọi tới host cố định `packages.typst.org`; tên và phiên bản đi qua `wasmassets.ParseSpec`.
- Chép template: chỉ đường dẫn tương đối nằm trong thư mục gói, bỏ qua symlink, giới hạn số tệp và dung lượng (dùng lại giới hạn của `wasmassets`).
- Tạo dự án giữ quy tắc hiện có (người đã đăng nhập, thư mục dưới `projectsBaseDir`).

## Kiểm chứng

- Go: lọc/phân trang chỉ mục, chọn phiên bản mới nhất, dùng bản cache khi mất mạng; chép template (chặn `..`, symlink); tạo dự án từ template và dọn dẹp khi lỗi (dùng máy chủ HTTP giả, không gọi mạng thật trong test).
- Web: vitest cho phần chèn `#import`.
- Chạy thật trong Chrome: tạo dự án từ một template Universe và một mẫu cờ vua, preview biên dịch được; chèn `#import`; xem README.
