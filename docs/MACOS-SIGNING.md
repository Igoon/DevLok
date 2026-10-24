# macOS Signing dan Notarization

Workflow `macos-build` mendukung signing dan notarization jika GitHub Secrets tersedia. Tanpa secret, build tetap berjalan sebagai artifact unsigned.

## 1. Buat Certificate

1. Buka Apple Developer > Certificates, Identifiers & Profiles.
2. Buat certificate **Developer ID Application**.
3. Buat CSR dari Keychain Access:
   - Certificate Assistant > Request a Certificate From a Certificate Authority.
   - Simpan ke disk.
4. Upload CSR, unduh `.cer`, lalu buka hingga masuk ke Keychain Access.
5. Di Keychain Access, buka **My Certificates**, pastikan certificate memiliki private key.
6. Export certificate sebagai `.p12` dan beri password.

Gunakan certificate **Developer ID Application**. Jangan memakai Apple Development atau Apple Distribution.

## 2. Buat App Store Connect API Key

1. Buka App Store Connect > Users and Access > Integrations.
2. Buat **Team Key** untuk App Store Connect API.
3. Beri akses yang dibutuhkan untuk notarization.
4. Unduh file `.p8` (hanya bisa diunduh sekali).
5. Catat:
   - Key ID
   - Issuer ID

## 3. Ubah ke Base64

macOS:

```bash
base64 -i certificate.p12 | pbcopy
base64 -i AuthKey_XXXX.p8 | pbcopy
```

Windows PowerShell:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("certificate.p12")) | Set-Clipboard
[Convert]::ToBase64String([IO.File]::ReadAllBytes("AuthKey_XXXX.p8")) | Set-Clipboard
```

## 4. Tambahkan GitHub Secrets

Repository > Settings > Secrets and variables > Actions > New repository secret:

| Secret | Isi |
|---|---|
| `APPLE_CERT_P12_BASE64` | certificate `.p12` dalam base64 |
| `APPLE_CERT_PASSWORD` | password export `.p12` |
| `APPLE_API_KEY_ID` | Key ID App Store Connect |
| `APPLE_API_ISSUER_ID` | Issuer ID App Store Connect |
| `APPLE_API_KEY_P8_BASE64` | file `.p8` dalam base64 |

## 5. Jalankan Build

Jalankan workflow `macos-build`. Jika secret lengkap, workflow akan:

1. Import certificate ke keychain sementara.
2. Menandatangani `dl-helper`.
3. Menandatangani `DevLok.app` dengan hardened runtime.
4. Membuat ZIP dan DMG.
5. Submit DMG ke `notarytool`.
6. Staple dan validasi tiket notarization.

## 6. Verifikasi

Di mesin macOS:

```bash
codesign --verify --deep --strict --verbose=2 DevLok.app
codesign -dv --verbose=4 DevLok.app
spctl --assess --type execute --verbose=2 DevLok.app
xcrun stapler validate DevLok-macos-intel.dmg
```

## Catatan

- Private key `.p12` dan `.p8` tidak boleh masuk repository.
- Simpan backup certificate dan API key.
- Tanpa secret, artifact tetap unsigned dan Gatekeeper dapat memblokir.
- Notarization membutuhkan koneksi internet dan waktu beberapa menit.

## Checklist Kesiapan (saat certificate tersedia)

Pipeline signing/notarization sudah ada di `.github/workflows/release.yml` (job `macos`). Yang perlu disiapkan saat Apple certificate tersedia:

1. Apple Developer Program aktif (akun berbayar).
2. Certificate **Developer ID Application** + private key, diexport `.p12` berpassword.
3. App Store Connect **Team Key** API (`.p8`, Key ID, Issuer ID).
4. Tambah GitHub Secrets:
   - `APPLE_CERT_P12_BASE64`, `APPLE_CERT_PASSWORD`
   - `APPLE_API_KEY_ID`, `APPLE_API_ISSUER_ID`, `APPLE_API_KEY_P8_BASE64`
5. Push tag rilis. Job `macos` akan import certificate, menandatangani `dl-helper` dan `DevLok.app` (hardened runtime), membuat ZIP/DMG, submit DMG ke `notarytool`, lalu staple + validate.
6. Verifikasi di Mac:

   ```bash
   codesign --verify --deep --strict --verbose=2 /Applications/DevLok.app
   spctl --assess --type execute --verbose=2 /Applications/DevLok.app
   xcrun stapler validate DevLok-macos-arm64.dmg
   ```

7. Self-update: DMG yang sudah dinotarisasi membuat `.app` hasil self-update lolos Gatekeeper tanpa perlu `xattr -dr com.apple.quarantine`.

Tanpa secret, rilis tetap dibuat (unsigned) dan self-update tetap berjalan; Gatekeeper hanya berpotensi memblokir bila file diunduh lewat browser.