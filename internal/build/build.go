// Package build turns a validated recipe into fonts and integration output.
package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/neko233-com/fontcreator233/internal/bmfont"
	"github.com/neko233-com/fontcreator233/internal/compose"
	"github.com/neko233-com/fontcreator233/internal/derive"
	"github.com/neko233-com/fontcreator233/internal/fontbuild"
	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/library"
	"github.com/neko233-com/fontcreator233/internal/parallel"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/skeleton"
	"github.com/neko233-com/fontcreator233/internal/specimen"
	"github.com/neko233-com/fontcreator233/internal/style"
	"github.com/neko233-com/fontcreator233/internal/targets"
	"github.com/neko233-com/fontcreator233/internal/ui"
	"github.com/neko233-com/fontcreator233/internal/unicodes"
)

// Options controls one build.
type Options struct {
	OutDir            string
	Targets           []string
	SplitCJK          int
	Jobs              int
	NoOpen            bool
	NoTUI             bool
	NoAutoClean       bool
	AssetsDir         string
	FinalOutDir       string
	SelfContainedHTML bool
}

// CoverageStat reports requested versus found code points.
type CoverageStat struct {
	Requested int
	Found     int
	Missing   int
}

// SplitFont describes one split subset.
type SplitFont struct {
	Path  string
	Style string
	Runes []rune
}

// Result is the outcome of a build.
type Result struct {
	Family         string
	Style          string
	RequestedStyle string
	FontPath       string
	WOFFPath       string
	WOFF2Path      string
	SpecimenPath   string
	SHA256         string
	Bytes          int
	Glyphs         int
	Coverage       map[string]int
	CoverageReport map[string]CoverageStat
	Warnings       []string
	Duration       time.Duration
	Targets        map[string][]string
	Split          []string
	GlyphRunes     []rune
	SplitFonts     []SplitFont
}

// Build compiles one recipe. The workspace layer wraps this in a staging
// directory and commits the result.
func Build(ctx context.Context, r *recipe.Recipe, o Options, rep ui.Reporter) (Result, error) {
	start := time.Now()
	if rep == nil {
		rep = ui.NopReporter{}
	}
	if o.AssetsDir != "" {
		abs, err := filepath.Abs(o.AssetsDir)
		if err != nil {
			return Result{}, err
		}
		o.AssetsDir = abs
	}
	resolvedStyle := style.ResolveStyle(r)
	params := style.Resolve(r)
	warnings := []string{}
	result := Result{
		Family:         r.Meta.Family,
		Style:          resolvedStyle,
		RequestedStyle: r.Meta.Style,
		Coverage:       map[string]int{},
		CoverageReport: map[string]CoverageStat{},
		Targets:        map[string][]string{},
	}
	var bmfontPaths []string
	rep.Event(ui.Event{Kind: ui.EvPhase, Phase: "generate"})
	gen, err := buildGenerated(ctx, r, params, o.Jobs, rep)
	if err != nil {
		return result, err
	}
	rep.Event(ui.Event{Kind: ui.EvPhase, Phase: "derive"})
	derived, stats, deriveWarnings, err := buildDerived(ctx, r, params, o.Jobs, rep)
	if err != nil {
		return result, err
	}
	warnings = append(warnings, deriveWarnings...)
	for _, script := range []string{"latin", "greek", "cyrillic"} {
		count := 0
		for rn := range gen {
			if unicodes.ScriptOf(rn) == script {
				count++
			}
		}
		if count > 0 {
			stats[script] = CoverageStat{Requested: count, Found: count}
		}
	}
	result.CoverageReport = stats

	merged := make(map[rune]fontbuild.Glyph, len(gen)+len(derived)+1)
	for k, v := range gen {
		merged[k] = v
	}
	for k, v := range derived {
		if _, ok := merged[k]; ok {
			continue
		}
		merged[k] = v
	}
	if r.Glyphs.Overrides != "" {
		dir := r.Glyphs.Overrides
		if !filepath.IsAbs(dir) && r.SourcePath != "" {
			dir = filepath.Join(filepath.Dir(r.SourcePath), dir)
		}
		overrides, err := library.LoadDir(dir)
		if err != nil {
			return result, fmt.Errorf("glyphs.overrides: %w", err)
		}
		for rn, ov := range overrides {
			merged[rn] = fontbuild.Glyph{Rune: rn, Advance: ov.Advance, Path: ov.Path}
		}
		if len(overrides) > 0 {
			warnings = append(warnings, fmt.Sprintf("applied %d SVG glyph overrides", len(overrides)))
		}
	}
	if _, ok := merged[0x20]; !ok {
		merged[0x20] = fontbuild.Glyph{Rune: 0x20, Advance: 250}
	}
	for rune, g := range merged {
		if rune != 0x20 && len(g.Path) == 0 {
			warnings = append(warnings, fmt.Sprintf("glyph %U is empty; dropped", rune))
			delete(merged, rune)
		}
		if rune == 0 || rune > 0x10FFFF {
			delete(merged, rune)
		}
	}
	for rune := range merged {
		result.Coverage[unicodes.ScriptOf(rune)]++
	}

	rep.Event(ui.Event{Kind: ui.EvPhase, Phase: "compile"})
	glyphs := make([]fontbuild.Glyph, 0, len(merged)+1)
	glyphs = append(glyphs, fontbuild.Glyph{Rune: 0, Name: ".notdef", Advance: 600, Path: notdefPath()})
	keys := make([]rune, 0, len(merged))
	for rune := range merged {
		keys = append(keys, rune)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, rune := range keys {
		g := merged[rune]
		g.Name = glyphName(rune)
		glyphs = append(glyphs, g)
	}
	result.GlyphRunes = append([]rune(nil), keys...)
	extraGlyphs, ligatures := makeLigatures(merged)
	featureGlyphs, substitutions := makeSubstitutions(merged, int(math.Round(r.Metrics.CapHeight*1000)), int(math.Round(r.Metrics.XHeight*1000)))
	fractionGlyphs, fractionLigatures := makeFractions(merged)
	var colorGlyphs []fontbuild.Glyph
	var colorSpec *fontbuild.ColorGlyph
	var colorSet *fontbuild.ColorSet
	var colorRunes []int
	if r.OpenType.Color != nil {
		colorGlyphs, colorSet, colorRunes = recipeColorGlyphs(r.OpenType.Color, merged, &warnings)
	} else {
		colorGlyphs, colorSpec = makeColorGlyph(merged)
		if colorSpec != nil {
			colorRunes = []int{int(colorSpec.Rune)}
		}
	}
	arrowGlyphs, contextual := makeArrowRule(merged)
	contextual = append(contextual, recipeContextualRules(r.OpenType.Contextual, merged, &warnings)...)
	glyphs = append(glyphs, extraGlyphs...)
	glyphs = append(glyphs, featureGlyphs...)
	glyphs = append(glyphs, fractionGlyphs...)
	glyphs = append(glyphs, colorGlyphs...)
	glyphs = append(glyphs, arrowGlyphs...)
	ligatures = append(ligatures, fractionLigatures...)
	ligatures = append(ligatures, makeComposedLigatures(merged)...)
	kerning := makeKerning(merged, r.Style.Weight)
	if wantsBMFont(o, r) {
		cfg := recipe.BMFontConfig{}
		if r.Output.BMFont != nil {
			cfg = *r.Output.BMFont
		}
		bmfontPaths, err = makeBMFont(o.OutDir, cfg, r, resolvedStyle, glyphs, kerning)
		if err != nil {
			warnings = append(warnings, "bmfont: "+err.Error())
		}
	}
	usedBase := len(derived) > 0
	hasMkmk := hasCombiningMarks(merged)
	splitSet := map[rune]bool{}
	for rn := range derived {
		if rn >= 0x2E80 {
			splitSet[rn] = true
		}
	}
	// Release the generation maps before the encoder allocates its own
	// buffers; only the sorted glyph slice is needed from here on.
	gen = nil
	derived = nil
	merged = nil
	cfg := fontConfig(r, resolvedStyle, glyphs)
	cfg.Ligatures = ligatures
	cfg.Substitutions = substitutions
	cfg.Contextual = contextual
	cfg.Kerning = kerning
	cfg.Color = colorSpec
	cfg.ColorSet = colorSet
	cfg.SVG = makeSVGGlyphs(glyphs, colorSet, colorSpec, r.Meta.Accent)
	if usedBase {
		designer := strings.TrimSpace(r.Meta.Designer)
		if designer == "" {
			designer = "fontcreator233"
		}
		cfg.Copyright = baseCopyright + " Modified by " + designer + "."
	}
	cfg.Jobs = o.Jobs
	ttf, err := fontbuild.BuildContext(ctx, cfg)
	if err != nil {
		return result, fmt.Errorf("compile font: %w", err)
	}
	result.SHA256 = hashHex(ttf)
	result.Bytes = len(ttf)
	result.Glyphs = len(glyphs)

	rep.Event(ui.Event{Kind: ui.EvPhase, Phase: "write"})
	if o.OutDir != "" {
		if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
			return result, err
		}
		stem := fontStem(r.Meta.Family, resolvedStyle)
		fontPath := filepath.Join(o.OutDir, stem+".ttf")
		if err := os.WriteFile(fontPath, ttf, 0o644); err != nil {
			return result, err
		}
		result.FontPath = fontPath
		if hasFormat(r, "woff") {
			woff, err := fontbuild.BuildWOFF(ttf)
			if err != nil {
				return result, fmt.Errorf("compile woff: %w", err)
			}
			woffPath := filepath.Join(o.OutDir, stem+".woff")
			if err := os.WriteFile(woffPath, woff, 0o644); err != nil {
				return result, err
			}
			result.WOFFPath = woffPath
		}
		if hasFormat(r, "woff2") {
			woff2, err := fontbuild.BuildWOFF2(ttf)
			if err != nil {
				return result, fmt.Errorf("compile woff2: %w", err)
			}
			woff2Path := filepath.Join(o.OutDir, stem+".woff2")
			if err := os.WriteFile(woff2Path, woff2, 0o644); err != nil {
				return result, err
			}
			result.WOFF2Path = woff2Path
		}
		if o.SplitCJK > 0 {
			split, infos, err := buildSplits(ctx, r, resolvedStyle, glyphs, splitSet, ligatures, kerning, o, stem)
			if err != nil {
				return result, err
			}
			result.Split = split
			result.SplitFonts = infos
		}
	}
	if r.Output.Specimen && result.FontPath != "" {
		glyphInfo := make([]specimen.GlyphInfo, 0, len(keys))
		for _, rn := range keys {
			glyphInfo = append(glyphInfo, specimen.GlyphInfo{Rune: rn, Script: unicodes.ScriptOf(rn)})
		}
		recipeName := ""
		if r.SourcePath != "" {
			recipeName = filepath.Base(r.SourcePath)
		}
		varMin, varDef, varMax := fontbuild.VariableRange(r.Style.Weight)
		colorRune := 0
		if len(colorRunes) > 0 {
			colorRune = colorRunes[0]
		}
		var colorPalettes []int
		if colorSet != nil {
			for i := range colorSet.Palettes {
				colorPalettes = append(colorPalettes, i)
			}
		}
		html, err := specimen.Build(specimen.Options{
			OutDir:        o.OutDir,
			FontPath:      result.FontPath,
			FontFile:      filepath.Base(result.FontPath),
			AssetsDir:     o.AssetsDir,
			LinkDir:       o.FinalOutDir,
			SelfContained: o.SelfContainedHTML,
			// Opening happens after the workspace commit, against the final
			// path; opening a stage file would leave a dead tab.
			Open: false,
			Data: specimen.Data{
				Family:        r.Meta.Family,
				Style:         resolvedStyle,
				Description:   r.Meta.Description,
				Version:       r.Meta.Version,
				SHA256:        result.SHA256,
				RecipeName:    recipeName,
				Accent:        r.Meta.Accent,
				SampleText:    r.Meta.SampleText,
				Glyphs:        glyphInfo,
				Coverage:      result.Coverage,
				HasKerning:    len(kerning) > 0,
				HasLigatures:  hasLigatureFeature(ligatures, "liga"),
				HasFractions:  hasLigatureFeature(ligatures, "frac"),
				HasCase:       hasSubstitutionFeature(substitutions, "case"),
				HasTnum:       hasSubstitutionFeature(substitutions, "tnum"),
				HasSS01:       hasSubstitutionFeature(substitutions, "ss01"),
				HasSmcp:       hasSubstitutionFeature(substitutions, "smcp"),
				HasOnum:       hasSubstitutionFeature(substitutions, "onum"),
				HasZero:       hasSubstitutionFeature(substitutions, "zero"),
				HasSups:       hasSubstitutionFeature(substitutions, "sups"),
				HasSubs:       hasSubstitutionFeature(substitutions, "subs"),
				HasCcmp:       hasLigatureFeature(ligatures, "ccmp"),
				HasMkmk:       hasMkmk,
				HasCalt:       len(contextual) > 0,
				HasCursive:    r.Output.Cursive,
				HasColor:      colorSpec != nil || colorSet != nil,
				ColorRune:     colorRune,
				ColorRunes:    colorRunes,
				ColorPalettes: colorPalettes,
				HasVariable:   r.Output.Variable,
				VarMin:        varMin,
				VarDefault:    varDef,
				VarMax:        varMax,
				KernPairs:     len(kerning),
			},
		})
		if err != nil {
			warnings = append(warnings, "specimen: "+err.Error())
		} else {
			result.SpecimenPath = html
		}
	}
	if result.FontPath != "" {
		inputs := []targets.FontInput{{
			Path:    result.FontPath,
			WOFF:    result.WOFFPath,
			SHA256:  result.SHA256,
			Primary: true,
			Style:   resolvedStyle,
			Runes:   result.GlyphRunes,
		}}
		for _, sf := range result.SplitFonts {
			woff := strings.TrimSuffix(sf.Path, ".ttf") + ".woff"
			if _, err := os.Stat(woff); err != nil {
				woff = ""
			}
			inputs = append(inputs, targets.FontInput{
				Path:   sf.Path,
				WOFF:   woff,
				Style:  sf.Style,
				Runes:  sf.Runes,
				SHA256: hashFileSHA(sf.Path),
			})
		}
		targetList := o.Targets
		if len(targetList) == 0 {
			targetList = r.Output.Targets
		}
		targetPaths, err := targets.Build(ctx, targets.Options{
			OutDir:         o.OutDir,
			Family:         r.Meta.Family,
			Style:          resolvedStyle,
			RequestedStyle: r.Meta.Style,
			Version:        r.Meta.Version,
			License:        r.Meta.License,
			Weight:         r.Style.Weight,
			Targets:        targetList,
			Fonts:          inputs,
			SpecimenPath:   result.SpecimenPath,
			BMFontPaths:    bmfontPaths,
			AssetsDir:      o.AssetsDir,
			FinalOutDir:    o.FinalOutDir,
			TMP:            r.Output.TMP,
		})
		if err != nil {
			return result, fmt.Errorf("targets: %w", err)
		}
		result.Targets = targetPaths
	}
	if usedBase && r.Style.Weight != 400 {
		warnings = append(warnings, "CJK/Hangul weight is synthetic (base outlines are emboldened, not redrawn)")
	}
	result.Warnings = warnings
	result.Duration = time.Since(start)
	summary := result.Family + " " + result.Style
	if result.SHA256 != "" {
		summary += " · " + result.SHA256[:12]
	}
	rep.Event(ui.Event{
		Kind:     ui.EvDone,
		Warnings: warnings,
		Summary:  summary,
		Duration: result.Duration,
	})
	return result, nil
}

func buildGenerated(ctx context.Context, r *recipe.Recipe, p style.Params, jobs int, rep ui.Reporter) (map[rune]fontbuild.Glyph, error) {
	out := map[rune]fontbuild.Glyph{}
	if len(r.Coverage.Generated) == 0 {
		return out, nil
	}
	builder := skeleton.NewBuilder(p)
	available := skeleton.Available()
	runes := make([]rune, 0, len(available))
	for _, rn := range available {
		if generatedEnabled(rn, r.Coverage.Generated) {
			runes = append(runes, rn)
		}
	}
	glyphs, err := parallel.Map(ctx, runes, jobs, func(_ context.Context, _ int, rn rune) (fontbuild.Glyph, error) {
		path, adv, ok, err := builder.Glyph(rn)
		if err != nil {
			return fontbuild.Glyph{}, err
		}
		if !ok {
			return fontbuild.Glyph{}, nil
		}
		return fontbuild.Glyph{Rune: rn, Advance: int(math.Round(adv)), Path: path}, nil
	})
	if err != nil {
		return nil, err
	}
	for i, rn := range runes {
		if len(glyphs[i].Path) > 0 || rn == 0x20 {
			out[rn] = glyphs[i]
		}
	}
	composed, err := compose.Build(builder)
	if err != nil {
		return nil, err
	}
	for rn, g := range composed {
		if !generatedEnabled(rn, r.Coverage.Generated) {
			continue
		}
		out[rn] = fontbuild.Glyph{Rune: rn, Advance: int(math.Round(g.Advance)), Path: g.Path}
	}
	rep.Event(ui.Event{Kind: ui.EvProgress, Phase: "generate", Current: len(out), Total: len(out)})
	return out, nil
}

func buildDerived(ctx context.Context, r *recipe.Recipe, p style.Params, jobs int, rep ui.Reporter) (map[rune]fontbuild.Glyph, map[string]CoverageStat, []string, error) {
	out := map[rune]fontbuild.Glyph{}
	stats := map[string]CoverageStat{}
	requested := baseRequested(r.Coverage.Base)
	if len(requested) == 0 {
		return out, stats, nil, nil
	}
	set, err := derive.Load()
	if err != nil {
		return nil, nil, nil, err
	}
	if r.Coverage.Base.CJK == "full" {
		// `full` means every glyph actually present in the embedded bases,
		// not every code point in the raw Unicode blocks.
		kept := requested[:0]
		for _, rn := range requested {
			if set.Has(rn) {
				kept = append(kept, rn)
			}
		}
		requested = kept
	}
	embolden := (float64(r.Style.Weight) - 400) / 500 * 30
	if embolden < 0 {
		embolden = 0
	}
	if embolden > 32 {
		embolden = 32
	}
	ds := derive.Style{WidthScale: p.WidthScale, SlantTan: p.SlantTan, Embolden: embolden}
	type derivedResult struct {
		path   geom.Path
		adv    int
		ok     bool
		script string
	}
	var processed atomic.Int64
	results, err := parallel.Map(ctx, requested, jobs, func(_ context.Context, i int, rn rune) (derivedResult, error) {
		if n := processed.Add(1); n%256 == 0 {
			rep.Event(ui.Event{Kind: ui.EvProgress, Phase: "derive", Current: int(n), Total: len(requested)})
		}
		path, adv, ok, err := set.Glyph(rn)
		if err != nil {
			return derivedResult{}, err
		}
		if !ok {
			return derivedResult{script: unicodes.ScriptOf(rn)}, nil
		}
		styled, styledAdv := derive.ApplyStyle(path, adv, ds)
		return derivedResult{path: styled, adv: styledAdv, ok: len(styled) > 0, script: unicodes.ScriptOf(rn)}, nil
	})
	if err != nil {
		return nil, nil, nil, err
	}
	full := r.Coverage.Base.CJK == "full"
	for i, rn := range requested {
		res := results[i]
		if full && !res.ok {
			// An empty cmap glyph (for example the ideographic space)
			// contributes no outline; it is not a missing coverage target.
			continue
		}
		st := stats[res.script]
		st.Requested++
		if !res.ok {
			st.Missing++
			stats[res.script] = st
			continue
		}
		st.Found++
		stats[res.script] = st
		out[rn] = fontbuild.Glyph{Rune: rn, Advance: res.adv, Path: res.path}
	}
	var warnings []string
	keys := make([]string, 0, len(stats))
	for k := range stats {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		st := stats[k]
		if st.Missing > 0 {
			warnings = append(warnings, fmt.Sprintf("base coverage %s: %d of %d requested codepoints missing", k, st.Missing, st.Requested))
		}
	}
	rep.Event(ui.Event{Kind: ui.EvProgress, Phase: "derive", Current: len(requested), Total: len(requested)})
	return out, stats, warnings, nil
}

func baseRequested(c recipe.BaseCoverage) []rune {
	seen := map[rune]bool{}
	var out []rune
	add := func(r rune) {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	addRange := func(lo, hi rune) {
		for r := lo; r <= hi; r++ {
			add(r)
		}
	}
	if c.CJK != "none" {
		runes, err := derive.CoverageRunes(c.CJK)
		if err == nil {
			for _, r := range runes {
				if unicodes.IsCJK(r) {
					add(r)
				}
			}
		}
	}
	if c.Kana {
		addRange(0x3040, 0x30FF)
		addRange(0x31F0, 0x31FF)
		addRange(0xFF66, 0xFF9F)
	}
	if c.Hangul {
		addRange(0x1100, 0x11FF)
		addRange(0x3130, 0x318F)
		addRange(0xA960, 0xA97F)
		addRange(0xAC00, 0xD7A3)
		addRange(0xD7B0, 0xD7FF)
		addRange(0xFFA0, 0xFFDC)
	}
	if c.Symbols {
		addRange(0x2000, 0x206F)
		addRange(0x20A0, 0x20CF)
		addRange(0x2100, 0x214F)
		addRange(0x2190, 0x23FF)
		addRange(0x2500, 0x27BF)
		addRange(0xFE30, 0xFE4F)
		for r := rune(0xFF00); r <= 0xFFEF; r++ {
			if !unicodes.IsKana(r) && !unicodes.IsHangul(r) {
				add(r)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func generatedEnabled(r rune, scripts []string) bool {
	enabled := func(name string) bool {
		for _, s := range scripts {
			if s == name {
				return true
			}
		}
		return false
	}
	switch unicodes.ScriptOf(r) {
	case "latin":
		return enabled("latin")
	case "greek":
		return enabled("greek")
	case "cyrillic":
		return enabled("cyrillic")
	case "other":
		// Combining marks are needed by every generated script.
		return r >= 0x0300 && r <= 0x036F && (enabled("latin") || enabled("greek") || enabled("cyrillic"))
	default:
		return false
	}
}

func fontConfig(r *recipe.Recipe, resolvedStyle string, glyphs []fontbuild.Glyph) fontbuild.Config {
	designer := strings.TrimSpace(r.Meta.Designer)
	if designer == "" {
		designer = "fontcreator233"
	}
	return fontbuild.Config{
		Family:      r.Meta.Family,
		Style:       resolvedStyle,
		Version:     r.Meta.Version,
		Designer:    designer,
		Description: r.Meta.Description,
		License:     r.Meta.License,
		LicenseURL:  licenseURL(r.Meta.License),
		Copyright:   "Copyright 2026 " + designer,
		SampleText:  r.Meta.SampleText,
		Ascender:    r.Metrics.Ascender,
		Descender:   r.Metrics.Descender,
		LineGap:     r.Metrics.LineGap,
		CapHeight:   int(math.Round(r.Metrics.CapHeight * 1000)),
		XHeight:     int(math.Round(r.Metrics.XHeight * 1000)),
		WeightClass: r.Style.Weight,
		WidthClass:  widthClass(r.Style.Width),
		ItalicAngle: r.Style.Slant,
		Created:     buildTime(),
		Variable:    r.Output.Variable,
		Cursive:     r.Output.Cursive,
		Accent:      r.Meta.Accent,
		Glyphs:      glyphs,
	}
}

func buildTime() time.Time {
	if s := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return time.Unix(n, 0).UTC()
		}
	}
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func licenseURL(license string) string {
	if license == "OFL-1.1" {
		return "https://scripts.sil.org/OFL"
	}
	return ""
}

func widthClass(width float64) int {
	switch {
	case width <= 80:
		return 3
	case width <= 90:
		return 4
	case width <= 100:
		return 5
	case width <= 110:
		return 6
	case width <= 125:
		return 7
	default:
		return 5
	}
}

func hasFormat(r *recipe.Recipe, f string) bool {
	for _, x := range r.Output.Formats {
		if x == f {
			return true
		}
	}
	return false
}

func hashFileSHA(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return hashHex(data)
}

func notdefPath() geom.Path {
	outer := geom.RectPath(50, 0, 550, 700)
	inner := geom.RectPath(120, 80, 480, 620).Reverse(0)
	return append(outer, inner...)
}

func hasLigatureFeature(ligatures []fontbuild.Ligature, feature string) bool {
	for _, l := range ligatures {
		f := l.Feature
		if f == "" {
			f = "liga"
		}
		if f == feature {
			return true
		}
	}
	return false
}

func hasCombiningMarks(glyphs map[rune]fontbuild.Glyph) bool {
	count := 0
	for rn, g := range glyphs {
		if rn >= 0x0300 && rn <= 0x036F && len(g.Path) > 0 {
			count++
		}
	}
	return count >= 2
}

func hasSubstitutionFeature(subs []fontbuild.Substitution, feature string) bool {
	for _, s := range subs {
		if s.Feature == feature {
			return true
		}
	}
	return false
}

func makeLigatures(glyphs map[rune]fontbuild.Glyph) ([]fontbuild.Glyph, []fontbuild.Ligature) {
	specs := []struct {
		name  string
		runes []rune
	}{
		{"f_f_i", []rune{'f', 'f', 'i'}},
		{"f_f_l", []rune{'f', 'f', 'l'}},
		{"f_f", []rune{'f', 'f'}},
		{"f_i", []rune{'f', 'i'}},
		{"f_l", []rune{'f', 'l'}},
	}
	var out []fontbuild.Glyph
	var ligs []fontbuild.Ligature
	for _, spec := range specs {
		var path geom.Path
		advance := 0
		ok := true
		for _, rn := range spec.runes {
			g, exists := glyphs[rn]
			if !exists || len(g.Path) == 0 {
				ok = false
				break
			}
			path = append(path, g.Path.Transformed(geom.Translate(float64(advance), 0))...)
			advance += g.Advance
		}
		if !ok || advance <= 0 {
			continue
		}
		out = append(out, fontbuild.Glyph{Name: spec.name, NoCmap: true, Advance: advance, Path: path})
		components := make([]string, len(spec.runes))
		for i, rn := range spec.runes {
			components[i] = glyphName(rn)
		}
		ligs = append(ligs, fontbuild.Ligature{Components: components, Glyph: spec.name})
	}
	return out, ligs
}

// makeComposedLigatures maps base + combining mark sequences to precomposed
// glyphs with a `ccmp` ligature when both the sequence and the target exist.
func makeComposedLigatures(glyphs map[rune]fontbuild.Glyph) []fontbuild.Ligature {
	type compose struct{ base, mark, composed rune }
	pairs := []compose{
		{'a', 0x0300, 0x00E0}, {'a', 0x0301, 0x00E1}, {'a', 0x0302, 0x00E2}, {'a', 0x0303, 0x00E3}, {'a', 0x0308, 0x00E4}, {'a', 0x030A, 0x00E5},
		{'A', 0x0300, 0x00C0}, {'A', 0x0301, 0x00C1}, {'A', 0x0302, 0x00C2}, {'A', 0x0303, 0x00C3}, {'A', 0x0308, 0x00C4}, {'A', 0x030A, 0x00C5},
		{'e', 0x0300, 0x00E8}, {'e', 0x0301, 0x00E9}, {'e', 0x0302, 0x00EA}, {'e', 0x0308, 0x00EB},
		{'E', 0x0300, 0x00C8}, {'E', 0x0301, 0x00C9}, {'E', 0x0302, 0x00CA}, {'E', 0x0308, 0x00CB},
		{'i', 0x0300, 0x00EC}, {'i', 0x0301, 0x00ED}, {'i', 0x0302, 0x00EE}, {'i', 0x0308, 0x00EF},
		{'I', 0x0300, 0x00CC}, {'I', 0x0301, 0x00CD}, {'I', 0x0302, 0x00CE}, {'I', 0x0308, 0x00CF},
		{'o', 0x0300, 0x00F2}, {'o', 0x0301, 0x00F3}, {'o', 0x0302, 0x00F4}, {'o', 0x0303, 0x00F5}, {'o', 0x0308, 0x00F6},
		{'O', 0x0300, 0x00D2}, {'O', 0x0301, 0x00D3}, {'O', 0x0302, 0x00D4}, {'O', 0x0303, 0x00D5}, {'O', 0x0308, 0x00D6},
		{'u', 0x0300, 0x00F9}, {'u', 0x0301, 0x00FA}, {'u', 0x0302, 0x00FB}, {'u', 0x0308, 0x00FC},
		{'U', 0x0300, 0x00D9}, {'U', 0x0301, 0x00DA}, {'U', 0x0302, 0x00DB}, {'U', 0x0308, 0x00DC},
		{'n', 0x0303, 0x00F1}, {'N', 0x0303, 0x00D1},
		{'c', 0x0327, 0x00E7}, {'C', 0x0327, 0x00C7},
		{'y', 0x0301, 0x00FD}, {'y', 0x0308, 0x00FF}, {'Y', 0x0308, 0x0178},
		{'s', 0x030C, 0x0161}, {'S', 0x030C, 0x0160},
		{'z', 0x030C, 0x017E}, {'Z', 0x030C, 0x017D},
		{'r', 0x030C, 0x0159}, {'R', 0x030C, 0x0158},
		{'l', 0x0301, 0x013A}, {'L', 0x0301, 0x0139},
		{'g', 0x0306, 0x011F}, {'G', 0x0306, 0x011E},
	}
	var out []fontbuild.Ligature
	for _, p := range pairs {
		base, okB := glyphs[p.base]
		mark, okM := glyphs[p.mark]
		composed, okC := glyphs[p.composed]
		if !okB || !okM || !okC || len(base.Path) == 0 || len(mark.Path) == 0 || len(composed.Path) == 0 {
			continue
		}
		out = append(out, fontbuild.Ligature{
			Feature:    "ccmp",
			Components: []string{glyphName(p.base), glyphName(p.mark)},
			Glyph:      glyphName(p.composed),
		})
	}
	return out
}

func makeSubstitutions(glyphs map[rune]fontbuild.Glyph, capHeight, xHeight int) ([]fontbuild.Glyph, []fontbuild.Substitution) {
	var out []fontbuild.Glyph
	var subs []fontbuild.Substitution
	digits := []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	maxAdvance := 0
	for _, rn := range digits {
		if g, ok := glyphs[rn]; ok && g.Advance > maxAdvance {
			maxAdvance = g.Advance
		}
	}
	for _, rn := range digits {
		g, ok := glyphs[rn]
		if !ok || len(g.Path) == 0 {
			continue
		}
		path := g.Path.Clone()
		if maxAdvance > g.Advance {
			path = path.Transformed(geom.Translate(float64(maxAdvance-g.Advance)/2, 0))
		}
		name := glyphName(rn) + ".tnum"
		out = append(out, fontbuild.Glyph{Name: name, NoCmap: true, Advance: maxAdvance, Path: path})
		subs = append(subs, fontbuild.Substitution{Feature: "tnum", From: glyphName(rn), To: name})
	}

	caseRunes := []rune{'-', 0x2013, 0x2014, '(', ')', '[', ']', '{', '}', 0x00AB, 0x00BB, 0x2039, 0x203A}
	for _, rn := range caseRunes {
		g, ok := glyphs[rn]
		if !ok || len(g.Path) == 0 {
			continue
		}
		b := g.Path.Bounds()
		center := (b.MinY + b.MaxY) / 2
		shift := float64(capHeight)/2 - center
		name := glyphName(rn) + ".case"
		out = append(out, fontbuild.Glyph{Name: name, NoCmap: true, Advance: g.Advance, Path: g.Path.Transformed(geom.Translate(0, shift))})
		subs = append(subs, fontbuild.Substitution{Feature: "case", From: glyphName(rn), To: name})
	}
	if g, ok := glyphs['a']; ok && len(g.Path) > 0 && g.Advance > 0 {
		path := g.Path.Transformed(geom.Translate(-float64(g.Advance)/2, 0))
		path = path.Transformed(geom.Scale(1.06, 1))
		path = path.Transformed(geom.Translate(float64(g.Advance)/2, 0))
		name := glyphName('a') + ".alt"
		out = append(out, fontbuild.Glyph{Name: name, NoCmap: true, Advance: g.Advance, Path: path})
		subs = append(subs, fontbuild.Substitution{Feature: "ss01", From: glyphName('a'), To: name})
	}

	// Small caps: scale uppercase outlines down to x-height.
	smcpScale := 0.72
	if capHeight > 0 && xHeight > 0 {
		smcpScale = float64(xHeight) / float64(capHeight)
	}
	if smcpScale < 0.6 {
		smcpScale = 0.6
	}
	if smcpScale > 0.85 {
		smcpScale = 0.85
	}
	for rn := 'a'; rn <= 'z'; rn++ {
		lower, okLower := glyphs[rn]
		upper, okUpper := glyphs[rn-32]
		if !okLower || !okUpper || len(lower.Path) == 0 || len(upper.Path) == 0 || upper.Advance <= 0 {
			continue
		}
		path := upper.Path.Transformed(geom.Scale(smcpScale, smcpScale))
		advance := int(math.Round(float64(upper.Advance) * smcpScale))
		name := glyphName(rn) + ".smcp"
		out = append(out, fontbuild.Glyph{Name: name, NoCmap: true, Advance: advance, Path: path})
		subs = append(subs, fontbuild.Substitution{Feature: "smcp", From: glyphName(rn), To: name})
	}

	// Oldstyle figures: approximate with a smaller size and vertical offsets.
	oldstyleScale := 0.82
	oldstyleDescenders := map[rune]bool{'3': true, '4': true, '5': true, '7': true, '9': true}
	oldstyleAscenders := map[rune]bool{'6': true, '8': true}
	for _, rn := range digits {
		g, ok := glyphs[rn]
		if !ok || len(g.Path) == 0 {
			continue
		}
		path := g.Path.Transformed(geom.Scale(oldstyleScale, oldstyleScale))
		switch {
		case oldstyleDescenders[rn]:
			path = path.Transformed(geom.Translate(0, -150))
		case oldstyleAscenders[rn]:
			path = path.Transformed(geom.Translate(0, 110))
		}
		advance := int(math.Round(float64(g.Advance) * oldstyleScale))
		name := glyphName(rn) + ".onum"
		out = append(out, fontbuild.Glyph{Name: name, NoCmap: true, Advance: advance, Path: path})
		subs = append(subs, fontbuild.Substitution{Feature: "onum", From: glyphName(rn), To: name})
	}

	// Superscript and subscript figures.
	for _, rn := range digits {
		g, ok := glyphs[rn]
		if !ok || len(g.Path) == 0 {
			continue
		}
		advance := int(math.Round(float64(g.Advance) * 0.62))
		supsPath := g.Path.Transformed(geom.Scale(0.62, 0.62)).Transformed(geom.Translate(0, 360))
		supsName := glyphName(rn) + ".sups"
		out = append(out, fontbuild.Glyph{Name: supsName, NoCmap: true, Advance: advance, Path: supsPath})
		subs = append(subs, fontbuild.Substitution{Feature: "sups", From: glyphName(rn), To: supsName})

		subsPath := g.Path.Transformed(geom.Scale(0.62, 0.62)).Transformed(geom.Translate(0, -90))
		subsName := glyphName(rn) + ".subs"
		out = append(out, fontbuild.Glyph{Name: subsName, NoCmap: true, Advance: advance, Path: subsPath})
		subs = append(subs, fontbuild.Substitution{Feature: "subs", From: glyphName(rn), To: subsName})
	}

	// Slashed zero.
	if zero, ok := glyphs['0']; ok && len(zero.Path) > 0 {
		b := zero.Path.Bounds()
		w, h := b.Width(), b.Height()
		if w > 0 && h > 0 {
			path := zero.Path.Clone()
			p1 := geom.Point{X: b.MinX + w*0.22, Y: b.MinY + h*0.06}
			p2 := geom.Point{X: b.MaxX - w*0.22, Y: b.MaxY - h*0.06}
			dir := p2.Sub(p1).Normalize()
			normal := geom.Point{X: -dir.Y, Y: dir.X}.Mul(math.Max(w, h) * 0.045)
			bar := geom.Contour{
				{P: p1.Add(normal), On: true},
				{P: p2.Add(normal), On: true},
				{P: p2.Sub(normal), On: true},
				{P: p1.Sub(normal), On: true},
			}
			path = append(path, bar)
			name := glyphName('0') + ".zero"
			out = append(out, fontbuild.Glyph{Name: name, NoCmap: true, Advance: zero.Advance, Path: path})
			subs = append(subs, fontbuild.Substitution{Feature: "zero", From: glyphName('0'), To: name})
		}
	}
	return out, subs
}

func recipeContextualRules(rules []recipe.ContextualRule, glyphs map[rune]fontbuild.Glyph, warnings *[]string) []fontbuild.ContextualRule {
	var out []fontbuild.ContextualRule
	for _, rule := range rules {
		feature := rule.Feature
		if feature == "" {
			feature = "calt"
		}
		target := []rune(rule.Target)
		if len(target) != 1 {
			continue
		}
		targetName := glyphName(target[0])
		if _, ok := glyphs[target[0]]; !ok {
			*warnings = append(*warnings, fmt.Sprintf("contextual target %U is not in the font; rule skipped", target[0]))
			continue
		}
		toGlyphs := func(text string, field string) ([]string, bool) {
			runes := []rune(text)
			out := make([]string, 0, len(runes))
			for _, rn := range runes {
				name := glyphName(rn)
				if _, ok := glyphs[rn]; !ok {
					*warnings = append(*warnings, fmt.Sprintf("contextual %s %U is not in the font; rule skipped", field, rn))
					return nil, false
				}
				out = append(out, name)
			}
			return out, true
		}
		components, okInput := toGlyphs(rule.Input, "input")
		if !okInput || len(components) < 2 {
			continue
		}
		backtrack, okBack := toGlyphs(rule.Backtrack, "backtrack")
		if !okBack {
			continue
		}
		lookahead, okLook := toGlyphs(rule.Lookahead, "lookahead")
		if !okLook {
			continue
		}
		out = append(out, fontbuild.ContextualRule{
			Feature:   feature,
			Backtrack: backtrack,
			Input:     components,
			Lookahead: lookahead,
			Glyph:     targetName,
		})
	}
	return out
}

func arrowPath() geom.Path {
	shaft := geom.RectPath(120, 430, 480, 510)
	head := geom.Path{geom.Contour{
		{P: geom.Point{X: 400, Y: 360}, On: true},
		{P: geom.Point{X: 590, Y: 470}, On: true},
		{P: geom.Point{X: 400, Y: 580}, On: true},
	}}
	return append(shaft, head...)
}

// makeArrowRule adds a real contextual `calt` rule: the exact sequence
// hyphen + greater is replaced by an arrow only when that context is present.
func makeArrowRule(glyphs map[rune]fontbuild.Glyph) ([]fontbuild.Glyph, []fontbuild.ContextualRule) {
	dash, okDash := glyphs['-']
	greater, okGreater := glyphs['>']
	if !okDash || !okGreater || len(dash.Path) == 0 || len(greater.Path) == 0 {
		return nil, nil
	}
	arrow := fontbuild.Glyph{Name: "context.arrow", NoCmap: true, Advance: 600, Path: arrowPath()}
	rule := fontbuild.ContextualRule{
		Feature: "calt",
		Input:   []string{glyphName('-'), glyphName('>')},
		Glyph:   "context.arrow",
	}
	return []fontbuild.Glyph{arrow}, []fontbuild.ContextualRule{rule}
}

// makeSVGGlyphs derives SVG color glyph documents from the recipe color set or
// the single demo glyph. The first palette supplies the layer colors.
func makeSVGGlyphs(all []fontbuild.Glyph, set *fontbuild.ColorSet, demo *fontbuild.ColorGlyph, accent string) []fontbuild.SVGColorGlyph {
	byName := make(map[string]fontbuild.Glyph, len(all))
	for _, g := range all {
		byName[g.Name] = g
	}
	var out []fontbuild.SVGColorGlyph
	if set != nil && len(set.Glyphs) > 0 && len(set.Palettes) > 0 {
		pal := set.Palettes[0]
		for _, cg := range set.Glyphs {
			entry := fontbuild.SVGColorGlyph{Base: cg.Base}
			for _, layer := range cg.Layers {
				g, ok := byName[layer.Glyph]
				if !ok || int(layer.PaletteIndex) >= len(pal) {
					continue
				}
				c := pal[layer.PaletteIndex]
				entry.Layers = append(entry.Layers, fontbuild.SVGColorLayer{
					Path:  g.Path,
					Color: [3]byte{c[2], c[1], c[0]}, // BGRA -> RGB
				})
			}
			if len(entry.Layers) > 0 {
				out = append(out, entry)
			}
		}
	} else if demo != nil {
		colors := [][3]byte{
			accentRGBLocal(accent),
			{0xFF, 0xFF, 0xFF},
			{0x00, 0x00, 0x00},
		}
		entry := fontbuild.SVGColorGlyph{Base: demo.Base}
		for _, layer := range demo.Layers {
			g, ok := byName[layer.Glyph]
			if !ok || int(layer.PaletteIndex) >= len(colors) {
				continue
			}
			entry.Layers = append(entry.Layers, fontbuild.SVGColorLayer{
				Path: g.Path, Color: colors[layer.PaletteIndex],
			})
		}
		if len(entry.Layers) > 0 {
			out = append(out, entry)
		}
	}
	return out
}

func accentRGBLocal(s string) [3]byte {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return [3]byte{0x5B, 0x8C, 0xFF}
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return [3]byte{0x5B, 0x8C, 0xFF}
	}
	return [3]byte{byte(v >> 16), byte(v >> 8), byte(v)}
}

func wantsBMFont(o Options, r *recipe.Recipe) bool {
	if r.Output.BMFont != nil {
		return true
	}
	list := o.Targets
	if len(list) == 0 {
		list = r.Output.Targets
	}
	for _, t := range list {
		if t == "bmfont" || t == "all" {
			return true
		}
	}
	return false
}

func makeBMFont(outDir string, cfg recipe.BMFontConfig, r *recipe.Recipe, resolvedStyle string, glyphs []fontbuild.Glyph, kern []fontbuild.KernPair) ([]string, error) {
	base := strings.Map(func(ch rune) rune {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' {
			return ch
		}
		return -1
	}, r.Meta.Family+"-"+resolvedStyle)
	if base == "" {
		base = "Font"
	}
	src := make([]bmfont.Glyph, 0, len(glyphs))
	for _, g := range glyphs {
		if g.Rune == 0 {
			continue
		}
		src = append(src, bmfont.Glyph{Rune: g.Rune, Name: g.Name, Advance: g.Advance, Path: g.Path})
	}
	pairs := make([]bmfont.KernPair, 0, len(kern))
	for _, k := range kern {
		pairs = append(pairs, bmfont.KernPair{Left: k.Left, Right: k.Right, Value: k.Value})
	}
	return bmfont.Build(bmfont.Options{
		OutDir:    outDir,
		BaseName:  base,
		Size:      cfg.Size,
		Padding:   cfg.Padding,
		AtlasSize: cfg.AtlasSize,
		Charset:   cfg.Charset,
		Ascender:  r.Metrics.Ascender,
		Descender: r.Metrics.Descender,
		LineGap:   r.Metrics.LineGap,
		Weight:    r.Style.Weight,
		Italic:    math.Abs(r.Style.Slant) > 0.01,
	}, src, pairs)
}

// colorShapePath builds a primitive outline at the standard color-glyph center.
func colorShapePath(shape string, scale float64, x, y float64) geom.Path {
	const cx, cy, r = 300.0, 350.0, 250.0
	var p geom.Path
	switch shape {
	case "square":
		p = geom.RectPath(cx-r, cy-r, cx+r, cy+r)
	case "triangle":
		p = geom.Path{{
			{P: geom.Point{X: cx, Y: cy + r}, On: true},
			{P: geom.Point{X: cx - r, Y: cy - r}, On: true},
			{P: geom.Point{X: cx + r, Y: cy - r}, On: true},
		}}
	case "star":
		contour := make(geom.Contour, 10)
		for i := range contour {
			angle := math.Pi/2 + float64(i)*math.Pi/5
			radius := r
			if i%2 == 1 {
				radius = r * 0.42
			}
			contour[i] = geom.Node{P: geom.Point{X: cx + radius*math.Cos(angle), Y: cy + radius*math.Sin(angle)}, On: true}
		}
		p = geom.Path{contour}
	case "ring":
		p = append(geom.Circle(cx, cy, r), geom.Circle(cx, cy, r*0.6).Reverse(0)...)
	case "circle", "":
		p = geom.Circle(cx, cy, r)
	default:
		p = geom.Circle(cx, cy, r)
	}
	if scale <= 0 {
		scale = 1
	}
	return p.Transformed(geom.Translate(-cx, -cy)).
		Transformed(geom.Scale(scale, scale)).
		Transformed(geom.Translate(cx+x, cy+y))
}

// recipeColorGlyphs turns openType.color into a real multi-glyph, multi-palette
// COLR/CPAL payload. It returns the generated layer/base glyphs, the ColorSet,
// and the codepoints that should be shown in the specimen.
func recipeColorGlyphs(cfg *recipe.ColorConfig, glyphs map[rune]fontbuild.Glyph, warnings *[]string) ([]fontbuild.Glyph, *fontbuild.ColorSet, []int) {
	if cfg == nil || len(cfg.Glyphs) == 0 || len(cfg.Palettes) == 0 {
		return nil, nil, nil
	}
	set := &fontbuild.ColorSet{}
	for _, palette := range cfg.Palettes {
		record := make([][4]byte, len(palette))
		for i, col := range palette {
			record[i] = fontbuild.ParseColorRecord(col)
		}
		set.Palettes = append(set.Palettes, record)
	}
	var out []fontbuild.Glyph
	var runes []int
	for gi, rule := range cfg.Glyphs {
		cp, err := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(rule.Codepoint), "U+"), 16, 32)
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("color glyph %d has invalid codepoint %q; skipped", gi, rule.Codepoint))
			continue
		}
		rn := rune(cp)
		if _, exists := glyphs[rn]; exists {
			*warnings = append(*warnings, fmt.Sprintf("color glyph U+%04X already exists; skipped", rn))
			continue
		}
		name := glyphName(rn)
		advance := rule.Advance
		if advance == 0 {
			advance = 600
		}
		var base geom.Path
		var layers []fontbuild.ColorLayer
		for li, layerRule := range rule.Layers {
			path := colorShapePath(layerRule.Shape, layerRule.Scale, layerRule.X, layerRule.Y)
			if li == 0 {
				base = path
			}
			layerName := fmt.Sprintf("color.g%d.l%d", gi, li)
			out = append(out, fontbuild.Glyph{Name: layerName, NoCmap: true, Advance: advance, Path: path})
			layers = append(layers, fontbuild.ColorLayer{Glyph: layerName, PaletteIndex: uint16(layerRule.Color)})
		}
		out = append(out, fontbuild.Glyph{Rune: rn, Name: name, Advance: advance, Path: base})
		set.Glyphs = append(set.Glyphs, fontbuild.ColorGlyph{Base: name, Rune: rn, Layers: layers})
		runes = append(runes, int(rn))
	}
	if len(set.Glyphs) == 0 {
		return nil, nil, nil
	}
	return out, set, runes
}

// makeColorGlyph adds a self-contained COLR/CPAL accent glyph. It is a
// demonstration color glyph, not arbitrary color-font authoring.
func makeColorGlyph(glyphs map[rune]fontbuild.Glyph) ([]fontbuild.Glyph, *fontbuild.ColorGlyph) {
	baseRune := rune(0)
	for _, candidate := range []rune{0xE000, 0xF8FF, 0xF0000} {
		if _, exists := glyphs[candidate]; !exists {
			baseRune = candidate
			break
		}
	}
	if baseRune == 0 {
		return nil, nil
	}
	baseName := glyphName(baseRune)
	base := fontbuild.Glyph{Rune: baseRune, Name: baseName, Advance: 600, Path: geom.Circle(300, 350, 250)}
	outer := fontbuild.Glyph{Name: "color.layer0", NoCmap: true, Advance: 600, Path: geom.Circle(300, 350, 250)}
	inner := fontbuild.Glyph{Name: "color.layer1", NoCmap: true, Advance: 600, Path: geom.Circle(300, 350, 105)}
	spec := &fontbuild.ColorGlyph{
		Base: baseName,
		Rune: baseRune,
		Layers: []fontbuild.ColorLayer{
			{Glyph: "color.layer0", PaletteIndex: 0},
			{Glyph: "color.layer1", PaletteIndex: 1},
		},
	}
	return []fontbuild.Glyph{base, outer, inner}, spec
}

func makeFractions(glyphs map[rune]fontbuild.Glyph) ([]fontbuild.Glyph, []fontbuild.Ligature) {
	specs := []struct {
		name string
		num  rune
		den  rune
	}{
		{"fraction_one_half", '1', '2'},
		{"fraction_one_third", '1', '3'},
		{"fraction_two_thirds", '2', '3'},
		{"fraction_one_quarter", '1', '4'},
		{"fraction_three_quarters", '3', '4'},
		{"fraction_one_eighth", '1', '8'},
		{"fraction_three_eighths", '3', '8'},
		{"fraction_five_eighths", '5', '8'},
		{"fraction_seven_eighths", '7', '8'},
	}
	slash, hasSlash := glyphs['/']
	if !hasSlash || len(slash.Path) == 0 {
		return nil, nil
	}
	const scale = 0.55
	var out []fontbuild.Glyph
	var ligs []fontbuild.Ligature
	for _, spec := range specs {
		num, okN := glyphs[spec.num]
		den, okD := glyphs[spec.den]
		if !okN || !okD || len(num.Path) == 0 || len(den.Path) == 0 {
			continue
		}
		var path geom.Path
		path = append(path, num.Path.Transformed(geom.Scale(scale, scale)).Transformed(geom.Translate(0, 280))...)
		path = append(path, slash.Path.Transformed(geom.Scale(scale*0.8, scale*0.8)).Transformed(geom.Translate(250, 0))...)
		path = append(path, den.Path.Transformed(geom.Scale(scale, scale)).Transformed(geom.Translate(420, 0))...)
		out = append(out, fontbuild.Glyph{Name: spec.name, NoCmap: true, Advance: 780, Path: path})
		ligs = append(ligs, fontbuild.Ligature{
			Feature:    "frac",
			Components: []string{glyphName(spec.num), glyphName('/'), glyphName(spec.den)},
			Glyph:      spec.name,
		})
	}
	return out, ligs
}

func makeKerning(glyphs map[rune]fontbuild.Glyph, weight int) []fontbuild.KernPair {
	type pair struct {
		left, right rune
		value       int16
	}
	pairs := []pair{
		{'A', 'V', -80}, {'A', 'W', -70}, {'A', 'Y', -90}, {'A', 'T', -60},
		{'F', 'A', -70}, {'L', 'T', -70}, {'L', 'V', -70}, {'L', 'W', -60},
		{'L', 'Y', -80}, {'P', 'A', -60}, {'T', 'A', -80}, {'T', 'o', -50},
		{'T', 'e', -45}, {'V', 'a', -50}, {'V', 'o', -50}, {'W', 'a', -40},
		{'W', 'o', -40}, {'Y', 'a', -60}, {'Y', 'o', -60}, {'F', 'i', -30},
		{'F', 'l', -30}, {'P', 'e', -40}, {'P', 'o', -40},
	}
	scale := float64(weight) / 400
	if scale < 0.65 {
		scale = 0.65
	}
	if scale > 1.5 {
		scale = 1.5
	}
	var out []fontbuild.KernPair
	for _, p := range pairs {
		if _, ok := glyphs[p.left]; !ok {
			continue
		}
		if _, ok := glyphs[p.right]; !ok {
			continue
		}
		out = append(out, fontbuild.KernPair{
			Left: glyphName(p.left), Right: glyphName(p.right),
			Value: int16(math.Round(float64(p.value) * scale)),
		})
	}
	return out
}

func glyphName(r rune) string {
	if r <= 0xFFFF {
		return fmt.Sprintf("uni%04X", r)
	}
	return fmt.Sprintf("u%06X", r)
}

func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

const baseCopyright = "Copyright 2014-2021 Adobe (http://www.adobe.com/), with Reserved Font Name 'Source'. Derived from Noto Sans SC/KR for fontcreator233 and renamed per the SIL Open Font License 1.1."

// FontStem returns the filesystem-safe stem for a family/style pair.
func FontStem(family, style string) string { return fontStem(family, style) }

func fontStem(family, style string) string {
	clean := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			switch {
			case r == ' ':
			case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' || r == '&' || r == '#':
				b.WriteRune('-')
			default:
				b.WriteRune(r)
			}
		}
		return strings.Trim(b.String(), "-")
	}
	f, s := clean(family), clean(style)
	if f == "" {
		f = "Font"
	}
	stem := f
	if s != "" {
		stem += "-" + s
	}
	stem = strings.TrimRight(stem, " .")
	if isWindowsReserved(stem) {
		stem = "font-" + stem
	}
	return stem
}

func isWindowsReserved(name string) bool {
	switch strings.ToUpper(name) {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	upper := strings.ToUpper(name)
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) {
		return upper[3] >= '1' && upper[3] <= '9'
	}
	return false
}

func buildSplits(ctx context.Context, r *recipe.Recipe, resolvedStyle string, glyphs []fontbuild.Glyph, splitSet map[rune]bool, ligatures []fontbuild.Ligature, kerning []fontbuild.KernPair, o Options, stem string) ([]string, []SplitFont, error) {
	splitRunes := make([]rune, 0, len(splitSet))
	for rn := range splitSet {
		splitRunes = append(splitRunes, rn)
	}
	sort.Slice(splitRunes, func(i, j int) bool { return splitRunes[i] < splitRunes[j] })
	if len(splitRunes) == 0 {
		return nil, nil, nil
	}
	var paths []string
	var infos []SplitFont
	for start, idx := 0, 1; start < len(splitRunes); start, idx = start+o.SplitCJK, idx+1 {
		end := start + o.SplitCJK
		if end > len(splitRunes) {
			end = len(splitRunes)
		}
		chunk := map[rune]bool{}
		for _, rn := range splitRunes[start:end] {
			chunk[rn] = true
		}
		var list []fontbuild.Glyph
		list = append(list, glyphs[0])
		for _, g := range glyphs[1:] {
			if splitSet[g.Rune] && !chunk[g.Rune] {
				continue
			}
			list = append(list, g)
		}
		styleName := fmt.Sprintf("%s CJK %02d", resolvedStyle, idx)
		cfg := fontConfig(r, styleName, list)
		cfg.Ligatures = ligatures
		cfg.Kerning = kerning
		// CJK split chunks stay static; the base CJK outlines do not vary.
		cfg.Variable = false
		cfg.Jobs = o.Jobs
		data, err := fontbuild.BuildContext(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		path := filepath.Join(o.OutDir, fmt.Sprintf("%s-CJK-%02d.ttf", stem, idx))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, nil, err
		}
		paths = append(paths, path)
		infos = append(infos, SplitFont{Path: path, Style: styleName, Runes: append([]rune(nil), splitRunes[start:end]...)})
		if hasFormat(r, "woff") {
			woff, err := fontbuild.BuildWOFF(data)
			if err != nil {
				return nil, nil, err
			}
			if err := os.WriteFile(filepath.Join(o.OutDir, fmt.Sprintf("%s-CJK-%02d.woff", stem, idx)), woff, 0o644); err != nil {
				return nil, nil, err
			}
		}
		if hasFormat(r, "woff2") {
			woff2, err := fontbuild.BuildWOFF2(data)
			if err != nil {
				return nil, nil, err
			}
			if err := os.WriteFile(filepath.Join(o.OutDir, fmt.Sprintf("%s-CJK-%02d.woff2", stem, idx)), woff2, 0o644); err != nil {
				return nil, nil, err
			}
		}
	}
	return paths, infos, nil
}
