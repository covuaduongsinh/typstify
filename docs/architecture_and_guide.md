# Hướng dẫn & Tài liệu Kiến trúc Chi tiết Phần mềm Typstify

## 1. Giới thiệu Tổng quan (Overview)

**Typstify** là một phần mềm soạn thảo và môi trường phát triển tích hợp (IDE / Desktop Editor) đa nền tảng (Windows, macOS, Linux) dành riêng cho **Typst** – ngôn ngữ định dạng và dàn trang tài liệu khoa học thế hệ mới được thiết kế để thay thế LaTeX với tốc độ biên dịch tức thời và cú pháp trực quan, dễ đọc hơn.

### Các điểm nổi bật:
- **Hiệu năng cao & Tiêu thụ tài nguyên tối ưu:** Viết hoàn toàn bằng ngôn ngữ Go và framework GUI **Gio** (`gioui.org`), trực tiếp kết xuất qua GPU (Direct3D 11 trên Windows, Metal trên macOS, OpenGL/Vulkan trên Linux).
- **Trải nghiệm soạn thảo thời gian thực (Live Preview):** Tích hợp sâu với **Tinymist LSP**, cập nhật bản xem trước ngay khi người dùng gõ phím mà không bị giật lag.
- **Language Server Protocol (LSP) đầy đủ:** Tự động gợi ý hoàn thành mã (code completion), kiểm tra lỗi cú pháp (diagnostics), xem tài liệu khi rê chuột (hover doc), cây phân cấp tài liệu (document outline/symbols).
- **AI Agent & MCP Integration:** Tích hợp giao thức **ACP** (Agent Client Protocol) và **MCP** (Model Context Protocol), cho phép trợ lý AI (như Claude Code, Cline, v.v.) trực tiếp đọc ngữ cảnh tài liệu, tra cứu thư viện, sửa lỗi và soạn thảo cùng người dùng.
- **Quản lý Thư viện / Package (Tpix Ecosystem):** Tích hợp Typst Package Index (Tpix) giúp tìm kiếm, cài đặt và quản lý các template, thư viện Typst một cách dễ dàng.
- **Xuất bản đa định dạng:** Hỗ trợ xuất tài liệu sang PDF, PNG, SVG với độ phân giải cao.

---

## 2. Bản đồ Cấu trúc Dự án (Codebase Map)

| Thư mục / File | Vai trò & Chức năng chính |
| :--- | :--- |
| [`app.go`](file:///D:/code/typstify/app.go) | Entry point của ứng dụng. Khởi tạo `ServiceFacade`, cấu hình Logger, khởi chạy vòng lặp GUI `ui.Loop(ctx)`. |
| [`editor/`](file:///D:/code/typstify/editor) | Bộ lõi trình soạn thảo văn bản dựa trên `gvcode`: hỗ trợ tô màu cú pháp (Chroma), đánh số dòng (ruler), tự động lưu (autosaver), tìm kiếm (search), hiển thị diff. |
| [`lsp/`](file:///D:/code/typstify/lsp) | Giao tiếp Language Server Protocol với `tinymist`: quản lý tiến trình nền, luồng chẩn đoán lỗi, completion provider, preview service. |
| [`typst/`](file:///D:/code/typstify/typst) | Trình bọc lệnh gọi thực thi `typst` CLI: biên dịch tài liệu, định cấu hình compiler options, xuất file PDF/SVG/PNG. |
| [`service/`](file:///D:/code/typstify/service) | Tầng kiến trúc Service Facade kết nối UI với các dịch vụ lõi: Event Bus, Workspace Manager, Settings Manager (dùng `bbolt` DB), Window Manager. |
| [`ui/`](file:///D:/code/typstify/ui) | Toàn bộ giao diện người dùng viết bằng Gio UI: thanh điều hướng (`navpanel`), cây thư mục (`filetree`), trình xem ảnh/preview (`viewer`), hộp thoại (`dialog`), quản lý gói (`pkgmgmt`), trợ lý AI (`assistant`). |
| [`agent/`](file:///D:/code/typstify/agent) | Hệ thống máy chủ MCP nhúng (Built-in MCP Server) và quản lý phiên ACP (Agent Client Protocol) kết nối với các AI agent bên ngoài. |
| [`i18n/`](file:///D:/code/typstify/i18n) | Bộ dịch đa ngôn ngữ cho giao diện (Tiếng Anh, Tiếng Trung, Tiếng Việt, v.v.). |
| [`fonts/`](file:///D:/code/typstify/fonts) | Bộ phông chữ nhúng sẵn: Hack, RobotoMono, NotoSansSC, NotoSansMath, NotoColorEmoji. |

---

## 3. Kiến trúc Chi tiết (Architecture Deep Dive)

### 3.1. Vòng lặp Giao diện (Gio Immediate Mode GUI)
- Ứng dụng hoạt động theo cơ chế **Immediate Mode** của `gioui.org`. Mỗi khi có sự kiện (chuột, bàn phím, file thay đổi, dữ liệu LSP gửi về), phương thức `Layout()` sẽ được gọi để tái tạo các thao tác vẽ (Ops buffer) và gửi tới GPU.
- Điều này giúp giao diện mượt mà, phản hồi ngay tức thì với chi phí bộ nhớ thấp.

### 3.2. Cơ chế Live Preview & Đồng bộ LSP
```
[User gõ mã trong Editor] 
         │ (Cập nhật buffer)
         ▼
[lsp.Document.DidChange] ──> [Tinymist LSP Server] ──> (Biên dịch theo luồng)
                                     │
                                     ▼
                            [Live Preview Stream]
                                     │
                                     ▼
                         [ui/preview & ui/viewer]
                                     │
                                     ▼
                          [Render lên màn hình]
```

### 3.3. Tích hợp AI Agent (ACP & MCP)
- Typstify khởi tạo một máy chủ **MCP nội bộ** (Local MCP Server) cung cấp các Tools và Resources đặc thù của Typst (truy vấn tài liệu đang mở, chạy compiler, tìm kiếm package).
- Giao diện chat Assistant sử dụng **ACP Session Manager** để tương tác trực tiếp với các Agent như Claude Code CLI, giúp hỗ trợ viết tài liệu khoa học ngay trong IDE.

---

## 4. Hướng dẫn Chạy & Vận hành (How to Run)

### Yêu cầu tiên quyết (Prerequisites)
1. **Go Toolchain**: Go 1.25 trở lên (Hệ thống hiện tại: Go 1.26.3).
2. **C Compiler (CGO)**: GCC (đã sẵn sàng trên máy).
3. **Typst CLI**: `typst.exe` (đã có tại `C:\Users\duongsinh\.cargo\bin\typst.exe`).
4. **Tinymist LSP**: `tinymist.exe` (đã cài đặt phiên bản 0.15.8 qua WinGet).

### Các lệnh thực thi
- **Chạy trực tiếp từ mã nguồn:**
  ```powershell
  go run .
  ```
- **Biên dịch ra file thực thi (.exe):**
  ```powershell
  go build -o typstify.exe .
  ./typstify.exe
  ```
