import { describe, expect, it } from 'vitest'
import {
  convertMarkdownToTypst,
  parseFrontmatter,
  transformInlineFormatting,
} from './markdownToTypst'

describe('markdownToTypst', () => {
  it('parses YAML frontmatter', () => {
    const md = `---
title: "Sách Cờ Vua Dương Sinh"
subtitle: "Tập 1: Chiến thuật Cơ bản"
author: "Dương Sinh"
layout: "book"
---

# Chương 1: Đòn Ghim
Nội dung chương 1...`

    const { meta, content } = parseFrontmatter(md)
    expect(meta.title).toBe('Sách Cờ Vua Dương Sinh')
    expect(meta.subtitle).toBe('Tập 1: Chiến thuật Cơ bản')
    expect(meta.author).toBe('Dương Sinh')
    expect(meta.layout).toBe('book')
    expect(content).toContain('# Chương 1: Đòn Ghim')
  })

  it('converts headings, lists, bold, italic, links', () => {
    const md = `# Tiêu đề Lớn
## Tiêu đề Nhỏ

Đây là văn bản **in đậm**, _in nghiêng_, và [liên kết](https://example.com).

- Mục 1
- Mục 2
1. Bước 1
2. Bước 2`

    const typ = convertMarkdownToTypst(md, { template: 'none' })
    expect(typ).toContain('= Tiêu đề Lớn')
    expect(typ).toContain('== Tiêu đề Nhỏ')
    expect(typ).toContain('*in đậm*')
    expect(typ).toContain('#link("https://example.com")[liên kết]')
    expect(typ).toContain('- Mục 1')
    expect(typ).toContain('+ Bước 1')
  })

  it('converts chess FEN code block into #teaching-diagram', () => {
    const md = `\`\`\`fen
r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5
title: "Thế trận thực chiến"
turn: b
caption: "Đen tạo sức ép lên c3"
arrows: "b4c3, g8f6"
\`\`\``

    const typ = convertMarkdownToTypst(md, { template: 'none' })
    expect(typ).toContain('#import "@local/chessbook:0.1.0": *')
    expect(typ).toContain('#teaching-diagram(')
    expect(typ).toContain('"r1bqk2r/pppp1ppp/2n5/4p3/1bB1P3/2NP1N2/PPP2PPP/R1BQK2R b KQkq - 0 5"')
    expect(typ).toContain('title: "Thế trận thực chiến"')
    expect(typ).toContain('turn: "b"')
    expect(typ).toContain('caption: "Đen tạo sức ép lên c3"')
    expect(typ).toContain('arrows: ("b4c3", "g8f6")')
  })

  it('converts PGN game code block into magazine game table', () => {
    const md = `\`\`\`pgn
[Event "FIDE World Championship 2024"]
[Site "Singapore"]
[Date "2024.11.25"]
[White "Ding Liren"]
[Black "Gukesh D"]
[Result "1-0"]
[ECO "C58"]

1. e4 e5 2. Nf3 Nc6 3. Bc4 Nf6 1-0
\`\`\``

    const typ = convertMarkdownToTypst(md, { template: 'none' })
    expect(typ).toContain('#game-header(')
    expect(typ).toContain('white: "Ding Liren"')
    expect(typ).toContain('black: "Gukesh D"')
    expect(typ).toContain('result: "1 - 0"')
    expect(typ).toContain('#table(')
    expect(typ).toContain('table.header([*\\#*], [*Trắng*], [*Đen*])')
  })

  it('converts GFM Markdown table with escaped hash in header', () => {
    const md = `| # | Khai cuộc | Đánh giá |
|---|-----------|----------|
| 1 | Ruy Lopez | Cân bằng |
| 2 | Sicilian  | Sắc nét  |`

    const typ = convertMarkdownToTypst(md, { template: 'none' })
    expect(typ).toContain('#table(')
    expect(typ).toContain('table.header([*\\#*], [*Khai cuộc*], [*Đánh giá*])')
    expect(typ).toContain('[1]')
    expect(typ).toContain('[Ruy Lopez]')
  })

  it('converts Callouts and Quotes to #concept-box, #instructor-note, and #chess-quote', () => {
    const md = `> [!CONCEPT] Khái niệm Đòn Ghim
> Đòn ghim là một chiến thuật cơ bản trong cờ vua.

> [!NOTE]
> HLV cần nhắc học viên quan sát kỹ hàng ngang mở.

> "Cờ vua là cuộc chiến của trí tuệ." -- Garry Kasparov`

    const typ = convertMarkdownToTypst(md, { template: 'none' })
    expect(typ).toContain('#concept-box(title: "Khái niệm Đòn Ghim")')
    expect(typ).toContain('#instructor-note')
    expect(typ).toContain('#chess-quote(author: "Garry Kasparov")')
  })

  it('converts figurines and NAG annotations in text', () => {
    const text = 'Trắng chơi 1. :wN:f3! chiếm trung tâm (±) và sau đó :wB:c4 $14'
    const res = transformInlineFormatting(text, true)
    expect(res).toContain('#wN')
    expect(res).toContain('#wB')
    expect(res).toContain('#nag("16")')
    expect(res).toContain('#nag("14")')
  })

  it('generates full book document with frontmatter and import at line 1', () => {
    const md = `---
title: "Bí quyết Chiến thuật Cờ vua"
subtitle: "Giáo trình Nâng cao"
author: "Dương Sinh"
layout: "book"
---

# Bài 1: Đòn Tấn công Đôi

\`\`\`fen
r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6
title: "Ví dụ đòn đánh đôi"
\`\`\`
`

    const typ = convertMarkdownToTypst(md)
    expect(typ.startsWith('#import "@local/chessbook:0.1.0": *')).toBe(true)
    expect(typ).toContain('chess-book-init.with(')
    expect(typ).toContain('title: "Bí quyết Chiến thuật Cờ vua"')
    expect(typ).toContain('paper-size: "16x24"')
    expect(typ).toContain('= Bài 1: Đòn Tấn công Đôi')
    expect(typ).toContain('#teaching-diagram(')
  })
})
