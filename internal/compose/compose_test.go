package compose

import (
	"math"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/skeleton"
	"github.com/neko233-com/fontcreator233/internal/style"
)

func testBuilder(t *testing.T) *skeleton.Builder {
	t.Helper()
	r := recipe.Default()
	r.Meta.Family = "Compose"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return skeleton.NewBuilder(style.Resolve(r))
}

func TestBuildPrecomposed(t *testing.T) {
	b := testBuilder(t)
	m, err := Build(b)
	if err != nil {
		t.Fatal(err)
	}
	want := []rune{'é', 'ñ', 'ü', 'Å', 'ç', 'ō', 'í', 'ế', 'Ώ', 'Ά', 'Ё', 'Й', 'Ї'}
	for _, r := range want {
		g, ok := m[r]
		if !ok {
			t.Fatalf("%U missing", r)
		}
		if len(g.Path) == 0 || g.Advance <= 0 {
			t.Fatalf("%U empty (adv=%v)", r, g.Advance)
		}
		for _, c := range g.Path {
			for _, n := range c {
				if math.IsNaN(n.P.X) || math.IsNaN(n.P.Y) {
					t.Fatalf("%U has NaN", r)
				}
			}
		}
	}
}

func TestComposedMetricsAndPlacement(t *testing.T) {
	b := testBuilder(t)
	basePath, baseAdv, _, _ := b.Glyph('e')
	comp, ok, err := BuildRune(b, 'é')
	if err != nil || !ok {
		t.Fatalf("é ok=%v err=%v", ok, err)
	}
	if comp.Advance != baseAdv {
		t.Fatalf("advance = %v, want %v", comp.Advance, baseAdv)
	}
	if comp.Path.Bounds().MaxY <= basePath.Bounds().MaxY {
		t.Fatalf("accent not above base: %v vs %v", comp.Path.Bounds().MaxY, basePath.Bounds().MaxY)
	}

	cBase, _, _, _ := b.Glyph('c')
	cedilla, ok, err := BuildRune(b, 'ç')
	if err != nil || !ok {
		t.Fatalf("ç ok=%v err=%v", ok, err)
	}
	if cedilla.Path.Bounds().MinY >= cBase.Bounds().MinY {
		t.Fatalf("cedilla not below base: %v vs %v", cedilla.Path.Bounds().MinY, cBase.Bounds().MinY)
	}
}

func TestDotlessIComposition(t *testing.T) {
	b := testBuilder(t)
	g, ok, err := BuildRune(b, 'í')
	if err != nil || !ok {
		t.Fatalf("í ok=%v err=%v", ok, err)
	}
	if len(g.Path) != 2 {
		t.Fatalf("í contours = %d, want 2 (dotless stem + acute)", len(g.Path))
	}
}

func TestCombiningMarksAvailable(t *testing.T) {
	b := testBuilder(t)
	for _, r := range []rune{0x0300, 0x0301, 0x0302, 0x0303, 0x0304, 0x0306, 0x0307, 0x0308, 0x030A, 0x030B, 0x030C, 0x0323, 0x0327, 0x0328, 0x0331} {
		p, adv, ok, err := b.Glyph(r)
		if err != nil || !ok || len(p) == 0 {
			t.Fatalf("%U ok=%v err=%v", r, ok, err)
		}
		if adv != 0 {
			t.Fatalf("%U advance = %v, want 0", r, adv)
		}
	}
}
