# Dong goi Typstify desktop (Windows x64) thanh bo cai dat.
#   powershell -ExecutionPolicy Bypass -File scripts\package-desktop.ps1 [-Version 0.1.0] [-SkipBuild]
# Ket qua trong dist\: Typstify-Setup-<ver>.exe (neu co Inno Setup) va Typstify-portable-<ver>.zip.
# Nang phien ban typst/tinymist: sua hang o duoi, dong thoi sua Dockerfile.
param(
    [string]$Version = "",
    [switch]$SkipBuild
)
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$TypstVersion = "v0.15.1"
$TypstSha256 = "19ce3551153c2fe7ee9fa2f95208310c8f4d3209fedb699e0333faf8913f6736"
$TinymistVersion = "v0.15.8"
$TinymistSha256 = "b36c3e2926b2fe870e377b27aeebcb6265dd6a19fdca148a9a3ac4fdf57ee8a5"
$BoardVersion = "0.9.0"
$ChessbookVersion = "0.1.0"

$Root = Resolve-Path (Join-Path $PSScriptRoot "..")
$Dist = Join-Path $Root "dist"
$Cache = Join-Path $Dist "cache"
$Stage = Join-Path $Dist "stage"
Set-Location $Root

if (-not $Version) {
    $Version = (git describe --tags --always 2>$null)
    if (-not $Version) { $Version = "0.0.0" }
}
$Version = $Version.TrimStart("v")
# Inno Setup chi nhan so phien ban dang so; giu ban day du cho ten file.
$FileVer = $Version

function Get-Verified([string]$Url, [string]$Name, [string]$Sha256) {
    $path = Join-Path $Cache $Name
    if (-not (Test-Path $path)) {
        Write-Host "Tai $Url"
        Invoke-WebRequest -Uri $Url -OutFile $path -UseBasicParsing
    }
    if ($Sha256) {
        $got = (Get-FileHash $path -Algorithm SHA256).Hash.ToLower()
        if ($got -ne $Sha256) {
            Remove-Item $path -Force
            throw "SHA-256 khong khop cho $Name (nhan $got, can $Sha256)"
        }
    }
    return $path
}

New-Item -ItemType Directory -Force -Path $Cache | Out-Null
if (Test-Path $Stage) { Remove-Item -Recurse -Force $Stage }
New-Item -ItemType Directory -Force -Path (Join-Path $Stage "bin") | Out-Null

# 1. Build app
$exe = Join-Path $Stage "Typstify.exe"
if ($SkipBuild -and (Test-Path (Join-Path $Dist "Typstify.exe"))) {
    Copy-Item (Join-Path $Dist "Typstify.exe") $exe
} else {
    $gogio = (Get-Command gogio -ErrorAction SilentlyContinue).Source
    if (-not $gogio) { $gogio = Join-Path (go env GOPATH) "bin\gogio.exe" }
    if (-not (Test-Path $gogio)) {
        Write-Host "Cai gogio..."
        go install gioui.org/cmd/gogio@latest
        $gogio = Join-Path (go env GOPATH) "bin\gogio.exe"
    }
    if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) { throw "Thieu gcc (CGO)." }
    $env:CGO_ENABLED = "1"
    $goVer = (go env GOVERSION)
    $now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $ld = "-X looz.ws/typstify/version.BinVersion=$Version -X looz.ws/typstify/version.BuildTime=$now -X looz.ws/typstify/version.BuildGoVersion=$goVer"
    Write-Host "Build Typstify.exe ($Version)"
    & $gogio -target windows -arch amd64 -icon version/appicon.png -ldflags $ld -o $exe .
    $gogioExit = $LASTEXITCODE
    # gogio de lai file .syso (icon/manifest) o goc repo; go build thuong se nhat no nham.
    Get-ChildItem $Root -Filter "*_windows_*.syso" -File | Remove-Item -Force
    if ($gogioExit -ne 0 -or -not (Test-Path $exe)) { throw "gogio build that bai" }
    Copy-Item $exe (Join-Path $Dist "Typstify.exe") -Force
}

# 2. typst + tinymist
$typstZip = Get-Verified "https://github.com/typst/typst/releases/download/$TypstVersion/typst-x86_64-pc-windows-msvc.zip" "typst-$TypstVersion.zip" $TypstSha256
$tmp = Join-Path $Cache "typst-x"; if (Test-Path $tmp) { Remove-Item -Recurse -Force $tmp }
Expand-Archive $typstZip $tmp
Copy-Item (Get-ChildItem $tmp -Recurse -Filter typst.exe | Select-Object -First 1).FullName (Join-Path $Stage "bin\typst.exe")

$tmZip = Get-Verified "https://github.com/Myriad-Dreamin/tinymist/releases/download/$TinymistVersion/tinymist-x86_64-pc-windows-msvc.zip" "tinymist-$TinymistVersion.zip" $TinymistSha256
$tmp = Join-Path $Cache "tinymist-x"; if (Test-Path $tmp) { Remove-Item -Recurse -Force $tmp }
Expand-Archive $tmZip $tmp
Copy-Item (Get-ChildItem $tmp -Recurse -Filter tinymist.exe | Select-Object -First 1).FullName (Join-Path $Stage "bin\tinymist.exe")

# 3. chessbook + board-n-pieces (goi Typst chay offline)
Copy-Item -Recurse (Join-Path $Root "chessbook\lib") (Join-Path $Stage "chessbook-lib")
$boardTgz = Get-Verified "https://packages.typst.org/preview/board-n-pieces-$BoardVersion.tar.gz" "board-n-pieces-$BoardVersion.tar.gz" ""
$boardDir = Join-Path $Stage "board-n-pieces"
New-Item -ItemType Directory -Force -Path $boardDir | Out-Null
tar -xzf $boardTgz -C $boardDir
if (-not (Test-Path (Join-Path $boardDir "typst.toml"))) { throw "board-n-pieces giai nen sai" }

# 4. Script cai cho ban portable (cung logic voi installer)
@"
# Cai Typstify portable: copy vao %LOCALAPPDATA%\Programs\Typstify va cai goi Typst.
`$ErrorActionPreference = "Stop"
`$src = `$PSScriptRoot
`$app = Join-Path `$env:LOCALAPPDATA "Programs\Typstify"
`$pk = Join-Path `$env:APPDATA "typst\packages"
New-Item -ItemType Directory -Force -Path `$app | Out-Null
Copy-Item -Recurse -Force (Join-Path `$src "Typstify.exe") `$app
Copy-Item -Recurse -Force (Join-Path `$src "bin") `$app
`$cb = Join-Path `$pk "local\chessbook\$ChessbookVersion"
`$bp = Join-Path `$pk "preview\board-n-pieces\$BoardVersion"
foreach (`$d in @(`$cb, `$bp)) { if (Test-Path `$d) { Remove-Item -Recurse -Force `$d }; New-Item -ItemType Directory -Force -Path `$d | Out-Null }
Copy-Item -Recurse -Force (Join-Path `$src "chessbook-lib\*") `$cb
Copy-Item -Recurse -Force (Join-Path `$src "board-n-pieces\*") `$bp
`$lnk = Join-Path `$env:APPDATA "Microsoft\Windows\Start Menu\Programs\Typstify.lnk"
`$s = (New-Object -ComObject WScript.Shell).CreateShortcut(`$lnk)
`$s.TargetPath = Join-Path `$app "Typstify.exe"; `$s.WorkingDirectory = `$app; `$s.Save()
Write-Host "Da cai Typstify vao `$app"
"@ | Set-Content -Encoding UTF8 (Join-Path $Stage "install.ps1")

# 5. Dong goi
$zip = Join-Path $Dist "Typstify-portable-$FileVer.zip"
if (Test-Path $zip) { Remove-Item $zip -Force }
Compress-Archive -Path (Join-Path $Stage "*") -DestinationPath $zip
Write-Host "Portable: $zip"

$iscc = @(
    (Get-Command ISCC.exe -ErrorAction SilentlyContinue).Source,
    "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
    "C:\Program Files\Inno Setup 6\ISCC.exe",
    (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6\ISCC.exe")
) | Where-Object { $_ -and (Test-Path $_) } | Select-Object -First 1
if ($iscc) {
    # Inno chi nhan x.y.z.w dang so cho VersionInfo; dung ban rut gon cho AppVersion.
    & $iscc "/DAppVersion=$Version" "/DStageDir=$Stage" "/DOutDir=$Dist" "/DChessbookVersion=$ChessbookVersion" "/DBoardVersion=$BoardVersion" (Join-Path $PSScriptRoot "typstify.iss")
    if ($LASTEXITCODE -ne 0) { throw "ISCC that bai" }
    Write-Host "Installer: $(Join-Path $Dist "Typstify-Setup-$Version.exe")"
} else {
    Write-Warning "Khong tim thay Inno Setup (ISCC.exe); chi tao ban portable. Cai: winget install JRSoftware.InnoSetup"
}
Write-Warning "Ban dong goi chua ky so; Windows SmartScreen co the canh bao. Font Roboto/Noto can cai rieng neu chua co."
