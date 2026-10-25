# Stack dan Toolchain

## 1. Keputusan inti

| Area | Pilihan | Versi target | Alasan |
|---|---|---|---|
| Bahasa core | Go | 1.24+ (target 1.25) | Cross-compile, tanpa CGO, stdlib kuat untuk proses/jaringan/crypto |
| GUI | Wails v2 | 2.10+ | WebView2 di Windows, WKWebView di macOS, satu frontend |
| Frontend | Svelte 5 + TypeScript + Vite | terbaru stabil | Ringan, tanpa virtual DOM, bundle kecil |
| State format | JSON | - | Sederhana, mudah diaudit, tanpa dependency |
| Manifest | YAML | - | Ramah manusia, dukungan `gopkg.in/yaml.v3` |
| CLI | `spf13/cobra` | 1.x | Subcommand, flag, completion |
| Windows API | `golang.org/x/sys/windows` | terbaru | Job Object, DPAPI, port table |
| Test | `testing` + `httptest` | stdlib | Minim dependency |
| Installer | Inno Setup | 6.4+ | Per-user install, portable, mature |
| CI | GitHub Actions | - | Matrix Windows + macOS |

## 2. Versi runtime yang dikelola

| Runtime | Versi v0.1.0 | Sumber |
|---|---|---|
| Nginx | 1.30.x | nginx.org Windows zip |
| PHP | 8.1, 8.2, 8.3, 8.4 (default), 8.5 | windows.php.net NTS x64 |
| MySQL | 8.4 LTS, 9.4.0 | cdn.mysql.com zip |
| Composer | 2.x | getcomposer.org phar |
| Node.js | 22 LTS | nodejs.org Windows zip |
| Adminer | 6.1.2 | adminer.org single file |
| phpMyAdmin | 5.2.3 | all-languages zip, domain `phpmyadmin.test` |
| Mailpit | 1.2x (v0.1.1) | github.com/axllent/mailpit |

Catatan teknis:

- PHP Windows: gunakan build **NTS** dan `php-cgi.exe` untuk FastCGI. `php-fpm` tidak tersedia di Windows.
- MySQL Windows: butuh Visual C++ Redistributable 2015-2022 x64.
- Nginx Windows: beberapa direktif Linux tidak berlaku; template dibatasi pada fitur yang didukung.
- Node.js: dideteksi per project, tidak mengubah PATH global.

## 3. Tool pengembangan

| Tool | Versi | Instalasi |
|---|---|---|
| Go | 1.24+ | `winget install GoLang.Go` |
| Node.js | 22 LTS | `winget install OpenJS.NodeJS.LTS` |
| npm | 10+ | ikut Node.js |
| Git | 2.4x+ | `winget install Git.Git` |
| PowerShell | 7.4+ | `winget install Microsoft.PowerShell` |
| Wails CLI | 2.10+ | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| Inno Setup | 6.4+ | `winget install JRSoftware.InnoSetup` |
| WebView2 Runtime | terbaru | `winget install Microsoft.EdgeWebView2Runtime` |
| VC++ Redist x64 | 2015-2022 | `winget install Microsoft.VCRedist.2015+.x64` |

Opsional:

- `golangci-lint` untuk lint.
- `goreleaser` untuk rilis (dievaluasi di M7).
- `gh` untuk otomasi GitHub.

## 4. Kenapa bukan alternatif lain

| Alternatif | Alasan ditolak |
|---|---|
| Electron | Binary ~150 MB, memori tinggi, proses management kurang natural untuk service lokal |
| Tauri | Porting mudah dan kecil, tetapi butuh Rust toolchain; tim belum memakai Rust |
| .NET WinUI/WPF | Windows-only, bertentangan dengan target macOS |
| Fyne | GUI non-web, cross-compile butuh CGO, tampilan kurang fleksibel |
| Python + Qt | Packaging berat, distribusi runtime rumit |
| Docker-based | Berat, butuh virtualisasi, bertentangan dengan target ringan dan offline |

Wails dipilih karena kode Go dan aset web yang sama dipakai di kedua OS, dengan ukuran binary jauh lebih kecil dari Electron.

## 5. Dependency Go yang direncanakan

| Modul | Fungsi |
|---|---|
| `github.com/spf13/cobra` | CLI |
| `gopkg.in/yaml.v3` | Manifest YAML |
| `golang.org/x/sys` | Windows API |
| `github.com/wailsapp/wails/v2` | GUI |
| `github.com/google/uuid` | ID sesi API (opsional) |

Prinsip: dependency minimal, tidak memakai ORM, tidak memakai framework web untuk API lokal.

## 6. Struktur frontend

```
desktop/
|-- app.go              binding Wails
|-- main.go
|-- wails.json
`-- frontend/
    |-- index.html
    |-- src/
    |   |-- App.svelte
    |   |-- lib/api.ts
    |   |-- lib/stores/
    |   `-- components/
    |-- package.json
    `-- vite.config.ts
```

Frontend tidak memanggil OS secara langsung. Semua aksi lewat binding Go atau HTTP API lokal, sehingga UI yang sama bisa dibuka di browser.

## 7. Konvensi kode

- Go: `gofmt`, `go vet`, nama paket pendek, error wrapping dengan `%w`.
- Tidak ada `panic` untuk error yang bisa dipulihkan.
- Tidak ada akses `runtime.GOOS` di luar `internal/platform`.
- Tidak ada path absolut Windows di core.
- Semua eksekusi proses memakai `exec.CommandContext` dengan timeout.
- Frontend: TypeScript strict, tanpa `any` implisit.
- Pesan error berbahasa Indonesia untuk UI, kode error berbahasa Inggris.

## 8. Build

```powershell
# CLI + tray
wails build -platform windows/amd64

# Helper privileged
go build -o build/bin/dl-helper.exe ./cmd/dl-helper

# Installer
iscc installer\devlok.iss
```

Target CI:

```yaml
matrix:
  os: [windows-latest, macos-latest]
```

Build macOS aktif sejak v0.2.0; CI mengompilasi core dan menjalankan test untuk kedua OS.
