# Typstify Chessbook Engine Guide for AI Agents

Typstify includes the built-in chess package `@local/chessbook:0.1.0`.

## ⚠️ Important Rules for AI Assistants

1. **NEVER manually declare `#let` mock definitions** for chessbook functions (e.g. `#let lesson-header`, `#let chess-quote`, `#let instructor-note`, `#let practice-question`, `#let eco-header`, `#let game-header`, `#let puzzle-card`).
2. **ALWAYS include `#import "@local/chessbook:0.1.0": *` at LINE 1 (the very top of the file)** before any chess notation, diagrams, templates, or helpers. Typst executes sequentially top-to-bottom; calling a function before its `#import` causes `unknown variable` compilation errors.
3. In Typst content blocks `[...]`, `#` unconditionally starts a code expression. Literal hash `#` MUST be escaped as `\#` (for example in table headers: `table.header([*\#*], [*Trắng*], [*Đen*])`).

## Available Functions in `@local/chessbook:0.1.0`

### 1. Document Initialization
- `#show: chess-book-init.with(title: "...", subtitle: "...", author: "...", paper-size: "16x24" | "a5" | "a4")`
- `#show: chess-worksheet-init.with(title: "...", subtitle: "...", author: "...", date: "...", paper-size: "a4")`
- `#show: chess-magazine-init.with(magazine-title: "...", issue: "...")`

### 2. Puzzle & Tactics Module (`puzzle.typ`)
- `#puzzle-grid-a4(puzzles: (...))` (Lưới 12 bài tập A4: 3 cột x 4 hàng, tự căn chỉnh `size: 13.5pt` vừa khít 1 trang)
- `#puzzle-grid-16x24(puzzles: (...))` (Lưới 6 bài tập 16x24cm: 2 cột x 3 hàng, `size: 15pt` chuẩn in sách)
- `#puzzle-card(fen, number: 1, title: "...", turn: "w" | "b", difficulty: 1..5, hint: "...", size: 16pt, compact: false, solution: "...", arrows: ())` (Note: both `turn` and `to-move` are accepted)
- `#difficulty-stars(level)`
- `#upside-down-solutions(( "1": "1. e4", "2": "1. d4" ))`
- `#render-puzzle-solutions()`

### 3. ECO & Openings Module (`eco.typ`)
- `#eco-header(code: "C 58", name: "...", subname: "...", intro-moves: "...")`
- `#eco-table(columns-header: ("Trắng", "Đen", "Đánh giá"), rows: (...))`
- `#opening-diagram-box(fen, title: "...", turn: "w", eval-text: "+=", caption: "...", arrows: ())`

### 4. Magazine & Articles Module (`magazine.typ`)
- `#game-header(white: "...", black: "...", event: "...", site: "...", date: "...", result: "1 - 0", eco: "...")`
- `#column-diagram(fen, move-num: "12...", caption: "...", turn: "w", arrows: ())`
- `#chess-quote(author: "Garry Kasparov")[Nội dung trích dẫn...]`

### 5. Courseware & Lesson Plans Module (`courseware.typ`)
- `#lesson-header(lesson-num: 1, title: "...", level: "...", duration: "...", objective: "...")`
- `#concept-box(title: "Khái niệm Then chốt")[Nội dung...]`
- `#teaching-diagram(fen, title: "...", turn: "w", size: 18pt, arrows: (), caption: "...")`
- `#practice-question(number: 1, question: "...", choices: ("A...", "B..."), answer: "...")`
- `#instructor-note[Lưu ý cho HLV...]`

### 6. Figurines & Symbols Module (`symbols.typ`)
- Pieces: `#wK`, `#wQ`, `#wR`, `#wB`, `#wN`, `#wP`, `#bK`, `#bQ`, `#bR`, `#bB`, `#bN`, `#bP`
- NAG Glyphs: `#nag("1")` (!), `#nag("2")` (?), `#nag("3")` (!!), `#nag("4")` (??), `#nag("14")` (⩲), `#nag("16")` (±), `#nag("10")` (=), `#nag("7")` (□)
- Indicators: `#turn-indicator("w")`, `#turn-indicator("b")`, `#note-num(1)`
- Board: `#chess-board(fen, size: 16pt, reverse: false, arrows: ())`
