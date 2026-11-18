// Package derive extracts and restyles glyphs from the embedded OFL base fonts.
package derive

import (
	"fmt"
	"math"
	"sync"

	"github.com/neko233-com/fontcreator233/assets"
	"github.com/neko233-com/fontcreator233/internal/geom"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Set is the parsed SC/KR base font pair.
type Set struct {
	SC *sfnt.Font
	KR *sfnt.Font
}

var (
	globalOnce sync.Once
	global     *Set
	globalErr  error
)

// Load parses the embedded base fonts once.
func Load() (*Set, error) {
	globalOnce.Do(func() {
		sc, err := sfnt.Parse(assets.BaseSC)
		if err != nil {
			globalErr = fmt.Errorf("parse SC base: %w", err)
			return
		}
		kr, err := sfnt.Parse(assets.BaseKR)
		if err != nil {
			globalErr = fmt.Errorf("parse KR base: %w", err)
			return
		}
		global = &Set{SC: sc, KR: kr}
	})
	return global, globalErr
}

var cache sync.Map // rune -> cachedGlyph

type cachedGlyph struct {
	path geom.Path
	adv  int
}

// Glyph returns the base outline and advance for r, with Y up and baseline 0.
func (s *Set) Glyph(r rune) (geom.Path, int, bool, error) {
	if v, ok := cache.Load(r); ok {
		c := v.(cachedGlyph)
		return c.path.Clone(), c.adv, true, nil
	}
	fonts := []*sfnt.Font{s.SC, s.KR}
	if isHangul(r) {
		fonts = []*sfnt.Font{s.KR, s.SC}
	}
	for _, f := range fonts {
		path, adv, ok, err := glyphFrom(f, r)
		if err != nil {
			return nil, 0, false, err
		}
		if ok {
			cache.Store(r, cachedGlyph{path: path.Clone(), adv: adv})
			return path, adv, true, nil
		}
	}
	return nil, 0, false, nil
}

// Has reports whether either base font contains r.
func (s *Set) Has(r rune) bool {
	_, _, ok, _ := s.Glyph(r)
	return ok
}

func glyphFrom(f *sfnt.Font, r rune) (geom.Path, int, bool, error) {
	var buf sfnt.Buffer
	gid, err := f.GlyphIndex(&buf, r)
	if err != nil {
		return nil, 0, false, err
	}
	if gid == 0 {
		return nil, 0, false, nil
	}
	ppem := fixed.Int26_6(f.UnitsPerEm())
	segs, err := f.LoadGlyph(&buf, gid, ppem, nil)
	if err != nil {
		return nil, 0, false, err
	}
	adv26, err := f.GlyphAdvance(&buf, gid, ppem, 0)
	if err != nil {
		return nil, 0, false, err
	}
	path := segmentsToPath(segs)
	normalizeOrientation(path)
	adv := int(math.Round(float64(adv26)))
	return path, adv, true, nil
}

// SegmentsToPath converts sfnt segments (Y down) to a font-unit path with Y up.
func SegmentsToPath(segs sfnt.Segments) geom.Path {
	p := segmentsToPath(segs)
	normalizeOrientation(p)
	return p
}

func segmentsToPath(segs sfnt.Segments) geom.Path {
	var path geom.Path
	var cur geom.Contour
	flush := func() {
		if len(cur) > 0 {
			path = append(path, cur)
			cur = nil
		}
	}
	toPoint := func(p fixed.Point26_6) geom.Point {
		return geom.Point{X: float64(p.X), Y: -float64(p.Y)}
	}
	for _, seg := range segs {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			flush()
			cur = append(cur, geom.Node{P: toPoint(seg.Args[0]), On: true})
		case sfnt.SegmentOpLineTo:
			cur = append(cur, geom.Node{P: toPoint(seg.Args[0]), On: true})
		case sfnt.SegmentOpQuadTo:
			cur = append(cur,
				geom.Node{P: toPoint(seg.Args[0]), On: false},
				geom.Node{P: toPoint(seg.Args[1]), On: true},
			)
		case sfnt.SegmentOpCubeTo:
			if len(cur) == 0 {
				continue
			}
			p0 := cur[len(cur)-1].P
			c1, c2, p1 := toPoint(seg.Args[0]), toPoint(seg.Args[1]), toPoint(seg.Args[2])
			flattenCubicPoints(p0, c1, c2, p1, &cur)
		}
	}
	flush()
	for i := range path {
		if len(path[i]) > 1 && path[i][0].P == path[i][len(path[i])-1].P {
			path[i] = path[i][:len(path[i])-1]
		}
	}
	return path
}

func flattenCubicPoints(p0, c1, c2, p1 geom.Point, out *geom.Contour) {
	const steps = 16
	for i := 1; i <= steps; i++ {
		t := float64(i) / steps
		u := 1 - t
		p := geom.Point{
			X: u*u*u*p0.X + 3*u*u*t*c1.X + 3*u*t*t*c2.X + t*t*t*p1.X,
			Y: u*u*u*p0.Y + 3*u*u*t*c1.Y + 3*u*t*t*c2.Y + t*t*t*p1.Y,
		}
		*out = append(*out, geom.Node{P: p, On: true})
	}
}

// normalizeOrientation makes the largest contour positive and therefore all
// holes negative. Base fonts are stored Y-down; after flipping, orientation is
// reversed, which would break synthetic bold offsetting.
func normalizeOrientation(p geom.Path) {
	if len(p) == 0 {
		return
	}
	maxArea := 0.0
	maxIdx := 0
	for i, c := range p {
		a := math.Abs(p.SignedArea(c))
		if a > maxArea {
			maxArea = a
			maxIdx = i
		}
	}
	if p.SignedArea(p[maxIdx]) < 0 {
		for i := range p {
			for l, r := 0, len(p[i])-1; l < r; l, r = l+1, r-1 {
				p[i][l], p[i][r] = p[i][r], p[i][l]
			}
		}
	}
}

func isHangul(r rune) bool {
	return (r >= 0x1100 && r <= 0x11FF) ||
		(r >= 0x3130 && r <= 0x318F) ||
		(r >= 0xA960 && r <= 0xA97F) ||
		(r >= 0xAC00 && r <= 0xD7AF) ||
		(r >= 0xD7B0 && r <= 0xD7FF) ||
		(r >= 0xFFA0 && r <= 0xFFDC)
}
