package config

func ApplyDefaults(m *Manifest) {
	if m.Schema == 0 {
		m.Schema = SchemaVersion
	}
	if m.Web.Server == "" {
		m.Web.Server = "nginx"
	}
	if m.Web.TLD == "" {
		m.Web.TLD = "test"
	}
	if m.Web.HTTPS == nil {
		v := true
		m.Web.HTTPS = &v
	}
	if len(m.Services) == 0 {
		m.Services = map[string]string{"mysql": "8.4"}
	}
	if m.Web.PHP == "" {
		if v, ok := m.Runtime["php"]; ok && v != "" {
			m.Web.PHP = v
		} else {
			m.Web.PHP = "8.4"
		}
	}
	if m.DB.Name == "" {
		m.DB.Name = dbSafeName(m.Name)
	}
	if m.DB.Charset == "" {
		m.DB.Charset = "utf8mb4"
	}
	if m.DB.Collation == "" {
		m.DB.Collation = "utf8mb4_unicode_ci"
	}
}

func dbSafeName(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == '-' && len(out) > 0:
			out = append(out, '_')
		}
	}
	if len(out) == 0 || out[0] < 'a' || out[0] > 'z' {
		return "app_" + string(out)
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return string(out)
}
