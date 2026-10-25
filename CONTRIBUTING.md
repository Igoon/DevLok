# Kontribusi

Terima kasih atas minat Anda. Proyek ini memakai lisensi **Apache-2.0** dan menerima kontribusi dengan **DCO** (Developer Certificate of Origin).

## DCO (wajib)

Setiap commit harus disertai sign-off:

```bash
git commit -s -m "pesan commit"
```

Sign-off menambahkan baris `Signed-off-by: Nama <email>` dan berarti Anda menyetujui teks [DCO](DCO). Dengan mengirim kontribusi, Anda melisensikannya di bawah Apache-2.0.

## Sebelum mengirim PR

- `go test ./...`, `go vet ./...`, dan `gofmt -l .` bersih (tidak ada file terdaftar).
- Frontend: `cd desktop/frontend && npm ci && npm run build`.
- Sertakan test untuk perubahan logika.
- Jangan meng-commit secret, kunci, atau kredensial.

## Dependency

- Dependency baru harus kompatibel dengan Apache-2.0 (umumnya MIT/BSD/Apache-2.0).
- Hindari menambahkan kode GPL/AGPL yang akan dibundel ke binary.
- Untuk dependency yang dibundel, notice/lisensi pihak ketiga perlu dicatat.

## Gaya kode

- Ikuti gaya yang ada; format dengan `gofmt` untuk Go dan konvensi Svelte/TypeScript yang berlaku.
- Perubahan perilaku didokumentasikan di `CHANGELOG.md`.
