# Audit Kekurangan Laragon

Dokumen ini memetakan kekurangan Laragon ke status implementasi DevLok saat ini. Status jujur, bukan klaim pemasaran.

Status: `done`, `partial`, `not yet`.

## 1. Sepuluh kekurangan awal

| # | Kekurangan Laragon | Status | Bukti / sisa kekurangan |
|---|---|---|---|
| 1 | Windows-only | partial | Core Go lintas platform dan `platform` layer sudah ada, tetapi build macOS belum dirilis. |
| 2 | Runtime global, tidak bisa pin versi per project | done | `.devlok.yaml` + `.devlok.lock`; PHP/Node/MySQL per project; versi berbeda berdampingan. |
| 3 | Versi manual, tanpa checksum/update | partial | Registry + SHA256 + unduhan otomatis. Belum ada auto-update registry/aplikasi. |
| 4 | TLS lokal lemah, trust manual | done | CA lokal, cert per domain, TLS 1.2/1.3, trust user store, `dl tls`/Up. |
| 5 | Tidak ada supervision/health/recovery | partial | Job Object, health check saat start, restart policy saat owner hidup. Setelah CLI keluar tidak ada daemon yang terus memonitor. |
| 6 | Port bentrok baru terdeteksi dari log | done | Cek port sebelum start, owner PID/nama, fallback 8080/8443, cek live di Konfigurasi. |
| 7 | Log tersebar | partial | GUI punya log viewer per service dan log JSONL aplikasi. Log nginx/php/mysql masih file terpisah, belum satu stream gabungan. |
| 8 | PATH global diubah | done | Tidak menyentuh PATH global; `dl run`/`dl shell` memberi PATH project. |
| 9 | Konfigurasi situs menumpuk | done | Config generated di state dir, dibersihkan otomatis, manifest sebagai source of truth. |
| 10 | Environment tidak reproducible | done | Lockfile, checksum, `dl lock`, `dl export`, `dl import`. Keterbatasan: export belum menyertakan dump DB. |

## 2. Kekurangan lain yang relevan

| Area | Status | Catatan |
|---|---|---|
| macOS | not yet | Arsitektur siap, build belum tersedia. |
| Windows ARM64 | not yet | Target v0.1.2. |
| Multi-site paralel | partial | Satu stack aktif (satu project + phpMyAdmin). Ganti site = restart stack. |
| Supervisi jangka panjang | not yet | Tidak ada background owner/daemon yang auto-restart setelah `dl up` selesai. |
| Auto-update aplikasi/runtime | not yet | Registry diperbarui manual. |
| Import Laragon | done | CLI + tab Import GUI: scan, pilih, tulis manifest, adopt ke GUI. Belum copy project atau impor dump DB. |
| Project di luar workspace | done | `dl site adopt` dan opsi adopt di Import Laragon; GUI menampilkan project dari registry. |
| Isolasi antar project | partial | DB/user terpisah dan runtime terpisah; bukan container/VM. |
| Sinkronisasi tim | partial | Export/import file, belum ada sinkronisasi cloud/team preset. |
| Docker/WSL | not a goal | Keputusan desain: native binary, ringan, tanpa Docker. |
| Ekstensibilitas plugin | partial | Registry JSON dapat ditambah; belum ada API plugin. |
| Instalasi/upgrade | not yet | Installer Inno Setup dan auto-update belum dibuat. |

## 3. Gap paling penting

1. **Background supervisor** yang tetap hidup setelah CLI/GUI ditutup, agar restart policy dan health check benar-benar berjalan.
2. **Import Laragon end-to-end**: UI, copy/adopt project, opsi impor dump database.
3. **Dukungan multi-site paralel** dalam satu Nginx, tanpa restart saat pindah site.
4. **Auto-update** bertanda tangan untuk aplikasi dan registry runtime.
5. **Build macOS** + notarization, lalu ARM64 Windows.

## 4. Yang sudah lebih baik dari Laragon

- Runtime per project dan lockfile.
- Versi service dapat dipilih dari GUI, Konfigurasi, dan tray.
- Port, domain, root, nama DB, dan komponen dapat diatur dari GUI.
- CA lokal dan TLS modern.
- Deteksi port dan pemilik proses.
- DB/user per project dengan password acak dan DPAPI.
- Tanpa PATH global.
- GUI/tray modern dan CLI penuh.
