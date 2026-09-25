// ============================================================================
// TYPSTIFY CHESS ENGINE - THEME (màu & font dùng chung)
// Nhận diện thương hiệu Dương Sinh: navy #2B3990 + gold. Mọi module lấy màu
// và font từ đây, nên đổi bộ nhận diện chỉ cần sửa một file.
// ============================================================================

// --- Màu ---
#let ds-brand = rgb("#2B3990")        // navy chủ đạo
#let ds-brand-dark = rgb("#1F2A6E")   // navy đậm (tiêu đề)
#let ds-brand-light = rgb("#5A67A8")  // navy nhạt (chữ phụ nổi bật)
#let ds-brand-soft = rgb("#EEF0F8")   // nền nhạt
#let ds-brand-border = rgb("#C9CFE8") // viền nhạt
#let ds-gold = rgb("#C9A227")         // gold nhấn
#let ds-gold-soft = rgb("#FBF6E6")    // nền gold nhạt
#let ds-ink = rgb("#1C2140")          // chữ chính
#let ds-text = rgb("#2D3748")         // chữ thường
#let ds-muted = rgb("#5D6384")        // chữ phụ
#let ds-subtle = rgb("#A0AEC0")       // chữ rất nhạt
#let ds-line = rgb("#D5D9EA")         // đường kẻ
#let ds-paper = rgb("#FFFFFF")
#let ds-board-light = rgb("#FFFFFF")  // ô trắng bàn cờ in sách
#let ds-board-dark = rgb("#D9DDEF")   // ô đen bàn cờ in sách (in được đơn sắc)
#let ds-warn = rgb("#C05621")         // ghi chú HLV
#let ds-warn-soft = rgb("#FFFAF0")

// --- Font ---
// Chỉ liệt kê font có sẵn trong image máy chủ (Roboto, Noto Serif), nên bản
// web biên dịch không cảnh báo. Trên máy cá nhân cần cài hai font miễn phí
// này (xem docs/CHESS_STUDIO_GUIDE.md); thiếu thì Typst tự dùng font dự
// phòng. Ký hiệu quân cờ ♔…♟ được Typst tự lấy từ font có glyph đó.
#let font-serif = ("Noto Serif",)
#let font-sans = ("Roboto",)
