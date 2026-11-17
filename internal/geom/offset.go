package geom

import "math"

// Offset expands (d>0) or shrinks (d<0) the filled region represented by p.
// Outer contours are assumed to have positive signed area and holes negative.
// The implementation is a per-vertex miter offset; it is intended for small
// synthetic weight changes, not for general polygon booleans.
func Offset(p Path, d float64) Path {
	if d == 0 || len(p) == 0 {
		return p.Clone()
	}
	out := p.Clone()
	for ci, c := range out {
		n := len(c)
		if n < 3 {
			continue
		}
		area := p.SignedArea(c)
		if math.Abs(area) < 1e-9 {
			continue
		}
		for i := 0; i < n; i++ {
			prev := c[(i-1+n)%n].P
			cur := c[i].P
			next := c[(i+1)%n].P
			e1 := cur.Sub(prev)
			e2 := next.Sub(cur)
			n1 := Point{e1.Y, -e1.X}.Normalize()
			n2 := Point{e2.Y, -e2.X}.Normalize()
			if n1 == (Point{}) {
				n1 = n2
			}
			if n2 == (Point{}) {
				n2 = n1
			}
			m := n1.Add(n2).Normalize()
			if m == (Point{}) {
				m = n1
			}
			denom := m.Dot(n1)
			if denom < 0.2 {
				denom = 0.2
			}
			off := m.Mul(d / denom)
			if l := off.Len(); l > 2*math.Abs(d) {
				off = off.Mul(2 * math.Abs(d) / l)
			}
			out[ci][i].P = cur.Add(off)
		}
	}
	return out
}
