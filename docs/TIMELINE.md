# Timeline Fitur DevLok

Dokumen ini merangkum perjalanan fitur DevLok dari versi awal sampai rencana v0.2.0.

## Ringkasan

| Versi | Tanggal | Status | Fokus |
|---|---|---|---|
| `0.1.0-dev` | 20 Oktober 2026 | Snapshot pengembangan | Fondasi Windows: runtime, service, site, database, GUI, tray |
| `0.1.1` | 20 Oktober 2026 | Rilis stable pertama | Update system, installer, release pipeline, GUI update, hardening |
| `0.1.2` | TBD | Direncanakan | ARM64, MySQL 8.0 opsional, Mailpit, perbaikan stabilitas |
| `0.2.0` | TBD | Sedang dikerjakan | Dukungan macOS: platform, runtime, GUI, `.app`/`.dmg`, signing |

Rilis `0.1.0-dev`, `0.1.1-dev`, dan `0.1.1-dev.2` sampai `.4` adalah snapshot pengembangan sebelum `0.1.1` stable.

---

## 0.1.0-dev - Fondasi Windows

**Status:** snapshot pengembangan  
**Platform:** Windows 10 22H2+ / Windows 11 x64  
**Catatan:** tidak dirilis sebagai stable terpisah; fitur fondasi digabung ke `0.1.1`.

### Runtime dan Service

- Registry runtime dengan URL resmi, ukuran, dan SHA256.
- Download dengan resume, verifikasi checksum, ekstraksi aman, dan mode offline.
- Nginx 1.30.x.
- PHP 8.1 sampai 8.5, beberapa versi berdampingan.
- MySQL 8.4 dan 9.4.
- Node.js 22, Composer, Adminer, phpMyAdmin, Notepad++ portable.
- Supervisor proses dengan Windows Job Object.
- Health check TCP/HTTP, restart policy, dan rekonsiliasi state.
- Service berjalan tanpa jendela konsol.
- Deteksi port dan pemilik proses.

### Site, Hosts, dan TLS

- Manifest project `.devlok.yaml`.
- Lockfile `.devlok.lock` dengan versi eksak dan checksum.
- Virtual host Nginx otomatis.
- Domain `.test`, domain kustom, dan alias.
- Hosts terkelola dengan backup, verifikasi, dan rollback.
- Helper `dl-helper.exe` dengan UAC dan allowlist operasi.
- CA lokal, certificate per domain, TLS 1.2/1.3.
- Trust certificate di `CurrentUser\Root`.

### Database

- Instance MySQL per versi.
- Root password acak dengan DPAPI.
- User dan database terpisah per project.
- Grant hanya pada database project.
- Backup, restore, seed, shell, dan credential.
- phpMyAdmin pada `phpmyadmin.test`.
- Adminer sebagai alternatif.

### CLI dan GUI

- CLI `dl` dengan `doctor`, `init`, `up`, `down`, `status`, `runtime`, `site`, `db`, `lock`, `export`, `import`, `run`, `shell`, `edit`, dan `ui`.
- GUI Wails + Svelte.
- Dashboard service dan site.
- Start/Stop stack.
- Site workspace, log viewer, konfigurasi project, dan import Laragon.
- Tray native:
  - Show, Open site, Sites.
  - Start All/Stop All.
  - Restart Stack.
  - Menu Nginx, MySQL, Node.js, dan PHP.
  - Tools, About, dan Quit.
- Close-to-tray dan single-instance lock.
- Import Laragon dan adopsi project di luar workspace.
- Export environment tanpa rahasia.

### Keamanan dan Default

- Service hanya loopback.
- Password acak, bukan default.
- Log dengan redaksi rahasia.
- Download runtime diverifikasi SHA256.
- Tidak mengubah PATH global.
- Tidak ada telemetri.
- Aplikasi berjalan sebagai user biasa.

---

## 0.1.1 - Update, Installer, dan Hardening

**Status:** rilis stable  
**Platform:** Windows

### Update System

- `dl update` untuk memeriksa update aplikasi dan runtime.
- Update aplikasi dari GitHub Releases.
- Signed update manifest Ed25519.
- Tool `devlok-release` untuk keygen, pubkey, dan sign manifest.
- Default manifest:
  - `https://github.com/Igoon/DevLok/releases/latest/download/updates.json`
- Public key resmi di-embed di aplikasi.
- Fallback ke GitHub Releases API + checksum `.sha256`.
- Download installer dengan verifikasi checksum dan batas ukuran.
- Apply installer `.exe`/`.msi`.
- Rollback binary dari backup installer.
- Notifikasi update di tray.
- Tab **Update** di GUI:
  - Cek update aplikasi/runtime.
  - Unduh installer.
  - Apply update.
  - Rollback.
  - Salin perintah update runtime.

### Installer dan Release

- Inno Setup per-user tanpa admin.
- Direktori default `%LOCALAPPDATA%\Programs\DevLok`.
- Shortcut Start Menu dan opsi Desktop.
- Deteksi Windows 10 22H2 / Windows 11.
- Deteksi WebView2 dan tautan unduh resmi.
- Peringatan Visual C++ Runtime.
- Backup binary lama sebelum upgrade.
- Smoke test installer otomatis:
  - Install silent.
  - Jalankan `dl.exe version`.
  - Uninstall.
- GitHub Actions release:
  - Build CLI, helper, dan GUI.
  - Portable ZIP + checksum.
  - Installer + checksum.
  - Signed `updates.json`.
  - Upload artifact dan publish release.
- E2E update otomatis:
  - Client lama `0.0.1-dev`.
  - Deteksi update.
  - Download.
  - Install.
  - Verifikasi versi.
  - Uninstall.
- Dukungan Authenticode opsional lewat secret certificate.
- Release stable `v0.1.1`.

### Konfigurasi Lanjutan

- Override PHP: `%USERPROFILE%\.devlok\config\php\<versi>.ini`.
- Override MySQL: `%USERPROFILE%\.devlok\config\mysql\<seri>.ini`.
- Override Nginx: `%USERPROFILE%\.devlok\config\nginx\<site>.conf`.
- PHP override ditambahkan ke akhir `php.ini`.
- MySQL memakai `!include`.
- Nginx memakai `include` di dalam blok server.
- Validasi MySQL dengan `mysqld --validate-config`.
- Validasi Nginx dengan `nginx -t`.
- Tray membuka file override, bukan file generated.

### Tray dan GUI Polish

- Menu MySQL lengkap: phpMyAdmin, Start/Stop MySQL, versi, create database, root password, backup, my.ini, log, folder data/backup.
- Menu PHP lengkap: versi, quick settings, extensions, php.ini, folder ext, log.
- Quick settings PHP:
  - Memory limit.
  - Upload limit.
  - Max execution.
  - Sinkron ke `client_max_body_size` dan `fastcgi_read_timeout` Nginx.
- Extension PHP per versi.
- Status site dan health di tray.
- Tools: folder project, terminal, editor, copy URL, folder config/log, About.
- Nginx: site config custom, error log, access log.

### Perbaikan Penting

- Port MySQL aktual disimpan; konflik dengan Laragon di 3306 tidak lagi membuat Start All gagal.
- CA bundle PHP ditambahkan untuk `curl.cainfo` dan `openssl.cafile`; memperbaiki Google OAuth `cURL error 60`.
- Kredensial di macOS/non-Windows memakai AES-256-GCM, bukan plaintext.
- Line ending dinormalkan untuk CI.
- CI Windows membangun frontend sebelum `go vet`.
- Test failure ditampilkan di annotation dan job summary.

---

## 0.1.2 - Rencana Patch Windows

**Status:** direncanakan

### Kandidat Fitur

- Mailpit opsional untuk email development.
- ARM64 native.
- MySQL 8.0 opsional.
- Peningkatan log viewer.
- Perbaikan bug dari feedback.
- Upgrade GitHub Actions yang masih memakai Node 20.
- Pembersihan release lama.
- Dukungan Authenticode setelah certificate tersedia.

### Kriteria Selesai

- Semua test Windows hijau.
- Installer dan update E2E tetap hijau.
- Tidak ada regresi pada tray, updater, hosts, atau MySQL.
- Dokumentasi dan CHANGELOG diperbarui.

---

## 0.2.0 - macOS

**Status:** sedang dikerjakan  
**Platform target:** macOS Apple Silicon dan Intel

### Sudah Dikerjakan

- `platform_darwin.go`: hosts path, trust CA, process start time, browser, folder, editor, terminal, flush DNS.
- `privileged_darwin.go`: sinkronisasi hosts via `osascript` dengan administrator prompt.
- Build tag desktop mendukung Windows dan macOS.
- Frontend dibangun di CI macOS.
- CI macOS Intel (`macos-15-intel`) berhasil: vet, test, race test, dan build.
- Kode GUI/tray macOS berhasil dikompilasi.
- Workflow `macos-build` menghasilkan `.app` bundle unsigned sebagai artifact.

### Platform

- `platform_darwin.go`.
- Hosts via helper dengan administrator prompt.
- Trust CA via `security add-trusted-cert`.
- Process group dan process start time.
- Browser, folder, editor, dan terminal macOS.
- Flush DNS macOS.

### Runtime

- Registry `darwin/arm64`.
- Registry `darwin/amd64`.
- Nginx, PHP, MySQL, Node.js, dan tool pendukung untuk macOS.
- Path executable dan layout runtime macOS.
- CA bundle dan extension PHP.

### GUI dan Tray

- Build Wails untuk macOS.
- Tray macOS.
- Dashboard, site, log, konfigurasi, update, dan import.
- Perbedaan path dan permission macOS.

### Packaging

- `.app` bundle.
- `.dmg`.
- Code signing dengan Apple Developer ID.
- Notarization dan stapling.
- Gatekeeper compatibility.

### Prasyarat Eksternal

- Apple Developer account.
- Developer ID Application certificate.
- Notarization credentials.
- Runtime macOS yang tersedia dan dapat diverifikasi checksum.

### Kriteria Selesai

- Aplikasi berjalan di Apple Silicon dan Intel.
- Nginx, PHP, MySQL berjalan dan dapat dihentikan dengan bersih.
- Hosts dan TLS berfungsi.
- `.dmg` dapat dipasang dan dibuka tanpa peringatan Gatekeeper.
- Update aplikasi macOS berfungsi.
- Uninstall tidak menghapus project pengguna.

---

## Horizon Setelah 0.2.0

Belum masuk komitmen rilis:

- Redis.
- PostgreSQL.
- Apache opsional.
- Dukungan Linux.
- Auto-update lintas platform yang sepenuhnya otomatis.
- Perbaikan UX dan log viewer lanjutan.

---

## Catatan

- Tanggal rilis mengikuti waktu publikasi GitHub Release.
- Fitur yang sudah ada di source tetapi belum dirilis tidak dihitung selesai.
- Authenticode dan notarization macOS bergantung pada sertifikat/akun eksternal.
- Dokumen terkait: [UPDATES.md](UPDATES.md), [RELEASE.md](RELEASE.md), [ROADMAP.md](ROADMAP.md), [CHANGELOG.md](../CHANGELOG.md).
