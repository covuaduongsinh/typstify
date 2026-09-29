# Kế hoạch: đóng gói Typstify desktop thành bộ cài Windows

## Bối cảnh

Sau nhiều đợt cập nhật (preview, sync desktop–web, chessbook…), người dùng muốn có **bản desktop cài được** trên máy Windows của họ. Hiện repo chỉ có `go run .` và README nói "dùng `gogio`, cần CGO"; chưa có script đóng gói, chưa có installer. Bản desktop còn cần các thứ ngoài binary Go:

- `typst.exe` (v0.15.1) và `tinymist.exe` (v0.15.8) — ghim phiên bản như `Dockerfile`. `utils.LookupExecutable` (`utils/executable.go`) đã tìm trong thư mục exe và `bin/` cạnh exe, nên chỉ cần đặt chúng cạnh app, không phải sửa code.
- Thư viện `@local/chessbook:0.1.0` → `%APPDATA%\typst\packages\local\chessbook\0.1.0` (logic của `scripts/install-chessbook.ps1`).
- Phụ thuộc `@preview/board-n-pieces` 0.9.0 (Dockerfile bundle sẵn để chạy offline) và font Roboto/Noto in ấn.

Đích: một file `Typstify-Setup-<version>.exe` (Inno Setup) cài đủ mọi thứ, chạy offline. Nếu máy không có Inno Setup thì dùng phương án dự phòng: zip portable + `install.ps1`.

## Cách làm

Một script duy nhất `scripts/package-desktop.ps1` (thư mục `dist/` được ignore), chia bước:

1. **Kiểm tiền đề**: Go 1.25, GCC (CGO), `gogio` (nếu thiếu: `go install gioui.org/cmd/gogio@latest`), Inno Setup `ISCC.exe` (tuỳ chọn).
2. **Build app**: `gogio -target windows -arch amd64 -icon version/appicon.png -o dist/stage/Typstify.exe .` với `-ldflags "-X looz.ws/typstify/version.BinVersion=<ver> -X ...BuildTime=<unix> -X ...BuildGoVersion=<go ver>"` (biến ở `version/buildinfo.go`). Phiên bản lấy từ tham số `-Version`, mặc định `git describe --tags --always`.
3. **Tải công cụ ngoài** vào `dist/stage/bin/`: `typst-x86_64-pc-windows-msvc.zip` và `tinymist-windows-x64.exe` từ GitHub Releases, kiểm SHA-256 (hằng số ghim trong script, cập nhật cùng `Dockerfile` khi nâng phiên bản). Cache tại `dist/cache/` để chạy lại không tải lại.
4. **Bundle chessbook**: copy `chessbook/lib` → `dist/stage/chessbook-lib`; tải `board-n-pieces-0.9.0.tar.gz` từ `packages.typst.org` → `dist/stage/packages/preview/board-n-pieces/0.9.0`. Font Roboto/Noto: tải (nếu có nguồn ổn định) vào `dist/stage/fonts`; nếu không thì bỏ qua và ghi cảnh báo, giống hướng dẫn hiện tại.
5. **Đóng gói**:
   - Có `ISCC.exe`: sinh `scripts/typstify.iss` → `dist/Typstify-Setup-<ver>.exe`. Cài vào `%LOCALAPPDATA%\Programs\Typstify` (không cần admin), tạo shortcut Start Menu/Desktop, copy chessbook + board-n-pieces vào `%APPDATA%\typst\packages\{local\chessbook\0.1.0, preview\board-n-pieces\0.9.0}`, cài font vào thư mục font người dùng, có uninstaller.
   - Không có: `dist/Typstify-portable-<ver>.zip` kèm `install.ps1` làm đúng các bước copy trên.
6. **Xác minh khói**: chạy `Typstify.exe` vài giây rồi thoát để chắc không crash lúc khởi động; chạy `bin\typst.exe --version` và biên dịch một demo `chessbook` bằng `typst` đã bundle (offline).

## File sẽ thay đổi

- Mới: `scripts/package-desktop.ps1`, `scripts/typstify.iss` (template Inno), `docs/plans/plan_desktop_installer.md` (bản sao kế hoạch này, theo yêu cầu đặt ở `docs/plans`).
- Sửa: `.gitignore` (thêm `dist/`), `README.md` mục Build, `docs/TECH.md`/`docs/ARCHITECTURE.md` (dòng về gogio → trỏ script), `docs/HISTORY.md` (thêm mục), `CLAUDE.md` + `AGENTS.md` (mục 3 lệnh build, sửa cả hai file cho giống nhau).
- Không đụng code Go/`service/` trừ khi bước 6 phát hiện lỗi thật.

## Rủi ro đã biết

- Chưa biết máy có `gogio`, GCC, Inno Setup hay chưa → script tự phát hiện, tự cài `gogio`, và có đường dự phòng zip; chỉ báo người dùng nếu thiếu GCC.
- Checksum SHA-256 của bản Windows chưa có sẵn → lấy từ file `.sha256`/release notes lúc chạy, ghi vào script, báo rõ nếu nguồn không cung cấp.
- Bản đóng gói chưa ký số nên Windows SmartScreen có thể cảnh báo; ghi trong tài liệu.
- Bản hiện có `bin\typstify.exe` ở repo là artefact cũ, không dùng làm nguồn.

## Kiểm chứng

- `pwsh scripts/package-desktop.ps1 -Version 0.1.0` chạy hết không lỗi, sinh file trong `dist/`.
- Cài thử installer vào thư mục tạm/silent (`/VERYSILENT /DIR=...`), xác nhận có `Typstify.exe`, `bin\typst.exe`, `bin\tinymist.exe`, và chessbook trong `%APPDATA%\typst\packages`.
- Khởi động app đã cài, mở một demo chessbook, thấy preview biên dịch được.
- `go vet ./...` và `go test ./...` không đổi kết quả (không sửa code Go).

## Việc làm ngay sau khi kế hoạch được duyệt

Copy file này vào `docs/plans/plan_desktop_installer.md`, rồi thực thi theo thứ tự trên và báo kết quả cùng đường dẫn file cài đặt.
