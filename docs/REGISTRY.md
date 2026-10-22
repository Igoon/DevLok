# Registry Runtime

## 1. Lokasi

Registry disimpan di `internal/runtime/registry/*.json` dan di-embed ke binary dengan `go:embed`. Setiap file mewakili satu runtime.

File saat ini:

| File | Runtime |
|---|---|
| `nginx.json` | Nginx |
| `php.json` | PHP 8.1-8.5 |
| `mysql.json` | MySQL 8.4 LTS, 9.4.0 |
| `node.json` | Node.js 22 LTS |
| `composer.json` | Composer |
| `adminer.json` | Adminer 6.1.2 |
| `phpmyadmin.json` | phpMyAdmin 5.2.3 |
| `notepadpp.json` | Notepad++ 8.9.8.1 (portable) |

## 2. Skema

```json
{
  "schema": 1,
  "name": "nginx",
  "display": "Nginx",
  "versions": [
    {
      "version": "1.30.6",
      "artifacts": {
        "windows/amd64": {
          "url": "https://nginx.org/download/nginx-1.30.6.zip",
          "sha256": "c6ee35edb0a861063323f709192ae50d1eefaf2ea7d00678bd2b22d8542dc61e",
          "size": 2773293,
          "archive": "zip",
          "executables": ["nginx.exe"]
        }
      }
    }
  ]
}
```

## 3. Field

### Root

| Field | Wajib | Keterangan |
|---|---|---|
| `schema` | Ya | Harus `1` |
| `name` | Ya | Huruf kecil, angka, tanda hubung; unik |
| `display` | Ya | Nama tampilan |
| `versions` | Ya | Minimal satu versi |

### Version

| Field | Wajib | Keterangan |
|---|---|---|
| `version` | Ya | Angka dan titik, contoh `8.4.26` |
| `artifacts` | Ya | Map `os/arch` ke artifact |
| `strip_root` | Tidak | `false` mematikan penghapusan folder root tunggal |

### Artifact

| Field | Wajib | Keterangan |
|---|---|---|
| `url` | Ya | HTTPS |
| `url_fallbacks` | Tidak | Daftar URL HTTPS cadangan; dicoba berurutan jika `url` gagal |
| `sha256` | Ya | 64 karakter hex huruf kecil |
| `size` | Tidak | Ukuran byte, untuk progress dan verifikasi awal |
| `archive` | Ya | `zip`, `tar.gz`, atau `file` |
| `filename` | Untuk `file` | Nama file tujuan; jika kosong diambil dari URL |
| `executables` | Ya | Path relatif yang wajib ada setelah ekstraksi |

## 4. Aturan verifikasi

1. URL wajib HTTPS.
2. SHA256 wajib ada dan cocok setelah unduh.
3. Ukuran total hasil ekstraksi dibatasi 4 GiB dan 200.000 entri.
4. Entri absolut, `..`, symlink, hardlink, dan nama reserved Windows ditolak.
5. Folder root tunggal dihapus otomatis agar layout konsisten.
6. Semua `executables` harus ada sebelum runtime dianggap terpasang.
7. Marker `.devlok-runtime.json` ditulis setelah verifikasi; tanpa marker, runtime dianggap belum terpasang.
8. Jika `url` gagal, `url_fallbacks` dicoba berurutan. Checksum tetap satu dan harus cocok dengan isi salah satu URL.

## 5. Menambah versi

1. Dapatkan URL resmi dan SHA256 dari sumber upstream.
2. Tambahkan objek versi di file runtime terkait.
3. Jalankan `go test ./internal/runtime`.
4. Jalankan `dl runtime add <name>@<version>` di mesin bersih.
5. Jalankan `dl runtime verify <name>@<version>`.
6. Pastikan `runtime list` menampilkan status yang benar.

Jangan menambahkan checksum yang belum diverifikasi. Jangan memakai URL `latest` untuk artifact yang checksum-nya dipin; gunakan URL berversi.

## 6. Platform

Key artifact memakai format `os/arch`:

- `windows/amd64` (v0.1.0)
- `windows/arm64` (v0.1.2)
- `darwin/arm64` dan `darwin/amd64` (v0.2.0)

Versi yang tidak punya artifact untuk platform aktif tidak akan dipilih oleh resolver.

## 7. Registry remote

Registry embedded tetap menjadi fallback. Manifest update `updates.json` yang ditandatangani dapat memuat array `runtimes` berisi registry lengkap.

- Client memverifikasi signature Ed25519 sebelum memakai registry remote.
- Cache disimpan di `%USERPROFILE%\.devlok\cache\registry.json` dengan TTL 6 jam.
- Manual "Periksa update" memaksa fetch baru; auto-update/startup memakai cache.
- Jika fetch gagal, cache lama yang masih valid dipakai; jika tidak ada, registry embedded dipakai.
- Registry remote digabung per nama runtime; entri remote menimpa embedded.
- Semua URL tetap HTTPS dan SHA256 divalidasi dengan aturan registry yang sama.

## 8. Batasan

- Registry belum mendukung signature PGP; integritas mengandalkan SHA256 dari registry yang di-review.
- Tidak ada mirror otomatis. Jika upstream menghapus versi lama, tambahkan URL cadangan tepercaya di `url_fallbacks` secara eksplisit.
- Runtime yang diunduh tidak dibundel di installer.
- Verifikasi ulang isi direktori terpasang tidak dilakukan; `runtime verify` memeriksa marker, executable, dan checksum arsip cache.
