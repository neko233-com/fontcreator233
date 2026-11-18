package importexport

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/build"
	"github.com/neko233-com/fontcreator233/internal/derive"
	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/svgout"
	"github.com/neko233-com/fontcreator233/internal/unicodes"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Filter selects glyphs for export.
type Filter struct {
	Unicodes []rune
	Scripts  []string
	Limit    int
	Force    bool
}

// ExportGlyphsFromFont writes one SVG per matching glyph.
func ExportGlyphsFromFont(fontPath, dir string, filter Filter) ([]string, error) {
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, err
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var buf sfnt.Buffer
	var paths []string
	limited := filter.Limit > 0
	var candidates []rune
	unicodeSet := make(map[rune]struct{}, len(filter.Unicodes))
	if len(filter.Unicodes) > 0 {
		candidates = filter.Unicodes
		for _, r := range filter.Unicodes {
			unicodeSet[r] = struct{}{}
		}
	} else {
		for r := rune(0); r <= 0x323AF; r++ {
			candidates = append(candidates, r)
		}
	}
	scale := 1.0
	if f.UnitsPerEm() > 0 && f.UnitsPerEm() != 1000 {
		scale = 1000 / float64(f.UnitsPerEm())
	}
	for _, r := range candidates {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		if len(unicodeSet) > 0 {
			if _, ok := unicodeSet[r]; !ok {
				continue
			}
		}
		if !matchScripts(r, filter.Scripts) {
			continue
		}
		gid, err := f.GlyphIndex(&buf, r)
		if err != nil || gid == 0 {
			continue
		}
		segs, err := f.LoadGlyph(&buf, gid, fixed.Int26_6(f.UnitsPerEm()), nil)
		if err != nil || len(segs) == 0 {
			continue
		}
		adv, err := f.GlyphAdvance(&buf, gid, fixed.Int26_6(f.UnitsPerEm()), 0)
		if err != nil {
			continue
		}
		pathData := derive.SegmentsToPath(segs)
		if len(pathData) == 0 {
			continue
		}
		advance := int(math.Round(float64(adv)))
		if scale != 1 {
			pathData = pathData.Transformed(geom.Affine{A: scale, E: scale})
			advance = int(math.Round(float64(advance) * scale))
		}
		name := fmt.Sprintf("U+%04X.svg", r)
		target := filepath.Join(dir, name)
		if !filter.Force {
			if _, err := os.Stat(target); err == nil {
				return nil, fmt.Errorf("export glyphs: refusing to overwrite %s (use --force)", target)
			}
		}
		file, err := os.Create(target)
		if err != nil {
			return nil, err
		}
		err = svgout.WriteGlyph(file, fmt.Sprintf("U+%04X %c", r, r), advance, pathData)
		_ = file.Close()
		if err != nil {
			return nil, err
		}
		paths = append(paths, filepath.Join(dir, name))
		if limited && len(paths) >= filter.Limit {
			break
		}
	}
	return paths, nil
}

// ExportGlyphsFromRecipe builds a recipe in a temp directory and exports from
// the generated universal font.
func ExportGlyphsFromRecipe(recipePath, dir string, filter Filter) ([]string, error) {
	r, err := recipe.Load(recipePath)
	if err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "fontcreator233-export-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	r.Output.Specimen = false
	r.Output.Open = false
	r.Output.Targets = []string{"universal"}
	r.Output.Formats = []string{"ttf"}
	res, err := build.Build(context.Background(), r, build.Options{OutDir: tmp, Targets: []string{"universal"}, NoOpen: true}, nil)
	if err != nil {
		return nil, err
	}
	return ExportGlyphsFromFont(res.FontPath, dir, filter)
}

// ParseUnicodeSpec parses "0041-007A,4E00" into runes.
func ParseUnicodeSpec(spec string) ([]rune, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	var out []rune
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lo, hi := part, part
		if i := strings.Index(part, "-"); i > 0 {
			lo, hi = part[:i], part[i+1:]
		}
		a, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(lo), "U+"), 16, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid unicode %q", lo)
		}
		b := a
		if hi != lo {
			b, err = strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(hi), "U+"), 16, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid unicode %q", hi)
			}
		}
		if a > b || b > 0x10FFFF {
			return nil, fmt.Errorf("invalid unicode range %q", part)
		}
		for r := a; r <= b; r++ {
			out = append(out, rune(r))
		}
	}
	return out, nil
}

func matchScripts(r rune, scripts []string) bool {
	if len(scripts) == 0 {
		return true
	}
	s := unicodes.ScriptOf(r)
	for _, want := range scripts {
		if s == want {
			return true
		}
	}
	return false
}
