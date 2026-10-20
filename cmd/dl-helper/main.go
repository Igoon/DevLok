package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"devlok/internal/config"
	"devlok/internal/hosts"
	"devlok/internal/platform"
)

var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]{0,251}[a-z0-9])?$`)

func main() {
	if len(os.Args) < 2 {
		fail("operasi tidak diberikan")
	}
	logHelper("mulai " + os.Args[1])
	hm := hosts.New(platform.Current().HostsPath())
	hm.Flush = platform.Current().FlushDNS
	switch os.Args[1] {
	case "hosts-sync":
		if len(os.Args) != 3 {
			fail("payload tidak diberikan")
		}
		domains, err := readPayload(os.Args[2])
		if err != nil {
			fail(err.Error())
		}
		if _, err := hm.Apply(domains); err != nil {
			fail("hosts: " + err.Error())
		}
	case "hosts-remove":
		if _, err := hm.Remove(); err != nil {
			fail("hosts: " + err.Error())
		}
	default:
		fail("operasi tidak dikenal: " + os.Args[1])
	}
	logHelper("selesai " + os.Args[1])
}

func logHelper(msg string) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return
	}
	dir := filepath.Join(state, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "hosts.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), msg)
}

func readPayload(path string) ([]string, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	allowed := filepath.Join(state, "state") + string(os.PathSeparator)
	if !strings.HasPrefix(strings.ToLower(abs), strings.ToLower(allowed)) {
		return nil, fmt.Errorf("payload di luar state dir")
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if len(data) > 64<<10 {
		return nil, fmt.Errorf("payload terlalu besar")
	}
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		d := strings.ToLower(strings.TrimSpace(line))
		if d == "" {
			continue
		}
		if !domainRe.MatchString(d) || !config.ValidDomain(d) {
			return nil, fmt.Errorf("domain tidak valid")
		}
		if seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
		if len(out) > 200 {
			return nil, fmt.Errorf("terlalu banyak domain")
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("daftar domain kosong")
	}
	return out, nil
}

func fail(msg string) {
	logHelper("gagal: " + msg)
	fmt.Fprintln(os.Stderr, "dl-helper:", msg)
	os.Exit(1)
}
