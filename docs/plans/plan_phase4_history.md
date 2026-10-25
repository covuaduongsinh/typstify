# Kế hoạch chi tiết — Giai đoạn 4: Lịch sử phiên bản và Git

> Thuộc [plan_typst_app_parity_2026-10.md](plan_typst_app_parity_2026-10.md). Ngày lập: 2026-10-25.

## Mục tiêu

- Mọi thay đổi trong project được giữ lại để xem và khôi phục, dù do ai hay cái gì ghi: người dùng, phiên cộng tác, agent AI hay Dropbox.
- Có thể đặt tên một phiên bản (ví dụ "Bản gửi nhà in").
- Xem khác biệt của từng tệp so với hiện tại.
- Khôi phục một tệp hoặc cả project.
- Đồng bộ project với một kho Git (GitHub, GitLab…): commit, push, pull.

## 4a. Lịch sử phiên bản (tự xây, không cần Git)

Lưu trong thư mục lưu trữ của máy chủ, không ghi gì vào thư mục project. Logic đặt ở `service/history`, transport ở `server/history_api.go`.

- **Kho nội dung** (`<storage>/history/<khoá project>/`):
  - `objects/<2 ký tự>/<sha256>.gz`: nội dung tệp nén, dùng chung giữa các phiên bản, nên tệp không đổi không tốn thêm chỗ;
  - `snapshots/<id>.json.gz`: manifest của phiên bản (đường dẫn → hash, kích thước);
  - `index.jsonl`: mỗi dòng một phiên bản (`id`, thời điểm, tác giả, nhãn, loại, số tệp thêm/sửa/xoá).
- **Khoá project:** id trong sổ đăng ký nếu project đã đăng ký, nếu không thì hash của đường dẫn.
- **Chụp phiên bản:**
  - quét project như `vpssync.ScanManifest`, bỏ `.git`, `.typstify`, `node_modules` và tệp lớn hơn 25 MB;
  - chỉ ghi các object còn thiếu;
  - phiên bản tự động bị bỏ qua nếu không có gì đổi so với phiên bản trước.
- **Loại phiên bản:**
  - `auto`: tự động mỗi 5 phút khi project có thay đổi;
  - `manual`: người dùng đặt tên;
  - `restore`: bản an toàn chụp ngay trước khi khôi phục.
- **Dọn dẹp:**
  - giữ mọi phiên bản tự động trong 24 giờ;
  - trong 7 ngày giữ mỗi giờ một bản, sau đó mỗi ngày một bản;
  - phiên bản có tên giữ mãi;
  - object không còn được phiên bản nào dùng thì bị xoá.
- **Khôi phục:**
  - luôn chụp bản `restore` trước;
  - ghi tệp nguyên tử qua `resolveInRoot`;
  - tệp đang được cộng tác cũng ghi thẳng xuống đĩa, phòng cộng tác coi đó là sửa từ bên ngoài và gộp ba chiều, nên nếu trùng chỗ đang sửa thì hỏi người dùng;
  - khôi phục cả project không xoá tệp mới hơn, trừ khi người dùng chọn.

### API

| Method | Path | Quyền | Việc |
|---|---|---|---|
| GET | `/api/history` | viewer | Danh sách phiên bản |
| POST | `/api/history` | editor | Tạo phiên bản có tên `{label}` |
| GET | `/api/history/{id}` | viewer | Tệp của phiên bản và trạng thái so với hiện tại (thêm/sửa/xoá/như cũ) |
| GET | `/api/history/{id}/file?path=` | viewer | Nội dung một tệp ở phiên bản đó |
| POST | `/api/history/{id}/restore` | editor | `{paths?: [], deleteNewer?: bool}` |

### Web

- **Bảng "Lịch sử"** ở panel phải:
  - danh sách phiên bản (thời gian, tác giả, nhãn, số thay đổi);
  - nút **Lưu phiên bản…**.
- **Mở một phiên bản:**
  - xem danh sách tệp đã đổi;
  - bấm một tệp để xem khác biệt với hiện tại, dùng `@codemirror/merge` hiển thị hai cột hoặc gộp;
  - nút **Khôi phục tệp này** và **Khôi phục cả phiên bản**.

## 4b. Đồng bộ Git (go-git)

Logic ở `service/gitsync`, dùng `github.com/go-git/go-git/v5` thuần Go. Ảnh Docker không có lệnh `git`.

- **Cấu hình theo project** (`remote`, `branch`, tên/email tác giả) do chủ project đặt. Token truy cập lưu phía máy chủ, không bao giờ trả về trình duyệt.
- **Thao tác:**
  - trạng thái (thay đổi chưa commit, số commit đi trước/đi sau);
  - commit tất cả kèm thông điệp;
  - pull chỉ fast-forward; nếu hai bên cùng đổi thì báo để người dùng chọn: lấy bản kho, giữ bản mình rồi push lại, hoặc tải bản kho về một nhánh để so;
  - push.
- **Trước pull** luôn chụp phiên bản `restore` ở 4a, nên lấy bản từ kho không bao giờ làm mất dữ liệu cục bộ.
- **Quyền:**
  - chủ project cấu hình;
  - editor được commit, push, pull;
  - viewer xem trạng thái.

## Kiểm chứng

- Go:
  - chụp và loại trùng object, bỏ qua phiên bản tự động khi không đổi;
  - dọn dẹp theo lịch;
  - khôi phục một tệp và cả project, có bản an toàn;
  - chặn đường dẫn ngoài project;
  - Git: commit, push và pull với một kho bare cục bộ, kèm trường hợp hai bên cùng đổi.
- Web: diff hiển thị đúng, khôi phục rồi editor cập nhật (qua phòng cộng tác).
- Chạy thật:
  - sửa vài lần, mở bảng Lịch sử, xem diff, khôi phục;
  - Git push lên một kho bare cục bộ rồi pull về.
