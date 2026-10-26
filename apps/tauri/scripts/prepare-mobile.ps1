# Chuan bi giao dien dong goi cho ban mobile (Android).
#   powershell -ExecutionPolicy Bypass -File apps\tauri\scripts\prepare-mobile.ps1 [-SkipWeb]
# Ket qua: src-tauri\resources\mobile-dist\ = web\dist + mobile\app-setup.html
# (frontendDist cua tauri.android.conf.json).
param(
    [switch]$SkipWeb
)
$ErrorActionPreference = "Stop"

$Repo = Resolve-Path (Join-Path $PSScriptRoot "..\..\..")
$App = Join-Path $Repo "apps\tauri"
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

$out = Join-Path $App "src-tauri\resources\mobile-dist"
if (Test-Path $out) { Remove-Item -Recurse -Force $out }
New-Item -ItemType Directory -Force -Path (Split-Path $out) | Out-Null
Copy-Item -Recurse (Join-Path $web "dist") $out
Copy-Item (Join-Path $App "mobile\app-setup.html") $out
Write-Host "Xong: $out"
