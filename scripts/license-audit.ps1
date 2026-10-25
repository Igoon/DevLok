param(
    [string]$OutDir = "dist\licenses"
)

$ErrorActionPreference = "Continue"
$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $root

$out = Join-Path $root $OutDir
New-Item -ItemType Directory -Force -Path $out | Out-Null

$goBin = Join-Path (go env GOPATH) "bin"
$goLicenses = Join-Path $goBin "go-licenses.exe"
if (-not (Test-Path $goLicenses)) {
    Write-Host "go-licenses belum ada, memasang..."
    go install github.com/google/go-licenses@latest
    if ($LASTEXITCODE -ne 0) { throw "go install go-licenses gagal" }
}

$goLog = Join-Path $out "go-licenses.log"
Write-Host "Audit lisensi Go -> $OutDir\go-licenses.csv"
& $goLicenses csv ./... --ignore devlok 2> $goLog | Set-Content -LiteralPath (Join-Path $out "go-licenses.csv") -Encoding utf8
if ($LASTEXITCODE -ne 0) {
    Get-Content -LiteralPath $goLog -ErrorAction SilentlyContinue | Write-Error
    throw "go-licenses gagal (kode $LASTEXITCODE)"
}

$npmLog = Join-Path $out "npm-licenses.log"
Write-Host "Audit lisensi frontend -> $OutDir\npm-licenses.txt"
Push-Location (Join-Path $root "desktop\frontend")
try {
    npx --yes license-checker --summary 2> $npmLog | Set-Content -LiteralPath (Join-Path $out "npm-licenses.txt") -Encoding utf8
    if ($LASTEXITCODE -ne 0) {
        Get-Content -LiteralPath $npmLog -ErrorAction SilentlyContinue | Write-Error
        throw "license-checker gagal (kode $LASTEXITCODE)"
    }
} finally {
    Pop-Location
}

$risky = 'GPL|AGPL|LGPL|SSPL|Commons Clause'
$hits = @()
Get-Content -LiteralPath (Join-Path $out "go-licenses.csv") | ForEach-Object {
    $parts = $_ -split ','
    if ($parts.Count -ge 2 -and $parts[1] -match $risky) {
        $hits += "Go: $($parts[0]) = $($parts[1])"
    }
}
Get-Content -LiteralPath (Join-Path $out "npm-licenses.txt") | ForEach-Object {
    if ($_ -match $risky) { $hits += "npm: $_" }
}

if ($hits.Count -gt 0) {
    Write-Warning "Lisensi berisiko ditemukan:"
    $hits | ForEach-Object { Write-Warning $_ }
    exit 2
}

Write-Host "Selesai. Tidak ada lisensi berisiko terdeteksi."
