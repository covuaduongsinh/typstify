// =========================================================================
// Tái tạo trang C58 (tr. 414-415) - ChessInformator: Enciclopedia de
// Aperturas de Ajedrez C, 2006 - phong cách bảng ECO / Chess Informant.
//
// Chú thích (1-37, 45-50) dàn trang qua staunton's notation()/game() --
// KHÔNG xác thực luật cờ (notation() chỉ là bộ chuyển đổi văn bản thuần),
// chỉ để có ký hiệu hình tượng quân cờ + NAG (∓/±/∞/=) + biến thể lồng
// tự động, thay cho việc tự gõ tay ký hiệu Unicode và tự escape ngoặc.
// =========================================================================

#counter(page).update(414)

#set page(
  width: 16.5cm,
  height: 23.5cm,
  margin: (x: 1.4cm, y: 0.8cm),
  header: context [
    #let is-odd = calc.odd(counter(page).get().first())
    #grid(
      columns: (1fr, 1fr),
      align: (left + horizon, right + horizon),
      if is-odd [
        #text(9pt, weight: "bold")[C 58]
      ] else [
        #text(9pt)[#counter(page).display()]
      ],
      if is-odd [
        #text(9pt)[#counter(page).display()]
      ] else [
        #text(9pt, weight: "bold")[C 58]
      ]
    )
    #v(2pt)
    #line(length: 100%, stroke: 0.5pt + black)
  ]
)

#set text(font: ("Times New Roman", "Segoe UI Symbol"), size: 8.5pt, lang: "es")
#set par(justify: true, leading: 0.5em, first-line-indent: 0em, spacing: 3pt)
#set block(spacing: 3pt)

// "Chess Merida Unicode" chỉ áp dụng cho ĐÚNG các ký tự quân cờ hình tượng
// (U+2654-265F). KHÔNG đặt font này vào #set text() toàn cục: nó chiếm dụng
// một số mã Unicode khác (∓/±...) bằng glyph riêng của nó, làm sai hình dù
// lớp text PDF vẫn đúng Unicode -- chỉ show rule theo regex mới an toàn.
#show regex("[♔♕♖♗♘♙♚♛♜♝♞♟]"): set text(font: ("Chess Merida Unicode", "Segoe UI Symbol"))

// --- Ký hiệu quân cờ hình tượng cho bảng chính: dùng chess-sym của
// board-n-pieces (rỗng = Trắng, đặc = Đen) ---
#import "@preview/board-n-pieces:0.9.0": chess-sym

#let wN = chess-sym.knight.white
#let wB = chess-sym.bishop.white
#let wR = chess-sym.rook.white
#let wQ = chess-sym.queen.white
#let wK = chess-sym.king.white
#let bN = chess-sym.knight.black
#let bB = chess-sym.bishop.black
#let bR = chess-sym.rook.black
#let bQ = chess-sym.queen.black
#let bK = chess-sym.king.black

// --- Chú thích: dùng staunton để dàn trang movetext (không xác thực luật) ---
#import "@preview/staunton:2.0.0": game, notation

// Số mục chú thích khoanh tròn, tách biệt hẳn khỏi các con số nước đi
// trong nội dung theo sau (dễ lẫn nếu chỉ in đậm thường).
#let note-num(n) = box(
  width: 12pt, height: 12pt, baseline: 2pt,
  stroke: 0.5pt + black, radius: 50%,
)[
  #align(center + horizon)[#text(weight: "bold", size: 6.5pt)[#n]]
]
#let entry(n, body) = block(width: 100%)[#note-num(n) #h(3pt) #body]

// Một "dòng" phân tích PGN -> hình tượng + NAG + biến thể lồng + trích dẫn,
// dàn trang tự động qua staunton. Kết thúc bằng "*" (kết quả không xác định)
// vì đây là đoạn phân tích, không phải ván đấu đầy đủ.
#let ln(pgn) = notation(
  game(pgn + " *"),
  figurine: true, variations: true, nags: true, comments: true,
  bold-mainline: false,
)

// =========================================================================
// BẢNG CHÍNH C 58
// =========================================================================
#block[
  // Bảng chính giữ nguyên cỡ chữ/khoảng cách nhỏ gọn ban đầu, không theo
  // cỡ chữ đã phóng to ở phần chú thích bên dưới.
  #set text(size: 7pt)
  #set par(leading: 0.35em, spacing: 1.5pt)
  #set block(spacing: 1.5pt)

  #grid(
    columns: (auto, 1fr),
    stroke: 0.7pt + black,
    inset: (x: 5pt, y: 3pt),
    grid.cell(fill: black)[#text(fill: white, weight: "bold", size: 11pt)[C 58]],
    grid.cell(align: horizon)[
      #text(weight: "bold")[
        1\. e4 e5 2. #wN f3 #wN c6 3. #wB c4 #wN f6 4. #wN g5 d5 5. ed5 #bN a5
      ]
    ]
  )

  #let cols9 = (auto,) + (1fr,) * 9

  #table(
    columns: cols9,
    stroke: 0.4pt + rgb("#888888"),
    align: (x, y) => if x == 0 { left } else { center },
    inset: (x: 3pt, y: 2.5pt),

    table.header(
      text(weight: "bold")[], text(weight: "bold")[6], text(weight: "bold")[7],
      text(weight: "bold")[8], text(weight: "bold")[9], text(weight: "bold")[10],
      text(weight: "bold")[11], text(weight: "bold")[12], text(weight: "bold")[13],
      text(weight: "bold")[14],
    ),

    // --- Fila 1 ---
    table.cell(rowspan: 2)[*1*],
    [d3], [#wN f3], [#wQ e2#super[1]], [dc4], [h3#super[2]],
    [#wN h2], [dc6], [#wB e3], [fe3],
    [h6], [e4], [#bN c4], [#bB c5], [0-0],
    [c6#super[3]], [e3], [#bB e3], [#bN e4#super[4] #h(4pt) #sym.minus.plus],

    // --- Fila 2 ---
    table.cell(rowspan: 2)[*2*],
    [#wB b5], [#wQ e2], [#wN c3], [#bB d7#super[6]], [0-0],
    [d3], [#wB d2], [#wB c3#super[9]], [#wQ d2],
    [#bB d7], [#bB e7#super[5]], [0-0], [#bR fe8#super[7]], [#bB b4],
    [#bB c3#super[8]], [#bQ d5], [#bN c6#super[10]], [#"="],

    // --- Fila 3 ---
    table.cell(rowspan: 2)[*3*],
    [...], [dc6], [#wQ f3#super[11]], [#wQ a8], [#wQ f3#super[14]],
    [#wQ e2], [d3#super[16]], [c3], [0-0],
    [c6], [bc6], [cb5#super[12]], [#bQ d7#super[13]], [#bB b7],
    [#bB e7#super[15]], [#bN c6#super[17]], [0-0], [#bN d5#super[18] #h(4pt) #sym.plus.minus],

    // --- Fila 4 ---
    table.cell(rowspan: 2)[*4*],
    [...], [...], [...], [#wB d3#super[19]], [0-0#super[21]],
    [#wB f3#super[23]], [#wN e4], [#wQ f5], [#wQ e4],
    [...], [#bQ c7], [#bB e7#super[20]], [0-0#super[22]], [h6#super[24]],
    [#bB f5#super[25]], [#bN e4], [f5#super[26]], [#sym.plus.minus],

    // --- Fila 5 ---
    table.cell(rowspan: 2)[*5*],
    [...], [...], [#wB c6#super[27]], [#wQ c6], [#wQ c4#super[28]],
    [#wN c3], [d3], [#wQ b5],
    [...],
    [...], [...], [#bB e7], [#bN c6], [#bB d7],
    [0-0], [#bB f5#super[29]], [#bR c8], [#bR c5#super[30] #h(4pt) #sym.minus.plus],

    // --- Fila 6 ---
    table.cell(rowspan: 2)[*6*],
    [...], [...], [...], [#wB d3#super[31]], [0-0#super[33]],
    [#wN c3#super[34]], [#wN h3#super[36]], [#wQ g3], [#wB e2],
    [...], [...], [#bR b8], [#bB e7#super[32]], [0-0],
    [h6#super[35]], [#bB g4], [#bQ d7], [#bN e2#super[37] #h(4pt) #sym.minus.plus],
  )
]

#v(6pt)

// =========================================================================
// PHẦN DƯỚI TRANG 414: chú thích 1-6
// =========================================================================
  #entry("1")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 8\"]\n\n8. de4? Nc4 9. Qd4 Nd6 10. Nc3 c6 (10... Nfe4 11. de4 Qe7 12. O-O Ne4 13. Re1 f5 14. Nd2 Qc5 $17 {H. Arens – R. Ludigk, corr. 1996}) 11. O-O cd5 12. e5 Nf5 13. Qd3 Ne4 14. Nd5 Nc5 15. Qd2 Be6 16. c4 Be7 17. b3 O-O $17 {L. Bronstein – Rai. García, Mar del Plata 1969}")
  ]

  #entry("2")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. O-O?! O-O 11. Nfd2 Bg4 12. Qe1 Qd7 13. Nb3 Bf3! 14. Bf4 Qg4 15. g3 Nh5 16. Nc5 Bf4 17. Ne4 Qh3!! {0:1 Field – Tenner, EE.UU. 1923}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. c3?! b5 11. Bd3 Ne4 12. e5 Nf5 13. Qd3 Ne4 14. Nd5 Nc5 15. cb5 O-O 16. O-O Re8 $17 {Grob – Keres, Dresden 1936}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. Bf4?! O-O 11. Nfd2 Bg4 12. Qf1 c6 13. Nc3 Re8 14. Be4 f3 (14. gf3 Bh5 15. b4 Ng6 17. cb5 O-O-O Bc3 18. bc3 Qb6 19. dc6 Qc6 20. Rd6 Qa4 $17) {Ferberov – Sheremeta, URSS 1962}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. Nd2?! O-O 11. Nb3 Bg4 12. Qf1 Ba4 (13. Nc3 c6 14. h3 Bh5 15. g4 Bg6 16. dc6 Nc6 $17 {Luckis – Keres, Buenos Aires (ol) 1939}) Be7 14. h3 Bh5 15. Be3 Bd7 16. g4 Bg6 17. Nd2 Rd5 18. cb5 Nd3 20. Bb1 Qd5 (20... a6 21. ba6 Ra6 (21. b6!?) Ra6 $17 {W. Schröder – P. Lau, corr. 1998}) 21. c4 (21. Ra1 Qb5 $17 {→ Salwe – Marshall, Viena 1908}) Qe6 22. Bd4 Qe5 $17 {Keres}")
  ]

  #entry("3")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 11\"]\n\n11... e3 12. Be3 Bxe3 13. fe3 Ne4 14. Nf1 (14. O-O Bg3 15. Qd3 Bf1 16. Rf1 Qg5 17. e4 Qc1 18. b3 Re8 19. Qd2 $10 {Chernyatin – A. Machulsky, corr. 1957}) Qh4 15. g3 Qf6 16. c3 Bf5 $17 {Korchnoi – Śliwa, Bucureşti 1954}")
  ]

  #entry("4")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 15\"]\n\n15. Rg1? bc6 16. Nf3 Qf6 17. c3 Rb8 18. dc4 Rd8 19. b4 c5! $17 {Naftalin – Fridman, corr. 1971 – 12/258}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 15\"]\n\n15. O-O Bg3 16. Qf3 Bf1 17. Rf1 Qb6 18. b3 bc6 $17 {Kopylov – Kondratev, URSS 1955}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 15\"]\n\n15. Nf1 Qh4 16. g3 Qf6 $13")
  ]

  #entry("5")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 7\"]\n\n7... Bd6 8. Nc3 O-O 9. Bd7 Qd7 10. a3! (10... b6 11. d3 Rae8 12. Ne4 Be7 13. Nf6 Bf6 14. Ne4 Bd8 15. c4 f5 16. Nc3 Nb7 17. Bb5 f4 18. f3 Be7!? 19. Na7 Rf6 20. Bc6 Bc5 21. Kh1 Nd6 $13 {→ Winkelmann – M. Keller, corr. 1970}) 11. de4 Nc6 12. d3 Nd4 13. Qd1 Rac8 14. Be3 Ba3! (15. Bc1!? {Beliavsky}) 15. Bd4?! ed4 16. Ne4 Ne4 17. Nb2 Rb1 Ba3 $17 {Šulskis – Beliavsky, Koszalin 1998 – 73/368}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 10\"]\n\n10... b6 11. d3 c6 12. b4 Nb7 13. dc6 Qc6 14. Ne4 Nd7 15. Qf3 Be7 16. Nh7 Rfc8 17. Nhg5 f6 18. h3 Nf8 19. O-O $16 {Morozevich – I. Sokolov, Sarajevo 1999 – 75/301}")
  ]

  #entry("6")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 9\"]\n\n9. O-O Re8!? (9... c6 10. dc6 Nc6 11. Bc6 Bc6 12. d3 Nd5 $10 {Bohak – T. Dmitrović, corr. 1979}) (9... Bg4 10. f3 Bh5 11. Ne5 Bg6 12. Ne4 a6 13. Bd3 Re8 14. Kh1 b5 15. a3 Nb7 16. Qg3 Nh5 $10 {Short – Hector, España 2003 – 87/(305)}) 10. Ne4 c6 11. dc6 Nc6 12. Nf6 Bf6 13. Bc6 Nd4")
  ]

// =========================================================================
// Chú thích 7-37 (tiếp tục liền mạch, không ngắt trang cố định)
// =========================================================================

  #entry("7")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 10\"]\n\n10... Nd5 11. Qe5 c6 12. d3 Rfe8 13. Bd2 Ba3 (13... Nd6 14. Qd4 $17 {A. Sokolov – R. Berziņš, Basel 2000 – 77/324}) 14. Qd4 Bb2 15. Rab1 Bc3 16. Bc3 f6 17. Ne4 $16 {A. Sokolov}")
  ]

  #entry("8")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 12\"]\n\n12. Nd5?? 13. Qe4 f5 14. Qd5 Qd5 15. Nd5 Bd2 16. Nf3 Rad8 17. Bc7 Be7 18. Nb5 $16 {Grétarsson – Mich. Rygaard, Estocolmo 1992}")
  ]

  #entry("9")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 13\"]\n\n13. bc3?!")
  ]

  #entry("10")[
    Pavasovič -- Flear, Asti 1996
  ]

  #entry("11")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 8\"]\n\n8. Bf1?! h6 9. Nh3 Bc5 10. d3 Qb6 11. Ne2 Qg4 12. f3 Bh3 13. gh3 O-O-O $17 {Steinitz – Chigorin, La Habana (m/8) 1892}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 8\"]\n\n8. Na4?! h6 9. Ne2 Qe2 (10. Ne5 Qd4 $17 {E. Bialkowski – W. Weissleder, corr. 1971}) Ne1 11. Rg1 Bc5 $17 {S. Kara – Th. Karius, Deutschland 1996}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 8\"]\n\n8. Bd3? h6 (8... Bd6 9. Nc3 O-O-O O-O h6 11. Nge4 Ne4 12. Ne4 Be7 {(G. Rabovszky – M. Fazekas, Aggtelek 2000)} 13. Qh5!?) (8... Nd5 9. h4! $13 {Sh. Kagan – Y. Porat, Israel (ch) 1965}) 9. h4!? $13 {Malada – E. Bursić, Umag 2000} Be7 11. Ne4 cd5 12. Bb5 Bd7 13. Bd7 Qd7 14. Ng3 O-O-O 15. O-O f5! $17 {Rivas Pastor – Re. Alonso, Marbella 2004 – 92/(319)}")
  ]

  #entry("12")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 8\"]\n\n8... h6 9. Ne4 Nd5 10. Ne2 (10. Bb5 11. Ne3 Nd3 10. O-O Nc6 14. d3 O-O-O $17 {Van der Wiel – Spassky, Reggio Emilia 1986/87}) Be7 11. Qg3 $10 {Taruffi – Gy. Rajna, España 1974 – 19/(235)}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. Ba4!? Be7 11. d3 O-O 12. O-O f5 13. Ng3!? (13. Nec3 Be6!? 14. Re1 Bf6 15. Bb3 Bb3 16. ab3 Nb4 17. a3! {Shanava – V. Gaprindashvili, Bakú 2005 – 95/(231)}) Be6 14. Bd2 Rb8 15. Re1! Bf1 16. Na3 $16 {Van der Wiel – Van Leent, Hoogeveen 2005 – 95/(231)}")
  ]

  #entry("13")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 9\"]\n\n9... Nb7 10. d4! Bb4 11. c3 Ba5 12. Qa7! {L. Abramov – Konstantinopolsky, URSS 1949}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 9\"]\n\n9... Nc5 10. O-O-O 11. b4! Bb4 12. Nc3! {Bogoljubov – Euwe, Karlovy Vary (m/3) 1941}")
  ]

  #entry("14")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. b4!? Bb4 11. a4! $17 {A. Barthel – A. Adrian, Trier 1952}")
  ]

  #entry("15")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 11\"]\n\n11. Bg2 Bc6 13. d3 Qf5 14. Bd2 Bb7 15. Nc3 Be7 16. O-O-O $16 {A. Zaitsev – Khokhlovkin, URSS 1954}")
  ]

  #entry("16")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 12\"]\n\n12. O-O Nc6 13. c3 h6 14. Nf3! $17 {Vasilev – Morozov 1951}")
  ]

  #entry("17")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 12\"]\n\n12... Rg1 Bb7 14. Ne4 (14. Ne5 O-O 15. Ne8 Re8 16. Nc3 Rd8 17. Qg6 18. f3 $16 {T. Szabó – M. Forgács, corr. 1998}) Qh3 15. Nd2 Qh5 {(V. Shcherbakov – Neishtadt 1954)} 17. Qg5! $17 {Gligorić}")
  ]

  #entry("18")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 15\"]\n\n15. Nf3? f5 16. Ne5 Ne5 17. Qe5 Qe8 $17 {Toran – Cortlever, Beverwijk 1953}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 15\"]\n\n15. Be2 Re8 $17 {Estrin}")
    ;
    #ln("Bd2 f5 17. Nb3 $17 {Estrin}")
  ]

  #entry("19")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 9\"]\n\n9. Be2 Bd6 10. Nc3 O-O 11. d3 Rb8 12. h4 Bg4 13. Qe3 Be2 14. Qe2 c5 15. Ne4 Ne6 16. de4 f5 $17 {Bird – Schiffers, Hastings 1895}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 16\"]\n\n16. de4!? {Gligorić}")
  ]

  #entry("20")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 9\"]\n\n9... Rg4 10. Qh3 Nc3 11. Nb7 f3 12. h4 13. Bc4 Bd6 15. Qb3! $17 {Ufimtsev – Borisenko, URSS 1953}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 9\"]\n\n9... h6 10. Ne4 Ne4 (11... Nc4 Be7 12. Ng3 Re1 Be6) 12. Bd3 Bb7 14. O-O Qc5 Nf4 {Van der Wiel – Gligorić, Baden 1980 – 30/256}")
  ]

  #entry("21")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. O-O O-O 11. Bf5 h6 12. Ne4 Nd5 {(Paoli – Witkowski, Ljubljana 1955)} 13. Ng3! Nb4 14. Qd1 $16 {Gligorić}")
  ]

  #entry("22")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 10\"]\n\n10... h6 11. Ne4 Nd5 12. Ng3 g6 13. Be4 Be6 14. Nc3 Rd8 {(Estrin – Novopashin, URSS 1958)} 15. d4! ed4 16. Nd5! $17 {Estrin}")
  ]

  #entry("23")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 11\"]\n\n11. b4!? Bb2 h6 13. Ne4 Nd5 14. Qg3 Nf4 15. Re1 Kh8 $17 {Bianchi – Rubinetti, Morón 1981}")
  ]

  #entry("24")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 11\"]\n\n11... Qa6 12. d3 g6 (12... Bd6 13. b4! Bb4 14. Nh7!? {Károly Honfi – Petar Genov, Ruse 1961}) 13. Nh3 c5 14. Nc3 Nc6 15. Ne4 Nd6 17. Qe4 Bd8 18. Rh6 Re8 19. Be3 $17 {Károly Honfi – Geller, Oberhausen 1961}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 11\"]\n\n11... Bb7 12. d3 c5 13. Nh3 g6 14. Be4 $17 {Kamyshov – Sopkov, URSS 1949}")
  ]

  #entry("25")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 12\"]\n\n12... Ne4 13. Qe4 Bb7 14. d3 $17 {E. Kratz – T. Post, corr. 1991}")
  ]

  #entry("27")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 9\"]\n\n9. Bd3 O-O 10. a3 (10... Bc3 11. Ne4 Nd5 12. Qg3 g6 $17 {Van der Wiel}) Bg4 11. Qg3 Qd4 12. Nc3 Kh8 13. Ba7 Kh8 14. Ne4 Ng3 15. h3 {1/2:1/2 P. van der Houwen – L. Thorn, corr. 2002}")
  ]

  #entry("28")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 11\"]\n\n11. Qf3 O-O (11... h6 12. Ne4 Nd5 13. O-O O-O 14. d3 Bc6 15. Qe2 f5 $17 {Gligorić}) (11... Rc8 12. Ne4 Nd5 13. a3 O-O 14. Nc3 Nf4 15. Qe2 Bc6 16. g4 Be6 17. d3 Ra8 18. c3 Nc5 $17 {Gavrilov – Poletaev, corr. 1957}) 12. Nc3 Rc8 13. d3 Bc6 14. Ng4 Be6 16. de4 Bb4 $17 {Macieja – Šulskis, Tripoli (m/2) 2004}")
  ]

  #entry("29")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 12\"]\n\n12... Rc8 13. Qe2 h6 14. Nf3 e4 15. Ne5 Bf5 $17 {Sakharov – Bakhmatov, URSS 1960}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 12\"]\n\n12... Nh5 13. Nf3 Rc8 $17 {Liegl – P. Heilemann, corr. 1954}")
  ]

  #entry("30")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 15\"]\n\n15. Qb3 h6 16. Ne4 Ne4 17. Qe4 Be6 $17 {N. Kopylov – G. Borisenko, Leningrado 1954}")
  ]

  #entry("31")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 9\"]\n\n9. Bc6 Bc6 10. Qc6 Qd7 11. d3 O-O 12. Ne7 {N. Kopylov – G. Borisenko, Leningrado 1954}")
  ]

  #entry("32")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 9\"]\n\n9... g6 10. Nc3 Bg7 11. b3 h6 12. Ne4 Nd5 13. Ba3 Bb4 14. Bb4 Rb4 15. a3 Rd4 16. Qe2 $16 {Sakharov – Shianovsky, URSS 1957}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 9\"]\n\n9... h6 10. Ne4 Nd5 (10... Ne4 11. Be4 Bc6 12. Nc6 Qc6 13. Qc6 Kf8 14. b3 Bf8 15. Bd6 Qd6 16. Ba3 $16 {Károly Honfi – Georgescu, Bucureşti 1962}) 11. Rb1 g6 12. c4 Nf6 13. Bb2 Bg7 14. Qf4! (14. Ba3 Rb7 15. Qf4! Na6 16. Bc5 17. f3! $17 {M. Jasinski – R. Radecki, corr. 1996}) ef4 15. Bg7 Rd7 16. Bf6! Ke8 17. O-O Kc7 18. Re1 $16 {Van der Wiel – S. Ernst, Groningen 2004 – 92/(319)}")
  ]

  #entry("33")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 10\"]\n\n10. Nc3 O-O 11. b3 {(11. O-O = 10. O-O)} Rb4 $17 {Hilge – H. Appel, Deutschland 1997}")
  ]

  #entry("34")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 11\"]\n\n11. c3 h6 12. Ne4 $17 {P. Đurić – Brenjo, Jugoslavija 1993}")
    ;
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 11\"]\n\n11. Ne4 {(A. Levin – B. Till, EE.UU. 1999)} Nd5 $17")
  ]

  #entry("35")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b KQkq - 0 11\"]\n\n11... c5 12. Re1 $16 {Sorbe – Rainfray, Francia 1998}")
  ]

  #entry("36")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 12\"]\n\n12. Ne4 Nd5 $17 {Parkin – T. Upton, Escocia (ch) 1997}")
  ]

  #entry("37")[
    #ln("[SetUp \"1\"][FEN \"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 15\"]\n\n15. Ne2 Bd6 16. d3 e4 17. Nef4 {Sakharov – Voronov, URSS 1971}")
  ]
