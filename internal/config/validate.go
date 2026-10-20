package config

import (
	"fmt"
	"path/filepath"
	"strings"

	"devlok/internal/devlokerr"
)

var allowedRuntime = map[string]bool{"php": true, "node": true, "nginx": true}
var allowedServices = map[string]bool{"mysql": true}
var reservedEnv = map[string]bool{
	"PATH":       true,
	"MYSQL_PWD":  true,
	"SYSTEMROOT": true,
	"WINDIR":     true,
}

func Validate(m *Manifest, baseDir string) error {
	if m.Schema != SchemaVersion {
		return invalid("Versi schema tidak didukung: "+fmt.Sprint(m.Schema), "Gunakan schema: 1 atau jalankan dl doctor untuk migrasi.")
	}
	if !nameRe.MatchString(m.Name) {
		return invalid("Nama project tidak valid: "+m.Name, "Gunakan huruf kecil, angka, dan tanda hubung, maksimal 63 karakter.")
	}
	if err := validateExtensions(m.Extensions, "root"); err != nil {
		return err
	}
	if err := validateWeb(m, baseDir); err != nil {
		return err
	}
	for k, v := range m.Runtime {
		if !allowedRuntime[k] {
			return invalid("Runtime tidak dikenal: "+k, "Runtime yang didukung: php, node.")
		}
		if !versionRe.MatchString(v) {
			return invalid("Versi runtime tidak valid untuk "+k+": "+v, "Contoh: 8.4 atau 8.4.23.")
		}
	}
	for k, v := range m.Services {
		if !allowedServices[k] {
			return invalid("Service tidak dikenal: "+k, "Service yang didukung: mysql.")
		}
		if !versionRe.MatchString(v) {
			return invalid("Versi service tidak valid untuk "+k+": "+v, "Contoh: 8.4.")
		}
	}
	if err := validateDatabase(m); err != nil {
		return err
	}
	for k := range m.Env {
		up := strings.ToUpper(k)
		if reservedEnv[up] || strings.HasPrefix(up, "DEVLOK_") {
			return invalid("Env dicadangkan: "+k, "Hapus kunci tersebut dari manifest atau ganti namanya.")
		}
	}
	return nil
}

func validateWeb(m *Manifest, baseDir string) error {
	if err := validateExtensions(m.Web.Extensions, "web"); err != nil {
		return err
	}
	if m.Web.Server != "nginx" {
		return invalid("Web server tidak didukung: "+m.Web.Server, "v0.1.0 hanya mendukung nginx.")
	}
	if !tldRe.MatchString(m.Web.TLD) {
		return invalid("TLD tidak valid: "+m.Web.TLD, "Gunakan huruf kecil, contoh: test atau local.")
	}
	if m.Web.PHP != "" && !versionRe.MatchString(m.Web.PHP) {
		return invalid("Versi PHP tidak valid: "+m.Web.PHP, "Contoh: 8.4 atau 8.4.23.")
	}
	if m.Web.Domain != "" && !validDomain(m.Web.Domain) {
		return invalid("Domain utama tidak valid: "+m.Web.Domain, "Contoh: app.local atau siso.test.")
	}
	if err := validatePorts(m.Ports); err != nil {
		return err
	}
	for name := range m.Components {
		if name != "mysql" && name != "phpmyadmin" {
			return invalid("Komponen tidak dikenal: "+name, "Komponen yang didukung: mysql, phpmyadmin.")
		}
	}
	if err := validateRoot(m.Web.Root, baseDir); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, d := range m.Domains() {
		if !validDomain(d) {
			return invalid("Domain tidak valid: "+d, "Gunakan label huruf kecil, angka, dan tanda hubung.")
		}
		if seen[d] {
			return invalid("Domain duplikat: "+d, "Hapus alias yang sama dengan domain utama.")
		}
		seen[d] = true
	}
	return nil
}

func validateRoot(root, baseDir string) error {
	if root == "" {
		return invalid("web.root tidak dapat ditentukan.", "Tambahkan index.php di public/, web/, www/, atau root project, atau isi web.root secara eksplisit.")
	}
	if filepath.IsAbs(root) || filepath.VolumeName(root) != "" {
		return invalid("web.root harus relatif terhadap project: "+root, "Gunakan path relatif, contoh: public.")
	}
	clean := filepath.Clean(root)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return invalid("web.root keluar dari folder project: "+root, "Gunakan path relatif di dalam project.")
	}
	if baseDir != "" {
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return invalid("Folder project tidak valid: "+baseDir, "")
		}
		absRoot := filepath.Join(absBase, clean)
		rel, err := filepath.Rel(absBase, absRoot)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return invalid("web.root keluar dari folder project: "+root, "Gunakan path relatif di dalam project.")
		}
	}
	return nil
}

func validateDatabase(m *Manifest) error {
	if err := validateExtensions(m.DB.Extensions, "db"); err != nil {
		return err
	}
	if !dbNameRe.MatchString(m.DB.Name) {
		return invalid("Nama database tidak valid: "+m.DB.Name, "Gunakan huruf kecil, angka, dan garis bawah, diawali huruf.")
	}
	if m.DB.Seed != "" {
		if filepath.IsAbs(m.DB.Seed) || filepath.VolumeName(m.DB.Seed) != "" {
			return invalid("db.seed harus relatif: "+m.DB.Seed, "Contoh: db/seed.sql.")
		}
	}
	if m.DB.BackupDir != "" {
		if filepath.IsAbs(m.DB.BackupDir) || filepath.VolumeName(m.DB.BackupDir) != "" {
			return invalid("db.backup_dir harus relatif atau dikosongkan: "+m.DB.BackupDir, "Biarkan kosong untuk memakai state dir.")
		}
	}
	return nil
}

func validateExtensions(ext map[string]any, scope string) error {
	for k := range ext {
		if !strings.HasPrefix(k, "x-") {
			return invalid("Field tidak dikenal di "+scope+": "+k, "Field ekstensi harus berawalan x-.")
		}
	}
	return nil
}

func ValidDomain(d string) bool { return validDomain(d) }

func validatePorts(p PortsConfig) error {
	for name, port := range map[string]int{"http": p.HTTP, "https": p.HTTPS, "mysql": p.MySQL, "php": p.PHP} {
		if port < 0 || port > 65535 {
			return invalid("Port "+name+" tidak valid: "+fmt.Sprint(port), "Gunakan 1-65535 atau 0 untuk otomatis.")
		}
	}
	if p.HTTP > 0 && p.HTTPS > 0 && p.HTTP == p.HTTPS {
		return invalid("Port HTTP dan HTTPS tidak boleh sama.", "Gunakan port berbeda.")
	}
	return nil
}

func validDomain(d string) bool {
	if len(d) == 0 || len(d) > 253 {
		return false
	}
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if !labelRe.MatchString(l) {
			return false
		}
	}
	return true
}

func invalid(msg, hint string) error {
	return devlokerr.New(devlokerr.CodeInvalidManifest, msg, hint)
}
