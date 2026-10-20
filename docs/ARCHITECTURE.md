# Arsitektur DevLok

## 1. Prinsip

1. **Satu core, dua antarmuka.** Logika ada di paket Go internal; CLI dan GUI hanya pemanggil.
2. **Source of truth = manifest.** Konfigurasi Nginx, hosts, dan database diturunkan dari `.devlok.yaml` + `.devlok.lock`, bukan diedit manual.
3. **Least privilege.** GUI berjalan non-admin. Elevation hanya untuk hosts dan firewall.
4. **Loopback-only.** Semua service listen `127.0.0.1` dan `::1`.
5. **Reconcile, bukan asumsi.** Setiap start membaca state nyata (PID, port, file) lalu memperbaiki selisih.
6. **Portable by design.** Tidak hardcode path Windows di core; semua lewat lapisan `platform`.
7. **No hidden magic.** Setiap perubahan pada sistem file atau proses dicatat dan bisa di-rollback.

## 2. Diagram komponen

```
+-------------------------------------------------------------+
|                           dl.exe                            |
|                                                             |
|  +--------------+   +--------------+   +----------------+  |
|  |  CLI (cobra) |   |  Wails GUI   |   |  Tray          |  |
|  +------+-------+   +------+-------+   +-------+--------+  |
|         +------------------+-------------------+           |
|                    +-------v--------+                      |
|                    |  API / Service |                      |
|                    |  (Go packages) |                      |
|                    +-------+--------+                      |
|      +---------+-----------+-----------+----------+        |
|      v         v           v           v          v        |
|  +-------+ +-------+ +----------+ +-------+ +--------+    |
|  |Runtime| |Super- | |Web config| |Hosts/ | |Database|    |
|  |manager| |visor  | |generator | |TLS    | |manager |    |
|  +---+---+ +---+---+ +----+-----+ +---+---+ +---+----+    |
|      |         |          |           |         |         |
+------+---------+----------+-----------+---------+---------+
       v         v          v           v         v
  %USERPROFILE%\.devlok  Nginx     hosts     MySQL
  runtimes/ cache/       php-cgi   store     data/
  logs/ certs/           node
```

## 3. Proses dan topologi runtime

### 3.1 Proses utama

`dl.exe` adalah CLI. `devlok-gui.exe` adalah GUI Wails yang memakai paket core yang sama. GUI dibuka lewat `dl ui`.

Hanya satu instance owner per state dir. Dijaga dengan named mutex Windows. Instance kedua mengirim perintah lewat named pipe ke owner dan keluar.

### 3.2 Proses anak

| Proses | Peran | Catatan |
|---|---|---|
| `nginx.exe` | Web server | Satu instance, banyak vhost |
| `php-cgi.exe` | FastCGI worker | Pool per versi PHP yang dipakai |
| `mysqld.exe` | Database | Instance per versi mayor |
| `dl-helper.exe` | Operasi privileged | On-demand, umur pendek |

PHP di Windows tidak punya `php-fpm`. FastCGI dijalankan dengan `php-cgi.exe -b 127.0.0.1:<port>` dan `PHP_FCGI_CHILDREN` untuk konkurensi. Satu pool per versi PHP; vhost memilih pool lewat upstream berbeda.

### 3.3 Job Object

Semua proses anak dimasukkan ke Job Object dengan `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. Konsekuensi:

- Service stack berjalan dengan Job Object persisten agar tetap hidup setelah CLI/GUI ditutup.
- `dl down` menghentikan semua service, termasuk proses anak.
- Crash owner tidak otomatis mematikan service; rekonsiliasi PID + start time dan `dl ports` mendeteksi kondisi nyata.
- Job Object non-persisten dipakai untuk proses yang harus mati bersama induk.

PID saja tidak cukup. State menyimpan PID + process creation time + executable path. Rekonsiliasi memverifikasi ketiganya agar PID reuse tidak disalahartikan.

## 4. Direktori

### 4.1 State pengguna

```
%USERPROFILE%\.devlok\
|-- config.json          preferensi global
|-- state.json           status runtime terakhir
|-- runtimes\
|   |-- nginx\1.30.6\
|   |-- php\8.4.26\
|   |-- mysql\8.4.10\
|   |-- node\22.23.3\
|   `-- tools\composer\2.10.3\
|-- cache\downloads\     arsip asli + checksum
|-- certs\
|   |-- ca.crt
|   |-- ca.key.dpapi
|   `-- sites\<site>.crt|key
|-- generated\
|   |-- nginx\nginx.conf
|   |-- nginx\sites\<site>.conf
|   `-- php\<version>\php.ini
|-- data\
|   |-- mysql\8.4\      datadir
|   |-- mysql\8.4\my.ini
|   `-- logs\
|-- logs\                JSONL per service
`-- backups\<site>\      dump database
```

### 4.2 Workspace project

```
<project>\
|-- .devlok.yaml
|-- .devlok.lock
|-- public\              document root (contoh Laravel)
`-- ...
```

Runtime tidak pernah ditulis ke dalam folder project. Yang masuk repo hanya manifest dan lockfile.

### 4.3 Portable mode

`dl --root D:\DevLokData up` memakai layout yang sama di root tersebut. Tidak ada penulisan ke `%USERPROFILE%`, `%APPDATA%`, atau registry.

## 5. Tanggung jawab komponen

### 5.1 `internal/config`

- Load/validasi `.devlok.yaml`.
- Merge manifest project + preferensi global.
- Baca/tulis `.devlok.lock`.
- Migrasi skema manifest antar versi.
- Tidak melakukan I/O jaringan atau spawn proses.

### 5.2 `internal/runtime`

- Resolve versi: `8.4` -> `8.4.23` berdasarkan lockfile atau registry.
- Unduh dengan resume, verifikasi SHA256 dan ukuran.
- Ekstrak aman: tolak path traversal, symlink keluar target, entri absolut.
- Deteksi arsitektur dan layout arsip.
- Menyediakan path executable dan environment yang dibutuhkan.

### 5.3 `internal/supervisor`

- Start/stop/restart proses.
- Job Object dan process group.
- Health check periodik.
- Restart policy dan backoff.
- Rekonsiliasi state saat startup.
- Mengumpulkan stdout/stderr ke logger.

### 5.4 `internal/ports`

- Cek ketersediaan port.
- Cek pemilik port (PID, nama proses, path).
- Alokasi port ephemeral untuk FastCGI.
- Strategi fallback 80/443 -> 8080/8443.
- Tidak pernah membunuh proses milik aplikasi lain.

### 5.5 `internal/web`

- Render template Nginx dari model vhost.
- Tulis config secara atomik.
- Validasi `nginx -t` sebelum reload.
- Reload graceful; rollback config jika validasi gagal.
- Menjaga config yang di-generate agar dapat dihapus otomatis.

### 5.6 `internal/hosts`

- Blok terkelola dengan marker.
- Backup + atomic replace.
- Deteksi konflik entri di luar blok.
- Flush DNS.
- Rollback saat gagal.

### 5.7 `internal/tls`

- Generate CA (ECDSA P-256) satu kali.
- Enkripsi private key dengan DPAPI.
- Generate leaf certificate per domain, SAN DNS + IP, masa berlaku 825 hari.
- Trust/untrust CA di user store.
- Verifikasi chain sebelum melaporkan sukses.

### 5.8 `internal/db`

- Inisialisasi datadir MySQL (`mysqld --initialize-insecure`).
- Set password root acak.
- Buat database + user per project.
- Backup/restore/seed.
- Menjalankan query admin lewat `mysql.exe` dengan kredensial sementara di environment, bukan argumen CLI.

### 5.9 `internal/logs`

- Writer JSONL dengan rotasi.
- Ring buffer untuk GUI.
- Tail dan follow.
- Redaksi nilai sensitif (password, token).

### 5.10 `internal/api`

- GUI Wails memanggil paket core lewat binding Go.
- HTTP API lokal untuk fallback browser direncanakan setelah v0.1.0.
- Operasi destruktif butuh konfirmasi eksplisit.

### 5.11 `internal/platform`

Interface minimum:

```go
type Platform interface {
    StateDir() (string, error)
    HostsPath() string
    WriteHosts(block []byte) error
    FlushDNS() error
    TrustCA(certPath string) error
    UntrustCA(certPath string) error
    KillTree(pid int) error
    ProcessStartTime(pid int) (time.Time, error)
    PortOwner(port int) (ProcessInfo, error)
    OpenBrowser(url string) error
    OpenTerminal(dir string, env []string) error
    SupportsElevation() bool
    Elevate(args []string) error
}
```

Implementasi: `platform_windows.go`, nanti `platform_darwin.go`. Core tidak boleh memanggil `golang.org/x/sys/windows` langsung.

## 6. Alur `dl up`

```
1. Baca .devlok.yaml
2. Baca .devlok.lock (jika ada); resolusi versi eksak
3. Acquire instance lock
4. Reconcile state.json dengan proses nyata
5. Pastikan runtime tersedia (cache -> unduh -> verifikasi -> ekstrak)
6. Cek port; tentukan port final
7. Generate config Nginx + php.ini + my.ini
8. Validasi config (nginx -t, mysqld --validate-config)
9. Start MySQL (jika belum), tunggu ready
10. Start pool php-cgi, tunggu FastCGI handshake
11. Start Nginx, health check HTTP
12. Pastikan hosts entries + certificate
13. Tulis state.json
14. Cetak URL dan status
```

Kegagalan di langkah mana pun memicu rollback terbatas: service yang baru dinyalakan dihentikan, config dikembalikan, hosts dikembalikan bila diubah.

## 7. Alur request

```
Browser https://siso.test
  -> DNS: hosts 127.0.0.1
  -> Nginx :443 (TLS 1.2/1.3, cert siso.test)
  -> location ~ \.php$ -> fastcgi_pass 127.0.0.1:<pool-port>
  -> php-cgi.exe (pool PHP 8.4)
  -> MySQL 127.0.0.1:3306 (db siso, user siso)
```

## 8. State dan rekonsiliasi

`state.json` menyimpan:

```json
{
  "schema": 1,
  "ownerPid": 1234,
  "ownerStart": "2026-10-20T10:00:00Z",
  "services": {
    "nginx": {"pid": 2001, "start": "...", "port": 443, "status": "running"},
    "php-8.4": {"pid": 2002, "start": "...", "port": 10987, "status": "running"},
    "mysql-8.4": {"pid": 2003, "start": "...", "port": 3306, "status": "running"}
  },
  "sites": {"siso": {"domain": "siso.test", "root": "C:/dev/siso/public"}}
}
```

Aturan rekonsiliasi:

- PID tidak ada -> status `stopped`.
- PID ada tetapi start time berbeda -> PID reuse, anggap `stopped`.
- PID ada dan cocok -> verifikasi port dan health.
- Owner PID mati -> semua service dianggap `stopped`; Job Object seharusnya sudah mematikan anak.
- Entri hosts/cert tanpa site -> tandai orphan, tawarkan cleanup.

## 9. Konkurensi dan locking

- Mutex per state dir mencegah dua owner.
- Mutex per service mencegah start/stop bersamaan.
- Semua penulisan config lewat satu writer serial.
- API memakai context dengan timeout; operasi panjang mengirim progress event.

## 10. Error model

Semua error internal memiliki:

- `code` stabil (contoh `PORT_IN_USE`, `RUNTIME_CHECKSUM_MISMATCH`, `HOSTS_PERMISSION_DENIED`).
- `message` untuk manusia.
- `hint` tindakan yang disarankan.
- `details` opsional (path, port, PID).

CLI memetakan `code` ke exit code. GUI menampilkan `message` + `hint` + tombol aksi.

## 11. Observability

- Log JSONL: `ts`, `level`, `service`, `event`, `msg`, `fields`.
- `dl doctor --json` untuk diagnosa.
- `dl support bundle` mengumpulkan log, config, dan versi tanpa data sensitif.
- Tidak ada telemetri keluar secara default.

## 12. Build dan packaging

- Windows: `wails build` di `desktop/` menghasilkan `devlok-gui.exe`; `dl-helper.exe` dibangun terpisah (rencana).
- Frontend di-embed ke binary.
- Installer Inno Setup: per-user install, tanpa service Windows, opsi autostart.
- Portable zip: tanpa installer, state di folder.
- Code signing menyusul; checksum rilis dipublikasikan.

## 13. Keputusan yang belum final

- Nama modul Go dan organisasi GitHub.
- Format API: JSON-RPC vs REST.
- Apakah tray memakai Wails atau `systray` terpisah.
- Apakah MySQL 8.0 juga didukung di v0.1.0.
- Strategi update otomatis (v0.1.1).
