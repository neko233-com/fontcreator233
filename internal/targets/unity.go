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

//go:embed unity/*
var unityFS embed.FS

func buildUnity(o Options, root *integration.Manifest) ([]string, error) {
	dir := filepath.Join(o.OutDir, "unity")
	fontsDir := filepath.Join(dir, "Fonts")
	charsetDir := filepath.Join(dir, "Charsets")
	editorDir := filepath.Join(dir, "Editor")
	for _, d := range []string{dir, fontsDir, charsetDir, editorDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	slug := packageSlug(o.Family)
	packageName := "com.neko233.fontcreator233.generated." + slug
	display := o.Family + " " + o.Style + " (fontcreator233)"
	version := o.Version
	if version == "" {
		version = "1.0.0"
	}

	replace := func(b []byte, pairs ...string) []byte {
		s := string(b)
		for i := 0; i+1 < len(pairs); i += 2 {
			s = strings.ReplaceAll(s, pairs[i], pairs[i+1])
		}
		return []byte(s)
	}
	pkg, err := unityFS.ReadFile("unity/package.json.tmpl")
	if err != nil {
		return nil, err
	}
	pkg = replace(pkg, "__PACKAGE_NAME__", packageName, "__VERSION__", version, "__DISPLAY_NAME__", display)
	readme, err := unityFS.ReadFile("unity/README.md.tmpl")
	if err != nil {
		return nil, err
	}
	readme = replace(readme, "__PACKAGE_NAME__", packageName, "__DISPLAY_NAME__", display, "__FAMILY__", o.Family, "__STYLE__", o.Style, "__LICENSE__", o.License)
	editor, err := unityFS.ReadFile("unity/FontCreator233TMPImporter.cs.tmpl")
	if err != nil {
		return nil, err
	}
	editor = replace(editor, "__PACKAGE_NAME__", packageName)

	files := map[string][]byte{
		filepath.Join(dir, "package.json"):                       pkg,
		filepath.Join(dir, "README.md"):                          readme,
		filepath.Join(editorDir, "FontCreator233TMPImporter.cs"): editor,
	}
	licenseFile := ""
	if o.License == "OFL-1.1" {
		licenseFile = "OFL.txt"
		files[filepath.Join(dir, "OFL.txt")] = assets.OFL
	}
	for path, data := range files {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, err
		}
	}
	var fontPaths []string
	manifestFonts := make([]integration.Font, 0, len(o.Fonts))
	fallbacks := make([]string, 0, len(o.Fonts))
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
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return nil, err
		}
		charsetNames = append(charsetNames, name)
	}
	sort.Strings(charsetNames)

	m := *root
	m.LicenseFile = licenseFile
	m.Fonts = manifestFonts
	m.Fallbacks = fallbacks
	m.Charsets = charsetNames
	m.UnityTMP.StaticCharsets = []string{"Charsets/all.txt"}
	m.Web = integration.Web{CSS: "../web/font.css", FontDisplay: "swap"}
	if err := integration.WriteJSON(filepath.Join(dir, "fontcreator233.manifest.json"), &m); err != nil {
		return nil, err
	}

	paths := []string{
		filepath.Join(dir, "package.json"),
		filepath.Join(dir, "README.md"),
		filepath.Join(dir, "fontcreator233.manifest.json"),
		filepath.Join(editorDir, "FontCreator233TMPImporter.cs"),
	}
	if licenseFile != "" {
		paths = append(paths, filepath.Join(dir, licenseFile))
	}
	paths = append(paths, fontPaths...)
	for _, name := range charsetNames {
		paths = append(paths, filepath.Join(dir, filepath.FromSlash(name)))
	}
	return paths, nil
}

func packageSlug(family string) string {
	var b strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(family) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "font"
	}
	if len(slug) > 48 {
		slug = slug[:48]
	}
	return slug
}
