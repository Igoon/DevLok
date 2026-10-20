# ADR 0002 - Runtime per project dan lockfile

- Status: Accepted
- Tanggal: 2026-10-20

## Konteks

Laragon memakai satu versi PHP/Node/MySQL global. Berganti project berarti berganti versi untuk semua project. Ini menyulitkan project lama dan onboarding tim.

## Keputusan

Setiap project menyimpan `.devlok.yaml` untuk kebutuhan runtime dan `.devlok.lock` untuk versi eksak. DevLok menjalankan beberapa versi PHP secara berdampingan dan memilih versi berdasarkan project.

## Aturan

- Manifest memuat versi parsial (`8.4`), lock memuat versi eksak (`8.4.23`) + SHA256.
- `dl up` selalu mengikuti lock bila tersedia.
- `dl lock` adalah satu-satunya penulis lock.
- Perbedaan manifest dan lock menghasilkan peringatan, bukan error.
- Versi berbeda memakai pool FastCGI dan instance MySQL terpisah bila perlu.
- Lock menyimpan platform agar checksum sesuai.

## Konsekuensi

- Konsumsi disk naik karena banyak versi runtime.
- Perlu port allocator untuk instance MySQL ganda.
- Perlu UI untuk melihat versi yang terpasang dan yang dipakai.
- Perlu strategi garbage collection runtime yang tidak dipakai.

## Alternatif yang ditolak

- **Global version manager** (seperti Laragon): masalah yang ingin diselesaikan tetap ada.
- **asdf/mise saja**: tidak menangani service web, database, hosts, dan TLS.
- **Docker per project**: berat, butuh virtualisasi, dan bertentangan dengan target ringan.

## Tindak lanjut

- Tetapkan skema lock di M1.
- Uji dua versi PHP paralel di M2/M3.
- Evaluasi runtime GC di v0.1.1.
