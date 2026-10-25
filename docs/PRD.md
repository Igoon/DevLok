# PRD - DevLok v0.1.0 (Windows)

> Catatan: dokumen ini mendefinisikan scope v0.1.0 (Windows). Dukungan macOS sejak v0.2.0 dijelaskan di [MACOS.md](MACOS.md).

## 1. Ringkasan

DevLok adalah local development environment untuk Windows yang menggabungkan Nginx, PHP, MySQL, dan tool pendukung dalam satu aplikasi tray dan CLI. Fokus v0.1.0 adalah paritas fitur inti Laragon dengan perbaikan pada reproducibility, keamanan, dan operasional.

## 2. Tujuan

1. Developer bisa membuat situs lokal dan mengakses `https://nama.test` tanpa edit konfigurasi manual.
2. Versi runtime ditentukan per project dan dapat direproduksi di mesin lain.
3. Proses service terkelola: start, stop, health check, restart, dan pemulihan setelah crash.
4. Default aman: service hanya loopback, kredensial acak, TLS 1.2/1.3.
5. Semua aksi tersedia di CLI dan GUI.

## 3. Non-goals v0.1.0

- macOS dan Linux (target v0.2.0).
- Docker, container, WSL integration.
- Apache, Redis, PostgreSQL, RabbitMQ, Mailpit (Mailpit target v0.1.1).
- Deploy remote, tunnel publik, sinkronisasi cloud.
- Windows Service mode.
- Dukungan ARM64 native (target v0.1.2).

## 4. Persona

| Persona | Kebutuhan |
|---|---|
| Backend developer PHP | Banyak project, versi PHP berbeda, MySQL, HTTPS lokal |
| Fullstack developer | Node.js dan PHP dalam satu workspace |
| Anggota tim baru | Setup environment cepat dan identik |
| Developer pindah OS | Konfigurasi project sama di Windows dan macOS |

## 5. User story

| ID | Story | Prioritas |
|---|---|---|
| US-01 | Sebagai developer, saya ingin `dl init` menyiapkan project dari folder yang ada | Must |
| US-02 | Sebagai developer, saya ingin menambah versi PHP tanpa menimpa versi lain | Must |
| US-03 | Sebagai developer, saya ingin situs otomatis dapat diakses lewat `nama.test` | Must |
| US-04 | Sebagai developer, saya ingin HTTPS lokal tanpa peringatan browser | Must |
| US-05 | Sebagai developer, saya ingin project memakai versi PHP/Node yang dipin | Must |
| US-06 | Sebagai developer, saya ingin tahu service mana yang mati dan kenapa | Must |
| US-07 | Sebagai developer, saya ingin backup dan restore database satu perintah | Must |
| US-08 | Sebagai developer, saya ingin environment yang sama di mesin lain lewat lockfile | Must |
| US-09 | Sebagai developer, saya ingin memakai GUI tanpa kehilangan kemampuan CLI | Must |
| US-10 | Sebagai developer, saya ingin memigrasikan project Laragon yang sudah ada | Should |
| US-11 | Sebagai developer, saya ingin DevLok tidak mengubah PATH global | Must |
| US-12 | Sebagai developer, saya ingin service tidak terbuka ke jaringan lokal | Must |
| US-13 | Sebagai developer, saya ingin melaporkan masalah lewat `dl doctor` | Should |
| US-14 | Sebagai developer, saya ingin workspace bisa dipindah ke USB (portable) | Could |

## 6. Kebutuhan fungsional

### FR-01 Workspace dan project

- `dl init` membuat `.devlok.yaml` dan `.devlok.lock`.
- Deteksi otomatis framework: Laravel (`public/index.php`, `artisan`), WordPress (`wp-config.php`), Symfony (`public/index.php`), generic PHP.
- Workspace default `%USERPROFILE%\DevLok\sites`, dapat diubah.
- Nama project divalidasi: huruf kecil, angka, tanda hubung.

### FR-02 Runtime manager

- Registry manifest berisi versi, URL, SHA256, ukuran, dan layout arsip.
- Unduh ke cache, verifikasi SHA256, ekstrak ke folder versi.
- Mendukung Nginx 1.30.x, PHP 8.1-8.5, MySQL 8.4 LTS, Composer 2.x, Node 22 LTS, Adminer 6.1.2, dan phpMyAdmin 5.2.3.
- Runtime bersifat read-only setelah diekstrak.
- Versi berbeda berjalan berdampingan.
- Mode offline memakai cache; unduhan tidak tersedia menghasilkan error jelas.

### FR-03 Service lifecycle

- `start`, `stop`, `restart`, `status` untuk Nginx, PHP, MySQL.
- Health check: TCP untuk MySQL, HTTP untuk Nginx, FastCGI handshake untuk PHP.
- Restart policy: `no`, `on-failure` (default), `always`, dengan backoff.
- Proses anak dijalankan dalam Job Object agar mati bersama induk.
- Stale PID dideteksi lewat PID + waktu mulai proses.

### FR-04 Port

- Port default: 80/443 untuk HTTP/HTTPS, 3306 untuk MySQL, pool FastCGI acak di loopback.
- Jika 80/443 terpakai, tawarkan fallback 8080/8443 dan tampilkan URL final.
- `dl ports` menampilkan port, status, dan proses pemilik.
- Konflik tidak boleh menyebabkan service lain dihentikan.

### FR-05 Site dan virtual host

- `dl site create` membuat folder, vhost, entri hosts, dan certificate.
- Root document dapat berupa `public`, `web`, `www`, atau root project.
- Template Nginx mendukung PHP-FPM/FastCGI, `try_files`, gzip, HTTP/2, TLS 1.2/1.3, security header dasar.
- Konfigurasi generated disimpan di state dir, bukan folder instalasi.
- Perubahan manifest memicu regenerate config dan reload Nginx.

### FR-06 Hosts terkelola

- Entri ditulis dalam blok bertanda `# >>> DevLok managed block >>>`.
- Backup file hosts sebelum perubahan.
- Penulisan atomik: file sementara, verifikasi, replace.
- Rollback otomatis bila penulisan gagal.
- `ipconfig /flushdns` dipanggil setelah perubahan.
- Hanya butuh elevation saat menulis hosts.

### FR-07 TLS

- CA lokal dibuat sekali, disimpan dengan proteksi DPAPI.
- Certificate per domain dengan SAN DNS dan IP.
- Trust CA di `CurrentUser\Root` via `certutil`, tanpa admin.
- Default TLS 1.2 dan 1.3; TLS 1.0/1.1 ditolak.
- `dl tls status` melaporkan masa berlaku dan trust.
- Regenerate certificate saat domain berubah.

### FR-08 Database

- Instance MySQL per versi mayor yang dipakai.
- Database dan user per project dengan password acak.
- Root password acak, disimpan terenkripsi, tidak pernah ditulis ke repo.
- `bind-address=127.0.0.1`.
- `db create`, `db drop`, `db backup`, `db restore`, `db seed`, `db shell`.
- Backup menggunakan `mysqldump` dengan opsi konsisten dan kompresi.

### FR-09 Reproducibility

- `.devlok.lock` menyimpan versi eksak, SHA256, platform, dan parameter service.
- `dl lock` membuat atau memperbarui lockfile.
- `dl up` mengikuti lockfile; perbedaan manifest dan lock memicu peringatan.
- `dl export` menghasilkan arsip konfigurasi tanpa data sensitif.

### FR-10 Import Laragon

- `dl import laragon C:\laragon` memindai `www` dan `etc/nginx/sites-enabled/auto.*.conf`.
- Menampilkan rencana migrasi sebelum menulis apa pun.
- `--dry-run` wajib tersedia.
- Konversi SSL Laragon ke certificate DevLok.
- Project yang tidak dikenali masuk daftar manual review.

### FR-11 Log

- Setiap service menulis JSONL ke state dir.
- `dl logs <service> --follow` dan filter level.
- GUI menampilkan log terpadu dengan filter service dan pencarian.
- Rotasi berdasarkan ukuran, default 10 MB x 5 file.

### FR-12 GUI

- Tray icon dengan status agregat: running, partial, stopped, error.
- Dashboard: service, site, port, versi runtime, kesehatan.
- Aksi: start/stop/restart, buka browser, buka folder, buka terminal, backup DB.
- Wizard tambah project.
- Log viewer real-time.
- Pengaturan: workspace, port, TLD, autostart.
- Semua aksi GUI memakai API yang sama dengan CLI.

### FR-13 CLI

- Semua perintah mendukung `--json`.
- Exit code: `0` sukses, `1` error umum, `2` validasi, `3` konflik port, `4` butuh elevation, `5` dependency hilang.
- Output manusiawi secara default, non-interaktif bila `--json` atau `--yes`.

### FR-14 Doctor

- Cek: versi Windows, arsitektur, WebView2, VC++ runtime, port, hosts writable, trust CA, runtime cache, sisa proses, ruang disk.
- Output ringkas dengan status `ok`, `warn`, `fail` dan saran perbaikan.

### FR-15 Keamanan default

- Service listen `127.0.0.1` dan `::1` saja.
- Tidak ada kredensial default.
- CA private key terenkripsi DPAPI.
- Tidak mengubah PATH global.
- Tidak menambahkan exclusion antivirus secara otomatis.
- Firewall rule opsional dan scoped ke loopback.

### FR-16 Portable mode

- `--root <path>` memakai folder sebagai state dan runtime.
- Tidak menulis registry.
- Autostart dinonaktifkan pada mode portable.

## 7. Kebutuhan non-fungsional

| ID | Kebutuhan | Target |
|---|---|---|
| NFR-01 | Waktu start CLI | < 300 ms |
| NFR-02 | Waktu buka GUI | < 3 detik |
| NFR-03 | Waktu `up` dengan cache hangat | < 10 detik |
| NFR-04 | Memori idle tray + core | < 120 MB |
| NFR-05 | Ukuran installer | < 15 MB tanpa runtime |
| NFR-06 | Tanpa admin untuk operasi normal | Kecuali hosts dan firewall |
| NFR-07 | Dukungan path berspasi dan Unicode | Wajib |
| NFR-08 | Pemulihan setelah crash | State direkonsiliasi saat start |
| NFR-09 | Kompatibilitas | Windows 10 22H2+, Windows 11 x64 |
| NFR-10 | Tidak butuh Docker | Wajib |
| NFR-11 | Log tidak memuat password/token | Wajib |

## 8. Metrik keberhasilan v0.1.0

- Situs pertama aktif dalam < 5 menit sejak installer selesai.
- Nol edit manual pada `hosts`, config Nginx, atau config MySQL.
- Nol proses zombie setelah 50 siklus start/stop.
- `https://<site>.test` tanpa peringatan certificate di Chrome dan Edge.
- Repo baru + `dl up` menghasilkan stack identik di mesin bersih.
- Tidak ada perubahan PATH global setelah pemakaian normal.

## 9. Kriteria terima v0.1.0

1. Installer berjalan di Windows 10 22H2 dan Windows 11 x64.
2. Nginx, PHP 8.4, MySQL 8.4 terpasang dan berjalan.
3. Minimal dua versi PHP dapat dipakai oleh dua project berbeda secara bersamaan.
4. `https://<site>.test` valid dan TLS 1.0/1.1 ditolak.
5. Port 80 terpakai menghasilkan fallback yang bekerja dan URL yang jelas.
6. `dl db backup` dan `restore` menghasilkan data identik.
7. `dl lock` + `dl up` berhasil di mesin kedua.
8. `dl import laragon` memigrasikan minimal satu project dengan `--dry-run`.
9. GUI menampilkan status, log, dan aksi service tanpa error.
10. `dl doctor` melaporkan lingkungan dengan benar.
11. Uninstall menghapus binary dan service tanpa menyentuh project pengguna.

## 10. Asumsi dan ketergantungan

- Pengguna memiliki hak administrator untuk menulis hosts (UAC sekali per sesi).
- Koneksi internet tersedia saat pertama kali mengunduh runtime.
- Antivirus tidak memblokir binary yang tidak ditandatangani; code signing menyusul.
- Binary PHP/MySQL Windows membutuhkan Visual C++ Redistributable.
- Nginx di Windows tidak mendukung semua direktif Linux; template dibatasi pada fitur yang didukung.
- PHP di Windows memakai `php-cgi.exe` untuk FastCGI, bukan `php-fpm`.
