package geom

import (
	"math"
	"testing"
)

func TestRectBoundsAndArea(t *testing.T) {
	p := RectPath(0, 0, 100, 100)
	if got := p.Bounds(); got != (Rect{0, 0, 100, 100}) {
		t.Fatalf("bounds = %#v", got)
	}
	if area := p.SignedArea(p[0]); math.Abs(area-10000) > 1e-9 {
		t.Fatalf("area = %v, want 10000", area)
	}
}

func TestAffineTransform(t *testing.T) {
	p := Path{{{P: Point{1, 2}, On: true}}}
	m := Affine{A: 2, B: 0.5, C: 10, D: 0, E: 2, F: -5}
	got := p.Transformed(m)[0][0].P
	if got != (Point{13, -1}) {
		t.Fatalf("got %v, want (13,-1)", got)
	}
}

func TestReverseFlipsArea(t *testing.T) {
	p := RectPath(0, 0, 10, 10)
	area := p.SignedArea(p[0])
	rev := p.Reverse(0)
	if math.Abs(area+rev.SignedArea(rev[0])) > 1e-9 {
		t.Fatalf("area %v, reversed %v", area, rev.SignedArea(rev[0]))
	}
}

func TestCircleArea(t *testing.T) {
	p := Circle(0, 0, 100)
	area := math.Abs(p.SignedArea(p[0]))
	want := math.Pi * 100 * 100
	if math.Abs(area-want)/want > 0.01 {
		t.Fatalf("area = %v, want ~%v", area, want)
	}
}

func TestOffsetExpandsSquare(t *testing.T) {
	p := RectPath(0, 0, 100, 100)
	out := Offset(p, 10)
	b := out.Bounds()
	if math.Abs(b.MinX+10) > 0.5 || math.Abs(b.MinY+10) > 0.5 || math.Abs(b.MaxX-110) > 0.5 || math.Abs(b.MaxY-110) > 0.5 {
		t.Fatalf("bounds = %#v, want [-10,110]", b)
	}
}

func TestOffsetRingKeepsHoleDirection(t *testing.T) {
	outer := RectPath(0, 0, 100, 100)
	hole := RectPath(20, 20, 80, 80).Reverse(0)
	p := append(outer, hole...)
	out := Offset(p, 5)
	if len(out) != 2 {
		t.Fatalf("contours = %d", len(out))
	}
	outerArea := out.SignedArea(out[0])
	holeArea := out.SignedArea(out[1])
	if outerArea <= 0 || holeArea >= 0 {
		t.Fatalf("signs: outer=%v hole=%v", outerArea, holeArea)
	}
	b := out.Bounds()
	if b.MinX != -5 || b.MaxX != 105 {
		t.Fatalf("outer bounds = %#v", b)
	}
	// The hole must shrink when the black region is emboldened.
	holeBounds := Path{out[1]}.Bounds()
	if math.Abs(holeBounds.MinX-25) > 0.5 || math.Abs(holeBounds.MaxX-75) > 0.5 {
		t.Fatalf("hole bounds = %#v, want [25,75]", holeBounds)
	}
}

func TestOffsetShrinks(t *testing.T) {
	out := Offset(RectPath(0, 0, 100, 100), -2)
	b := out.Bounds()
	if math.Abs(b.MinX-2) > 0.5 || math.Abs(b.MaxX-98) > 0.5 {
		t.Fatalf("bounds = %#v", b)
	}
}

func TestEmptyBounds(t *testing.T) {
	var p Path
	if b := p.Bounds(); b != (Rect{}) {
		t.Fatalf("bounds = %#v", b)
	}
}

func TestPointHelpers(t *testing.T) {
	a, b := Point{3, 4}, Point{1, 2}
	if a.Add(b) != (Point{4, 6}) || a.Sub(b) != (Point{2, 2}) || a.Mul(2) != (Point{6, 8}) {
		t.Fatalf("basic point ops failed")
	}
	if a.Len() != 5 || a.Normalize() != (Point{0.6, 0.8}) {
		t.Fatalf("len/normalize failed")
	}
	if a.Dot(b) != 11 || a.Cross(b) != 2 {
		t.Fatalf("dot/cross failed")
	}
	if a.Lerp(b, 0.5) != (Point{2, 3}) {
		t.Fatalf("lerp failed")
	}
	if got := Translate(10, -5).Apply(a); got != (Point{13, -1}) {
		t.Fatalf("translate = %v", got)
	}
	if got := Scale(2, 3).Apply(a); got != (Point{6, 12}) {
		t.Fatalf("scale = %v", got)
	}
	if got := ShearX(0.5).Apply(a); got != (Point{5, 4}) {
		t.Fatalf("shear = %v", got)
	}
}

func TestFlattenContours(t *testing.T) {
	cmds, err := ParseCommands("M 0 0 L 10 0 L 10 10 Z M 20 20 L 30 20 L 30 30 Z")
	if err != nil {
		t.Fatal(err)
	}
	cs := FlattenContours(cmds, 0.1)
	if len(cs) != 2 {
		t.Fatalf("contours = %d", len(cs))
	}
	if len(cs[0]) != 4 || len(cs[1]) != 4 {
		t.Fatalf("points = %d,%d", len(cs[0]), len(cs[1]))
	}
	if cs[0][3] != (Point{0, 0}) || cs[1][3] != (Point{20, 20}) {
		t.Fatalf("closed endpoints = %v, %v", cs[0][3], cs[1][3])
	}
}
