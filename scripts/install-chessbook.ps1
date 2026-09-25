# Cài thư viện cờ vua chessbook làm gói Typst cục bộ @local/chessbook:0.1.0
# cho Windows, để bản desktop dùng được:
#   #import "@local/chessbook:0.1.0": *
# Chạy: powershell -ExecutionPolicy Bypass -File scripts\install-chessbook.ps1
# Chạy lại mỗi khi cập nhật chessbook\lib.
$ErrorActionPreference = "Stop"
$Version = "0.1.0"
$Src = Join-Path $PSScriptRoot "..\chessbook\lib"
$Dest = Join-Path $env:APPDATA "typst\packages\local\chessbook\$Version"
if (Test-Path $Dest) { Remove-Item -Recurse -Force $Dest }
New-Item -ItemType Directory -Force -Path $Dest | Out-Null
Copy-Item -Recurse -Force (Join-Path $Src "*") $Dest
Write-Host "Da cai @local/chessbook:$Version vao $Dest"
Write-Host "Nho cai font Roboto va Noto Serif (mien phi, Google Fonts) de ban in dung nhan dien."
