package supervisor

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
	"devlok/internal/ports"
)

type Policy string

const (
	PolicyNo        Policy = "no"
	PolicyOnFailure Policy = "on-failure"
	PolicyAlways    Policy = "always"
)

type State string

const (
	StateRunning    State = "running"
	StateStopped    State = "stopped"
	StateRestarting State = "restarting"
	StateFailed     State = "failed"
)

type Health struct {
	TCPPort  int           `json:"tcp_port,omitempty"`
	HTTPURL  string        `json:"http_url,omitempty"`
	Timeout  time.Duration `json:"timeout,omitempty"`
	Interval time.Duration `json:"interval,omitempty"`
}

type Spec struct {
	Name        string        `json:"name"`
	Executable  string        `json:"executable"`
	Args        []string      `json:"args,omitempty"`
	Dir         string        `json:"dir,omitempty"`
	Env         []string      `json:"env,omitempty"`
	Health      Health        `json:"health,omitempty"`
	Policy      Policy        `json:"policy,omitempty"`
	MaxRestarts int           `json:"max_restarts,omitempty"`
	Backoff     time.Duration `json:"backoff,omitempty"`
	Persistent  bool          `json:"persistent,omitempty"`
}

type Status struct {
	Name       string    `json:"name"`
	PID        int       `json:"pid"`
	State      State     `json:"state"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	Restarts   int       `json:"restarts"`
	LastError  string    `json:"last_error,omitempty"`
	Policy     Policy    `json:"policy,omitempty"`
	Port       int       `json:"port,omitempty"`
	Executable string    `json:"executable,omitempty"`
}

type fileState struct {
	Schema   int      `json:"schema"`
	Services []Status `json:"services"`
}

type proc struct {
	cmd      *exec.Cmd
	guard    *procHandle
	spec     Spec
	status   Status
	done     chan struct{}
	stopping bool
	restarts int
}

type Manager struct {
	mu     sync.Mutex
	path   string
	closed bool
	procs  map[string]*proc
	orphan map[string]Status
}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

func NewManager(statePath string) (*Manager, error) {
	if statePath == "" {
		return nil, devlokerr.New(devlokerr.CodeInvalidArgument, "Path state supervisor wajib diisi.", "")
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder state supervisor tidak dapat dibuat.", "")
	}
	m := &Manager{
		path:   statePath,
		procs:  map[string]*proc{},
		orphan: map[string]Status{},
	}
	if err := m.load(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) Start(spec Spec) (Status, error) {
	spec = withDefaults(spec)
	if err := validateSpec(spec); err != nil {
		return Status{}, err
	}
	m.mu.Lock()
	if st, ok := m.statusLocked(spec.Name); ok && (st.State == StateRunning || st.State == StateRestarting) {
		m.mu.Unlock()
		return st, devlokerr.New(devlokerr.CodeStateInconsistent, "Service sudah berjalan: "+spec.Name, "Hentikan service terlebih dahulu.")
	}
	p, err := m.launchLocked(spec, 0)
	if err != nil {
		m.mu.Unlock()
		return Status{}, err
	}
	m.mu.Unlock()

	go m.watch(spec.Name, p)

	if p.spec.Health.TCPPort != 0 || p.spec.Health.HTTPURL != "" {
		if err := m.waitReady(spec.Name, p); err != nil {
			_ = m.Stop(spec.Name)
			return Status{Name: spec.Name, State: StateFailed, LastError: err.Error()}, err
		}
	}
	st, _ := m.Get(spec.Name)
	return st, nil
}

func (m *Manager) launchLocked(spec Spec, restarts int) (*proc, error) {
	guard, err := newProcHandle(spec.Persistent)
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Job object tidak dapat dibuat.", "Periksa versi Windows dan hak akses.")
	}
	cmd := exec.Command(spec.Executable, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = append(os.Environ(), spec.Env...)
	if err := guard.beforeStart(cmd); err != nil {
		_ = guard.close()
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "Proses tidak dapat disiapkan.", "")
	}
	if err := cmd.Start(); err != nil {
		_ = guard.close()
		return nil, devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Service tidak dapat dijalankan: "+spec.Name, "Periksa executable dan dependency.")
	}
	if err := guard.afterStart(cmd.Process.Pid); err != nil {
		_ = guard.kill()
		_ = guard.close()
		_ = cmd.Wait()
		return nil, devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Proses tidak dapat dimasukkan ke Job Object.", "")
	}
	now := time.Now().UTC()
	st := Status{
		Name:       spec.Name,
		PID:        cmd.Process.Pid,
		State:      StateRunning,
		StartedAt:  now,
		Restarts:   restarts,
		Policy:     spec.Policy,
		Port:       healthPort(spec.Health),
		Executable: spec.Executable,
	}
	p := &proc{
		cmd:      cmd,
		guard:    guard,
		spec:     spec,
		status:   st,
		done:     make(chan struct{}),
		restarts: restarts,
	}
	delete(m.orphan, spec.Name)
	m.procs[spec.Name] = p
	if err := m.saveLocked(); err != nil {
		_ = guard.kill()
		_ = guard.close()
		_ = cmd.Wait()
		delete(m.procs, spec.Name)
		return nil, err
	}
	return p, nil
}

func (m *Manager) watch(name string, p *proc) {
	err := p.cmd.Wait()
	_ = p.guard.close()

	m.mu.Lock()
	if m.procs[name] != p {
		m.mu.Unlock()
		close(p.done)
		return
	}
	p.status.PID = 0
	if p.stopping || m.closed {
		p.status.State = StateStopped
		p.status.LastError = ""
		_ = m.saveLocked()
		m.mu.Unlock()
		close(p.done)
		return
	}
	if err != nil {
		p.status.LastError = err.Error()
	}
	restart := p.spec.Policy == PolicyAlways || (p.spec.Policy == PolicyOnFailure && err != nil)
	if restart && p.restarts < p.spec.MaxRestarts {
		p.restarts++
		p.status.Restarts = p.restarts
		p.status.State = StateRestarting
		_ = m.saveLocked()
		m.mu.Unlock()
		close(p.done)
		go m.restartAfter(name, p)
		return
	}
	p.status.State = StateFailed
	_ = m.saveLocked()
	m.mu.Unlock()
	close(p.done)
}

func (m *Manager) restartAfter(name string, old *proc) {
	time.Sleep(old.spec.Backoff)
	m.mu.Lock()
	if m.closed || old.stopping || m.procs[name] != old {
		if old.stopping {
			old.status.State = StateStopped
			_ = m.saveLocked()
		}
		m.mu.Unlock()
		return
	}
	p, err := m.launchLocked(old.spec, old.restarts)
	if err != nil {
		old.status.State = StateFailed
		old.status.LastError = err.Error()
		_ = m.saveLocked()
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	go m.watch(name, p)
}

func (m *Manager) waitReady(name string, p *proc) error {
	timeout := p.spec.Health.Timeout
	interval := p.spec.Health.Interval
	deadline := time.Now().Add(timeout)
	for {
		if healthOK(p.spec.Health) {
			return nil
		}
		st, _ := m.Get(name)
		if st.State != StateRunning {
			return devlokerr.New(devlokerr.CodeDependencyMissing, "Service berhenti sebelum siap: "+name, st.LastError)
		}
		select {
		case <-p.done:
			st, _ := m.Get(name)
			return devlokerr.New(devlokerr.CodeDependencyMissing, "Service berhenti sebelum siap: "+name, st.LastError)
		default:
		}
		if time.Now().After(deadline) {
			return devlokerr.New(devlokerr.CodeTimeout, "Service tidak siap sebelum timeout: "+name, fmt.Sprintf("Naikkan Health.Timeout (saat ini %s).", timeout))
		}
		time.Sleep(interval)
	}
}

func healthPort(h Health) int {
	if h.TCPPort > 0 {
		return h.TCPPort
	}
	if h.HTTPURL == "" {
		return 0
	}
	u, err := url.Parse(h.HTTPURL)
	if err != nil {
		return 0
	}
	if p := u.Port(); p != "" {
		n, _ := strconv.Atoi(p)
		return n
	}
	if u.Scheme == "https" {
		return 443
	}
	return 80
}

func HealthOK(h Health) bool {
	return healthOK(h)
}

func healthOK(h Health) bool {
	if h.HTTPURL != "" {
		client := &http.Client{
			Timeout:       2 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
		resp, err := client.Get(h.HTTPURL)
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode >= 200 && resp.StatusCode < 400
	}
	if h.TCPPort > 0 {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", h.TCPPort), 700*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}
	return true
}

func (m *Manager) Stop(name string) error {
	if !nameRe.MatchString(name) {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Nama service tidak valid: "+name, "")
	}
	m.mu.Lock()
	p := m.procs[name]
	if p == nil {
		st, ok := m.orphan[name]
		if !ok || st.State != StateRunning || st.PID <= 0 {
			m.mu.Unlock()
			return nil
		}
		m.mu.Unlock()
		if err := stopOrphan(st); err != nil {
			return err
		}
		m.mu.Lock()
		st.State = StateStopped
		st.PID = 0
		m.orphan[name] = st
		err := m.saveLocked()
		m.mu.Unlock()
		return err
	}
	if p.status.State == StateStopped || p.status.State == StateFailed {
		m.mu.Unlock()
		return nil
	}
	p.stopping = true
	if p.status.State == StateRestarting {
		p.status.State = StateStopped
		_ = m.saveLocked()
	}
	guard := p.guard
	done := p.done
	pid := p.status.PID
	m.mu.Unlock()

	if err := guard.kill(); err != nil {
		if pid > 0 {
			_ = platform.Current().KillTree(pid)
		}
	}
	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		if pid > 0 {
			_ = platform.Current().KillTree(pid)
		}
		select {
		case <-done:
			return nil
		case <-time.After(3 * time.Second):
			return devlokerr.New(devlokerr.CodeTimeout, "Service tidak berhenti: "+name, "Periksa proses yang masih berjalan.")
		}
	}
}

func (m *Manager) Restart(spec Spec) (Status, error) {
	if err := m.Stop(spec.Name); err != nil {
		return Status{}, err
	}
	return m.Start(spec)
}

func (m *Manager) Get(name string) (Status, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked(name)
}

func (m *Manager) statusLocked(name string) (Status, bool) {
	if p, ok := m.procs[name]; ok {
		return p.status, true
	}
	st, ok := m.orphan[name]
	return st, ok
}

func (m *Manager) List() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.listLocked()
}

func (m *Manager) Reconcile() {
	m.mu.Lock()
	defer m.mu.Unlock()
	changed := false
	for name, st := range m.orphan {
		if st.State != StateRunning && st.State != StateRestarting {
			continue
		}
		alive := false
		if st.PID > 0 {
			started, err := platform.Current().ProcessStartTime(st.PID)
			if err == nil && absDuration(started.Sub(st.StartedAt)) <= 3*time.Second {
				alive = true
			}
		}
		if !alive {
			st.State = StateStopped
			st.PID = 0
			m.orphan[name] = st
			changed = true
		}
	}
	if changed {
		_ = m.saveLocked()
	}
}

func (m *Manager) listLocked() []Status {
	seen := map[string]bool{}
	out := make([]Status, 0, len(m.procs)+len(m.orphan))
	for name, p := range m.procs {
		out = append(out, p.status)
		seen[name] = true
	}
	for name, st := range m.orphan {
		if !seen[name] {
			out = append(out, st)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func stopOrphan(st Status) error {
	if st.PID > 0 {
		_ = platform.Current().KillTree(st.PID)
	}
	if st.Port <= 0 {
		return nil
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		ps := ports.Check(st.Port)
		if ps.Available {
			return nil
		}
		if ps.PID > 0 {
			expected := st.Executable != "" && sameFile(ps.Path, st.Executable)
			if expected || ps.PID == st.PID {
				_ = platform.Current().KillTree(ps.PID)
			} else {
				return devlokerr.New(devlokerr.CodeStateInconsistent,
					fmt.Sprintf("Port %d dipakai proses lain: %s (PID %d).", st.Port, ps.Process, ps.PID),
					"Tutup proses tersebut secara manual.")
			}
		}
		if time.Now().After(deadline) {
			return devlokerr.New(devlokerr.CodeStateInconsistent, fmt.Sprintf("Service %s tidak berhenti; port %d masih dipakai.", st.Name, st.Port), "Periksa dl ports.")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func sameFile(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func (m *Manager) Close() error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	procs := make([]*proc, 0, len(m.procs))
	for _, p := range m.procs {
		procs = append(procs, p)
	}
	m.mu.Unlock()

	for _, p := range procs {
		_ = p.guard.kill()
	}
	deadline := time.After(10 * time.Second)
	for _, p := range procs {
		select {
		case <-p.done:
		case <-deadline:
			return devlokerr.New(devlokerr.CodeTimeout, "Sebagian service tidak berhenti saat close.", "")
		}
	}
	return nil
}

func withDefaults(spec Spec) Spec {
	if spec.Policy == "" {
		spec.Policy = PolicyOnFailure
	}
	if spec.MaxRestarts == 0 {
		spec.MaxRestarts = 3
	}
	if spec.Backoff <= 0 {
		spec.Backoff = 2 * time.Second
	}
	if spec.Health.Timeout <= 0 {
		spec.Health.Timeout = 15 * time.Second
	}
	if spec.Health.Interval <= 0 {
		spec.Health.Interval = 200 * time.Millisecond
	}
	return spec
}

func validateSpec(spec Spec) error {
	if !nameRe.MatchString(spec.Name) {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Nama service tidak valid: "+spec.Name, "Gunakan huruf kecil, angka, dan tanda hubung.")
	}
	if spec.Executable == "" {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Executable service kosong.", "")
	}
	info, err := os.Stat(spec.Executable)
	if err != nil || info.IsDir() {
		return devlokerr.New(devlokerr.CodeDependencyMissing, "Executable tidak ditemukan: "+spec.Executable, "Pasang runtime terlebih dahulu.")
	}
	if spec.Health.HTTPURL != "" {
		u, err := url.Parse(spec.Health.HTTPURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return devlokerr.New(devlokerr.CodeInvalidArgument, "Health URL tidak valid: "+spec.Health.HTTPURL, "")
		}
	}
	if spec.Health.TCPPort < 0 || spec.Health.TCPPort > 65535 {
		return devlokerr.New(devlokerr.CodeInvalidArgument, "Health port tidak valid.", "")
	}
	return nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "State supervisor tidak dapat dibaca.", "")
	}
	if len(data) > 1<<20 {
		return devlokerr.New(devlokerr.CodeStateInconsistent, "State supervisor terlalu besar.", "")
	}
	var fs fileState
	if err := json.Unmarshal(data, &fs); err != nil {
		return devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "State supervisor rusak.", "Hapus file state atau jalankan dl doctor.")
	}
	if fs.Schema != 1 {
		return devlokerr.New(devlokerr.CodeStateInconsistent, "Schema state supervisor tidak didukung.", "")
	}
	plat := platform.Current()
	for _, st := range fs.Services {
		if st.Name == "" {
			continue
		}
		if st.State == StateRunning || st.State == StateRestarting {
			alive := false
			if st.PID > 0 {
				started, err := plat.ProcessStartTime(st.PID)
				if err == nil && absDuration(started.Sub(st.StartedAt)) <= 3*time.Second {
					alive = true
				}
			}
			if !alive {
				st.State = StateStopped
				st.PID = 0
			}
		}
		m.orphan[st.Name] = st
	}
	return nil
}

func (m *Manager) saveLocked() error {
	fs := fileState{Schema: 1, Services: m.listLocked()}
	data, err := json.MarshalIndent(fs, "", "  ")
	if err != nil {
		return devlokerr.Wrap(devlokerr.CodeInternal, err, "State supervisor tidak dapat diserialisasi.", "")
	}
	data = append(data, '\n')
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "State supervisor tidak dapat ditulis.", "")
	}
	_ = os.Remove(m.path)
	if err := os.Rename(tmp, m.path); err != nil {
		return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "State supervisor tidak dapat dipindahkan.", "")
	}
	return nil
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
