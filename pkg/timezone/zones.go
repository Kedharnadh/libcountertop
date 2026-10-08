package tz

type Zone struct {
	Name string

	Spec string
}

// America/Vancouver is absent: tzdata puts British Columbia on permanent MST from 2026.
func Zones() []Zone {
	return []Zone{
		{"UTC", "UTC0"},

		{"Europe/London", "GMT0BST,M3.5.0/1,M10.5.0"},
		{"Europe/Lisbon", "WET0WEST,M3.5.0/1,M10.5.0"},
		{"Europe/Paris", "CET-1CEST,M3.5.0,M10.5.0/3"},
		{"Europe/Berlin", "CET-1CEST,M3.5.0,M10.5.0/3"},
		{"Europe/Madrid", "CET-1CEST,M3.5.0,M10.5.0/3"},
		{"Europe/Rome", "CET-1CEST,M3.5.0,M10.5.0/3"},
		{"Europe/Amsterdam", "CET-1CEST,M3.5.0,M10.5.0/3"},
		{"Europe/Warsaw", "CET-1CEST,M3.5.0,M10.5.0/3"},
		{"Europe/Athens", "EET-2EEST,M3.5.0/3,M10.5.0/4"},
		{"Europe/Helsinki", "EET-2EEST,M3.5.0/3,M10.5.0/4"},
		{"Europe/Moscow", "MSK-3"},

		{"America/New_York", "EST5EDT,M3.2.0,M11.1.0"},
		{"America/Chicago", "CST6CDT,M3.2.0,M11.1.0"},
		{"America/Denver", "MST7MDT,M3.2.0,M11.1.0"},
		{"America/Phoenix", "MST7"},
		{"America/Los_Angeles", "PST8PDT,M3.2.0,M11.1.0"},
		{"America/Anchorage", "AKST9AKDT,M3.2.0,M11.1.0"},
		{"Pacific/Honolulu", "HST10"},
		{"America/Toronto", "EST5EDT,M3.2.0,M11.1.0"},
		{"America/Mexico_City", "CST6"},
		{"America/Sao_Paulo", "<-03>3"},

		{"Asia/Jerusalem", "IST-2IDT,M3.4.4/26,M10.5.0"},
		{"Asia/Dubai", "<+04>-4"},
		{"Asia/Kolkata", "IST-5:30"},
		{"Asia/Bangkok", "<+07>-7"},
		{"Asia/Shanghai", "CST-8"},
		{"Asia/Singapore", "<+08>-8"},
		{"Asia/Hong_Kong", "HKT-8"},
		{"Asia/Tokyo", "JST-9"},
		{"Asia/Seoul", "KST-9"},

		{"Australia/Perth", "AWST-8"},
		{"Australia/Brisbane", "AEST-10"},
		{"Australia/Sydney", "AEST-10AEDT,M10.1.0,M4.1.0/3"},
		{"Australia/Adelaide", "ACST-9:30ACDT,M10.1.0,M4.1.0/3"},
		{"Pacific/Auckland", "NZST-12NZDT,M9.5.0,M4.1.0/3"},

		{"Africa/Johannesburg", "SAST-2"},
		{"Africa/Lagos", "WAT-1"},
		{"Africa/Cairo", "EET-2EEST,M4.5.5/0,M10.5.4/24"},
	}
}

func Region(name string) string {
	for i := range name {
		if name[i] == '/' {
			return name[:i]
		}
	}
	return ""
}

func Regions() []string {
	var out []string
	seen := map[string]bool{}

	for _, z := range Zones() {
		r := Region(z.Name)
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

func In(region string) []Zone {
	var out []Zone

	for _, z := range Zones() {
		if Region(z.Name) == region {
			out = append(out, z)
		}
	}
	return out
}

func Named(name string) (Zone, bool) {
	for _, z := range Zones() {
		if z.Name == name {
			return z, true
		}
	}
	return Zone{}, false
}

func Names() []string {
	zones := Zones()

	out := make([]string, 0, len(zones))
	for _, z := range zones {
		out = append(out, z.Name)
	}
	return out
}
