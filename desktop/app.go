//go:build windows

package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"devlok/internal/config"
	"devlok/internal/db"
	"devlok/internal/devlokerr"
	"devlok/internal/editor"
	"devlok/internal/importlaragon"
	"devlok/internal/platform"
	"devlok/internal/ports"
	"devlok/internal/projects"
	rt "devlok/internal/runtime"
	"devlok/internal/stack"
	"devlok/internal/supervisor"
	"devlok/internal/version"
)

type App struct {
	ctx context.Context
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type StatusView struct {
	Running        bool          `json:"running"`
	Health         string        `json:"health"`
	Project        string        `json:"project"`
	Path           string        `json:"path"`
	Domain         string        `json:"domain"`
	URL            string        `json:"url"`
	LocalURL       string        `json:"local_url"`
	DBName         string        `json:"db_name"`
	DBUser         string        `json:"db_user"`
	DBPort         int           `json:"db_port"`
	PHPPort        int           `json:"php_port"`
	NginxVersion   string        `json:"nginx_version"`
	PHPVersion     string        `json:"php_version"`
	MySQLVersion   string        `json:"mysql_version"`
	HTTPPort       int           `json:"http_port"`
	HTTPSPort      int           `json:"https_port"`
	HTTPListening  bool          `json:"http_listening"`
	HTTPSListening bool          `json:"https_listening"`
	DatabaseOn     bool          `json:"database_enabled"`
	PHPMyAdminOn   bool          `json:"phpmyadmin_enabled"`
	Services       []ServiceView `json:"services"`
}

type ComponentsView struct {
	Database     bool   `json:"database"`
	PHPMyAdmin   bool   `json:"phpmyadmin"`
	MySQLVersion string `json:"mysql_version"`
}

type ProjectConfigView struct {
	Path       string            `json:"path"`
	Root       string            `json:"root"`
	HTTPS      bool              `json:"https"`
	DBName     string            `json:"db_name"`
	Name       string            `json:"name"`
	Domain     string            `json:"domain"`
	Aliases    []string          `json:"aliases"`
	HTTPPort   int               `json:"http_port"`
	HTTPSPort  int               `json:"https_port"`
	MySQLPort  int               `json:"mysql_port"`
	PHPPort    int               `json:"php_port"`
	Versions   map[string]string `json:"versions"`
	Database   bool              `json:"database"`
	PHPMyAdmin bool              `json:"phpmyadmin"`
}

type ProjectConfigInput struct {
	Domain     string            `json:"domain"`
	Root       string            `json:"root"`
	HTTPS      bool              `json:"https"`
	DBName     string            `json:"db_name"`
	Aliases    []string          `json:"aliases"`
	HTTPPort   int               `json:"http_port"`
	HTTPSPort  int               `json:"https_port"`
	MySQLPort  int               `json:"mysql_port"`
	PHPPort    int               `json:"php_port"`
	Versions   map[string]string `json:"versions"`
	Database   bool              `json:"database"`
	PHPMyAdmin bool              `json:"phpmyadmin"`
}

type ServiceView struct {
	Name      string `json:"name"`
	PID       int    `json:"pid"`
	State     string `json:"state"`
	Restarts  int    `json:"restarts"`
	LastError string `json:"last_error,omitempty"`
}

type SiteView struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
	Root   string `json:"root"`
	PHP    string `json:"php"`
	Path   string `json:"path"`
	Active bool   `json:"active"`
}

type RuntimeView struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

func (a *App) Status() StatusView {
	services, _ := stack.Services("")
	st, err := stack.LoadState("")
	if err != nil {
		return StatusView{Running: false, Services: toServiceViews(services)}
	}
	http := ports.Check(st.HTTPPort)
	https := ports.Check(st.HTTPSPort)
	svcs := toServiceViews(services)
	dbOn, pmaOn := true, true
	nginxVersion, phpVersion, mysqlVersion := "1.30", "8.4", "8.4"
	if m, err := loadManifest(st.ProjectDir); err == nil {
		dbOn = m.ComponentEnabled("mysql")
		pmaOn = dbOn && m.ComponentEnabled("phpmyadmin")
		if m.Runtime["nginx"] != "" {
			nginxVersion = m.Runtime["nginx"]
		}
		if m.Web.PHP != "" {
			phpVersion = m.Web.PHP
		}
		if m.Services["mysql"] != "" {
			mysqlVersion = m.Services["mysql"]
		}
	}
	if lock, err := config.LoadLock(filepath.Join(st.ProjectDir, config.LockFileName)); err == nil {
		if lock.Web.Version != "" {
			nginxVersion = lock.Web.Version
		}
		if lock.Runtime["php"].Version != "" {
			phpVersion = lock.Runtime["php"].Version
		}
		if lock.Services["mysql"].Version != "" {
			mysqlVersion = lock.Services["mysql"].Version
		}
	}
	health := "stopped"
	anyRunning := false
	have := map[string]bool{}
	for _, s := range services {
		if s.State != "running" {
			continue
		}
		anyRunning = true
		if strings.HasPrefix(s.Name, "mysql-") {
			have["mysql"] = true
		} else {
			have[s.Name] = true
		}
	}
	if anyRunning {
		health = "partial"
		if have["nginx"] && have["php"] && (!dbOn || have["mysql"]) {
			health = "running"
		}
	}
	if !dbOn {
		filtered := svcs[:0]
		for _, s := range svcs {
			if strings.HasPrefix(s.Name, "mysql-") {
				continue
			}
			filtered = append(filtered, s)
		}
		svcs = filtered
	}
	filtered := svcs[:0]
	for _, s := range svcs {
		if s.State == "stopped" {
			continue
		}
		filtered = append(filtered, s)
	}
	svcs = filtered
	return StatusView{
		Running:        health != "stopped",
		Health:         health,
		Project:        st.Name,
		Path:           st.ProjectDir,
		Domain:         st.Domain,
		URL:            st.URL,
		LocalURL:       st.LocalURL,
		DBName:         st.DBName,
		DBUser:         st.DBUser,
		DBPort:         st.DBPort,
		PHPPort:        st.PHPPort,
		NginxVersion:   nginxVersion,
		PHPVersion:     phpVersion,
		MySQLVersion:   mysqlVersion,
		HTTPPort:       st.HTTPPort,
		HTTPSPort:      st.HTTPSPort,
		HTTPListening:  !http.Available,
		HTTPSListening: !https.Available,
		DatabaseOn:     dbOn,
		PHPMyAdminOn:   pmaOn,
		Services:       svcs,
	}
}

func (a *App) Up(path string) (stack.Result, error) {
	if path == "" {
		return stack.Result{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Pilih folder project terlebih dahulu.", "")
	}
	saveLastProject(path)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	return stack.Up(ctx, stack.Options{ProjectDir: path})
}

func (a *App) Down() error {
	return stack.Down("")
}

func (a *App) Sites() ([]SiteView, error) {
	ws, err := stack.WorkspaceDir()
	if err != nil {
		return nil, err
	}
	var dirs []string
	if entries, err := os.ReadDir(ws); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(ws, e.Name()))
			}
		}
	}
	if state, err := platform.Current().StateDir(); err == nil {
		if extra, err := projects.List(state); err == nil {
			dirs = append(dirs, extra...)
		}
	}
	activePath := ""
	if st, err := stack.LoadState(""); err == nil {
		activePath = st.ProjectDir
	}
	seen := map[string]bool{}
	out := make([]SiteView, 0, len(dirs))
	for _, dir := range dirs {
		key := strings.ToLower(filepath.Clean(dir))
		if seen[key] {
			continue
		}
		seen[key] = true
		m, err := config.Load(filepath.Join(dir, config.FileName))
		if err != nil {
			continue
		}
		out = append(out, SiteView{
			Name:   m.Name,
			Domain: m.BaseDomain(),
			Root:   m.Web.Root,
			PHP:    m.Web.PHP,
			Path:   dir,
			Active: strings.EqualFold(filepath.Clean(dir), filepath.Clean(activePath)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (a *App) ScanLaragon(path string) ([]importlaragon.Site, error) {
	return importlaragon.Plan(path)
}

func (a *App) ImportLaragon(path string, names []string, adopt bool) ([]importlaragon.Result, error) {
	sites, err := importlaragon.Plan(path)
	if err != nil {
		return nil, err
	}
	if len(names) > 0 {
		selected := map[string]bool{}
		for _, n := range names {
			selected[n] = true
		}
		filtered := sites[:0]
		for _, s := range sites {
			if selected[s.Name] {
				filtered = append(filtered, s)
			}
		}
		sites = filtered
	}
	results, err := importlaragon.Import(sites, false)
	if err != nil {
		return nil, err
	}
	if adopt {
		if state, err := platform.Current().StateDir(); err == nil {
			for _, r := range results {
				if r.Action == "create" || r.Action == "skip" {
					_ = projects.Add(state, r.Site.ProjectDir)
				}
			}
		}
	}
	return results, nil
}
func (a *App) CreateSite(name, php string, withDatabase bool) (stack.Result, error) {
	if name == "" {
		return stack.Result{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Nama site wajib diisi.", "")
	}
	ws, err := stack.WorkspaceDir()
	if err != nil {
		return stack.Result{}, err
	}
	name = config.SanitizeName(name)
	dir := filepath.Join(ws, name)
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		return stack.Result{}, err
	}
	index := filepath.Join(dir, "public", "index.php")
	if _, err := os.Stat(index); os.IsNotExist(err) {
		content := "<?php\necho '<h1>" + name + "</h1><p>DevLok site aktif.</p>';\n"
		if err := os.WriteFile(index, []byte(content), 0o644); err != nil {
			return stack.Result{}, err
		}
	}
	if _, err := os.Stat(filepath.Join(dir, config.FileName)); os.IsNotExist(err) {
		if _, _, err := config.Init(config.InitOptions{Dir: dir, Name: name, PHP: php, NoDatabase: !withDatabase}); err != nil {
			return stack.Result{}, err
		}
	}
	if !withDatabase {
		m, err := loadManifest(dir)
		if err == nil {
			m.Components = map[string]bool{"mysql": false, "phpmyadmin": false}
			if err := saveManifest(dir, m); err != nil {
				return stack.Result{}, err
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	saveLastProject(dir)
	return stack.Up(ctx, stack.Options{ProjectDir: dir})
}

func (a *App) Components(path string) (ComponentsView, error) {
	m, err := loadManifest(path)
	if err != nil {
		return ComponentsView{}, err
	}
	dbOn := m.ComponentEnabled("mysql")
	version := m.Services["mysql"]
	if version == "" {
		version = "8.4"
	}
	return ComponentsView{Database: dbOn, PHPMyAdmin: dbOn && m.ComponentEnabled("phpmyadmin"), MySQLVersion: version}, nil
}

func (a *App) MySQLVersions() ([]string, error) {
	return a.ServiceVersions("mysql")
}

func (a *App) ServiceVersions(service string) ([]string, error) {
	mgr, err := a.runtimeManager()
	if err != nil {
		return nil, err
	}
	key := mgr.OS + "/" + mgr.Arch
	var out []string
	for _, reg := range mgr.Registries {
		if reg.Name != service {
			continue
		}
		for _, v := range reg.Versions {
			if _, ok := v.Artifacts[key]; ok {
				out = append(out, v.Version)
			}
		}
	}
	if len(out) == 0 {
		return nil, devlokerr.New(devlokerr.CodeDependencyMissing, "Tidak ada versi "+service+" di registry.", "")
	}
	return out, nil
}

func (a *App) SetMySQLVersion(path, version string) error {
	return a.SetServiceVersion(path, "mysql", version)
}

func (a *App) SetServiceVersion(path, service, version string) error {
	saveLastProject(path)
	m, err := loadManifest(path)
	if err != nil {
		return err
	}
	mgr, err := a.runtimeManager()
	if err != nil {
		return err
	}
	res, err := mgr.Resolve(service, version)
	if err != nil {
		return err
	}
	exact := res.Version.Version
	switch service {
	case "mysql":
		if m.Services == nil {
			m.Services = map[string]string{}
		}
		m.Services["mysql"] = exact
	case "nginx":
		if m.Runtime == nil {
			m.Runtime = map[string]string{}
		}
		m.Runtime["nginx"] = exact
	case "php":
		if m.Runtime == nil {
			m.Runtime = map[string]string{}
		}
		m.Runtime["php"] = exact
		m.Web.PHP = exact
	case "node":
		if m.Runtime == nil {
			m.Runtime = map[string]string{}
		}
		m.Runtime["node"] = exact
	default:
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Service tidak dikenal: "+service, "")
	}
	if err := saveManifest(path, m); err != nil {
		return err
	}
	lockPath := filepath.Join(path, config.LockFileName)
	if lock, err := config.LoadLock(lockPath); err == nil {
		lr := config.LockedRuntime{Version: exact, SHA256: res.Artifact.SHA256, URL: res.Artifact.URL}
		switch service {
		case "mysql":
			if lock.Services == nil {
				lock.Services = map[string]config.LockedRuntime{}
			}
			lock.Services["mysql"] = lr
		case "nginx":
			lock.Web.Version = exact
			lock.Web.SHA256 = res.Artifact.SHA256
			lock.Web.URL = res.Artifact.URL
		case "php", "node":
			if lock.Runtime == nil {
				lock.Runtime = map[string]config.LockedRuntime{}
			}
			lock.Runtime[service] = lr
		}
		if err := config.SaveLock(lockPath, lock); err != nil {
			return err
		}
	}
	return nil
}
func (a *App) ProjectConfig(path string) (ProjectConfigView, error) {
	m, err := loadManifest(path)
	if err != nil {
		return ProjectConfigView{}, err
	}
	v := ProjectConfigView{
		Path:       path,
		Root:       m.Web.Root,
		HTTPS:      m.HTTPSEnabled(),
		DBName:     m.DB.Name,
		Name:       m.Name,
		Domain:     m.BaseDomain(),
		Aliases:    append([]string{}, m.Web.Aliases...),
		HTTPPort:   m.Ports.HTTP,
		HTTPSPort:  m.Ports.HTTPS,
		MySQLPort:  m.Ports.MySQL,
		PHPPort:    m.Ports.PHP,
		Database:   m.ComponentEnabled("mysql"),
		PHPMyAdmin: m.ComponentEnabled("phpmyadmin"),
		Versions: map[string]string{
			"nginx": m.Runtime["nginx"],
			"php":   m.Web.PHP,
			"mysql": m.Services["mysql"],
			"node":  m.Runtime["node"],
		},
	}
	if v.Versions["nginx"] == "" {
		v.Versions["nginx"] = "1.30"
	}
	if v.Versions["mysql"] == "" {
		v.Versions["mysql"] = "8.4"
	}
	return v, nil
}

func (a *App) SaveProjectConfig(path string, in ProjectConfigInput) error {
	m, err := loadManifest(path)
	if err != nil {
		return err
	}
	mgr, err := a.runtimeManager()
	if err != nil {
		return err
	}
	resolved := map[string]rt.Resolved{}
	for _, service := range []string{"nginx", "php", "mysql", "node"} {
		version := strings.TrimSpace(in.Versions[service])
		if version == "" {
			if service == "nginx" || service == "node" {
				delete(m.Runtime, service)
			}
			continue
		}
		res, err := mgr.Resolve(service, version)
		if err != nil {
			return err
		}
		resolved[service] = res
		exact := res.Version.Version
		switch service {
		case "nginx":
			if m.Runtime == nil {
				m.Runtime = map[string]string{}
			}
			m.Runtime["nginx"] = exact
		case "php":
			if m.Runtime == nil {
				m.Runtime = map[string]string{}
			}
			m.Runtime["php"] = exact
			m.Web.PHP = exact
		case "mysql":
			if m.Services == nil {
				m.Services = map[string]string{}
			}
			m.Services["mysql"] = exact
		case "node":
			if m.Runtime == nil {
				m.Runtime = map[string]string{}
			}
			m.Runtime["node"] = exact
		}
	}
	m.Web.Root = strings.TrimSpace(in.Root)
	https := in.HTTPS
	m.Web.HTTPS = &https
	if in.DBName != "" {
		m.DB.Name = strings.TrimSpace(in.DBName)
	}
	m.Web.Domain = strings.TrimSpace(in.Domain)
	aliases := make([]string, 0, len(in.Aliases))
	for _, alias := range in.Aliases {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			aliases = append(aliases, alias)
		}
	}
	m.Web.Aliases = aliases
	m.Ports = config.PortsConfig{HTTP: in.HTTPPort, HTTPS: in.HTTPSPort, MySQL: in.MySQLPort, PHP: in.PHPPort}
	if !in.Database {
		in.PHPMyAdmin = false
	}
	m.Components = map[string]bool{"mysql": in.Database, "phpmyadmin": in.PHPMyAdmin}
	if err := config.Validate(m, path); err != nil {
		return err
	}
	rootAbs := filepath.Join(path, m.Web.Root)
	if info, err := os.Stat(rootAbs); err != nil || !info.IsDir() {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Document root tidak ditemukan: "+rootAbs, "Perbaiki isian root.")
	}
	if err := saveManifest(path, m); err != nil {
		return err
	}
	lockPath := filepath.Join(path, config.LockFileName)
	if lock, err := config.LoadLock(lockPath); err == nil {
		lock.Domain = m.BaseDomain()
		for service, res := range resolved {
			lr := config.LockedRuntime{Version: res.Version.Version, SHA256: res.Artifact.SHA256, URL: res.Artifact.URL}
			switch service {
			case "nginx":
				lock.Web.Version = lr.Version
				lock.Web.SHA256 = lr.SHA256
				lock.Web.URL = lr.URL
			case "mysql":
				if lock.Services == nil {
					lock.Services = map[string]config.LockedRuntime{}
				}
				lock.Services["mysql"] = lr
			case "php", "node":
				if lock.Runtime == nil {
					lock.Runtime = map[string]config.LockedRuntime{}
				}
				lock.Runtime[service] = lr
			}
		}
		if err := config.SaveLock(lockPath, lock); err != nil {
			return err
		}
	}
	saveLastProject(path)
	return nil
}

type PortCheckView struct {
	Port      int    `json:"port"`
	Available bool   `json:"available"`
	PID       int    `json:"pid"`
	Process   string `json:"process"`
	Path      string `json:"path"`
}

func (a *App) CheckPorts(list []int) []PortCheckView {
	out := make([]PortCheckView, 0, len(list))
	for _, p := range list {
		st := ports.Check(p)
		out = append(out, PortCheckView{Port: p, Available: st.Available, PID: st.PID, Process: st.Process, Path: st.Path})
	}
	return out
}
func (a *App) runtimeManager() (*rt.Manager, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, err
	}
	return rt.NewManager(rt.Options{
		Root:  filepath.Join(state, "runtimes"),
		Cache: filepath.Join(state, "cache", "downloads"),
	})
}

func (a *App) SetComponents(path string, database, phpmyadmin bool) error {
	m, err := loadManifest(path)
	if err != nil {
		return err
	}
	if !database {
		phpmyadmin = false
	}
	saveLastProject(path)
	m.Components = map[string]bool{"mysql": database, "phpmyadmin": phpmyadmin}
	if err := config.Validate(m, path); err != nil {
		return err
	}
	return saveManifest(path, m)
}

func loadManifest(dir string) (*config.Manifest, error) {
	return config.Load(filepath.Join(dir, config.FileName))
}

func saveManifest(dir string, m *config.Manifest) error {
	data, err := config.Marshal(m)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, config.FileName), data, 0o644); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Manifest tidak dapat ditulis.", "")
	}
	return nil
}

func (a *App) OpenURL(url string) error {
	return platform.Current().OpenBrowser(url)
}

func (a *App) OpenFolder(path string) error {
	if path == "" {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Path kosong.", "")
	}
	return platform.Current().OpenFolder(path)
}

func (a *App) OpenEditor(path string) error {
	return editor.Open(path)
}

func (a *App) OpenTerminal(path string) error {
	if path == "" {
		path, _ = os.Getwd()
	}
	return platform.Current().OpenTerminal(path, nil)
}

func (a *App) RuntimeList() ([]RuntimeView, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, err
	}
	mgr, err := rt.NewManager(rt.Options{
		Root:  filepath.Join(state, "runtimes"),
		Cache: filepath.Join(state, "cache", "downloads"),
	})
	if err != nil {
		return nil, err
	}
	installed, err := mgr.ListInstalled()
	if err != nil {
		return nil, err
	}
	out := make([]RuntimeView, 0, len(installed))
	for _, in := range installed {
		out = append(out, RuntimeView{Name: in.Name, Version: in.Version, Path: in.Path})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Version > out[j].Version
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (a *App) Logs(service string, lines int) (string, error) {
	if lines <= 0 {
		lines = 200
	}
	state, err := platform.Current().StateDir()
	if err != nil {
		return "", err
	}
	var path string
	switch strings.ToLower(service) {
	case "nginx":
		path = filepath.Join(state, "logs", "nginx", "error.log")
	case "php":
		path = filepath.Join(state, "logs", "php-error.log")
	case "mysql":
		matches, _ := filepath.Glob(filepath.Join(state, "logs", "mysql", "*-error.log"))
		if len(matches) > 0 {
			sort.Strings(matches)
			path = matches[len(matches)-1]
		}
	default:
		matches, _ := filepath.Glob(filepath.Join(state, "logs", "*.jsonl"))
		if len(matches) > 0 {
			sort.Strings(matches)
			path = matches[len(matches)-1]
		}
	}
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	text := string(data)
	parts := strings.Split(text, "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n"), nil
}

func (a *App) DBInfo(version string) (map[string]any, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, err
	}
	mgr, err := db.New(db.Options{StateDir: state, Version: version})
	if err != nil {
		return nil, err
	}
	dbs, err := mgr.Databases()
	if err != nil {
		return nil, err
	}
	return map[string]any{"port": mgr.Port, "databases": dbs}, nil
}

func toServiceViews(services []supervisor.Status) []ServiceView {
	out := make([]ServiceView, 0, len(services))
	for _, s := range services {
		out = append(out, ServiceView{Name: s.Name, PID: s.PID, State: string(s.State), Restarts: s.Restarts, LastError: s.LastError})
	}
	return out
}

func (a *App) Version() string {
	return version.Current().Version
}
