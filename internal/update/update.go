package update

import (
	"sort"
	"strconv"
	"strings"

	rt "devlok/internal/runtime"
)

type RuntimeUpdate struct {
	Name      string   `json:"name"`
	Installed string   `json:"installed"`
	Latest    string   `json:"latest"`
	Newer     []string `json:"newer"`
}

func Check(mgr *rt.Manager) ([]RuntimeUpdate, error) {
	installed, err := mgr.ListInstalled()
	if err != nil {
		return nil, err
	}
	highest := map[string]string{}
	for _, in := range installed {
		if highest[in.Name] == "" || Compare(in.Version, highest[in.Name]) > 0 {
			highest[in.Name] = in.Version
		}
	}
	key := mgr.OS + "/" + mgr.Arch
	var out []RuntimeUpdate
	for _, reg := range mgr.Registries {
		current := highest[reg.Name]
		if current == "" {
			continue
		}
		var newer []string
		latest := ""
		for _, v := range reg.Versions {
			if _, ok := v.Artifacts[key]; !ok {
				continue
			}
			if Compare(v.Version, current) > 0 {
				newer = append(newer, v.Version)
			}
			if latest == "" || Compare(v.Version, latest) > 0 {
				latest = v.Version
			}
		}
		if len(newer) == 0 {
			continue
		}
		sort.Slice(newer, func(i, j int) bool { return Compare(newer[i], newer[j]) < 0 })
		out = append(out, RuntimeUpdate{
			Name:      reg.Name,
			Installed: current,
			Latest:    latest,
			Newer:     newer,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func Compare(a, b string) int {
	as := strings.Split(strings.TrimSpace(a), ".")
	bs := strings.Split(strings.TrimSpace(b), ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av, _ = strconv.Atoi(numeric(as[i]))
		}
		if i < len(bs) {
			bv, _ = strconv.Atoi(numeric(bs[i]))
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func numeric(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		break
	}
	if b.Len() == 0 {
		return "0"
	}
	return b.String()
}
