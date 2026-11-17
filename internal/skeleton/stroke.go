package skeleton

import (
	"math"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

func (b *Builder) expand(s Stroke) (geom.Path, error) {
	if len(s.Cmds) == 0 {
		return nil, nil
	}
	pts := geom.Flatten(s.Cmds, 0.3)
	pts = cleanPoints(pts, s.Closed)
	if s.Closed {
		if len(pts) < 3 {
			return nil, nil
		}
		return b.expandClosed(s, pts), nil
	}
	if len(pts) < 2 {
		return nil, nil
	}
	return b.expandOpen(s, pts), nil
}

func cleanPoints(pts []geom.Point, closed bool) []geom.Point {
	out := make([]geom.Point, 0, len(pts))
	for _, p := range pts {
		if len(out) > 0 && out[len(out)-1] == p {
			continue
		}
		out = append(out, p)
	}
	if closed && len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

func tangents(pts []geom.Point, closed bool) []geom.Point {
	n := len(pts)
	out := make([]geom.Point, n)
	if n < 2 {
		return out
	}
	if closed {
		for i := 0; i < n; i++ {
			out[i] = pts[(i+1)%n].Sub(pts[(i-1+n)%n]).Normalize()
		}
		return out
	}
	out[0] = pts[1].Sub(pts[0]).Normalize()
	out[n-1] = pts[n-1].Sub(pts[n-2]).Normalize()
	for i := 1; i < n-1; i++ {
		out[i] = pts[i+1].Sub(pts[i-1]).Normalize()
	}
	return out
}

func (b *Builder) widthAt(t geom.Point, mul, tParam float64, flare bool) float64 {
	if mul <= 0 {
		mul = 1
	}
	c := b.P.Contrast
	sin := math.Abs(t.Y)
	f := 1 - c + c*(0.12+0.88*math.Pow(sin, 4))
	w := b.P.Stem * mul * f
	if flare && tParam >= 0 {
		near := math.Exp(-math.Pow(tParam/0.08, 2)) + math.Exp(-math.Pow((1-tParam)/0.08, 2))
		w *= 1 + 0.9*near
	}
	if w < 6 {
		w = 6
	}
	return w
}

func (b *Builder) expandOpen(s Stroke, pts []geom.Point) geom.Path {
	n := len(pts)
	tans := tangents(pts, false)
	normals := make([]geom.Point, n)
	widths := make([]float64, n)
	flare := s.Flare || b.P.Terminal == "flare"
	for i := 0; i < n; i++ {
		normals[i] = geom.Point{X: -tans[i].Y, Y: tans[i].X}
		widths[i] = b.widthAt(tans[i], s.Width, float64(i)/float64(n-1), flare)
	}
	left := make([]geom.Point, n)
	right := make([]geom.Point, n)
	for i := 0; i < n; i++ {
		left[i] = pts[i].Add(normals[i].Mul(widths[i] / 2))
		right[i] = pts[i].Sub(normals[i].Mul(widths[i] / 2))
	}
	capStart := s.CapStart
	if capStart == "" {
		capStart = b.P.Terminal
	}
	capEnd := s.CapEnd
	if capEnd == "" {
		capEnd = b.P.Terminal
	}
	hasSerif := b.P.Serif != "none"
	if hasSerif && s.SerifStart {
		capStart = "flat"
	}
	if hasSerif && s.SerifEnd {
		capEnd = "flat"
	}

	var nodes geom.Contour
	nodes = append(nodes, onCurve(left[0]))
	for i := 1; i < n; i++ {
		nodes = append(nodes, onCurve(left[i]))
	}
	for _, p := range endCap(pts[n-1], tans[n-1], normals[n-1], widths[n-1], capEnd) {
		nodes = append(nodes, onCurve(p))
	}
	for i := n - 1; i >= 0; i-- {
		nodes = append(nodes, onCurve(right[i]))
	}
	for _, p := range startCap(pts[0], tans[0], normals[0], widths[0], capStart) {
		nodes = append(nodes, onCurve(p))
	}
	nodes = cleanContour(nodes)
	if len(nodes) < 3 {
		return nil
	}
	out := geom.Path{ensurePositive(nodes)}

	if hasSerif && s.SerifStart {
		if serif := b.addSerif(pts[0], tans[0].Mul(-1), widths[0]); len(serif) > 0 {
			out = append(out, serif...)
		}
	}
	if hasSerif && s.SerifEnd {
		if serif := b.addSerif(pts[n-1], tans[n-1], widths[n-1]); len(serif) > 0 {
			out = append(out, serif...)
		}
	}
	return out
}

func endCap(p, t, n geom.Point, w float64, style string) []geom.Point {
	switch style {
	case "round":
		out := make([]geom.Point, 0, 7)
		for k := 1; k <= 7; k++ {
			phi := math.Pi * float64(k) / 8
			out = append(out, p.Add(n.Mul(math.Cos(phi)*w/2)).Add(t.Mul(math.Sin(phi)*w/2)))
		}
		return out
	case "angled":
		return []geom.Point{p.Add(t.Mul(w * 0.35))}
	default:
		return nil
	}
}

func startCap(p, t, n geom.Point, w float64, style string) []geom.Point {
	switch style {
	case "round":
		out := make([]geom.Point, 0, 7)
		for k := 1; k <= 7; k++ {
			phi := math.Pi * float64(k) / 8
			out = append(out, p.Sub(n.Mul(math.Cos(phi)*w/2)).Sub(t.Mul(math.Sin(phi)*w/2)))
		}
		return out
	case "angled":
		return []geom.Point{p.Sub(t.Mul(w * 0.35))}
	default:
		return nil
	}
}

func (b *Builder) expandClosed(s Stroke, pts []geom.Point) geom.Path {
	n := len(pts)
	tans := tangents(pts, true)
	left := make([]geom.Point, n)
	right := make([]geom.Point, n)
	for i := 0; i < n; i++ {
		normal := geom.Point{X: -tans[i].Y, Y: tans[i].X}
		w := b.widthAt(tans[i], s.Width, -1, false)
		left[i] = pts[i].Add(normal.Mul(w / 2))
		right[i] = pts[i].Sub(normal.Mul(w / 2))
	}
	l := makeContour(left)
	r := makeContour(right)
	la := signedArea(l)
	ra := signedArea(r)
	if la == 0 || ra == 0 {
		return nil
	}
	var outer, hole geom.Contour
	if math.Abs(la) >= math.Abs(ra) {
		outer, hole = l, r
	} else {
		outer, hole = r, l
	}
	if signedArea(outer) < 0 {
		reverseContour(outer)
	}
	if signedArea(hole) > 0 {
		reverseContour(hole)
	}
	return geom.Path{outer, hole}
}

func (b *Builder) addSerif(p, dir geom.Point, w float64) geom.Path {
	if b.P.Serif == "none" {
		return nil
	}
	dir = dir.Normalize()
	if dir == (geom.Point{}) || math.Abs(dir.Y) < 0.55 {
		return nil
	}
	n := geom.Point{X: -dir.Y, Y: dir.X}
	half := w*0.55 + b.P.SerifSize*0.45
	var thick float64
	switch b.P.Serif {
	case "hairline":
		thick = math.Max(8, w*0.08)
		half *= 1.15
	case "bracketed":
		thick = math.Max(10, w*0.14)
	default:
		thick = math.Max(12, w*0.16)
	}
	v0, v1 := -0.8*thick, 0.2*thick
	var local []geom.Point
	switch b.P.Serif {
	case "bracketed":
		local = []geom.Point{
			{X: -half, Y: v0}, {X: half, Y: v0},
			{X: half * 0.72, Y: v1}, {X: -half * 0.72, Y: v1},
		}
	default:
		local = []geom.Point{
			{X: -half, Y: v0}, {X: half, Y: v0},
			{X: half, Y: v1}, {X: -half, Y: v1},
		}
	}
	c := make(geom.Contour, len(local))
	for i, q := range local {
		c[i] = onCurve(p.Add(n.Mul(q.X)).Add(dir.Mul(q.Y)))
	}
	c = ensurePositive(c)
	return geom.Path{c}
}

func makeContour(pts []geom.Point) geom.Contour {
	c := make(geom.Contour, len(pts))
	for i, p := range pts {
		c[i] = onCurve(p)
	}
	return c
}

func onCurve(p geom.Point) geom.Node { return geom.Node{P: p, On: true} }

func cleanContour(c geom.Contour) geom.Contour {
	out := c[:0]
	for _, n := range c {
		if len(out) > 0 && out[len(out)-1].P == n.P {
			continue
		}
		out = append(out, n)
	}
	if len(out) > 1 && out[0].P == out[len(out)-1].P {
		out = out[:len(out)-1]
	}
	return out
}

func signedArea(c geom.Contour) float64 {
	return (geom.Path{c}).SignedArea(c)
}

func reverseContour(c geom.Contour) {
	for l, r := 0, len(c)-1; l < r; l, r = l+1, r-1 {
		c[l], c[r] = c[r], c[l]
	}
}

func ensurePositive(c geom.Contour) geom.Contour {
	if signedArea(c) < 0 {
		reverseContour(c)
	}
	return c
}
