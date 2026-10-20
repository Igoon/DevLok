# ADR 0003 - Hosts per site, bukan DNS server lokal

- Status: Accepted
- Tanggal: 2026-10-20

## Konteks

Domain lokal harus mengarah ke `127.0.0.1`. Windows `hosts` tidak mendukung wildcard. Alternatifnya menjalankan DNS server lokal (Acrylic, dnsmasq, atau server sendiri) yang mendengarkan port 53.

## Keputusan

v0.1.0 menulis satu entri hosts per site di dalam blok terkelola. DNS server lokal tidak dijalankan.

## Alasan

- Port 53 sering dipakai atau dibatasi oleh Windows dan antivirus.
- Menjalankan DNS server lokal menambah komponen privileged dan titik gagal.
- Entri per site cukup untuk jumlah project tipikal (< 100).
- Lebih mudah diaudit dan di-rollback.
- Tidak perlu mengubah konfigurasi DNS sistem.

## Konsekuensi

- Domain wildcard (`*.site.test`) tidak otomatis.
- Setiap site baru membutuhkan elevation untuk menulis hosts.
- Entri lama harus dibersihkan saat site dihapus.
- Perlu deteksi orphan entry.

## Alternatif yang ditolak

- **DNS server lokal**: kompleks, butuh admin permanen, konflik port 53.
- **Acrylic DNS Proxy**: dependency eksternal, service tambahan, kurang terkontrol.
- **`.localhost`**: resolusi tidak konsisten di semua klien dan tool.

## Tindak lanjut

- Blok hosts dengan marker dan backup di M3.
- `dl hosts clean` menghapus orphan.
- v0.2.0 evaluasi opsi wildcard untuk macOS (dnsmasq) tanpa mengubah default Windows.
