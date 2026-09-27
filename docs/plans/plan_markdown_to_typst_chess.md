# Kế hoạch Triển khai Tính năng Chuyển đổi Markdown (.md) sang Typst (.typ) Chuyên biệt Cờ Vua & Triển khai VPS Dokploy

## 1. Tổng quan & Mục tiêu

Tính năng **Markdown to Typst Converter (Chuyên biệt Cờ vua)** cho phép người dùng chuyển đổi các tệp hoặc đoạn văn bản Markdown (`.md`) sang định dạng Typst (`.typ`) chuẩn của **Typstify & CLB Cờ vua Dương Sinh**, tự động nhận diện và chuyển hóa các thành phần chuyên biệt của cờ vua:
- **Thế cờ (FEN)** & sơ đồ bàn cờ (`teaching-diagram`, `chess-board`, `opening-diagram-box`).
- **Ván cờ (PGN)** với các biến thể, chú thích, kết quả, định dạng bảng 2 cột tạp chí (`game-header`, `#table(...)`, `#nag`).
- **Bộ bài tập chiến thuật (Puzzles & Tactics)**: Tự động gom nhóm hoặc tạo thẻ bài tập (`puzzle-card`, `puzzle-grid-a4`, `puzzle-grid-16x24`, `upside-down-solutions`).
- **Khai cuộc (ECO)** & Giáo trình (`eco-header`, `lesson-header`, `concept-box`, `instructor-note`).
- **Hộp ghi chú / Callouts** (`> [!NOTE]`, `> [!CONCEPT]`, `> [!IMPORTANT]`, `> [!QUOTE]`).
- **Ký hiệu cờ vua (Figurines & NAGs)**: Tự động chuyển đổi `!`, `?`, `!!`, `??`, `!?`, `?!`, `(±)`, `(∓)`, `(⩲)`, `(=)` sang hàm `#nag(...)` và quân cờ `#wK`, `#wQ`, `#bN`...
- **Bảo đảm chuẩn Typst**: Luôn chèn `#import "@local/chessbook:0.1.0": *` ở dòng 1 và escape các ký tự đặc biệt (`#` -> `\#` trong table/text).

Sau khi hoàn thiện và kiểm thử tự động, mã nguồn sẽ được tự động commit, push lên remote repository và cập nhật lên VPS Dokploy tại `217.15.160.118` (`typst.dsc.edu.vn`).

---

## 2. Thiết kế Kiến trúc & Chi tiết Tính năng

```mermaid
flowchart TD
    MD[File .md / Text Markdown] --> Parser[Markdown Parser & AST Analyzer]
    
    subgraph Engine [Chess Markdown-to-Typst Engine]
        Parser --> Frontmatter[YAML Frontmatter Detector]
        Parser --> FEN[FEN / Diagram Blocks]
        Parser --> PGN[PGN / Game Header & Moves]
        Parser --> Puzzles[Puzzle Cards & Grid Blocks]
        Parser --> ECO[ECO & Lesson Blocks]
        Parser --> Callouts[Callouts & Quotes]
        Parser --> Table[GFM Tables to Typst #table]
        Parser --> Standard[Headings, Lists, Emph, Code]
        
        Frontmatter --> TemplateInit[#show: chess-book-init / worksheet-init]
        FEN --> DiagOut[#teaching-diagram / #chess-board]
        PGN --> GameOut[#game-header + #table / movetext]
        Puzzles --> PuzOut[#puzzle-grid / #puzzle-card]
        ECO --> EcoOut[#eco-header]
        Callouts --> BoxOut[#concept-box / #chess-quote]
        Table --> TypTable[#table(...) with escaped hash]
        Standard --> TypMarkup[= Headings, *bold*, _italic_]
    end

    TemplateInit --> Assembler[Document Assembler & Hoister]
    DiagOut --> Assembler
    GameOut --> Assembler
    PuzOut --> Assembler
    EcoOut --> Assembler
    BoxOut --> Assembler
    TypTable --> Assembler
    TypMarkup --> Assembler

    Assembler --> TypOutput[File .typ hoàn chỉnh với #import chessbook ở Line 1]

    subgraph UI [Giao diện Người dùng]
        TypOutput --> Modal[MarkdownImportModal UI]
        TypOutput --> ContextMenu[FileTree Context Menu: Convert .md -> .typ]
        TypOutput --> NewDoc[NewDocModal: Import Markdown tab]
        TypOutput --> Toolbar[ChessToolbar: Nút chuyển đổi MD]
    end
```

---

## 3. Quy cách Chuyển đổi Markdown Cờ vua (Specification)

### 3.1. Frontmatter (YAML metadata)
```yaml
---
title: "Khai cuộc Thông dụng"
subtitle: "Giáo trình Cờ vua Dương Sinh"
author: "CLB Cờ vua Dương Sinh"
layout: "book" # book (16x24) | worksheet (a4) | magazine (a4)
---
```
👉 **Đầu ra Typst:**
```typst
#import "@local/chessbook:0.1.0": *

#show: chess-book-init.with(
  title: "Khai cuộc Thông dụng",
  subtitle: "Giáo trình Cờ vua Dương Sinh",
  author: "CLB Cờ vua Dương Sinh",
  paper-size: "16x24",
)
```

### 3.2. Thế cờ & Bàn cờ (FEN Block)
````markdown
```fen
r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5
title: "Thế trận thực chiến"
turn: b
caption: "Đen tạo sức ép lên c3"
arrows: "b4c3"
```
````
👉 **Đầu ra Typst:**
```typst
#teaching-diagram(
  "r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5",
  title: "Thế trận thực chiến",
  turn: "b",
  caption: "Đen tạo sức ép lên c3",
  arrows: ("b4c3",),
)
```

### 3.3. Ván cờ PGN (PGN Block)
````markdown
```pgn
[Event "FIDE World Championship 2024"]
[Site "Singapore"]
[Date "2024.11.25"]
[White "Ding Liren"]
[Black "Gukesh D"]
[Result "1-0"]
[ECO "C58"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 4. Ng5 d5 5. exd5 Na5 1-0
```
````
👉 **Đầu ra Typst:**
```typst
#game-header(
  white: "Ding Liren",
  black: "Gukesh D",
  event: "FIDE World Championship 2024",
  site: "Singapore",
  date: "2024.11.25",
  result: "1 - 0",
  eco: "C58",
)

#v(6pt)

#table(
  columns: (22pt, 1fr, 1fr),
  stroke: none,
  inset: (x: 4pt, y: 3pt),
  fill: (col, row) => if calc.odd(row) { rgb("#f8fafc") } else { none },
  table.header([*\#*], [*Trắng*], [*Đen*]),
  [1.], [#strong[e4]], [#strong[e5]],
  [2.], [#strong[Nf3]], [#strong[Nc6]],
  [3.], [#strong[Bc4]], [#strong[Nf6]],
  [4.], [#strong[Ng5]], [#strong[d5]],
  [5.], [#strong[exd5]], [#strong[Na5]],
)
```

### 3.4. Callouts, Khái niệm & Trích dẫn (Quotes)
- `> [!CONCEPT] Đòn ghim tuyệt đối...` 👉 `#concept-box(title: "Khái niệm Then chốt")[Đòn ghim tuyệt đối...]`
- `> [!NOTE] Lưu ý cho HLV...` 👉 `#instructor-note[Lưu ý cho HLV...]`
- `> "Cờ vua là cuộc chiến của trí tuệ." -- Garry Kasparov` 👉 `#chess-quote(author: "Garry Kasparov")[Cờ vua là cuộc chiến của trí tuệ.]`

### 3.5. Bảng Markdown (GFM Tables)
- Bảng Markdown:
```markdown
| STT | Tên Biến thể | Đánh giá |
| --- | ------------ | -------- |
| 1   | Giuoco Piano | Cân bằng |
```
👉 Chuyển thành `#table(columns: (1fr, 1fr, 1fr), table.header([*STT*], [*Tên Biến thể*], [*Đánh giá*]), ...)` với ký tự `#` được escape an toàn.

---

## 4. Các tệp cần thêm mới & chỉnh sửa

### 4.1. Core Library:
1. `web/src/lib/markdownToTypst.ts` **[NEW]**:
   - Bộ phân tích cú pháp Markdown sang Typst hỗ trợ đầy đủ các khối cờ vua và markup chuẩn.
2. `web/src/lib/markdownToTypst.test.ts` **[NEW]**:
   - Bộ kiểm thử đơn vị toàn diện (Headings, Tables, FEN blocks, PGN games, Puzzles, Callouts, NAG symbols, Escape rules).

### 4.2. Giao diện Người dùng (UI):
3. `web/src/components/MarkdownImportModal.tsx` **[NEW]**:
   - Modal nhập Markdown với tính năng: Paste trực tiếp, Upload file `.md`, chọn file `.md` trong dự án, cấu hình loại văn bản xuất ra, live preview Typst và nút tạo file/chèn vào editor.
4. `web/src/components/FileTree.tsx` **[MODIFY]**:
   - Thêm nút / menu chuột phải: "Chuyển thành Typst (.typ)" khi chọn file có đuôi `.md`.
5. `web/src/components/ChessToolbar.tsx` **[MODIFY]**:
   - Thêm nút "Nhập Markdown" trong nhóm công cụ nhập liệu CSDL/PGN.
6. `web/src/components/NewDocModal.tsx` **[MODIFY]**:
   - Thêm tab hoặc tùy chọn "Chuyển đổi từ file Markdown (.md)".
7. `web/src/components/Workspace.tsx` **[MODIFY]**:
   - Tích hợp `MarkdownImportModal`, kết nối với phím tắt và menu ngữ cảnh.

---

## 5. Kế hoạch Kiểm thử & Triển khai VPS Dokploy

### 5.1. Kiểm thử Tự động & Build cục bộ
1. `cd web && npm test` — Đảm bảo toàn bộ test case mới và cũ đều pass 100%.
2. `cd web && npm run build` — Đảm bảo TypeScript compilation và Vite build thành công không có lỗi type.
3. `git status`, `git add .`, `git commit -m "feat(markdown): add markdown to typst converter with chess diagram & PGN support"`, `git push origin main`.

### 5.2. Triển khai lên VPS Dokploy
- **VPS Target**: IP `217.15.160.118`, Port `22`, User `root`, SSH Private Key: `C:\Users\duongsinh\.ssh\id_ed25519`.
- **Dokploy Panel**: `https://dokploy.dsc.edu.vn`
- **Application Domain**: `typst.dsc.edu.vn`
- **Dokploy API Key**: `typst_appoTVulVTtjNGQppUWoRcWJZVcqzmGmqlzOfvyRaiDdWYhoFrVsWOWKItbqLxleGtj`
- Thực hiện trigger redeploy Dokploy thông qua API hoặc SSH pull & rebuild `docker compose up -d --build` trên VPS, sau đó kiểm tra health check qua domain `https://typst.dsc.edu.vn`.

---

## 6. Verification Plan
- **Unit Tests**: Kiểm tra các trường hợp chuyển đổi: Markdown thuần, Markdown có FEN, Markdown có PGN, Markdown có bảng, YAML Frontmatter.
- **E2E / UI Test**: Mở modal, nhập Markdown có cờ vua, tạo file `.typ` mới, kiểm tra preview Typst hiển thị chính xác bàn cờ và ván cờ.
- **Production Check**: Truy cập `https://typst.dsc.edu.vn`, xác nhận ứng dụng chạy phiên bản mới với đầy đủ tính năng.
