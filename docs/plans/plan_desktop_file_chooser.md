# Kế hoạch cải tiến toàn diện File Chooser & Quản lý File/Thư mục Desktop

## Bối cảnh (Context)
Người dùng phản ánh giao diện chọn file/thư mục (File/Folder Chooser modal) trên desktop Gio UI rất khó quản lý:
1. Mục **Locations** hiển thị các ổ đĩa trên Windows thành hàng loạt dấu `\` (do `filepath.Base("C:\\")` trả về `\`), không hiển thị tên ổ đĩa và nhãn ổ đĩa (`Local Disk (C:)`, `Data (D:)`, `Google Drive (G:)`).
2. Thanh đường dẫn (Header) chỉ hiển thị `\` hoặc tên thư mục đơn lẻ thay vì toàn bộ đường dẫn đầy đủ có thể điều hướng.
3. Mục **Favorites** chỉ có duy nhất thư mục `home`, thiếu các thư mục thường dùng (Desktop, Documents, Downloads, Recent Projects).
4. Không có thanh nhập trực tiếp đường dẫn (Address Bar) để dán đường dẫn thư mục có sẵn.
5. Danh sách file/thư mục chưa tối ưu: thiếu phân loại rõ ràng (thư mục đưa lên đầu), chưa hỗ trợ sắp xếp theo cột (Name, Date, Size), thiếu nút lên thư mục cha (Up Directory).

Mục tiêu: Xây dựng mới hoàn toàn component `ui/filechooser` trong Typstify thay thế cho `github.com/oligo/gioview/explorer` để mang lại trải nghiệm duyệt và quản lý file/thư mục hiện đại, trực quan, chính xác trên Windows, macOS và Linux.

---

## Kiến trúc và Các Module đã thực hiện

### 1. `locations.go`, `locations_windows.go`, `locations_unix.go`
- **Phát hiện ổ đĩa Windows**: Dùng `golang.org/x/sys/windows` (`GetLogicalDrives`, `GetVolumeInformation`, `GetDriveType`) quét toàn bộ ổ logic `A:` đến `Z:`.
- **Định dạng nhãn ổ đĩa**: `Local Disk (C:)`, `Data (D:)`, `New Volume (E:)`, `Google Drive (G:)`... chấm dứt triệt để lỗi hiển thị `\`.
- **Favorites**: Tự động nhận diện các thư mục khả dụng: `Home`, `Desktop`, `Documents`, `Downloads`.

### 2. `breadcrumbs.go` & `navigation.go`
- **Clickable Breadcrumbs**: Phân đoạn đường dẫn theo từng cấp (`D:` > `code` > `typstify`), click nhảy trực tiếp đến thư mục cha/con tương ứng.
- **Direct Address Bar Mode**: Cho phép nhập/dán đường dẫn trực tiếp (`D:\code\typstify`) và nhấn Enter hoặc nút Go để chuyển thư mục ngay lập tức.
- **Điều hướng lịch sử**: Nút Back (`←`), Forward (`→`), Up Directory (`↑`), Refresh (`↻`), ô tìm kiếm file/folder (`🔍 Search`).

### 3. `entry_list.go`
- **Quy tắc phân loại**: Thư mục (Folders) **luôn luôn xếp trước Files** trong mọi chế độ sắp xếp.
- **Sắp xếp đa cột**: Click vào header cột để đảo chiều sort theo `Name` (A-Z / Z-A), `Date Modified` (Mới - Cũ), `Size` (Lớn - Nhỏ).
- **Icon trực quan**: Màu sắc riêng biệt cho từng loại tệp (Folder vàng, .typ cyan, .pdf đỏ, image xanh lá, font tím, code xanh dương, generic xám).
- **Dung lượng thân thiện**: Định dạng `humanize.Bytes` (`12.5 KB`, `1.4 MB`), thư mục hiển thị `--`.
- **Tương tác**: Single-click chọn, Double-click mở folder hoặc xác nhận chọn file, multi-select khi chọn nhiều file.

### 4. `bottom_bar.go`
- Nút `+ New Folder` tạo thư mục inline ngay tại chỗ.
- Hiển thị tên file/thư mục đang chọn.
- Ô nhập tên tệp trong chế độ Save As.
- Nút `Cancel` và `Confirm` (`Open` / `Select Folder` / `Save`).

### 5. `dialog.go` & `file_chooser.go`
- View modal dialog chuẩn `view.View` của Gio UI.
- API công khai tương thích 100%: `ChooseFolder()`, `ChooseFile()`, `ChooseFiles()`, `CreateFile()`.
- Hỗ trợ phím tắt: `Enter` (xác nhận), `Escape` (hủy), `F5` (làm mới).

---

## Callsites Migration
Chuyển đổi toàn bộ sang `looz.ws/typstify/ui/filechooser`:
1. `ui/ui.go`
2. `ui/welcome.go`
3. `ui/home.go`
4. `ui/commandbar/commandbar.go`
5. `ui/dialog/create_project.go`
6. `ui/navpanel/menu_panel.go`
7. `ui/settings/fonts.go`

---

## Kiểm chứng (Verification)
1. Unit tests `ui/filechooser`: `TestDetectVolumes`, `TestGetFavorites`, `TestParseBreadcrumbs`, `TestHistoryStack`, `TestReadDirectoryAndSorting`, `TestExtensionFilter` đạt **PASS 100%**.
2. Toàn bộ test suite `service/`, `editor/`, `typst/`, `utils/`, `ui/` đạt **PASS 100%**.
3. `go build -o typstify.exe .` và `go build ./cmd/typstify-server` build thành công hoàn hảo.
