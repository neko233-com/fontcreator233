package skeleton

import (
	"math"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/recipe"
)

func TestRoundCapsStayOnMetrics(t *testing.T) {
	b := builder(t, nil)
	p, _, _, err := b.Glyph('I')
	if err != nil {
		t.Fatal(err)
	}
	bounds := p.Bounds()
	// 700 skeleton units scaled by capHeight/0.7 = 720.
	if math.Abs(bounds.MaxY-720) > 1.5 {
		t.Fatalf("I top = %.2f, want ~720 (round cap overshoot)", bounds.MaxY)
	}
	if math.Abs(bounds.MinY) > 1.5 {
		t.Fatalf("I bottom = %.2f, want ~0", bounds.MinY)
	}
}

func TestAngledCapsStayOnMetrics(t *testing.T) {
	b := builder(t, func(r *recipe.Recipe) { r.Style.Terminal = "angled" })
	p, _, _, err := b.Glyph('I')
	if err != nil {
		t.Fatal(err)
	}
	bounds := p.Bounds()
	if bounds.MaxY > 722 {
		t.Fatalf("angled I top = %.2f, want <= 722", bounds.MaxY)
	}
	if bounds.MinY < -2 {
		t.Fatalf("angled I bottom = %.2f, want >= -2", bounds.MinY)
	}
	// The cut must be slanted: the two top corners differ in height.
	var hi []float64
	for _, c := range p {
		for _, n := range c {
			if n.P.Y > bounds.MaxY-40 {
				hi = append(hi, n.P.Y)
			}
		}
	}
	if len(hi) < 2 {
		t.Fatalf("angled I has no slanted top edge: %v", hi)
	}
	lo, up := hi[0], hi[0]
	for _, y := range hi {
		if y < lo {
			lo = y
		}
		if y > up {
			up = y
		}
	}
	if up-lo < 10 {
		t.Fatalf("angled I top edge is not slanted: %.2f..%.2f", lo, up)
	}
}

func TestHookTerminalIsFlatAndInBounds(t *testing.T) {
	b := builder(t, nil)
	p, _, _, err := b.Glyph('j')
	if err != nil {
		t.Fatal(err)
	}
	bounds := p.Bounds()
	if bounds.MinX < 40 {
		t.Fatalf("j left sidebearing = %.2f, want >= 40 (flat cut, no ball cap)", bounds.MinX)
	}
	if bounds.MinY < -200 || bounds.MinY > -160 {
		t.Fatalf("j bottom = %.2f, want inside the descender and not cut too short", bounds.MinY)
	}
}

func TestShouldersHaveNoCapEars(t *testing.T) {
	b := builder(t, nil)
	for _, r := range "nmr" {
		p, _, _, err := b.Glyph(r)
		if err != nil {
			t.Fatal(err)
		}
		bounds := p.Bounds()
		// The stem and arch now share one continuous centreline, so the top is
		// the arch apex plus half a stroke, with only the normal overshoot.
		if bounds.MaxY < 518 || bounds.MaxY > 535 {
			t.Fatalf("%q top = %.2f, want a rounded shoulder between 518 and 535", r, bounds.MaxY)
		}
		if bounds.MinY < -5 {
			t.Fatalf("%q bottom = %.2f, want on or above the baseline", r, bounds.MinY)
		}
	}
}

func TestLowercaseBowlsStayOnMetrics(t *testing.T) {
	b := builder(t, nil)
	bounds := func(r rune) (x, y float64) {
		p, _, _, err := b.Glyph(r)
		if err != nil {
			t.Fatal(err)
		}
		return p.Bounds().MinY, p.Bounds().MaxY
	}
	// g is a single-storey bowl with a long open tail: the bowl must top out at
	// x-height, not overshoot like a second storey.
	lo, hi := bounds('g')
	if hi < 518 || hi > 522 {
		t.Fatalf("g top = %.2f, want on x-height", hi)
	}
	if lo > -200 {
		t.Fatalf("g bottom = %.2f, want a real descender tail", lo)
	}
	// e had a 571-unit top before the upper bowl was lowered: it must stay on
	// x-height instead of towering over a/o.
	_, hi = bounds('e')
	if hi > 521 {
		t.Fatalf("e top = %.2f, want <= x-height", hi)
	}
	// The a stem used to end in a round cap below the baseline. It now shares
	// the baseline with the bowl.
	lo, _ = bounds('a')
	if lo < -1 {
		t.Fatalf("a bottom = %.2f, want on the baseline", lo)
	}
}

func TestJoinedStrokeEndsOverlap(t *testing.T) {
	b := builder(t, nil)
	p, _, _, err := b.Glyph('J')
	if err != nil {
		t.Fatal(err)
	}
	var stemBottom, hookTop bool
	for _, c := range p {
		for _, n := range c {
			if n.P.X > 395 && n.P.X < 405 {
				if n.P.Y < 119.5 {
					stemBottom = true
				}
				if n.P.Y > 120.5 {
					hookTop = true
				}
			}
		}
	}
	if !stemBottom || !hookTop {
		t.Fatalf("J joint does not overlap: stemBottom=%v hookTop=%v", stemBottom, hookTop)
	}
}
