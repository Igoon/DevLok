# macOS

## Status

Dukungan macOS tersedia sejak v0.2.0 dan dirilis (CLI, GUI, tray). Build `.app`/`.dmg` dilampirkan ke GitHub Release oleh workflow `release` (`DevLok-macos-arm64`/`-intel`), unsigned kecuali secret signing tersedia (lihat [MACOS-SIGNING.md](MACOS-SIGNING.md)).

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

## Wildcard domain (opsional, dnsmasq)

`/etc/hosts` tidak mendukung wildcard. Untuk `*.test` (mis. `apa.test`) tanpa entri hosts per-domain, aktifkan resolver lokal:

```bash
brew install dnsmasq
dl wildcard on --tld test      # tulis /etc/resolver/test (prompt admin) + jalankan dnsmasq :5353
dl wildcard status
dl wildcard off --tld test     # hentikan dnsmasq + hapus resolver
```

`dl wildcard on` menulis konfigurasi di state dir (port 5353, `address=/test/127.0.0.1`), menjalankan `dnsmasq`, lalu menulis `/etc/resolver/test` lewat prompt administrator. Setelah itu `test` dan subdomainnya diselesaikan ke `127.0.0.1`; entri hosts per-domain tidak lagi diperlukan. `dl doctor` melaporkan apakah dnsmasq terpasang.

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
- Formula Homebrew berversi didukung: bila `php@8.4`, `mysql@8.4`, dst. terpasang, seri tersebut dipakai otomatis dan muncul di dropdown Konfigurasi. Pasang dengan `brew install php@8.4` / `brew install mysql@8.4`.
- DMG unsigned berisi aplikasi, symlink Applications, README prasyarat, dan `dl-helper` di dalam bundle.
- CI Intel dan Apple Silicon (`macos-build` menghasilkan artifact arm64 dan intel).
- Tray/menu bar berjalan (systray `nativeStart` dijadwalkan ke main queue Cocoa via `dispatch_async_f`).
- Update aplikasi: deteksi `.dmg`, unduh + verifikasi SHA256, self-update `.app` (ganti bundle atomik + relaunch), fallback buka DMG.
- Update runtime lewat Homebrew: tab Update/`dl update` menampilkan paket usang dan menjalankan `brew upgrade <formula>` lalu restart stack.
- Editor (`dl edit`, tray/GUI): VS Code/Cursor/Zed/Sublime/TextMate, fallback TextEdit (file) / Finder (folder).

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

- Installer macOS belum tersedia (distribusi lewat `.dmg`/`.zip`).
- Aplikasi belum ditandatangani Developer ID.
- Belum dinotarisasi; Gatekeeper dapat memblokir distribusi publik.
- Rollback otomatis update aplikasi macOS belum tersedia.
- Runtime harus dipasang manual lewat Homebrew.
- Versi runtime mengikuti Homebrew. Seri spesifik didukung bila formula berversi terpasang (`php@8.4`, `mysql@8.4`, dst.); jika tidak, formula `php`/`mysql` terpasang yang dipakai.
- Data MySQL disimpan per seri yang diminta (mis. `8.4`). Berpindah antar biner seri berbeda pada seri yang sama (mis. `mysql` lalu `mysql@8.4`) dapat membuat `mysqld` menolak start karena datadir dibuat versi lain; jalankan `dl db reset --yes` lalu `dl up`.
- DevLok belum mengelola `brew services`.