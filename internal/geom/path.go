package geom

import "math"

// Node is one point on an outline. Off-curve nodes are quadratic control
// points. Generated outlines normally use only on-curve nodes.
type Node struct {
	P  Point
	On bool
}

// Contour is a closed outline. The closing segment is implicit.
type Contour []Node

// Path is a set of closed contours. Outer contours have positive signed area;
// holes have negative signed area.
type Path []Contour

// Clone returns a deep copy.
func (p Path) Clone() Path {
	out := make(Path, len(p))
	for i, c := range p {
		out[i] = append(Contour(nil), c...)
	}
	return out
}

// Bounds returns the axis-aligned bounds of all nodes.
func (p Path) Bounds() Rect {
	if len(p) == 0 {
		return Rect{}
	}
	first := true
	var b Rect
	for _, c := range p {
		for _, n := range c {
			if first {
				b = Rect{n.P.X, n.P.Y, n.P.X, n.P.Y}
				first = false
				continue
			}
			if n.P.X < b.MinX {
				b.MinX = n.P.X
			}
			if n.P.Y < b.MinY {
				b.MinY = n.P.Y
			}
			if n.P.X > b.MaxX {
				b.MaxX = n.P.X
			}
			if n.P.Y > b.MaxY {
				b.MaxY = n.P.Y
			}
		}
	}
	return b
}

// SignedArea returns the shoelace area of a contour. Positive means
// counter-clockwise in a y-up coordinate system.
func (p Path) SignedArea(c Contour) float64 {
	if len(c) < 3 {
		return 0
	}
	var sum float64
	for i := range c {
		a := c[i].P
		b := c[(i+1)%len(c)].P
		sum += a.Cross(b)
	}
	return sum / 2
}

// Reverse returns a copy of p with contour i reversed.
func (p Path) Reverse(i int) Path {
	out := p.Clone()
	if i < 0 || i >= len(out) {
		return out
	}
	c := out[i]
	for l, r := 0, len(c)-1; l < r; l, r = l+1, r-1 {
		c[l], c[r] = c[r], c[l]
	}
	return out
}

// Transformed returns a transformed deep copy.
func (p Path) Transformed(m Affine) Path {
	out := p.Clone()
	for ci := range out {
		for ni := range out[ci] {
			out[ci][ni].P = m.Apply(out[ci][ni].P)
		}
	}
	return out
}

// Circle returns a 48-point closed circle with positive (CCW) area.
func Circle(cx, cy, r float64) Path {
	const n = 48
	c := make(Contour, n)
	for i := 0; i < n; i++ {
		a := 2 * math.Pi * float64(i) / n
		c[i] = Node{P: Point{cx + r*math.Cos(a), cy + r*math.Sin(a)}, On: true}
	}
	return Path{c}
}

// RectPath returns a CCW rectangle with positive area.
func RectPath(x0, y0, x1, y1 float64) Path {
	return Path{{
		{P: Point{x0, y0}, On: true},
		{P: Point{x1, y0}, On: true},
		{P: Point{x1, y1}, On: true},
		{P: Point{x0, y1}, On: true},
	}}
}
