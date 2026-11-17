// Package style resolves recipe parameters into concrete numeric glyph-engine
// values and resolves the effective subfamily name.
package style

import (
	"math"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/recipe"
)

// Params are concrete values consumed by the skeleton and derive engines.
type Params struct {
	Stem           float64
	Thin           float64
	WidthScale     float64
	SlantTan       float64
	XScale         float64
	CapScale       float64
	SerifSize      float64
	Contrast       float64
	Overshoot      float64
	Roughness      float64
	JitterRotation float64
	Weight         int
	Seed           int64
	Skeleton       string
	Terminal       string
	Serif          string
}

// Resolve maps a recipe to concrete style parameters.
func Resolve(r *recipe.Recipe) Params {
	weight := r.Style.Weight
	stem := 34 + float64(weight-100)*136/800
	thin := stem * (1 - 0.85*r.Style.Contrast)
	if thin < 6 {
		thin = 6
	}
	return Params{
		Stem:           stem,
		Thin:           thin,
		WidthScale:     r.Style.Width / 100,
		SlantTan:       math.Tan(r.Style.Slant * math.Pi / 180),
		XScale:         r.Metrics.XHeight / 0.5,
		CapScale:       r.Metrics.CapHeight / 0.7,
		SerifSize:      r.Style.SerifSize * stem,
		Contrast:       r.Style.Contrast,
		Overshoot:      r.Style.Overshoot,
		Roughness:      r.Style.Roughness,
		JitterRotation: r.Style.JitterRotation,
		Weight:         weight,
		Seed:           r.Style.Seed,
		Skeleton:       r.Style.Skeleton,
		Terminal:       r.Style.Terminal,
		Serif:          r.Style.Serif,
	}
}

// ResolveStyle returns the effective subfamily name. When slant is non-zero and
// the requested style is not already italic/oblique, it appends Italic (or
// replaces Regular).
func ResolveStyle(r *recipe.Recipe) string {
	s := strings.TrimSpace(r.Meta.Style)
	if s == "" {
		s = "Regular"
	}
	if r.Style.Slant == 0 {
		return s
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "italic") || strings.Contains(lower, "oblique") {
		return s
	}
	if strings.EqualFold(s, "Regular") {
		return "Italic"
	}
	return s + " Italic"
}
