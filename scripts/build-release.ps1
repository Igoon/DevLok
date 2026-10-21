param(
    [switch]$SkipBuild,
    [switch]$Installer
)

$ErrorActionPreference = "Stop"
$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$go = "C:\Program Files\Go\bin"
if (Test-Path $go) { $env:Path = "$go;$env:Path" }

if (-not $SkipBuild) {
    Write-Host "Build CLI..."
    & go build -trimpath -o (Join-Path $root "build\bin\dl.exe") ./cmd/dl
    if ($LASTEXITCODE -ne 0) { throw "go build dl gagal" }
    & go build -trimpath -o (Join-Path $root "build\bin\dl-helper.exe") ./cmd/dl-helper
    if ($LASTEXITCODE -ne 0) { throw "go build helper gagal" }

    New-Item -ItemType Directory -Path (Join-Path $root "desktop\build\windows") -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $root "desktop\assets\app.ico") -Destination (Join-Path $root "desktop\build\windows\icon.ico") -Force
    Write-Host "Build GUI..."
    $wails = Join-Path $env:USERPROFILE "go\bin\wails.exe"
    if (-not (Test-Path $wails)) { $wails = "wails" }
    Push-Location (Join-Path $root "desktop")
    try {
        & $wails build -clean
        if ($LASTEXITCODE -ne 0) { throw "wails build gagal" }
    } finally {
        Pop-Location
    }
    Copy-Item -LiteralPath (Join-Path $root "build\bin\dl-helper.exe") -Destination (Join-Path $root "desktop\build\bin\dl-helper.exe") -Force
}

$dl = Join-Path $root "build\bin\dl.exe"
$version = ((& $dl version --json | ConvertFrom-Json).data.version)
$numericVersion = ($version -split "-")[0]

$distRoot = Join-Path $root "dist"
$pkg = Join-Path (Join-Path $distRoot "DevLok") $version
if (Test-Path $pkg) { Remove-Item -LiteralPath $pkg -Recurse -Force }
New-Item -ItemType Directory -Path $pkg -Force | Out-Null

Copy-Item -LiteralPath (Join-Path $root "build\bin\dl.exe") -Destination $pkg
Copy-Item -LiteralPath (Join-Path $root "build\bin\dl-helper.exe") -Destination $pkg
Copy-Item -LiteralPath (Join-Path $root "desktop\build\bin\devlok-gui.exe") -Destination $pkg
Copy-Item -LiteralPath (Join-Path $root "README.md") -Destination $pkg
New-Item -ItemType Directory -Path (Join-Path $pkg "docs") -Force | Out-Null
Copy-Item -LiteralPath (Join-Path $root "docs\APLIKASI.md") -Destination (Join-Path $pkg "docs")
Copy-Item -LiteralPath (Join-Path $root "docs\UPDATES.md") -Destination (Join-Path $pkg "docs")
Set-Content -LiteralPath (Join-Path $pkg "VERSION") -Value "$version`n" -NoNewline

$zip = Join-Path $distRoot ("DevLok-" + $version + "-win64.zip")
if (Test-Path $zip) { Remove-Item -LiteralPath $zip -Force }
Compress-Archive -Path (Join-Path $pkg "*") -DestinationPath $zip -CompressionLevel Optimal
$hash = (Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
[IO.File]::WriteAllText($zip + ".sha256", "$hash  $([IO.Path]::GetFileName($zip))`n", (New-Object Text.UTF8Encoding($false)))

Write-Host "Portable: $zip"
Write-Host "SHA256:   $hash"

if ($Installer) {
    $iscc = "C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
    if (-not (Test-Path $iscc)) { $iscc = "C:\Program Files\Inno Setup 6\ISCC.exe" }
    if (-not (Test-Path $iscc)) { throw "ISCC.exe tidak ditemukan. Instal Inno Setup 6." }
    & $iscc "/DAppVersion=$version" "/DNumericVersion=$numericVersion" "/DDistDir=$pkg" (Join-Path $root "installer\DevLok.iss")
    if ($LASTEXITCODE -ne 0) { throw "Inno Setup gagal" }
    $setupPath = Join-Path $distRoot ("DevLok-Setup-" + $version + ".exe")
    $installerHash = (Get-FileHash -LiteralPath $setupPath -Algorithm SHA256).Hash.ToLowerInvariant()
    [IO.File]::WriteAllText($setupPath + ".sha256", "$installerHash  $([IO.Path]::GetFileName($setupPath))`n", (New-Object Text.UTF8Encoding($false)))
    Write-Host "Installer: $setupPath"
    Write-Host "SHA256:    $installerHash"
}
