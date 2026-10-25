//go:build darwin

package stack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"devlok/internal/brew"
	"devlok/internal/devlokerr"
	rt "devlok/internal/runtime"
)

func installPlatformRuntime(ctx context.Context, mgr *rt.Manager, name, constraint, wantSHA, stateDir string) (rt.Installed, error) {
	switch name {
	case "nginx":
		return brewNginx(ctx, stateDir)
	case "php":
		return brewPHP(ctx, constraint)
	default:
		return installRuntime(ctx, mgr, name, constraint, wantSHA)
	}
}

func brewNginx(ctx context.Context, stateDir string) (rt.Installed, error) {
	prefix, err := brew.Prefix(ctx, "nginx")
	if err != nil {
		return rt.Installed{}, err
	}
	base, err := brew.Prefix(ctx, "")
	if err != nil {
		return rt.Installed{}, err
	}
	binary := filepath.Join(prefix, "bin", "nginx")
	if _, err := os.Stat(binary); err != nil {
		return rt.Installed{}, devlokerr.New(devlokerr.CodeDependencyMissing, "Nginx Homebrew tidak lengkap.", "Jalankan brew install nginx.")
	}
	confDir := filepath.Join(base, "etc", "nginx")
	if _, err := os.Stat(filepath.Join(confDir, "mime.types")); err != nil {
		confDir = filepath.Join(prefix, "conf")
	}
	version := brew.Version(ctx, "nginx")
	if version == "" {
		version = "brew"
	}
	shim := filepath.Join(stateDir, "runtimes", "nginx", version)
	if err := os.MkdirAll(shim, 0o755); err != nil {
		return rt.Installed{}, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Shim Nginx tidak dapat dibuat.", "")
	}
	if err := replaceSymlink(filepath.Join(shim, "nginx"), binary); err != nil {
		return rt.Installed{}, err
	}
	if err := replaceSymlink(filepath.Join(shim, "conf"), confDir); err != nil {
		return rt.Installed{}, err
	}
	return rt.Installed{Name: "nginx", Version: version, Path: shim}, nil
}

func brewPHP(ctx context.Context, requested string) (rt.Installed, error) {
	formula := brew.InstalledFormula(ctx, "php", requested)
	prefix, err := brew.Prefix(ctx, formula)
	if err != nil {
		return rt.Installed{}, err
	}
	phpBin := filepath.Join(prefix, "bin", "php")
	out, err := exec.CommandContext(ctx, phpBin, "-r", "echo PHP_VERSION;").Output()
	if err != nil {
		return rt.Installed{}, devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "PHP Homebrew tidak dapat dijalankan.", "Jalankan brew install "+formula+".")
	}
	version := strings.TrimSpace(string(out))
	if version == "" {
		version = "brew"
	}
	return rt.Installed{Name: "php", Version: version, Path: prefix}, nil
}

func replaceSymlink(path, target string) error {
	if _, err := os.Lstat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Shim runtime tidak dapat diganti.", "")
		}
	}
	if err := os.Symlink(target, path); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, fmt.Sprintf("Shim %s tidak dapat dibuat.", filepath.Base(path)), "")
	}
	return nil
}
