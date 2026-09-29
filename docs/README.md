# Mục lục tài liệu Typstify

Thứ tự đọc gợi ý cho người mới / AI agent mới: `../CLAUDE.md` → `TECH.md` → `ARCHITECTURE.md` → `MODULES.md` → `ALGORITHMS.md`.

## Tổng quan và kỹ thuật
| File | Nội dung | Trạng thái |
|---|---|---|
| [../README.md](../README.md) | Giới thiệu, tính năng, chạy/build | có |
| [../CLAUDE.md](../CLAUDE.md), [../AGENTS.md](../AGENTS.md) | Luật cho AI agent, bản đồ repo, lệnh, quy ước, API chessbook (hai file giống nhau) | có |
| [TECH.md](TECH.md) | Stack, phụ thuộc, biến môi trường, cờ CLI, Docker | có |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Kiến trúc lớp, bus, settings, luồng dữ liệu | có |
| [MODULES.md](MODULES.md) | Danh mục module và trạng thái | có |
| [ALGORITHMS.md](ALGORITHMS.md) | Thuật toán chi tiết | có |
| [API.md](API.md) | Route HTTP/WebSocket, giao thức agent | có |
| [CHESSBOOK.md](CHESSBOOK.md) | Thư viện cờ vua đầy đủ | có |

## Vận hành dự án và bối cảnh cho AI
| File | Nội dung |
|---|---|
| [SKILLS.md](SKILLS.md) | Skill dự án và cách thêm skill mới |
| [MEMORY.md](MEMORY.md) | Quyết định thiết kế, cạm bẫy, điều cấm — ngữ cảnh bền vững |
| [HISTORY.md](HISTORY.md) | Lịch sử phát triển theo mốc, rút từ git log và các plan |
| [ROADMAP.md](ROADMAP.md) | Khoảng trống và việc tiếp theo |
| [REPLICATION.md](REPLICATION.md) | Nhân bản phần mềm, thêm module/tính năng mới |

## Tài liệu có sẵn từ trước
- [architecture_and_guide.md](architecture_and_guide.md) — bản đồ kiến trúc desktop (tiếng Việt). Lưu ý: mục i18n ghi có tiếng Việt nhưng desktop hiện chỉ có en-US, zh-CN, de-DE.
- [web-server.md](web-server.md) — cấu hình, bảo mật, triển khai Dokploy cho server web.
- [USER_AUTH_GUIDE.md](USER_AUTH_GUIDE.md), [LIVE_PREVIEW_USER_GUIDE.md](LIVE_PREVIEW_USER_GUIDE.md).
- [CHESS_STUDIO_GUIDE.md](CHESS_STUDIO_GUIDE.md), [CHESS_IMPORT_USER_GUIDE.md](CHESS_IMPORT_USER_GUIDE.md).

## Kế hoạch lịch sử (`plans/`)
Xem bảng đối chiếu ở [HISTORY.md](HISTORY.md#kế-hoạch-trong-docsplans). Các file này là nhật ký ý định tại thời điểm viết, không phải mô tả hiện trạng.

## Quy tắc duy trì
1. Đổi hành vi → sửa tài liệu tương ứng trong cùng commit.
2. Mọi đường dẫn/hàm nhắc trong tài liệu phải tồn tại; kiểm bằng grep trước khi chốt.
3. Thêm mục vào [HISTORY.md](HISTORY.md) cho mỗi mốc tính năng hoặc phiên làm việc đáng kể.
4. Không ghi token, mật khẩu, API key vào tài liệu.
