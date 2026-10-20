package config

import (
	"os"
	"path/filepath"

	"devlok/internal/devlokerr"
)

type InitOptions struct {
	Dir        string
	Name       string
	Root       string
	TLD        string
	PHP        string
	Domain     string
	HTTPPort   int
	HTTPSPort  int
	NoDatabase bool
	Force      bool
}

func Init(opts InitOptions) (string, *Manifest, error) {
	dir := opts.Dir
	if dir == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", nil, devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Folder project tidak valid.", "")
	}
	info, err := os.Stat(absDir)
	if err != nil || !info.IsDir() {
		return "", nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Folder project tidak ditemukan: "+absDir, "Buat folder terlebih dahulu.")
	}
	name := opts.Name
	if name == "" {
		name = SanitizeName(filepath.Base(absDir))
	} else {
		name = SanitizeName(name)
	}
	tld := opts.TLD
	if tld == "" {
		tld = "test"
	}
	php := opts.PHP
	if php == "" {
		php = "8.4"
	}
	root := opts.Root
	if root == "" {
		root = DetectRoot(absDir)
		if root == "" {
			root = "."
		}
	}
	m := &Manifest{
		Schema: SchemaVersion,
		Name:   name,
		Web: Web{
			Server: "nginx",
			Root:   root,
			Domain: opts.Domain,
			TLD:    tld,
			PHP:    php,
		},
		Runtime:  map[string]string{"php": php},
		Services: map[string]string{"mysql": "8.4"},
		Ports: PortsConfig{
			HTTP:  opts.HTTPPort,
			HTTPS: opts.HTTPSPort,
		},
	}
	ApplyDefaults(m)
	if opts.NoDatabase {
		m.Components = map[string]bool{"mysql": false, "phpmyadmin": false}
	}
	if err := Validate(m, absDir); err != nil {
		return "", nil, err
	}
	path := filepath.Join(absDir, FileName)
	if _, err := os.Stat(path); err == nil && !opts.Force {
		return "", nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Manifest sudah ada: "+path, "Gunakan --force untuk menimpa.")
	}
	data, err := Marshal(m)
	if err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Manifest tidak dapat ditulis.", "Periksa izin folder project.")
	}
	return path, m, nil
}
