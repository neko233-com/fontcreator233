package derive

import (
	"math"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

func TestLoadAndGlyphs(t *testing.T) {
	set, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	tests := []struct {
		r    rune
		want bool
	}{
		{'A', true},
		{'中', true},
		{'あ', true},
		{'한', true},
		{'€', true},
		{'→', true},
		{0x10FFFF, false},
	}
	for _, tt := range tests {
		p, adv, ok, err := set.Glyph(tt.r)
		if err != nil {
			t.Fatalf("%U: %v", tt.r, err)
		}
		if ok != tt.want {
			t.Fatalf("%U ok=%v, want %v", tt.r, ok, tt.want)
		}
		if !ok {
			continue
		}
		if len(p) == 0 || adv <= 0 {
			t.Fatalf("%U empty (adv=%v)", tt.r, adv)
		}
		b := p.Bounds()
		if math.IsNaN(b.MinX) || math.IsNaN(b.MaxY) {
			t.Fatalf("%U NaN bounds", tt.r)
		}
	}
}

func TestYAxisIsUp(t *testing.T) {
	set, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	p, _, ok, err := set.Glyph('A')
	if err != nil || !ok {
		t.Fatalf("A ok=%v err=%v", ok, err)
	}
	b := p.Bounds()
	if b.MaxY <= 0 || b.MinY < -50 {
		t.Fatalf("A bounds = %#v, want y-up with positive ascender", b)
	}
	cjk, adv, ok, err := set.Glyph('中')
	if err != nil || !ok {
		t.Fatalf("中 ok=%v err=%v", ok, err)
	}
	cb := cjk.Bounds()
	if cb.Width() < 700 || cb.Height() < 700 {
		t.Fatalf("中 bounds = %#v, want a full-size CJK glyph", cb)
	}
	if adv < 900 || adv > 1100 {
		t.Fatalf("中 advance = %d, want ~1000", adv)
	}
}

func TestDeterministic(t *testing.T) {
	set, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	a, aa, _, _ := set.Glyph('中')
	b, ba, _, _ := set.Glyph('中')
	if aa != ba || len(a) != len(b) {
		t.Fatalf("advance/contours differ: %d/%d %d/%d", aa, ba, len(a), len(b))
	}
	for i := range a {
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				t.Fatalf("node differs at %d,%d", i, j)
			}
		}
	}
}

func TestApplyStyle(t *testing.T) {
	p := geom.RectPath(0, 0, 500, 700)
	adv := 600
	wide, wadv := ApplyStyle(p, adv, Style{WidthScale: 1.1})
	if wadv != 660 {
		t.Fatalf("wide advance = %d", wadv)
	}
	if wide.Bounds().MaxX <= p.Bounds().MaxX {
		t.Fatalf("width did not expand: %#v", wide.Bounds())
	}
	slanted, _ := ApplyStyle(p, adv, Style{WidthScale: 1, SlantTan: 0.2})
	if slanted.Bounds().MaxX <= wide.Bounds().MaxX {
		// Top-right corner should shift right by 0.2*700.
		t.Fatalf("slant did not shift: %#v", slanted.Bounds())
	}
	bold, _ := ApplyStyle(p, adv, Style{WidthScale: 1, Embolden: 20})
	area := math.Abs(bold.SignedArea(bold[0]))
	if area <= math.Abs(p.SignedArea(p[0])) {
		t.Fatalf("embolden area = %v", area)
	}
}
