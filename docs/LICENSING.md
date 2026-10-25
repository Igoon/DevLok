# Lisensi dan Distribusi

## 1. Status lisensi kode DevLok

Belum ditentukan. Opsi yang dipertimbangkan:

| Opsi | Kelebihan | Kekurangan |
|---|---|---|
| MIT | Paling permisif, adopsi mudah | Tidak ada proteksi terhadap fork komersial |
| Apache-2.0 | Patent grant, cocok untuk tool | Sedikit lebih kompleks |
| AGPL-3.0 | Mencegah layanan tertutup | Kurang ramah adopsi |
| Proprietary | Kontrol penuh | Menghambat kontribusi dan kepercayaan |

Rekomendasi sementara: **Apache-2.0** untuk core dan CLI, dengan keputusan final sebelum rilis publik. Dokumen ini diperbarui setelah lisensi dipilih.

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
| Mailpit | MIT | Redistribusi diizinkan (v0.1.1) |
| WebView2 Runtime | Microsoft terms | Bootstrap, bukan redistribusi bebas |
| Visual C++ Redistributable | Microsoft terms | Bootstrap via installer resmi |

Catatan: tabel ini bukan nasihat hukum. Verifikasi lisensi versi spesifik sebelum rilis.

## 4. Kewajiban notice

Bila kelak ada komponen yang dibundel:

- Sertakan file `THIRD-PARTY-NOTICES.md`.
- Cantumkan teks lisensi lengkap untuk komponen yang mewajibkan.
- Tautkan sumber dan versi.
- Jangan menghapus notice dari binary upstream.

Untuk runtime yang diunduh, notice disediakan di folder runtime masing-masing dan ditampilkan di halaman About.

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

- Kontributor wajib menyetujui DCO atau CLA (belum ditentukan).
- Kontribusi kode harus kompatibel dengan lisensi proyek.
- Jangan menambahkan dependency dengan lisensi yang tidak kompatibel.
- Dependency baru harus melalui review lisensi.

## 9. Data pengguna

- Tidak ada telemetri keluar secara default.
- `support bundle` dibuat manual dan dapat ditinjau pengguna.
- Tidak ada data project yang dikirim ke server mana pun.
- Log lokal menjadi tanggung jawab pengguna.

## 10. To-do sebelum rilis publik

- [ ] Pilih dan terapkan lisensi kode.
- [ ] Tambahkan `LICENSE` dan `THIRD-PARTY-NOTICES.md`.
- [ ] Audit lisensi dependency Go dan npm.
- [ ] Verifikasi syarat redistribusi WebView2 dan VC++.
- [ ] Siapkan `SHA256SUMS` untuk setiap artefak.
- [ ] Tentukan kebijakan trademark dan nama produk.
- [ ] Siapkan akun signing bila memungkinkan.
