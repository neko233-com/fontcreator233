package style

import (
	"math"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/recipe"
)

func TestResolve(t *testing.T) {
	r := recipe.Default()
	r.Style.Weight = 400
	r.Style.Width = 100
	r.Style.Slant = 8
	r.Metrics.XHeight = 0.52
	r.Metrics.CapHeight = 0.70
	p := Resolve(r)
	if math.Abs(p.Stem-85) > 1e-9 {
		t.Fatalf("stem = %v, want 85", p.Stem)
	}
	if p.WidthScale != 1 {
		t.Fatalf("width = %v", p.WidthScale)
	}
	if math.Abs(p.SlantTan-math.Tan(8*math.Pi/180)) > 1e-12 {
		t.Fatalf("slant = %v", p.SlantTan)
	}
	if math.Abs(p.XScale-1.04) > 1e-9 || math.Abs(p.CapScale-1.0) > 1e-9 {
		t.Fatalf("scales = %v, %v", p.XScale, p.CapScale)
	}
	r.Style.Weight = 900
	if p := Resolve(r); math.Abs(p.Stem-170) > 1e-9 {
		t.Fatalf("stem 900 = %v", p.Stem)
	}
}

func TestResolveStyle(t *testing.T) {
	tests := []struct {
		style string
		slant float64
		want  string
	}{
		{"Regular", 0, "Regular"},
		{"", 0, "Regular"},
		{"Regular", 8, "Italic"},
		{"", 8, "Italic"},
		{"Display", 8, "Display Italic"},
		{"Oblique", 8, "Oblique"},
		{"Noir Italic", 8, "Noir Italic"},
		{"Noir", 0, "Noir"},
	}
	for _, tt := range tests {
		r := recipe.Default()
		r.Meta.Style = tt.style
		r.Style.Slant = tt.slant
		if got := ResolveStyle(r); got != tt.want {
			t.Fatalf("ResolveStyle(%q,%v) = %q, want %q", tt.style, tt.slant, got, tt.want)
		}
	}
}
