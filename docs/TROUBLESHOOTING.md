# Troubleshooting

## MySQL gagal setelah pindah versi

DevLok memakai data directory terpisah per seri MySQL:

```text
%USERPROFILE%\.devlok\data\mysql\8.4
%USERPROFILE%\.devlok\data\mysql\9.4
```

Berpindah dari `8.4` ke `9.4` **tidak memindahkan tabel** dari data lama. Versi baru akan memakai data directory baru dan database project yang kosong.

### Gejala

- Service MySQL 9.x failed atau tidak mau start.
- Website menampilkan error koneksi database.
- Login aplikasi gagal setelah upgrade.

### Penyebab umum

1. Port MySQL bentrok.
   - Laragon atau instance MySQL lain memakai `3306`.
   - DevLok memakai port lain, tetapi `.env` project masih menunjuk `3306`.
2. Data directory 9.x sudah pernah dibuat dengan konfigurasi/kredensial lama.
3. Config custom `%USERPROFILE%\.devlok\config\mysql\9.4.ini` tidak valid.
4. Tabel belum dimigrasikan ke instance baru.

### Urutan aman

Sebelum pindah versi:

```powershell
cd C:\laragon\www\vakpro
C:\DevLok\build\bin\dl.exe db backup vakpro -o vakpro-8.4.sql.gz
```

Pindah versi lewat Konfigurasi atau tab Update. Setelah MySQL 9.x berjalan:

```powershell
C:\DevLok\build\bin\dl.exe db restore vakpro -f vakpro-8.4.sql.gz --yes
```

Jika service gagal start:

```powershell
C:\DevLok\build\bin\dl.exe status
C:\DevLok\build\bin\dl.exe ports
Get-Content "$env:USERPROFILE\.devlok\logs\mysql\9.4-error.log" -Tail 100
```

Jika port bentrok, kosongkan `ports.mysql` di manifest agar DevLok memilih port bebas, lalu perbarui `.env` project.

Jika data directory 9.x rusak dan sudah ada backup:

```powershell
Remove-Item "$env:USERPROFILE\.devlok\data\mysql\9.4" -Recurse -Force
Remove-Item "$env:USERPROFILE\.devlok\state\mysql-9.4.json" -Force
Remove-Item "$env:USERPROFILE\.devlok\secrets\mysql-9.4.dpapi" -Force
```

Setelah itu jalankan ulang stack dan restore backup.

Jangan menyalin data directory 8.4 ke 9.4. Gunakan dump/restore.

## Update runtime dari GUI

Tab Update > runtime > **Update ke versi terbaru** akan:

1. Mengunduh runtime.
2. Memasang versi baru.
3. Memperbarui manifest dan lockfile project.
4. Restart stack jika project aktif.

Untuk MySQL, data tidak dimigrasikan otomatis.