package doctor

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"devlok/internal/config"
	"devlok/internal/platform"
	"devlok/internal/version"
)

type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Check struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Report struct {
	Version     string    `json:"version"`
	Platform    string    `json:"platform"`
	GeneratedAt time.Time `json:"generated_at"`
	Checks      []Check   `json:"checks"`
}

func (r Report) OK() bool {
	for _, c := range r.Checks {
		if c.Status == StatusFail {
			return false
		}
	}
	return true
}

func (r Report) Counts() (ok, warn, fail int) {
	for _, c := range r.Checks {
		switch c.Status {
		case StatusOK:
			ok++
		case StatusWarn:
			warn++
		case StatusFail:
			fail++
		}
	}
	return
}

type Options struct {
	ManifestPath string
	WorkDir      string
	StateDir     string
	Platform     platform.Platform
	Ports        []int
}

func Run(ctx context.Context, opts Options) Report {
	plat := opts.Platform
	if plat == nil {
		plat = platform.Current()
	}
	if opts.WorkDir == "" {
		opts.WorkDir, _ = os.Getwd()
	}
	if len(opts.Ports) == 0 {
		opts.Ports = []int{80, 443, 3306}
	}
	rep := Report{
		Version:     version.Current().Version,
		Platform:    plat.Name(),
		GeneratedAt: time.Now().UTC(),
		Checks:      make([]Check, 0, 12),
	}
	rep.Checks = append(rep.Checks, platformChecks(plat)...)
	rep.Checks = append(rep.Checks, checkStateDir(plat, opts.StateDir))
	rep.Checks = append(rep.Checks, checkHosts(plat))
	rep.Checks = append(rep.Checks, checkPorts(ctx, opts.Ports)...)
	rep.Checks = append(rep.Checks, checkManifest(opts))
	return rep
}

func checkStateDir(plat platform.Platform, override string) Check {
	dir := override
	if dir == "" {
		var err error
		dir, err = plat.StateDir()
		if err != nil {
			return Check{Name: "state_dir", Status: StatusFail, Message: "State dir tidak dapat ditentukan.", Hint: err.Error()}
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Check{Name: "state_dir", Status: StatusFail, Message: "State dir tidak dapat dibuat: " + dir, Hint: "Periksa izin folder."}
	}
	probe := filepath.Join(dir, ".doctor-probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return Check{Name: "state_dir", Status: StatusFail, Message: "State dir tidak dapat ditulis: " + dir, Hint: "Periksa izin folder atau antivirus."}
	}
	_ = os.Remove(probe)
	return Check{Name: "state_dir", Status: StatusOK, Message: dir}
}

func checkHosts(plat platform.Platform) Check {
	path := plat.HostsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return Check{Name: "hosts", Status: StatusFail, Message: "Hosts tidak dapat dibaca: " + path, Hint: "Jalankan sebagai user biasa; file hosts biasanya dapat dibaca."}
	}
	if len(data) == 0 {
		return Check{Name: "hosts", Status: StatusWarn, Message: "Hosts kosong: " + path}
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return Check{Name: "hosts", Status: StatusWarn, Message: "Hosts tidak dapat ditulis tanpa elevation.", Hint: "DevLok akan meminta izin administrator saat menambah site."}
	}
	_ = f.Close()
	return Check{Name: "hosts", Status: StatusOK, Message: path + " (writable)"}
}

func checkPorts(ctx context.Context, ports []int) []Check {
	out := make([]Check, 0, len(ports))
	for _, port := range ports {
		if err := ctx.Err(); err != nil {
			out = append(out, Check{Name: "port_" + strconv.Itoa(port), Status: StatusWarn, Message: "Pemeriksaan port dibatalkan."})
			continue
		}
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			out = append(out, Check{
				Name:    "port_" + strconv.Itoa(port),
				Status:  StatusWarn,
				Message: fmt.Sprintf("Port %d tidak tersedia.", port),
				Hint:    "DevLok dapat memakai fallback 8080/8443 untuk HTTP/HTTPS.",
			})
			continue
		}
		_ = ln.Close()
		out = append(out, Check{Name: "port_" + strconv.Itoa(port), Status: StatusOK, Message: fmt.Sprintf("Port %d tersedia.", port)})
	}
	return out
}

func checkManifest(opts Options) Check {
	path := opts.ManifestPath
	if path == "" {
		found, ok := config.Find(opts.WorkDir)
		if !ok {
			return Check{Name: "manifest", Status: StatusOK, Message: "Tidak ada .devlok.yaml di folder ini (opsional)."}
		}
		path = found
	}
	m, err := config.Load(path)
	if err != nil {
		return Check{Name: "manifest", Status: StatusFail, Message: "Manifest tidak valid: " + path, Hint: err.Error()}
	}
	return Check{Name: "manifest", Status: StatusOK, Message: fmt.Sprintf("%s (%s -> https://%s)", path, m.Name, m.BaseDomain())}
}
