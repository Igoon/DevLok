# Backlog v0.1.0

Status per 2026-10-20: A1-A3, A5-A9 selesai; A4 (migrasi schema) dan A6 (forwarding named pipe) partial. B1-B11 selesai. C1-C9 selesai. D1-D12 selesai kecuali reload queue (D4) yang masih berupa restart. E1-E9 selesai. F1-F6 selesai. G1-G3 selesai; G4 pending. I1-I8 selesai (termasuk tray); I9 (fallback browser) dan I10 (uji idle) pending. Berikutnya: M7 installer dan hardening.

Per v0.2.9: M7 (installer/release/updater) selesai, auto-update dan supervisor latar selesai, Epic K (macOS) dirilis, Mailpit tersedia opsional. Sisa utama: A4 (migrasi schema), A6 (forwarding pipe), I9 (browser fallback), Windows ARM64, notarization macOS.

Estimasi dalam hari kerja ideal (1 hari = 6-7 jam fokus). Urutan mengikuti milestone. Status awal semua `todo`.

## Epic A - Fondasi

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| A1 | Inisialisasi repo, modul Go, layout `cmd/` + `internal/` | 0.5 | - | `go build ./...` hijau |
| A2 | Logger JSONL + rotasi 10 MB x 5 | 1 | A1 | Log valid JSON per baris, rotasi teruji |
| A3 | Loader + validator `.devlok.yaml` | 2 | A1 | Field wajib, tipe, enum, unknown field ditolak |
| A4 | Schema version + migrasi manifest | 1 | A3 | Manifest v0 dimigrasikan tanpa kehilangan data |
| A5 | `platform` interface + implementasi Windows minimum | 1.5 | A1 | State dir, hosts path, kill tree, process start time |
| A6 | Named mutex single instance + named pipe | 1.5 | A5 | Instance kedua meneruskan perintah lalu keluar |
| A7 | `dl doctor` v1 | 1.5 | A5 | Deteksi OS, arch, WebView2, VC++, port, state dir |
| A8 | CI Windows + kompilasi core macOS | 0.5 | A1 | Build hijau di PR |
| A9 | Konvensi error + kode stabil + hint | 1 | A1 | Semua error punya code/message/hint |

**Subtotal: 10.5 hari**

## Epic B - Runtime manager

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| B1 | Skema registry manifest runtime | 1 | A3 | Versi, URL, SHA256, size, layout, OS/arch |
| B2 | Registry Nginx 1.30.x Windows | 0.5 | B1 | Minimal 2 versi terdaftar |
| B3 | Registry PHP 8.1-8.5 NTS x64 | 1 | B1 | Minimal 5 versi terdaftar |
| B4 | Registry MySQL 8.4 LTS | 0.5 | B1 | URL + checksum terverifikasi |
| B5 | Registry Composer, Node 22, Adminer | 1 | B1 | Semua dapat diunduh dan diverifikasi |
| B6 | Downloader: resume, progress, timeout, proxy | 2 | B1 | Unduhan terputus dapat dilanjutkan |
| B7 | Verifikasi SHA256 + ukuran, hapus file rusak | 1 | B6 | Checksum salah menolak instalasi |
| B8 | Ekstraksi aman + deteksi layout | 2 | B7 | Path traversal dan symlink keluar ditolak |
| B9 | Resolusi versi `8.4` -> eksak + cache | 1.5 | B8 | Resolusi deterministik dan tercatat |
| B10 | Mode offline + pesan error jelas | 1 | B8 | Tanpa jaringan, cache dipakai; cache kosong -> hint |
| B11 | `runtime list`, `runtime add`, `runtime remove` + validasi disk | 2 | B9 | Hapus versi yang sedang dipakai ditolak |

**Subtotal: 13.5 hari**

## Epic C - Supervisor dan port

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| C1 | Wrapper proses + Job Object KILL_ON_JOB_CLOSE | 2 | A5 | Kill induk mematikan seluruh anak |
| C2 | State service + rekonsiliasi PID/start time/path | 2 | C1 | PID reuse tidak dianggap hidup |
| C3 | Start/stop/restart/status | 2 | C2 | Idempoten, tidak ada proses ganda |
| C4 | Health check TCP, HTTP, FastCGI | 2 | C3 | Status akurat < 2 detik setelah perubahan |
| C5 | Restart policy + backoff + batas percobaan | 1.5 | C4 | Crash loop tidak membebani CPU |
| C6 | Deteksi pemilik port (GetExtendedTcpTable) | 2 | A5 | Menampilkan PID, nama, path |
| C7 | Alokasi port dan fallback 8080/8443 | 1.5 | C6 | Konflik 80/443 tidak menghentikan service lain |
| C8 | Graceful shutdown Nginx/MySQL/PHP | 1.5 | C3 | Tidak ada file lock atau proses sisa |
| C9 | Stress test 50 siklus + perbaikan | 1.5 | C5, C8 | Nol zombie, state konsisten |

**Subtotal: 16 hari**

## Epic D - Web, hosts, TLS

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| D1 | Model site + validasi nama/domain/root | 1 | A3 | Nama invalid ditolak dengan hint |
| D2 | Template Nginx: PHP FastCGI, try_files, gzip, HTTP/2 | 2 | D1 | Dua framework berbeda jalan benar |
| D3 | Generator config atomik + `nginx -t` + rollback | 2 | D2 | Config invalid tidak pernah aktif |
| D4 | Reload graceful + reload queue | 1 | D3 | 10 perubahan berturut tidak race |
| D5 | Blok hosts terkelola + marker + backup | 2 | A5 | Entri di luar blok tidak tersentuh |
| D6 | Hosts atomic replace + rollback + flush DNS | 1.5 | D5 | Gagal tulis mengembalikan kondisi awal |
| D7 | CA lokal (ECDSA P-256) + DPAPI | 2 | A5 | Private key tidak bisa dibaca proses lain |
| D8 | Leaf cert per domain + SAN + renew | 2 | D7 | Cert valid untuk `site.test` dan `*.site.test` |
| D9 | Trust/untrust CA user store via certutil | 1.5 | D7 | Chrome/Edge tanpa peringatan, tanpa admin |
| D10 | TLS 1.2/1.3 only + verifikasi negasi | 1 | D8 | TLS 1.0/1.1 gagal |
| D11 | `site create`, `site list`, `site remove` end-to-end | 2.5 | D4, D6, D9 | Create dan remove bersih total |
| D12 | Health check HTTPS + laporan status | 1 | D10 | Status site mencerminkan kondisi nyata |

**Subtotal: 19.5 hari**

## Epic E - Database

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| E1 | Init MySQL datadir + `my.ini` loopback-only | 2 | B8 | Instance start dan siap menerima koneksi |
| E2 | Root password acak + DPAPI + rotate | 1.5 | E1, D7 | Password tidak muncul di log/argumen |
| E3 | Database + user per project + grant terbatas | 2 | E2 | User A tidak bisa akses DB B |
| E4 | `db create`, `db drop`, `db shell` | 1.5 | E3 | Shell memakai kredensial sementara |
| E5 | Backup `mysqldump` konsisten + kompresi | 2 | E3 | Backup saat write tetap konsisten |
| E6 | Restore + verifikasi checksum + rollback | 2 | E5 | Restore gagal tidak merusak DB aktif |
| E7 | Seed dari `db/seed.sql` + idempotensi | 1 | E3 | Seed dua kali tidak duplikasi data |
| E8 | Integrasi Adminer + auto-login aman | 1.5 | E3 | Adminer tidak menyimpan kredensial di URL |
| E9 | Instance per versi mayor + port allocator | 2 | C7, E1 | Dua versi MySQL berjalan berdampingan |

**Subtotal: 15.5 hari**

## Epic F - Reproducibility

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| F1 | Format `.devlok.lock` + checksum + platform | 1 | B9 | Lock deterministik |
| F2 | `dl lock` + diff manifest vs lock | 1.5 | F1 | Perbedaan menghasilkan peringatan |
| F3 | `dl up` orkestrasi penuh + rollback | 2.5 | C9, D11, E4 | Up berhasil dari repo bersih |
| F4 | `dl export/import` tanpa data sensitif | 1.5 | F3 | Export tidak memuat password/cert key |
| F5 | `dl run` + `dl shell` dengan env project | 1.5 | B11, F3 | PATH project benar, PATH global tidak berubah |
| F6 | Uji mesin kedua + dokumentasi | 1 | F3 | Stack identik, langkah terdokumentasi |

**Subtotal: 9 hari**

## Epic G - Import Laragon

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| G1 | Parser `sites-enabled/auto.*.conf` | 2 | D1 | server_name, root, SSL terbaca |
| G2 | Scan `www` + deteksi framework | 1 | G1 | Laravel/WordPress/generic terdeteksi |
| G3 | Rencana migrasi + `--dry-run` | 1.5 | G2 | Tidak ada penulisan saat dry-run |
| G4 | Eksekusi migrasi + laporan manual review | 2 | G3, F3 | Project berjalan setelah migrasi |

**Subtotal: 6.5 hari**

## Epic H - CLI polish

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| H1 | Cobra root + completion + help | 1 | A1 | `dl --help` lengkap |
| H2 | Output `--json` untuk semua perintah | 1.5 | A9 | Envelope konsisten, exit code benar |
| H3 | Mode non-interaktif + `--yes` | 1 | H2 | Aman untuk CI |
| H4 | `dl logs --follow` + filter | 1 | A2 | Follow tanpa kehilangan baris |
| H5 | `support bundle` redaksi sensitif | 1 | A7 | Tidak ada password/token di arsip |

**Subtotal: 5.5 hari**

## Epic I - GUI

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| I1 | Wails scaffold + Svelte + TS strict | 1 | A1 | App kosong berjalan di Windows |
| I2 | API lokal + token sesi + binding | 2 | A6 | Browser fallback berfungsi |
| I3 | Dashboard status service + site | 2 | C4, D11 | Data real-time, tanpa freeze |
| I4 | Aksi service: start/stop/restart | 1.5 | I3 | Konfirmasi untuk aksi destruktif |
| I5 | Log viewer real-time + filter + search | 2 | H4 | Auto-scroll dapat dijeda |
| I6 | Wizard project + deteksi framework | 2 | F3 | Project baru langsung jalan |
| I7 | Halaman database + backup/restore | 1.5 | E6 | Progress dan error terlihat |
| I8 | Pengaturan + tray menu + autostart | 2 | I2 | Autostart bisa dimatikan, portable menonaktifkan |
| I9 | Fallback UI browser + deep link | 1 | I2 | UI sama di browser |
| I10 | Uji idle 8 jam + perbaikan | 1.5 | I8 | Tidak ada memory leak signifikan |

**Subtotal: 16.5 hari**

## Epic J - Distribusi

| ID | Task | Est | Dep | Acceptance |
|---|---|---|---|---|
| J1 | Inno Setup per-user + opsi autostart | 2 | I8 | Install tanpa admin |
| J2 | Deteksi/bootstrap WebView2 + VC++ | 1.5 | J1 | Pesan jelas bila gagal |
| J3 | Portable zip + `--root` | 1 | F5 | Tidak menulis ke profil pengguna |
| J4 | Uninstall bersih | 1 | J1 | Service, autostart, dan cache terhapus; project aman |
| J5 | E2E Win10 22H2 + Win11 | 2 | J4 | Matriks lulus |
| J6 | Dokumentasi pengguna + troubleshooting | 2 | J4 | Situs pertama < 5 menit |
| J7 | Checksum rilis + catatan rilis | 0.5 | J5 | Artefak terverifikasi |

**Subtotal: 11 hari**

## Epic K - macOS (v0.2.0, belum masuk v0.1.0)

| ID | Task | Est |
|---|---|---|
| K1 | `platform_darwin.go` (hosts, trust, kill, port owner) | 3 |
| K2 | Runtime manifest macOS arm64/amd64 | 2 |
| K3 | Package `.app` + `.dmg` | 2 |
| K4 | Code signing + notarization | 2 |
| K5 | dnsmasq opsional untuk wildcard | 2 |
| K6 | Uji di macOS 14/15 | 2 |

**Subtotal: 13 hari (~3 minggu)**

## Ringkasan estimasi

| Epic | Hari |
|---|---|
| A Fondasi | 10.5 |
| B Runtime | 13.5 |
| C Supervisor | 16 |
| D Web/hosts/TLS | 19.5 |
| E Database | 15.5 |
| F Reproducibility | 9 |
| G Import | 6.5 |
| H CLI | 5.5 |
| I GUI | 16.5 |
| J Distribusi | 11 |
| **Total v0.1.0** | **123.5** |

Dengan buffer 15-20% dan overhead komunikasi, setara 14-15 minggu full-time.

## Aturan pengerjaan

1. Satu task = satu PR, maksimal ~400 baris perubahan non-generated.
2. Setiap PR wajib punya test untuk logika baru.
3. Task yang menyentuh hosts, TLS, atau database wajib punya test rollback.
4. Tidak ada task GUI sebelum API-nya stabil.
5. Task Mac tidak dikerjakan sebelum v0.1.0 dirilis.
