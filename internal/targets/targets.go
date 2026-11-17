// Package targets writes web, system, Unity, and Unreal integration bundles.
package targets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neko233-com/fontcreator233/assets"
	"github.com/neko233-com/fontcreator233/internal/integration"
	"github.com/neko233-com/fontcreator233/internal/recipe"
)

// FontInput is one compiled font file.
type FontInput struct {
	Path    string
	WOFF    string
	SHA256  string
	Primary bool
	Style   string
	Runes   []rune
}

// Options controls target generation.
type Options struct {
	OutDir         string
	Family         string
	Style          string
	RequestedStyle string
	Version        string
	License        string
	Weight         int
	Targets        []string
	Fonts          []FontInput
	SpecimenPath   string
	TMP            recipe.TMP
}

// Build writes the selected targets and returns their paths.
func Build(ctx context.Context, o Options) (map[string][]string, error) {
	if len(o.Fonts) == 0 {
		return nil, fmt.Errorf("targets: no fonts")
	}
	if o.OutDir == "" {
		return nil, fmt.Errorf("targets: empty output directory")
	}
	selected, err := expandTargets(o.Targets)
	if err != nil {
		return nil, err
	}
	result := map[string][]string{}
	manifest, err := buildManifest(o)
	if err != nil {
		return nil, err
	}
	manifestPath := filepath.Join(o.OutDir, "fontcreator233.manifest.json")
	if err := integration.WriteJSON(manifestPath, manifest); err != nil {
		return nil, err
	}
	oflPath := filepath.Join(o.OutDir, "OFL.txt")
	if err := os.WriteFile(oflPath, assets.OFL, 0o644); err != nil {
		return nil, err
	}
	var allRunes []rune
	for _, f := range o.Fonts {
		allRunes = append(allRunes, f.Runes...)
	}
	charsets := integration.Charsets(allRunes)
	all := append([]rune(nil), allRunes...)
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	charsets["all"] = string(all)
	charsetDir := filepath.Join(o.OutDir, "Charsets")
	if err := os.MkdirAll(charsetDir, 0o755); err != nil {
		return nil, err
	}
	for script, text := range charsets {
		if err := os.WriteFile(filepath.Join(charsetDir, script+".txt"), []byte(text), 0o644); err != nil {
			return nil, err
		}
	}

	universal := []string{o.Fonts[0].Path}
	if o.Fonts[0].WOFF != "" {
		universal = append(universal, o.Fonts[0].WOFF)
	}
	if o.SpecimenPath != "" {
		universal = append(universal, o.SpecimenPath)
	}
	universal = append(universal, manifestPath, oflPath)
	result["universal"] = universal

	for _, target := range selected {
		switch target {
		case "universal":
		case "web":
			paths, err := buildWeb(o, manifest)
			if err != nil {
				return nil, err
			}
			result["web"] = paths
		case "system":
			paths, err := buildSystem(o)
			if err != nil {
				return nil, err
			}
			result["system"] = paths
		case "unity":
			paths, err := buildUnity(o, manifest)
			if err != nil {
				return nil, err
			}
			result["unity"] = paths
		case "unreal":
			paths, err := buildUnreal(o, manifest)
			if err != nil {
				return nil, err
			}
			result["unreal"] = paths
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
	return result, nil
}

func expandTargets(in []string) ([]string, error) {
	if len(in) == 0 {
		return []string{"universal"}, nil
	}
	allowed := map[string]bool{"universal": true, "web": true, "system": true, "unity": true, "unreal": true}
	seen := map[string]bool{}
	var out []string
	for _, t := range in {
		if t == "all" {
			for _, a := range []string{"universal", "web", "system", "unity", "unreal"} {
				if !seen[a] {
					seen[a] = true
					out = append(out, a)
				}
			}
			continue
		}
		if !allowed[t] {
			return nil, fmt.Errorf("targets: unknown target %q", t)
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out, nil
}

func buildManifest(o Options) (*integration.Manifest, error) {
	m := &integration.Manifest{
		Schema:         "fontcreator233/integration/v1",
		Generator:      "fontcreator233 0.1.0",
		Family:         o.Family,
		Style:          o.Style,
		RequestedStyle: o.RequestedStyle,
		License:        o.License,
		LicenseFile:    "OFL.txt",
		UnityTMP: integration.TMP{
			Population:        o.TMP.Population,
			SamplingPointSize: o.TMP.SamplingPointSize,
			Padding:           o.TMP.Padding,
			AtlasWidth:        o.TMP.AtlasWidth,
			AtlasHeight:       o.TMP.AtlasHeight,
			MultiAtlas:        o.TMP.MultiAtlas,
			RenderMode:        o.TMP.RenderMode,
		},
		Web: integration.Web{CSS: "web/font.css", FontDisplay: "swap"},
	}
	var allRunes []rune
	for _, f := range o.Fonts {
		allRunes = append(allRunes, f.Runes...)
	}
	scripts := integration.Scripts(allRunes)
	for _, s := range scripts {
		m.Charsets = append(m.Charsets, "Charsets/"+s+".txt")
	}
	for i, f := range o.Fonts {
		rel, err := relPath(o.OutDir, f.Path)
		if err != nil {
			return nil, err
		}
		sha := f.SHA256
		if sha == "" {
			sha, err = hashFile(f.Path)
			if err != nil {
				return nil, err
			}
		}
		font := integration.Font{
			Path:    rel,
			SHA256:  sha,
			Primary: f.Primary || i == 0,
			Scripts: integration.Scripts(f.Runes),
			Ranges:  integration.Ranges(f.Runes),
		}
		if f.WOFF != "" {
			font.WOFF, err = relPath(o.OutDir, f.WOFF)
			if err != nil {
				return nil, err
			}
		}
		m.Fonts = append(m.Fonts, font)
		m.Fallbacks = append(m.Fallbacks, rel)
		if !font.Primary {
			m.UnrealComposite.SubFonts = append(m.UnrealComposite.SubFonts, integration.Composite{
				Path:   rel,
				Ranges: integration.Ranges(f.Runes),
			})
		} else {
			m.UnrealComposite.Default = rel
			m.Charsets = append(m.Charsets, "Charsets/all.txt")
		}
	}
	sort.Strings(m.Charsets)
	m.Charsets = dedupe(m.Charsets)
	return m, nil
}

func buildWeb(o Options, m *integration.Manifest) ([]string, error) {
	dir := filepath.Join(o.OutDir, "web")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var css strings.Builder
	for _, f := range o.Fonts {
		base := filepath.Base(f.Path)
		if err := copyFile(f.Path, filepath.Join(dir, base)); err != nil {
			return nil, err
		}
		if f.WOFF != "" {
			if err := copyFile(f.WOFF, filepath.Join(dir, filepath.Base(f.WOFF))); err != nil {
				return nil, err
			}
		}
		srcs := []string{}
		if f.WOFF != "" {
			srcs = append(srcs, fmt.Sprintf(`url("./%s") format("woff")`, filepath.Base(f.WOFF)))
		}
		srcs = append(srcs, fmt.Sprintf(`url("./%s") format("truetype")`, base))
		fmt.Fprintf(&css, "@font-face {\n  font-family: %s;\n  src: %s;\n  font-display: swap;\n  font-weight: %d;\n",
			cssString(o.Family), strings.Join(srcs, ",\n       "), o.Weight)
		if len(o.Fonts) > 1 {
			fmt.Fprintf(&css, "  unicode-range: %s;\n", unicodeRange(integration.Ranges(f.Runes)))
		}
		css.WriteString("}\n\n")
	}
	cssPath := filepath.Join(dir, "font.css")
	if err := os.WriteFile(cssPath, []byte(css.String()), 0o644); err != nil {
		return nil, err
	}
	paths := []string{cssPath}
	if o.SpecimenPath != "" {
		htmlBase := filepath.Base(o.SpecimenPath)
		if err := copyFile(o.SpecimenPath, filepath.Join(dir, htmlBase)); err != nil {
			return nil, err
		}
		paths = append(paths, filepath.Join(dir, htmlBase))
		assetsSrc := filepath.Join(o.OutDir, "assets")
		if _, err := os.Stat(assetsSrc); err == nil {
			if err := copyTree(assetsSrc, filepath.Join(dir, "assets")); err != nil {
				return nil, err
			}
			paths = append(paths, filepath.Join(dir, "assets"))
		}
		stem := strings.TrimSuffix(htmlBase, filepath.Ext(htmlBase))
		glyph := filepath.Join(o.OutDir, stem+".glyphs.js")
		if _, err := os.Stat(glyph); err == nil {
			if err := copyFile(glyph, filepath.Join(dir, filepath.Base(glyph))); err != nil {
				return nil, err
			}
			paths = append(paths, filepath.Join(dir, filepath.Base(glyph)))
		}
	}
	return paths, nil
}

func buildSystem(o Options) ([]string, error) {
	dir := filepath.Join(o.OutDir, "system")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	font := filepath.Join(dir, filepath.Base(o.Fonts[0].Path))
	if err := copyFile(o.Fonts[0].Path, font); err != nil {
		return nil, err
	}
	install := fmt.Sprintf(`# %s %s — system install

## Windows
Right-click `+"`%s`"+` → Install, or copy it to %%WINDIR%%\Fonts and refresh the font cache.

## macOS
Double-click the font file (Font Book), or copy it to ~/Library/Fonts.

## Linux
Copy it to ~/.local/share/fonts, then run:
`+"```"+`sh
fc-cache -f -v
`+"```"+`

The font is licensed under OFL-1.1; see ../OFL.txt.
`, o.Family, o.Style, filepath.Base(font))
	installPath := filepath.Join(dir, "INSTALL.md")
	if err := os.WriteFile(installPath, []byte(install), 0o644); err != nil {
		return nil, err
	}
	return []string{font, installPath}, nil
}

func unicodeRange(ranges []integration.Range) string {
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		if r[0] == r[1] {
			parts = append(parts, fmt.Sprintf("U+%04X", r[0]))
		} else {
			parts = append(parts, fmt.Sprintf("U+%04X-%04X", r[0], r[1]))
		}
	}
	return strings.Join(parts, ", ")
}

func cssString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func relPath(root, path string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
