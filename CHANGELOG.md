# Changelog

Format mengikuti [Keep a Changelog](https://keepachangelog.com/id/1.1.0/) dan [Semantic Versioning](https://semver.org/lang/id/).

## [Unreleased]

### Changed

- Dokumentasi: hapus `docs/LARAGON-GAPS.md` dan `docs/POSTER.md`; tambah `docs/FITUR.md` sebagai ringkasan fitur netral. README dan `docs/APLIKASI.md` tidak lagi membandingkan dengan produk lain.

## [0.2.8] - 2026-10-24

### Changed

- Dokumentasi direkonsiliasi dengan status v0.2.7: README (dukungan Windows+macOS, update macOS, struktur repo, milestone), `docs/CLI.md` (perintah nyata; perintah fiktif dipindah ke "Direncanakan"), `docs/MACOS.md` (status rilis + pemisahan fitur/batasan), `docs/LARAGON-GAPS.md`, `docs/PLATFORM.md`, `docs/APLIKASI.md`, `docs/TIMELINE.md`, `docs/ROADMAP.md`, `docs/BACKLOG.md`, `docs/REGISTRY.md`, `docs/LICENSING.md`, `docs/ARCHITECTURE.md`, `docs/TESTING.md`.
- README: bagian "Masalah yang diselesaikan" (daftar kelemahan Laragon) diganti dengan "Kenapa DevLok" yang positif; `docs/LARAGON-GAPS.md` diubah menjadi perbandingan fitur yang netral; `docs/APLIKASI.md` memakai "Kemampuan Utama".

## [0.2.7] - 2026-10-24

### Fixed

- Dashboard: tombol **Editor** kini ada di toolbar (tersedia juga saat stack berjalan). Sebelumnya hanya muncul di state "stack belum berjalan".
- Konfirmasi aksi (update, hapus runtime, bersihkan cache, rollback) kini memakai modal in-app, bukan dialog native Wails, sehingga tidak menggantung/tak terlihat di Windows. Tombol "Periksa update" menampilkan status "Memeriksa…" selama proses.

## [0.2.6] - 2026-10-24

### Fixed

- macOS: `dl edit` dan tombol buka-editor di tray/GUI kini berfungsi. Sebelumnya `internal/editor` memakai logika Notepad++/Windows dan gagal di Mac; sekarang memilih VS Code/Cursor/Zed/Sublime/TextMate bila terpasang, fallback TextEdit (file) atau Finder (folder).
- Windows: update runtime kini juga menangani tool (phpMyAdmin, Notepad++, Adminer). Sebelumnya update tool gagal karena mencoba `SetServiceVersion`; tool kini dipasang tanpa mengubah versi project, dan update phpMyAdmin/Adminer me-restart stack bila aktif.

### Changed

- `docs/TESTING.md`: checklist smoke manual Windows GUI/tray.

## [0.2.5] - 2026-10-24

### Added

- macOS: `dl wildcard on|off|status [--tld test]` mengelola wildcard domain lewat dnsmasq (konfigurasi di state dir, port 5353, dan `/etc/resolver/<tld>` via prompt administrator).
- Docs: checklist kesiapan notarization di `docs/MACOS-SIGNING.md`.

### Changed

- Footer aplikasi: "AI-powered by DeepSeek-v4-Flash".

## [0.2.4] - 2026-10-24

### Added

- macOS: self-update aplikasi. `dl update apply` / tombol Update men-stage `DevLok.app` dari DMG/ZIP, mengganti bundle secara atomik lewat skrip terpisah setelah app keluar, lalu relaunch (fallback membuka DMG bila folder tidak writable).

## [0.2.3] - 2026-10-24

### Added

- macOS: update runtime lewat Homebrew. `dl update` dan tab Update menampilkan paket usang (`brew outdated` untuk nginx/php/mysql/node); tombol/aksi Update menjalankan `brew upgrade <formula>` lalu me-restart stack yang aktif.

## [0.2.2] - 2026-10-24

### Added

- Signed update manifest (`updates.json`) mendukung rilis per-platform lewat map `apps` (`darwin/arm64`, `darwin/amd64`); `cmd/devlok-release payload` menerima `-macos-arm64-url/-sha256` dan `-macos-intel-url/-sha256`, dan workflow `release` membangun manifest di job `manifest` setelah Windows + macOS.

## [0.2.1] - 2026-10-24

### Added

- macOS: dukungan formula Homebrew berversi. Bila `php@8.4`, `mysql@8.4`, dst. terpasang, DevLok otomatis memakai seri tersebut dan menampilkannya di dropdown Konfigurasi; `brew install php@8.4` cukup untuk pin seri 8.4.
- phpMyAdmin dipilih otomatis berdasarkan seri PHP: PHP >= 8.4 memakai `6.x` bila tersedia di registry, selain itu `5.2.x` (fallback otomatis).
- `dl db reset --yes` untuk memindahkan datadir MySQL ke folder `.bak` dan menginisialisasi ulang (mis. setelah berpindah versi MySQL yang menolak datadir lama).
- Update aplikasi macOS: `dl update`/tab Update mendeteksi aset `DevLok-macos-<arch>.dmg` (atau `.zip`) dari GitHub, `dl update download` mengunduh + verifikasi SHA256, dan `dl update apply` membuka DMG untuk pemasangan manual. Runtime macOS diarahkan ke `brew upgrade`.

### Fixed

- GUI macOS: `window.confirm` tidak didukung WKWebView sehingga tombol Hapus runtime, Bersihkan cache, dan konfirmasi update tidak bereaksi; kini memakai dialog native Wails lewat binding `App.Confirm`.
- Start All/deploy di macOS tidak lagi menampilkan warning "Runtime Homebrew dipakai ... versi per project tidak didukung" (memang by-design di macOS).
- Pesan kegagalan start MySQL kini menyertakan tail log error MySQL dan menyarankan `dl db reset --yes` bila datadir dibuat versi lain.

## [0.2.0] - 2026-10-24

### Fixed

- `dl ui` kini menemukan bundle `.app` di macOS; sebelumnya hanya mencari `devlok-gui.exe` sehingga GUI tidak pernah diluncurkan di Mac.
- `scripts/build-macos.sh` membangun `dl-helper` setelah `wails build -clean` agar helper tidak terhapus, dan memakai `find -print -quit` yang aman dari `pipefail`.
- Lock single-instance macOS/Linux mendeteksi lock basi (PID pemegang sudah mati) sehingga GUI tidak lagi keluar diam-diam tanpa menampilkan window setelah proses sebelumnya berhenti paksa.
- `OpenApp` macOS melaporkan error `open` (mis. Gatekeeper menolak) alih-alih gagal diam-diam.
- `dl ui`, `scripts/build-macos.sh`, dan workflow `macos-build` mencari bundle `.app` di `desktop/build` juga, bukan hanya `desktop/build/bin`.
- GUI macOS tidak lagi crash saat startup: `systray.Run` tidak lagi dipanggil di goroutine; tray macOS memakai `systray.Register` dan menjadwalkan `nativeStart` ke main queue Cocoa via cgo `dispatch_async_f`, sehingga tidak menabrak run loop Wails.
- Ikon aplikasi macOS/Windows kini memakai ikon DevLok "DL": `desktop/build/appicon.png` (1024×1024) dibuat oleh `scripts/make-icon.py` dan ikut di-commit, menggantikan ikon default Wails.
- Deteksi pemilik port macOS memakai `lsof` (sebelumnya tidak tersedia) sehingga `dl ports` menampilkan proses/PID di macOS.
- GUI macOS: tab Konfigurasi tidak lagi kosong. `ServiceVersions` mengembalikan versi Homebrew di macOS alih-alih error, dan kegagalan satu service kini tidak mematikan seluruh muatan tab (`Promise.allSettled`). `SetServiceVersion` di macOS tidak lagi resolve ke registry.
- Pencarian `brew` tidak lagi bergantung `PATH` (fallback ke `/opt/homebrew/bin/brew`, `/usr/local/bin/brew`).
- Trust CA macOS kini berfungsi: `tlsca` memakai `security add-trusted-cert`/`delete-certificate` ke login keychain (sebelumnya memakai stub `store_other.go` yang selalu error).
- Warning `dl up` di macOS diperbaiki: tidak lagi memunculkan "manifest meminta ." saat manifest tidak mengisi versi Nginx, dan pesan versi per project diperjelas. Pesan gagal hosts kini menampilkan output osascript asli.
- Nginx di macOS/Linux kini memakai `daemon off;` sehingga master tetap foreground dan dilacak supervisor; sebelumnya nginx daemonize sehingga dianggap "berhenti sebelum siap".
- `dl site list` menampilkan kolom `PATH` sehingga folder project terlihat langsung.
- Sinkronisasi hosts macOS diperbaiki: skrip `osascript` sebelumnya mengutip ganda (`do shell script ""…""`) sehingga selalu gagal `syntax error (-2740)`; kini hosts dapat ditulis setelah prompt administrator.

### Changed

- Workflow `macos-build` menghasilkan artifact arm64 (`macos-15`) dan intel (`macos-15-intel`), dengan nama `DevLok-macos-arm64` dan `DevLok-macos-intel`.
- Workflow `release` kini juga membangun macOS (arm64 + intel) dan melampirkan `DevLok-macos-<arch>.dmg`/`.zip` + `.sha256` ke GitHub Release.
- `dl ui` juga mencari bundle `.app` di `/Applications` dan `~/Applications`.
- `dl doctor` di macOS memeriksa Homebrew dan ketersediaan formula `nginx`, `php`, `mysql` (sebelumnya stub "belum lengkap"), serta melaporkan `dnsmasq`.
- `nginx.conf` (non-Windows) menetapkan `worker_rlimit_nofile 4096` sehingga warning `worker_connections` di macOS hilang; warning versi runtime Homebrew digabung menjadi satu pesan.
- `KillProcessesUnder` macOS memakai `pgrep -f` untuk membersihkan proses di bawah folder runtime; `WriteHosts` macOS menulis `/etc/hosts` secara langsung.
- `internal/update` dipisah per platform: eksekusi installer/rollback berada di `install_windows.go`; macOS/Linux mengembalikan `CodeUnsupported` dengan hint alih-alih mencoba `cmd.exe`. `InstallerCommand` tetap lintas platform.

## [0.1.10] - 2026-10-22

### Fixed

- Payload `updates.json` memakai generator Go agar daftar `runtimes` terserialisasi sebagai array yang valid.
- Verifikasi update aplikasi pada rilis 0.1.9 gagal karena format payload dari PowerShell.

## [0.1.9] - 2026-10-22

### Added

- Registry runtime dinamis dari `updates.json` bertanda tangan; cache 6 jam, fallback registry embedded.
- Release workflow ikut mempublikasikan daftar runtime di manifest update.
- Periksa update manual mengambil registry terbaru; auto-update/startup memakai cache.

## [0.1.8] - 2026-10-22

### Added

- Deteksi update runtime per seri terpasang (mis. PHP 8.4.26 ke 8.4.27) dan deteksi seri baru (mis. PHP 8.5.x) secara terpisah.
- Hapus runtime terpasang dari tab Konfigurasi; runtime yang masih dipakai project mana pun ditolak.
- Progress bar untuk unduh/verifikasi/ekstrak update runtime dan unduh installer update aplikasi.
- Tombol Batal dan Coba lagi pada proses update.
- Update semua patch runtime sekaligus dengan satu restart stack.
- Tampilan ukuran runtime, status pemakaian per project, hapus versi lama, dan bersihkan cache unduhan.
- Preferensi auto-update di tab Update: aplikasi, patch runtime, upgrade seri baru, dan interval pemeriksaan.
- Supervisor latar (`dl supervise`) memantau liveness dan health service, lalu restart dengan backoff setelah GUI/CLI ditutup.
- Registry PHP 8.4.27 dan 8.5.12.

### Changed

- Auto-update aplikasi dan runtime berjalan selama GUI hidup; update aplikasi tetap menghentikan stack sebelum installer.
- Quit dari tray tidak lagi menghentikan stack; gunakan Stop All atau `dl down`. Update/rollback aplikasi tetap menghentikan stack sebelum installer.

## [0.1.7] - 2026-10-22

### Fixed

- Unduhan runtime mencoba `url_fallbacks` bila URL utama gagal, mis. arsip PHP lama di windows.php.net.

## [0.1.6] - 2026-10-21

### Changed

- Icon aplikasi dan installer memakai inisial DL.

## [0.1.5] - 2026-10-20

### Fixed

- Tombol Database memakai URL phpMyAdmin langsung dari state stack, sama seperti tray.
- Dashboard refresh dan tombol Database konsisten setelah Start All.

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
