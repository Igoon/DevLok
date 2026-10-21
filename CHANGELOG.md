# Changelog

Format mengikuti [Keep a Changelog](https://keepachangelog.com/id/1.1.0/) dan [Semantic Versioning](https://semver.org/lang/id/).

## [Unreleased]

## [0.1.4] - 2026-10-20

### Fixed

- Dashboard refresh setelah Start All memakai sequence guard dan polling sampai status running.
- Tab Dashboard me-refresh status saat dibuka.
- Tombol Database memberi pesan jika URL phpMyAdmin belum tersedia.
- Polling status dipercepat menjadi 3 detik.

## [0.1.3] - 2026-10-20

### Fixed

- Updater Windows memakai file `.cmd` sementara, menghindari error `Windows cannot find //`.
- Rollback Windows memakai mekanisme script yang sama.
- Icon aplikasi dan installer memakai icon DevLok.
- Update runtime dari GUI beserta peringatan migrasi MySQL.

## [0.1.2] - 2026-10-20

### Added

- Icon aplikasi DevLok menggantikan icon default Wails.
- Update runtime langsung dari tab Update GUI.
- Tombol update PHP/Nginx/MySQL/Node beserta perubahan project dan restart stack.

### Fixed

- Icon Windows tidak lagi memakai huruf W bawaan Wails.
- Update runtime tidak lagi harus lewat CLI.

## [0.1.1] - 2026-10-20

### Added

- Tab Update di GUI: cek update, unduh installer, apply, dan rollback.
- Update aplikasi via GitHub Releases dengan fallback checksum `.sha256`.
- Signed update manifest Ed25519 dan tool `devlok-release`.
- Installer check WebView2/Windows 22H2 dan smoke test otomatis.
- Release GitHub otomatis untuk ZIP, installer, checksum, dan manifest.
- CLI `dl` dengan `doctor`, `init`, `version`, `ports`, dan `runtime`.
- Manifest `.devlok.yaml` dengan validasi ketat dan default.
- Runtime manager: registry, download resume, SHA256, ekstraksi aman, mode offline.
- Registry Nginx, PHP 8.1-8.5, MySQL 8.4, Node 22, Composer, Adminer.
- Deteksi port dan proses pemilik di Windows.
- Supervisor proses: Job Object, health check, restart policy, rekonsiliasi state.
- Generator config Nginx per site, TLS 1.2/1.3, FastCGI pool.
- CA lokal, certificate per domain, trust tanpa admin.
- Hosts terkelola dengan backup dan rollback.
- Stack `dl up`, `dl down`, `dl status`.
- `dl site create/list/remove` dan workspace default.
- phpMyAdmin 5.2.3 sebagai aplikasi `phpmyadmin.test`.
- `dl edit` memakai VS Code, Notepad++, atau Notepad.
- MySQL 8.4 per project: init datadir, root password acak (DPAPI), database/user terpisah.
- `dl db list/create/drop/backup/restore/seed/shell/creds/admin`.
- Backup gzip konsisten dan restore terverifikasi.
- `.devlok.lock`: versi eksak, SHA256, platform, diverifikasi saat `dl up`.
- `dl lock`, `dl export`, `dl import laragon`, `dl run`, dan `dl shell`.
- `dl up` memakai lockfile dan memberi peringatan bila manifest berubah.
- GUI Wails: dashboard stack, site workspace, service, database, log, runtime.
- Aksi GUI: up/down, buka site, phpMyAdmin, folder, editor, terminal, buat site.
- `dl ui` meluncurkan `devlok-gui.exe`.
- Service latar dijalankan dengan `CREATE_NO_WINDOW`; tidak ada jendela konsol tambahan.
- GUI memakai single-instance lock; klik `dl ui` berulang tidak membuka banyak window.
- Cleanup proses runtime yatim berdasarkan port dan path runtime.
- `dl-helper.exe` untuk sinkronisasi hosts lewat UAC dengan validasi allowlist.
- Hosts idempotent: tidak menulis ulang/UAC jika domain tidak berubah.
- `web.domain` untuk nama virtual host bebas, termasuk domain multi-label.
- `ports.http`, `ports.https`, `ports.mysql`, `ports.php` untuk port eksplisit.
- `components.mysql` dan `components.phpmyadmin` untuk menyalakan/mematikan komponen.
- Flag `dl init`/`dl site create`: `--domain`, `--http-port`, `--https-port`, `--no-db`.
- GUI: toggle komponen Database/phpMyAdmin, checkbox database saat membuat site.
- GUI: tombol database membuka phpMyAdmin.
- MySQL 9.4.0 tersedia di registry dan bisa dipilih dari GUI/CLI.
- GUI service bar: toggle Database/phpMyAdmin dan dropdown versi MySQL.
- Service tabel hanya menampilkan service aktif/failed, bukan entri stopped lama.
- Tombol Buka phpMyAdmin di kartu Database.
- Tray native: Show, Open site, phpMyAdmin, submenu versi Nginx/PHP/MySQL/Node.js, Stop stack, Quit; close-to-tray.
- Tanda cek versi default/terpilih di tray.
- Icon tray hijau saat semua service utama running, abu-abu saat berhenti/parsial.
- Tab Konfigurasi GUI: domain, alias, port HTTP/HTTPS/MySQL/PHP, versi service, komponen.
- Simpan konfigurasi otomatis restart stack jika sedang berjalan.
- Status GUI membedakan running, partial, dan stopped.
- Log auto-refresh saat dashboard aktif, dengan tombol Pause/Follow.
- Runtime terpasang dipindah ke tab Konfigurasi; dashboard lebih ringkas.
- Save konfigurasi nonaktif jika belum ada perubahan.
- Tombol phpMyAdmin tidak lagi muncul tiga kali; tersisa di toolbar dan kartu Database.
- Konfigurasi service disusun per baris: checkbox, versi, dan port di samping nama service.
- Port kosong menampilkan placeholder default (80/443/3306) dan berarti otomatis.
- Nginx/PHP ditandai core; MySQL, phpMyAdmin, dan Node.js dapat diaktifkan/nonaktifkan.
- Konfigurasi menambah Document root, nama database, dan toggle HTTPS.
- Cek port live: free / in use beserta nama proses.
- Daftar runtime di Konfigurasi hanya menampilkan tool tambahan yang terpasang, tanpa menduplikasi service aktif.
- Dashboard: toolbar Start All yang berubah menjadi Stop saat stack berjalan.
- Dashboard: Web, Database, Terminal, Root.
- Slider service mengikuti status service sebenarnya.
- Versi Nginx/PHP/MySQL ditampilkan sesuai konfigurasi.
- Site workspace dan Log dipindah ke tab sendiri; form site baru ada di tab Sites.
- Tray menampilkan daftar site; klik site untuk mengaktifkan/membukanya.
- Footer menampilkan "Powered by DeepSeek-v4-Flash".
- Ganti versi service dari tray: update manifest/lock, restart stack jika sedang berjalan.
- Quit memaksa proses keluar setelah cleanup agar instance lock dilepas dan app bisa dijalankan ulang.
- `dl ui` memunculkan kembali jendela yang sudah berjalan.
- Service MySQL memakai nama versi binary (mysql-8-4-10), data per seri 8.4.
- Editor: Notepad++ portable (registry) dan deteksi VS Code langsung.
- Tab Import Laragon: scan, pilih project, import, dan adopt ke GUI.
- `dl site adopt <path>` untuk mendaftarkan project di luar workspace.
- `dl import laragon --adopt` mendaftarkan project setelah import.
- Logger JSONL dengan rotasi dan redaksi rahasia.
- CI Windows dan macOS.

### Changed

- Nama project menjadi DevLok; binary `dl.exe`, state dir `%USERPROFILE%\.devlok`.

### Planned

- Windows v0.1.0: Nginx, PHP multi-versi, MySQL, Composer, Node.js, Adminer.
- Runtime per project dengan `.devlok.yaml` dan `.devlok.lock`.
- Supervisi proses dengan Job Object, health check, dan restart policy.
- Deteksi port bentrok dan fallback 8080/8443.
- Virtual host otomatis, blok hosts terkelola, dan rollback.
- CA lokal, certificate per domain, TLS 1.2/1.3, trust tanpa admin.
- Database per project, password acak, backup/restore/seed.
- CLI lengkap dengan output `--json`.
- GUI tray: service, site, log, wizard project.
- Import project dan vhost dari Laragon.
- Installer Inno Setup dan mode portable.

## [0.1.0] - 2026-10-20

Fondasi Windows. Tidak dirilis sebagai stable terpisah; fitur digabung ke rilis 0.1.1.
