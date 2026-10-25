# Audit Kekurangan Laragon

Dokumen ini memetakan kekurangan Laragon ke status implementasi DevLok saat ini. Status jujur, bukan klaim pemasaran.

Status: `done`, `partial`, `not yet`.

## 1. Sepuluh kekurangan awal

| # | Kekurangan Laragon | Status | Bukti / sisa kekurangan |
|---|---|---|---|
| 1 | Windows-only | done | Core Go lintas platform; Windows dan macOS dirilis (GUI/tray/CLI). macOS memakai runtime Homebrew. |
| 2 | Runtime global, tidak bisa pin versi per project | done | `.devlok.yaml` + `.devlok.lock`; PHP/Node/MySQL per project (Windows); macOS memakai seri formula Homebrew (`php@8.4`, dst.). |
| 3 | Versi manual, tanpa checksum/update | done | Registry + SHA256 + unduhan otomatis, update runtime & aplikasi (manifest bertanda tangan, self-update macOS). |
| 4 | TLS lokal lemah, trust manual | done | CA lokal, cert per domain, TLS 1.2/1.3, trust user store, `dl tls`/Up. |
| 5 | Tidak ada supervision/health/recovery | done | Job Object/process group, health check, restart policy, dan supervisor latar `dl supervise`. |
| 6 | Port bentrok baru terdeteksi dari log | done | Cek port sebelum start, owner PID/nama, fallback 8080/8443, cek live di Konfigurasi. |
| 7 | Log tersebar | partial | GUI punya log viewer per service dan log JSONL aplikasi. Log nginx/php/mysql masih file terpisah, belum satu stream gabungan. |
| 8 | PATH global diubah | done | Tidak menyentuh PATH global; `dl run`/`dl shell` memberi PATH project. |
| 9 | Konfigurasi situs menumpuk | done | Config generated di state dir, dibersihkan otomatis, manifest sebagai source of truth. |
| 10 | Environment tidak reproducible | done | Lockfile, checksum, `dl lock`, `dl export`, `dl import`. Keterbatasan: export belum menyertakan dump DB. |

## 2. Kekurangan lain yang relevan

| Area | Status | Catatan |
|---|---|---|
| macOS | done | CLI, GUI, tray, CA/hosts, update aplikasi & runtime dirilis. Artifact belum notarized. |
| Windows ARM64 | not yet | Target roadmap. |
| Multi-site paralel | partial | Satu stack aktif (satu project + phpMyAdmin). Ganti site = restart stack. |
| Supervisi jangka panjang | done | Supervisor latar `dl supervise` memantau dan me-restart service dengan backoff. |
| Auto-update aplikasi/runtime | done | Manifest bertanda tangan per-platform, self-update macOS, `brew upgrade` untuk runtime macOS. |
| Import Laragon | done | CLI + tab Import GUI: scan, pilih, tulis manifest, adopt ke GUI. Belum copy project atau impor dump DB. |
| Project di luar workspace | done | `dl site adopt` dan opsi adopt di Import Laragon; GUI menampilkan project dari registry. |
| Isolasi antar project | partial | DB/user terpisah dan runtime terpisah; bukan container/VM. |
| Sinkronisasi tim | partial | Export/import file, belum ada sinkronisasi cloud/team preset. |
| Docker/WSL | not a goal | Keputusan desain: native binary, ringan, tanpa Docker. |
| Ekstensibilitas plugin | partial | Registry JSON dapat ditambah; belum ada API plugin. |
| Instalasi/upgrade | done | Installer Windows, portable ZIP, DMG/ZIP macOS, auto-update, dan rollback. Authenticode/notarization opsional menunggu certificate. |

## 3. Gap paling penting (sisa)

1. **Multi-site paralel** dalam satu Nginx, tanpa restart saat pindah site.
2. **Notarization macOS** dan **Authenticode Windows** (menunggu certificate).
3. **Windows ARM64**.
4. **Import Laragon end-to-end**: copy project dan impor dump database.
5. **Browser fallback** dan migrasi schema manifest.

## 4. Yang sudah lebih baik dari Laragon

- Runtime per project dan lockfile.
- Versi service dapat dipilih dari GUI, Konfigurasi, dan tray.
- Port, domain, root, nama DB, dan komponen dapat diatur dari GUI.
- CA lokal dan TLS modern.
- Deteksi port dan pemilik proses.
- DB/user per project dengan password acak dan DPAPI.
- Tanpa PATH global.
- GUI/tray modern dan CLI penuh.
