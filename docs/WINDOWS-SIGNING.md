# Windows Signing (Authenticode)

Workflow `release` (job `windows`) mendukung signing Authenticode jika GitHub Secrets tersedia. Tanpa secret, langkah signing dilewati dan rilis tetap dibuat sebagai artifact unsigned.

## 1. Siapkan Certificate

Untuk distribusi publik, gunakan certificate **code signing** dari CA tepercaya:

- **OV (Organization Validation)** — file `.pfx` biasa.
- **EV (Extended Validation)** — token/HSM (mis. Azure Trusted Signing, DigiCert KeyLocker). Untuk CI, EV biasanya butuh integrasi cloud signing, bukan `.pfx`.

Untuk uji internal, Anda bisa membuat self-signed certificate (hanya tepercaya di mesin yang memasang root-nya).

Membuat self-signed untuk uji (PowerShell, admin):

```powershell
$cert = New-SelfSignedCertificate `
  -Type CodeSigningCert `
  -Subject "CN=I Go On (Igoon)" `
  -CertStoreLocation Cert:\CurrentUser\My `
  -NotAfter (Get-Date).AddYears(2)

$pw = ConvertTo-SecureString -String "ganti-password" -Force -AsPlainText
Export-PfxCertificate -Cert $cert -FilePath DevLok-sign.pfx -Password $pw
```

## 2. Ubah `.pfx` ke Base64

Windows PowerShell:

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes("DevLok-sign.pfx")) | Set-Clipboard
```

macOS:

```bash
base64 -i DevLok-sign.pfx | pbcopy
```

## 3. Tambahkan GitHub Secrets

Repository > Settings > Secrets and variables > Actions > New repository secret:

| Secret | Isi |
|---|---|
| `WINDOWS_SIGN_PFX_BASE64` | berkas `.pfx` (sertifikat + private key) dalam base64 |
| `WINDOWS_SIGN_PFX_PASSWORD` | password `.pfx` |

## 4. Perilaku Pipeline

Jika `WINDOWS_SIGN_PFX_BASE64` tersedia, job `windows` akan menandatangani (signtool, SHA-256, timestamp DigiCert):

1. `build\bin\dl.exe`
2. `build\bin\dl-helper.exe`
3. `desktop\build\bin\devlok-gui.exe`
4. `dist\DevLok-Setup-<versi>.exe` (setelah installer dibuat)

Lalu menulis `<installer>.sha256`. Tanpa secret, langkah ini dilewati (rilis tetap dibuat).

## 5. Verifikasi

Windows:

```powershell
Get-AuthenticodeSignature .\DevLok-Setup-0.2.13.exe | Format-List Status, SignerCertificate
signtool verify /pa /v .\DevLok-Setup-0.2.13.exe
```

`Status` harus `Valid`. Untuk self-signed, pasang root certificate-nya lebih dulu agar `Valid`.

## Catatan

- Private key `.pfx` tidak boleh masuk repository; simpan backup dan password.
- Gunakan timestamp server (sudah dipakai: `http://timestamp.digicert.com`) agar signature tetap valid setelah sertifikat kedaluwarsa.
- Certificate dari CA (OV/EV) mengurangi peringatan SmartScreen; self-signed tidak.
- Signing `dl.exe`/`devlok-gui.exe` penting agar antarmuka bisa lolos kebijakan antivirus/SmartScreen.

## Checklist Kesiapan (saat certificate tersedia)

Pipeline signing sudah ada di `.github/workflows/release.yml` (job `windows`). Yang perlu disiapkan:

1. Certificate code signing (OV `.pfx`, atau EV via cloud signing).
2. Export `.pfx` berpassword; ubah ke base64.
3. Tambah secret `WINDOWS_SIGN_PFX_BASE64`, `WINDOWS_SIGN_PFX_PASSWORD`.
4. Push tag rilis; verifikasi signature installer dan binary.
