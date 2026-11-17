// Package compose builds precomposed accented characters from skeleton base
// glyphs and combining marks using Unicode canonical decomposition.
package compose

import (
	"sort"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/skeleton"
	"golang.org/x/text/unicode/norm"
)

// Glyph is a composed outline and advance.
type Glyph struct {
	Path    geom.Path
	Advance float64
}

// Ranges are the Unicode blocks scanned for precomposed characters.
var Ranges = [][2]rune{
	{0x00C0, 0x024F},
	{0x0370, 0x03FF},
	{0x0400, 0x04FF},
	{0x1E00, 0x1EFF},
}

// Build returns all composable precomposed characters keyed by rune.
func Build(b *skeleton.Builder) (map[rune]Glyph, error) {
	out := map[rune]Glyph{}
	for _, rg := range Ranges {
		for r := rg[0]; r <= rg[1]; r++ {
			g, ok, err := BuildRune(b, r)
			if err != nil {
				return nil, err
			}
			if ok {
				out[r] = g
			}
		}
	}
	return out, nil
}

// BuildRune composes one rune if its NFD base and every mark are available.
func BuildRune(b *skeleton.Builder, r rune) (Glyph, bool, error) {
	if skeleton.Lookup(r) != nil {
		return Glyph{}, false, nil
	}
	nfd := norm.NFD.String(string(r))
	runes := []rune(nfd)
	if len(runes) < 2 {
		return Glyph{}, false, nil
	}
	baseRune := runes[0]
	marks := runes[1:]
	if (baseRune == 'i' || baseRune == 'j') && hasAbove(marks) {
		baseRune = 0x0131
	}
	basePath, baseAdv, ok, err := b.Glyph(baseRune)
	if err != nil || !ok || len(basePath) == 0 {
		return Glyph{}, false, err
	}
	baseBounds := basePath.Bounds()
	out := basePath.Clone()
	centerX := (baseBounds.MinX + baseBounds.MaxX) / 2
	aboveTop := baseBounds.MaxY
	belowBottom := baseBounds.MinY
	aboveCount, belowCount := 0, 0

	for _, markRune := range marks {
		markPath, _, ok, err := b.Glyph(markRune)
		if err != nil {
			return Glyph{}, false, err
		}
		if !ok || len(markPath) == 0 {
			return Glyph{}, false, nil
		}
		mb := markPath.Bounds()
		markCenter := (mb.MinX + mb.MaxX) / 2
		dx := centerX - markCenter
		var dy float64
		if isBelow(markRune) {
			top := belowBottom - 30
			if belowCount > 0 {
				top -= 20
			}
			dy = top - mb.MaxY
			belowBottom = top - (mb.MaxY - mb.MinY)
			belowCount++
		} else {
			bottom := aboveTop + 30
			if aboveCount > 0 {
				bottom += 20
			}
			dy = bottom - mb.MinY
			aboveTop = bottom + (mb.MaxY - mb.MinY)
			aboveCount++
		}
		out = append(out, markPath.Transformed(geom.Translate(dx, dy))...)
	}
	return Glyph{Path: out, Advance: baseAdv}, true, nil
}

// ComposedRunes returns sorted keys of a composed map.
func ComposedRunes(m map[rune]Glyph) []rune {
	out := make([]rune, 0, len(m))
	for r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func hasAbove(marks []rune) bool {
	for _, m := range marks {
		if !isBelow(m) {
			return true
		}
	}
	return false
}

func isBelow(r rune) bool {
	switch r {
	case 0x0323, 0x0327, 0x0328, 0x0331:
		return true
	default:
		return false
	}
}
