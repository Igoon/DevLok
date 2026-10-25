# Fitur

Ringkasan fitur dan kemampuan DevLok per area. Untuk riwayat perubahan lihat [CHANGELOG](../CHANGELOG.md); untuk macOS lihat [MACOS.md](MACOS.md).

## Platform

- Windows 10 22H2+ / Windows 11 x64.
- macOS 14/15, Intel & Apple Silicon.
- GUI Wails (window + tray) dan CLI `dl` berbagi core yang sama.

## Runtime

- Windows: Nginx, PHP multi-versi, MySQL per seri, Node.js, Composer. Versi dipilih/pin per project lewat manifest + lockfile, diunduh dengan verifikasi SHA256 dan ekstraksi aman, serta mendukung mode offline dari cache.
- macOS: Nginx, PHP-FPM, dan MySQL dari Homebrew. Formula berversi didukung (`php@8.4`, `mysql@8.4`, dst.).

## Site, domain, hosts

- Manifest `.devlok.yaml`: domain, TLD, alias, document root, port, dan komponen.
- Generator vhost Nginx; config divalidasi (`nginx -t`) sebelum dijalankan.
- Hosts terkelola dalam blok bertanda dengan backup/rollback; di macOS lewat prompt administrator.
- Wildcard domain (`*.test`) opsional via dnsmasq (`dl wildcard`, macOS).

## TLS

- CA lokal (ECDSA P-256) dan certificate per domain, TLS 1.2/1.3.
- Trust/untrust otomatis: Windows `certutil`, macOS `security` (login keychain).

## Database

- MySQL per project dan per seri; user + password acak; kredensial terenkripsi (DPAPI di Windows, AES-GCM di non-Windows).
- `dl db create/drop/reset/backup/restore/seed/shell/creds`.
- phpMyAdmin (dipilih otomatis sesuai seri PHP) dan Adminer.

## Supervisor dan log

- Health check, restart dengan backoff, supervisor latar `dl supervise`, dan rekonsiliasi state.
- Log JSONL aplikasi + log per service; log viewer di GUI/tray.

## GUI, tray, CLI

- Tab Dashboard, Sites, Logs, Konfigurasi, Update, dan Import.
- Tray: status, start/stop, pilih versi service, serta tools (folder project, terminal, editor, URL, config/log).
- CLI lengkap untuk aksi inti dan automasi (`--json`).

## Update

- Aplikasi: manifest bertanda tangan per-platform. Windows memakai installer + rollback; macOS mendeteksi `.dmg`, mengunduh + verifikasi SHA256, lalu self-update `.app`.
- Runtime: registry bertanda tangan (Windows); `brew upgrade` (macOS).

## Reproducibility dan import

- Lockfile berisi versi eksak, SHA256, dan platform; `dl lock`, `dl export`, `dl import`.
- Import project yang sudah ada dari instalasi Laragon (scan, tulis manifest, adopt) tanpa memindahkan folder.

## Keamanan default

- Service hanya mendengarkan loopback.
- Password database acak dan terenkripsi.
- TLS modern (1.2/1.3), redaksi log, dan allowlist penulisan hosts.

## Batasan

- Windows ARM64 belum didukung.
- macOS: artifact belum di-sign/notarize, belum ada installer native, dan rollback update aplikasi belum tersedia.
- Browser fallback dan migrasi schema manifest belum tersedia.
