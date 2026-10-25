# Kế hoạch chi tiết — Giai đoạn 3: Cộng tác thời gian thực

> Thuộc [plan_typst_app_parity_2026-10.md](plan_typst_app_parity_2026-10.md). Ngày lập: 2026-10-25.

## Mục tiêu

Nhiều người cùng sửa một file `.typ` và thấy thay đổi của nhau ngay lập tức, kèm con trỏ và tên người đang sửa. Có thể bình luận theo luồng, neo vào đoạn văn bản. Mất mạng vẫn sửa tiếp được, có mạng lại thì tự gộp. Vai trò ở Giai đoạn 1 được máy chủ kiểm tra thật: editor sửa chữ, commenter chỉ bình luận, viewer chỉ xem.

## Phạm vi

- **3a (giai đoạn này):** cộng tác trên project đang mở của máy chủ. Gồm CRDT, presence, bình luận, offline, đồng bộ với file trên đĩa.
- **3b (sau):** mỗi người mở một project riêng cùng lúc. Việc này cần tách `ServiceFacade` (tinymist, file watcher, agent) theo project, là thay đổi kiến trúc lớn nên làm riêng. Cho tới lúc đó máy chủ vẫn có một project hiện hành dùng chung, như hiện nay.

## Thiết kế

### CRDT ở client, máy chủ chỉ chuyển tiếp và lưu

Yjs chạy trong trình duyệt. Máy chủ Go không giữ Y.Doc: nó chuyển tiếp update giữa các client trong cùng "phòng" (một file) và ghi lại update vào log. Không cần cài Node hay bản Yjs cho Go.

- Mỗi file có hai tài liệu Yjs, gọi là kênh:
  - **text**: `Y.Text("content")`;
  - **comments**: `Y.Map("threads")`, neo bằng `Y.RelativePosition` vào văn bản.

  Tách kênh để máy chủ phân quyền được mà không phải hiểu nội dung update: update trên kênh text cần vai trò editor, update trên kênh comments chỉ cần commenter.
- **Khung tin nhắn** (binary): `[loại][kênh][payload]`.

  | Loại | Hướng | Nội dung |
  |---|---|---|
  | 0 update | hai chiều | Update Yjs, được ghi log và chuyển tiếp cho các client khác |
  | 1 awareness | hai chiều | Con trỏ, tên, màu. Chỉ chuyển tiếp, không lưu |
  | 2 control | server → client | JSON: `hello` (vai trò, có phải leader, epoch, danh sách người), `peers`, `leader`, `compact`, `external` |
  | 3 synced | server → client | Đã phát lại hết log |
  | 4 compact | leader → server | Trạng thái đầy đủ thay cho log |
  | 5 save | leader → server | Nội dung văn bản để ghi xuống đĩa |
- **Leader:** máy chủ chọn editor vào phòng sớm nhất làm leader. Leader:
  - ghi file xuống đĩa (debounce 1,5 giây, ghi nguyên tử, qua `resolveInRoot`);
  - gửi trạng thái nén khi log dài;
  - áp thay đổi từ bên ngoài vào Y.Text (agent AI sửa file, Dropbox, desktop).
- **Khởi tạo phòng:** khi chưa có log, máy chủ tự sinh update Yjs "chèn toàn bộ nội dung file", nên không có chuyện hai client cùng chèn rồi bị nhân đôi văn bản. Encoder Go được kiểm tra bằng cách so từng byte với update do chính Yjs tạo ra.
- **Thay đổi từ bên ngoài:** máy chủ nhớ hash của lần ghi gần nhất. Khi file trên đĩa khác hash đó, máy chủ báo `external` cho leader (kiểm tra lúc mở phòng và định kỳ 2 giây). Leader đọc file rồi áp diff tối thiểu (tiền tố và hậu tố chung) vào Y.Text.
- **Epoch:** mỗi lần log được tạo mới thì sinh một epoch. Client lưu epoch cùng bản IndexedDB và xoá bản cục bộ khi epoch đổi, để tránh gộp hai lịch sử độc lập (nhân đôi văn bản).
- **Offline:** dùng `y-indexeddb` theo phòng. Khi nối lại, client chỉ gửi phần máy chủ chưa có, tính bằng state vector của log vừa nhận.
- **Lưu trữ:** `<storage>/collab/<hash project>/<hash file>.{text,comments}.ylog` cùng `meta.json` (đường dẫn, epoch, hash đĩa). Log được nén khi vượt 256 mục hoặc 1 MiB.

### Phân quyền

- Route `GET /ws/collab?path=` đăng ký bằng `handleRole(..., RoleViewer)`.
- Vai trò được kiểm tra lại ở mỗi tin nhắn, giống `/ws/lsp`:
  - update kênh text, compact, save: editor;
  - update kênh comments: commenter;
  - awareness: viewer.

  Tin nhắn vượt quyền thì bị bỏ, còn người bị thu hồi quyền thì bị ngắt kết nối.

### Web

| Tệp | Nội dung |
|---|---|
| `web/src/lib/collab/protocol.ts` | Mã hoá và giải mã khung tin nhắn |
| `web/src/lib/collab/provider.ts` | `CollabProvider`: WebSocket tự nối lại, hai Y.Doc, awareness, IndexedDB, nhiệm vụ leader |
| `web/src/lib/collab/textDiff.ts` | Áp diff tối thiểu vào Y.Text |
| `web/src/lib/collab/comments.ts` | Mô hình bình luận: tạo luồng, trả lời, đánh dấu xong, xoá, vị trí hiện tại |
| `Editor.tsx` | Prop `collab`: `yCollab` (con trỏ người khác, undo riêng từng người), tô sáng đoạn có bình luận |
| `CommentsPanel.tsx` | Bảng bình luận ở panel phải: danh sách luồng, trả lời, xong, nhắc `@tên` |
| `Workspace.tsx` | Tạo provider cho file `.typ` đang mở, hiện avatar người đang cùng sửa, trạng thái đồng bộ thay cho "chưa lưu" |

Thư viện: `yjs`, `y-codemirror.next`, `y-protocols` (awareness), `y-indexeddb`, `lib0`.

## Giới hạn đã biết

- Đổi tên hoặc xoá file làm mất lịch sử cộng tác và bình luận của file đó (log gắn với đường dẫn).
- LSP (tinymist) vẫn dùng chung một tiến trình cho máy chủ. Mỗi client gửi nội dung đã hội tụ nên kết quả như nhau.
- File không phải `.typ` vẫn sửa theo cách cũ (lưu thủ công).

## Thay đổi so với thiết kế ban đầu

- **Sửa từ bên ngoài không còn ép văn bản theo đĩa.**
  - Máy chủ giữ `base.txt`, tức nội dung đĩa ở lần đồng bộ trước, và gửi `{base, text}`.
  - Client chỉ áp phần thay đổi `base → text`. Nếu chỗ đó trùng với chỗ đang sửa thì hỏi người dùng (`rebaseChange`).
  - Lý do: chạy thật cho thấy cách cũ xoá mất sửa đổi ngoại tuyến.
- **Client gửi kèm `project`.** Máy chủ từ chối nếu đang mở project khác.

## Kiểm chứng

- Go:
  - encoder update Yjs khớp từng byte với Yjs;
  - phân quyền theo kênh, chọn lại leader khi leader rời;
  - nén log, phát hiện thay đổi trên đĩa;
  - khởi tạo phòng từ file.
- Web (vitest, Yjs thật):
  - khung tin nhắn, diff văn bản;
  - bình luận neo đúng chỗ sau khi văn bản bị sửa phía trước;
  - hai provider giả lập hội tụ.
- Chạy thật: hai tab trong Chrome với hai tài khoản (editor và viewer/commenter). Cần thấy:
  - gõ ở tab này thì tab kia thấy ngay, kèm con trỏ có tên;
  - viewer không sửa được;
  - commenter thêm được bình luận;
  - ngắt mạng một tab, sửa, nối lại thì hội tụ;
  - sửa file trên đĩa thì editor cập nhật.
