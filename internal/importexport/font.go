// Package importexport profiles fonts, exports glyphs, and moves bundles.
package importexport

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/derive"
	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/svgout"
	"github.com/neko233-com/fontcreator233/internal/unicodes"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"gopkg.in/yaml.v3"
)

// Profile summarizes an imported font and the inferred recipe values.
type Profile struct {
	File        string            `json:"file"`
	Family      string            `json:"family"`
	Style       string            `json:"style"`
	Version     string            `json:"version"`
	License     string            `json:"license"`
	LicenseURL  string            `json:"licenseUrl,omitempty"`
	UnitsPerEm  int               `json:"unitsPerEm"`
	Glyphs      int               `json:"glyphs"`
	Tables      []string          `json:"tables"`
	WeightClass int               `json:"weightClass"`
	WidthClass  int               `json:"widthClass"`
	ItalicAngle float64           `json:"italicAngle"`
	Ascender    int               `json:"ascender"`
	Descender   int               `json:"descender"`
	LineGap     int               `json:"lineGap"`
	XHeight     int               `json:"xHeight"`
	CapHeight   int               `json:"capHeight"`
	Coverage    map[string]int    `json:"coverage"`
	Runes       []rune            `json:"-"`
	Confidence  map[string]string `json:"confidence"`
	Warnings    []string          `json:"warnings,omitempty"`

	RecipePath string   `json:"recipePath,omitempty"`
	GlyphDir   string   `json:"glyphDir,omitempty"`
	Extracted  []string `json:"extracted,omitempty"`
}

// ImportFont profiles path and writes profile.json, recipe.yaml, and optional
// SVG glyphs under outDir.
func ImportFont(path, outDir string, extractGlyphs bool, unicodesFilter []rune, force bool) (Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return Profile{}, fmt.Errorf("parse font: %w", err)
	}
	var buf sfnt.Buffer
	p := Profile{
		File:       filepath.Base(path),
		UnitsPerEm: int(f.UnitsPerEm()),
		Glyphs:     f.NumGlyphs(),
		Tables:     tableTags(data),
		Coverage:   map[string]int{},
		Confidence: map[string]string{"serif": "low", "contrast": "low"},
	}
	p.Family, _ = f.Name(&buf, sfnt.NameIDFamily)
	p.Style, _ = f.Name(&buf, sfnt.NameIDSubfamily)
	p.Version, _ = f.Name(&buf, sfnt.NameIDVersion)
	p.License, _ = f.Name(&buf, sfnt.NameIDLicense)
	p.LicenseURL, _ = f.Name(&buf, sfnt.NameIDLicenseURL)
	metrics, err := f.Metrics(&buf, 1000, 0)
	if err != nil {
		return Profile{}, fmt.Errorf("metrics: %w", err)
	}
	p.Ascender = int(metrics.Ascent)
	p.Descender = -int(metrics.Descent)
	p.XHeight = int(metrics.XHeight)
	p.CapHeight = int(metrics.CapHeight)
	if pt := f.PostTable(); pt != nil {
		p.ItalicAngle = pt.ItalicAngle
	}
	if os2 := rawTable(data, "OS/2"); len(os2) >= 6 {
		p.WeightClass = int(binary.BigEndian.Uint16(os2[4:]))
		p.WidthClass = int(binary.BigEndian.Uint16(os2[6:]))
		if len(os2) >= 90 {
			upm := maxInt(p.UnitsPerEm, 1)
			if h := int(int16(binary.BigEndian.Uint16(os2[86:]))); h > 0 {
				p.XHeight = h * 1000 / upm
			}
			if h := int(int16(binary.BigEndian.Uint16(os2[88:]))); h > 0 {
				p.CapHeight = h * 1000 / upm
			}
		}
	} else {
		p.Warnings = append(p.Warnings, "OS/2 table missing; metrics were inferred from the font")
	}
	if p.XHeight <= 0 || p.CapHeight <= 0 {
		p.Warnings = append(p.Warnings, "OS/2 x-height or cap-height missing; review metrics.xHeight and metrics.capHeight")
	}
	if p.WeightClass == 0 {
		p.WeightClass = 400
	}
	if p.WidthClass == 0 {
		p.WidthClass = 5
	}
	for r := rune(0); r <= 0x323AF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		gid, err := f.GlyphIndex(&buf, r)
		if err != nil {
			return Profile{}, err
		}
		if gid != 0 {
			p.Runes = append(p.Runes, r)
			p.Coverage[unicodes.ScriptOf(r)]++
		}
	}
	if entries, err := os.ReadDir(outDir); err == nil && len(entries) > 0 && !force {
		return Profile{}, fmt.Errorf("import font: output directory %s is not empty (use --force)", outDir)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return Profile{}, err
	}
	if extractGlyphs {
		dir := filepath.Join(outDir, "glyphs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Profile{}, err
		}
		requested := p.Runes
		if len(unicodesFilter) > 0 {
			requested = unicodesFilter
		}
		if len(requested) > 20000 {
			requested = requested[:20000]
		}
		for _, r := range requested {
			gid, err := f.GlyphIndex(&buf, r)
			if err != nil || gid == 0 {
				continue
			}
			segs, err := f.LoadGlyph(&buf, gid, fixed.Int26_6(f.UnitsPerEm()), nil)
			if err != nil || len(segs) == 0 {
				continue
			}
			adv26, err := f.GlyphAdvance(&buf, gid, fixed.Int26_6(f.UnitsPerEm()), 0)
			if err != nil {
				continue
			}
			pathData := derive.SegmentsToPath(segs)
			if len(pathData) == 0 {
				continue
			}
			advance := int(math.Round(float64(adv26)))
			if p.UnitsPerEm > 0 && p.UnitsPerEm != 1000 {
				scale := 1000 / float64(p.UnitsPerEm)
				pathData = pathData.Transformed(geom.Affine{A: scale, E: scale})
				advance = int(math.Round(float64(advance) * scale))
			}
			name := fmt.Sprintf("U+%04X.svg", r)
			file, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				return Profile{}, err
			}
			err = svgout.WriteGlyph(file, fmt.Sprintf("%s U+%04X", p.Family, r), advance, pathData)
			_ = file.Close()
			if err != nil {
				return Profile{}, err
			}
			p.Extracted = append(p.Extracted, name)
		}
		p.GlyphDir = dir
	}
	profData, _ := json.MarshalIndent(p, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "profile.json"), append(profData, '\n'), 0o644); err != nil {
		return Profile{}, err
	}
	r := profileRecipe(p, extractGlyphs)
	recipePath := filepath.Join(outDir, "recipe.yaml")
	recipeData, err := yaml.Marshal(r)
	if err != nil {
		return Profile{}, err
	}
	if err := os.WriteFile(recipePath, recipeData, 0o644); err != nil {
		return Profile{}, err
	}
	p.RecipePath = recipePath
	return p, nil
}

func profileRecipe(p Profile, extractGlyphs bool) *recipe.Recipe {
	r := recipe.Default()
	family := strings.TrimSpace(p.Family)
	if family == "" {
		family = "Imported Font"
	}
	if strings.HasPrefix(strings.ToLower(family), "noto") {
		family = "Imported " + family
	}
	if len(family) > 55 {
		family = family[:55]
	}
	r.Meta.Family = family + " Import"
	r.Meta.Style = strings.TrimSpace(p.Style)
	if r.Meta.Style == "" {
		r.Meta.Style = "Regular"
	}
	r.Meta.Version = cleanVersion(p.Version)
	r.Meta.Description = "Profile inferred from " + p.File + ". Review style, contrast, and serif fields."
	if strings.Contains(strings.ToLower(p.License), "open font license") {
		r.Meta.License = "OFL-1.1"
	} else {
		r.Meta.License = "UNLICENSED"
	}
	r.Meta.SampleText = "Hamburgefonstiv 字体创造器 0123"
	r.Metrics.Ascender = clampInt(p.Ascender, 600, 1000)
	r.Metrics.Descender = clampInt(p.Descender, -400, -50)
	r.Metrics.XHeight = clampFloat(float64(p.XHeight)/1000, 0.40, 0.60)
	r.Metrics.CapHeight = clampFloat(float64(p.CapHeight)/1000, 0.64, 0.78)
	if r.Metrics.Ascender <= int(r.Metrics.CapHeight*1000) {
		r.Metrics.Ascender = int(r.Metrics.CapHeight*1000) + 20
	}
	r.Style.Weight = clampInt(p.WeightClass, 100, 900)
	r.Style.Width = widthPercent(p.WidthClass)
	r.Style.Slant = clampFloat(-p.ItalicAngle, -15, 15)
	r.Style.Skeleton = "geometric"
	r.Style.Contrast = 0.10
	r.Style.Terminal = "round"
	r.Style.Serif = "none"
	r.Style.SerifSize = 0
	r.Coverage.Generated = []string{"latin"}
	if p.Coverage["greek"] > 0 {
		r.Coverage.Generated = append(r.Coverage.Generated, "greek")
	}
	if p.Coverage["cyrillic"] > 0 {
		r.Coverage.Generated = append(r.Coverage.Generated, "cyrillic")
	}
	r.Coverage.Base = recipe.BaseCoverage{CJK: "none"}
	if extractGlyphs {
		r.Glyphs.Overrides = "glyphs"
	}
	r.Output.Formats = []string{"ttf", "woff", "woff2"}
	r.Output.Targets = []string{"universal", "web", "system", "unity", "apps"}
	r.Output.Specimen = true
	r.Output.Open = false
	return r
}

func cleanVersion(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "Version"))
	s = strings.TrimSpace(s)
	if s == "" {
		return "1.0.0"
	}
	return s
}

func widthPercent(class int) float64 {
	values := []float64{50, 62.5, 75, 87.5, 100, 112.5, 125, 150, 200}
	if class < 1 || class > len(values) {
		return 100
	}
	return clampFloat(values[class-1], 75, 125)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if math.IsNaN(v) || v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func tableTags(data []byte) []string {
	if len(data) < 12 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	if 12+n*16 > len(data) {
		return nil
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, string(data[12+i*16:12+i*16+4]))
	}
	sort.Strings(out)
	return out
}

func rawTable(data []byte, tag string) []byte {
	if len(data) < 12 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := 0; i < n; i++ {
		base := 12 + i*16
		if base+16 > len(data) {
			return nil
		}
		if string(data[base:base+4]) != tag {
			continue
		}
		off := binary.BigEndian.Uint32(data[base+8:])
		length := binary.BigEndian.Uint32(data[base+12:])
		if int(off)+int(length) > len(data) {
			return nil
		}
		return data[off : off+length]
	}
	return nil
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
