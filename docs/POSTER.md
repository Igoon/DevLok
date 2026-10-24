# Poster & Ringkasan v0.2.x

## Ringkasan fitur/perbaikan/dukungan (v0.2.0 → v0.2.6)

### Dukungan platform
- Windows x64 dan macOS 14/15 (Intel & Apple Silicon).
- Setiap GitHub Release memuat installer/portable Windows serta `DevLok-macos-<arch>.dmg`/`.zip` (arm64 + intel) dengan `.sha256`.
- Signed update manifest per-platform (`updates.json` dengan map `apps`).

### macOS — inti (v0.2.0)
- GUI + tray native (fix crash systray di main thread), `dl ui` menemukan `.app`, ikon "DL".
- Runtime Homebrew: Nginx, PHP-FPM, MySQL; state dir macOS; port owner via `lsof`; `dl doctor` memeriksa Homebrew.
- CA trust via `security`, hosts via `osascript` admin (fix bug kutip ganda), nginx `daemon off` agar terkelola supervisor.
- Tab Konfigurasi berfungsi (versi Homebrew), dialog konfirmasi native, `dl site list` menampilkan PATH.

### macOS — runtime & update (v0.2.1–v0.2.4)
- Formula Homebrew berversi (`php@8.4`, `mysql@8.4`) dipilih otomatis; phpMyAdmin per seri PHP.
- `dl db reset --yes` memulihkan datadir MySQL setelah ganti seri.
- Update aplikasi: deteksi `.dmg`, unduh + verifikasi SHA256, self-update `.app` atomik + relaunch (fallback buka DMG).
- Update runtime: `brew outdated` + `brew upgrade` di tab Update & `dl update`, lalu restart stack.

### macOS — fitur tambahan (v0.2.5–v0.2.6)
- `dl wildcard on|off|status` (dnsmasq) untuk wildcard `*.test` via `/etc/resolver`.
- Editor lintas platform: VS Code/Cursor/Zed/Sublime/TextMate → TextEdit/Finder.
- Footer "AI-powered by DeepSeek-v4-Flash".

### Windows — perbaikan dalam v0.2.x
- Editor tetap Notepad++ (→ VS Code → Notepad).
- Update tool: phpMyAdmin, Notepad++, Adminer bisa di-update; phpMyAdmin/Adminer restart stack.
- Checklist smoke manual GUI/tray di `docs/TESTING.md`.

### Rilis & kesiapan
- Pipeline signing/notarization + checklist (`docs/MACOS-SIGNING.md`), tinggal Apple certificate.

## Poster 9:16

Poster jadi (dibuat script): `docs/poster/devlok-poster-9x16.png` — 1080×1920 (rasio 9:16).

Regenerate:

```bash
python scripts/make-poster.py
```

Spesifikasi desain:

- Palet: dark navy `#0B1220`, panel `#111827`, cyan `#22D3EE`, hijau `#22C55E`, teks `#F8FAFC`, muted `#94A3B8`.
- Hero: ikon rounded-square "DL" (border cyan, dot hijau), glow cyan halus.
- Elemen: terminal `$ dl up`, 4 feature chip, footer "AI-powered by DeepSeek-v4-Flash".
- Tipografi: Arial/Arial Bold (fallback DejaVu Sans).

## Prompt text-to-image (9:16)

### English (Midjourney / DALL·E / Ideogram)

> Vertical 9:16 poster (1080x1920) for "DevLok", a modern local development environment app for Windows and macOS. Premium developer-tool aesthetic: dark navy background (#0B1220), cyan accent (#22D3EE), green status dot (#22C55E), subtle glassmorphism and soft cyan glow, high contrast, crisp bold sans-serif typography. Hero: rounded-square app icon with bold white "DL", thin cyan border, green dot at top-right, floating above a faint code-grid pattern. Accompany with a minimal translucent GUI window showing "Nginx • PHP • MySQL — running", a macOS-style menu-bar tray icon, and a terminal line "$ dl up". Headline: "DevLok". Subheadline: "Local development, simplified". Platform line: "Windows • macOS (Intel & Apple Silicon)". Four feature chips: "One-click Nginx + PHP + MySQL stack", "Runtime pinned per project", "Automatic local HTTPS (trusted CA)", "App & runtime updates in-app / CLI". Footer: "AI-powered by DeepSeek-v4-Flash". Clean vertical hierarchy, generous negative space, vector-clean, ultra-sharp 4k. Negative: blurry/misspelled text, extra logos, watermark, clutter, stock-photo people.

### Indonesia (ringkas)

> Poster vertikal 9:16 untuk "DevLok", app local development environment (Windows & macOS). Dark navy `#0B1220`, aksen cyan `#22D3EE`, dot hijau `#22C55E`, glassmorphism, glow cyan halus. Hero ikon "DL" rounded-square; jendela GUI minimal "Nginx/PHP/MySQL running", ikon tray, terminal `$ dl up`. Headline "DevLok", subheadline "Local development, simplified", "Windows • macOS (Intel & Apple Silicon)", 4 chip fitur, footer "AI-powered by DeepSeek-v4-Flash". Bersih, hierarki jelas, tipografi sans-serif tebal.

Tip: untuk tipografi paling rapi, hasilkan background/visual di AI lalu tempel teks di Figma/Canva/Ideogram. Prompt ini juga cocok untuk 1:1 (thumbnail) dan A3 (cetak) dengan penyesuaian rasio.
