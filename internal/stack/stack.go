package stack

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"devlok/internal/config"
	"devlok/internal/db"
	"devlok/internal/devlokerr"
	"devlok/internal/hosts"
	"devlok/internal/instance"
	"devlok/internal/phpcfg"
	"devlok/internal/platform"
	"devlok/internal/ports"
	"devlok/internal/privileged"
	rt "devlok/internal/runtime"
	"devlok/internal/supervisor"
	"devlok/internal/tlsca"
	"devlok/internal/usercfg"
	"devlok/internal/web"
)

//go:embed cacert.pem
var caBundle []byte

type Options struct {
	StateDir   string
	ProjectDir string
	SkipHosts  bool
	SkipTrust  bool
	Progress   func(rt.Progress)
	Registries []rt.Registry
}

type Result struct {
	Name          string   `json:"name"`
	Domain        string   `json:"domain"`
	URL           string   `json:"url"`
	LocalURL      string   `json:"local_url"`
	PHPMyAdminURL string   `json:"phpmyadmin_url,omitempty"`
	MailpitURL    string   `json:"mailpit_url,omitempty"`
	MailpitSMTP   int      `json:"mailpit_smtp,omitempty"`
	DBName        string   `json:"db_name,omitempty"`
	DBUser        string   `json:"db_user,omitempty"`
	DBPort        int      `json:"db_port,omitempty"`
	HTTPPort      int      `json:"http_port"`
	HTTPSPort     int      `json:"https_port"`
	PHPPort       int      `json:"php_port"`
	Warnings      []string `json:"warnings,omitempty"`
}

type State struct {
	Schema     int       `json:"schema"`
	Name       string    `json:"name"`
	ProjectDir string    `json:"project_dir"`
	Domain     string    `json:"domain"`
	URL        string    `json:"url"`
	LocalURL   string    `json:"local_url"`
	HTTPPort   int       `json:"http_port"`
	HTTPSPort  int       `json:"https_port"`
	PHPPort    int       `json:"php_port"`
	DBName     string    `json:"db_name,omitempty"`
	DBUser     string    `json:"db_user,omitempty"`
	DBPort     int       `json:"db_port,omitempty"`
	MySQL      string    `json:"mysql_version,omitempty"`
	MailpitURL string    `json:"mailpit_url,omitempty"`
	StartedAt  time.Time `json:"started_at"`
}

type paths struct {
	stateDir   string
	projectDir string
	manifest   *config.Manifest
	runtimes   string
	cache      string
	generated  string
	logs       string
	certs      string
	services   string
	stackState string
	lock       *config.Lock
}

func Up(ctx context.Context, opts Options) (Result, error) {
	p, err := prepare(opts)
	if err != nil {
		return Result{}, err
	}
	_ = os.Remove(supervisorStopPath(p.stateDir))
	endPause := beginSupervisorPause(p.stateDir)
	defer endPause()
	if _, err := os.Stat(p.stackState); os.IsNotExist(err) {
		cleanupRuntimeProcesses(p.stateDir)
	}
	mgr, err := rt.NewManager(rt.Options{
		Root:       p.runtimes,
		Cache:      p.cache,
		Progress:   opts.Progress,
		Registries: opts.Registries,
	})
	if err != nil {
		return Result{}, err
	}
	nginxVersion, nginxSHA := p.manifest.Runtime["nginx"], ""
	phpVersion, phpSHA := p.manifest.Web.PHP, ""
	mysqlVersion := p.manifest.Services["mysql"]
	if mysqlVersion == "" {
		mysqlVersion = "8.4"
	}
	if p.lock != nil {
		if p.lock.Web.Version != "" {
			nginxVersion, nginxSHA = p.lock.Web.Version, p.lock.Web.SHA256
		}
		if v := p.lock.Runtime["php"]; v.Version != "" {
			phpVersion, phpSHA = v.Version, v.SHA256
		}
		if v := p.lock.Services["mysql"]; v.Version != "" {
			mysqlVersion = v.Version
		}
	}
	nginx, err := installPlatformRuntime(ctx, mgr, "nginx", nginxVersion, nginxSHA, p.stateDir)
	if err != nil {
		return Result{}, err
	}
	php, err := installPlatformRuntime(ctx, mgr, "php", phpVersion, phpSHA, p.stateDir)
	if err != nil {
		return Result{}, err
	}
	var preWarnings []string
	mysqlEnabled := p.manifest.ComponentEnabled("mysql")
	pmaEnabled := mysqlEnabled && p.manifest.ComponentEnabled("phpmyadmin")
	var pma rt.Installed
	pmaOK := false
	if pmaEnabled {
		if inst, _, err := installPhpMyAdmin(ctx, mgr, php.Version); err != nil {
			preWarnings = append(preWarnings, "phpMyAdmin tidak dapat disiapkan: "+err.Error())
		} else {
			pma = inst
			pmaOK = true
		}
	}
	if !mysqlEnabled && p.manifest.ComponentEnabled("phpmyadmin") {
		preWarnings = append(preWarnings, "phpMyAdmin dilewati karena komponen mysql dimatikan.")
	}

	mailpitEnabled := p.manifest.MailpitEnabled()
	var mailpit rt.Installed
	mailpitOK := false
	if mailpitEnabled {
		if inst, _, err := mgr.Install(ctx, "mailpit", "1.32"); err != nil {
			preWarnings = append(preWarnings, "Mailpit tidak dapat disiapkan: "+err.Error())
		} else {
			mailpit = inst
			mailpitOK = true
		}
	}

	var dbc db.DatabaseCreds
	var dbm *db.Manager
	mysqlPort := 0
	if mysqlEnabled {
		dbm, err = db.New(db.Options{StateDir: p.stateDir, Version: mysqlVersion, Port: p.manifest.Ports.MySQL, Progress: opts.Progress, Registries: opts.Registries})
		if err != nil {
			return Result{}, err
		}
		if err := dbm.Ensure(ctx); err != nil {
			return Result{}, err
		}
		dbc, err = dbm.EnsureDatabase(ctx, p.manifest.Name, p.manifest.DB.Name)
		if err != nil {
			return Result{}, err
		}
		mysqlPort = dbm.Port
	}

	sm, err := supervisor.NewManager(p.services)
	if err != nil {
		return Result{}, err
	}
	stopExisting(sm)
	if !mysqlEnabled {
		for _, svc := range sm.List() {
			if strings.HasPrefix(svc.Name, "mysql-") {
				_ = sm.Stop(svc.Name)
			}
		}
	}

	httpPort, err := chooseHTTPPort(p.manifest.Ports.HTTP)
	if err != nil {
		return Result{}, err
	}
	httpsPort, err := chooseHTTPSPort(p.manifest.Ports.HTTPS)
	if err != nil {
		return Result{}, err
	}
	phpPort, err := choosePHPPort(p.manifest.Ports.PHP)
	if err != nil {
		return Result{}, err
	}

	mailpitSMTPPort := 0
	mailpitHTTPPort := 0
	if mailpitOK {
		mailpitSMTPPort, err = choosePort(p.manifest.Ports.MailpitSMTP, "Mailpit SMTP", 1025, 1026, 1027)
		if err != nil {
			preWarnings = append(preWarnings, "Mailpit dilewati: "+err.Error())
			mailpitOK = false
		}
	}
	if mailpitOK {
		mailpitHTTPPort, err = choosePort(p.manifest.Ports.MailpitHTTP, "Mailpit HTTP", 8025, 8026, 8027)
		if err != nil {
			preWarnings = append(preWarnings, "Mailpit dilewati: "+err.Error())
			mailpitOK = false
		}
	}

	res := Result{
		Name:        p.manifest.Name,
		Domain:      p.manifest.BaseDomain(),
		HTTPPort:    httpPort,
		HTTPSPort:   httpsPort,
		PHPPort:     phpPort,
		DBName:      dbc.Name,
		DBUser:      dbc.User,
		DBPort:      mysqlPort,
		MailpitSMTP: mailpitSMTPPort,
	}
	res.Warnings = append(res.Warnings, preWarnings...)
	if p.lock != nil {
		res.Warnings = append(res.Warnings, config.CompareLock(p.manifest, p.lock)...)
		if p.lock.Platform != "" && p.lock.Platform != mgr.OS+"/"+mgr.Arch {
			res.Warnings = append(res.Warnings, "Lockfile dibuat untuk platform "+p.lock.Platform+", bukan "+mgr.OS+"/"+mgr.Arch+".")
		}
	}
	if p.manifest.HTTPSEnabled() {
		res.URL = urlFor("https", p.manifest.BaseDomain(), httpsPort)
	} else {
		res.URL = urlFor("http", p.manifest.BaseDomain(), httpPort)
	}
	res.LocalURL = urlFor("http", "127.0.0.1", httpPort)

	plat := platform.Current()
	ca, err := tlsca.EnsureCA(p.certs)
	if err != nil {
		return Result{}, err
	}
	if !opts.SkipTrust {
		if err := ca.Trust(); err != nil {
			res.Warnings = append(res.Warnings, "CA belum dipercaya: "+err.Error())
		}
	}

	var servers []web.Server
	siteServer := web.Server{
		Name:        p.manifest.Name,
		Domain:      p.manifest.BaseDomain(),
		Aliases:     p.manifest.Web.Aliases,
		Root:        filepath.Join(p.projectDir, p.manifest.Web.Root),
		PHPUpstream: fmt.Sprintf("127.0.0.1:%d", phpPort),
		HTTPS:       p.manifest.HTTPSEnabled(),
	}
	var domains []string
	domains = append(domains, p.manifest.Domains()...)
	if siteServer.HTTPS {
		siteServer.CertFile, siteServer.KeyFile, err = ca.Issue(p.manifest.Name, p.manifest.Domains())
		if err != nil {
			return Result{}, err
		}
	}
	servers = append(servers, siteServer)

	if pmaOK {
		pmaDomain := "phpmyadmin.test"
		pmaServer := web.Server{
			Name:        "phpmyadmin",
			Domain:      pmaDomain,
			Root:        pma.Path,
			PHPUpstream: fmt.Sprintf("127.0.0.1:%d", phpPort),
			HTTPS:       p.manifest.HTTPSEnabled(),
		}
		if pmaServer.HTTPS {
			pmaServer.CertFile, pmaServer.KeyFile, err = ca.Issue("phpmyadmin", []string{pmaDomain})
			if err != nil {
				return Result{}, err
			}
		}
		servers = append(servers, pmaServer)
		domains = append(domains, pmaDomain)
		res.PHPMyAdminURL = urlFor(scheme(pmaServer.HTTPS), pmaDomain, portFor(pmaServer.HTTPS, httpPort, httpsPort))
		if err := writePHPMyAdminConfig(pma.Path, mysqlPort); err != nil {
			res.Warnings = append(res.Warnings, "Konfigurasi phpMyAdmin gagal: "+err.Error())
		}
	}

	if mailpitOK {
		mailpitDomain := "mailpit.test"
		mailpitServer := web.Server{
			Name:   "mailpit",
			Domain: mailpitDomain,
			Proxy:  fmt.Sprintf("127.0.0.1:%d", mailpitHTTPPort),
			HTTPS:  p.manifest.HTTPSEnabled(),
		}
		if mailpitServer.HTTPS {
			mailpitServer.CertFile, mailpitServer.KeyFile, err = ca.Issue("mailpit", []string{mailpitDomain})
			if err != nil {
				return Result{}, err
			}
		}
		servers = append(servers, mailpitServer)
		domains = append(domains, mailpitDomain)
		res.MailpitURL = urlFor(scheme(mailpitServer.HTTPS), mailpitDomain, portFor(mailpitServer.HTTPS, httpPort, httpsPort))
	}

	iniPath := filepath.Join(p.generated, "php", php.Version, "php.ini")
	caPath := filepath.Join(p.generated, "php", "cacert.pem")
	if err := os.MkdirAll(filepath.Dir(caPath), 0o755); err != nil {
		return Result{}, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder CA bundle tidak dapat dibuat.", "")
	}
	if err := os.WriteFile(caPath, caBundle, 0o644); err != nil {
		return Result{}, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "CA bundle tidak dapat ditulis.", "")
	}
	extensions := phpcfg.Load(p.stateDir, php.Version)
	phpCustomPath := usercfg.Path(p.stateDir, "php", php.Version+".ini")
	phpCustomHeader := fmt.Sprintf(`; DevLok PHP override untuk PHP %s
; File ini ditambahkan ke akhir php.ini setiap Start All.
; Contoh:
; memory_limit = 512M
; upload_max_filesize = 128M
; post_max_size = 128M
; max_execution_time = 300
`, php.Version)
	if err := usercfg.Ensure(phpCustomPath, phpCustomHeader); err != nil {
		return Result{}, err
	}
	phpCustom, err := usercfg.Read(phpCustomPath)
	if err != nil {
		return Result{}, err
	}
	sendmail := ""
	if mailpitOK {
		mailpitExe := filepath.Join(mailpit.Path, "mailpit"+platform.Current().ExeSuffix())
		sendmail = fmt.Sprintf("\"%s\" sendmail -S 127.0.0.1:%d", mailpitExe, mailpitSMTPPort)
	}
	if err := writePHPIni(iniPath, php.Path, filepath.Join(p.logs, "php-error.log"), caPath, extensions, phpCustom, sendmail); err != nil {
		return Result{}, err
	}

	if err := web.Generate(web.Config{
		PrefixDir: nginx.Path,
		ConfPath:  filepath.Join(p.generated, "nginx", "nginx.conf"),
		SitesDir:  filepath.Join(p.generated, "nginx", "sites"),
		CustomDir: usercfg.Dir(p.stateDir, "nginx"),
		LogsDir:   filepath.Join(p.logs, "nginx"),
		MimeTypes: filepath.Join(nginx.Path, "conf", "mime.types"),
		HTTPPort:  httpPort,
		HTTPSPort: httpsPort,
		Servers:   servers,
	}); err != nil {
		return Result{}, err
	}
	confPath := filepath.Join(p.generated, "nginx", "nginx.conf")
	if err := web.Validate(filepath.Join(nginx.Path, "nginx"+platform.Current().ExeSuffix()), nginx.Path, confPath); err != nil {
		return Result{}, err
	}

	if !opts.SkipHosts {
		hm := hosts.New(plat.HostsPath())
		hm.Flush = plat.FlushDNS
		hr, err := hm.Apply(domains)
		if err != nil {
			if perr := privileged.SyncHosts(domains); perr != nil {
				res.Warnings = append(res.Warnings, "Hosts tidak dapat diperbarui: "+perr.Error())
			}
		} else if len(hr.Conflicts) > 0 {
			res.Warnings = append(res.Warnings, "Hosts sudah punya entri untuk: "+strings.Join(hr.Conflicts, ", "))
		}
	}

	phpSpec, err := newPHPSpec(php, iniPath, filepath.Join(p.logs, "php-error.log"), p.generated, p.projectDir, phpPort)
	if err != nil {
		return Result{}, err
	}
	if _, err := sm.Start(phpSpec); err != nil {
		return Result{}, err
	}
	nginxSpec := supervisor.Spec{
		Name:       "nginx",
		Executable: filepath.Join(nginx.Path, "nginx"+platform.Current().ExeSuffix()),
		Args:       []string{"-p", nginx.Path, "-c", confPath},
		Dir:        nginx.Path,
		Health:     supervisor.Health{HTTPURL: fmt.Sprintf("http://127.0.0.1:%d/", httpPort), Timeout: 10 * time.Second},
		Policy:     supervisor.PolicyNo,
		Persistent: true,
	}
	if _, err := sm.Start(nginxSpec); err != nil {
		_ = sm.Stop("php")
		return Result{}, err
	}

	var mailpitSpec supervisor.Spec
	mailpitStarted := false
	if mailpitOK {
		dbDir := filepath.Join(p.stateDir, "data", "mailpit")
		if err := os.MkdirAll(dbDir, 0o700); err != nil {
			res.Warnings = append(res.Warnings, "Mailpit dilewati: "+err.Error())
			res.MailpitURL = ""
		} else if err := os.MkdirAll(p.logs, 0o755); err != nil {
			res.Warnings = append(res.Warnings, "Mailpit dilewati: "+err.Error())
			res.MailpitURL = ""
		} else {
			mailpitSpec = supervisor.Spec{
				Name:       "mailpit",
				Executable: filepath.Join(mailpit.Path, "mailpit"+platform.Current().ExeSuffix()),
				Args: []string{
					"--smtp", fmt.Sprintf("127.0.0.1:%d", mailpitSMTPPort),
					"--listen", fmt.Sprintf("127.0.0.1:%d", mailpitHTTPPort),
					"--database", filepath.Join(dbDir, "mailpit.db"),
					"--log-file", filepath.Join(p.logs, "mailpit.log"),
				},
				Dir:        mailpit.Path,
				Health:     supervisor.Health{HTTPURL: fmt.Sprintf("http://127.0.0.1:%d/", mailpitHTTPPort), Timeout: 15 * time.Second},
				Policy:     supervisor.PolicyNo,
				Persistent: true,
			}
			if _, err := sm.Start(mailpitSpec); err != nil {
				res.Warnings = append(res.Warnings, "Mailpit tidak dapat dijalankan: "+err.Error())
				res.MailpitURL = ""
			} else {
				mailpitStarted = true
			}
		}
	}

	st := State{
		Schema:     1,
		Name:       res.Name,
		ProjectDir: p.projectDir,
		Domain:     res.Domain,
		URL:        res.URL,
		LocalURL:   res.LocalURL,
		HTTPPort:   httpPort,
		HTTPSPort:  httpsPort,
		PHPPort:    phpPort,
		DBName:     dbc.Name,
		DBUser:     dbc.User,
		DBPort:     mysqlPort,
		MySQL:      mysqlVersion,
		MailpitURL: res.MailpitURL,
		StartedAt:  time.Now().UTC(),
	}
	if err := writeState(p.stackState, st); err != nil {
		_ = sm.Stop("nginx")
		_ = sm.Stop("php")
		return Result{}, err
	}
	specs := []supervisor.Spec{phpSpec, nginxSpec}
	if mysqlEnabled {
		specs = append(specs, dbm.Spec())
	}
	if mailpitStarted {
		specs = append(specs, mailpitSpec)
	}
	if err := writeSupervisorSpecs(p.stateDir, specs); err != nil {
		res.Warnings = append(res.Warnings, "Supervisor latar tidak dapat disiapkan: "+err.Error())
	} else if err := startSupervisor(p.stateDir); err != nil {
		res.Warnings = append(res.Warnings, "Supervisor latar tidak dapat dijalankan: "+err.Error())
	}
	return res, nil
}

func Down(stateDir string) error {
	dirs, err := stateDirs(stateDir)
	if err != nil {
		return err
	}
	state := stateDirOrDefault(stateDir)
	endPause := beginSupervisorPause(state)
	defer endPause()
	_ = os.Remove(dirs.stackState)
	_ = os.Remove(supervisorSpecsPath(state))
	sm, err := supervisor.NewManager(dirs.services)
	if err != nil {
		return err
	}
	var errs []string
	for _, name := range []string{"nginx", "php", "phpmyadmin", "mailpit"} {
		if err := sm.Stop(name); err != nil {
			errs = append(errs, name+": "+err.Error())
		}
	}
	for _, svc := range sm.List() {
		if strings.HasPrefix(svc.Name, "mysql-") {
			if err := sm.Stop(svc.Name); err != nil {
				errs = append(errs, svc.Name+": "+err.Error())
			}
		}
	}
	cleanupRuntimeProcesses(state)
	StopSupervisor(state)
	if len(errs) > 0 {
		return devlokerr.New(devlokerr.CodeStateInconsistent, "Sebagian service gagal dihentikan.", strings.Join(errs, "; "))
	}
	return nil
}

func stateDirOrDefault(stateDir string) string {
	if stateDir != "" {
		return stateDir
	}
	dir, _ := platform.Current().StateDir()
	return dir
}

func cleanupRuntimeProcesses(stateDir string) {
	servicesPath := filepath.Join(stateDir, "state", "services.json")
	if sm, err := supervisor.NewManager(servicesPath); err == nil {
		for _, svc := range sm.List() {
			if svc.State == supervisor.StateRunning {
				_ = sm.Stop(svc.Name)
			}
		}
	}
	root := strings.ToLower(filepath.Join(stateDir, "runtimes") + string(os.PathSeparator))
	for _, port := range []int{80, 443, 3306, 8080, 8443} {
		st := ports.Check(port)
		if st.Available || st.PID <= 0 || st.Path == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(st.Path), root) {
			_ = platform.Current().KillTree(st.PID)
		}
	}
	_, _ = platform.Current().KillProcessesUnder(filepath.Join(stateDir, "runtimes"))
}

func LoadState(stateDir string) (State, error) {
	dirs, err := stateDirs(stateDir)
	if err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(dirs.stackState)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, devlokerr.New(devlokerr.CodeStateInconsistent, "Stack belum berjalan.", "Jalankan dl up.")
		}
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil || st.Schema != 1 {
		return State{}, devlokerr.New(devlokerr.CodeStateInconsistent, "State stack rusak.", "Jalankan dl down lalu dl up.")
	}
	return st, nil
}

func Services(stateDir string) ([]supervisor.Status, error) {
	dirs, err := stateDirs(stateDir)
	if err != nil {
		return nil, err
	}
	sm, err := supervisor.NewManager(dirs.services)
	if err != nil {
		return nil, err
	}
	return sm.List(), nil
}

func prepare(opts Options) (paths, error) {
	stateDir := opts.StateDir
	if stateDir == "" {
		var err error
		stateDir, err = platform.Current().StateDir()
		if err != nil {
			return paths{}, err
		}
	}
	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return paths{}, err
	}
	if long, err := filepath.EvalSymlinks(abs); err == nil {
		abs = long
	}
	manifestPath := filepath.Join(abs, config.FileName)
	if _, err := os.Stat(manifestPath); err != nil {
		found, ok := config.Find(abs)
		if !ok {
			return paths{}, devlokerr.New(devlokerr.CodeManifestNotFound, "Manifest tidak ditemukan.", "Jalankan dl init atau dl site create.")
		}
		manifestPath = found
		abs = filepath.Dir(found)
	}
	m, err := config.Load(manifestPath)
	if err != nil {
		return paths{}, err
	}
	var lock *config.Lock
	if l, err := config.LoadLock(filepath.Join(abs, config.LockFileName)); err == nil {
		lock = l
	} else if !os.IsNotExist(err) {
		return paths{}, err
	}
	dirs, err := stateDirs(stateDir)
	if err != nil {
		return paths{}, err
	}
	p := paths{
		stateDir:   stateDir,
		projectDir: abs,
		manifest:   m,
		runtimes:   filepath.Join(stateDir, "runtimes"),
		cache:      filepath.Join(stateDir, "cache", "downloads"),
		generated:  filepath.Join(stateDir, "generated"),
		logs:       filepath.Join(stateDir, "logs"),
		certs:      filepath.Join(stateDir, "certs"),
		services:   dirs.services,
		stackState: dirs.stackState,
		lock:       lock,
	}
	return p, nil
}

type runtimeDirs struct {
	services   string
	stackState string
	lock       *config.Lock
}

func stateDirs(stateDir string) (runtimeDirs, error) {
	if stateDir == "" {
		var err error
		stateDir, err = platform.Current().StateDir()
		if err != nil {
			return runtimeDirs{}, err
		}
	}
	state := filepath.Join(stateDir, "state")
	if err := os.MkdirAll(state, 0o700); err != nil {
		return runtimeDirs{}, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder state tidak dapat dibuat.", "")
	}
	return runtimeDirs{
		services:   filepath.Join(state, "services.json"),
		stackState: filepath.Join(state, "stack.json"),
	}, nil
}

type supervisorFile struct {
	Schema int               `json:"schema"`
	Specs  []supervisor.Spec `json:"specs"`
}

func supervisorSpecsPath(stateDir string) string {
	return filepath.Join(stateDir, "state", "supervisor.json")
}

func supervisorPausePath(stateDir string) string {
	return filepath.Join(stateDir, "state", "supervisor.pause")
}

func supervisorStopPath(stateDir string) string {
	return filepath.Join(stateDir, "state", "supervisor.stop")
}

func StopSupervisor(stateDir string) {
	state := stateDirOrDefault(stateDir)
	_ = os.WriteFile(supervisorStopPath(state), []byte("stop\n"), 0o600)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		lock, err := instance.Acquire("supervisor:" + state)
		if err == nil {
			_ = lock.Close()
			_ = os.Remove(supervisorStopPath(state))
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = os.Remove(supervisorStopPath(state))
}

func writeSupervisorSpecs(stateDir string, specs []supervisor.Spec) error {
	if len(specs) == 0 {
		return nil
	}
	data, err := json.MarshalIndent(supervisorFile{Schema: 1, Specs: specs}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := supervisorSpecsPath(stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

func LoadSupervisorSpecs(stateDir string) ([]supervisor.Spec, error) {
	if stateDir == "" {
		var err error
		stateDir, err = platform.Current().StateDir()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(supervisorSpecsPath(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) > 1<<20 {
		return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Konfigurasi supervisor terlalu besar.", "")
	}
	var sf supervisorFile
	if err := json.Unmarshal(data, &sf); err != nil || sf.Schema != 1 {
		return nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Konfigurasi supervisor rusak.", "Jalankan dl down lalu dl up.")
	}
	return sf.Specs, nil
}

func SupervisorPaused(stateDir string) bool {
	if stateDir == "" {
		var err error
		stateDir, err = platform.Current().StateDir()
		if err != nil {
			return false
		}
	}
	info, err := os.Stat(supervisorPausePath(stateDir))
	if err != nil {
		return false
	}
	if time.Since(info.ModTime()) > 60*time.Minute {
		_ = os.Remove(supervisorPausePath(stateDir))
		return false
	}
	return true
}

func beginSupervisorPause(stateDir string) func() {
	path := supervisorPausePath(stateDir)
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
	return func() { _ = os.Remove(path) }
}

func stopExisting(sm *supervisor.Manager) {
	for _, name := range []string{"nginx", "php", "mailpit"} {
		_ = sm.Stop(name)
	}
}

func chooseHTTPPort(explicit int) (int, error) {
	if platform.Current().Name() == "darwin" && explicit == 0 {
		return choosePort(0, "HTTP", 8080, 8000)
	}
	return choosePort(explicit, "HTTP", 80, 8080)
}

func chooseHTTPSPort(explicit int) (int, error) {
	if platform.Current().Name() == "darwin" && explicit == 0 {
		return choosePort(0, "HTTPS", 8443, 9443)
	}
	return choosePort(explicit, "HTTPS", 443, 8443)
}

func installPhpMyAdmin(ctx context.Context, mgr *rt.Manager, phpVersion string) (rt.Installed, bool, error) {
	preferred := "5.2"
	if versionAtLeast(phpVersion, 8, 4) {
		preferred = "6"
	}
	inst, ok, err := mgr.Install(ctx, "phpmyadmin", preferred)
	if err == nil || preferred == "5.2" {
		return inst, ok, err
	}
	inst, ok, err2 := mgr.Install(ctx, "phpmyadmin", "5.2")
	if err2 == nil {
		return inst, ok, nil
	}
	return rt.Installed{}, false, err
}

func versionAtLeast(v string, major, minor int) bool {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) < 2 {
		return false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	if maj != major {
		return maj > major
	}
	return min >= minor
}

func choosePort(explicit int, label string, fallback ...int) (int, error) {
	if explicit > 0 {
		st := ports.Check(explicit)
		if st.Available {
			return explicit, nil
		}
		hint := "Bebaskan port atau ubah bagian ports di manifest."
		if st.PID > 0 {
			hint = fmt.Sprintf("Dipakai %s (PID %d). Bebaskan port atau ubah ports.", st.Process, st.PID)
		}
		return 0, devlokerr.New(devlokerr.CodePortInUse, fmt.Sprintf("Port %s %d sudah dipakai.", label, explicit), hint)
	}
	port, err := ports.Pick(fallback...)
	if err != nil {
		return 0, devlokerr.Wrap(devlokerr.CodePortInUse, err, "Tidak ada port "+label+" yang tersedia.", "Bebaskan port atau set ports di manifest.")
	}
	return port, nil
}

func choosePHPPort(explicit int) (int, error) {
	if explicit > 0 {
		if ports.Check(explicit).Available {
			return explicit, nil
		}
		return 0, devlokerr.New(devlokerr.CodePortInUse, fmt.Sprintf("Port PHP %d sudah dipakai.", explicit), "Ubah ports.php di manifest.")
	}
	return freePort()
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, devlokerr.Wrap(devlokerr.CodePortInUse, err, "Port FastCGI tidak dapat dialokasikan.", "")
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port, nil
}

func writePHPIni(path, phpDir, logPath, caPath string, extensions []string, custom []byte, sendmail string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder php.ini tidak dapat dibuat.", "")
	}
	var b strings.Builder
	b.WriteString("[PHP]\n")
	fmt.Fprintf(&b, "error_log = %q\n", filepath.ToSlash(logPath))
	b.WriteString("date.timezone = Asia/Jakarta\n")
	b.WriteString("display_errors = On\n")
	b.WriteString("display_startup_errors = On\n")
	b.WriteString("error_reporting = E_ALL\n")
	b.WriteString("log_errors = On\n")
	b.WriteString("memory_limit = 256M\n")
	b.WriteString("upload_max_filesize = 64M\n")
	b.WriteString("post_max_size = 64M\n")
	b.WriteString("max_execution_time = 120\n")
	b.WriteString("cgi.fix_pathinfo = 1\n")
	b.WriteString("expose_php = Off\n")
	fmt.Fprintf(&b, "curl.cainfo = %q\n", filepath.ToSlash(caPath))
	fmt.Fprintf(&b, "openssl.cafile = %q\n\n", filepath.ToSlash(caPath))
	if sendmail != "" {
		fmt.Fprintf(&b, "sendmail_path = %q\n", sendmail)
	}
	writePHPExtensionSettings(&b, phpDir, extensions)
	if text := strings.TrimSpace(string(custom)); text != "" {
		b.WriteString("\n; >>> DevLok user override >>>\n")
		b.WriteString(text)
		b.WriteString("\n; <<< DevLok user override <<<\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "php.ini tidak dapat ditulis.", "")
	}
	return nil
}

func writePHPMyAdminConfig(appDir string, port int) error {
	path := filepath.Join(appDir, "config.inc.php")
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	content := fmt.Sprintf(`<?php
declare(strict_types=1);

$cfg['blowfish_secret'] = '%s';

$i = 0;
$i++;
$cfg['Servers'][$i]['auth_type'] = 'cookie';
$cfg['Servers'][$i]['host'] = '127.0.0.1';
$cfg['Servers'][$i]['port'] = '%d';
$cfg['Servers'][$i]['compress'] = false;
$cfg['Servers'][$i]['AllowNoPassword'] = false;
$cfg['UploadDir'] = '';
$cfg['SaveDir'] = '';
`, hex.EncodeToString(secret), port)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "config.inc.php tidak dapat ditulis.", "")
	}
	return nil
}

func writeState(path string, st State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder state tidak dapat dibuat.", "")
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "State stack tidak dapat ditulis.", "")
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "State stack tidak dapat diselesaikan.", "")
	}
	return nil
}

func installRuntime(ctx context.Context, mgr *rt.Manager, name, constraint, wantSHA string) (rt.Installed, error) {
	inst, _, err := mgr.Install(ctx, name, constraint)
	if err != nil {
		return rt.Installed{}, err
	}
	if wantSHA != "" && !strings.EqualFold(inst.SHA256, wantSHA) {
		return rt.Installed{}, devlokerr.New(devlokerr.CodeChecksumMismatch, "Checksum "+name+" tidak cocok dengan lockfile.", "Jalankan dl lock untuk memperbarui lockfile.")
	}
	return inst, nil
}

func urlFor(scheme, host string, port int) string {
	if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
		return scheme + "://" + host
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

func scheme(https bool) string {
	if https {
		return "https"
	}
	return "http"
}

func portFor(https bool, httpPort, httpsPort int) int {
	if https {
		return httpsPort
	}
	return httpPort
}

func CertFiles(stateDir, name string) []string {
	if stateDir == "" {
		stateDir, _ = platform.Current().StateDir()
	}
	cert, _ := filepath.Abs(filepath.Join(stateDir, "certs", "sites", name+".crt"))
	key, _ := filepath.Abs(filepath.Join(stateDir, "certs", "sites", name+".key"))
	return []string{cert, key}
}

func WorkspaceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "DevLok", "sites"), nil
}

func ListProjects(workspace string) ([]config.Manifest, error) {
	entries, err := os.ReadDir(workspace)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []config.Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(workspace, e.Name(), config.FileName)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		m, err := config.Load(path)
		if err != nil {
			continue
		}
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
