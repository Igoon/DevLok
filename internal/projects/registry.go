package projects

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"devlok/internal/config"
)

type Registry struct {
	Schema int      `json:"schema"`
	Paths  []string `json:"paths"`
}

func Load(stateDir string) (Registry, error) {
	data, err := os.ReadFile(path(stateDir))
	if err != nil {
		if os.IsNotExist(err) {
			return Registry{Schema: 1}, nil
		}
		return Registry{}, err
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return Registry{}, err
	}
	if r.Schema != 1 {
		return Registry{}, errors.New("schema registry project tidak didukung")
	}
	return r, nil
}

func Add(stateDir, projectDir string) error {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(abs, config.FileName)); err != nil {
		return err
	}
	r, err := Load(stateDir)
	if err != nil {
		return err
	}
	for _, p := range r.Paths {
		if samePath(p, abs) {
			return nil
		}
	}
	r.Paths = append(r.Paths, abs)
	sort.Slice(r.Paths, func(i, j int) bool { return strings.ToLower(r.Paths[i]) < strings.ToLower(r.Paths[j]) })
	return save(stateDir, r)
}

func Remove(stateDir, projectDir string) error {
	abs, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}
	r, err := Load(stateDir)
	if err != nil {
		return err
	}
	out := r.Paths[:0]
	for _, p := range r.Paths {
		if !samePath(p, abs) {
			out = append(out, p)
		}
	}
	r.Paths = out
	return save(stateDir, r)
}

func List(stateDir string) ([]string, error) {
	r, err := Load(stateDir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(r.Paths))
	for _, p := range r.Paths {
		if _, err := os.Stat(filepath.Join(p, config.FileName)); err == nil {
			out = append(out, p)
		}
	}
	return out, nil
}

func save(stateDir string, r Registry) error {
	if err := os.MkdirAll(filepath.Join(stateDir, "state"), 0o700); err != nil {
		return err
	}
	r.Schema = 1
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	target := path(stateDir)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	_ = os.Remove(target)
	return os.Rename(tmp, target)
}

func path(stateDir string) string {
	return filepath.Join(stateDir, "state", "projects.json")
}

func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}
