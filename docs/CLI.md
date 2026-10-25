# Spesifikasi CLI

Binary: `dl.exe` (Windows) / `dl` (macOS)

## 1. Prinsip

- Semua aksi GUI tersedia di CLI.
- Perintah yang relevan untuk automasi mendukung `--json`.
- Perintah yang mengubah sistem mendukung `--dry-run` bila masuk akal.
- Default non-destruktif; aksi destruktif butuh `--yes` di mode non-interaktif.
- Exit code stabil untuk automasi.

## 2. Global flag

| Flag | Default | Fungsi |
|---|---|---|
| `--json` | false | Output JSON |
| `--verbose` | false | Log detail |

Catatan: `--yes` hanya ada pada perintah destruktif (mis. `db drop`, `update apply`). Flag lain (`--path`, `--state`, `--workspace`, `--php`, dst.) bersifat per-perintah.

## 3. Perintah

### Diagnosa

```powershell
dl doctor [--json] [--manifest path]
dl version [--json]
```

`doctor` memeriksa OS, arsitektur, dependency (WebView2/VC++ di Windows; Homebrew di macOS), state dir, hosts, port, dan manifest.

### Project

```powershell
dl init [path] [--name demo] [--root public] [--tld test] [--php 8.4] [--domain app.local] [--http-port 0] [--https-port 0] [--no-db] [--force]
dl lock [--path dir]
dl up [--path dir] [--skip-hosts] [--skip-trust]
dl down
dl status [--json]
dl export [-o env.zip]
```

### Runtime dan update

```powershell
dl runtime list [--json]
dl runtime add php@8.4 [--offline]
dl runtime remove php@8.3.35
dl runtime verify [php@8.4]
dl update [--check] [--json]
dl update download
dl update apply --yes
dl update rollback --yes
```

`runtime list` menampilkan versi terpasang/tersedia untuk platform aktif beserta kolom `CHANNEL` (label `stable`/`lts`/`latest`/`legacy`/`eol`/pra-rilis, dan penanda `disarankan` untuk versi unggulan) agar pemilihan versi lebih terseleksi. `runtime add` memverifikasi SHA256; `--offline` hanya memakai cache. `dl update` memeriksa aplikasi + runtime. Di macOS, runtime diperbarui lewat `brew upgrade` dan `dl update apply` melakukan self-update `.app`.

### Site

```powershell
dl site list
dl site create <name> [--workspace path] [--php 8.4] [--domain app.local] [--http-port 0] [--https-port 0] [--no-db]
dl site remove <name> [--delete]
dl site adopt <path>
```

### Supervisor

```powershell
dl supervise [--state dir] [--once] [--json]
```

Supervisor latar memantau liveness/health service dan me-restart dengan backoff. Perintah ini **internal** (disembunyikan dari `--help`) dan dijalankan otomatis oleh `dl up`.

### Database

```powershell
dl db list
dl db create [name] [--mysql 8.4]
dl db drop <name> --yes
dl db reset --yes
dl db backup <name> [-o file.sql.gz]
dl db restore <name> -f file.sql.gz --yes
dl db seed [name]
dl db shell [name]
dl db creds [name] [--root]
dl db admin
```

### Wildcard (macOS)

```bash
dl wildcard status
dl wildcard on --tld test
dl wildcard off --tld test
```

### Import dan tool

```powershell
dl import laragon <path> [--dry-run] [--yes] [--adopt]
dl edit [path]
dl ports [--json] [--port 9000]
dl run -- php artisan
dl shell
dl ui
```

### Direncanakan

- `dl ui --browser` dan browser fallback.
- `dl support bundle`.
- Forwarding perintah GUI lewat named pipe.
- Migrasi schema manifest.

## 4. Contoh output manusia

```text
$ dl up

DevLok 0.2.7

Runtime
  php 8.4.23        ok (cache)
  nginx 1.30.6      ok (cache)
  mysql 8.4.10      ok (cache)

Services
  mysql@8.4         127.0.0.1:3306   running
  php@8.4           127.0.0.1:10987  running
  nginx             127.0.0.1:443    running

Sites
  siso              https://siso.test   public/

Database
  siso              user siso          ok

Ready in 4.2s
```

## 5. Contoh output JSON

Semua perintah mengembalikan envelope:

```json
{
  "ok": true,
  "command": "up",
  "duration_ms": 4210,
  "data": {
    "services": [
      {"name": "nginx", "pid": 2001, "port": 443, "status": "running"},
      {"name": "php@8.4", "pid": 2002, "port": 10987, "status": "running"},
      {"name": "mysql@8.4", "pid": 2003, "port": 3306, "status": "running"}
    ],
    "sites": [
      {"name": "siso", "url": "https://siso.test", "root": "C:/dev/siso/public"}
    ]
  }
}
```

Error:

```json
{
  "ok": false,
  "command": "up",
  "error": {
    "code": "PORT_IN_USE",
    "message": "Port 80 dipakai proses lain.",
    "hint": "Jalankan dl ports atau gunakan --http-port 8080.",
    "details": {"port": 80, "pid": 4321, "process": "httpd.exe"}
  }
}
```

## 6. Exit code

| Code | Arti |
|---|---|
| 0 | Sukses |
| 1 | Error umum |
| 2 | Validasi manifest/argumen |
| 3 | Konflik port |
| 4 | Butuh elevation |
| 5 | Dependency/runtime tidak tersedia |
| 6 | Timeout |
| 7 | Operasi dibatalkan pengguna |
| 8 | State tidak konsisten, perlu `dl doctor` |

## 7. Perilaku interaktif

- Tanpa `--yes`, aksi destruktif menampilkan konfirmasi.
- Di lingkungan non-TTY, aksi destruktif gagal dengan exit 7 dan hint `--yes`.
- `--json` menyiratkan non-interaktif.
- Progress ditulis ke stderr agar stdout tetap bersih untuk JSON.

## 8. Completion

```powershell
dl completion powershell | Out-String | Invoke-Expression
```

## 9. Autostart (direncanakan)

Belum ada perintah autostart. Rencananya:

```powershell
dl config set ui.autostart true
dl config set ui.minimize_on_close true
```

Pada portable mode, autostart tidak diaktifkan.
