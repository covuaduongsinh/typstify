# Hướng dẫn Sử dụng Tính năng Quản lý Tài khoản & Đăng nhập trên Typstify Web

Tài liệu này hướng dẫn chi tiết cách sử dụng các tính năng mới trên **Typstify Web (Dương Sinh Chess Studio)** bao gồm: **Tạo tài khoản mới**, **Đăng nhập**, **Lưu thông tin đăng nhập & Duy trì phiên làm việc**, **Đổi mật khẩu** và **Đăng xuất**.

---

## 1. Tạo tài khoản mới (Đăng ký)

Khi truy cập vào đường dẫn website (ví dụ: `https://typst.dsc.edu.vn`):

1. Trên màn hình chào mừng, bấm chọn tab **Tạo tài khoản** (hoặc bấm vào dòng liên kết *"Chưa có tài khoản? Tạo tài khoản mới"* ở cuối khung).
2. Điền các thông tin:
   - **Tên đăng nhập**: Nhập tên tài khoản của bạn (từ 3 đến 32 ký tự, chỉ gồm chữ cái, số, dấu gạch dưới, gạch ngang hoặc dấu chấm). *Ví dụ: `duongsinh` hoặc `thaysinh_chess`*.
   - **Tên hiển thị (Tùy chọn)**: Nhập tên bạn muốn hiển thị trên thanh công cụ và giao diện. *Ví dụ: `Thầy Dương Sinh`*.
   - **Mật khẩu**: Nhập mật khẩu (tối thiểu 4 ký tự).
   - **Xác nhận mật khẩu**: Nhập lại chính xác mật khẩu ở trên.
3. Tích chọn **"Ghi nhớ đăng nhập trên máy này"** (mặc định đã bật).
4. Nhấn nút **Tạo tài khoản & Bắt đầu**.

> **Lưu ý**: Hệ thống sẽ tự động đăng nhập ngay sau khi tài khoản được tạo thành công và chuyển thẳng vào khu vực làm việc của bạn.

---

## 2. Đăng nhập & Tính năng Ghi nhớ mật khẩu

### A. Đăng nhập thông thường
1. Chọn tab **Đăng nhập**.
2. Nhập **Tên đăng nhập** và **Mật khẩu**.
3. Bấm **Đăng nhập**.

### B. Tính năng Ghi nhớ mật khẩu & Duy trì phiên làm việc
- **Lưu phiên làm việc tự động (Session Persistence)**: Khi bạn đã đăng nhập, phiên làm việc có hiệu lực lên tới **30 ngày**. Kể cả khi **máy chủ VPS khởi động lại** hoặc container Docker được triển khai phiên bản mới, **phiên đăng nhập vẫn được bảo lưu bền vững** trên hệ thống và bạn không bị văng ra ngoài.
- **Ghi nhớ tên tài khoản (Remember Me)**: Khi bật tùy chọn này, trình duyệt sẽ tự động nhớ tên đăng nhập của bạn cho các lần mở sau.
- **Tương thích Trình quản lý mật khẩu**: Biểu mẫu hỗ trợ đầy đủ tiêu chuẩn tự động điền mật khẩu của **Google Chrome, Microsoft Edge, Safari, 1Password, Bitwarden,...** giúp bạn đăng nhập chỉ bằng 1 cú nhấp chuột hoặc FaceID/Vân tay.

### C. Đăng nhập bằng Mật khẩu máy chủ (Legacy Server Password)
Nếu máy chủ được cấu hình mật khẩu quản trị chung qua biến môi trường `TYPSTIFY_SERVER_PASSWORD`, bạn có thể để trống trường *Tên đăng nhập* và nhập trực tiếp mật khẩu máy chủ vào ô *Mật khẩu* để đăng nhập với quyền quản trị viên `admin`.

---

## 3. Thay đổi Mật khẩu

Khi đã đăng nhập vào hệ thống, bạn có thể đổi mật khẩu bất kỳ lúc nào:

1. Trên thanh công cụ hoặc menu, mở bảng **Cài đặt** (biểu tượng bánh răng ⚙️ hoặc bấm vào Avatar/Tên người dùng).
2. Tại mục đầu tiên **Tài khoản & Bảo mật**:
   - Nhập **Mật khẩu hiện tại** đang dùng.
   - Nhập **Mật khẩu mới** (tối thiểu 4 ký tự).
   - Nhập lại **Xác nhận mật khẩu mới**.
3. Bấm **Cập nhật mật khẩu**.
4. Khi thấy thông báo màu xanh `✓ Đã đổi mật khẩu thành công`, mật khẩu mới đã có hiệu lực ngay lập tức.

---

## 4. Đăng xuất (Sign Out)

1. Mở **Cài đặt** ⚙️.
2. Tại phần **Tài khoản & Bảo mật**, bấm nút **Đăng xuất** màu đỏ.
3. Xác nhận khi hộp thoại xuất hiện. Hệ thống sẽ hủy phiên làm việc hiện tại và quay về màn hình Đăng nhập.

---

## 5. Cơ chế Bảo mật Kỹ thuật

- **Mật khẩu băm an toàn (Salted SHA-256 / Constant-Time Compare)**: Mật khẩu người dùng không bao giờ được lưu dưới dạng văn bản thuần (plain text). Mỗi tài khoản được cấp một chuỗi ngẫu nhiên (Salt 16 bytes) riêng biệt trước khi băm, giúp chống lại các cuộc tấn công Rainbow Table.
- **Bảo vệ chống Brute-Force (Rate Limiting)**: Hệ thống tự động khóa tạm thời IP nếu có hành vi cố tình thử sai mật khẩu nhiều lần liên tiếp.
- **Lưu trữ dữ liệu độc lập & Bền vững**: Dữ liệu tài khoản (`auth_users.json`) và phiên (`auth_sessions.json`) được lưu trực tiếp trong thư mục cấu hình bền vững `/data/config/` (được gắn volume trên Docker), đảm bảo không bao giờ bị mất dữ liệu khi cập nhật ứng dụng.
