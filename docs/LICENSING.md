# Lisensi dan Distribusi

## 1. Lisensi kode DevLok

DevLok dilisensikan di bawah **Apache License, Version 2.0**. Teks lengkap ada di [`LICENSE`](../LICENSE) dan atribusi di [`NOTICE`](../NOTICE). Copyright 2026 I Go On (Igoon).

Alasan: permisif (termasuk penggunaan komersial), ada patent grant dan klausa trademark, serta kompatibel dengan dependency proyek.

Konsekuensi: bukan copyleft (fork tertutup diperbolehkan selama mematuhi lisensi/notice), dan relicense ke proprietary di kemudian hari membutuhkan izin seluruh kontributor kecuali diterapkan CLA.

## 2. Prinsip distribusi runtime

DevLok **tidak membundel** runtime pihak ketiga di installer. Runtime diunduh dari sumber resmi saat pertama kali dibutuhkan, dengan verifikasi SHA256.

Alasan:

- Menghormati lisensi dan syarat redistribusi.
- Menghindari installer besar.
- Memungkinkan update runtime tanpa rilis aplikasi baru.
- Menghindari masalah lisensi MySQL GPL.

Konsekuensi:

- Instalasi pertama membutuhkan internet.
- Registry harus menyimpan URL resmi dan checksum.
- Perubahan URL upstream harus dipantau.

## 3. Lisensi runtime

| Komponen | Lisensi | Catatan distribusi |
|---|---|---|
| Nginx | BSD-2-Clause | Redistribusi diizinkan dengan notice |
| PHP | PHP License 3.01 | Redistribusi diizinkan dengan notice |
| MySQL | GPLv2 (server) | Jangan dibundel tanpa analisis; unduh saat runtime |
| Composer | MIT | Redistribusi diizinkan |
| Node.js | MIT | Redistribusi diizinkan dengan notice |
| Adminer | Apache-2.0 / GPL-2.0 | Unduh saat runtime |
| Git | GPLv2 | Tidak dibundel; memakai instalasi sistem |
| Mailpit | MIT | Diunduh saat runtime (tidak dibundel) |
| WebView2 Runtime | Microsoft terms | Bootstrap, bukan redistribusi bebas |
| Visual C++ Redistributable | Microsoft terms | Bootstrap via installer resmi |

Catatan: tabel ini bukan nasihat hukum. Verifikasi lisensi versi spesifik sebelum rilis.

## 4. Kewajiban notice

Dependency Go dan npm yang dibundel ke binary/aset dicatat di [`THIRD-PARTY-NOTICES.md`](../THIRD-PARTY-NOTICES.md).

- Sertakan file `THIRD-PARTY-NOTICES.md` pada distribusi.
- Cantumkan teks lisensi lengkap untuk komponen yang mewajibkan (gunakan audit lisensi saat rilis).
- Tautkan sumber dan versi.
- Jangan menghapus notice dari binary upstream.

Untuk runtime yang diunduh, notice disediakan di folder runtime masing-masing dan ditampilkan di halaman About.

Audit otomatis:

```bash
./scripts/license-audit.sh          # macOS/Linux
```
```powershell
.\scripts\license-audit.ps1         # Windows
```

Skrip memakai `go-licenses` (dependency Go pada platform aktif) dan `license-checker` (npm), menulis hasil ke `dist/licenses/`, dan memperingatkan lisensi GPL/AGPL/SSPL.

## 5. Merek dagang

- "DevLok" belum didaftarkan.
- Jangan memakai nama atau logo Laragon, XAMPP, Herd, atau merek lain.
- Jangan mengklaim kompatibilitas resmi dengan pihak ketiga.
- Fitur "import Laragon" adalah interoperabilitas, bukan afiliasi.

## 6. Code signing

| Platform | Kebutuhan | Status |
|---|---|---|
| Windows | Authenticode, idealnya EV | Belum (opsional) |
| macOS | Developer ID + notarization | Belum; wajib untuk distribusi publik |

Tanpa signing:

- Windows SmartScreen dapat menampilkan peringatan.
- Antivirus dapat memberi false positive.
- Checksum rilis menjadi verifikasi utama.

## 7. Update dan checksum

- Setiap rilis menyertakan `SHA256SUMS`.
- Update otomatis (v0.1.1) memverifikasi signature manifest dan checksum artefak.
- Update tidak boleh menurunkan versi tanpa persetujuan pengguna.
- Rollback ke versi sebelumnya disediakan minimal satu versi.

## 8. Kontribusi

- Kontribusi memakai **DCO** (`git commit -s`; lihat [DCO](../DCO) dan [CONTRIBUTING.md](../CONTRIBUTING.md)), dengan kontribusi dilisensikan di bawah Apache-2.0. CLA belum diterapkan; dapat ditambahkan bila dibutuhkan untuk dual-license.
- Kontribusi kode harus kompatibel dengan lisensi proyek.
- Jangan menambahkan dependency dengan lisensi yang tidak kompatibel.
- Dependency baru harus melalui review lisensi.

## 9. Data pengguna

- Tidak ada telemetri keluar secara default.
- `support bundle` dibuat manual dan dapat ditinjau pengguna.
- Tidak ada data project yang dikirim ke server mana pun.
- Log lokal menjadi tanggung jawab pengguna.

## 10. To-do sebelum rilis publik

- [x] Pilih dan terapkan lisensi kode (Apache-2.0).
- [x] Tambahkan `LICENSE` dan `NOTICE`.
- [x] Tambahkan `THIRD-PARTY-NOTICES.md` untuk dependency yang dibundel.
- [ ] Audit lisensi dependency Go dan npm.
- [ ] Verifikasi syarat redistribusi WebView2 dan VC++.
- [ ] Siapkan `SHA256SUMS` untuk setiap artefak.
- [ ] Tentukan kebijakan trademark dan nama produk.
- [ ] Siapkan akun signing bila memungkinkan.
