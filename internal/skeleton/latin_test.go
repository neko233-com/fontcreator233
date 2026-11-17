package skeleton

import (
	"math"
	"testing"
)

func TestASCIILatinCoverage(t *testing.T) {
	b := builder(t, nil)
	for r := rune(0x20); r <= 0x7E; r++ {
		p, adv, ok, err := b.Glyph(r)
		if err != nil {
			t.Fatalf("%q: %v", r, err)
		}
		if !ok {
			t.Fatalf("%q not available", r)
		}
		if r == ' ' {
			if len(p) != 0 {
				t.Fatalf("space has contours")
			}
			if adv <= 0 {
				t.Fatalf("space advance = %v", adv)
			}
			continue
		}
		if len(p) == 0 {
			t.Fatalf("%q empty", r)
		}
		if adv < 200 || adv > 1100 {
			t.Fatalf("%q advance = %v", r, adv)
		}
		for _, c := range p {
			for _, n := range c {
				if math.IsNaN(n.P.X) || math.IsNaN(n.P.Y) || math.IsInf(n.P.X, 0) || math.IsInf(n.P.Y, 0) {
					t.Fatalf("%q has non-finite node", r)
				}
			}
		}
	}
	if got := len(Available()); got < 95 {
		t.Fatalf("available = %d, want >= 95", got)
	}
}

func TestComplexLettersHaveMultipleContours(t *testing.T) {
	b := builder(t, nil)
	for _, r := range []rune{'A', 'a', 'g', 'W', '@', '%', '&', '?'} {
		p, _, _, err := b.Glyph(r)
		if err != nil {
			t.Fatal(err)
		}
		if len(p) < 2 {
			t.Fatalf("%q contours = %d, want >= 2", r, len(p))
		}
	}
}
