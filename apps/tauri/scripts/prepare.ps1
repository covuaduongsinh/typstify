# Chuan bi sidecar va tai nguyen cho ban desktop Tauri (Windows x64).
#   powershell -ExecutionPolicy Bypass -File apps\tauri\scripts\prepare.ps1 [-SkipWeb]
# Ket qua:
#   src-tauri\binaries\{typstify-server,typst,tinymist}-x86_64-pc-windows-msvc.exe
#   src-tauri\resources\web\       (web\dist)
#   src-tauri\resources\packages\  (local\chessbook, preview\board-n-pieces)
# Phien ban typst/tinymist/board-n-pieces: giu dong bo voi scripts\package-desktop.ps1 va Dockerfile.
param(
    [switch]$SkipWeb
)
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$TypstVersion = "v0.15.1"
$TypstSha256 = "19ce3551153c2fe7ee9fa2f95208310c8f4d3209fedb699e0333faf8913f6736"
$TinymistVersion = "v0.15.8"
$TinymistSha256 = "b36c3e2926b2fe870e377b27aeebcb6265dd6a19fdca148a9a3ac4fdf57ee8a5"
$BoardVersion = "0.9.0"
$BoardSha256 = "7b8d4cca2ea026d882e865f6b0307a4f95d1c065a5965e7088846f9820013f36"
$ChessbookVersion = "0.1.0"
$Triple = "x86_64-pc-windows-msvc"

$Repo = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
$Tauri = Join-Path $Repo "apps\tauri\src-tauri"
$Bin = Join-Path $Tauri "binaries"
$Res = Join-Path $Tauri "resources"
$Cache = Join-Path $Repo "dist\cache"
New-Item -ItemType Directory -Force -Path $Bin, $Res, $Cache | Out-Null

function Get-Verified([string]$Url, [string]$Name, [string]$Sha256) {
    $path = Join-Path $Cache $Name
    if (-not (Test-Path $path)) {
        Write-Host "Tai $Url"
        Invoke-WebRequest -Uri $Url -OutFile $path -UseBasicParsing
    }
    # Moi tep tai ve deu phai co SHA-256 (ke ca ban cache cu).
    if (-not $Sha256) { throw "Thieu SHA-256 cho $Name" }
    $got = (Get-FileHash -Algorithm SHA256 $path).Hash.ToLower()
    if ($got -ne $Sha256) {
        Remove-Item $path -Force
        throw "SHA-256 khong khop cho $Name (nhan $got, can $Sha256)"
    }
    return $path
}

function Expand-Exe([string]$Zip, [string]$Exe, [string]$Target) {
    $tmp = Join-Path $Cache ([IO.Path]::GetFileNameWithoutExtension($Zip) + "-x")
    if (Test-Path $tmp) { Remove-Item -Recurse -Force $tmp }
    Expand-Archive $Zip $tmp
    Copy-Item (Get-ChildItem $tmp -Recurse -Filter $Exe | Select-Object -First 1).FullName $Target -Force
}

# 1. typstify-server (khong can CGO)
Write-Host "Build typstify-server"
Push-Location $Repo
try {
    $env:CGO_ENABLED = "0"
    go build -trimpath -o (Join-Path $Bin "typstify-server-$Triple.exe") ./cmd/typstify-server
    if ($LASTEXITCODE -ne 0) { throw "go build that bai" }
} finally { Pop-Location }

# 2. typst + tinymist
$z = Get-Verified "https://github.com/typst/typst/releases/download/$TypstVersion/typst-x86_64-pc-windows-msvc.zip" "typst-$TypstVersion.zip" $TypstSha256
Expand-Exe $z "typst.exe" (Join-Path $Bin "typst-$Triple.exe")
$z = Get-Verified "https://github.com/Myriad-Dreamin/tinymist/releases/download/$TinymistVersion/tinymist-x86_64-pc-windows-msvc.zip" "tinymist-$TinymistVersion.zip" $TinymistSha256
Expand-Exe $z "tinymist.exe" (Join-Path $Bin "tinymist-$Triple.exe")

# 3. Giao dien web
$web = Join-Path $Repo "web"
if (-not $SkipWeb) {
    Push-Location $web
    try {
        npm ci
        if ($LASTEXITCODE -ne 0) { throw "npm ci that bai" }
        npm run build
        if ($LASTEXITCODE -ne 0) { throw "build web that bai" }
    } finally { Pop-Location }
}
$webOut = Join-Path $Res "web"
if (Test-Path $webOut) { Remove-Item -Recurse -Force $webOut }
Copy-Item -Recurse (Join-Path $web "dist") $webOut

# 4. Goi Typst chay offline: @local/chessbook va phu thuoc @preview/board-n-pieces
$pk = Join-Path $Res "packages"
if (Test-Path $pk) { Remove-Item -Recurse -Force $pk }
$cb = Join-Path $pk "local\chessbook\$ChessbookVersion"
New-Item -ItemType Directory -Force -Path $cb | Out-Null
Copy-Item -Recurse (Join-Path $Repo "chessbook\lib\*") $cb
$tgz = Get-Verified "https://packages.typst.org/preview/board-n-pieces-$BoardVersion.tar.gz" "board-n-pieces-$BoardVersion.tar.gz" $BoardSha256
$bp = Join-Path $pk "preview\board-n-pieces\$BoardVersion"
New-Item -ItemType Directory -Force -Path $bp | Out-Null
tar -xzf $tgz -C $bp
if (-not (Test-Path (Join-Path $bp "typst.toml"))) { throw "board-n-pieces giai nen sai" }

Write-Host "Xong: $Bin va $Res"
