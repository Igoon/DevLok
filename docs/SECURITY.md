# Keamanan

## 1. Prinsip

1. Service lokal hanya mendengarkan loopback.
2. Tidak ada kredensial default.
3. Privilege sekecil mungkin; GUI tidak berjalan sebagai admin.
4. Semua unduhan diverifikasi checksum.
5. Rahasia tidak pernah masuk repo, log, atau argumen proses.
6. Setiap perubahan sistem file yang berisiko punya backup dan rollback.

## 2. Aset yang dilindungi

| Aset | Lokasi | Risiko |
|---|---|---|
| Root password MySQL | state dir, terenkripsi DPAPI | Akses database lokal |
| CA private key | `certs/ca.key.dpapi` | Penyalahgunaan certificate |
| Source code project | workspace pengguna | Kerusakan atau kebocoran |
| File hosts | `%WINDIR%\System32\drivers\etc\hosts` | Redirect domain |
| Runtime binary | state dir | Supply chain |
| Log | state dir | Kebocoran data sensitif |

## 3. Model ancaman

| Ancaman | Vektor | Dampak | Mitigasi |
|---|---|---|---|
| Malware lokal membaca password DB | File state | Tinggi | DPAPI per user, file ACL ketat |
| Malware lokal memakai CA | Private key | Tinggi | DPAPI, tidak ada ekspor otomatis |
| Supply chain runtime | Unduhan | Tinggi | SHA256 dari registry, HTTPS, tolak mismatch |
| Path traversal saat ekstrak | Arsip jahat | Tinggi | Validasi path, tolak symlink keluar |
| Service terekspos ke LAN | Bind address | Tinggi | Loopback-only, firewall scoped |
| Hosts dimanipulasi | Edit manual | Sedang | Blok terkelola, deteksi konflik |
| Log memuat rahasia | Debug verbose | Sedang | Redaksi, tidak pernah log argv kredensial |
| Elevation abuse | Helper | Tinggi | Allowlist operasi, validasi input, pipe ACL |
| Config injection | Manifest jahat | Sedang | Validasi ketat, template escaping |
| SQL injection via CLI | Argumen | Rendah | Parameterisasi, quoting benar |
| DoS lokal | Port/process | Rendah | Timeout, rate limit restart |

## 4. Kontrol per area

### 4.1 Hosts

- Blok bertanda jelas; entri di luar blok tidak diubah.
- Backup `hosts.devlok.bak` sebelum perubahan.
- Tulis ke file sementara di direktori yang sama, lalu `MoveFileEx` dengan replace.
- Verifikasi hasil baca ulang sebelum flush DNS.
- Rollback otomatis bila verifikasi gagal.
- Tidak menambahkan entri wildcard.

### 4.2 TLS dan CA

- CA ECDSA P-256, dibuat sekali per state dir.
- Private key dienkripsi DPAPI (CurrentUser).
- File key dengan ACL hanya user saat ini.
- Leaf certificate per domain, SAN eksplisit, masa berlaku 825 hari.
- Hanya TLS 1.2 dan 1.3; cipher suite modern.
- Trust hanya di `CurrentUser\Root`, tidak di machine store.
- `dl tls untrust` menghapus CA dari store.
- Tidak ada CA yang dibagikan antar pengguna.

### 4.3 Database

- `bind-address=127.0.0.1`, `skip-name-resolve`.
- Root password 32 karakter acak, disimpan terenkripsi.
- User project hanya mendapat grant pada database sendiri.
- Kredensial diteruskan lewat environment sementara atau file opsi dengan ACL, bukan argumen `-p`.
- Backup default disimpan di state dir, bukan project.
- `db drop` dan `db restore` butuh konfirmasi eksplisit.
- Log koneksi tidak mencatat password.

### 4.4 Proses

- Job Object dengan `KILL_ON_JOB_CLOSE`.
- Validasi executable path sebelum kill.
- Rekonsiliasi PID + start time untuk mencegah PID reuse.
- Restart policy dengan backoff untuk mencegah crash loop.
- Timeout pada semua health check.

### 4.5 Unduhan

- HTTPS dengan validasi sertifikat standar.
- Checksum SHA256 dari registry yang di-review.
- Ukuran file diverifikasi.
- File diunduh ke `.part`, diverifikasi, baru di-rename.
- Ekstraksi ke direktori sementara, verifikasi layout, baru dipindahkan.
- Tolak entri arsip absolut, `..`, symlink keluar target, hardlink mencurigakan.
- Runtime ditandai read-only setelah ekstrak.

### 4.6 Helper privileged

- `dl-helper.exe` hanya menerima operasi allowlist:
  - tulis blok hosts dengan konten yang divalidasi,
  - tambah/hapus firewall rule loopback,
  - flush DNS.
- Input divalidasi ulang di sisi helper; tidak mempercayai pemanggil.
- Named pipe dengan ACL hanya user saat ini.
- Token sesi acak, sekali pakai.
- Helper berumur pendek; tidak ada daemon privileged permanen.

### 4.7 API lokal

- Listen `127.0.0.1` dengan port acak.
- Token sesi Bearer wajib untuk semua request.
- CORS ditolak secara default; UI di-serve dari origin yang sama.
- Tidak ada endpoint yang menerima path arbitrer tanpa validasi.
- Operasi destruktif butuh token + konfirmasi.

### 4.8 Log dan support bundle

- Redaksi otomatis untuk field bernama `password`, `passwd`, `secret`, `token`, `key`, `authorization`.
- `support bundle` menghapus nilai rahasia dan path pribadi yang tidak relevan.
- Pengguna dapat meninjau isi bundle sebelum mengirim.

## 5. Default aman

| Item | Default |
|---|---|
| Bind address | `127.0.0.1`, `::1` |
| MySQL root password | Acak, bukan kosong |
| MySQL user project | Acak, grant terbatas |
| TLS minimum | 1.2 |
| HTTP/2 | Aktif |
| Directory listing | Nonaktif |
| `display_errors` | `On` untuk local, `Off` untuk log |
| `expose_php` | `Off` |
| Firewall | Tidak ada rule baru kecuali diminta |
| Autostart | Nonaktif sampai diaktifkan |
| Telemetri | Tidak ada |
| PATH global | Tidak diubah |

## 6. Non-goals keamanan

- DevLok bukan pengganti hardening server produksi.
- Tidak menjamin keamanan project yang dijalankan; kode project tetap tanggung jawab pengguna.
- Tidak melindungi dari malware dengan hak user yang sama.
- Tidak menyediakan enkripsi disk.

## 7. Checklist rilis

- [ ] Semua unduhan punya SHA256 dan terverifikasi.
- [ ] Tidak ada kredensial di repo, fixture, atau log.
- [ ] CA key terenkripsi dan ACL benar.
- [ ] Service hanya loopback (diverifikasi dengan `netstat`).
- [ ] TLS 1.0/1.1 ditolak.
- [ ] Helper menolak input di luar allowlist.
- [ ] Uninstall tidak meninggalkan CA di store tanpa pemberitahuan.
- [ ] `support bundle` teruji tidak memuat rahasia.
- [ ] Dependency Go diaudit (`govulncheck`).
- [ ] Binary rilis punya checksum dan ditandatangani bila memungkinkan.

## 8. Pelaporan masalah keamanan

Belum ada kanal resmi. Sementara: buka issue privat atau hubungi maintainer secara langsung. Jangan publikasikan detail eksploit sebelum ada perbaikan.
