# Windows ARM64 (Riset Kelayakan)

Status: **riset/rencana**, belum diimplementasikan.

## 1. Ringkasan

Windows 11 on ARM dapat menjalankan binary **x64 melalui emulasi** (Prism). Karena itu, dukungan ARM64 untuk DevLok realistis dilakukan secara bertahap: binary DevLok native `windows/arm64`, sedangkan runtime inti (Nginx/PHP/MySQL) tetap memakai artifact x64 yang berjalan via emulasi.

## 2. Ketersediaan upstream (terverifikasi)

| Runtime | `windows/arm64` resmi | Keterangan |
|---|---|---|
| Node.js | Ya | `node-vXX-win-arm64.zip` (nodejs.org) |
| Mailpit | Ya | `mailpit-windows-arm64.zip` (GitHub Releases) |
| Composer | N/A | `composer.phar` (platform-independent, dijalankan PHP) |
| phpMyAdmin / Adminer | N/A | PHP murni |
| **Nginx** | **Tidak** | Rilis Windows hanya `nginx-x.y.z.zip` (x64) |
| **PHP** | **Tidak** | Rilis Windows hanya x64/x86 (`...-Win32-vs17-x64.zip`); tidak ada `arm64` |
| **MySQL** | **Tidak** | Hanya `mysql-x.y.z-winx64.zip` (x64) |

Kesimpulan: **tidak mungkin** native penuh untuk Nginx/PHP/MySQL. Satu-satunya jalur realistis adalah emulasi x64 untuk ketiganya.

## 3. Dampak ke DevLok

Masalah saat ini di host ARM64: `internal/runtime.registry.resolve` memakai `platformKey()` = `windows/arm64` dan mensyaratkan artifact ada untuk key itu. Semua runtime inti hanya punya `windows/amd64`, sehingga resolusi gagal dengan pesan "belum tersedia untuk windows/arm64".

Perbaikan inti (kecil, berdampak besar): **fallback artifact `windows/amd64` saat host `windows/arm64`** dan artifact arm64 tidak tersedia.

## 4. Desain yang diusulkan

### 4.1 Resolusi registry (fallback arch)

- `platformKey()` tetap `windows/arm64`.
- `resolve()`: bila artifact untuk key host tidak ada, coba `windows/amd64` (khusus Windows ARM64; dan `darwin/amd64` untuk Apple Silicon bila perlu — walau di macOS sudah pakai Homebrew).
- Tandai pada hasil bahwa artifact dijalankan via emulasi (untuk info di `dl runtime list`/GUI bila diinginkan).

### 4.2 Registry arm64 untuk yang tersedia

- Tambah artifact `windows/arm64` untuk Node dan Mailpit (URL + SHA256 resmi).
- Nginx/PHP/MySQL tetap `windows/amd64`.

### 4.3 Build & rilis

- Build target `windows/arm64`:
  - CLI: `GOOS=windows GOARCH=arm64 go build ./cmd/dl ./cmd/dl-helper`.
  - GUI: Wails mendukung `-platform windows/arm64` (perlu uji; WebView2 ARM64 tersedia).
- Installer (Inno Setup): tinjau `ArchitecturesAllowed`/`ArchitecturesInstallIn64BitMode`. `x64compatible` mencakup ARM64 di Inno 6.x, tetapi pola `Setup-*.exe` tunggal vs build arm64 terpisah perlu diputuskan (mis. satu installer yang cocok untuk x64 dan ARM64, atau `-arm64` terpisah).
- CI: tambah runner Windows ARM64 (GitHub-hosted arm64 Windows masih terbatas) atau build silang + uji di perangkat ARM.

### 4.4 Platform & proses

- Lapisan `internal/platform` Windows memakai API (Job Object, `GetExtendedTcpTable`, process start time) yang netral arsitektur — tidak perlu cabang arm64.
- Supervisor/health check tidak arsitektur-spesifik.

### 4.5 Keterbatasan yang harus didokumentasikan

- Emulasi x64 lebih lambat untuk MySQL/PHP (I/O dan CPU).
- Sebagian paket antivirus/driver x64 mungkin bermasalah di ARM.
- Binary GUI/CLI native arm64 tetapi runtime x64: campuran ini wajar dan harus jelas bagi pengguna.

## 5. Rekomendasi bertahap

1. **Fallback resolusi `windows/amd64` di host arm64** (kecil; membuka jalan).
2. Tambah artifact `windows/arm64` untuk **Node** dan **Mailpit**.
3. Build target **`windows/arm64`** untuk CLI + GUI; uji di Windows 11 ARM.
4. Tinjau installer Inno (satu file vs terpisah) dan CI.
5. Dokumentasikan keterbatasan emulasi di README/`docs/PLATFORM.md`.

## 6. Keputusan

Untuk rilis 0.2.x, fokus pada langkah **1** (fallback) sebelum pengerjaan penuh ARM64. Native ARM64 penuh untuk Nginx/PHP/MySQL berada di luar kendali upstream.
