# DevLok

## Identitas Aplikasi

| Item | Keterangan |
|---|---|
| Nama | **DevLok** |
| Arti | Development lokal |
| Jenis | Local development environment manager |
| Platform | Windows 10 22H2+ / Windows 11 x64; macOS 14/15 Intel & Apple Silicon |
| Versi | `0.2.7` |
| Lisensi | Belum ditentukan |
| GUI | `devlok-gui.exe` / `DevLok.app` |
| CLI | `dl.exe` |
| Helper privileged | `dl-helper.exe` |
| Bahasa | Go, TypeScript, Svelte |
| Repository | `C:\DevLok` |

DevLok adalah aplikasi untuk menyiapkan, menjalankan, dan mengelola lingkungan pengembangan web lokal di Windows dan macOS. Pendekatannya reproducible, aman secara default, dan berbasis konfigurasi per project.

---

## Kegunaan

DevLok menyatukan Nginx, PHP, MySQL, dan tool pendukung dalam satu aplikasi GUI dan CLI.

Pengguna dapat:

- Menjalankan beberapa project PHP dengan versi runtime berbeda.
- Membuat domain lokal seperti `project.test` tanpa mengedit hosts manual.
- Mengaktifkan HTTPS lokal dengan certificate yang dipercaya sistem.
- Membuat database dan user MySQL otomatis untuk setiap project.
- Menjalankan atau menghentikan seluruh service dari GUI, tray, atau CLI.
- Melihat status service, port, log, dan konfigurasi dari satu tempat.
- Mengimpor project dan virtual host dari Laragon.
- Membagikan konfigurasi project ke anggota tim melalui manifest dan lockfile.

DevLok ditujukan untuk developer yang membutuhkan lingkungan lokal yang konsisten, tetapi tidak ingin mengubah PATH global, memasang runtime secara manual, atau berurusan dengan konfigurasi Nginx dan hosts setiap kali membuat project baru.

### Kemampuan Utama

DevLok menyediakan:

1. Runtime per project dengan versi yang dapat dipin.
2. Lockfile dan verifikasi checksum runtime.
3. Supervisi proses, health check, dan pemulihan service.
4. Deteksi port bentrok sebelum service dijalankan.
5. TLS lokal dan trust certificate yang dikelola otomatis.
6. Kredensial MySQL acak dan terenkripsi.
7. Hosts terkelola dengan backup dan rollback.
8. Log terpusat dan dapat dibaca dari GUI.
9. Tanpa perubahan PATH global.
10. Environment yang dapat diekspor dan direproduksi.

---

## Fitur Utama

### GUI

- Dashboard status stack dan service.
- Tombol **Start All** yang berubah menjadi **Stop** saat stack berjalan.
- Toolbar **Web**, **Database**, **Terminal**, **Editor**, dan **Root**.
- Daftar service Nginx, PHP, MySQL, dan phpMyAdmin.
- Status service real-time: running, stopped, failed, restarting.
- Versi runtime ditampilkan di dashboard tanpa dropdown versi.
- Tab **Sites** untuk daftar project dan pembuatan site baru.
- Tab **Logs** untuk melihat log Nginx, PHP, MySQL, dan aplikasi.
- Tab **Konfigurasi** untuk domain, alias, document root, HTTPS, database, port, versi service, dan komponen.
- Tab **Update** untuk cek update aplikasi/runtime, download installer, apply, dan rollback. Update runtime dipisah antara patch seri terpasang (mis. 8.4.26 ke 8.4.27) dan seri baru (mis. 8.5.x), dengan progress bar untuk unduh, verifikasi, ekstrak, update project, dan restart stack.
- Preferensi **Pembaruan otomatis**: update aplikasi, patch runtime, upgrade seri baru, dan interval pemeriksaan. Default mati; update aplikasi otomatis tetap menghentikan stack sebelum installer.
- Tab **Import** untuk scan dan migrasi project Laragon.
- Pemeriksaan port langsung: free atau in use beserta nama proses.
- Simpan konfigurasi otomatis me-restart stack jika sedang berjalan.
- Tampilan tray dengan daftar site dan pemilihan versi service.

### Tray

- Status site dan health di bagian atas menu.
- **Start All** saat belum ada service berjalan.
- **Stop All** saat ada service berjalan.
- **Restart Stack** untuk menerapkan perubahan konfigurasi.
- Menu **Tools**: folder project, terminal, editor, copy URL, folder config, folder log, About.
- Label berubah otomatis mengikuti status service, termasuk perubahan dari GUI.
- Show DevLok, Open site, phpMyAdmin, dan Quit. Quit tidak menghentikan stack; supervisor latar tetap menjaga service sampai Stop All atau `dl down`.
- Submenu site untuk mengaktifkan atau membuka project.
- Submenu versi Nginx, PHP, MySQL, dan Node.js dengan tanda cek.
- Menu MySQL: phpMyAdmin, Start/Stop MySQL, version, create database, change/get root password, backup semua database, my.ini, mysqld.log, folder data, dan folder backup.
- Menu PHP: version, Quick settings (memory, upload, timeout), Extensions, php.ini custom, folder ext, dan php_errors.log.
- Menu Nginx: version, site config custom, error.log, dan access.log.
- CA bundle untuk cURL/OpenSSL tersedia otomatis agar Google OAuth dan HTTPS request dari PHP tidak gagal verifikasi.
- Icon hijau saat seluruh service utama berjalan; abu-abu saat berhenti atau parsial.
- Menutup jendela GUI menyembunyikannya ke tray, bukan menghentikan aplikasi.

### Runtime

- Registry runtime dengan URL resmi, ukuran file, dan SHA256.
- Download dengan resume, verifikasi checksum, dan mode offline.
- Ekstraksi aman: menolak path traversal, symlink keluar target, dan hardlink mencurigakan.
- Runtime dipasang di state directory, bukan dicampur dengan tool sistem.
- Versi runtime dapat dipilih per project.
- Pemeriksaan update runtime lokal lewat `dl update` dan menu tray **Updates (N)**; tidak ada auto-install.
- Deteksi update patch untuk setiap versi terpasang (mis. PHP 8.4.26 ke 8.4.27) dan deteksi seri baru (mis. PHP 8.5.x) secara terpisah.
- Hapus runtime terpasang dari tab Konfigurasi atau `dl runtime remove`; runtime yang masih dipakai project mana pun ditolak.
- Tab Konfigurasi menampilkan ukuran tiap runtime, status pemakaian, aksi hapus versi lama, dan pembersihan cache unduhan.
- Update semua patch runtime sekaligus; proses update dapat dibatalkan atau diulang dari progress bar.
- Nginx, PHP, MySQL, Node.js, Composer, Adminer, phpMyAdmin, dan Notepad++ portable.

### Service dan Supervisi

- Supervisor proses dengan Windows Job Object.
- Health check TCP/HTTP dengan timeout.
- Restart policy dan backoff.
- Rekonsiliasi PID dan start time untuk mencegah PID reuse.
- Service berjalan tanpa jendela konsol.
- Pembersihan proses runtime yatim.
- Deteksi port dan proses pemilik.
- MySQL otomatis memilih port bebas dan menyimpan port aktual.
- Nama service MySQL mengikuti versi binary, misalnya `mysql-8-4-10`.

### Domain dan TLS

- Virtual host Nginx dibuat otomatis per project.
- Domain default `<nama>.test`, dapat diganti dengan domain multi-label.
- Alias domain.
- CA lokal ECDSA P-256.
- Certificate per domain dengan SAN eksplisit.
- Hanya TLS 1.2 dan 1.3.
- Trust certificate di `CurrentUser\Root`, bukan machine store.
- HTTP/2 aktif dan directory listing nonaktif.

### Hosts

- Blok hosts DevLok ditandai dan tidak mengubah entri lain.
- Backup sebelum perubahan.
- Tulis atomik dan verifikasi baca ulang.
- Rollback otomatis jika verifikasi gagal.
- Flush DNS setelah perubahan.
- Helper `dl-helper.exe` dengan allowlist operasi dan UAC sekali.
- Tidak menulis ulang hosts atau meminta UAC jika domain tidak berubah.

### Database

- Instance MySQL per versi.
- Root password acak 32 karakter, disimpan dengan DPAPI.
- User dan database terpisah untuk setiap project.
- Grant hanya pada database milik project.
- MySQL hanya mendengarkan loopback.
- Perintah database melalui CLI dan GUI.
- Backup gzip, restore terverifikasi, dan seed.
- phpMyAdmin 5.2.3 pada domain `phpmyadmin.test`.
- Adminer tersedia sebagai tool database alternatif.

### Project dan Reproducibility

- Manifest project `.devlok.yaml`.
- Lockfile `.devlok.lock` berisi versi eksak, SHA256, dan platform.
- Verifikasi checksum runtime saat `dl up`.
- Peringatan jika manifest berubah dari lockfile.
- Export manifest, lockfile, dan seed tanpa rahasia.
- Import project dari Laragon.
- Adopsi project di luar workspace tanpa memindahkan folder.
- Workspace default: `%USERPROFILE%\DevLok\sites`.

### Keamanan

- GUI dan service berjalan sebagai user biasa, bukan Administrator.
- Service hanya mendengarkan `127.0.0.1` dan `::1`.
- Tidak ada password default.
- Rahasia tidak ditulis ke log atau argumen proses.
- Redaksi otomatis pada log.
- Semua unduhan diverifikasi SHA256.
- Helper privileged menerima operasi allowlist saja.
- Tidak ada telemetri.
- PATH global tidak diubah.
- Autostart nonaktif sampai diaktifkan.

### CLI

Perintah utama:

```powershell
dl doctor
dl init
dl up
dl down
dl status
dl ports
dl runtime list
dl runtime add php@8.4
dl runtime verify
dl update
dl site create
dl site list
dl site remove
dl site adopt
dl db list
dl db create
dl db drop
dl db backup
dl db restore
dl db seed
dl db shell
dl db creds
dl db admin
dl lock
dl export
dl import laragon
dl run -- php artisan
dl shell
dl edit
dl ui
```

Output `--json` tersedia untuk kebutuhan otomasi.

---

## Spesifikasi Teknis

### Stack

| Komponen | Spesifikasi |
|---|---|
| Bahasa backend | Go |
| Versi Go minimum | 1.25 |
| Framework GUI | Wails v2.17 |
| Frontend | Svelte 5 + TypeScript + Vite |
| Tray | `github.com/energye/systray` |
| CLI | Cobra |
| Konfigurasi | YAML |
| Database | MySQL |
| Web server | Nginx |
| PHP | PHP-CGI / FastCGI |

### Runtime yang Dikelola

| Komponen | Versi |
|---|---|
| Nginx | 1.30.6 |
| PHP | 8.1.34, 8.2.34, 8.3.35, 8.4.26, 8.5.11 |
| MySQL | 8.4.10, 9.4.0 |
| Node.js | 22.23.3 |
| Composer | 2.x |
| phpMyAdmin | 5.2.3 |
| Adminer | 6.x |
| Notepad++ | 8.9.8.1 |

Versi default PHP adalah 8.4. Versi default MySQL adalah 8.4.

### Port Default

| Layanan | Port |
|---|---|
| HTTP | 80, fallback 8080 |
| HTTPS | 443, fallback 8443 |
| MySQL | 3306, fallback 3306-3310 |
| PHP FastCGI | Port acak |
| phpMyAdmin | Mengikuti HTTP/HTTPS |

Port kosong atau `0` berarti otomatis. Port eksplisit yang sudah dipakai menghasilkan error, bukan fallback diam-diam.

### Lokasi Data

| Data | Lokasi |
|---|---|
| State aplikasi | `%USERPROFILE%\.devlok` |
| Runtime | `%USERPROFILE%\.devlok\runtimes` |
| Cache download | `%USERPROFILE%\.devlok\cache\downloads` |
| Config generated | `%USERPROFILE%\.devlok\generated` |
| Log | `%USERPROFILE%\.devlok\logs` |
| Certificate | `%USERPROFILE%\.devlok\certs` |
| Data MySQL | `%USERPROFILE%\.devlok\data\mysql` |
| Override PHP | `%USERPROFILE%\.devlok\config\php\<versi>.ini` |
| Override MySQL | `%USERPROFILE%\.devlok\config\mysql\<seri>.ini` |
| Override Nginx | `%USERPROFILE%\.devlok\config\nginx\<site>.conf` |
| Workspace site | `%USERPROFILE%\DevLok\sites` |
| Manifest project | `<project>\.devlok.yaml` |
| Lockfile project | `<project>\.devlok.lock` |

### Konfigurasi Runtime Lanjutan

File hasil generate tidak boleh diedit karena ditulis ulang setiap `Start All`:

- `generated\php\<versi>\php.ini`
- `generated\mysql\<seri>\my.ini`
- `generated\nginx\sites\<site>.conf`

Perubahan permanen dilakukan di file override:

- PHP: isi file ditambahkan ke akhir php.ini.
- MySQL: file dibaca melalui `!include` di my.ini.
- Nginx: file di-include di dalam blok `server`.

Tray membuka file override, bukan file generated maupun source `.go`. Konfigurasi MySQL dan Nginx divalidasi sebelum stack dijalankan.

### Konfigurasi Project

Contoh `.devlok.yaml`:

```yaml
name: siso
web:
  server: nginx
  root: public
  tld: test
  php: "8.4"
  https: true
runtime:
  php: "8.4"
services:
  mysql: "8.4"
db:
  name: siso
  charset: utf8mb4
  collation: utf8mb4_unicode_ci
ports:
  http: 0
  https: 0
  mysql: 0
  php: 0
components:
  mysql: true
  phpmyadmin: true
```

### Perilaku Versi Service

- Dashboard hanya menampilkan versi, tanpa dropdown.
- Pemilihan versi dilakukan di tab **Konfigurasi**.
- Dropdown versi Nginx muncul jika tersedia lebih dari satu versi.
- Jika hanya satu versi tersedia, versi ditampilkan sebagai teks.
- Perubahan versi disimpan ke manifest dan lockfile.
- Jika stack sedang berjalan, perubahan versi me-restart stack.
- Submenu versi di tray juga dapat mengubah versi.

### Batasan Saat Ini

- Dukungan resmi Windows x64 dan macOS 14/15 (Intel & Apple Silicon).
- Installer Windows tersedia; macOS memakai `.dmg`/`.zip` (belum ada installer dan belum notarized).
- Browser fallback dan migrasi schema manifest belum tersedia.
- Di macOS, runtime memakai Homebrew (bukan runtime per project); versi seri bisa dipin lewat formula seperti `php@8.4`.
- Linux belum menjadi target.
- Apache, Redis, PostgreSQL, Docker, deploy remote, dan Windows Service tidak termasuk scope.
- Aplikasi tidak boleh dijalankan sebagai Administrator.
- Jika DevLok pernah dijalankan elevated, service dapat dimiliki Administrator dan perlu dihentikan dari terminal Administrator atau restart Windows.
- Lisensi kode belum ditentukan.

---

## Alur Mulai Cepat

```powershell
# Cek lingkungan
.\build\bin\dl.exe doctor

# Buat site baru
.\build\bin\dl.exe site create demo --php 8.4

# Jalankan stack
.\build\bin\dl.exe up --path "%USERPROFILE%\DevLok\sites\demo"

# Cek status
.\build\bin\dl.exe status

# Buka GUI
.\build\bin\dl.exe ui
```

Build dari source:

```powershell
go build -trimpath -o build\bin\dl.exe ./cmd/dl
go build -trimpath -o build\bin\dl-helper.exe ./cmd/dl-helper
go test ./...

cd desktop
wails build -clean
copy ..\build\bin\dl-helper.exe build\bin\dl-helper.exe
```

---

## Dokumentasi Terkait

| Dokumen | Isi |
|---|---|
| [README.md](../README.md) | Gambaran umum, instalasi, dan perintah |
| [PRD.md](PRD.md) | Kebutuhan produk dan kriteria terima |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Arsitektur dan alur data |
| [MANIFEST.md](MANIFEST.md) | Spesifikasi manifest dan lockfile |
| [REGISTRY.md](REGISTRY.md) | Registry runtime dan verifikasi |
| [CLI.md](CLI.md) | Spesifikasi CLI |
| [SECURITY.md](SECURITY.md) | Model ancaman dan kontrol keamanan |
| [UPDATES.md](UPDATES.md) | Kebijakan update aplikasi dan runtime |
| [FITUR.md](FITUR.md) | Ringkasan fitur dan kemampuan |
| [ROADMAP.md](ROADMAP.md) | Rencana milestone |
