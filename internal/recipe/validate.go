package recipe

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var accentRE = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// Normalize fills values that have a safe derived default and removes duplicate
// list entries. It is idempotent.
func (r *Recipe) Normalize() {
	if r.Style.Serif != "" && r.Style.Serif != "none" && r.Style.SerifSize == 0 {
		r.Style.SerifSize = 0.5
	}
	r.Coverage.Generated = dedupe(r.Coverage.Generated)
	r.Output.Formats = dedupe(r.Output.Formats)
	r.Output.Targets = dedupe(r.Output.Targets)
}

func dedupe(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// Validate checks the complete recipe. The first error is returned as a
// *ValidationError.
func (r *Recipe) Validate() error {
	r.Normalize()
	if r.APIVersion != "fontcreator233/v1" {
		return fail("apiVersion", "must be fontcreator233/v1", r.APIVersion)
	}
	if err := validateMeta(&r.Meta); err != nil {
		return err
	}
	if err := validateMetrics(&r.Metrics); err != nil {
		return err
	}
	if err := validateStyle(&r.Style); err != nil {
		return err
	}
	if err := validateCoverage(&r.Coverage); err != nil {
		return err
	}
	if err := validateOpenType(&r.OpenType); err != nil {
		return err
	}
	if err := validateOutput(&r.Output); err != nil {
		return err
	}
	baseEnabled := r.Coverage.Base.CJK != "none" || r.Coverage.Base.Kana || r.Coverage.Base.Hangul || r.Coverage.Base.Symbols
	if baseEnabled && r.Meta.License != "OFL-1.1" {
		return fail("meta.license", "must be OFL-1.1 when embedded base glyphs are used", r.Meta.License)
	}
	return nil
}

func validateMeta(m *Meta) error {
	family := strings.TrimSpace(m.Family)
	if family == "" {
		return fail("meta.family", "must not be empty", m.Family)
	}
	if len(family) > 64 {
		return fail("meta.family", "must be at most 64 characters", m.Family)
	}
	if strings.HasPrefix(strings.ToLower(family), "noto") {
		return fail("meta.family", "must not start with the reserved name Noto", m.Family)
	}
	if hasControl(family) {
		return fail("meta.family", "must not contain control characters", m.Family)
	}
	if len(m.Style) > 64 || hasControl(m.Style) {
		return fail("meta.style", "must be at most 64 characters without control characters", m.Style)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fail("meta.version", "must not be empty", m.Version)
	}
	if m.License == "" {
		return fail("meta.license", "must not be empty", m.License)
	}
	if m.Accent != "" && !accentRE.MatchString(m.Accent) {
		return fail("meta.accent", "must be #RGB or #RRGGBB", m.Accent)
	}
	return nil
}

func validateMetrics(m *Metrics) error {
	if !finite(m.XHeight) {
		return fail("metrics.xHeight", "must be finite", m.XHeight)
	}
	if !finite(m.CapHeight) {
		return fail("metrics.capHeight", "must be finite", m.CapHeight)
	}
	if m.Ascender < 600 || m.Ascender > 1000 {
		return fail("metrics.ascender", "must be between 600 and 1000", m.Ascender)
	}
	if m.Descender < -400 || m.Descender > -50 {
		return fail("metrics.descender", "must be between -400 and -50", m.Descender)
	}
	if m.LineGap < 0 || m.LineGap > 300 {
		return fail("metrics.lineGap", "must be between 0 and 300", m.LineGap)
	}
	if m.XHeight < 0.40 || m.XHeight > 0.60 {
		return fail("metrics.xHeight", "must be between 0.40 and 0.60", m.XHeight)
	}
	if m.CapHeight < 0.64 || m.CapHeight > 0.78 {
		return fail("metrics.capHeight", "must be between 0.64 and 0.78", m.CapHeight)
	}
	if float64(m.Ascender) <= m.CapHeight*1000 {
		return fail("metrics.ascender", "must be greater than capHeight * 1000", m.Ascender)
	}
	return nil
}

func validateStyle(s *Style) error {
	if err := oneOf("style.skeleton", s.Skeleton, "geometric", "humanist", "serif"); err != nil {
		return err
	}
	if s.Weight < 100 || s.Weight > 900 {
		return fail("style.weight", "must be between 100 and 900", s.Weight)
	}
	if !finite(s.Width) || s.Width < 75 || s.Width > 125 {
		return fail("style.width", "must be between 75 and 125", s.Width)
	}
	if !finite(s.Slant) || s.Slant < -15 || s.Slant > 15 {
		return fail("style.slant", "must be between -15 and 15", s.Slant)
	}
	if !finite(s.Contrast) || s.Contrast < 0 || s.Contrast > 1 {
		return fail("style.contrast", "must be between 0 and 1", s.Contrast)
	}
	if err := oneOf("style.terminal", s.Terminal, "flat", "round", "angled", "flare"); err != nil {
		return err
	}
	if err := oneOf("style.serif", s.Serif, "none", "slab", "bracketed", "hairline"); err != nil {
		return err
	}
	if !finite(s.SerifSize) || s.SerifSize < 0 || s.SerifSize > 1 {
		return fail("style.serifSize", "must be between 0 and 1", s.SerifSize)
	}
	if !finite(s.Overshoot) || s.Overshoot < 0 || s.Overshoot > 0.1 {
		return fail("style.overshoot", "must be between 0 and 0.1", s.Overshoot)
	}
	if !finite(s.Tracking) || s.Tracking < -5 || s.Tracking > 10 {
		return fail("style.tracking", "must be between -5 and 10 percent", s.Tracking)
	}
	if !finite(s.Roughness) || s.Roughness < 0 || s.Roughness > 1 {
		return fail("style.roughness", "must be between 0 and 1", s.Roughness)
	}
	if !finite(s.JitterRotation) || s.JitterRotation < 0 || s.JitterRotation > 2 {
		return fail("style.jitterRotation", "must be between 0 and 2", s.JitterRotation)
	}
	return nil
}

func validateCoverage(c *Coverage) error {
	for _, g := range c.Generated {
		if err := oneOf("coverage.generated", g, "latin", "greek", "cyrillic"); err != nil {
			return err
		}
	}
	if err := oneOf("coverage.base.cjk", c.Base.CJK, "none", "sample", "common", "full"); err != nil {
		return err
	}
	if len(c.Generated) == 0 && c.Base.CJK == "none" && !c.Base.Kana && !c.Base.Hangul && !c.Base.Symbols {
		return fail("coverage", "must enable at least one generated or base script", nil)
	}
	return nil
}

func validateOpenType(o *OpenType) error {
	for i, rule := range o.Contextual {
		path := fmt.Sprintf("openType.contextual[%d]", i)
		if len([]rune(rule.Input)) < 2 {
			return fail(path+".input", "must contain at least two characters", rule.Input)
		}
		if len([]rune(rule.Target)) != 1 {
			return fail(path+".target", "must be exactly one character", rule.Target)
		}
		feature := rule.Feature
		if feature == "" {
			feature = "calt"
		}
		if len(feature) != 4 || strings.TrimSpace(feature) != feature {
			return fail(path+".feature", "must be a four-character OpenType tag", rule.Feature)
		}
	}
	if o.Color != nil {
		if err := validateColor(o.Color); err != nil {
			return err
		}
	}
	return nil
}

func validateColor(c *ColorConfig) error {
	if len(c.Palettes) == 0 {
		return fail("openType.color.palettes", "must not be empty", c.Palettes)
	}
	if len(c.Palettes[0]) < 2 {
		return fail("openType.color.palettes[0]", "must have at least two colors", c.Palettes[0])
	}
	entries := len(c.Palettes[0])
	for pi, palette := range c.Palettes {
		ppath := fmt.Sprintf("openType.color.palettes[%d]", pi)
		if len(palette) != entries {
			return fail(ppath, "must have the same number of colors as palette 0", len(palette))
		}
		for ci, col := range palette {
			if !accentRE.MatchString(col) {
				return fail(fmt.Sprintf("%s[%d]", ppath, ci), "must be #RGB or #RRGGBB", col)
			}
		}
	}
	if len(c.Glyphs) == 0 {
		return fail("openType.color.glyphs", "must not be empty", c.Glyphs)
	}
	seen := map[rune]bool{}
	for gi, g := range c.Glyphs {
		path := fmt.Sprintf("openType.color.glyphs[%d]", gi)
		cp, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(g.Codepoint), "U+"), 16, 32)
		if err != nil || cp > 0x10FFFF {
			return fail(path+".codepoint", "must be a hex codepoint", g.Codepoint)
		}
		rn := rune(cp)
		if seen[rn] {
			return fail(path+".codepoint", "duplicated codepoint", g.Codepoint)
		}
		seen[rn] = true
		if g.Advance != 0 && (g.Advance < 100 || g.Advance > 4000) {
			return fail(path+".advance", "must be between 100 and 4000", g.Advance)
		}
		if len(g.Layers) == 0 {
			return fail(path+".layers", "must not be empty", g.Layers)
		}
		for li, layer := range g.Layers {
			lpath := fmt.Sprintf("%s.layers[%d]", path, li)
			if err := oneOf(lpath+".shape", layer.Shape, "circle", "square", "triangle", "star", "ring"); err != nil {
				return err
			}
			if layer.Color < 0 || layer.Color >= entries {
				return fail(lpath+".color", "out of palette range", layer.Color)
			}
			if layer.Scale < 0 || layer.Scale > 2 {
				return fail(lpath+".scale", "must be between 0 and 2", layer.Scale)
			}
		}
	}
	return nil
}

func validateOutput(o *Output) error {
	if len(o.Formats) == 0 {
		return fail("output.formats", "must not be empty", o.Formats)
	}
	for _, f := range o.Formats {
		if err := oneOf("output.formats", f, "ttf", "woff", "woff2"); err != nil {
			return err
		}
	}
	if len(o.Targets) == 0 {
		return fail("output.targets", "must not be empty", o.Targets)
	}
	for _, t := range o.Targets {
		if err := oneOf("output.targets", t, "universal", "web", "system", "unity", "unreal", "apps", "bmfont", "all"); err != nil {
			return err
		}
	}
	if o.BMFont != nil {
		b := o.BMFont
		if b.Size != 0 && (b.Size < 8 || b.Size > 256) {
			return fail("output.bmfont.size", "must be between 8 and 256", b.Size)
		}
		if b.Padding < 0 || b.Padding > 16 {
			return fail("output.bmfont.padding", "must be between 0 and 16", b.Padding)
		}
		if b.AtlasSize != 0 && (b.AtlasSize < 64 || b.AtlasSize > 4096) {
			return fail("output.bmfont.atlasSize", "must be between 64 and 4096", b.AtlasSize)
		}
	}
	if o.SplitCJK < 0 {
		return fail("output.splitCJK", "must be >= 0", o.SplitCJK)
	}
	if err := oneOf("output.tmp.population", o.TMP.Population, "dynamic", "static"); err != nil {
		return err
	}
	if o.TMP.SamplingPointSize < 8 || o.TMP.SamplingPointSize > 200 {
		return fail("output.tmp.samplingPointSize", "must be between 8 and 200", o.TMP.SamplingPointSize)
	}
	if o.TMP.Padding < 0 || o.TMP.Padding > 64 {
		return fail("output.tmp.padding", "must be between 0 and 64", o.TMP.Padding)
	}
	if v := o.TMP.AtlasWidth; v < 256 || v > 4096 || v&(v-1) != 0 {
		return fail("output.tmp.atlasWidth", "must be a power of two between 256 and 4096", v)
	}
	if v := o.TMP.AtlasHeight; v < 256 || v > 4096 || v&(v-1) != 0 {
		return fail("output.tmp.atlasHeight", "must be a power of two between 256 and 4096", v)
	}
	if err := oneOf("output.tmp.renderMode", o.TMP.RenderMode, "SDFAA", "SDF8", "SDF16", "SDF32"); err != nil {
		return err
	}
	return nil
}

func oneOf(field, value string, allowed ...string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return fail(field, fmt.Sprintf("must be one of %s", strings.Join(allowed, ", ")), value)
}

func fail(field, message string, value any) error {
	return &ValidationError{Field: field, Message: message, Value: value}
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7F {
			return true
		}
	}
	return false
}
