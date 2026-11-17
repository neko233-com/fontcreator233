package skeleton

import (
	"math"
	"reflect"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/style"
)

func builder(t *testing.T, mut func(*recipe.Recipe)) *Builder {
	t.Helper()
	r := recipe.Default()
	r.Meta.Family = "Test"
	if mut != nil {
		mut(r)
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return NewBuilder(style.Resolve(r))
}

func totalAbsArea(p geom.Path) float64 {
	var sum float64
	for _, c := range p {
		sum += math.Abs(p.SignedArea(c))
	}
	return sum
}

func TestBasicGlyphs(t *testing.T) {
	b := builder(t, nil)
	for _, r := range []rune{'H', 'o', 'A', '0'} {
		p, adv, ok, err := b.Glyph(r)
		if err != nil || !ok {
			t.Fatalf("%q: ok=%v err=%v", r, ok, err)
		}
		if len(p) == 0 {
			t.Fatalf("%q: empty path", r)
		}
		if adv <= 0 {
			t.Fatalf("%q: advance %v", r, adv)
		}
		bounds := p.Bounds()
		if bounds.MinX < -50 || bounds.MaxX > 1050 || bounds.MinY < -250 || bounds.MaxY > 1050 {
			t.Fatalf("%q: bounds %#v", r, bounds)
		}
		for ci, c := range p {
			if len(c) < 3 {
				t.Fatalf("%q contour %d has %d nodes", r, ci, len(c))
			}
			for _, n := range c {
				if math.IsNaN(n.P.X) || math.IsNaN(n.P.Y) {
					t.Fatalf("%q: NaN node", r)
				}
			}
		}
	}
}

func TestRingHasHole(t *testing.T) {
	b := builder(t, nil)
	p, _, _, err := b.Glyph('o')
	if err != nil {
		t.Fatal(err)
	}
	var pos, neg int
	for _, c := range p {
		if p.SignedArea(c) > 0 {
			pos++
		} else if p.SignedArea(c) < 0 {
			neg++
		}
	}
	if pos == 0 || neg == 0 {
		t.Fatalf("o contours: pos=%d neg=%d, want both", pos, neg)
	}
}

func TestWeightChangesArea(t *testing.T) {
	light := builder(t, func(r *recipe.Recipe) { r.Style.Weight = 100 })
	heavy := builder(t, func(r *recipe.Recipe) { r.Style.Weight = 900 })
	pl, _, _, _ := light.Glyph('H')
	ph, _, _, _ := heavy.Glyph('H')
	if totalAbsArea(ph) <= totalAbsArea(pl)*1.5 {
		t.Fatalf("heavy area %v, light %v", totalAbsArea(ph), totalAbsArea(pl))
	}
}

func TestDeterministicJitter(t *testing.T) {
	mk := func(seed int64) geom.Path {
		b := builder(t, func(r *recipe.Recipe) {
			r.Style.Roughness = 0.4
			r.Style.Seed = seed
		})
		p, _, _, err := b.Glyph('g')
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	if !reflect.DeepEqual(mk(1), mk(1)) {
		t.Fatal("same seed differs")
	}
	if reflect.DeepEqual(mk(1), mk(2)) {
		t.Fatal("different seed produced identical path")
	}
}

func TestSerifAddsContours(t *testing.T) {
	plain := builder(t, func(r *recipe.Recipe) { r.Style.Serif = "none" })
	serif := builder(t, func(r *recipe.Recipe) {
		r.Style.Serif = "slab"
		r.Style.SerifSize = 0.6
	})
	pp, _, _, _ := plain.Glyph('H')
	ps, _, _, _ := serif.Glyph('H')
	if len(ps) <= len(pp) {
		t.Fatalf("serif contours %d, plain %d", len(ps), len(pp))
	}
}

func TestRoundCapAddsPoints(t *testing.T) {
	flat := builder(t, func(r *recipe.Recipe) { r.Style.Terminal = "flat" })
	round := builder(t, func(r *recipe.Recipe) { r.Style.Terminal = "round" })
	// 'l' is a stem whose endpoints are open when no serifs are requested.
	pf, _, _, _ := flat.Glyph('l')
	pr, _, _, _ := round.Glyph('l')
	if len(pr[0]) <= len(pf[0]) {
		t.Fatalf("round nodes %d, flat %d", len(pr[0]), len(pf[0]))
	}
}

func TestFlareChangesPath(t *testing.T) {
	flat := builder(t, func(r *recipe.Recipe) { r.Style.Terminal = "flat" })
	flare := builder(t, func(r *recipe.Recipe) { r.Style.Terminal = "flare" })
	pf, _, _, _ := flat.Glyph('I')
	pr, _, _, _ := flare.Glyph('I')
	if reflect.DeepEqual(pf, pr) {
		t.Fatal("flare made no difference")
	}
}

func TestSkeletonVariants(t *testing.T) {
	geometric := builder(t, func(r *recipe.Recipe) { r.Style.Skeleton = "geometric" })
	serif := builder(t, func(r *recipe.Recipe) { r.Style.Skeleton = "serif" })
	for _, r := range []rune{'a', 'g'} {
		pg, _, _, err := geometric.Glyph(r)
		if err != nil {
			t.Fatal(err)
		}
		ps, _, _, err := serif.Glyph(r)
		if err != nil {
			t.Fatal(err)
		}
		if len(pg) == 0 || len(ps) == 0 {
			t.Fatalf("%q empty variant", r)
		}
		if reflect.DeepEqual(pg, ps) {
			t.Fatalf("%q variant did not change the outline", r)
		}
	}
}
