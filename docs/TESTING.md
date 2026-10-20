# Strategi Pengujian

## 1. Piramida

| Level | Porsi | Fokus | Tool |
|---|---|---|---|
| Unit | 60% | Logika murni, parser, resolver, generator | `go test` |
| Integrasi | 30% | Proses nyata, port, file, TLS, DB | `go test` + helper |
| E2E | 10% | Installer, GUI, alur pengguna | Playwright + PowerShell |

Target coverage baris: minimal 70% untuk paket core, 80% untuk parser/validator. Coverage bukan tujuan utama; skenario rollback dan error lebih penting.

## 2. Unit test wajib

- Parser dan validator `.devlok.yaml` (valid, invalid, unknown field, tipe salah).
- Migrasi schema manifest.
- Resolusi versi (`8`, `8.4`, `8.4.23`, tidak tersedia, ambigu).
- Perhitungan nama domain dan validasi bentrok.
- Generator config Nginx (golden file).
- Escape dan quoting argumen proses.
- Alokasi port dan strategi fallback.
- Redaksi log.
- Parser config Laragon.
- Perhitungan exit code dari kode error.

## 3. Integrasi

Semua test integrasi memakai state dir sementara (`t.TempDir()`) dan port acak.

| Skenario | Kriteria |
|---|---|
| Start/stop Nginx nyata | HTTP 200 dari document root |
| Start/stop php-cgi | FastCGI handshake sukses |
| Vhost + TLS | `https://site.test` valid, chain benar |
| TLS negasi | TLS 1.0/1.1 gagal |
| Hosts rollback | Simulasi gagal tulis, file kembali identik |
| MySQL init | Instance siap < 30 detik |
| DB isolation | User A ditolak saat akses DB B |
| Backup/restore | Checksum data sebelum dan sesudah sama |
| Kill induk | Semua anak mati, tidak ada proses sisa |
| PID reuse | State palsu tidak dianggap hidup |
| Checksum mismatch | Runtime tidak diekstrak, file dihapus |
| Arsip traversal | Ekstraksi ditolak |
| Port bentrok | Fallback bekerja, proses lain tidak terganggu |
| Config invalid | `nginx -t` gagal, config lama tetap aktif |

## 4. E2E

| Alur | Langkah |
|---|---|
| Instalasi baru | Install -> doctor -> init -> up -> buka HTTPS -> down |
| Dua project, dua PHP | Buat dua site dengan versi berbeda, akses keduanya |
| Reproduksi | Export repo + lock -> mesin bersih -> up -> verifikasi |
| Import Laragon | Dry-run -> migrasi -> akses -> rollback |
| GUI | Start/stop dari tray, log real-time, wizard project |
| Uninstall | Uninstall -> tidak ada autostart/service -> project utuh |
| Portable | Jalankan dari folder lain -> tidak menulis profil |

E2E GUI memakai Playwright terhadap UI browser fallback agar tidak bergantung pada WebView2 automation.

## 5. Matriks lingkungan

| Dimensi | Nilai |
|---|---|
| OS | Windows 10 22H2, Windows 11 23H2/24H2 |
| Arsitektur | x64 (v0.1.0), ARM64 (v0.1.2) |
| Path | `C:\dev`, path berspasi, path Unicode, path panjang |
| Jaringan | Online, offline dengan cache, proxy |
| Antivirus | Defender aktif, third-party sample |
| Privilege | User biasa, admin |
| Port | 80/443 bebas, 80 dipakai IIS, 3306 dipakai MySQL lain |

## 6. Uji beban dan ketahanan

- 50 siklus start/stop berturut: nol zombie, memori stabil.
- 100 reload Nginx berturut: tidak ada config rusak.
- 20 site aktif: waktu reload < 2 detik.
- Crash `dl.exe` saat service jalan: semua anak mati dalam < 5 detik.
- Matikan paksa MySQL: supervisor mendeteksi dan menerapkan policy.
- Disk penuh saat download/backup: error jelas, tidak ada file parsial aktif.
- Idle GUI 8 jam: memori tidak naik > 20%.

## 7. Uji keamanan

- `netstat -ano` memastikan semua service loopback.
- Percobaan akses dari mesin lain gagal.
- `openssl s_client` untuk TLS 1.0/1.1 gagal dan 1.2/1.3 sukses.
- Grep log dan support bundle untuk pola password/token.
- Fuzz parser manifest dan parser arsip.
- `govulncheck ./...`.
- Verifikasi ACL file CA key.

## 8. CI

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    strategy:
      matrix:
        os: [windows-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
      - run: go vet ./...
      - shell: bash
        run: test -z "$(gofmt -l .)" || (gofmt -l . && exit 1)
      - run: go test ./... -count=1
      - if: runner.os == 'macOS'
        run: go test ./... -race -count=1
      - run: go build ./...
```

Catatan:

- `-race` dijalankan di macOS karena Windows runner tidak selalu punya CGO/gcc.
- Test integrasi yang butuh binary runtime dijalankan di job terpisah dengan cache.
- macOS job hanya kompilasi dan unit test pada v0.1.0.
- E2E Windows dijalankan nightly dan sebelum rilis.

## 9. Manual QA sebelum rilis

- [ ] Installer di mesin bersih tanpa Go/Node.
- [ ] Installer di mesin dengan Laragon terpasang.
- [ ] Dua versi PHP aktif bersamaan.
- [ ] HTTPS hijau di Chrome dan Edge.
- [ ] Import Laragon dengan `--dry-run` dan eksekusi.
- [ ] Uninstall dan install ulang.
- [ ] Portable mode dari USB.
- [ ] `dl doctor` pada kondisi rusak (port bentrok, CA hilang, runtime hilang).
- [ ] Log viewer saat service crash.
- [ ] Backup/restore database project nyata.

## 10. Data uji

- Project contoh: Laravel minimal, WordPress minimal, PHP polos, Symfony minimal.
- Repo contoh untuk uji lockfile.
- Arsip uji: normal, traversal, symlink, checksum salah, file terpotong.
- Manifest uji: valid, field asing, versi tidak ada, path keluar project, YAML rusak.
