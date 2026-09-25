#!/usr/bin/env bash
# Compiles every chessbook demo/template plus a document that imports the
# library the way the web editor does (@local/chessbook), and fails on any
# error or warning. Used by CI; also handy before committing library edits.
#
#   scripts/check-chessbook.sh [--font-path DIR]
#
# Needs `typst` on PATH (or $TYPST). @preview/board-n-pieces is downloaded
# by typst on first use (network), or taken from $TYPST_PACKAGE_PATH.
set -euo pipefail
cd "$(dirname "$0")/.."
TYPST="${TYPST:-typst}"
FONT_ARGS=()
if [[ "${1:-}" == "--font-path" ]]; then FONT_ARGS=(--font-path "$2"); fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Package dir holding @local/chessbook (this checkout) and, when given,
# the preview packages from $TYPST_PACKAGE_PATH.
pkgs="$work/packages"
mkdir -p "$pkgs/local/chessbook"
ln -s "$PWD/chessbook/lib" "$pkgs/local/chessbook/0.1.0"
if [[ -n "${TYPST_PACKAGE_PATH:-}" && -d "$TYPST_PACKAGE_PATH/preview" ]]; then
  ln -s "$TYPST_PACKAGE_PATH/preview" "$pkgs/preview"
fi

# A new document as created by the web UI, exercising every inserted snippet.
mkdir -p "$work/project"
cat > "$work/project/main.typ" <<'TYP'
#import "@local/chessbook:0.1.0": *
#show: chess-book-init.with(title: "KIỂM TRA", author: "CI", paper-size: "a5")
= Chương 1
Quân #wK #bQ, ký hiệu #nag("16") #nag("13") #nag("99").
#puzzle-card("r1bqk2r/pppp1ppp/2n5/4p3/1bB1n3/2NP1N2/PPP2PPP/R1BQK2R w KQkq - 0 6",
  number: 1, title: "Đòn đánh đôi", turn: "w", difficulty: 2, hint: none, solution: "1. dxe4")
#puzzle-card("8/8/8/8/8/8/8/9 w - - 0 1", number: 2, title: "FEN sai", to-move: "b")
#opening-diagram-box("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1", title: "1.e4", turn: "b")
#column-diagram("8/8/8/4k3/8/8/8/4K3 w - - 0 1", move-num: "Sau 40.Kf2", turn: "w")
#teaching-diagram("8/8/8/4k3/8/8/8/4K3 w - - 0 1", title: "Tàn cuộc", turn: "Đen")
#eco-header(code: "C 58", name: "Phòng thủ Hai Mã", subname: "", intro-moves: "1. e4 e5")
#game-header(white: "A", white-title: "", white-elo: "", white-fed: "", black: "B", black-title: "",
  black-elo: "", black-fed: "", event: "Giải", site: "", date: "", round: "", result: "", eco: "", opening: "")
#lesson-header(lesson-num: 1, title: "ĐÒN GHIM", level: "Cấp Mã", duration: "60 phút", objective: "Hiểu đòn ghim.")
#render-puzzle-solutions()
TYP

status=0
check() {
  local root="$1" file="$2"
  local out
  if ! out="$("$TYPST" compile --root "$root" --package-path "$pkgs" "${FONT_ARGS[@]}" "$file" "$work/out.pdf" 2>&1)"; then
    echo "FAIL  $file"; echo "$out"; status=1
  elif grep -q "^warning" <<<"$out"; then
    echo "WARN  $file"; echo "$out"; status=1
  else
    echo "ok    $file"
  fi
}

for f in chessbook/demo_collection/*.typ chessbook/templates/*.typ; do
  check chessbook "$f"
done
check "$work/project" "$work/project/main.typ"
exit $status
