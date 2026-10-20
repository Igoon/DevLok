# ADR 0001 - Go + Wails untuk core dan GUI

- Status: Accepted
- Tanggal: 2026-10-20

## Konteks

DevLok perlu GUI tray, manajemen proses, akses Windows API, dan rencana port ke macOS. Kandidat: Go + Wails, Tauri, Electron, .NET WinUI, Fyne.

## Keputusan

Gunakan **Go** untuk seluruh logika dan **Wails v2** untuk GUI, dengan frontend **Svelte + TypeScript**.

## Alasan

- Go cross-compile ke Windows dan macOS tanpa CGO.
- Wails memakai WebView2 di Windows dan WKWebView di macOS; kode aplikasi sama.
- Binary jauh lebih kecil dari Electron (~20 MB vs ~150 MB).
- Manajemen proses, Job Object, dan Windows API dapat diakses langsung.
- Satu bahasa untuk CLI, core, dan bridge GUI.
- Frontend web memungkinkan fallback mode browser tanpa kerja tambahan.

## Konsekuensi

- Wails tidak cross-compile antar OS; build macOS harus di macOS atau CI.
- WebView2 Runtime perlu dipastikan ada di Windows 10.
- Tim perlu menguasai Go dan TypeScript.
- Ekosistem komponen UI lebih kecil dari Electron.

## Alternatif yang ditolak

- **Electron**: ukuran dan memori terlalu besar untuk tool yang harus ringan.
- **Tauri**: secara teknis cocok, tetapi menambah Rust toolchain yang belum dipakai tim.
- **.NET WinUI**: Windows-only.
- **Fyne**: cross-compile membutuhkan CGO dan UI kurang fleksibel.

## Tindak lanjut

- Aktifkan CI build Windows sejak M0.
- Tambahkan job kompilasi macOS (tanpa fitur) sejak M0.
- Dokumentasikan batasan WebView2 di installer.
