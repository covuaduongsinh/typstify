#!/usr/bin/env sh
# Cài thư viện cờ vua chessbook làm gói Typst cục bộ @local/chessbook:0.1.0
# cho máy cá nhân (Linux/macOS), để bản desktop dùng được:
#   #import "@local/chessbook:0.1.0": *
# Chạy lại mỗi khi cập nhật chessbook/lib.
set -eu
VERSION=0.1.0
SRC="$(cd "$(dirname "$0")/../chessbook/lib" && pwd)"
case "$(uname -s)" in
  Darwin) DATA="$HOME/Library/Application Support" ;;
  *) DATA="${XDG_DATA_HOME:-$HOME/.local/share}" ;;
esac
DEST="$DATA/typst/packages/local/chessbook/$VERSION"
rm -rf "$DEST"
mkdir -p "$DEST"
cp -R "$SRC"/. "$DEST"/
echo "Đã cài @local/chessbook:$VERSION vào $DEST"
echo "Nhớ cài font Roboto và Noto Serif (miễn phí, Google Fonts) để bản in đúng nhận diện."
