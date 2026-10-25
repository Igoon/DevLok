//go:build windows || darwin

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"devlok/internal/brew"
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
	"devlok/internal/update"
	"devlok/internal/version"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx             context.Context
	updateMu        sync.Mutex
	updateCancel    context.CancelFunc
	autoMu          sync.Mutex
	autoLastCheck   time.Time
	registryMu      sync.Mutex
	registryFetchMu sync.Mutex
	registries      []rt.Registry
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.refreshRemoteRegistries(10*time.Second, false)
	go a.autoUpdateLoop(ctx)
}

type StatusView struct {
	Running        bool          `json:"running"`
	Health         string        `json:"health"`
	Platform       string        `json:"platform"`
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
	MailpitURL     string        `json:"mailpit_url,omitempty"`
	Services       []ServiceView `json:"services"`
}

type ComponentsView struct {
	Database     bool   `json:"database"`
	PHPMyAdmin   bool   `json:"phpmyadmin"`
	Mailpit      bool   `json:"mailpit"`
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
	Mailpit    bool              `json:"mailpit"`
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
	Mailpit    bool              `json:"mailpit"`
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
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Path    string   `json:"path"`
	Bytes   int64    `json:"bytes"`
	InUse   bool     `json:"in_use"`
	UsedBy  []string `json:"used_by,omitempty"`
}

type RuntimeCacheView struct {
	Bytes int64 `json:"bytes"`
	Files int   `json:"files"`
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
		Platform:       platform.Current().Name(),
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
		MailpitURL:     st.MailpitURL,
		Services:       svcs,
	}
}

func (a *App) PHPMyAdminURL() string {
	return activePHPMyAdminURL()
}

func (a *App) stackOptions(path string) stack.Options {
	return stack.Options{ProjectDir: path, Registries: a.currentRegistries()}
}

func (a *App) Up(path string) (stack.Result, error) {
	if path == "" {
		return stack.Result{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Pilih folder project terlebih dahulu.", "")
	}
	saveLastProject(path)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	return stack.Up(ctx, a.stackOptions(path))
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
	return stack.Up(ctx, a.stackOptions(dir))
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
	return ComponentsView{Database: dbOn, PHPMyAdmin: dbOn && m.ComponentEnabled("phpmyadmin"), Mailpit: m.MailpitEnabled(), MySQLVersion: version}, nil
}

func (a *App) MySQLVersions() ([]string, error) {
	return a.ServiceVersions("mysql")
}

func (a *App) ServiceVersions(service string) ([]string, error) {
	if platform.Current().Name() == "darwin" {
		return brewServiceVersions(service), nil
	}
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

func brewServiceVersions(service string) []string {
	switch service {
	case "nginx", "php", "mysql", "node":
	default:
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		out = append(out, v)
	}
	add(brew.Version(ctx, service))
	for _, v := range brew.InstalledVersioned(ctx, service) {
		add(v)
	}
	return out
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
	exact := version
	artifactSHA := ""
	artifactURL := ""
	if platform.Current().Name() != "darwin" {
		mgr, err := a.runtimeManager()
		if err != nil {
			return err
		}
		res, err := mgr.Resolve(service, version)
		if err != nil {
			return err
		}
		exact = res.Version.Version
		artifactSHA = res.Artifact.SHA256
		artifactURL = res.Artifact.URL
	}
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
		lr := config.LockedRuntime{Version: exact, SHA256: artifactSHA, URL: artifactURL}
		switch service {
		case "mysql":
			if lock.Services == nil {
				lock.Services = map[string]config.LockedRuntime{}
			}
			lock.Services["mysql"] = lr
		case "nginx":
			lock.Web.Version = exact
			lock.Web.SHA256 = artifactSHA
			lock.Web.URL = artifactURL
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
		Mailpit:    m.MailpitEnabled(),
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
	m.Ports = config.PortsConfig{
		HTTP:        in.HTTPPort,
		HTTPS:       in.HTTPSPort,
		MySQL:       in.MySQLPort,
		PHP:         in.PHPPort,
		MailpitSMTP: m.Ports.MailpitSMTP,
		MailpitHTTP: m.Ports.MailpitHTTP,
	}
	if !in.Database {
		in.PHPMyAdmin = false
	}
	m.Components = map[string]bool{"mysql": in.Database, "phpmyadmin": in.PHPMyAdmin, "mailpit": in.Mailpit}
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
	return a.runtimeManagerWithProgress(nil)
}

func (a *App) runtimeManagerWithProgress(progress func(rt.Progress)) (*rt.Manager, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, err
	}
	return rt.NewManager(rt.Options{
		Root:       filepath.Join(state, "runtimes"),
		Cache:      filepath.Join(state, "cache", "downloads"),
		Progress:   progress,
		Registries: a.currentRegistries(),
	})
}

func (a *App) currentRegistries() []rt.Registry {
	a.registryMu.Lock()
	defer a.registryMu.Unlock()
	if len(a.registries) == 0 {
		return nil
	}
	return append([]rt.Registry(nil), a.registries...)
}

func (a *App) refreshRemoteRegistries(timeout time.Duration, force bool) {
	if !a.registryFetchMu.TryLock() {
		return
	}
	defer a.registryFetchMu.Unlock()
	state, err := platform.Current().StateDir()
	if err != nil {
		return
	}
	cfg, err := update.LoadConfig(state)
	if err != nil {
		trayLog("registry config: " + err.Error())
		return
	}
	if cfg == nil {
		cfg = update.DefaultConfig()
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ttl := update.DefaultRegistryTTL
	if force {
		ttl = 0
	}
	remote, err := update.RemoteRegistries(ctx, cfg, update.RegistryCachePath(state), ttl)
	if err != nil {
		trayLog("registry remote: " + err.Error())
		return
	}
	if len(remote) == 0 {
		return
	}
	embedded, err := rt.LoadEmbedded()
	if err != nil {
		embedded = nil
	}
	merged := mergeRegistries(embedded, remote)
	a.registryMu.Lock()
	a.registries = merged
	a.registryMu.Unlock()
}

func mergeRegistries(base, override []rt.Registry) []rt.Registry {
	byName := map[string]rt.Registry{}
	order := make([]string, 0, len(base)+len(override))
	for _, reg := range base {
		if _, ok := byName[reg.Name]; !ok {
			order = append(order, reg.Name)
		}
		byName[reg.Name] = reg
	}
	for _, reg := range override {
		if _, ok := byName[reg.Name]; !ok {
			order = append(order, reg.Name)
		}
		byName[reg.Name] = reg
	}
	out := make([]rt.Registry, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out
}

func (a *App) SetComponents(path string, database, phpmyadmin, mailpit bool) error {
	m, err := loadManifest(path)
	if err != nil {
		return err
	}
	if !database {
		phpmyadmin = false
	}
	saveLastProject(path)
	m.Components = map[string]bool{"mysql": database, "phpmyadmin": phpmyadmin, "mailpit": mailpit}
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
	uses := a.runtimeUses()
	out := make([]RuntimeView, 0, len(installed))
	for _, in := range installed {
		var usedBy []string
		for _, use := range uses {
			if use.Name == in.Name && versionMatches(in.Version, use.Version) && !containsString(usedBy, use.Project) {
				usedBy = append(usedBy, use.Project)
			}
		}
		out = append(out, RuntimeView{
			Name:    in.Name,
			Version: in.Version,
			Path:    in.Path,
			Bytes:   dirSize(in.Path),
			InUse:   len(usedBy) > 0,
			UsedBy:  usedBy,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Version > out[j].Version
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (a *App) Confirm(title, message string) (bool, error) {
	if title == "" {
		title = "DevLok"
	}
	res, err := wruntime.MessageDialog(a.ctx, wruntime.MessageDialogOptions{
		Type:    wruntime.QuestionDialog,
		Title:   title,
		Message: message,
		Buttons: []string{"Ya", "Batal"},
	})
	if err != nil {
		return false, err
	}
	return res == "Ya", nil
}

func (a *App) RemoveRuntime(name, version string) (string, error) {
	usedBy := a.projectsUsingRuntime(name, version)
	if len(usedBy) > 0 {
		return "", devlokerr.New(devlokerr.CodeInvalidArgument,
			fmt.Sprintf("%s %s masih dipakai project %s.", name, version, strings.Join(usedBy, ", ")),
			"Ubah versi project terkait dulu, lalu hapus runtime ini.")
	}
	mgr, err := a.runtimeManager()
	if err != nil {
		return "", err
	}
	if err := mgr.Remove(name, version); err != nil {
		return "", err
	}
	return fmt.Sprintf("Runtime %s %s dihapus.", name, version), nil
}

func (a *App) RuntimeCache() (RuntimeCacheView, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return RuntimeCacheView{}, err
	}
	dir := filepath.Join(state, "cache", "downloads")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return RuntimeCacheView{}, nil
		}
		return RuntimeCacheView{}, err
	}
	out := RuntimeCacheView{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out.Files++
		out.Bytes += info.Size()
	}
	return out, nil
}

func (a *App) CleanupRuntimeCache() (string, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(state, "cache", "downloads")
	info, _ := a.RuntimeCache()
	if err := os.RemoveAll(dir); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Cache runtime tidak dapat dibersihkan.", "")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return fmt.Sprintf("Cache dibersihkan: %d file, %s.", info.Files, formatBytes(info.Bytes)), nil
}

func (a *App) CleanupUnusedRuntimes() (string, error) {
	mgr, err := a.runtimeManager()
	if err != nil {
		return "", err
	}
	installed, err := mgr.ListInstalled()
	if err != nil {
		return "", err
	}
	var removed []string
	var bytes int64
	for _, in := range installed {
		if in.Name != "php" && in.Name != "nginx" && in.Name != "mysql" && in.Name != "node" {
			continue
		}
		if len(a.projectsUsingRuntime(in.Name, in.Version)) > 0 {
			continue
		}
		superseded := false
		for _, other := range installed {
			if other.Name == in.Name && update.SameSeries(other.Version, in.Version) && update.Compare(other.Version, in.Version) > 0 {
				superseded = true
				break
			}
		}
		if !superseded {
			continue
		}
		size := dirSize(in.Path)
		if err := mgr.Remove(in.Name, in.Version); err != nil {
			return "", err
		}
		removed = append(removed, in.Name+" "+in.Version)
		bytes += size
	}
	if len(removed) == 0 {
		return "Tidak ada versi lama yang tidak terpakai.", nil
	}
	return fmt.Sprintf("%d versi lama dihapus (%s): %s.", len(removed), formatBytes(bytes), strings.Join(removed, ", ")), nil
}

type runtimeUse struct {
	Name    string
	Version string
	Project string
}

func (a *App) runtimeUses() []runtimeUse {
	var uses []runtimeUse
	for _, dir := range a.projectDirs() {
		m, err := config.Load(filepath.Join(dir, config.FileName))
		if err != nil {
			continue
		}
		for _, service := range []string{"php", "nginx", "mysql", "node"} {
			version := projectServiceVersion(m, service)
			if version != "" {
				uses = append(uses, runtimeUse{Name: service, Version: version, Project: dir})
			}
		}
	}
	return uses
}

func (a *App) projectsUsingRuntime(name, version string) []string {
	var out []string
	for _, use := range a.runtimeUses() {
		if use.Name == name && versionMatches(version, use.Version) && !containsString(out, use.Project) {
			out = append(out, use.Project)
		}
	}
	return out
}

func (a *App) projectDirs() []string {
	var dirs []string
	if ws, err := stack.WorkspaceDir(); err == nil {
		if entries, err := os.ReadDir(ws); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					dirs = append(dirs, filepath.Join(ws, entry.Name()))
				}
			}
		}
	}
	if state, err := platform.Current().StateDir(); err == nil {
		if extra, err := projects.List(state); err == nil {
			dirs = append(dirs, extra...)
		}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		key := strings.ToLower(filepath.Clean(dir))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, dir)
	}
	return out
}

func projectServiceVersion(m *config.Manifest, service string) string {
	switch service {
	case "php":
		if m.Web.PHP != "" {
			return m.Web.PHP
		}
		return m.Runtime["php"]
	case "nginx":
		return m.Runtime["nginx"]
	case "mysql":
		return m.Services["mysql"]
	case "node":
		return m.Runtime["node"]
	}
	return ""
}

func isServiceRuntime(name string) bool {
	switch name {
	case "nginx", "php", "mysql", "node":
		return true
	}
	return false
}

func versionMatches(installed, used string) bool {
	if installed == "" || used == "" {
		return false
	}
	return installed == used || strings.HasPrefix(installed, used+".") || strings.HasPrefix(used, installed+".")
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, err := entry.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGT"[exp])
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

const updateProgressEvent = "update:progress"

type UpdateProgressView struct {
	Kind       string `json:"kind"`
	Phase      string `json:"phase"`
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
	Percent    int    `json:"percent"`
	Message    string `json:"message,omitempty"`
}

func (a *App) emitUpdateProgress(p UpdateProgressView) {
	if p.Total > 0 {
		p.Percent = int(p.Downloaded * 100 / p.Total)
	}
	if p.Phase == "done" {
		p.Percent = 100
	}
	if p.Percent < 0 {
		p.Percent = 0
	}
	if p.Percent > 100 {
		p.Percent = 100
	}
	if a.ctx == nil {
		return
	}
	wruntime.EventsEmit(a.ctx, updateProgressEvent, p)
}

func (a *App) emitUpdateError(kind, name, version string, err error) {
	a.emitUpdateProgress(UpdateProgressView{
		Kind:    kind,
		Phase:   "error",
		Name:    name,
		Version: version,
		Message: err.Error(),
	})
}

func (a *App) CancelUpdate() {
	a.updateMu.Lock()
	cancel := a.updateCancel
	a.updateMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *App) startUpdateContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	a.updateMu.Lock()
	if a.updateCancel != nil {
		a.updateCancel()
	}
	a.updateCancel = cancel
	a.updateMu.Unlock()
	return ctx, cancel
}

func (a *App) finishUpdate(cancel context.CancelFunc) {
	cancel()
	a.updateMu.Lock()
	a.updateCancel = nil
	a.updateMu.Unlock()
}

func (a *App) UpdatePreferences() (update.Preferences, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return update.Preferences{}, err
	}
	return update.LoadPreferences(state)
}

func (a *App) SaveUpdatePreferences(prefs update.Preferences) error {
	state, err := platform.Current().StateDir()
	if err != nil {
		return err
	}
	if err := update.SavePreferences(state, prefs); err != nil {
		return err
	}
	a.autoMu.Lock()
	a.autoLastCheck = time.Time{}
	a.autoMu.Unlock()
	return nil
}

func (a *App) autoUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.autoUpdateTick()
		}
	}
}

func (a *App) autoUpdateTick() {
	state, err := platform.Current().StateDir()
	if err != nil {
		return
	}
	a.refreshRemoteRegistries(10*time.Second, false)
	prefs, err := update.LoadPreferences(state)
	if err != nil {
		trayLog("auto update config: " + err.Error())
		return
	}
	if !prefs.AutoUpdateApp && !prefs.AutoUpdateRuntime {
		return
	}
	interval := time.Duration(prefs.IntervalMinutes) * time.Minute
	a.autoMu.Lock()
	due := a.autoLastCheck.IsZero() || time.Since(a.autoLastCheck) >= interval
	if due {
		a.autoLastCheck = time.Now()
	}
	a.autoMu.Unlock()
	if !due {
		return
	}
	if a.updateInProgress() {
		trayLog("auto update dilewati: update lain sedang berjalan")
		return
	}
	if prefs.AutoUpdateRuntime {
		message, err := a.updateRuntimes(prefs.AutoUpdateRuntimeSeries)
		if err != nil {
			trayLog("auto runtime update: " + err.Error())
		} else if message != "" {
			trayLog("auto runtime update: " + message)
		}
	}
	if !prefs.AutoUpdateApp {
		return
	}
	appUpdate, err := a.checkAppUpdate(state)
	if err != nil {
		trayLog("auto app update check: " + err.Error())
		return
	}
	if appUpdate == nil {
		return
	}
	trayLog("auto app update: " + appUpdate.Latest)
	if err := a.ApplyAppUpdate(); err != nil {
		trayLog("auto app update: " + err.Error())
	}
}

func (a *App) updateInProgress() bool {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	return a.updateCancel != nil
}

type UpdateInfoView struct {
	Current     string                 `json:"current"`
	App         *update.AppUpdate      `json:"app_update,omitempty"`
	Runtimes    []update.RuntimeUpdate `json:"runtimes"`
	RemoteError string                 `json:"remote_error,omitempty"`
}

func (a *App) UpdateInfo() (UpdateInfoView, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return UpdateInfoView{}, err
	}
	a.refreshRemoteRegistries(3*time.Second, true)
	mgr, err := a.runtimeManager()
	if err != nil {
		return UpdateInfoView{}, err
	}
	runtimes, err := update.Check(mgr)
	if err != nil {
		return UpdateInfoView{}, err
	}
	if runtimes == nil {
		runtimes = []update.RuntimeUpdate{}
	}
	out := UpdateInfoView{Current: a.Version(), Runtimes: runtimes}
	appUpdate, err := a.checkAppUpdate(state)
	if err != nil {
		out.RemoteError = err.Error()
	} else {
		out.App = appUpdate
	}
	return out, nil
}

func (a *App) InstallRuntimeUpdate(name, version string) (string, error) {
	if platform.Current().Name() == "darwin" {
		return a.installBrewRuntime(name, version)
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "start", Name: name, Version: version})
	mgr, err := a.runtimeManagerWithProgress(func(p rt.Progress) {
		a.emitUpdateProgress(UpdateProgressView{
			Kind:       "runtime",
			Phase:      p.Phase,
			Name:       p.Name,
			Version:    p.Version,
			Downloaded: p.Downloaded,
			Total:      p.Total,
		})
	})
	if err != nil {
		a.emitUpdateError("runtime", name, version, err)
		return "", err
	}
	ctx, cancel := a.startUpdateContext(30 * time.Minute)
	defer a.finishUpdate(cancel)
	inst, _, err := mgr.Install(ctx, name, version)
	if err != nil {
		a.emitUpdateError("runtime", name, version, err)
		return "", err
	}
	project := a.trayProjectPath()
	service := isServiceRuntime(name)
	message := "Runtime " + inst.Name + " " + inst.Version + " terpasang."
	if service {
		if project != "" {
			a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "configure", Name: inst.Name, Version: inst.Version})
			if err := a.SetServiceVersion(project, name, inst.Version); err != nil {
				a.emitUpdateError("runtime", name, inst.Version, err)
				return "", err
			}
			message += " dan project diperbarui."
		} else {
			message += " Belum ada site aktif untuk diperbarui."
		}
		if name == "mysql" {
			message += " Data MySQL tidak dimigrasikan otomatis; restore dump lama bila diperlukan."
		}
	}
	restart := (service && name != "node") || name == "phpmyadmin" || name == "adminer"
	if project != "" && restart {
		if st, err := stack.LoadState(""); err == nil && sameTrayPath(st.ProjectDir, project) {
			a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "restart", Name: inst.Name, Version: inst.Version})
			if err := stack.Down(""); err != nil {
				a.emitUpdateError("runtime", name, inst.Version, err)
				return message, err
			}
			upCtx, upCancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer upCancel()
			if _, err := stack.Up(upCtx, a.stackOptions(project)); err != nil {
				a.emitUpdateError("runtime", name, inst.Version, err)
				return message, err
			}
			message += " Stack direstart."
		}
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "done", Name: inst.Name, Version: inst.Version})
	return message, nil
}
func (a *App) installBrewRuntime(name, version string) (string, error) {
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "start", Name: name, Version: version})
	ctx, cancel := a.startUpdateContext(30 * time.Minute)
	defer a.finishUpdate(cancel)
	if _, err := brew.Upgrade(ctx, name); err != nil {
		a.emitUpdateError("runtime", name, version, err)
		return "", err
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "done", Name: name, Version: version})
	return a.restartActiveStack("Runtime " + name + " diperbarui via Homebrew."), nil
}

func (a *App) updateRuntimesBrew(patches []update.RuntimeUpdate) (string, error) {
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "start", Name: "semua patch"})
	ctx, cancel := a.startUpdateContext(30 * time.Minute)
	defer a.finishUpdate(cancel)
	var done []string
	var failures []string
	for _, u := range patches {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if _, err := brew.Upgrade(ctx, u.Name); err != nil {
			failures = append(failures, u.Name+": "+err.Error())
			continue
		}
		done = append(done, u.Name)
	}
	message := a.restartActiveStack("Runtime diperbarui via Homebrew: " + strings.Join(done, ", ") + ".")
	if len(failures) > 0 {
		err := devlokerr.New(devlokerr.CodeDependencyMissing,
			fmt.Sprintf("%d diperbarui, %d gagal: %s", len(done), len(failures), strings.Join(failures, "; ")),
			"Jalankan brew upgrade secara manual.")
		a.emitUpdateError("runtime", "semua patch", "", err)
		return message, err
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "done", Name: "semua patch"})
	return message, nil
}

func (a *App) restartActiveStack(message string) string {
	project := a.trayProjectPath()
	if project == "" {
		return message
	}
	if st, err := stack.LoadState(""); err != nil || !sameTrayPath(st.ProjectDir, project) {
		return message
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "restart", Name: "stack"})
	if err := stack.Down(""); err != nil {
		return message
	}
	upCtx, upCancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer upCancel()
	if _, err := stack.Up(upCtx, a.stackOptions(project)); err != nil {
		return message
	}
	return message + " Stack direstart."
}

func (a *App) UpdateAllPatches() (string, error) {
	message, err := a.updateRuntimes(false)
	if err != nil {
		return "", err
	}
	if message == "" {
		return "Tidak ada patch runtime untuk dipasang.", nil
	}
	return message, nil
}

func (a *App) updateRuntimes(includeNewSeries bool) (string, error) {
	checkMgr, err := a.runtimeManager()
	if err != nil {
		return "", err
	}
	updates, err := update.Check(checkMgr)
	if err != nil {
		return "", err
	}
	var patches []update.RuntimeUpdate
	for _, u := range updates {
		if u.Name == "mysql" {
			continue
		}
		if u.Kind == update.KindPatch || (includeNewSeries && u.Kind == update.KindNew) {
			patches = append(patches, u)
		}
	}
	if len(patches) == 0 {
		return "", nil
	}
	if platform.Current().Name() == "darwin" {
		return a.updateRuntimesBrew(patches)
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "start", Name: "semua patch"})
	mgr, err := a.runtimeManagerWithProgress(func(p rt.Progress) {
		a.emitUpdateProgress(UpdateProgressView{
			Kind:       "runtime",
			Phase:      p.Phase,
			Name:       p.Name,
			Version:    p.Version,
			Downloaded: p.Downloaded,
			Total:      p.Total,
		})
	})
	if err != nil {
		a.emitUpdateError("runtime", "semua patch", "", err)
		return "", err
	}
	ctx, cancel := a.startUpdateContext(30 * time.Minute)
	defer a.finishUpdate(cancel)
	var installed []update.RuntimeUpdate
	var failures []string
	for _, u := range patches {
		if ctx.Err() != nil {
			a.emitUpdateError("runtime", "semua patch", "", ctx.Err())
			return "", ctx.Err()
		}
		if _, _, err := mgr.Install(ctx, u.Name, u.Latest); err != nil {
			failures = append(failures, u.Name+" "+u.Latest+": "+err.Error())
			continue
		}
		installed = append(installed, u)
	}
	project := a.trayProjectPath()
	changed := false
	if project != "" {
		if m, err := loadManifest(project); err == nil {
			for _, u := range installed {
				current := projectServiceVersion(m, u.Name)
				if current == "" || !update.SameSeries(current, u.Latest) || update.Compare(current, u.Latest) >= 0 {
					continue
				}
				if err := a.SetServiceVersion(project, u.Name, u.Latest); err != nil {
					failures = append(failures, u.Name+" "+u.Latest+": "+err.Error())
					continue
				}
				changed = true
			}
		}
	}
	restart := changed
	for _, u := range installed {
		if u.Name == "phpmyadmin" || u.Name == "adminer" {
			restart = true
		}
	}
	if restart {
		a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "restart", Name: "semua patch"})
		if st, err := stack.LoadState(""); err == nil && sameTrayPath(st.ProjectDir, project) {
			if err := stack.Down(""); err != nil {
				failures = append(failures, "stop stack: "+err.Error())
			} else {
				upCtx, upCancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer upCancel()
				if _, err := stack.Up(upCtx, a.stackOptions(project)); err != nil {
					failures = append(failures, "start stack: "+err.Error())
				}
			}
		}
	}
	if len(failures) > 0 {
		err := devlokerr.New(devlokerr.CodeDependencyMissing,
			fmt.Sprintf("%d patch dipasang, %d gagal: %s", len(installed), len(failures), strings.Join(failures, "; ")),
			"Periksa koneksi lalu ulangi update.")
		a.emitUpdateError("runtime", "semua patch", "", err)
		return "", err
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "runtime", Phase: "done", Name: "semua patch"})
	message := fmt.Sprintf("%d update runtime dipasang.", len(installed))
	if changed {
		message += " Project diperbarui."
	}
	return message, nil
}

func (a *App) DownloadAppUpdate() (string, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return "", err
	}
	appUpdate, err := a.checkAppUpdate(state)
	if err != nil {
		return "", err
	}
	if appUpdate == nil {
		return "", devlokerr.New(devlokerr.CodeInvalidArgument, "Tidak ada update aplikasi.", "")
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "app", Phase: "start", Version: appUpdate.Latest})
	ctx, cancel := a.startUpdateContext(15 * time.Minute)
	defer a.finishUpdate(cancel)
	path, err := update.DownloadWithProgress(ctx, appUpdate, filepath.Join(state, "cache", "updates"), func(done, total int64) {
		a.emitUpdateProgress(UpdateProgressView{Kind: "app", Phase: "download", Version: appUpdate.Latest, Downloaded: done, Total: total})
	})
	if err != nil {
		a.emitUpdateError("app", "", appUpdate.Latest, err)
		return "", err
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "app", Phase: "done", Version: appUpdate.Latest})
	return path, nil
}

func (a *App) ApplyAppUpdate() error {
	state, err := platform.Current().StateDir()
	if err != nil {
		return err
	}
	appUpdate, err := a.checkAppUpdate(state)
	if err != nil {
		return err
	}
	if appUpdate == nil {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Tidak ada update aplikasi.", "")
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "app", Phase: "start", Version: appUpdate.Latest})
	ctx, cancel := a.startUpdateContext(15 * time.Minute)
	defer a.finishUpdate(cancel)
	path, err := update.DownloadWithProgress(ctx, appUpdate, filepath.Join(state, "cache", "updates"), func(done, total int64) {
		a.emitUpdateProgress(UpdateProgressView{Kind: "app", Phase: "download", Version: appUpdate.Latest, Downloaded: done, Total: total})
	})
	if err != nil {
		a.emitUpdateError("app", "", appUpdate.Latest, err)
		return err
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "app", Phase: "installer", Version: appUpdate.Latest})
	if err := stack.Down(""); err != nil {
		a.emitUpdateError("app", "", appUpdate.Latest, err)
		return err
	}
	if err := update.LaunchInstallerDetached(path); err != nil {
		a.emitUpdateError("app", "", appUpdate.Latest, err)
		return err
	}
	a.emitUpdateProgress(UpdateProgressView{Kind: "app", Phase: "done", Version: appUpdate.Latest})
	go a.quitApp()
	return nil
}

func (a *App) RollbackAppUpdate() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	appDir := filepath.Dir(exe)
	backup := update.RollbackBackupDir(appDir)
	if _, err := os.Stat(filepath.Join(backup, "devlok-gui.exe")); err != nil {
		return devlokerr.New(devlokerr.CodeDependencyMissing, "Backup aplikasi sebelumnya tidak ditemukan.", "Rollback hanya tersedia setelah upgrade melalui installer DevLok.")
	}
	if err := stack.Down(""); err != nil {
		return err
	}
	if err := update.ScheduleRollback(appDir); err != nil {
		return err
	}
	go a.quitApp()
	return nil
}
func (a *App) Version() string {
	return version.Current().Version
}
