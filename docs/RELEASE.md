# Release dan Packaging

## Build Portable

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-release.ps1
```

Output:

```text
dist\DevLok\<versi>\
dist\DevLok-<versi>-win64.zip
dist\DevLok-<versi>-win64.zip.sha256
```

Isi paket:

- `devlok-gui.exe`
- `dl.exe`
- `dl-helper.exe`
- `README.md`
- `docs\APLIKASI.md`
- `docs\UPDATES.md`
- `VERSION`

Script juga membangun ulang CLI, helper, dan GUI kecuali memakai `-SkipBuild`.

## Build Installer

Prasyarat: Inno Setup 6.

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\scripts\build-release.ps1 -Installer
```

Output:

```text
dist\DevLok-Setup-<versi>.exe
```

Installer:

- Per-user, tanpa admin.
- Direktori default `%LOCALAPPDATA%\Programs\DevLok`.
- Menutup aplikasi yang sedang berjalan sebelum upgrade.
- Menyalin binary lama ke `{app}\backup\previous` sebelum upgrade.
- Tidak menghapus `%USERPROFILE%\.devlok` atau project pengguna saat uninstall.
- Shortcut Start Menu dan opsi Desktop.

## GitHub Release

Workflow `.github/workflows/release.yml` berjalan saat tag `v*` di-push atau manual dari Actions.

```powershell
git tag v0.1.0-dev
git push origin v0.1.0-dev
```

Workflow membangun portable ZIP dan installer, mengunggah artifact, lalu membuat GitHub Release. Tag yang mengandung `-` dibuat sebagai pre-release.

Aset rilis:

- `DevLok-<versi>-win64.zip`
- `DevLok-<versi>-win64.zip.sha256`
- `DevLok-Setup-<versi>.exe`
- `DevLok-Setup-<versi>.exe.sha256`

Installer saat ini belum ditandatangani Authenticode. Windows SmartScreen dapat menampilkan peringatan.

## Manifest Update

Installer yang dihasilkan dapat dipakai sebagai `app.url` di manifest update. Hitung SHA256 installer:

```powershell
Get-FileHash .\dist\DevLok-Setup-0.1.1.exe -Algorithm SHA256
```

Masukkan versi, URL HTTPS, dan hash ke payload bertanda tangan Ed25519. Format ada di [UPDATES.md](UPDATES.md).

## Rollback

Installer menyimpan binary versi sebelumnya di:

```text
%LOCALAPPDATA%\Programs\DevLok\backup\previous
```

Rollback:

```powershell
dl update rollback --yes
```

Atau tray **Tools > Rollback app update**. Proses akan menutup GUI, menyalin binary dari folder backup, lalu menjalankan GUI kembali. Stack dihentikan terlebih dahulu. Rollback hanya tersedia jika upgrade dilakukan melalui installer DevLok.

## Signing

Authenticode bersifat opsional dan diaktifkan lewat GitHub Secrets:

- `WINDOWS_SIGN_PFX_BASE64`: isi file PFX dalam base64.
- `WINDOWS_SIGN_PFX_PASSWORD`: password PFX.

Jika secret tersedia, workflow menandatangani `dl.exe`, `dl-helper.exe`, `devlok-gui.exe`, dan installer sebelum smoke test dan publikasi. Checksum installer dihitung ulang setelah signing.

Timestamp server: `http://timestamp.digicert.com`.

Tanpa certificate, installer tetap dibuat dan diuji tetapi belum ditandatangani. Manifest update tetap mendukung signature Ed25519.

## Signed Update Manifest

Buat key sekali:

```powershell
go run ./cmd/devlok-release keygen -out update-signing.key
```

Perintah mencetak public key base64. Simpan private key di tempat aman, lalu tambahkan isi file ke GitHub Secret:

```text
UPDATE_SIGNING_KEY
```

Release workflow akan membuat `updates.json` bertanda tangan jika secret tersedia.

Client yang ingin memakai manifest ini membuat `%USERPROFILE%\.devlok\config\update.json`:

```json
{
  "schema": 1,
  "manifest_url": "https://github.com/Igoon/DevLok/releases/download/v0.1.0-dev/updates.json",
  "public_key": "PUBLIC_KEY_BASE64"
}
```

Tanpa config tersebut, DevLok memakai GitHub Releases API + checksum `.sha256`.
