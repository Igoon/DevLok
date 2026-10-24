package update

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"devlok/internal/brew"
	"devlok/internal/platform"
	rt "devlok/internal/runtime"
)

const (
	KindPatch = "patch"
	KindNew   = "new"
)

type RuntimeUpdate struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Series    string   `json:"series,omitempty"`
	Installed string   `json:"installed"`
	Latest    string   `json:"latest"`
	Newer     []string `json:"newer"`
}

func Check(mgr *rt.Manager) ([]RuntimeUpdate, error) {
	if platform.Current().Name() == "darwin" {
		return brewRuntimeUpdates()
	}
	installed, err := mgr.ListInstalled()
	if err != nil {
		return nil, err
	}
	return checkInstalled(installed, mgr.Registries, mgr.OS+"/"+mgr.Arch), nil
}

func brewRuntimeUpdates() ([]RuntimeUpdate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	list, err := brew.Outdated(ctx)
	if err != nil {
		return nil, nil
	}
	var out []RuntimeUpdate
	for _, o := range list {
		out = append(out, RuntimeUpdate{
			Name:      o.Name,
			Kind:      KindPatch,
			Series:    versionSeries(o.Installed),
			Installed: o.Installed,
			Latest:    o.Latest,
			Newer:     []string{o.Latest},
		})
	}
	return out, nil
}

func checkInstalled(installed []rt.Installed, regs []rt.Registry, platform string) []RuntimeUpdate {
	byName := map[string][]string{}
	for _, in := range installed {
		byName[in.Name] = append(byName[in.Name], in.Version)
	}
	var out []RuntimeUpdate
	for _, reg := range regs {
		current := byName[reg.Name]
		if len(current) == 0 {
			continue
		}
		var available []string
		for _, v := range reg.Versions {
			if _, ok := v.Artifacts[platform]; ok {
				available = append(available, v.Version)
			}
		}
		if len(available) == 0 {
			continue
		}
		sort.Slice(current, func(i, j int) bool { return Compare(current[i], current[j]) > 0 })
		highestBySeries := map[string]string{}
		for _, version := range current {
			series := versionSeries(version)
			if have, ok := highestBySeries[series]; !ok || Compare(version, have) > 0 {
				highestBySeries[series] = version
			}
		}
		seriesList := make([]string, 0, len(highestBySeries))
		for series := range highestBySeries {
			seriesList = append(seriesList, series)
		}
		sort.Slice(seriesList, func(i, j int) bool { return Compare(seriesList[i], seriesList[j]) > 0 })
		for _, series := range seriesList {
			version := highestBySeries[series]
			var newer []string
			latest := ""
			for _, v := range available {
				if versionSeries(v) != series || Compare(v, version) <= 0 {
					continue
				}
				newer = append(newer, v)
				if latest == "" || Compare(v, latest) > 0 {
					latest = v
				}
			}
			if latest == "" {
				continue
			}
			sort.Slice(newer, func(i, j int) bool { return Compare(newer[i], newer[j]) < 0 })
			out = append(out, RuntimeUpdate{
				Name:      reg.Name,
				Kind:      KindPatch,
				Series:    series,
				Installed: version,
				Latest:    latest,
				Newer:     newer,
			})
		}
		highestInstalled := current[0]
		latest := ""
		for _, v := range available {
			if latest == "" || Compare(v, latest) > 0 {
				latest = v
			}
		}
		if latest != "" && Compare(versionSeries(latest), versionSeries(highestInstalled)) > 0 {
			out = append(out, RuntimeUpdate{
				Name:      reg.Name,
				Kind:      KindNew,
				Series:    versionSeries(latest),
				Installed: highestInstalled,
				Latest:    latest,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind == KindPatch
		}
		return Compare(out[i].Installed, out[j].Installed) > 0
	})
	return out
}

func SameSeries(a, b string) bool {
	return versionSeries(a) == versionSeries(b)
}

func versionSeries(v string) string {
	base, _ := versionParts(v)
	parts := strings.Split(base, ".")
	if len(parts) <= 2 {
		return base
	}
	return parts[0] + "." + parts[1]
}

func Compare(a, b string) int {
	baseA, preA := versionParts(a)
	baseB, preB := versionParts(b)
	n := compareBase(baseA, baseB)
	if n != 0 {
		return n
	}
	if preA == preB {
		return 0
	}
	if preA == "" {
		return 1
	}
	if preB == "" {
		return -1
	}
	return strings.Compare(preA, preB)
}

func versionParts(v string) (string, string) {
	v = strings.TrimSpace(v)
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

func compareBase(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
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
