# ADR 0005 - CA lokal dan certificate per domain

- Status: Accepted
- Tanggal: 2026-10-20

## Konteks

Laragon memakai satu certificate bersama untuk semua situs. Trust sering harus diatur manual dan browser dapat menampilkan peringatan. TLS lama juga masih diaktifkan pada konfigurasi contoh.

## Keputusan

DevLok membuat CA lokal sendiri dan menerbitkan leaf certificate per domain, dengan trust di user store tanpa admin.

## Detail

- CA: ECDSA P-256, dibuat sekali per state dir.
- Private key: dienkripsi DPAPI (CurrentUser).
- Leaf: SAN untuk domain utama, alias, dan wildcard bila diminta.
- Masa berlaku: 825 hari.
- Trust: `certutil -user -addstore Root`.
- TLS minimum 1.2, HTTP/2 aktif.
- Regenerate otomatis saat domain berubah.

## Alasan

- Certificate per domain lebih jelas dan mudah dicabut.
- Trust user store tidak membutuhkan admin.
- Menghindari password admin berulang.
- Memperbaiki masalah peringatan browser.
- Private key tidak perlu dibagikan antar project.

## Konsekuensi

- Pengguna harus menjalankan `dl tls trust` sekali, atau installer menawarkannya.
- CA harus dihapus saat uninstall (dengan konfirmasi).
- Beberapa aplikasi non-browser mungkin tidak memakai user trust store.
- Perlu penanganan jika CA key hilang atau state dir dipindah.

## Alternatif yang ditolak

- **Satu certificate bersama**: tidak bisa dicabut per domain dan kurang aman.
- **mkcert**: dependency eksternal; konsepnya sama, tetapi DevLok dapat mengimplementasikan sendiri.
- **CA machine-wide**: butuh admin dan meninggalkan jejak global.

## Tindak lanjut

- Uji Chrome, Edge, dan curl di M3.
- Sediakan `dl tls status` dan `untrust`.
- Dokumentasikan batasan aplikasi non-browser.
