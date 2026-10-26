# Giai đoạn 1: Nền tảng đa người dùng (2026-10-25)

Thuộc [plan_typst_app_parity_2026-10.md](plan_typst_app_parity_2026-10.md). Đối chiếu typst.app: [typst-app-analysis.md](../research/typst-app-analysis.md), mục 2 và 3.

## Ràng buộc hiện trạng

Server giữ **một project hiện hành toàn cục** (`appSrv.CurrentProjectDir()`). tinymist, preview và agent đều gắn với project đó. Vì vậy giai đoạn này **không** tách phiên làm việc theo từng người dùng; việc đó thuộc Giai đoạn 3, khi tài liệu đi qua Yjs theo từng file.

Giai đoạn 1 làm phần cần trước:
- danh tính và quyền trên project;
- chia sẻ;
- MFA.

## Phạm vi

1. **Sổ đăng ký project** (`service/projects`, Go thuần, không phụ thuộc Gio).
   - Mỗi `Project` gồm:
     - `ID` ngẫu nhiên, `Name`, `Dir` (tuyệt đối);
     - `Owner`, `Members map[username]Role`;
     - `CreatedAt`, `UpdatedAt`.
   - Vai trò theo typst.app: `owner` > `editor` (Write) > `commenter` (Review) > `viewer` (Read).
   - Lưu file JSON `projects.json` cạnh `auth_users.json`, ghi nguyên tử (temp + rename). Chọn JSON thay cho bbolt để thống nhất với kho auth hiện có; dữ liệu nhỏ.
   - **Share link**: token 32 byte ngẫu nhiên, chỉ lưu SHA-256, kèm vai trò (không cho `owner`), hạn dùng tuỳ chọn, thu hồi được. Ai đăng nhập và có token thì `Join` để thành thành viên với vai trò đó; nếu đã có vai trò cao hơn thì giữ vai trò cao hơn.
2. **Quản trị viên**: `User.Admin`.
   - Người đăng ký đầu tiên và tài khoản `admin` cũ là admin.
   - Khi tải dữ liệu cũ chưa có admin nào: user tạo sớm nhất thành admin.
   - Admin xem và quản lý mọi project.
3. **API** (`server/projects_api.go`, chỉ làm transport):

   | Phương thức | Đường dẫn | Việc |
   |---|---|---|
   | GET | `/api/projects` | Project mà user có quyền, kèm vai trò |
   | POST | `/api/projects` | Tạo project mới `{name}` dưới `ProjectRoot` (hoặc `RootDir`), tạo `main.typ`, owner = user |
   | POST | `/api/projects/register` | Đăng ký thư mục có sẵn `{path, name}`: chỉ admin, hoặc người mở khi thư mục chưa đăng ký |
   | POST | `/api/projects/{id}/open` | Mở làm project hiện hành; cần ≥ viewer |
   | GET | `/api/projects/{id}/members` | Danh sách thành viên |
   | PUT | `/api/projects/{id}/members/{user}` | Đặt vai trò; chỉ owner hoặc admin |
   | DELETE | `/api/projects/{id}/members/{user}` | Xoá thành viên; chỉ owner hoặc admin, không xoá được owner |
   | GET | `/api/projects/{id}/shares` | Danh sách share link |
   | POST | `/api/projects/{id}/shares` | Tạo link `{role, expiresInDays}`; trả token một lần |
   | DELETE | `/api/projects/{id}/shares/{id}` | Thu hồi link |
   | POST | `/api/projects/join` | Tham gia bằng `{token}` |
   | GET | `/api/workspace/role` | Vai trò của user trên project hiện hành |

4. **Thực thi quyền** trên project hiện hành, khi project đó đã đăng ký:
   - Các route ghi (`PUT/POST/DELETE /api/workspace/file`, `rename`, `/api/sync/push|delete`) cần ≥ editor.
   - Đọc cần ≥ viewer. Không đủ quyền trả 403.
   - Thư mục **chưa đăng ký** giữ hành vi cũ (mọi user đã đăng nhập), để không phá các triển khai đang chạy.
   - `/api/workspace/open|create` theo đường dẫn: thư mục đã đăng ký thì kiểm tra quyền như `/{id}/open`. Thư mục mới tạo qua `create` thì tự đăng ký, owner = người tạo.
5. **MFA TOTP** (RFC 6238: HMAC-SHA1, 30 s, 6 số, chấp nhận lệch ±1 bước, tự viết, không thêm thư viện).
   - `POST /api/auth/mfa/setup`: sinh secret tạm, trả `otpauth://` URI và secret base32.
   - `POST /api/auth/mfa/enable {code}`: bật MFA, trả 10 mã khôi phục (lưu dạng argon2id, dùng một lần).
   - `POST /api/auth/mfa/disable {password, code}`.
   - Đăng nhập và cấp API token: user đã bật MFA mà thiếu hoặc sai `code` thì trả 401 kèm `{"mfaRequired": true}`. Mã khôi phục dùng được thay cho mã TOTP.
   - `GET /api/auth/status` thêm `mfaEnabled`, `admin`.
6. **Web**
   - `ProjectPicker` có mục "Dự án của tôi" (từ `/api/projects`), nút tạo project theo tên, nhãn vai trò.
   - Liên kết `?join=<token>` tự tham gia sau khi đăng nhập.
   - Header Workspace có nút **Chia sẻ** (owner/admin): quản lý thành viên và share link. User chỉ có quyền xem thì ẩn Lưu và chặn sửa.
   - Login có ô mã xác thực khi server trả `mfaRequired`.
   - Settings có mục **Bảo mật**: bật MFA (hiện secret/URI để quét bằng ứng dụng authenticator, nhập mã xác nhận, xem mã khôi phục) và tắt MFA.

## Ngoài phạm vi (để giai đoạn sau)

- Phiên làm việc tách theo user, nhiều project mở đồng thời (Giai đoạn 3).
- Comment (Giai đoạn 3).
- Mời qua email, team, OIDC (P3).

## Kiểm chứng

- `go test ./service/projects/ ./server/`:
  - vai trò và so sánh vai trò;
  - join bằng link, link hết hạn hoặc đã thu hồi;
  - viewer bị 403 khi ghi file;
  - owner không bị xoá;
  - TOTP theo vector của RFC 6238;
  - login có MFA và mã khôi phục dùng một lần.
- `cd web && npm test && npm run lint && npm run build`.
- Chạy thử server: đăng ký, tạo project, tạo link viewer, đăng nhập user thứ hai, join, thử sửa (bị chặn), bật MFA.
