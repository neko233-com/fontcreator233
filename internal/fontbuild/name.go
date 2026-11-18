package fontbuild

import (
	"sort"
	"strings"
	"unicode"
)

func buildName(cfg Config) []byte {
	ps := postScriptName(cfg.Family, cfg.Style)
	licenseDescription := cfg.License
	if cfg.License == "OFL-1.1" {
		licenseDescription = "This Font Software is licensed under the SIL Open Font License, Version 1.1."
	}
	values := map[uint16]string{
		0:  cfg.Copyright,
		1:  cfg.Family,
		2:  cfg.Style,
		3:  cfg.Family + " " + cfg.Style + " " + cfg.Version + ";2026",
		4:  cfg.Family + " " + cfg.Style,
		5:  "Version " + cfg.Version,
		6:  ps,
		8:  "fontcreator233",
		9:  cfg.Designer,
		10: cfg.Description,
		11: "https://github.com/neko233-com/fontcreator233",
		13: licenseDescription,
		14: cfg.LicenseURL,
		16: cfg.Family,
		17: cfg.Style,
		19: cfg.SampleText,
	}
	if cfg.Designer == "" {
		values[9] = "fontcreator233"
	}
	ids := make([]int, 0, len(values))
	for id := range values {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	var records []uint16
	var storage []byte
	for _, id := range ids {
		v := values[uint16(id)]
		if v == "" {
			continue
		}
		enc := utf16be(v)
		if len(enc) > 65535 || len(storage) > 65535 {
			continue
		}
		records = append(records, uint16(id), uint16(len(enc)), uint16(len(storage)))
		storage = append(storage, enc...)
	}
	count := len(records) / 3
	stringOffset := 6 + 12*count
	out := make([]byte, stringOffset+len(storage))
	putU16(out, 0, 0)
	putU16(out, 2, uint16(count))
	putU16(out, 4, uint16(stringOffset))
	for i := 0; i < count; i++ {
		base := 6 + i*12
		putU16(out, base, 3)
		putU16(out, base+2, 1)
		putU16(out, base+4, 0x0409)
		putU16(out, base+6, records[i*3])
		putU16(out, base+8, records[i*3+1])
		putU16(out, base+10, records[i*3+2])
	}
	copy(out[stringOffset:], storage)
	return out
}

func postScriptName(family, style string) string {
	f := sanitizePS(family)
	s := sanitizePS(style)
	name := f
	if s != "" {
		name += "-" + s
	}
	name = strings.Trim(name, "-")
	if name == "" {
		name = "Font"
	}
	if len(name) > 63 {
		name = name[:63]
	}
	return name
}

func sanitizePS(s string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		switch {
		case r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			lastHyphen = false
		case r == '-':
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		case r == ' ' || r == '_':
			// PostScript names omit spaces and underscores.
		}
	}
	return strings.Trim(b.String(), "-")
}
