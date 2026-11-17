package geom

import "math"

// Point is a location in font design units.
type Point struct {
	X, Y float64
}

// Add returns p+q.
func (p Point) Add(q Point) Point { return Point{p.X + q.X, p.Y + q.Y} }

// Sub returns p-q.
func (p Point) Sub(q Point) Point { return Point{p.X - q.X, p.Y - q.Y} }

// Mul returns p*s.
func (p Point) Mul(s float64) Point { return Point{p.X * s, p.Y * s} }

// Len returns the Euclidean length.
func (p Point) Len() float64 { return math.Hypot(p.X, p.Y) }

// Normalize returns a unit vector; zero vectors are returned unchanged.
func (p Point) Normalize() Point {
	l := p.Len()
	if l == 0 {
		return Point{}
	}
	return Point{p.X / l, p.Y / l}
}

// Dot returns the dot product.
func (p Point) Dot(q Point) float64 { return p.X*q.X + p.Y*q.Y }

// Cross returns the scalar cross product.
func (p Point) Cross(q Point) float64 { return p.X*q.Y - p.Y*q.X }

// Lerp interpolates from p to q at t.
func (p Point) Lerp(q Point, t float64) Point {
	return Point{p.X + (q.X-p.X)*t, p.Y + (q.Y-p.Y)*t}
}

// Rect is an axis-aligned rectangle.
type Rect struct {
	MinX, MinY, MaxX, MaxY float64
}

// Width returns the rectangle width.
func (r Rect) Width() float64 { return r.MaxX - r.MinX }

// Height returns the rectangle height.
func (r Rect) Height() float64 { return r.MaxY - r.MinY }

// Affine is a 2D affine transform: x'=A*x+B*y+C, y'=D*x+E*y+F.
type Affine struct {
	A, B, C, D, E, F float64
}

// Apply transforms p.
func (m Affine) Apply(p Point) Point {
	return Point{
		X: m.A*p.X + m.B*p.Y + m.C,
		Y: m.D*p.X + m.E*p.Y + m.F,
	}
}

// Scale returns a scaling transform.
func Scale(sx, sy float64) Affine { return Affine{A: sx, E: sy} }

// Translate returns a translation transform.
func Translate(dx, dy float64) Affine { return Affine{A: 1, E: 1, C: dx, F: dy} }

// ShearX returns a transform that moves x by k*y.
func ShearX(k float64) Affine { return Affine{A: 1, B: k, E: 1} }
