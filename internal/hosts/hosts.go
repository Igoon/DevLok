package hosts

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	StartMarker = "# >>> DevLok managed block >>>"
	EndMarker   = "# <<< DevLok managed block <<<"
)

type Manager struct {
	Path  string
	Flush func() error
}

type Result struct {
	Changed   bool     `json:"changed"`
	Added     []string `json:"added,omitempty"`
	Removed   []string `json:"removed,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
	Backup    string   `json:"backup,omitempty"`
}

func New(path string) *Manager {
	return &Manager{Path: path}
}

func (m *Manager) Apply(domains []string) (Result, error) {
	data, mode, err := m.read()
	if err != nil {
		return Result{}, err
	}
	text := string(data)
	body, block, err := stripBlock(text)
	if err != nil {
		return Result{}, err
	}
	existing := hostNames(body)
	wanted := uniqueSorted(domains)
	res := Result{}
	var lines []string
	for _, d := range wanted {
		if existing[d] {
			res.Conflicts = append(res.Conflicts, d)
			continue
		}
		lines = append(lines, "127.0.0.1 "+d)
		res.Added = append(res.Added, d)
	}
	if len(lines) == 0 && block == "" {
		return res, nil
	}
	if len(lines) > 0 && len(res.Conflicts) == 0 && sameDomains(block, lines) {
		return Result{}, nil
	}
	out := strings.TrimRight(body, "\r\n")
	if out != "" {
		out += "\n"
	}
	out += StartMarker + "\n" + strings.Join(lines, "\n")
	if len(lines) > 0 {
		out += "\n"
	}
	out += EndMarker + "\n"
	if block != "" && len(lines) == 0 {
		out = strings.TrimRight(body, "\r\n") + "\n"
	}
	out = toCRLFIfNeeded(text, out)
	backup, err := m.write(out, mode)
	if err != nil {
		return Result{}, err
	}
	res.Changed = true
	res.Backup = backup
	m.flush()
	return res, nil
}

func (m *Manager) Remove() (Result, error) {
	data, mode, err := m.read()
	if err != nil {
		return Result{}, err
	}
	text := string(data)
	body, block, err := stripBlock(text)
	if err != nil {
		return Result{}, err
	}
	if block == "" {
		return Result{}, nil
	}
	res := Result{Changed: true, Removed: sortedNames(hostNames(block))}
	out := strings.TrimRight(body, "\r\n")
	if out != "" {
		out += "\n"
	}
	out = toCRLFIfNeeded(text, out)
	backup, err := m.write(out, mode)
	if err != nil {
		return Result{}, err
	}
	res.Backup = backup
	m.flush()
	return res, nil
}

func (m *Manager) List() ([]string, error) {
	data, _, err := m.read()
	if err != nil {
		return nil, err
	}
	_, block, err := stripBlock(string(data))
	if err != nil {
		return nil, err
	}
	return sortedNames(hostNames(block)), nil
}

func (m *Manager) read() ([]byte, os.FileMode, error) {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0o644, nil
		}
		return nil, 0, err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(m.Path); err == nil {
		mode = info.Mode().Perm()
	}
	return data, mode, nil
}

func (m *Manager) write(content string, mode os.FileMode) (string, error) {
	backup := ""
	if _, err := os.Stat(m.Path); err == nil {
		backup = m.Path + ".devlok.bak"
		data, err := os.ReadFile(m.Path)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(backup, data, mode); err != nil {
			backup = ""
		}
	}
	tmp := m.Path + ".devlok.tmp"
	if err := os.WriteFile(tmp, []byte(content), mode); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, m.Path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return backup, nil
}

func (m *Manager) flush() {
	if m.Flush != nil {
		_ = m.Flush()
	}
}

func stripBlock(text string) (string, string, error) {
	lines := strings.Split(text, "\n")
	var body, block []string
	inBlock := false
	started := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == StartMarker:
			if inBlock {
				return "", "", fmt.Errorf("marker mulai ganda di hosts")
			}
			inBlock = true
			started = true
		case trimmed == EndMarker:
			if !inBlock {
				return "", "", fmt.Errorf("marker akhir tanpa awal di hosts")
			}
			inBlock = false
		case inBlock:
			block = append(block, line)
		default:
			body = append(body, line)
		}
	}
	if inBlock {
		return "", "", fmt.Errorf("blok DevLok tidak tertutup di hosts")
	}
	if !started {
		return strings.Join(body, "\n"), "", nil
	}
	return strings.Join(body, "\n"), strings.Join(block, "\n"), nil
}

func hostNames(text string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		for _, name := range fields[1:] {
			out[strings.ToLower(name)] = true
		}
	}
	return out
}

func uniqueSorted(in []string) []string {
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

func toCRLFIfNeeded(original, content string) string {
	if !strings.Contains(original, "\r\n") {
		return content
	}
	return strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\n", "\r\n")
}

func BackupPath(path string) string { return filepath.Clean(path + ".devlok.bak") }

func sortedNames(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sameDomains(block string, lines []string) bool {
	existing := sortedNames(hostNames(block))
	wanted := make([]string, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			wanted = append(wanted, strings.ToLower(fields[1]))
		}
	}
	sort.Strings(wanted)
	if len(existing) != len(wanted) {
		return false
	}
	for i := range existing {
		if existing[i] != wanted[i] {
			return false
		}
	}
	return true
}
