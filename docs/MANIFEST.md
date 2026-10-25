# Manifest Project

## 1. File

| File | Fungsi | Masuk repo |
|---|---|---|
| `.devlok.yaml` | Konfigurasi project yang ditulis manusia | Ya |
| `.devlok.lock` | Versi eksak + checksum, dihasilkan mesin | Ya |
| `state.json` | Status runtime, di state dir pengguna | Tidak |
| `config.json` | Preferensi global pengguna | Tidak |

## 2. Contoh `.devlok.yaml`

```yaml
schema: 1
name: siso

web:
  server: nginx
  root: public
  domain: app.local
  tld: test
  aliases:
    - www.app.local
  php: "8.4"

runtime:
  php: "8.4"
  node: "22"

services:
  mysql: "8.4"

ports:
  http: 8081
  https: 8444
  mysql: 3307
  php: 19000
  mailpit_smtp: 1025
  mailpit_http: 8025

components:
  mysql: true
  phpmyadmin: false
  mailpit: true          # opt-in; SMTP + UI mailpit.test

db:
  name: siso
  charset: utf8mb4
  collation: utf8mb4_unicode_ci
  seed: db/seed.sql

env:
  APP_ENV: local
  APP_DEBUG: "true"
```

## 3. Field

### Root

| Field | Tipe | Wajib | Default | Validasi |
|---|---|---|---|---|
| `schema` | int | Ya | - | Harus `1` |
| `name` | string | Ya | - | `^[a-z0-9][a-z0-9-]{0,62}$` |
| `web` | object | Ya | - | Lihat bawah |
| `runtime` | map | Tidak | `{}` | Nama runtime yang dikenal |
| `services` | map | Tidak | `{mysql: "8.4"}` | Nama service dan versi |
| `db` | object | Tidak | nama project | Lihat bawah |
| `ports` | object | Tidak | otomatis | Port HTTP/HTTPS/MySQL/PHP/Mailpit |
| `components` | map[string]bool | Tidak | semua aktif (kecuali `mailpit`) | `mysql`, `phpmyadmin`, `mailpit` |
| `env` | map[string]string | Tidak | `{}` | Hanya untuk proses project, bukan service |
| `x-*` | any | Tidak | - | Ekstensi, diabaikan core |

### `web`

| Field | Tipe | Wajib | Default | Validasi |
|---|---|---|---|---|
| `server` | enum | Tidak | `nginx` | `nginx` (v0.1.0) |
| `root` | string | Tidak | auto-detect | Path relatif, tidak boleh keluar project |
| `domain` | string | Tidak | `<name>.<tld>` | Domain utama, valid dan unik |
| `tld` | string | Tidak | `test` | `^[a-z]{2,}$` |
| `aliases` | []string | Tidak | `[]` | Domain valid, unik |
| `php` | string | Tidak | `runtime.php` | Versi PHP untuk vhost |
| `https` | bool | Tidak | `true` | - |

Auto-detect root, berurutan:

1. `public/` jika ada `index.php`
2. `web/` jika ada `index.php`
3. `www/` jika ada `index.php`
4. root project jika ada `index.php`
5. gagal dengan hint `web.root`

### `db`

| Field | Tipe | Wajib | Default |
|---|---|---|---|
| `name` | string | Tidak | `name` project, `-` -> `_` |
| `charset` | string | Tidak | `utf8mb4` |
| `collation` | string | Tidak | `utf8mb4_unicode_ci` |
| `seed` | string | Tidak | `db/seed.sql` bila ada |
| `backup_dir` | string | Tidak | state dir |

### `ports`

| Field | Tipe | Default | Keterangan |
|---|---|---|---|
| `http` | int | `0` (auto 80/8080) | Port HTTP vhost |
| `https` | int | `0` (auto 443/8443) | Port HTTPS vhost |
| `mysql` | int | `0` (auto 3306-3310) | Port MySQL |
| `php` | int | `0` (ephemeral) | Port FastCGI php-cgi |
| `mailpit_smtp` | int | `0` (auto 1025-1027) | Port SMTP Mailpit |
| `mailpit_http` | int | `0` (auto 8025-8027) | Port web UI Mailpit |

Port `0` atau kosong berarti otomatis. Port eksplisit yang terpakai menghasilkan error.

### `components`

| Nama | Default | Efek |
|---|---|---|
| `mysql` | `true` | Jalankan MySQL dan database project |
| `phpmyadmin` | `true` | Pasang dan layani phpMyAdmin |
| `mailpit` | `false` | Aktifkan Mailpit (SMTP + web UI `mailpit.test`) |

Komponen yang tidak dikenal ditolak. `phpmyadmin` otomatis dilewati bila `mysql` dimatikan. `mailpit` bersifat opt-in: hanya aktif bila diset `true`.

## 4. Aturan validasi

1. Unknown field di root, `web`, `db` -> error, kecuali prefix `x-`.
2. Versi runtime boleh parsial (`8.4`, `8`, `22`) dan diresolusi ke eksak.
3. Versi yang tidak tersedia -> error dengan daftar versi yang tersedia.
4. `web.root` tidak boleh absolut, tidak boleh mengandung `..`, harus di dalam project.
5. `name` unik per workspace.
6. Domain dihitung: `<name>.<tld>` + aliases. Bentrok domain -> error.
7. `env` tidak boleh memuat kunci yang dicadangkan: `PATH`, `MYSQL_PWD`, `DEVLOK_*`.
8. Nilai rahasia tidak boleh di manifest. Gunakan `.env` project.
9. File maksimum 1 MB.
10. `schema` lebih baru dari yang didukung -> error dengan saran upgrade.

## 5. Precedence konfigurasi

```
default internal
  < config.json global
    < .devlok.yaml project
      < .devlok.lock versi eksak
        < flag CLI eksplisit
```

Lockfile tidak menimpa preferensi non-versi (port, TLD), hanya versi dan checksum.

## 6. `.devlok.lock`

```yaml
schema: 1
generated_at: "2026-10-20T10:00:00Z"
platform: windows/amd64
name: siso

runtime:
  php:
    version: 8.4.23
    sha256: "abcdef..."
    url: "https://windows.php.net/downloads/releases/php-8.4.23-nts-Win32-vs17-x64.zip"
  node:
    version: 22.14.0
    sha256: "..."
    url: "https://nodejs.org/dist/v22.14.0/node-v22.14.0-win-x64.zip"

services:
  mysql:
    version: 8.4.10
    sha256: "..."
    port: 3306

web:
  server: nginx
  version: 1.30.6
  http_port: 80
  https_port: 443
```

Aturan:

- Lock ditulis hanya oleh `dl lock`.
- `dl up` memakai versi di lock; jika lock tidak ada, resolve dari manifest lalu sarankan `dl lock`.
- Perbedaan manifest dan lock bukan error, tetapi menghasilkan peringatan.
- Lock bersifat platform-specific. Untuk tim lintas OS, lock menyimpan map per platform (v0.2.0).
- Checksum diverifikasi ulang saat runtime belum ada di cache.

## 7. Contoh WordPress

```yaml
schema: 1
name: blog
web:
  root: .
  php: "8.3"
runtime:
  php: "8.3"
services:
  mysql: "8.4"
db:
  name: blog
  seed: db/seed.sql
env:
  WP_ENVIRONMENT_TYPE: local
```

## 8. Contoh Laravel

```yaml
schema: 1
name: api
web:
  root: public
  php: "8.4"
runtime:
  php: "8.4"
  node: "22"
services:
  mysql: "8.4"
db:
  name: api
env:
  APP_ENV: local
  APP_URL: "https://api.test"
```

## 9. Kompatibilitas

- `schema: 1` adalah versi pertama.
- Perubahan yang memecah kompatibilitas menaikkan `schema`.
- Core mendukung minimal dua schema terakhir.
- `dl init` selalu menulis schema terbaru.
- `dl doctor` memeriksa schema manifest dan menyarankan migrasi.
