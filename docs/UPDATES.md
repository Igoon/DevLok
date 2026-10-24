# Update dan Patch

## Prinsip

1. Update aplikasi, runtime, dan dependency project adalah tiga hal berbeda.
2. Runtime yang sedang dipakai project tidak berubah otomatis.
3. `.devlok.lock` tetap menjadi sumber versi eksak.
4. Versi lama runtime dipertahankan agar rollback memungkinkan.
5. Semua unduhan runtime diverifikasi SHA256.
6. Update tidak mengubah file project pengguna.

## Kondisi Saat Ini

`dl update` memeriksa runtime terpasang terhadap registry yang dibundel dan update aplikasi. Jika `config/update.json` tidak ada, DevLok memakai GitHub Releases resmi `Igoon/DevLok`.

```powershell
dl update
dl update --check
dl update --json
dl update download
dl update apply --yes
```

`dl update download` mengunduh installer aplikasi, memverifikasi SHA256, lalu menyimpannya di `%USERPROFILE%\.devlok\cache\updates`. Installer tidak dijalankan otomatis.

`dl update apply --yes` menghentikan stack lalu menjalankan installer `.exe` atau `.msi`. Installer tetap interaktif. Tutup GUI DevLok terlebih dahulu jika masih terbuka.

Contoh:

```text
DevLok 0.1.0-dev
Update runtime tersedia:
  php 8.4.26 -> 8.5.11

Pasang versi baru dengan: dl runtime add <nama>@<versi>
```

GUI/tray juga menampilkan menu **Updates (N)** jika ada update aplikasi atau runtime. Mengeklik menu tersebut menampilkan daftar update. Pemeriksaan manual tidak mengunduh atau memasang apa pun kecuali opsi auto-update aktif. Signed manifest di `config/update.json` selalu diprioritaskan; GitHub fallback dipakai hanya jika config tidak ada.

## Auto-update dari GUI

Tab **Update** punya preferensi otomatis yang disimpan di `%USERPROFILE%\.devlok\config\preferences.json`:

```json
{
  "schema": 1,
  "auto_update_app": false,
  "auto_update_runtime": false,
  "auto_update_runtime_series": false,
  "interval_minutes": 360
}
```

- `auto_update_runtime`: pasang patch untuk seri runtime yang sudah terpasang, perbarui project aktif, lalu restart stack sekali.
- `auto_update_runtime_series`: izinkan upgrade seri baru (mis. PHP 8.4 ke 8.5). Default mati.
- `auto_update_app`: unduh dan jalankan installer DevLok otomatis. Stack dihentikan dan GUI ditutup sebelum installer.
- `interval_minutes`: jeda pemeriksaan, 5-10080 menit.

Auto-update berjalan hanya selama GUI DevLok hidup. Supervisor latar tetap menjaga service tetapi tidak melakukan update. Update aplikasi otomatis tidak berjalan diam-diam: opsi harus diaktifkan eksplisit dan installer tetap memakai mekanisme update normal.

## Registry Runtime Remote

Manifest `updates.json` bertanda tangan juga memuat array `runtimes`. Client memverifikasi signature sebelum memakai registry tersebut.

- Cache: `%USERPROFILE%\.devlok\cache\registry.json`, TTL 6 jam.
- "Periksa update" manual memaksa fetch terbaru.
- Auto-update dan startup memakai cache; fetch gagal memakai cache lama atau registry embedded.
- Registry remote digabung per nama runtime; versi baru bisa terdeteksi tanpa merilis ulang aplikasi, selama manifest pusat diperbarui.
- URL tetap HTTPS dan checksum tetap diverifikasi.

Fallback GitHub memakai API resmi dan checksum asset `.sha256`. Ini tidak memakai signature Ed25519, sehingga trust berakar pada keamanan repo GitHub. Untuk verifikasi rantai penuh, gunakan signed manifest.

Memasang versi baru:

```powershell
dl runtime add php@8.5
```

Mengganti versi project dilakukan lewat tab Konfigurasi, menu tray Version, atau manifest. Lockfile harus diperbarui dengan `dl lock`.

## Update Runtime

- Registry berisi URL resmi, ukuran, dan SHA256.
- Versi baru dipasang berdampingan, bukan menimpa versi lama.
- Project tetap memakai versi di manifest/lock sampai diubah manual atau oleh aksi update GUI/auto-update patch.
- Setelah versi project diubah, stack divalidasi dan direstart.
- Config yang tidak valid membatalkan start; versi lama tetap tersedia.
- MySQL lintas mayor membutuhkan backup dan pengujian; jangan mengandalkan downgrade.

## Update Aplikasi DevLok

Alur update aplikasi:

1. Manifest rilis bertanda tangan (atau GitHub Releases fallback).
2. Pemeriksaan versi aplikasi.
3. Unduh installer, verifikasi signature dan SHA256.
4. Hentikan stack dan supervisor sebelum apply.
5. Installer mengganti binary setelah proses lama berhenti.
6. Installer menyimpan binary sebelumnya; rollback dijalankan dengan `dl update rollback --yes` atau tray Tools > Rollback app update.

Apply manual meminta konfirmasi. Apply otomatis hanya berjalan jika `auto_update_app` diaktifkan dan tetap menghentikan stack lebih dulu. Rollback memulihkan binary dari `backup\previous` yang dibuat installer.

## Update Dependency Project

Composer, npm, migration, dan `.env` adalah tanggung jawab project. DevLok hanya menyediakan runtime dan environment. Jangan menjalankan `composer update` atau `npm update` otomatis dari tray.

## Patch Keamanan

- Registry menandai update keamanan secara terpisah.
- Notifikasi tidak memblokir pekerjaan.
- Update keamanan tetap membutuhkan konfirmasi.
- Setelah patch, jalankan `dl runtime verify` dan `nginx -t`/health check.
- Jangan hapus versi lama sampai project yang memakainya sudah dimigrasikan.

## Remote Manifest Bertanda Tangan

Aktifkan pemeriksaan update aplikasi dengan membuat `%USERPROFILE%\.devlok\config\update.json`:

```json
{
  "schema": 1,
  "manifest_url": "https://contoh.com/devlok/updates.json",
  "public_key": "BASE64_ED25519_PUBLIC_KEY_32_BYTE"
}
```

Manifest memakai envelope:

```json
{
  "schema": 1,
  "payload": "BASE64_JSON_PAYLOAD",
  "signature": "BASE64_ED25519_SIGNATURE"
}
```

Payload yang ditandatangani:

```json
{
  "app": {
    "version": "0.1.1",
    "url": "https://contoh.com/devlok/DevLok-Setup-0.1.1.exe",
    "sha256": "64_HEX_CHARACTERS",
    "severity": "security",
    "notes": "Perbaikan keamanan",
    "published_at": "2026-10-20T00:00:00Z"
  },
  "apps": {
    "darwin/arm64": {
      "version": "0.1.1",
      "url": "https://contoh.com/devlok/DevLok-macos-arm64.dmg",
      "sha256": "64_HEX_CHARACTERS",
      "severity": "security"
    },
    "darwin/amd64": {
      "version": "0.1.1",
      "url": "https://contoh.com/devlok/DevLok-macos-intel.dmg",
      "sha256": "64_HEX_CHARACTERS",
      "severity": "security"
    }
  }
}
```

`app` adalah rilis default (Windows). Opsional `apps` memetakan `GOOS/GOARCH` ke rilis khusus platform; client memakai entri `apps[<os>/<arch>]` bila ada, selain itu `app`. Di macOS, `dl update apply` membuka DMG untuk pemasangan manual.

Ketentuan:

- Signature Ed25519 dibuat dari byte payload asli sebelum base64.
- Public key dan manifest URL hanya berasal dari config lokal, bukan dari isi manifest.
- Manifest wajib HTTPS; HTTP hanya untuk localhost saat pengujian.
- Ukuran manifest maksimum 1 MB dan timeout 10 detik.
- Client dapat mengunduh installer terverifikasi lewat `dl update download` atau tray **Tools > Download app update**. Tray **Tools > Apply app update** menghentikan stack, menutup DevLok, lalu menjalankan installer setelah proses lama keluar.
- Signature tidak valid = update ditolak.
