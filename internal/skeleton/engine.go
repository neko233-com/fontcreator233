// Package skeleton generates parametric glyph outlines from built-in centerline
// skeletons.
package skeleton

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/style"
)

// Zone selects which vertical metric scales a glyph.
type Zone uint8

const (
	ZoneLower Zone = iota
	ZoneUpper
	ZoneDigit
	ZoneMark
	ZoneSymbol
)

// Stroke is a centerline stroke.
type Stroke struct {
	Cmds       []geom.Cmd
	Width      float64
	Closed     bool
	CapStart   string
	CapEnd     string
	SerifStart bool
	SerifEnd   bool
	Flare      bool
	// TaperStart/TaperEnd shrink the stroke width near the corresponding
	// endpoint (0 disables, 0.2 means the tip is 80% of the stem width).
	TaperStart float64
	TaperEnd   float64
	// JoinStart/JoinEnd mark endpoints shared with another stroke. The
	// expander extends those ends slightly so the two outlines overlap instead
	// of merely abutting (which leaves an anti-aliasing seam).
	JoinStart bool
	JoinEnd   bool
}

// Dot is a filled circular contour.
type Dot struct {
	X, Y, R float64
}

// Def is one skeleton glyph.
type Def struct {
	Rune     rune
	Advance  float64
	Zone     Zone
	Strokes  []Stroke
	Dots     []Dot
	Variants map[string]*Def
}

// Builder turns skeleton definitions into styled outlines.
type Builder struct {
	P style.Params
}

// NewBuilder returns a Builder for concrete style parameters.
func NewBuilder(p style.Params) *Builder { return &Builder{P: p} }

// Glyph returns the outline and advance for r.
func (b *Builder) Glyph(r rune) (geom.Path, float64, bool, error) {
	def := Lookup(r)
	if def == nil {
		return nil, 0, false, nil
	}
	if alt := def.Variants[b.P.Skeleton]; alt != nil {
		def = alt
	}
	p, err := b.build(def)
	if err != nil {
		return nil, 0, false, fmt.Errorf("glyph %U: %w", r, err)
	}
	scale := b.zoneScale(def.Zone)
	adv := def.Advance*b.P.WidthScale + b.P.Tracking*10
	if len(p) > 0 {
		m := geom.Affine{
			A: b.P.WidthScale,
			B: b.P.SlantTan * scale,
			E: scale,
		}
		p = p.Transformed(m)
		p = b.jitter(p, r)
	}
	return p, adv, true, nil
}

// BuildDef builds an unscaled definition. It is used by composition and
// library export.
func (b *Builder) BuildDef(def *Def) (geom.Path, error) { return b.build(def) }

// ZoneScale returns the vertical scale for a zone.
func (b *Builder) ZoneScale(z Zone) float64 { return b.zoneScale(z) }

func (b *Builder) zoneScale(z Zone) float64 {
	switch z {
	case ZoneLower:
		return b.P.XScale
	case ZoneUpper, ZoneDigit:
		return b.P.CapScale
	default:
		return 1
	}
}

func (b *Builder) build(def *Def) (geom.Path, error) {
	strokes := append([]Stroke(nil), def.Strokes...)
	flattenJoints(strokes)
	var out geom.Path
	for _, s := range strokes {
		p, err := b.expand(s)
		if err != nil {
			return nil, err
		}
		out = append(out, p...)
	}
	for _, d := range def.Dots {
		out = append(out, geom.Circle(d.X, d.Y, d.R)...)
	}
	return out, nil
}

// flattenJoints turns round caps into flat cuts where two strokes meet.
// A stroke end can meet another stroke either at that stroke's end (a
// corner) or along its interior (a T-junction, as in y). Both cases need a
// flat, overlapping cut: two round ends that share one point only touch
// tangentially and leave a pinch or a spur.
func flattenJoints(strokes []Stroke) {
	type point struct{ x, y int64 }
	key := func(p geom.Point) point {
		return point{int64(math.Round(p.X * 100)), int64(math.Round(p.Y * 100))}
	}
	ends := map[point]int{}
	endpoints := make([]struct {
		a, b point
		ok   bool
	}, len(strokes))
	interior := map[point]map[int]bool{}
	for i, s := range strokes {
		if s.Closed || len(s.Cmds) == 0 {
			continue
		}
		pts := geom.Flatten(s.Cmds, 0.3)
		pts = cleanPoints(pts, false)
		if len(pts) < 2 {
			continue
		}
		a, b := key(pts[0]), key(pts[len(pts)-1])
		endpoints[i] = struct {
			a, b point
			ok   bool
		}{a, b, true}
		ends[a]++
		ends[b]++
		for k := 1; k < len(pts)-1; k++ {
			k := key(pts[k])
			if interior[k] == nil {
				interior[k] = map[int]bool{}
			}
			interior[k][i] = true
		}
	}
	hasOtherInterior := func(p point, self int) bool {
		for i := range interior[p] {
			if i != self {
				return true
			}
		}
		return false
	}
	for i := range strokes {
		e := endpoints[i]
		if !e.ok {
			continue
		}
		if ends[e.a] > 1 || hasOtherInterior(e.a, i) {
			// An explicit cap wins: authors use "round" for a deliberate ball
			// foot, and flattening it would turn the foot into a hard edge.
			if strokes[i].CapStart == "" {
				strokes[i].CapStart = "flat"
			}
			if strokes[i].CapStart == "flat" {
				strokes[i].JoinStart = true
			}
		}
		if ends[e.b] > 1 || hasOtherInterior(e.b, i) {
			if strokes[i].CapEnd == "" {
				strokes[i].CapEnd = "flat"
			}
			if strokes[i].CapEnd == "flat" {
				strokes[i].JoinEnd = true
			}
		}
	}
}

func (b *Builder) jitter(p geom.Path, r rune) geom.Path {
	if b.P.Roughness <= 0 && b.P.JitterRotation <= 0 {
		return p
	}
	out := p.Clone()
	for ci := range out {
		for ni := range out[ci] {
			idx := ci*10000 + ni
			dx := hashUnit(b.P.Seed, r, idx*2+1) * b.P.Roughness * 8
			dy := hashUnit(b.P.Seed, r, idx*2+2) * b.P.Roughness * 8
			out[ci][ni].P.X += dx
			out[ci][ni].P.Y += dy
		}
	}
	if b.P.JitterRotation > 0 && len(out) > 0 {
		bounds := out.Bounds()
		cx := (bounds.MinX + bounds.MaxX) / 2
		cy := (bounds.MinY + bounds.MaxY) / 2
		a := hashUnit(b.P.Seed, r, 999999) * b.P.JitterRotation * math.Pi / 180
		cos, sin := math.Cos(a), math.Sin(a)
		for ci := range out {
			for ni := range out[ci] {
				q := out[ci][ni].P
				x, y := q.X-cx, q.Y-cy
				out[ci][ni].P = geom.Point{X: cx + x*cos - y*sin, Y: cy + x*sin + y*cos}
			}
		}
	}
	return out
}

func hashUnit(seed int64, r rune, idx int) float64 {
	h := fnv.New64a()
	var buf [16]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(uint64(seed) >> (8 * i))
	}
	for i := 0; i < 8; i++ {
		buf[8+i] = byte(uint64(uint32(r)) >> (8 * i))
	}
	_, _ = h.Write(buf[:])
	var ib [8]byte
	for i := 0; i < 8; i++ {
		ib[i] = byte(uint64(idx) >> (8 * i))
	}
	_, _ = h.Write(ib[:])
	return float64(h.Sum64()%100000)/100000 - 0.5
}

// Available returns all built-in code points in sorted order.
func Available() []rune {
	out := make([]rune, 0, len(glyphs))
	for r := range glyphs {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Lookup returns the definition for r, or nil.
func Lookup(r rune) *Def { return glyphs[r] }
