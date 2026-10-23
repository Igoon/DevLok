# Spesifikasi CLI

Binary: `dl.exe`

## 1. Prinsip

- Semua aksi GUI tersedia di CLI.
- Semua perintah mendukung `--json`.
- Perintah yang mengubah sistem mendukung `--dry-run` bila masuk akal.
- Default non-destruktif; aksi destruktif butuh `--yes` di mode non-interaktif.
- Exit code stabil untuk automasi.

## 2. Global flag

| Flag | Default | Fungsi |
|---|---|---|
| `--root <path>` | `%USERPROFILE%\.devlok` | State dir / portable mode |
| `--workspace <path>` | `%USERPROFILE%\DevLok\sites` | Folder project |
| `--json` | false | Output JSON |
| `--yes` | false | Setujui konfirmasi |
| `--verbose` | false | Log detail |
| `--no-color` | false | Matikan warna |
| `--timeout <dur>` | 120s | Timeout operasi |
| `--version` | - | Versi |
| `--help` | - | Bantuan |

## 3. Perintah

### Diagnosa

```powershell
dl doctor [--json]
dl version
dl support bundle [-o file.zip]
```

`doctor` memeriksa: Windows, arsitektur, WebView2, VC++ runtime, port 80/443/3306, akses hosts, trust CA, cache runtime, proses sisa, ruang disk, schema manifest.

### Project

```powershell
dl init [path] [--name siso] [--tld test] [--domain app.local] [--http-port 8081] [--https-port 8444] [--no-db]
dl lock [--path dir]
dl up [--path dir] [--skip-hosts] [--skip-trust]
dl down [--all]
dl restart [service]
dl status [--json]
dl open [site]
dl export [-o env.zip] [--path dir]
```

### Runtime

```powershell
dl runtime list [--json]
dl runtime add php@8.4 [--offline]
dl runtime remove php@8.3.35
dl runtime verify [php@8.4]
```

`runtime list` menampilkan semua versi yang tersedia untuk platform saat ini beserta status `installed` atau `available`. `runtime add` memverifikasi SHA256 sebelum ekstraksi; `--offline` hanya memakai cache. `runtime verify` memeriksa marker, executable, dan checksum arsip cache.

### Site

```powershell
dl site list
dl site create <name> [--php 8.4] [--domain app.local] [--http-port 8081] [--https-port 8444] [--no-db]
dl site remove <name> [--delete]
```

### Service

```powershell
dl service list
dl service start <name>
dl service stop <name>
dl service restart <name>
dl service logs <name> [--follow] [--level error]
```

Nama service: `nginx`, `php@8.4`, `mysql@8.4`.

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

### TLS

```powershell
dl tls status
dl tls trust
dl tls untrust
dl tls renew [site]
```

### Hosts

```powershell
dl hosts list
dl hosts sync
dl hosts clean
```

### Import

```powershell
dl import laragon <path> [--dry-run] [--yes]
dl import laragon <path> --site siso
dl run -- php artisan
dl shell
```

### Lain-lain

```powershell
dl config get <key>
dl config set <key> <value>
dl ui
dl logs [--follow] [--service nginx]
dl ports
dl edit [path]
dl completion powershell
```

## 4. Contoh output manusia

```text
$ dl up

DevLok 0.1.0

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

## 9. Autostart

```powershell
dl config set ui.autostart true
dl config set ui.minimize_on_close true
```

Pada portable mode, autostart selalu `false` dan tidak dapat diaktifkan.
