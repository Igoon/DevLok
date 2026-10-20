# Lapisan Platform

## 1. Tujuan

Windows adalah target v0.1.0. macOS adalah target v0.2.0. Agar port murah, semua perbedaan OS diisolasi di `internal/platform`.

## 2. Aturan

1. Core tidak boleh mengimpor `golang.org/x/sys/windows`.
2. Core tidak boleh memakai `runtime.GOOS`; hanya factory platform.
3. Semua path memakai `filepath.Join` dan `os.UserConfigDir`/`os.UserCacheDir`, bukan string `C:\`.
4. Semua eksekusi biner memakai nama tanpa `.exe`; platform menambahkan suffix.
5. Semua pembunuhan proses lewat `Platform.KillTree`, bukan `exec.Command("taskkill")` di core.
6. Semua operasi privileged lewat `Platform.Elevate` atau helper.
7. Semua pembukaan browser/terminal lewat `Platform`.
8. Template config tidak boleh memuat path OS-specific; path disuntikkan sebagai parameter.
9. Build tag: `//go:build windows` dan `//go:build darwin`.
10. CI mengompilasi core untuk kedua OS sejak M0, meski fitur Mac belum ada.

## 3. Pemetaan operasi

| Operasi | Windows | macOS |
|---|---|---|
| State dir | `%USERPROFILE%\.devlok` | `~/Library/Application Support/DevLok` |
| Cache | state dir `cache/` | `~/Library/Caches/DevLok` |
| Hosts | `%WINDIR%\System32\drivers\etc\hosts` | `/etc/hosts` |
| Elevation | UAC + helper | `osascript` dengan administrator privileges atau LaunchDaemon helper |
| Tulis hosts privileged | helper + allowlist | helper + allowlist |
| Trust CA | `certutil -user -addstore Root` | `security add-trusted-cert -d -r trustRoot -k ~/Library/Keychains/login.keychain-db` |
| Untrust CA | `certutil -user -delstore Root` | `security delete-certificate` |
| Kill tree | Job Object | `Setpgid` + `kill(-pgid, SIGKILL)` |
| Process start time | `GetProcessTimes` | `sysctl` / `ps -o lstart=` |
| Port owner | `GetExtendedTcpTable` | `lsof -nP -iTCP:<port> -sTCP:LISTEN` |
| Firewall | `netsh advfirewall` | `pf` (opsional, biasanya tidak perlu) |
| Flush DNS | `ipconfig /flushdns` | `dscacheutil -flushcache; killall -HUP mDNSResponder` |
| Open browser | `rundll32 url.dll,FileProtocolHandler` atau `cmd /c start` | `open` |
| Open terminal | Windows Terminal / `cmd.exe` | Terminal.app / iTerm |
| Open folder | `explorer.exe` | `open` |
| Path separator | `\` | `/` |
| Case sensitivity | Case-insensitive | Case-sensitive (default APFS) |
| Line ending config | `\n` eksplisit | `\n` |
| Runtime binary | `php-cgi.exe`, `nginx.exe`, `mysqld.exe` | `php-cgi`, `nginx`, `mysqld` |
| FastCGI | `php-cgi.exe -b` | `php-cgi -b` (atau php-fpm) |
| Installer | Inno Setup | `.app` + `.dmg` |
| Signing | Authenticode (opsional) | Developer ID + notarization (wajib) |
| Autostart | Registry `Run` per user | LaunchAgent |

## 4. Perbedaan perilaku yang harus ditangani

### 4.1 DNS

Windows `hosts` tidak mendukung wildcard. v0.1.0 menulis satu entri per site. macOS v0.2.0 dapat memakai mekanisme yang sama, dengan opsi dnsmasq untuk wildcard.

### 4.2 Elevation

Windows: helper dijalankan dengan UAC saat operasi privileged pertama, lalu tetap hidup selama sesi dengan pipe ACL.

macOS: helper dapat berupa LaunchDaemon yang di-install sekali, atau `osascript` per operasi. v0.2.0 memilih salah satu setelah spike.

### 4.3 Process group

Windows tidak punya process group POSIX. Job Object adalah pengganti. Di macOS, `Setpgid` dan sinyal ke `-pid` diperlukan; Job Object tidak ada.

### 4.4 Case sensitivity

Nama project dan file config harus diperlakukan case-sensitive secara logis, meski Windows case-insensitive. Jangan membuat dua site yang hanya beda kapitalisasi.

### 4.5 Runtime macOS

- PHP dan MySQL untuk macOS arm64 memerlukan sumber build yang tepat; Homebrew dapat menjadi fallback.
- Nginx macOS dapat dibangun dari source atau diambil dari paket terkelola.
- Manifest runtime memakai field `os` dan `arch`, sehingga sumber berbeda per platform adalah hal normal.

### 4.6 Signing

- macOS membutuhkan Apple Developer ID ($99/tahun) dan notarization agar tidak diblokir Gatekeeper.
- Windows SmartScreen dapat diatasi dengan certificate EV, tetapi tidak wajib untuk v0.1.0.
- Checksum rilis dipublikasikan terlepas dari signing.

## 5. Interface platform

```go
type ProcessInfo struct {
    PID     int
    Name    string
    ExePath string
}

type Platform interface {
    Name() string
    StateDir() (string, error)
    CacheDir() (string, error)

    HostsPath() string
    ReadHosts() ([]byte, error)
    WriteHosts(data []byte) error
    PrivilegedWriteHosts(block []byte) error
    FlushDNS() error

    TrustCA(certPath string) error
    UntrustCA(certPath string) error

    StartProcess(cmd *exec.Cmd) (Process, error)
    KillTree(pid int) error
    ProcessStartTime(pid int) (time.Time, error)
    PortOwner(port int) (ProcessInfo, error)

    OpenBrowser(url string) error
    OpenFolder(path string) error
    OpenTerminal(dir string, env []string) error

    ExeSuffix() string
    SupportsElevation() bool
    Elevate(op string, args []string) error
}
```

Implementasi:

- `platform_windows.go`
- `platform_darwin.go` (v0.2.0)
- `platform_stub.go` untuk build non-target agar CI tetap kompilasi

## 6. Checklist port macOS

- [ ] Semua file `_windows.go` punya padanan `_darwin.go`.
- [ ] Core kompilasi di `macos-latest`.
- [ ] Test unit lulus di macOS.
- [ ] Hosts, CA trust, kill tree, port owner teruji nyata.
- [ ] Runtime arm64 dan amd64 tersedia di registry.
- [ ] `.app` berjalan tanpa terminal.
- [ ] Tray dan menu bar berfungsi.
- [ ] Notarization lulus `spctl --assess`.
- [ ] Dokumentasi perbedaan perilaku diperbarui.

## 7. Estimasi port

| Pekerjaan | Hari |
|---|---|
| Platform darwin | 3 |
| Runtime registry macOS | 2 |
| Packaging `.app`/`.dmg` | 2 |
| Signing + notarization | 2 |
| dnsmasq opsional | 2 |
| Uji macOS 14/15 | 2 |
| **Total** | **13** |

Sekitar 3 minggu, dengan asumsi core Windows sudah stabil dan tidak ada kebocoran API OS di core.
