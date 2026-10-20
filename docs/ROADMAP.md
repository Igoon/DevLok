# Roadmap

## 1. Ringkasan

| Versi | Fokus | Target |
|---|---|---|
| v0.1.0 | Windows: Nginx + PHP + MySQL + tools, CLI + GUI | 14-15 minggu |
| v0.1.1 | Mailpit, auto-update, perbaikan stabilitas | +2-3 minggu |
| v0.1.2 | ARM64 native, MySQL 8.0 opsional | +2 minggu |
| v0.2.0 | macOS (arm64 + amd64), notarization | +3-4 minggu |
| v0.3.0 | Redis, PostgreSQL, Apache | TBD |

Estimasi: satu developer, full-time, tanpa gangguan besar. Part-time 20 jam/minggu: kalikan 2-2.5.

## 2. Milestone v0.1.0

### M0 - Fondasi (minggu 1)

**Deliverable**

- Repo, modul Go, layout paket.
- `dl doctor` versi awal.
- Loader + validator `.devlok.yaml`.
- Logging JSONL.
- `platform` interface + implementasi Windows minimum.
- CI build Windows (dan kompilasi core macOS).
- Named mutex single instance.

**Exit criteria**

- `go test ./...` hijau.
- `dl doctor` melaporkan OS, arsitektur, WebView2, VC++, port, dan state dir.
- Manifest invalid menghasilkan error dengan kode dan hint.
- Build reproducible di mesin bersih.

### M1 - Runtime manager (minggu 2-3)

**Deliverable**

- Registry manifest untuk Nginx, PHP, MySQL, Composer, Node, Adminer.
- Downloader dengan resume + progress.
- Verifikasi SHA256 dan ukuran.
- Ekstraksi aman (anti path traversal).
- `dl runtime list`, `runtime add`, `runtime remove`.
- Cache offline.
- Resolusi versi `8.4` -> versi eksak.

**Exit criteria**

- `dl runtime add php@8.4` berhasil di mesin bersih.
- Checksum salah ditolak dan file dihapus.
- Dua versi PHP berdampingan di disk.
- Mode offline memakai cache tanpa error.
- Tidak ada penulisan ke folder instalasi.

### M2 - Supervisor dan port (minggu 4)

**Deliverable**

- Supervisor proses dengan Job Object.
- Start/stop/restart/status.
- Health check TCP/HTTP/FastCGI.
- Restart policy dan backoff.
- Deteksi port + pemilik port.
- Fallback 80/443 -> 8080/8443.
- Rekonsiliasi `state.json`.

**Exit criteria**

- 50 siklus start/stop tanpa proses zombie.
- Kill `dl.exe` mematikan semua anak.
- Port bentrok terdeteksi < 1 detik dengan saran.
- PID reuse tidak disalahartikan sebagai service hidup.
- `dl status --json` akurat.

### M3 - Site, hosts, TLS (minggu 5-6)

**Deliverable**

- Model site + generator vhost Nginx.
- `site create`, `site list`, `site remove`.
- Blok hosts terkelola + rollback.
- CA lokal + certificate per domain.
- Trust CA tanpa admin.
- TLS 1.2/1.3, HTTP/2.
- Health check HTTPS end-to-end.

**Exit criteria**

- `https://<site>.test` terbuka tanpa peringatan di Chrome dan Edge.
- `openssl s_client -tls1` dan `-tls1_1` gagal.
- Hosts kembali ke kondisi semula bila penulisan gagal.
- `nginx -t` selalu dijalankan sebelum reload.
- Menghapus site membersihkan vhost, hosts, dan certificate.

### M4 - Database (minggu 7-8)

**Deliverable**

- Inisialisasi MySQL per versi mayor.
- Root password acak + proteksi DPAPI.
- Database + user per project.
- `db create`, `db drop`, `db backup`, `db restore`, `db seed`, `db shell`.
- Adminer terintegrasi.
- Health check dan shutdown bersih.

**Exit criteria**

- User project tidak bisa mengakses database project lain.
- Backup -> drop -> restore menghasilkan data identik.
- Kredensial tidak muncul di log, argumen proses, atau repo.
- MySQL hanya listen loopback.
- Shutdown tidak meninggalkan file lock.

### M5 - Reproducibility dan import (minggu 9-10)

**Deliverable**

- `.devlok.lock` lengkap dengan SHA256 dan platform.
- `lock`, `up`, `down`, `export`.
- `run` dan `shell` dengan environment project.
- `import laragon` dengan `--dry-run`.
- Migrasi project dari `C:\laragon\www`.

**Exit criteria**

- Repo baru + `dl up` di mesin kedua menghasilkan stack identik.
- Perbedaan manifest vs lock menghasilkan peringatan jelas.
- Import tidak mengubah apa pun saat `--dry-run`.
- Minimal satu project Laragon nyata berhasil dimigrasikan.

### M6 - GUI (minggu 11-13)

**Deliverable**

- Wails app + tray.
- Dashboard service dan site.
- Aksi start/stop/restart, buka browser/folder/terminal.
- Log viewer real-time.
- Wizard project.
- Pengaturan workspace, port, TLD.
- Fallback UI browser.

**Exit criteria**

- Semua aksi CLI inti tersedia di GUI.
- UI tidak freeze saat operasi panjang.
- Status berubah < 2 detik setelah aksi.
- Tray mencerminkan status agregat dengan benar.
- Tidak ada crash setelah 8 jam idle.

### M7 - Installer dan hardening (minggu 14-15)

**Deliverable**

- Installer Inno Setup per-user.
- Portable zip.
- Deteksi dependency (WebView2, VC++).
- Uninstall bersih.
- `support bundle`.
- Dokumentasi pengguna.
- E2E test di Windows 10 dan 11.

**Exit criteria**

- Install/uninstall tidak meninggalkan service atau autostart.
- Uninstall tidak menghapus project pengguna.
- Installer berjalan tanpa admin.
- E2E lulus di Win10 22H2 dan Win11.
- Ukuran installer < 15 MB tanpa runtime.

## 3. Dependency antar milestone

```
M0 -> M1 -> M2 -> M3 -> M4
                 |      |
                 v      v
                M5 -> M6 -> M7
```

- M3 dan M4 bisa paralel bila ada dua developer.
- M6 bisa dimulai setelah M3 untuk UI read-only, tetapi aksi tulis menunggu M5.
- M7 tidak bisa dimulai sebelum M5.

## 4. Buffer dan risiko jadwal

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Hosts/elevation rumit | +1-2 minggu | Helper allowlist, uji awal di M0 |
| MySQL init di Windows | +1 minggu | Spike di M1, dokumentasikan flags |
| TLS trust store | +1 minggu | Uji `certutil` di M0 |
| Bug proses zombie | +1-2 minggu | Job Object sejak awal, stress test M2 |
| Scope creep fitur | +2-4 minggu | Non-goals dikunci di PRD |
| Notarization Mac | +2 minggu | Ditunda ke v0.2.0 |

Buffer keseluruhan: 15-20% dari total estimasi.

## 5. Setelah v0.1.0

### v0.1.1

- Mailpit.
- Auto-update bertanda tangan.
- Peningkatan log viewer.
- Perbaikan bug dari feedback.

### v0.1.2

- ARM64 native.
- MySQL 8.0 opsional.
- Apache opsional (dievaluasi).

### v0.2.0 macOS

- `platform_darwin.go`: hosts, `security add-trusted-cert`, process group, `lsof`.
- Packaging `.app` + `.dmg`.
- Code signing + notarization.
- dnsmasq opsional untuk wildcard.
- Runtime macOS arm64/amd64.
- Estimasi 3-4 minggu setelah Windows stabil.
