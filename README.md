# DevLok

## Download

- [GitHub Releases](https://github.com/Igoon/DevLok/releases)
- Portable ZIP dan installer Windows tersedia di halaman rilis.
- Installer belum ditandatangani Authenticode.

Local development environment untuk Windows: Nginx + PHP + MySQL, dengan GUI dan CLI. Dibuat sebagai alternatif Laragon yang reproducible, aman secara default, dan siap diport ke macOS.

- **Versi awal**: `0.1.1` (Windows 10 22H2+ / Windows 11, x64)
- **Status**: `0.1.1` stable untuk Windows. macOS memakai Homebrew fallback untuk Nginx/PHP; MySQL eksternal; packaging unsigned sudah tersedia sebagai artifact.
- **Stack**: Go + Wails v2 + Svelte + TypeScript
- **Binary**: `dl.exe` (CLI), `devlok-gui.exe` (GUI Wails), `dl-helper.exe` (hosts privileged)
- **Manifest project**: `.devlok.yaml` + `.devlok.lock`

Sudah tersedia: manifest loader/validator, logger JSONL, platform Windows, `doctor`, `init`, `version`, runtime manager (SHA256 + ekstraksi aman), deteksi port dan owner, supervisor proses (Job Object, health check, restart, rekonsiliasi), generator config Nginx, hosts terkelola, CA lokal + TLS per domain, stack `up/down/status`, MySQL per project (password DPAPI, database/user acak), backup/restore/seed, lockfile dan verifikasi checksum, export environment, import Laragon, `run/shell` dengan env project, GUI Wails, phpMyAdmin, editor teks, single-instance lock, dan CI Windows/macOS.

Belum tersedia: migrasi schema manifest, forwarding perintah lewat named pipe, browser fallback, dan dukungan macOS.

## Masalah yang diselesaikan

Laragon cepat dan praktis, tetapi memiliki keterbatasan:

1. Windows-only.
2. Runtime global, tidak bisa pin versi per project.
3. Penambahan versi runtime manual, tanpa checksum dan update.
4. TLS lokal lemah dan trust certificate tidak dikelola otomatis.
5. Tidak ada supervision proses, health check, atau pemulihan setelah crash.
6. Port bentrok baru terdeteksi dari log.
7. Log tersebar di banyak folder.
8. PATH global diubah sehingga bentrok dengan tool sistem.
9. Konfigurasi situs menumpuk tanpa lifecycle.
10. Environment tidak reproducible dan tidak bisa dibagikan ke tim.

Sebagian besar kekurangan Laragon sudah ditangani. Audit status lengkap ada di [docs/LARAGON-GAPS.md](docs/LARAGON-GAPS.md). Tiga pembeda utama:

- **Runtime per project** dan lockfile.
- **Supervisi proses** yang benar.
- **Default aman**: loopback-only, password acak, TLS 1.2/1.3.

## Scope v0.1.0

Runtime dan tools yang dikelola:

| Komponen | Versi awal | Catatan |
|---|---|---|
| Nginx | 1.30.x | Web server default |
| PHP | 8.1 - 8.5 | Default 8.4, banyak versi berdampingan |
| MySQL | 8.4 LTS, 9.4 | Instance per versi; pilih di GUI/CLI |
| Composer | 2.x | Dikelola sebagai tool |
| Node.js | 22 LTS | Opsional, per project |
| Adminer | 6.1.2 | GUI database |
| phpMyAdmin | 5.2.3 | GUI database, domain `phpmyadmin.test` |
| Notepad++ | 8.9.8.1 | Editor portable, dipakai `dl edit` |
| Git | sistem | Dideteksi, tidak dibundel |
| Mailpit | 1.2x | Opsional, target v0.1.1 |

Bukan bagian dari v0.1.0: macOS, Linux, Docker, Apache, Redis, PostgreSQL, deploy remote, Windows Service.

## Prasyarat build

Windows 10 22H2+ atau Windows 11 x64, PowerShell 7, Git.

```powershell
winget install GoLang.Go
winget install OpenJS.NodeJS.LTS
winget install Git.Git
winget install Microsoft.PowerShell
winget install JRSoftware.InnoSetup
winget install Microsoft.EdgeWebView2Runtime
winget install Microsoft.VCRedist.2015+.x64

go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Verifikasi:

```powershell
go version
node --version
wails version
```

## Prasyarat runtime (end user)

Installer DevLok menangani:

- Deteksi WebView2 Runtime dan tautan unduh resmi jika belum ada.
- Visual C++ Redistributable 2015-2022 x64.
- Runtime Nginx/PHP/MySQL diunduh saat pertama kali dipakai, bukan dibundel.

## Konfigurasi domain, port, dan komponen

Semua diatur di `.devlok.yaml`:

```yaml
name: siso
web:
  domain: app.local        # opsional; default <name>.<tld>
  tld: test
  aliases:
    - www.app.local
  php: "8.4"

services:
  mysql: "8.4"             # katalog: 8.4.10 dan 9.4.0

ports:
  http: 8081               # 0 / kosong = otomatis
  https: 8444
  mysql: 3307
  php: 19000               # port FastCGI internal

components:
  mysql: true
  phpmyadmin: false        # matikan phpMyAdmin
```

- Port eksplisit yang dipakai proses lain menghasilkan error, bukan fallback diam-diam.
- `components.mysql: false` membuat `dl up` tidak menjalankan MySQL/phpMyAdmin; `dl db` tetap bisa dipakai eksplisit.
- `web.domain` mengatur nama virtual host sepenuhnya, termasuk domain multi-label.
- Versi MySQL dipilih lewat `services.mysql` atau dropdown di area service GUI; katalog saat ini 8.4.10 dan 9.4.0.
- GUI: toggle Database/phpMyAdmin dan pilihan versi MySQL langsung di area service; perubahan otomatis restart stack.
- Tab **Konfigurasi**: atur domain, alias, document root, nama database, HTTPS, port HTTP/HTTPS/MySQL/PHP, versi Nginx/PHP/MySQL/Node, dan komponen. Port dicek live (free/in use). Simpan akan restart stack bila aktif.
- Tab **Update**: cek update aplikasi/runtime, unduh installer, apply, dan rollback.
- Tab **Import**: scan Laragon, pilih project, tulis manifest, dan daftarkan project ke GUI tanpa memindahkan folder.
- `dl site create --no-db` membuat site tanpa database.
- `dl edit` memakai Notepad++ bila terpasang (`dl runtime add notepadpp`), lalu VS Code, lalu Notepad.

## Build dan jalankan

```powershell
go build -trimpath -o build\bin\dl.exe ./cmd/dl
go build -trimpath -o build\bin\dl-helper.exe ./cmd/dl-helper
go test ./...
.\build\bin\dl.exe doctor
.\build\bin\dl.exe version

# GUI
cd desktop
wails build -clean
copy ..\build\bin\dl-helper.exe build\bin\dl-helper.exe
cd ..
.\build\bin\dl.exe ui

# Portable ZIP / installer
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-release.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-release.ps1 -Installer
```

Binary GUI dihasilkan di `desktop\build\bin\devlok-gui.exe`. `dl ui` mencarinya otomatis.

## Catatan menjalankan

- Jalankan DevLok sebagai user biasa, jangan sebagai Administrator.
- Service latar (nginx, php-cgi, mysqld) dijalankan tanpa jendela konsol.
- Menutup jendela GUI menyembunyikannya ke tray. Menu tray: status site, Show, Open site, Sites, Start All/Stop All, Restart Stack, Nginx/PHP/MySQL/Node.js (pilih versi), serta Tools (folder project, terminal, editor, copy URL, folder config/log, About). MySQL: phpMyAdmin, Start/Stop MySQL, create database, root password, backup, my.ini, log, folder data/backup. PHP: preset Quick settings (memory/upload/timeout), Extensions, php.ini, folder ext, php_errors.log. Nginx: site config custom dan log.
- Konfigurasi hasil generate tidak diedit manual. Override permanen ada di `%USERPROFILE%\.devlok\config\php\<versi>.ini`, `config\mysql\<seri>.ini`, dan `config\nginx\<site>.conf`; tray membuka file override ini.
- PHP override ditambahkan ke akhir php.ini; MySQL memakai `!include`; Nginx memakai `include` di dalam blok server. Konfigurasi MySQL dan Nginx divalidasi sebelum service dijalankan.
- Icon tray hijau bila nginx, PHP, dan MySQL (jika aktif) semuanya running; abu-abu bila berhenti/parsial.
- Versi default/terpilih diberi tanda cek di submenu tray.
- Pilih versi service dari tray: jika stack berhenti, versi dipakai saat start berikutnya; jika berjalan dengan versi berbeda, stack dihentikan lalu dijalankan ulang. Node.js tidak butuh restart.
- `dl ui` saat aplikasi sudah berjalan akan menampilkan jendela yang tersembunyi.
- Service MySQL ditampilkan dengan versi binary, misalnya `mysql-8-4-10`; data dan kredensial tetap per seri `8.4`.
- Jika DevLok pernah dijalankan elevated, service tersebut dimiliki Administrator dan `dl down` biasa tidak bisa menghentikannya. Hentikan lewat terminal Administrator atau restart Windows.
- Perubahan hosts memunculkan UAC sekali; helper `dl-helper.exe` menulis hosts dengan allowlist domain.

## Perintah

Sudah tersedia:

```powershell
dl doctor [--json] [--manifest path]
dl init [path] [--name demo] [--root public] [--php 8.4] [--domain app.local] [--no-db]
dl version [--json]
dl runtime list [--json]
dl runtime add php@8.4 [--offline]
dl runtime add notepadpp
dl runtime remove php@8.3.35
dl runtime verify [php@8.4]
dl update [--check] [--json]
dl update download
dl update apply --yes
dl update rollback --yes
dl ports [--json] [--port 9000]
dl up [--path dir] [--skip-hosts] [--skip-trust]
dl down
dl status
dl site create demo [--php 8.4] [--domain app.local] [--no-db]
dl site list
dl site remove demo --delete
dl site adopt C:\laragon\www\siso
dl edit [path]
dl db list
dl db create [name]
dl db drop <name> --yes
dl db backup [name] [-o file.sql.gz]
dl db restore <name> -f file.sql.gz --yes
dl db seed [name]
dl db shell [name]
dl db creds [name] [--root]
dl db admin
dl lock
dl export [-o env.zip]
dl import laragon C:\laragon [--dry-run] [--yes] [--adopt]
dl run -- php artisan
dl shell
dl ui
```

Direncanakan (lihat roadmap):

```powershell
dl ui --browser
```

Daftar lengkap: [docs/CLI.md](docs/CLI.md).

## Struktur repo

```
DevLok/
|-- cmd/
|   `-- dl/              CLI
|-- internal/
|   |-- cli/                perintah cobra
|   |-- config/             manifest, default, validasi, init
|   |-- db/                 MySQL instance, database/user, backup/restore
|   |-- devlokerr/          kode error + exit code
|   |-- doctor/             pemeriksaan lingkungan
|   |-- envarchive/         export manifest + lock + seed
|   |-- hosts/              blok hosts terkelola
|   |-- importlaragon/      parser dan migrasi Laragon
|   |-- instance/           single-instance lock
|   |-- logging/            JSONL + rotasi + redaksi
|   |-- platform/           abstraksi OS
|   |-- ports/              deteksi port dan owner
|   |-- runtime/            registry, downloader, verifikasi, ekstraksi
|   |   `-- registry/       manifest runtime (nginx, php, mysql, node, tools)
|   |-- secret/             DPAPI untuk kredensial
|   |-- stack/              orkestrasi up/down/status
|   |-- supervisor/         job object, health check, restart, reconcile
|   |-- tlsca/              CA lokal dan certificate per domain
|   |-- web/                generator config Nginx
|   `-- version/            info versi
|-- docs/
|-- desktop/                GUI Wails + Svelte
|   |-- main.go
|   |-- app.go
|   `-- frontend/
|-- .github/workflows/      CI Windows + macOS
`-- build/                  output build (ignored)
```

Paket yang menyusul: `db`, `importlaragon`, `api`, `desktop/`, `templates/`, `installer/`.

## Dokumentasi

| Dokumen | Isi |
|---|---|
| [docs/PRD.md](docs/PRD.md) | Kebutuhan produk, user story, kriteria terima |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Arsitektur, alur data, proses, direktori |
| [docs/STACK.md](docs/STACK.md) | Keputusan stack dan versi tool |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Milestone dan estimasi |
| [docs/BACKLOG.md](docs/BACKLOG.md) | Backlog per task |
| [docs/MANIFEST.md](docs/MANIFEST.md) | Spesifikasi `.devlok.yaml` dan lockfile |
| [docs/REGISTRY.md](docs/REGISTRY.md) | Skema registry runtime dan aturan verifikasi |
| [docs/CLI.md](docs/CLI.md) | Spesifikasi CLI |
| [docs/SECURITY.md](docs/SECURITY.md) | Model ancaman dan kontrol keamanan |
| [docs/TESTING.md](docs/TESTING.md) | Strategi dan matriks pengujian |
| [docs/PLATFORM.md](docs/PLATFORM.md) | Perbedaan Windows/macOS dan lapisan abstraksi |
| [docs/LICENSING.md](docs/LICENSING.md) | Lisensi runtime dan distribusi |
| [docs/UPDATES.md](docs/UPDATES.md) | Kebijakan update aplikasi dan runtime |
| [docs/RELEASE.md](docs/RELEASE.md) | Build portable, installer, GitHub Release, dan rollback |
| [docs/TIMELINE.md](docs/TIMELINE.md) | Timeline fitur 0.1.0 sampai rencana 0.2.0 |
| [docs/LARAGON-GAPS.md](docs/LARAGON-GAPS.md) | Audit kekurangan Laragon vs DevLok |
| [docs/STITCH_PROMPTS.md](docs/STITCH_PROMPTS.md) | Prompt Google Stitch untuk alternatif UI app dan tray |
| [docs/adr/](docs/adr/) | Architecture Decision Records |

## Status milestone

| Milestone | Status |
|---|---|
| M0 Fondasi | Sebagian besar selesai (migrasi schema dan forwarding pipe pending) |
| M1 Runtime manager | Selesai (inti) |
| M2 Supervisor | Selesai (inti) |
| M3 Site, hosts, TLS | Selesai (inti) |
| M4 Database | Selesai (inti) |
| M5 Reproducibility dan import | Selesai (inti) |
| M6 GUI | Selesai (tray tersedia; browser fallback pending) |
| M7 Installer dan hardening | Installer, release, updater, dan rollback selesai; Authenticode menunggu certificate |

## Lisensi

Lisensi kode DevLok belum ditentukan. Runtime pihak ketiga tidak dibundel dan diunduh dari sumber resmi. Lihat [docs/LICENSING.md](docs/LICENSING.md).
