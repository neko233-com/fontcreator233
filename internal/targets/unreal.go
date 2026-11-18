package targets

import (
	"embed"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neko233-com/fontcreator233/assets"
	"github.com/neko233-com/fontcreator233/internal/integration"
)

//go:embed unreal/*
var unrealFS embed.FS

func buildUnreal(o Options, root *integration.Manifest) ([]string, error) {
	dir := filepath.Join(o.OutDir, "unreal")
	fontsDir := filepath.Join(dir, "Fonts")
	scriptsDir := filepath.Join(dir, "Scripts")
	charsetDir := filepath.Join(dir, "Charsets")
	for _, d := range []string{dir, fontsDir, scriptsDir, charsetDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	var fontPaths []string
	manifestFonts := make([]integration.Font, 0, len(o.Fonts))
	fallbacks := make([]string, 0, len(o.Fonts))
	var defaultFont string
	for i, f := range o.Fonts {
		base := filepath.Base(f.Path)
		dst := filepath.Join(fontsDir, base)
		if err := copyFile(f.Path, dst); err != nil {
			return nil, err
		}
		fontPaths = append(fontPaths, dst)
		rel := "Fonts/" + base
		manifestFonts = append(manifestFonts, integration.Font{
			Path:    rel,
			SHA256:  f.SHA256,
			Primary: f.Primary || i == 0,
			Scripts: integration.Scripts(f.Runes),
			Ranges:  integration.Ranges(f.Runes),
		})
		fallbacks = append(fallbacks, rel)
		if i == 0 {
			defaultFont = base
		}
	}
	var allRunes []rune
	for _, f := range o.Fonts {
		allRunes = append(allRunes, f.Runes...)
	}
	charsets := integration.Charsets(allRunes)
	all := append([]rune(nil), allRunes...)
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	charsets["all"] = string(all)
	charsetNames := make([]string, 0, len(charsets))
	for script, text := range charsets {
		name := "Charsets/" + script + ".txt"
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
			return nil, err
		}
		charsetNames = append(charsetNames, name)
	}
	sort.Strings(charsetNames)

	m := *root
	licenseFile := ""
	if o.License == "OFL-1.1" {
		licenseFile = "OFL.txt"
	}
	m.LicenseFile = licenseFile
	m.Fonts = manifestFonts
	m.Fallbacks = fallbacks
	m.Charsets = charsetNames
	m.Web = integration.Web{CSS: "../web/font.css", FontDisplay: "swap"}
	m.UnrealComposite = integration.Unreal{Default: "Fonts/" + defaultFont}
	for i, f := range manifestFonts {
		if i == 0 {
			continue
		}
		m.UnrealComposite.SubFonts = append(m.UnrealComposite.SubFonts, integration.Composite{
			Path:   f.Path,
			Ranges: f.Ranges,
		})
	}
	if err := integration.WriteJSON(filepath.Join(dir, "fontcreator233.manifest.json"), &m); err != nil {
		return nil, err
	}

	replace := func(b []byte, pairs ...string) []byte {
		s := string(b)
		for i := 0; i+1 < len(pairs); i += 2 {
			s = strings.ReplaceAll(s, pairs[i], pairs[i+1])
		}
		return []byte(s)
	}
	readme, err := unrealFS.ReadFile("unreal/README.md.tmpl")
	if err != nil {
		return nil, err
	}
	readme = replace(readme, "__DISPLAY_NAME__", o.Family+" "+o.Style, "__FAMILY__", o.Family, "__DEFAULT_FONT__", "Fonts/"+defaultFont, "__LICENSE__", o.License)
	script, err := unrealFS.ReadFile("unreal/fontcreator233_import.py.tmpl")
	if err != nil {
		return nil, err
	}
	script = replace(script, "__FAMILY__", o.Family)
	readmePath := filepath.Join(dir, "README.md")
	scriptPath := filepath.Join(scriptsDir, "fontcreator233_import.py")
	if err := os.WriteFile(readmePath, readme, 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(scriptPath, script, 0o644); err != nil {
		return nil, err
	}
	paths := []string{
		readmePath,
		scriptPath,
		filepath.Join(dir, "fontcreator233.manifest.json"),
	}
	if licenseFile != "" {
		oflPath := filepath.Join(dir, licenseFile)
		if err := os.WriteFile(oflPath, assets.OFL, 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, oflPath)
	}
	paths = append(paths, fontPaths...)
	for _, name := range charsetNames {
		paths = append(paths, filepath.Join(dir, filepath.FromSlash(name)))
	}
	return paths, nil
}
