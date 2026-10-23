# macOS

## Status

Dukungan macOS masih tahap pengembangan. Build `.app`/`.dmg` tersedia sebagai artifact GitHub Actions dan dilampirkan ke GitHub Release oleh workflow `release` (`DevLok-macos-arm64`/`-intel`), unsigned kecuali secret signing tersedia.

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

# GUI: temukan bundle .app di folder build atau /Applications
./dl ui
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
- `dl doctor` memeriksa Homebrew dan formula `nginx`/`php`/`mysql`.
- MySQL Homebrew dengan data DevLok.
- phpMyAdmin.
- Hosts melalui `osascript` dengan administrator prompt.
- Trust CA melalui `security` (login keychain); untrust memakai `security delete-certificate`.
- Build GUI Wails dan `.app`.
- `dl ui` menemukan bundle `.app` di folder build dan `/Applications`/`~/Applications`.
- `KillProcessesUnder` memakai `pgrep -f` untuk membersihkan proses di bawah folder runtime.
- Pemilik port lewat `lsof` (port default HTTP 8080 / HTTPS 8443).
- Tab Konfigurasi GUI terisi: versi Nginx/PHP/MySQL/Node diambil dari Homebrew (tidak bisa dipin per project).
- DMG unsigned berisi aplikasi, symlink Applications, README prasyarat, dan `dl-helper` di dalam bundle.
- CI Intel dan Apple Silicon (`macos-build` menghasilkan artifact arm64 dan intel).

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

- Tray/menu bar macOS memakai `systray` dengan `nativeStart` dijadwalkan ke main queue Cocoa (`dispatch_async_f`); perlu pengujian di perangkat.
- Installer macOS belum tersedia.
- Aplikasi belum ditandatangani Developer ID.
- Belum dinotarisasi; Gatekeeper dapat memblokir.
- Update aplikasi macOS belum tersedia.
- Runtime harus dipasang manual lewat Homebrew.
- Versi runtime mengikuti Homebrew, bukan versi per project.
- DevLok belum mengelola `brew services`.