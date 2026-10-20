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
- DMG unsigned.
- CI Intel dan Apple Silicon.

## Batasan

- Installer macOS belum tersedia.
- Aplikasi belum ditandatangani Developer ID.
- Belum dinotarisasi; Gatekeeper dapat memblokir.
- Update aplikasi macOS belum tersedia.
- Runtime harus dipasang manual lewat Homebrew.
- Versi runtime mengikuti Homebrew, bukan versi per project.
- DevLok belum mengelola `brew services`.