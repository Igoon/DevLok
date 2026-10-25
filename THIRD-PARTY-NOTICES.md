# Third-Party Notices

DevLok sendiri dilisensikan di bawah Apache-2.0 (lihat [LICENSE](LICENSE)). Proyek ini mencakup perangkat lunak pihak ketiga berikut. Daftar ini untuk atribusi; teks lisensi lengkap tersedia di repositori masing-masing proyek.

## Dependency Go (dibundel ke binary)

| Modul | Lisensi |
|---|---|
| github.com/energye/systray | Apache-2.0 |
| github.com/spf13/cobra | Apache-2.0 |
| github.com/spf13/pflag | BSD-3-Clause |
| github.com/wailsapp/wails/v2 | MIT |
| github.com/wailsapp/go-webview2 | MIT |
| golang.org/x/sys | BSD-3-Clause |
| golang.org/x/crypto | BSD-3-Clause |
| golang.org/x/net | BSD-3-Clause |
| golang.org/x/text | BSD-3-Clause |
| gopkg.in/yaml.v3 | MIT / Apache-2.0 |
| git.sr.ht/~jackmordaunt/go-toast/v2 | Unlicense / MIT |
| github.com/bep/debounce | MIT |
| github.com/gabriel-vasile/mimetype | MIT |
| github.com/go-ole/go-ole | MIT |
| github.com/godbus/dbus/v5 | BSD-2-Clause |
| github.com/google/uuid | BSD-3-Clause |
| github.com/gorilla/websocket | BSD-2-Clause |
| github.com/inconshreveable/mousetrap | Apache-2.0 |
| github.com/jchv/go-winloader | ISC |
| github.com/labstack/echo/v4 | MIT |
| github.com/labstack/gommon | MIT |
| github.com/leaanthony/go-ansi-parser | MIT |
| github.com/leaanthony/gosod | MIT |
| github.com/leaanthony/slicer | MIT |
| github.com/leaanthony/u | MIT |
| github.com/mattn/go-colorable | MIT |
| github.com/mattn/go-isatty | MIT |
| github.com/pkg/browser | BSD-2-Clause |
| github.com/pkg/errors | BSD-2-Clause |
| github.com/rivo/uniseg | MIT |
| github.com/samber/lo | MIT |
| github.com/tkrajina/go-reflector | Apache-2.0 |
| github.com/valyala/bytebufferpool | MIT |
| github.com/valyala/fasttemplate | MIT |

## Dependency frontend (dibundel ke aset UI)

| Paket | Lisensi |
|---|---|
| svelte | MIT |
| vite | MIT |
| typescript | Apache-2.0 |
| @sveltejs/vite-plugin-svelte | MIT |
| svelte-check | MIT |

## Runtime yang diunduh saat runtime (tidak dibundel)

Nginx, PHP, MySQL, Node.js, Composer, Adminer, phpMyAdmin, Notepad++, Mailpit. Runtime ini diunduh dari sumber resmi saat dibutuhkan dan **tidak** dibundel ke installer. Lisensi dan syarat redistribusinya dijelaskan di [docs/LICENSING.md](docs/LICENSING.md); notice disediakan oleh masing-masing paket upstream.

## Catatan

- Lisensi MIT/BSD/ISC/Unlicense mewajibkan penyertaan notice/lisensi pihak ketiga pada distribusi. Untuk rilis resmi, jalankan audit lisensi (mis. `go-licenses` untuk Go dan `license-checker` untuk npm) guna menghasilkan berkas teks lisensi gabungan bila diperlukan.
- Teks lisensi lengkap juga tersedia pada modul di `go.sum` dan `desktop/frontend/package-lock.json`, serta di repositori upstream masing-masing.
