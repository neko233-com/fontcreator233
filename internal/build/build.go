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
	"strings"
	"time"

	"github.com/neko233-com/fontcreator233/internal/compose"
	"github.com/neko233-com/fontcreator233/internal/derive"
	"github.com/neko233-com/fontcreator233/internal/fontbuild"
	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/skeleton"
	"github.com/neko233-com/fontcreator233/internal/style"
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
	SelfContainedHTML bool
}

// CoverageStat reports requested versus found code points.
type CoverageStat struct {
	Requested int
	Found     int
	Missing   int
}

// Result is the outcome of a build.
type Result struct {
	Family         string
	Style          string
	RequestedStyle string
	FontPath       string
	WOFFPath       string
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
}

// Build compiles one recipe. The workspace layer wraps this in a staging
// directory and commits the result.
func Build(ctx context.Context, r *recipe.Recipe, o Options, rep ui.Reporter) (Result, error) {
	start := time.Now()
	if rep == nil {
		rep = ui.NopReporter{}
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
	rep.Event(ui.Event{Kind: ui.EvPhase, Phase: "generate"})
	gen, err := buildGenerated(ctx, r, params, rep)
	if err != nil {
		return result, err
	}
	rep.Event(ui.Event{Kind: ui.EvPhase, Phase: "derive"})
	derived, stats, deriveWarnings, err := buildDerived(ctx, r, params, rep)
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

	if r.Glyphs.Overrides != "" {
		return result, fmt.Errorf("glyphs.overrides is not implemented yet")
	}

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
	cfg := fontConfig(r, resolvedStyle, glyphs)
	ttf, err := fontbuild.Build(cfg)
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
		if o.SplitCJK > 0 {
			split, err := buildSplits(ctx, r, resolvedStyle, glyphs, merged, derived, o, stem)
			if err != nil {
				return result, err
			}
			result.Split = split
		}
	}
	if r.Output.Specimen {
		warnings = append(warnings, "specimen generation is not wired yet")
	}
	if !onlyUniversal(r.Output.Targets) {
		warnings = append(warnings, "platform target generation is not wired yet")
	}
	if hasBaseGlyphs(derived) && r.Style.Weight != 400 {
		warnings = append(warnings, "CJK/Hangul weight is synthetic (base outlines are emboldened, not redrawn)")
	}
	result.Warnings = warnings
	result.Duration = time.Since(start)
	rep.Event(ui.Event{
		Kind:     ui.EvDone,
		Warnings: warnings,
		Duration: result.Duration,
	})
	return result, nil
}

func buildGenerated(ctx context.Context, r *recipe.Recipe, p style.Params, rep ui.Reporter) (map[rune]fontbuild.Glyph, error) {
	out := map[rune]fontbuild.Glyph{}
	if len(r.Coverage.Generated) == 0 {
		return out, nil
	}
	builder := skeleton.NewBuilder(p)
	available := skeleton.Available()
	for i, rn := range available {
		if i%512 == 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}
		if !generatedEnabled(rn, r.Coverage.Generated) {
			continue
		}
		path, adv, ok, err := builder.Glyph(rn)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		out[rn] = fontbuild.Glyph{Rune: rn, Advance: int(math.Round(adv)), Path: path}
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

func buildDerived(ctx context.Context, r *recipe.Recipe, p style.Params, rep ui.Reporter) (map[rune]fontbuild.Glyph, map[string]CoverageStat, []string, error) {
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
	embolden := (float64(r.Style.Weight) - 400) / 500 * 30
	if embolden < 0 {
		embolden = 0
	}
	if embolden > 32 {
		embolden = 32
	}
	ds := derive.Style{WidthScale: p.WidthScale, SlantTan: p.SlantTan, Embolden: embolden}
	for i, rn := range requested {
		if i%256 == 0 {
			select {
			case <-ctx.Done():
				return nil, nil, nil, ctx.Err()
			default:
			}
			rep.Event(ui.Event{Kind: ui.EvProgress, Phase: "derive", Current: i, Total: len(requested)})
		}
		script := unicodes.ScriptOf(rn)
		st := stats[script]
		st.Requested++
		path, adv, ok, err := set.Glyph(rn)
		if err != nil {
			return nil, nil, nil, err
		}
		if !ok {
			st.Missing++
			stats[script] = st
			continue
		}
		styled, styledAdv := derive.ApplyStyle(path, adv, ds)
		if len(styled) == 0 {
			st.Missing++
			stats[script] = st
			continue
		}
		st.Found++
		stats[script] = st
		out[rn] = fontbuild.Glyph{Rune: rn, Advance: styledAdv, Path: styled}
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
	return fontbuild.Config{
		Family:      r.Meta.Family,
		Style:       resolvedStyle,
		Version:     r.Meta.Version,
		Designer:    r.Meta.Designer,
		Description: r.Meta.Description,
		License:     r.Meta.License,
		LicenseURL:  "https://scripts.sil.org/OFL",
		Copyright:   "Copyright 2026 " + r.Meta.Designer,
		SampleText:  r.Meta.SampleText,
		Ascender:    r.Metrics.Ascender,
		Descender:   r.Metrics.Descender,
		LineGap:     r.Metrics.LineGap,
		CapHeight:   int(math.Round(r.Metrics.CapHeight * 1000)),
		XHeight:     int(math.Round(r.Metrics.XHeight * 1000)),
		WeightClass: r.Style.Weight,
		WidthClass:  widthClass(r.Style.Width),
		ItalicAngle: r.Style.Slant,
		Created:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Glyphs:      glyphs,
	}
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

func onlyUniversal(targets []string) bool {
	for _, t := range targets {
		if t != "universal" {
			return false
		}
	}
	return true
}

func hasBaseGlyphs(m map[rune]fontbuild.Glyph) bool {
	return len(m) > 0
}

func notdefPath() geom.Path {
	outer := geom.RectPath(50, 0, 550, 700)
	inner := geom.RectPath(120, 80, 480, 620).Reverse(0)
	return append(outer, inner...)
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

func fontStem(family, style string) string {
	clean := func(s string) string {
		var b strings.Builder
		for _, r := range s {
			switch {
			case r == ' ':
			case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
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
	if s == "" {
		return f
	}
	return f + "-" + s
}

func buildSplits(ctx context.Context, r *recipe.Recipe, resolvedStyle string, glyphs []fontbuild.Glyph, merged map[rune]fontbuild.Glyph, derived map[rune]fontbuild.Glyph, o Options, stem string) ([]string, error) {
	var splitRunes []rune
	for rn := range derived {
		if rn >= 0x2E80 {
			splitRunes = append(splitRunes, rn)
		}
	}
	sort.Slice(splitRunes, func(i, j int) bool { return splitRunes[i] < splitRunes[j] })
	if len(splitRunes) == 0 {
		return nil, nil
	}
	var paths []string
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
			if _, isSplit := derived[g.Rune]; isSplit && g.Rune >= 0x2E80 && !chunk[g.Rune] {
				continue
			}
			list = append(list, g)
		}
		styleName := fmt.Sprintf("%s CJK %02d", resolvedStyle, idx)
		cfg := fontConfig(r, styleName, list)
		data, err := fontbuild.Build(cfg)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(o.OutDir, fmt.Sprintf("%s-CJK-%02d.ttf", stem, idx))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return nil, err
		}
		paths = append(paths, path)
		if hasFormat(r, "woff") {
			woff, err := fontbuild.BuildWOFF(data)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(o.OutDir, fmt.Sprintf("%s-CJK-%02d.woff", stem, idx)), woff, 0o644); err != nil {
				return nil, err
			}
		}
	}
	return paths, nil
}
