# ADR 0004 - Tanpa Docker, binary portable

- Status: Accepted
- Tanggal: 2026-10-20

## Konteks

DDEV dan Lando menyediakan environment reproducible dengan container. Kelebihan utamanya isolasi dan konsistensi. Kekurangannya konsumsi resource, ketergantungan Docker Desktop/WSL2, dan startup lambat.

## Keputusan

DevLok v0.1.0 tidak memakai Docker. Runtime dijalankan sebagai binary native yang diunduh dan diverifikasi.

## Alasan

- Target pengguna adalah developer Windows yang ingin ringan dan offline.
- Startup < 10 detik dengan cache.
- Tidak membutuhkan lisensi Docker Desktop untuk penggunaan komersial.
- Tidak bergantung pada virtualisasi yang sering bermasalah di Windows.
- Kontrol proses lebih langsung (Job Object).

## Konsekuensi

- Isolasi lebih lemah daripada container; service berbagi host.
- Konsistensi environment bergantung pada lockfile, bukan image.
- Perbedaan versi library sistem dapat memengaruhi hasil.
- Keamanan default harus lebih ketat untuk mengompensasi.

## Alternatif yang ditolak

- **Docker Compose**: berat, startup lambat, dependency eksternal.
- **WSL2**: pengalaman file system lintas OS lambat, tidak cocok untuk semua pengguna.
- **Podman**: masih membutuhkan VM di Windows.

## Tindak lanjut

- Fokus pada lockfile dan verifikasi checksum.
- Sediakan per-project database dan user untuk isolasi data.
- Evaluasi integrasi Docker opsional di v0.3.0, bukan sebagai default.
