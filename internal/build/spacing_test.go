package build

import (
	"testing"

	"github.com/neko233-com/fontcreator233/internal/fontbuild"
	"github.com/neko233-com/fontcreator233/internal/geom"
)

// Diagonals from the skeleton used to sit ~18 units from the advance edge and 'W'
// overflowed its advance. The pass must fix only the violators.
func TestNormalizeSidebearings(t *testing.T) {
	glyphs := map[rune]fontbuild.Glyph{
		'A': {Rune: 'A', Advance: 667, Path: geom.RectPath(18, 0, 648, 700)},
		'W': {Rune: 'W', Advance: 944, Path: geom.RectPath(18, 0, 982, 700)},
		'H': {Rune: 'H', Advance: 722, Path: geom.RectPath(58, 0, 665, 700)},
		' ': {Rune: ' ', Advance: 250},
	}
	normalizeSidebearings(glyphs, spacingFloor)

	for _, rn := range []rune{'A', 'W', 'H'} {
		g := glyphs[rn]
		b := g.Path.Bounds()
		lsb := b.MinX
		rsb := float64(g.Advance) - b.MaxX
		if lsb < spacingFloor-0.5 {
			t.Fatalf("%q lsb = %.1f, want >= %.1f", rn, lsb, spacingFloor)
		}
		if rsb < spacingFloor-0.5 {
			t.Fatalf("%q rsb = %.1f, want >= %.1f", rn, rsb, spacingFloor)
		}
	}
	if got := glyphs[' '].Advance; got != 250 {
		t.Fatalf("space advance changed: %d", got)
	}
	// A and W must have grown; H already satisfied the floor and must stay identical.
	if glyphs['A'].Advance <= 667 {
		t.Fatalf("A advance not widened: %d", glyphs['A'].Advance)
	}
	if glyphs['W'].Advance <= 944 {
		t.Fatalf("W advance not widened: %d", glyphs['W'].Advance)
	}
	if glyphs['H'].Advance != 722 {
		t.Fatalf("H advance should be untouched, got %d", glyphs['H'].Advance)
	}
	if b := glyphs['H'].Path.Bounds(); b.MinX != 58 || b.MaxX != 665 {
		t.Fatalf("H outline moved: %#v", b)
	}
}

// Kerning must never pull two outlines closer than kerningFloor.
func TestKerningNeverCollides(t *testing.T) {
	glyphs := map[rune]fontbuild.Glyph{
		'A': {Rune: 'A', Advance: 690, Path: geom.RectPath(30, 0, 660, 700)},
		'V': {Rune: 'V', Advance: 690, Path: geom.RectPath(30, 0, 660, 700)},
		'T': {Rune: 'T', Advance: 611, Path: geom.RectPath(30, 0, 581, 700)},
		'o': {Rune: 'o', Advance: 556, Path: geom.RectPath(30, 0, 526, 700)},
	}
	byName := map[string]rune{}
	for rn := range glyphs {
		byName[glyphName(rn)] = rn
	}

	pairs := makeKerning(glyphs, 400)
	if len(pairs) == 0 {
		t.Fatal("no kern pairs generated")
	}
	for _, p := range pairs {
		lr, ok := byName[p.Left]
		if !ok {
			t.Fatalf("unknown left glyph %q", p.Left)
		}
		rr, ok := byName[p.Right]
		if !ok {
			t.Fatalf("unknown right glyph %q", p.Right)
		}
		gap := pairGap(glyphs[lr], glyphs[rr]) + float64(p.Value)
		if gap < kerningFloor-0.5 {
			t.Fatalf("%s/%s gap = %.1f after kern %d, want >= %.1f",
				p.Left, p.Right, gap, p.Value, kerningFloor)
		}
		if p.Value > 0 {
			t.Fatalf("%s/%s unexpected positive kern %d", p.Left, p.Right, p.Value)
		}
	}
}
