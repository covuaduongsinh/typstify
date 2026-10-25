# Kế hoạch chi tiết — Giai đoạn 2: Biên dịch WASM trong trình duyệt

> Thuộc [plan_typst_app_parity_2026-10.md](plan_typst_app_parity_2026-10.md). Ngày lập: 2026-10-25.

## Mục tiêu

Preview web cập nhật ngay khi gõ, không chờ máy chủ, giống typst.app (compiler WASM ~31 MB chạy trong worker). Biên dịch phía server (`/api/preview/render`, `/api/export`) vẫn giữ nguyên làm phương án dự phòng và để xuất PDF.

## Lựa chọn công nghệ

| Hạng mục | Chọn | Lý do |
|---|---|---|
| Compiler | `@myriaddreamin/typst-ts-web-compiler@0.8.0-rc3` | Nhân Typst 0.15 (đã kiểm `typst-assets-0.15.0` trong wasm), khớp typst 0.15.1 của server và Docker. Bản ổn định 0.7.0 còn ở Typst 0.14.2, nên tài liệu có thể lệch so với PDF xuất từ server. |
| Renderer | `@myriaddreamin/typst-ts-renderer@0.8.0-rc3` | Nhận delta vector, vá DOM SVG tăng dần (`renderToSvg` trên cùng một session). |
| Wrapper | `@myriaddreamin/typst.ts@0.8.0-rc3` | API `createTypstCompiler`, `withAccessModel`, `withPackageRegistry`, `loadFonts`. |

Ghim chính xác phiên bản (`--save-exact`). Lên bản 0.8.0 chính thức khi có.

## Kiến trúc

```
Editor (CM6) ──content──▶ WasmCompilerClient ──postMessage──▶ compiler.worker.ts
                                                              ├─ ProjectAccessModel  (manifest + /api/sync/pull, sync XHR, cache theo hash)
                                                              ├─ ServerPackageRegistry (/api/wasm/packages/..., sync XHR, IndexedDB)
                                                              └─ fonts (/api/wasm/fonts + font mặc định, Cache Storage)
WasmPreview ◀── delta vector + diagnostics ──────────────────┘
   └─ renderer (main thread): session.manipulateData(reset|merge) → renderToSvg(container)
```

- Worker giữ một `IncrementalServer` sống suốt phiên, nên mỗi lần biên dịch chỉ gửi delta. Khi đổi file chính thì reset.
- Nội dung đang gõ đưa vào bằng `addSource`; các file khác đọc lười từ server khi compiler cần.
- Các yêu cầu biên dịch bị gộp: đang biên dịch mà có nội dung mới thì chỉ giữ bản mới nhất.

## Phía server (Go)

Logic đặt ở `service/wasmassets/`, transport ở `server/wasm_api.go`. Mọi route cần đăng nhập (`s.handle`), không phụ thuộc project.

| Route | Việc |
|---|---|
| `GET /api/wasm/fonts` | Danh sách font `{id, name, size, url}`. Nguồn: thư mục font tải lên (`fonts.Dir`), `TYPSTIFY_WASM_FONT_DIRS`, và font hệ thống khớp `TYPSTIFY_WASM_SYSTEM_FONTS` (mặc định Roboto, Noto Serif/Sans, Noto Sans Symbols 2 — đúng các font `chessbook/lib/theme.typ` dùng). Không lộ đường dẫn: `id` là hash. |
| `GET /api/wasm/fonts/{id}` | Bytes font, `Cache-Control: private, max-age=31536000, immutable` (URL có tham số phiên bản theo size+mtime). |
| `GET /api/wasm/packages/{namespace}/{name}/{version}` | `.tar.gz` của package. Tìm lần lượt ở `EffectivePackageDir`, thư mục data của Typst, thư mục cache package. Nếu thiếu và namespace là `preview` thì tải từ `packages.typst.org` vào cache (giới hạn kích thước, chống path traversal khi giải nén). Package `preview` cache bất biến; package `local` dùng ETag. |

Kiểm tra đầu vào: namespace/name `^[a-z0-9][a-z0-9-]*$`, version `^\d+\.\d+\.\d+$`.

## Phía web

| Tệp | Nội dung |
|---|---|
| `web/src/lib/wasm/protocol.ts` | Kiểu message worker ↔ main. |
| `web/src/lib/wasm/accessModel.ts` | `ProjectAccessModel` (fetch đồng bộ tiêm vào, test được). |
| `web/src/lib/wasm/packageRegistry.ts` | `ServerPackageRegistry` + bộ nhớ đệm. |
| `web/src/lib/wasm/diagnostics.ts` | Chuyển diagnostics của typst.ts thành thông báo có dòng/cột. |
| `web/src/lib/wasm/compiler.worker.ts` | Worker: khởi tạo compiler, biên dịch tăng dần, query vị trí heading. |
| `web/src/lib/wasm/client.ts` | `WasmCompilerClient` (singleton, lazy, gộp yêu cầu). |
| `web/src/lib/wasm/sourceMap.ts` | Click-to-source theo văn bản: lấy chữ dưới con trỏ (`.tsel`), tìm trong nguồn, chọn lần xuất hiện gần dòng ước lượng nhất. |
| `web/src/components/WasmPreview.tsx` | Renderer, cuộn đồng bộ, click-to-source. |
| `PreviewPane.tsx` | Chế độ mới **Tức thì** (mặc định nếu trình duyệt hỗ trợ), nhớ lựa chọn trong `localStorage`. Lỗi khởi tạo WASM thì tự quay về chế độ server. |
| `Workspace.tsx` | Nối click-to-source → nhảy editor tới dòng. |

## Giới hạn đã biết (ghi rõ trong tài liệu)

- typst.ts web compiler 0.8.0-rc3 không có API đổi span → vị trí nguồn. Vì vậy click-to-source dùng khớp văn bản, chính xác với văn bản thường nhưng không với nội dung sinh ra từ code (bàn cờ, bảng dữ liệu).
- Lần đầu phải tải compiler (~30 MB, nén khoảng 9 MB qua Caddy `encode zstd gzip`) và font. Từ lần sau dùng cache trình duyệt.
- Offline đầy đủ (service worker) để Giai đoạn 7 (PWA). Giai đoạn này chỉ cache package trong IndexedDB và font trong Cache Storage.
- Chưa làm (ghi nhận sau rà soát): hạn mức/dọn dẹp IndexedDB, nạp package đã lưu theo nhu cầu thay vì toàn bộ khi khởi động, báo rõ khi hết phiên đăng nhập, tự host font mặc định của Typst.

## Kết quả (2026-10-25)

Đã triển khai đủ các mục trên; kiểm chứng và các lỗi đã sửa sau rà soát ghi ở `docs/HISTORY.md` (mục 2026-10-25, Giai đoạn 2).

## Kiểm chứng

- `go test ./server/ ./service/wasmassets/`: liệt kê font, chặn id lạ, tar.gz package local, chặn tên package xấu, không tải mạng trong test (tiêm downloader giả).
- `npm test`: access model, registry, diagnostics, sourceMap.
- `npm run lint && npm run build`.
- Chạy thật: mở tài liệu chessbook demo ở chế độ Tức thì trong Chrome, đo thời gian biên dịch mỗi lần gõ, click vào chữ trên preview nhảy đúng dòng, cuộn đồng bộ hai chiều, tắt server vẫn biên dịch tiếp được với nội dung đang gõ.
