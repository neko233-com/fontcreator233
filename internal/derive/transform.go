package derive

import (
	"math"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

// Style is the geometric transform applied to base outlines.
type Style struct {
	WidthScale float64
	SlantTan   float64
	Embolden   float64
}

// ApplyStyle applies width, slant, and synthetic bold to a base glyph.
func ApplyStyle(p geom.Path, advance int, s Style) (geom.Path, int) {
	if s.WidthScale == 0 {
		s.WidthScale = 1
	}
	out := p.Transformed(geom.Affine{A: s.WidthScale, B: s.SlantTan, E: 1})
	if s.Embolden > 0 {
		out = geom.Offset(out, s.Embolden)
	}
	adv := int(math.Round(float64(advance) * s.WidthScale))
	return out, adv
}
