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

GUI/tray juga menampilkan menu **Updates (N)** jika ada update aplikasi atau runtime. Mengeklik menu tersebut menampilkan daftar update. Pemeriksaan tidak mengunduh atau memasang apa pun. Signed manifest di `config/update.json` selalu diprioritaskan; GitHub fallback dipakai hanya jika config tidak ada.

Fallback GitHub memakai API resmi dan checksum asset `.sha256`. Ini tidak memakai signature Ed25519, sehingga trust berakar pada keamanan repo GitHub. Untuk verifikasi rantai penuh, gunakan signed manifest.

Memasang versi baru:

```powershell
dl runtime add php@8.5
```

Mengganti versi project dilakukan lewat tab Konfigurasi, menu tray Version, atau manifest. Lockfile harus diperbarui dengan `dl lock`.

## Update Runtime

- Registry berisi URL resmi, ukuran, dan SHA256.
- Versi baru dipasang berdampingan, bukan menimpa versi lama.
- Project tetap memakai versi di manifest/lock sampai diubah.
- Setelah versi project diubah, stack divalidasi dan direstart.
- Config yang tidak valid membatalkan start; versi lama tetap tersedia.
- MySQL lintas mayor membutuhkan backup dan pengujian; jangan mengandalkan downgrade.

## Update Aplikasi DevLok

Belum ada auto-update. Rencana v0.1.1:

1. Manifest rilis bertanda tangan.
2. Pemeriksaan versi aplikasi.
3. Unduh installer/portable, verifikasi signature dan SHA256.
4. Minta pengguna menghentikan stack sebelum apply.
5. Installer atau helper mengganti binary setelah proses lama berhenti.
6. Installer menyimpan binary sebelumnya; rollback dijalankan dengan `dl update rollback --yes` atau tray Tools > Rollback app update.

Apply tidak berjalan diam-diam: pengguna harus mengonfirmasi dan installer tetap interaktif. Rollback memulihkan binary dari `backup\previous` yang dibuat installer.

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
  }
}
```

Ketentuan:

- Signature Ed25519 dibuat dari byte payload asli sebelum base64.
- Public key dan manifest URL hanya berasal dari config lokal, bukan dari isi manifest.
- Manifest wajib HTTPS; HTTP hanya untuk localhost saat pengujian.
- Ukuran manifest maksimum 1 MB dan timeout 10 detik.
- Client dapat mengunduh installer terverifikasi lewat `dl update download` atau tray **Tools > Download app update**. Tray **Tools > Apply app update** menghentikan stack, menutup DevLok, lalu menjalankan installer setelah proses lama keluar.
- Signature tidak valid = update ditolak.
