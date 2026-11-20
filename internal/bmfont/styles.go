// Package bmfont writes AngelCode BMFont (.fnt v3) files and PNG atlases from
// the generated vector glyphs. It supports the full SnowB-style layered
// render pipeline (fill / stroke / inner shadow / drop shadow / background)
// and SDF/MSDF render modes.
package bmfont

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/recipe"
)

// Style mirrors recipe.BMFontStyle with defaults applied.
type Style struct {
	RenderMode       string
	FillColor        color.RGBA
	HasFill          bool
	StrokeColor      color.RGBA
	HasStroke        bool
	StrokeWidth      int
	StrokePosition   string
	StrokeCap        string
	StrokeJoin       string
	DropShadowColor  color.RGBA
	HasDropShadow    bool
	DropShadowX      int
	DropShadowY      int
	DropShadowBlur   int
	InnerShadowColor color.RGBA
	HasInnerShadow   bool
	InnerShadowX     int
	InnerShadowY     int
	InnerShadowBlur  int
	BackgroundColor  color.RGBA
	HasBackground    bool
}

// DefaultStyle returns a Style that renders white text on transparent, which
// matches the historical output of bmfont.go and is what vanilla BMFont
// consumers (AngelCode BMFont, Unity TMP) expect.
func DefaultStyle() Style {
	return Style{
		RenderMode:     "default",
		FillColor:      color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF},
		HasFill:        true,
		StrokePosition: "outer",
		StrokeCap:      "butt",
		StrokeJoin:     "miter",
	}
}

// ApplyStyle populates a Style from a recipe.BMFontStyle, returning the
// effective style. It tolerates nil input — in which case the default style
// is returned unchanged.
func ApplyStyle(src *recipe.BMFontStyle) Style {
	s := DefaultStyle()
	if src == nil {
		return s
	}
	if src.RenderMode != "" {
		s.RenderMode = src.RenderMode
	}
	if c, ok := parseColor(src.FillColor); ok {
		s.FillColor = c
		s.HasFill = true
	} else if src.FillColor != "" {
		// explicit empty string: disable fill (used to render stroke-only)
		s.HasFill = false
	}
	if c, ok := parseColor(src.StrokeColor); ok && src.StrokeWidth > 0 {
		s.StrokeColor = c
		s.HasStroke = true
		s.StrokeWidth = src.StrokeWidth
	}
	if src.StrokePosition != "" {
		s.StrokePosition = src.StrokePosition
	}
	if src.StrokeCap != "" {
		s.StrokeCap = src.StrokeCap
	}
	if src.StrokeJoin != "" {
		s.StrokeJoin = src.StrokeJoin
	}
	if c, ok := parseColor(src.DropShadowColor); ok {
		s.DropShadowColor = c
		s.HasDropShadow = true
		s.DropShadowX = src.DropShadowX
		s.DropShadowY = src.DropShadowY
		s.DropShadowBlur = src.DropShadowBlur
	}
	if c, ok := parseColor(src.InnerShadowColor); ok {
		s.InnerShadowColor = c
		s.HasInnerShadow = true
		s.InnerShadowX = src.InnerShadowX
		s.InnerShadowY = src.InnerShadowY
		s.InnerShadowBlur = src.InnerShadowBlur
	}
	if c, ok := parseColor(src.BackgroundColor); ok {
		s.BackgroundColor = c
		s.HasBackground = true
	}
	return s
}

func parseColor(s string) (color.RGBA, bool) {
	if s == "" {
		return color.RGBA{}, false
	}
	s = strings.TrimSpace(s)
	if len(s) == 7 && s[0] == '#' {
		return color.RGBA{
			R: hexByte(s[1:3]), G: hexByte(s[3:5]), B: hexByte(s[5:7]), A: 0xFF,
		}, true
	}
	if len(s) == 9 && s[0] == '#' {
		return color.RGBA{
			R: hexByte(s[1:3]), G: hexByte(s[3:5]), B: hexByte(s[5:7]), A: hexByte(s[7:9]),
		}, true
	}
	return color.RGBA{}, false
}

func hexByte(s string) byte {
	var v byte
	for i := 0; i < 2; i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			v = v*16 + c - '0'
		case c >= 'a' && c <= 'f':
			v = v*16 + c - 'a' + 10
		case c >= 'A' && c <= 'F':
			v = v*16 + c - 'A' + 10
		}
	}
	return v
}

// compositeRaster applies the layered rendering pipeline to the base path
// coverage. outPix is written in (w*h) RGBA order.
func compositeRaster(path geom.Path, style Style, scale float64, w, h, minPX, minPY, maxPX, maxPY int) []color.RGBA {
	if w <= 0 || h <= 0 {
		return nil
	}
	alpha := supersamplePaths(path, scale, w, h, minPX, minPY, maxPX, maxPY)
	pix := make([]color.RGBA, w*h)
	for i := range pix {
		pix[i] = color.RGBA{A: 0}
	}
	if style.HasBackground {
		for i := range pix {
			pix[i] = style.BackgroundColor
		}
	}
	if style.RenderMode == "SDF" || style.RenderMode == "MSDF" || style.RenderMode == "MTSDF" {
		sdf := signedDistanceField(alpha, w, h)
		applySDF(&pix, style, sdf, w, h)
		return pix
	}
	// Default layered raster: drop shadow → inner shadow → fill → stroke.
	if style.HasDropShadow {
		applyShadow(&pix, alpha, w, h, style.DropShadowX, style.DropShadowY, style.DropShadowBlur, style.DropShadowColor)
	}
	if style.HasFill {
		overlay(&pix, alpha, style.FillColor, 0, 0, w, h)
	}
	if style.HasInnerShadow {
		applyInnerShadow(&pix, alpha, w, h, style.InnerShadowX, style.InnerShadowY, style.InnerShadowBlur, style.InnerShadowColor)
	}
	if style.HasStroke {
		strokeMask := supersampleStroke(path, scale, w, h, minPX, minPY, maxPX, maxPY, style.StrokeWidth, style.StrokePosition)
		// Stroke layer sits on top of fill, so draw it last.
		overlay(&pix, strokeMask, style.StrokeColor, 0, 0, w, h)
	}
	return pix
}

// supersamplePaths returns an 8-bit alpha coverage map for a path, rasterized
// at 4× supersampling and downsampled to (w, h). 255 means fully inside the
// glyph, 0 means fully outside.
func supersamplePaths(path geom.Path, scale float64, w, h, minPX, minPY, maxPX, maxPY int) []uint8 {
	const ss = 4
	cover := make([]uint16, w*ss*h*ss)
	type edge struct{ x1, y1, x2, y2 float64 }
	var edges []edge
	for _, contour := range path {
		if len(contour) < 3 {
			continue
		}
		for i := range contour {
			a, b := contour[i].P, contour[(i+1)%len(contour)].P
			edges = append(edges, edge{
				x1: (a.X*scale - float64(minPX)) * ss,
				y1: (float64(maxPY) - a.Y*scale) * ss,
				x2: (b.X*scale - float64(minPX)) * ss,
				y2: (float64(maxPY) - b.Y*scale) * ss,
			})
		}
	}
	xs := make([]float64, 0, len(edges))
	for sy := 0; sy < h*ss; sy++ {
		sampleY := float64(sy) + 0.5
		xs = xs[:0]
		for _, e := range edges {
			if (e.y1 <= sampleY && e.y2 > sampleY) || (e.y2 <= sampleY && e.y1 > sampleY) {
				xs = append(xs, e.x1+(sampleY-e.y1)*(e.x2-e.x1)/(e.y2-e.y1))
			}
		}
		if len(xs) < 2 {
			continue
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			from, to := int(math.Floor(xs[i])), int(math.Ceil(xs[i+1]))
			if from < 0 {
				from = 0
			}
			if to > w*ss {
				to = w * ss
			}
			for x := from; x < to; x++ {
				cover[sy*w*ss+x]++
			}
		}
	}
	alpha := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum int
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					sum += int(cover[(y*ss+dy)*w*ss+x*ss+dx])
				}
			}
			alpha[y*w+x] = uint8(sum * 255 / (ss * ss))
		}
	}
	return alpha
}

// supersampleStroke returns the stroke coverage mask. It offsets each
// contour outward by thickness/2 and fills the resulting ring (even-odd).
func supersampleStroke(path geom.Path, scale float64, w, h, minPX, minPY, maxPX, maxPY int, thickness int, position string) []uint8 {
	if thickness <= 0 {
		return make([]uint8, w*h)
	}
	t := float64(thickness) / 2.0
	switch position {
	case "inner":
		t = -t
	case "outer":
		// keep default
	}
	var offsetPath geom.Path
	for _, contour := range path {
		if len(contour) < 3 {
			continue
		}
		offset := offsetContour(contour, t)
		if len(offset) >= 3 {
			offsetPath = append(offsetPath, offset)
		}
	}
	if len(offsetPath) == 0 {
		return make([]uint8, w*h)
	}
	// Even-odd union: draw original AND offset. For "outer", stroke is
	// outside glyph bbox → union has a gap between original outline and
	// offset outline. The even-odd fill then treats the stroke region as
	// inside. Same for "inner": inward ring. For "middle", offset outward
	// by thickness/2 and draw original again — the union ring between
	// original and offset is exactly thickness/2 on each side.
	return supersamplePaths(append(path.Clone(), offsetPath...), scale, w, h, minPX, minPY, maxPX, maxPY)
}

func offsetContour(contour geom.Contour, d float64) geom.Contour {
	n := len(contour)
	out := make(geom.Contour, n)
	for i := 0; i < n; i++ {
		prev := contour[(i-1+n)%n].P
		cur := contour[i].P
		next := contour[(i+1)%n].P
		nt := normalize(plus(perp(cur.X-prev.X, cur.Y-prev.Y), perp(next.X-cur.X, next.Y-cur.Y)))
		out[i].P.X = cur.X + nt.X*d
		out[i].P.Y = cur.Y + nt.Y*d
		out[i].On = contour[i].On
	}
	return out
}

func perp(dx, dy float64) geom.Point { return geom.Point{X: -dy, Y: dx} }
func normalize(v geom.Point) geom.Point {
	l := math.Sqrt(v.X*v.X + v.Y*v.Y)
	if l < 1e-9 {
		return v
	}
	return geom.Point{X: v.X / l, Y: v.Y / l}
}

func plus(a, b geom.Point) geom.Point { return geom.Point{X: a.X + b.X, Y: a.Y + b.Y} }

func overlay(pix *[]color.RGBA, mask []uint8, fg color.RGBA, ox, oy, w, h int) {
	p := *pix
	for y := 0; y < h; y++ {
		my := oy + y
		if my < 0 || my >= h {
			continue
		}
		for x := 0; x < w; x++ {
			mx := ox + x
			if mx < 0 || mx >= w {
				continue
			}
			a := float64(mask[y*w+x]) / 255.0
			if a < 0.002 {
				continue
			}
			idx := y*w + x
			bg := p[idx]
			src := color.RGBA{
				R: uint8(float64(fg.R) * a),
				G: uint8(float64(fg.G) * a),
				B: uint8(float64(fg.B) * a),
				A: uint8(float64(fg.A) * a),
			}
			oa := float64(bg.A) / 255.0
			sa := float64(src.A) / 255.0
			outA := sa + oa*(1-sa)
			if outA < 0.001 {
				continue
			}
			blend := func(b, s uint8) uint8 {
				v := float64(b)*oa*(1-sa) + float64(s)*sa
				return uint8(math.Round(v / outA))
			}
			p[idx] = color.RGBA{R: blend(bg.R, src.R), G: blend(bg.G, src.G), B: blend(bg.B, src.B), A: uint8(math.Round(outA * 255))}
		}
	}
}

func applyShadow(pix *[]color.RGBA, alpha []uint8, w, h, ox, oy, blur int, col color.RGBA) {
	// Drop shadow = alpha shifted then optionally blurred, drawn behind fill.
	shifted := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx := x - ox
			sy := y - oy
			if sx >= 0 && sx < w && sy >= 0 && sy < h {
				shifted[y*w+x] = alpha[sy*w+sx]
			}
		}
	}
	if blur > 0 {
		shifted = boxBlur(shifted, w, h, blur)
	}
	overlay(pix, shifted, col, 0, 0, w, h)
}

func applyInnerShadow(pix *[]color.RGBA, alpha []uint8, w, h, ox, oy, blur int, col color.RGBA) {
	// Inner shadow = invert alpha → shift → blur → mask by original alpha.
	inv := make([]uint8, w*h)
	for i := range alpha {
		inv[i] = 255 - alpha[i]
	}
	applyShadow(pix, inv, w, h, ox, oy, blur, col)
}

func boxBlur(src []uint8, w, h, r int) []uint8 {
	if r <= 0 {
		return src
	}
	dst := make([]uint8, w*h)
	tmp := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum int
			var count int
			for dx := -r; dx <= r; dx++ {
				xx := x + dx
				if xx < 0 {
					xx = 0
				}
				if xx >= w {
					xx = w - 1
				}
				sum += int(src[y*w+xx])
				count++
			}
			tmp[y*w+x] = uint8(sum / count)
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum int
			var count int
			for dy := -r; dy <= r; dy++ {
				yy := y + dy
				if yy < 0 {
					yy = 0
				}
				if yy >= h {
					yy = h - 1
				}
				sum += int(tmp[yy*w+x])
				count++
			}
			dst[y*w+x] = uint8(sum / count)
		}
	}
	return dst
}

// signedDistanceField converts the 8-bit coverage into a distance field in
// [-w, w] pixels scaled to [0, 255] where 128 is the edge, <128 is inside.
// Uses a brute-force Euclidean transform with O(w·h·k) complexity, k bounded
// by 8× padding — OK for glyph-sized buffers (a few hundred pixels each).
func signedDistanceField(alpha []uint8, w, h int) []uint8 {
	const pad = 8
	pw, ph := w+2*pad, h+2*pad
	big := make([]uint8, pw*ph)
	for y := 0; y < h; y++ {
		copy(big[(y+pad)*pw+pad:(y+pad)*pw+pad+w], alpha[y*w:(y+1)*w])
	}
	// outside: nearest inside, inside: nearest outside
	outside := nearestTransform(big, pw, ph, 0, 255)
	inside := nearestTransform(big, pw, ph, 255, 0)
	out := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := big[(y+pad)*pw+pad+x]
			var d float64
			if a > 128 {
				d = -outside[(y+pad)*pw+pad+x]
			} else {
				d = inside[(y+pad)*pw+pad+x]
			}
			// Map [-pad, pad] → [0, 255], 128 = edge.
			d = math.Max(-float64(pad), math.Min(float64(pad), d))
			out[y*w+x] = uint8(128 + math.Round(d*127/float64(pad)))
		}
	}
	return out
}

func nearestTransform(src []uint8, w, h int, target, other uint8) []float64 {
	dist := make([]float64, w*h)
	for i := range dist {
		if src[i] == target {
			dist[i] = 0
		} else {
			dist[i] = math.Inf(1)
		}
	}
	// 2D SSEDT in two passes.
	for y := 0; y < h; y++ {
		for x := 1; x < w; x++ {
			a := y*w + x
			b := y*w + x - 1
			dist[a] = math.Min(dist[a], dist[b]+1)
		}
		for x := w - 2; x >= 0; x-- {
			a := y*w + x
			b := y*w + x + 1
			dist[a] = math.Min(dist[a], dist[b]+1)
		}
	}
	for x := 0; x < w; x++ {
		for y := 1; y < h; y++ {
			a := y*w + x
			b := (y-1)*w + x
			dist[a] = math.Min(dist[a], dist[b]+1)
			if x > 0 {
				c := (y-1)*w + x - 1
				dist[a] = math.Min(dist[a], dist[c]+1.4142)
			}
			if x < w-1 {
				c := (y-1)*w + x + 1
				dist[a] = math.Min(dist[a], dist[c]+1.4142)
			}
		}
		for y := h - 2; y >= 0; y-- {
			a := y*w + x
			b := (y+1)*w + x
			dist[a] = math.Min(dist[a], dist[b]+1)
			if x > 0 {
				c := (y+1)*w + x - 1
				dist[a] = math.Min(dist[a], dist[c]+1.4142)
			}
			if x < w-1 {
				c := (y+1)*w + x + 1
				dist[a] = math.Min(dist[a], dist[c]+1.4142)
			}
		}
	}
	_ = other
	return dist
}

func applySDF(pix *[]color.RGBA, style Style, sdf []uint8, w, h int) {
	// SDF color atlas: R=inside mask, G=distance, B=distance, A=1 (opaque).
	// Unity TMP / BMFont SDF shaders read (R-0.5) as the signed distance
	// normalized to 1 unit; they sample the alpha channel for coverage.
	// We write the same 8-bit byte to R/G/B so grayscale and color SDF
	// consumers both work.
	edge := uint8(128)
	feather := 16
	fg := style.FillColor
	for i, d := range sdf {
		// coverage = clamp((edge - d + feather) / (2 * feather), 0, 1)
		t := float64(int(d)-int(edge)+feather) / float64(2*feather)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		c := uint8(math.Round(t * 255))
		(*pix)[i] = color.RGBA{R: c, G: c, B: c, A: fg.A}
	}
	_ = fmt.Sprintf // suppress unused in linters
	_ = image.Point{}
}
