# macOS

## Status

Dukungan macOS masih tahap pengembangan. Build `.app`/`.dmg` tersedia sebagai artifact GitHub Actions, belum sebagai release resmi.

## Prasyarat

- macOS 14 atau 15.
- Intel atau Apple Silicon.
- Xcode Command Line Tools.
- Homebrew.
- Runtime Homebrew:

```bash
brew install nginx php mysql
```

phpMyAdmin diunduh otomatis oleh DevLok.

## Menjalankan

```bash
./dl up --path /path/ke/project
./dl status
./dl down
```

Port default macOS:

- HTTP: `8080`
- HTTPS: `8443`
- MySQL: `3306`, fallback otomatis ke port bebas.

## State

```text
~/Library/Application Support/DevLok
~/Library/Caches/DevLok
```

Data MySQL, kredensial, log, certificate, dan runtime shim berada di state dir DevLok.

## Yang Sudah Berfungsi

- Nginx Homebrew.
- PHP-FPM Homebrew.
- MySQL Homebrew dengan data DevLok.
- phpMyAdmin.
- Hosts melalui `osascript` dengan administrator prompt.
- Trust CA melalui `security`.
- Build GUI Wails dan `.app`.
- DMG unsigned berisi aplikasi, symlink Applications, README prasyarat, dan `dl-helper` di dalam bundle.
- CI Intel dan Apple Silicon.

## Tanpa Apple Developer Berbayar

Apple hanya mengizinkan distribusi macOS yang dipercaya secara otomatis melalui Developer ID dan notarization. Tanpa itu, tidak ada cara resmi agar aplikasi publik langsung lolos Gatekeeper.

Pilihan tanpa biaya:

1. **Build lokal**
   ```bash
   ./scripts/build-macos.sh
   ```
   Menghasilkan `dist/DevLok-macos-local.zip` dan `.dmg` dengan ad-hoc signature. Aplikasi yang dibangun sendiri di Mac biasanya tidak memiliki atribut quarantine, sehingga dapat dibuka.

2. **Tester tepercaya**
   Unduh DMG unsigned, lalu:
   ```bash
   xattr -dr com.apple.quarantine /Applications/DevLok.app
   ```
   Ini melewati Gatekeeper. Hanya untuk tester yang Anda percaya, bukan distribusi publik.

3. **Build dari source**
   Pengguna menjalankan build macOS sendiri. Ini menghindari masalah quarantine dan tidak membutuhkan certificate.

4. **Homebrew tap**
   Distribusi untuk pengguna teknis; belum diimplementasikan dan tidak menggantikan notarization.

Yang tidak disarankan:
- Mematikan Gatekeeper global dengan `spctl --master-disable`.
- Membagikan private key atau certificate self-signed sebagai solusi publik.

Untuk distribusi publik yang aman, tetap diperlukan Apple Developer Program berbayar.

## Batasan

- Installer macOS belum tersedia.
- Aplikasi belum ditandatangani Developer ID.
- Belum dinotarisasi; Gatekeeper dapat memblokir.
- Update aplikasi macOS belum tersedia.
- Runtime harus dipasang manual lewat Homebrew.
- Versi runtime mengikuti Homebrew, bukan versi per project.
- DevLok belum mengelola `brew services`.