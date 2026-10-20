package web

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"devlok/internal/devlokerr"
	"devlok/internal/hidden"
	"devlok/internal/usercfg"
)

type Server struct {
	Name        string
	Domain      string
	Aliases     []string
	Root        string
	PHPUpstream string
	HTTPS       bool
	CertFile    string
	KeyFile     string
}

type Config struct {
	PrefixDir string
	ConfPath  string
	SitesDir  string
	CustomDir string
	LogsDir   string
	MimeTypes string
	HTTPPort  int
	HTTPSPort int
	Servers   []Server
}

func Generate(cfg Config) error {
	if cfg.ConfPath == "" || cfg.SitesDir == "" || cfg.LogsDir == "" {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Konfigurasi web tidak lengkap.", "")
	}
	if err := os.MkdirAll(cfg.SitesDir, 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder site Nginx tidak dapat dibuat.", "")
	}
	if err := os.MkdirAll(cfg.LogsDir, 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder log Nginx tidak dapat dibuat.", "")
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ConfPath), 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder konfigurasi Nginx tidak dapat dibuat.", "")
	}
	for _, srv := range cfg.Servers {
		if err := validateServer(srv); err != nil {
			return err
		}
		customPath := ""
		if cfg.CustomDir != "" {
			if err := os.MkdirAll(cfg.CustomDir, 0o755); err != nil {
				return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder konfigurasi Nginx pengguna tidak dapat dibuat.", "")
			}
			customPath = filepath.Join(cfg.CustomDir, srv.Name+".conf")
			header := "# DevLok Nginx override untuk site " + srv.Name + "\n# Di-include di dalam blok server.\n# Contoh:\n# client_max_body_size 128m;\n# fastcgi_read_timeout 300s;\n"
			if err := usercfg.Ensure(customPath, header); err != nil {
				return err
			}
		}
		content := renderSite(cfg, srv, customPath)
		path := filepath.Join(cfg.SitesDir, srv.Name+".conf")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Konfigurasi site tidak dapat ditulis: "+srv.Name, "")
		}
	}
	if err := cleanupSites(cfg.SitesDir, cfg.Servers); err != nil {
		return err
	}
	main := renderMain(cfg)
	if err := os.WriteFile(cfg.ConfPath, []byte(main), 0o644); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Konfigurasi Nginx tidak dapat ditulis.", "")
	}
	return nil
}

func Validate(nginxExe, prefixDir, confPath string) error {
	cmd := exec.Command(nginxExe, "-t", "-p", prefixDir, "-c", confPath)
	hidden.Hide(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Konfigurasi Nginx tidak valid.", strings.TrimSpace(string(out)))
	}
	return nil
}

func validateServer(srv Server) error {
	if !nameRe.MatchString(srv.Name) {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Nama site tidak valid: "+srv.Name, "")
	}
	if srv.Domain == "" || srv.Root == "" || srv.PHPUpstream == "" {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Data site tidak lengkap: "+srv.Name, "")
	}
	if srv.HTTPS && (srv.CertFile == "" || srv.KeyFile == "") {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Certificate site tidak lengkap: "+srv.Name, "")
	}
	if _, err := os.Stat(srv.Root); err != nil {
		return devlokerr.New(devlokerr.CodeDependencyMissing, "Document root tidak ditemukan: "+srv.Root, "")
	}
	return nil
}

func renderMain(cfg Config) string {
	var b strings.Builder
	b.WriteString("worker_processes 1;\n")
	b.WriteString("error_log " + quote(cfg.LogsDir+"/error.log") + " warn;\n")
	b.WriteString("pid " + quote(cfg.LogsDir+"/nginx.pid") + ";\n\n")
	b.WriteString("events {\n    worker_connections 2048;\n}\n\n")
	b.WriteString("http {\n")
	if cfg.MimeTypes != "" {
		b.WriteString("    include " + quote(cfg.MimeTypes) + ";\n")
	}
	b.WriteString("    default_type application/octet-stream;\n")
	b.WriteString("    access_log " + quote(cfg.LogsDir+"/access.log") + ";\n")
	b.WriteString("    sendfile on;\n")
	b.WriteString("    keepalive_timeout 65;\n")
	b.WriteString("    server_tokens off;\n")
	b.WriteString("    client_max_body_size 64m;\n")
	b.WriteString("    gzip on;\n")
	b.WriteString("    gzip_types text/plain text/css application/json application/javascript text/xml application/xml;\n")
	b.WriteString("    include " + quote(cfg.SitesDir+"/*.conf") + ";\n")
	b.WriteString("}\n")
	return b.String()
}

func renderSite(cfg Config, srv Server, customPath string) string {
	names := uniqueStrings(append([]string{srv.Domain}, srv.Aliases...))
	var b strings.Builder
	if srv.HTTPS {
		fmt.Fprintf(&b, "server {\n    listen %d;\n    server_name %s;\n", cfg.HTTPPort, strings.Join(names, " "))
		if cfg.HTTPSPort == 443 {
			b.WriteString("    return 301 https://$host$request_uri;\n}\n\n")
		} else {
			fmt.Fprintf(&b, "    return 301 https://$host:%d$request_uri;\n}\n\n", cfg.HTTPSPort)
		}
		fmt.Fprintf(&b, "server {\n    listen %d ssl;\n    http2 on;\n", cfg.HTTPSPort)
	} else {
		fmt.Fprintf(&b, "server {\n    listen %d;\n", cfg.HTTPPort)
	}
	fmt.Fprintf(&b, "    server_name %s;\n", strings.Join(names, " "))
	fmt.Fprintf(&b, "    root %s;\n", quote(srv.Root))
	b.WriteString("    index index.php index.html index.htm;\n")
	if srv.HTTPS {
		fmt.Fprintf(&b, "    ssl_certificate %s;\n", quote(srv.CertFile))
		fmt.Fprintf(&b, "    ssl_certificate_key %s;\n", quote(srv.KeyFile))
		b.WriteString("    ssl_protocols TLSv1.2 TLSv1.3;\n")
		b.WriteString("    ssl_ciphers HIGH:!aNULL:!MD5;\n")
		b.WriteString("    ssl_session_cache shared:SSL:10m;\n")
		b.WriteString("    ssl_session_timeout 10m;\n")
	}
	b.WriteString("    add_header X-Content-Type-Options nosniff always;\n")
	b.WriteString("    add_header X-Frame-Options SAMEORIGIN always;\n")
	b.WriteString("    location / {\n        try_files $uri $uri/ /index.php?$query_string;\n    }\n")
	fmt.Fprintf(&b, "    location ~ \\.php$ {\n        try_files $uri =404;\n        fastcgi_pass %s;\n        fastcgi_index index.php;\n", srv.PHPUpstream)
	if cfg.PrefixDir != "" {
		b.WriteString("        include " + quote(cfg.PrefixDir+"/conf/fastcgi_params") + ";\n")
	}
	b.WriteString("        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n    }\n")
	b.WriteString("    location ~ /\\.(ht|git) {\n        deny all;\n    }\n")
	if customPath != "" {
		fmt.Fprintf(&b, "    include %s;\n", quote(customPath))
	}
	b.WriteString("}\n")
	return b.String()
}

func cleanupSites(dir string, servers []Server) error {
	keep := map[string]bool{}
	for _, s := range servers {
		keep[s.Name+".conf"] = true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder site tidak dapat dibaca.", "")
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") || keep[e.Name()] {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Konfigurasi site lama tidak dapat dihapus.", "")
		}
	}
	return nil
}

func quote(p string) string {
	p = filepath.ToSlash(p)
	p = strings.ReplaceAll(p, `"`, `\"`)
	return `"` + p + `"`
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
