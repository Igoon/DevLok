package db

import (
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"devlok/internal/brew"
	"devlok/internal/devlokerr"
	"devlok/internal/hidden"
	"devlok/internal/platform"
	"devlok/internal/ports"
	rt "devlok/internal/runtime"
	"devlok/internal/secret"
	"devlok/internal/supervisor"
	"devlok/internal/usercfg"
)

const credsSchema = 1

type Options struct {
	StateDir   string
	Version    string
	Port       int
	Progress   func(rt.Progress)
	Registries []rt.Registry
}

type Manager struct {
	StateDir         string
	Version          string
	InstallVersion   string
	ServiceName      string
	Install          rt.Installed
	DataDir          string
	ConfigPath       string
	CustomConfigPath string
	LogPath          string
	Port             int
	PreferredPort    int
	Progress         func(rt.Progress)
	Registries       []rt.Registry
}

type InstanceState struct {
	Schema      int    `json:"schema"`
	Version     string `json:"version"`
	Port        int    `json:"port"`
	DataDir     string `json:"data_dir"`
	Initialized bool   `json:"initialized"`
}

type DatabaseCreds struct {
	Project  string `json:"project"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type credentials struct {
	Schema       int                      `json:"schema"`
	RootPassword string                   `json:"root_password"`
	Databases    map[string]DatabaseCreds `json:"databases"`
}

var (
	nameRe    = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	versionRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,2}$`)
)

func New(opts Options) (*Manager, error) {
	if opts.StateDir == "" {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "State dir wajib diisi.", "")
	}
	version := opts.Version
	if version == "" {
		version = "8.4"
	}
	if !versionRe.MatchString(version) {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Versi MySQL tidak valid: "+version, "")
	}
	key := majorMinor(version)
	m := &Manager{
		StateDir:         opts.StateDir,
		Version:          key,
		InstallVersion:   version,
		ServiceName:      "mysql-" + strings.ReplaceAll(key, ".", "-"),
		DataDir:          filepath.Join(opts.StateDir, "data", "mysql", key),
		ConfigPath:       filepath.Join(opts.StateDir, "generated", "mysql", key, "my.ini"),
		CustomConfigPath: usercfg.Path(opts.StateDir, "mysql", key+".ini"),
		LogPath:          filepath.Join(opts.StateDir, "logs", "mysql", key+"-error.log"),
		PreferredPort:    opts.Port,
		Progress:         opts.Progress,
		Registries:       opts.Registries,
	}
	st, err := m.loadState()
	if err == nil && st.Port != 0 {
		m.Port = st.Port
	}
	if opts.Port > 0 {
		m.Port = opts.Port
	}
	return m, nil
}

func (m *Manager) Ensure(ctx context.Context) error {
	if err := m.install(ctx); err != nil {
		return err
	}
	if m.Install.Version != "" {
		m.ServiceName = "mysql-" + strings.ReplaceAll(m.Install.Version, ".", "-")
	}
	st, err := m.loadState()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if st.Version != "" && st.Version != m.Version {
		return devlokerr.New(devlokerr.CodeStateInconsistent, "State MySQL versi tidak cocok.", "")
	}
	sm, err := supervisor.NewManager(m.servicesPath())
	if err != nil {
		return err
	}
	legacyName := "mysql-" + strings.ReplaceAll(m.Version, ".", "-")
	if legacyName != m.ServiceName {
		if svc, ok := sm.Get(legacyName); ok && svc.State == supervisor.StateRunning {
			_ = sm.Stop(legacyName)
		}
	}
	running := false
	portChanged := false
	if svc, ok := sm.Get(m.ServiceName); ok && svc.State == supervisor.StateRunning {
		running = true
		if svc.Port > 0 && st.Port != svc.Port {
			st.Port = svc.Port
			portChanged = true
		}
	}
	if m.PreferredPort > 0 {
		if st.Port != m.PreferredPort {
			if running {
				if err := sm.Stop(m.ServiceName); err != nil {
					return err
				}
				running = false
			}
			if !ports.Check(m.PreferredPort).Available {
				owner := ports.Check(m.PreferredPort)
				return devlokerr.New(devlokerr.CodePortInUse,
					fmt.Sprintf("Port MySQL %d sudah dipakai.", m.PreferredPort),
					fmt.Sprintf("Pemilik: %s (PID %d). Ubah ports.mysql atau bebaskan port.", owner.Process, owner.PID))
			}
			st.Port = m.PreferredPort
			portChanged = true
		} else if !running && !ports.Check(m.PreferredPort).Available {
			owner := ports.Check(m.PreferredPort)
			return devlokerr.New(devlokerr.CodePortInUse,
				fmt.Sprintf("Port MySQL %d sudah dipakai.", m.PreferredPort),
				fmt.Sprintf("Pemilik: %s (PID %d).", owner.Process, owner.PID))
		}
	} else if st.Port == 0 || (!running && !ports.Check(st.Port).Available) {
		port, err := ports.Pick(3306, 3307, 3308, 3309, 3310)
		if err != nil {
			return devlokerr.Wrap(devlokerr.CodePortInUse, err, "Tidak ada port MySQL yang tersedia.", "Bebaskan port 3306-3310.")
		}
		st.Port = port
		portChanged = true
	}
	st.Schema = 1
	st.Version = m.Version
	st.DataDir = m.DataDir
	m.Port = st.Port

	if err := m.writeConfig(); err != nil {
		return err
	}
	if err := m.validateConfig(); err != nil {
		return err
	}
	if !st.Initialized {
		if err := m.initialize(ctx); err != nil {
			return err
		}
		st.Initialized = true
		if err := m.saveState(st); err != nil {
			return err
		}
		portChanged = false
	}
	if portChanged {
		if err := m.saveState(st); err != nil {
			return err
		}
	}
	if !running {
		if _, err := sm.Start(m.spec()); err != nil {
			if tail := readTail(m.LogPath, 30); tail != "" {
				hint := tail
				lower := strings.ToLower(tail)
				if strings.Contains(lower, "initialized") || strings.Contains(lower, "incompatible") || strings.Contains(lower, "upgrade") || strings.Contains(lower, "data dictionary") {
					hint = "Datadir MySQL tampak dibuat oleh versi MySQL lain. Jalankan `dl db reset --yes` lalu `dl up` untuk inisialisasi ulang.\n" + tail
				}
				return devlokerr.New(devlokerr.CodeDependencyMissing, "MySQL "+m.Install.Version+" tidak dapat dijalankan.", hint)
			}
			return err
		}
	}
	creds, err := m.loadCreds()
	if err != nil {
		return err
	}
	if creds.RootPassword == "" {
		if err := m.waitReady(ctx, "root", ""); err != nil {
			return err
		}
		pw, err := randomPassword(32)
		if err != nil {
			return err
		}
		if err := m.setRootPassword(ctx, pw); err != nil {
			return err
		}
		creds.RootPassword = pw
		if creds.Databases == nil {
			creds.Databases = map[string]DatabaseCreds{}
		}
		if err := m.saveCreds(creds); err != nil {
			return err
		}
	}
	if err := m.waitReady(ctx, "root", creds.RootPassword); err != nil {
		return devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Root MySQL tidak dapat login dengan kredensial tersimpan.", "Hapus data MySQL atau perbaiki password secara manual.")
	}
	return nil
}

func (m *Manager) Reset(ctx context.Context) (string, error) {
	if err := m.install(ctx); err != nil {
		return "", err
	}
	if m.Install.Version != "" {
		m.ServiceName = "mysql-" + strings.ReplaceAll(m.Install.Version, ".", "-")
	}
	sm, err := supervisor.NewManager(m.servicesPath())
	if err != nil {
		return "", err
	}
	_ = sm.Stop(m.ServiceName)
	_ = sm.Stop("mysql-" + strings.ReplaceAll(m.Version, ".", "-"))
	backup := ""
	if info, err := os.Stat(m.DataDir); err == nil && info.IsDir() {
		backup = m.DataDir + ".bak-" + time.Now().Format("20060102-150405")
		if err := os.Rename(m.DataDir, backup); err != nil {
			return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Data MySQL tidak dapat dipindahkan.", "")
		}
	}
	_ = os.Remove(m.statePath())
	_ = os.Remove(m.credsPath())
	return backup, nil
}

func (m *Manager) install(ctx context.Context) error {
	if platform.Current().Name() == "darwin" {
		formula := brew.InstalledFormula(ctx, "mysql", m.InstallVersion)
		prefix, err := brew.Prefix(ctx, formula)
		if err != nil {
			return err
		}
		version := brew.Version(ctx, formula)
		if version == "" {
			version = "brew"
		}
		m.Install = rt.Installed{Name: "mysql", Version: version, Path: prefix}
		return nil
	}
	mgr, err := rt.NewManager(rt.Options{
		Root:       filepath.Join(m.StateDir, "runtimes"),
		Cache:      filepath.Join(m.StateDir, "cache", "downloads"),
		Progress:   m.Progress,
		Registries: m.Registries,
	})
	if err != nil {
		return err
	}
	inst, _, err := mgr.Install(ctx, "mysql", m.InstallVersion)
	if err != nil {
		return err
	}
	m.Install = inst
	return nil
}

func (m *Manager) initialize(ctx context.Context) error {
	if info, err := os.Stat(m.DataDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(m.DataDir)
		if len(entries) > 0 {
			return devlokerr.New(devlokerr.CodeStateInconsistent, "Data dir MySQL tidak kosong tetapi belum diinisialisasi.", "Hapus folder "+m.DataDir+" lalu coba lagi.")
		}
	}
	if err := os.MkdirAll(m.DataDir, 0o700); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Data dir MySQL tidak dapat dibuat.", "")
	}
	if err := os.MkdirAll(filepath.Dir(m.LogPath), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, m.mysqldExe(), "--defaults-file="+m.ConfigPath, "--initialize-insecure", "--console")
	hidden.Hide(cmd)
	cmd.Dir = m.Install.Path
	out, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.RemoveAll(m.DataDir)
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = readTail(m.LogPath, 20)
		}
		return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "MySQL gagal diinisialisasi.", msg)
	}
	return nil
}

func (m *Manager) setRootPassword(ctx context.Context, pw string) error {
	sql := fmt.Sprintf(`
ALTER USER 'root'@'localhost' IDENTIFIED BY '%s';
CREATE USER IF NOT EXISTS 'root'@'127.0.0.1' IDENTIFIED BY '%s';
ALTER USER 'root'@'127.0.0.1' IDENTIFIED BY '%s';
GRANT ALL PRIVILEGES ON *.* TO 'root'@'127.0.0.1' WITH GRANT OPTION;
FLUSH PRIVILEGES;
`, pw, pw, pw)
	if err := m.execSQL(ctx, "root", "", "", sql); err != nil {
		return err
	}
	return m.waitReady(ctx, "root", pw)
}

func (m *Manager) EnsureDatabase(ctx context.Context, project, dbName string) (DatabaseCreds, error) {
	if !nameRe.MatchString(dbName) {
		return DatabaseCreds{}, devlokerr.New(devlokerr.CodeInvalidArgument, "Nama database tidak valid: "+dbName, "Gunakan huruf kecil, angka, dan garis bawah.")
	}
	if err := m.Ensure(ctx); err != nil {
		return DatabaseCreds{}, err
	}
	creds, err := m.loadCreds()
	if err != nil {
		return DatabaseCreds{}, err
	}
	if existing, ok := creds.Databases[dbName]; ok {
		if m.canLogin(ctx, existing.User, existing.Password, dbName) {
			return existing, nil
		}
	}
	user := dbUser(project)
	pw := ""
	for _, other := range creds.Databases {
		if other.User == user {
			pw = other.Password
			break
		}
	}
	if pw == "" {
		var err error
		pw, err = randomPassword(24)
		if err != nil {
			return DatabaseCreds{}, err
		}
	}
	sql := fmt.Sprintf(`
CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s';
CREATE USER IF NOT EXISTS '%s'@'127.0.0.1' IDENTIFIED BY '%s';
ALTER USER '%s'@'localhost' IDENTIFIED BY '%s';
ALTER USER '%s'@'127.0.0.1' IDENTIFIED BY '%s';
GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost';
GRANT ALL PRIVILEGES ON %s.* TO '%s'@'127.0.0.1';
FLUSH PRIVILEGES;
`, quoteIdent(dbName), user, pw, user, pw, user, pw, user, pw, quoteIdent(dbName), user, quoteIdent(dbName), user)
	if err := m.execSQL(ctx, "root", creds.RootPassword, "", sql); err != nil {
		return DatabaseCreds{}, err
	}
	dc := DatabaseCreds{Project: project, Name: dbName, User: user, Password: pw}
	if creds.Databases == nil {
		creds.Databases = map[string]DatabaseCreds{}
	}
	creds.Databases[dbName] = dc
	if err := m.saveCreds(creds); err != nil {
		return DatabaseCreds{}, err
	}
	return dc, nil
}

func (m *Manager) Backup(ctx context.Context, dbName, outPath string) (string, error) {
	dc, err := m.Database(dbName)
	if err != nil {
		return "", err
	}
	if err := m.Ensure(ctx); err != nil {
		return "", err
	}
	if outPath == "" {
		outPath = filepath.Join(m.StateDir, "backups", dbName, time.Now().Format("20060102-150405")+".sql.gz")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder backup tidak dapat dibuat.", "")
	}
	cmd := exec.CommandContext(ctx, m.dumpExe(),
		"--protocol=TCP",
		"--host=127.0.0.1",
		"--port="+strconv.Itoa(m.Port),
		"--user="+dc.User,
		"--single-transaction",
		"--quick",
		"--routines",
		"--triggers",
		"--events",
		"--default-character-set=utf8mb4",
		dc.Name,
	)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+dc.Password)
	hidden.Hide(cmd)
	cmd.Dir = m.Install.Path
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "mysqldump tidak dapat dijalankan.", "")
	}
	tmp := outPath + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = cmd.Process.Kill()
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "File backup tidak dapat ditulis.", "")
	}
	gz := gzip.NewWriter(f)
	_, copyErr := io.Copy(gz, stdout)
	closeGzErr := gz.Close()
	closeFileErr := f.Close()
	waitErr := cmd.Wait()
	if copyErr != nil || closeGzErr != nil || closeFileErr != nil || waitErr != nil {
		_ = os.Remove(tmp)
		msg := strings.TrimSpace(stderr.String())
		if msg == "" && waitErr != nil {
			msg = waitErr.Error()
		}
		return "", devlokerr.New(devlokerr.CodeStateInconsistent, "Backup database gagal.", msg)
	}
	if err := os.Rename(tmp, outPath); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Backup tidak dapat diselesaikan.", "")
	}
	return outPath, nil
}

func (m *Manager) Restore(ctx context.Context, dbName, file string) error {
	dc, err := m.Database(dbName)
	if err != nil {
		return err
	}
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	if err := m.pipeSQLFile(ctx, dc.User, dc.Password, dc.Name, file); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Seed(ctx context.Context, dbName, file string) error {
	dc, err := m.Database(dbName)
	if err != nil {
		return err
	}
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	return m.pipeSQLFile(ctx, dc.User, dc.Password, dc.Name, file)
}

func (m *Manager) Drop(ctx context.Context, dbName string) error {
	dc, err := m.Database(dbName)
	if err != nil {
		return err
	}
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	creds, err := m.loadCreds()
	if err != nil {
		return err
	}
	sharedUser := false
	for name, other := range creds.Databases {
		if name != dbName && other.User == dc.User {
			sharedUser = true
			break
		}
	}
	sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s;", quoteIdent(dbName))
	if !sharedUser {
		sql += fmt.Sprintf("\nDROP USER IF EXISTS '%s'@'localhost';\nDROP USER IF EXISTS '%s'@'127.0.0.1';", dc.User, dc.User)
	}
	sql += "\nFLUSH PRIVILEGES;"
	if err := m.execSQL(ctx, "root", creds.RootPassword, "", sql); err != nil {
		return err
	}
	delete(creds.Databases, dbName)
	return m.saveCreds(creds)
}

func (m *Manager) Shell(ctx context.Context, dbName string) error {
	if err := m.Ensure(ctx); err != nil {
		return err
	}
	creds, err := m.loadCreds()
	if err != nil {
		return err
	}
	user, password, database := "root", creds.RootPassword, ""
	if dbName != "" {
		dc, err := m.Database(dbName)
		if err != nil {
			return err
		}
		user, password, database = dc.User, dc.Password, dc.Name
	}
	args := []string{"--protocol=TCP", "--host=127.0.0.1", "--port=" + strconv.Itoa(m.Port), "--user=" + user}
	if database != "" {
		args = append(args, "--database="+database)
	}
	cmd := exec.CommandContext(ctx, m.mysqlExe(), args...)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+password)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (m *Manager) Credentials() (DatabaseCreds, string, error) {
	creds, err := m.loadCreds()
	if err != nil {
		return DatabaseCreds{}, "", err
	}
	return DatabaseCreds{}, creds.RootPassword, nil
}

func (m *Manager) Database(dbName string) (DatabaseCreds, error) {
	creds, err := m.loadCreds()
	if err != nil {
		return DatabaseCreds{}, err
	}
	dc, ok := creds.Databases[dbName]
	if !ok {
		return DatabaseCreds{}, devlokerr.New(devlokerr.CodeStateInconsistent, "Database belum dibuat: "+dbName, "Jalankan dl db create atau dl up.")
	}
	return dc, nil
}

func (m *Manager) Databases() ([]DatabaseCreds, error) {
	creds, err := m.loadCreds()
	if err != nil {
		return nil, err
	}
	out := make([]DatabaseCreds, 0, len(creds.Databases))
	for _, dc := range creds.Databases {
		out = append(out, dc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *Manager) RootPassword() (string, error) {
	creds, err := m.loadCreds()
	if err != nil {
		return "", err
	}
	return creds.RootPassword, nil
}

func (m *Manager) ChangeRootPassword(ctx context.Context) (string, error) {
	if err := m.Ensure(ctx); err != nil {
		return "", err
	}
	creds, err := m.loadCreds()
	if err != nil {
		return "", err
	}
	pw, err := randomPassword(32)
	if err != nil {
		return "", err
	}
	sql := fmt.Sprintf(`
ALTER USER 'root'@'localhost' IDENTIFIED BY '%s';
ALTER USER 'root'@'127.0.0.1' IDENTIFIED BY '%s';
FLUSH PRIVILEGES;
`, pw, pw)
	if err := m.execSQL(ctx, "root", creds.RootPassword, "", sql); err != nil {
		return "", err
	}
	if err := m.waitReady(ctx, "root", pw); err != nil {
		return "", err
	}
	creds.RootPassword = pw
	if err := m.saveCreds(creds); err != nil {
		return "", err
	}
	return pw, nil
}

func (m *Manager) Stop() error {
	sm, err := supervisor.NewManager(m.servicesPath())
	if err != nil {
		return err
	}
	var first error
	for _, svc := range sm.List() {
		if !strings.HasPrefix(svc.Name, "mysql-") {
			continue
		}
		if err := sm.Stop(svc.Name); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (m *Manager) BackupAll(ctx context.Context, outDir string) (string, []string, error) {
	if err := m.Ensure(ctx); err != nil {
		return "", nil, err
	}
	dbs, err := m.Databases()
	if err != nil {
		return "", nil, err
	}
	if len(dbs) == 0 {
		return "", nil, devlokerr.New(devlokerr.CodeStateInconsistent, "Belum ada database untuk dibackup.", "Jalankan dl up atau dl db create.")
	}
	if outDir == "" {
		outDir = filepath.Join(m.StateDir, "backups", "mysql", time.Now().Format("20060102-150405"))
	}
	files := make([]string, 0, len(dbs))
	for _, dc := range dbs {
		out := filepath.Join(outDir, dc.Name+".sql.gz")
		path, err := m.Backup(ctx, dc.Name, out)
		if err != nil {
			return outDir, files, err
		}
		files = append(files, path)
	}
	return outDir, files, nil
}

func (m *Manager) canLogin(ctx context.Context, user, password, database string) bool {
	return m.execSQL(ctx, user, password, database, "SELECT 1;") == nil
}

func (m *Manager) waitReady(ctx context.Context, user, password string) error {
	deadline := time.Now().Add(60 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		if err := m.execSQL(ctx, user, password, "", "SELECT 1;"); err == nil {
			return nil
		} else {
			last = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return fmt.Errorf("MySQL tidak siap: %w", last)
}

func (m *Manager) execSQL(ctx context.Context, user, password, database, sql string) error {
	args := []string{
		"--protocol=TCP",
		"--host=127.0.0.1",
		"--port=" + strconv.Itoa(m.Port),
		"--user=" + user,
		"--default-character-set=utf8mb4",
	}
	if database != "" {
		args = append(args, "--database="+database)
	}
	cmd := exec.CommandContext(ctx, m.mysqlExe(), args...)
	hidden.Hide(cmd)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+password)
	cmd.Stdin = strings.NewReader(sql)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Query MySQL gagal.", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (m *Manager) pipeSQLFile(ctx context.Context, user, password, database, file string) error {
	f, err := os.Open(file)
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "File SQL tidak ditemukan.", "")
	}
	defer f.Close()
	var reader io.Reader = f
	if strings.HasSuffix(strings.ToLower(file), ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "File gzip rusak.", "")
		}
		defer gz.Close()
		reader = gz
	}
	args := []string{
		"--protocol=TCP",
		"--host=127.0.0.1",
		"--port=" + strconv.Itoa(m.Port),
		"--user=" + user,
		"--database=" + database,
		"--default-character-set=utf8mb4",
	}
	cmd := exec.CommandContext(ctx, m.mysqlExe(), args...)
	hidden.Hide(cmd)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+password)
	cmd.Stdin = reader
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Restore SQL gagal.", strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (m *Manager) Spec() supervisor.Spec {
	return m.spec()
}

func (m *Manager) spec() supervisor.Spec {
	return supervisor.Spec{
		Name:       m.ServiceName,
		Executable: m.mysqldExe(),
		Args:       []string{"--defaults-file=" + m.ConfigPath},
		Dir:        m.Install.Path,
		Health:     supervisor.Health{TCPPort: m.Port, Timeout: 60 * time.Second},
		Policy:     supervisor.PolicyNo,
		Persistent: true,
	}
}

func (m *Manager) writeConfig() error {
	if err := os.MkdirAll(filepath.Dir(m.ConfigPath), 0o755); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder config MySQL tidak dapat dibuat.", "")
	}
	if err := os.MkdirAll(m.DataDir, 0o700); err != nil {
		return err
	}
	header := fmt.Sprintf(`# DevLok MySQL override untuk seri %s
# File ini dibaca lewat !include di my.ini.
# Letakkan setting di bawah [mysqld].
[mysqld]
# max_connections = 200
# sql_mode = STRICT_TRANS_TABLES,NO_ZERO_DATE,NO_ZERO_IN_DATE,ERROR_FOR_DIVISION_BY_ZERO
`, m.Version)
	if err := usercfg.Ensure(m.CustomConfigPath, header); err != nil {
		return err
	}
	ini := fmt.Sprintf(`[mysqld]
basedir="%s"
datadir="%s"
port=%d
bind-address=127.0.0.1
mysqlx=0
character-set-server=utf8mb4
collation-server=utf8mb4_unicode_ci
log-error="%s"
pid-file="%s"
max_connections=100

!include "%s"
`, filepath.ToSlash(m.Install.Path), filepath.ToSlash(m.DataDir), m.Port, filepath.ToSlash(m.LogPath), filepath.ToSlash(filepath.Join(m.DataDir, "mysqld.pid")), filepath.ToSlash(m.CustomConfigPath))
	if err := os.WriteFile(m.ConfigPath, []byte(ini), 0o644); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "my.ini tidak dapat ditulis.", "")
	}
	return nil
}

func (m *Manager) validateConfig() error {
	cmd := exec.Command(m.mysqldExe(), "--defaults-file="+m.ConfigPath, "--validate-config")
	hidden.Hide(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return devlokerr.New(devlokerr.CodeStateInconsistent, "Konfigurasi MySQL tidak valid.", msg)
	}
	return nil
}

func (m *Manager) loadState() (InstanceState, error) {
	data, err := os.ReadFile(m.statePath())
	if err != nil {
		return InstanceState{}, err
	}
	var st InstanceState
	if err := json.Unmarshal(data, &st); err != nil {
		return InstanceState{}, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "State MySQL rusak.", "")
	}
	return st, nil
}

func (m *Manager) saveState(st InstanceState) error {
	if err := os.MkdirAll(filepath.Dir(m.statePath()), 0o700); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder state MySQL tidak dapat dibuat.", "")
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(m.statePath(), data, 0o600)
}

func (m *Manager) loadCreds() (credentials, error) {
	data, err := os.ReadFile(m.credsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return credentials{Schema: credsSchema, Databases: map[string]DatabaseCreds{}}, nil
		}
		return credentials{}, err
	}
	plain, err := secret.Unseal(data)
	if err != nil {
		return credentials{}, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Kredensial MySQL tidak dapat dibuka.", "DPAPI hanya bisa dibuka oleh user yang membuatnya.")
	}
	var c credentials
	if err := json.Unmarshal(plain, &c); err != nil {
		return credentials{}, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Kredensial MySQL rusak.", "")
	}
	if c.Databases == nil {
		c.Databases = map[string]DatabaseCreds{}
	}
	return c, nil
}

func (m *Manager) saveCreds(c credentials) error {
	if err := os.MkdirAll(filepath.Dir(m.credsPath()), 0o700); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder kredensial tidak dapat dibuat.", "")
	}
	c.Schema = credsSchema
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	sealed, err := secret.Seal(data)
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeInternal, err, "Kredensial MySQL tidak dapat dienkripsi.", "")
	}
	return atomicWrite(m.credsPath(), sealed, 0o600)
}

func (m *Manager) statePath() string {
	return filepath.Join(m.StateDir, "state", "mysql-"+m.Version+".json")
}

func (m *Manager) credsPath() string {
	return filepath.Join(m.StateDir, "secrets", "mysql-"+m.Version+".dpapi")
}

func (m *Manager) servicesPath() string {
	return filepath.Join(m.StateDir, "state", "services.json")
}

func (m *Manager) mysqldExe() string {
	return filepath.Join(m.Install.Path, "bin", "mysqld"+platform.Current().ExeSuffix())
}

func (m *Manager) mysqlExe() string {
	return filepath.Join(m.Install.Path, "bin", "mysql"+platform.Current().ExeSuffix())
}

func (m *Manager) dumpExe() string {
	return filepath.Join(m.Install.Path, "bin", "mysqldump"+platform.Current().ExeSuffix())
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "File state tidak dapat ditulis.", "")
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "File state tidak dapat diselesaikan.", "")
	}
	return nil
}

func randomPassword(n int) (string, error) {
	const alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", devlokerr.Wrap(devlokerr.CodeInternal, err, "Password tidak dapat dibuat.", "")
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf), nil
}

func dbUser(project string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(project) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" || out[0] < 'a' || out[0] > 'z' {
		out = "db_" + out
	}
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}

func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func majorMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

func readTail(path string, lines int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimRight(string(data), "\r\n"), "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n")
}
