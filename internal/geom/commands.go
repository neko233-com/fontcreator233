package geom

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// CmdKind is a source path command.
type CmdKind byte

const (
	CmdMove CmdKind = iota
	CmdLine
	CmdQuad
	CmdCubic
	CmdClose
)

func (k CmdKind) String() string {
	switch k {
	case CmdMove:
		return "M"
	case CmdLine:
		return "L"
	case CmdQuad:
		return "Q"
	case CmdCubic:
		return "C"
	case CmdClose:
		return "Z"
	default:
		return "?"
	}
}

// Cmd is one absolute source command. P[0] is the only point for move/line,
// control+end for quad, and control1+control2+end for cubic.
type Cmd struct {
	Kind CmdKind
	P    [3]Point
}

type pathParser struct {
	s      string
	i      int
	points int
}

func (p *pathParser) eof() bool { return p.i >= len(p.s) }

func (p *pathParser) skipSep() {
	for !p.eof() {
		switch p.s[p.i] {
		case ' ', '\t', '\n', '\r', ',':
			p.i++
		default:
			return
		}
	}
}

func (p *pathParser) errf(format string, args ...any) error {
	return fmt.Errorf("path offset %d: %s", p.i, fmt.Sprintf(format, args...))
}

func isCommandByte(c byte) bool {
	switch c {
	case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v', 'C', 'c', 'S', 's',
		'Q', 'q', 'T', 't', 'Z', 'z':
		return true
	}
	return false
}

func isNumberStart(c byte) bool {
	return c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9')
}

func (p *pathParser) hasNumber() bool {
	p.skipSep()
	if p.eof() {
		return false
	}
	if isNumberStart(p.s[p.i]) {
		return true
	}
	return p.nonFinitePrefix()
}

func (p *pathParser) nonFinitePrefix() bool {
	rest := p.s[p.i:]
	for _, lit := range []string{"NaN", "nan", "Inf", "inf", "Infinity", "infinity"} {
		if strings.HasPrefix(rest, lit) {
			return true
		}
	}
	return false
}

func (p *pathParser) number() (float64, error) {
	p.skipSep()
	start := p.i
	if p.eof() {
		return 0, p.errf("expected number")
	}
	if p.nonFinitePrefix() {
		for _, lit := range []string{"Infinity", "infinity", "NaN", "nan", "Inf", "inf"} {
			if strings.HasPrefix(p.s[p.i:], lit) {
				p.i += len(lit)
				break
			}
		}
		return 0, p.errf("non-finite number %q", p.s[start:p.i])
	}
	if p.s[p.i] == '+' || p.s[p.i] == '-' {
		p.i++
	}
	digits := false
	for !p.eof() && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
		digits = true
	}
	if !p.eof() && p.s[p.i] == '.' {
		p.i++
		for !p.eof() && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
			digits = true
		}
	}
	if !digits {
		return 0, p.errf("expected number")
	}
	if !p.eof() && (p.s[p.i] == 'e' || p.s[p.i] == 'E') {
		save := p.i
		p.i++
		if !p.eof() && (p.s[p.i] == '+' || p.s[p.i] == '-') {
			p.i++
		}
		expDigits := false
		for !p.eof() && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
			expDigits = true
		}
		if !expDigits {
			p.i = save
		}
	}
	v, err := strconv.ParseFloat(p.s[start:p.i], 64)
	if err != nil {
		return 0, p.errf("invalid number %q", p.s[start:p.i])
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, p.errf("non-finite number %q", p.s[start:p.i])
	}
	return v, nil
}

func (p *pathParser) point() (Point, error) {
	p.points++
	if p.points > 200000 {
		return Point{}, p.errf("too many path points")
	}
	x, err := p.number()
	if err != nil {
		return Point{}, err
	}
	y, err := p.number()
	if err != nil {
		return Point{}, err
	}
	return Point{x, y}, nil
}

func (p *pathParser) group(n int) ([]Point, error) {
	pts := make([]Point, n)
	for i := range pts {
		pt, err := p.point()
		if err != nil {
			return nil, err
		}
		pts[i] = pt
	}
	return pts, nil
}

// ParseCommands parses the SVG path subset used by skeletons and SVG overrides.
// Arc commands are rejected explicitly.
func ParseCommands(d string) ([]Cmd, error) {
	if strings.TrimSpace(d) == "" {
		return nil, fmt.Errorf("path offset 0: empty path")
	}
	p := &pathParser{s: d}
	var out []Cmd
	var cur, start Point
	var lastCubic, lastQuad Point
	lastKind := CmdMove
	haveMove := false

	for {
		p.skipSep()
		if p.eof() {
			break
		}
		if !isCommandByte(p.s[p.i]) {
			if p.s[p.i] == 'A' || p.s[p.i] == 'a' {
				return nil, p.errf("arc commands are not supported")
			}
			return nil, p.errf("expected command letter, got %q", p.s[p.i])
		}
		cmd := p.s[p.i]
		p.i++
		if !haveMove && cmd != 'M' && cmd != 'm' {
			return nil, p.errf("path must start with a move command")
		}
		relative := cmd >= 'a' && cmd <= 'z'
		switch cmd | 0x20 {
		case 'a':
			return nil, p.errf("arc commands are not supported")
		case 'm':
			if !p.hasNumber() {
				return nil, p.errf("move command requires coordinates")
			}
			first := true
			for p.hasNumber() {
				pt, err := p.point()
				if err != nil {
					return nil, err
				}
				if relative {
					pt = Point{cur.X + pt.X, cur.Y + pt.Y}
				}
				if first {
					cur = pt
					start = pt
					out = append(out, Cmd{Kind: CmdMove, P: [3]Point{pt}})
					haveMove = true
					first = false
				} else {
					cur = pt
					out = append(out, Cmd{Kind: CmdLine, P: [3]Point{pt}})
				}
				lastKind = CmdLine
			}
			lastCubic, lastQuad = Point{}, Point{}
		case 'l':
			if !p.hasNumber() {
				return nil, p.errf("line command requires coordinates")
			}
			for p.hasNumber() {
				pt, err := p.point()
				if err != nil {
					return nil, err
				}
				if relative {
					pt = Point{cur.X + pt.X, cur.Y + pt.Y}
				}
				out = append(out, Cmd{Kind: CmdLine, P: [3]Point{pt}})
				cur = pt
				lastKind = CmdLine
			}
		case 'h':
			if !p.hasNumber() {
				return nil, p.errf("horizontal command requires a coordinate")
			}
			for p.hasNumber() {
				x, err := p.number()
				if err != nil {
					return nil, err
				}
				if relative {
					x += cur.X
				}
				cur = Point{x, cur.Y}
				out = append(out, Cmd{Kind: CmdLine, P: [3]Point{cur}})
				lastKind = CmdLine
			}
		case 'v':
			if !p.hasNumber() {
				return nil, p.errf("vertical command requires a coordinate")
			}
			for p.hasNumber() {
				y, err := p.number()
				if err != nil {
					return nil, err
				}
				if relative {
					y += cur.Y
				}
				cur = Point{cur.X, y}
				out = append(out, Cmd{Kind: CmdLine, P: [3]Point{cur}})
				lastKind = CmdLine
			}
		case 'c':
			if !p.hasNumber() {
				return nil, p.errf("cubic command requires coordinates")
			}
			for p.hasNumber() {
				pts, err := p.group(3)
				if err != nil {
					return nil, err
				}
				if relative {
					for i := range pts {
						pts[i] = Point{cur.X + pts[i].X, cur.Y + pts[i].Y}
					}
				}
				out = append(out, Cmd{Kind: CmdCubic, P: [3]Point{pts[0], pts[1], pts[2]}})
				cur = pts[2]
				lastCubic = pts[1]
				lastKind = CmdCubic
			}
		case 's':
			if !p.hasNumber() {
				return nil, p.errf("smooth cubic command requires coordinates")
			}
			for p.hasNumber() {
				pts, err := p.group(2)
				if err != nil {
					return nil, err
				}
				if relative {
					for i := range pts {
						pts[i] = Point{cur.X + pts[i].X, cur.Y + pts[i].Y}
					}
				}
				c1 := cur
				if lastKind == CmdCubic {
					c1 = Point{2*cur.X - lastCubic.X, 2*cur.Y - lastCubic.Y}
				}
				out = append(out, Cmd{Kind: CmdCubic, P: [3]Point{c1, pts[0], pts[1]}})
				cur = pts[1]
				lastCubic = pts[0]
				lastKind = CmdCubic
			}
		case 'q':
			if !p.hasNumber() {
				return nil, p.errf("quadratic command requires coordinates")
			}
			for p.hasNumber() {
				pts, err := p.group(2)
				if err != nil {
					return nil, err
				}
				if relative {
					for i := range pts {
						pts[i] = Point{cur.X + pts[i].X, cur.Y + pts[i].Y}
					}
				}
				out = append(out, Cmd{Kind: CmdQuad, P: [3]Point{pts[0], pts[1]}})
				cur = pts[1]
				lastQuad = pts[0]
				lastKind = CmdQuad
			}
		case 't':
			if !p.hasNumber() {
				return nil, p.errf("smooth quadratic command requires coordinates")
			}
			for p.hasNumber() {
				pt, err := p.point()
				if err != nil {
					return nil, err
				}
				if relative {
					pt = Point{cur.X + pt.X, cur.Y + pt.Y}
				}
				c := cur
				if lastKind == CmdQuad {
					c = Point{2*cur.X - lastQuad.X, 2*cur.Y - lastQuad.Y}
				}
				out = append(out, Cmd{Kind: CmdQuad, P: [3]Point{c, pt}})
				cur = pt
				lastQuad = c
				lastKind = CmdQuad
			}
		case 'z':
			out = append(out, Cmd{Kind: CmdClose})
			cur = start
			lastKind = CmdClose
			lastCubic, lastQuad = Point{}, Point{}
		}
	}
	return out, nil
}

// Flatten converts commands to on-curve points. It is intended for a single
// subpath (skeleton strokes). Closed subpaths repeat their start point.
func Flatten(cmds []Cmd, tolerance float64) []Point {
	if tolerance <= 0 {
		tolerance = 0.35
	}
	var out []Point
	var cur, start Point
	for _, c := range cmds {
		switch c.Kind {
		case CmdMove:
			cur = c.P[0]
			start = cur
			out = append(out, cur)
		case CmdLine:
			cur = c.P[0]
			out = append(out, cur)
		case CmdQuad:
			flattenQuad(cur, c.P[0], c.P[1], tolerance, 0, &out)
			cur = c.P[1]
		case CmdCubic:
			flattenCubic(cur, c.P[0], c.P[1], c.P[2], tolerance, 0, &out)
			cur = c.P[2]
		case CmdClose:
			if len(out) == 0 || out[len(out)-1] != start {
				out = append(out, start)
			}
			cur = start
		}
	}
	return out
}

// FlattenContours is Flatten for paths with multiple subpaths.
func FlattenContours(cmds []Cmd, tolerance float64) [][]Point {
	if tolerance <= 0 {
		tolerance = 0.35
	}
	var out [][]Point
	var cur, start Point
	for _, c := range cmds {
		switch c.Kind {
		case CmdMove:
			cur = c.P[0]
			start = cur
			out = append(out, []Point{cur})
		case CmdLine:
			cur = c.P[0]
			if len(out) == 0 {
				out = append(out, []Point{cur})
			} else {
				out[len(out)-1] = append(out[len(out)-1], cur)
			}
		case CmdQuad:
			if len(out) == 0 {
				out = append(out, []Point{cur})
			}
			flattenQuad(cur, c.P[0], c.P[1], tolerance, 0, &out[len(out)-1])
			cur = c.P[1]
		case CmdCubic:
			if len(out) == 0 {
				out = append(out, []Point{cur})
			}
			flattenCubic(cur, c.P[0], c.P[1], c.P[2], tolerance, 0, &out[len(out)-1])
			cur = c.P[2]
		case CmdClose:
			if len(out) > 0 {
				last := out[len(out)-1]
				if len(last) == 0 || last[len(last)-1] != start {
					out[len(out)-1] = append(last, start)
				}
			}
			cur = start
		}
	}
	return out
}

func flattenQuad(p0, c, p1 Point, tol float64, depth int, out *[]Point) {
	if depth >= 16 {
		*out = append(*out, p1)
		return
	}
	c1 := p0.Lerp(c, 0.5)
	c2 := c.Lerp(p1, 0.5)
	m := c1.Lerp(c2, 0.5)
	dev := distPointLine(c, p0, p1)
	if dev <= tol {
		*out = append(*out, p1)
		return
	}
	flattenQuad(p0, c1, m, tol, depth+1, out)
	flattenQuad(m, c2, p1, tol, depth+1, out)
}

func flattenCubic(p0, c1, c2, p1 Point, tol float64, depth int, out *[]Point) {
	if depth >= 16 {
		*out = append(*out, p1)
		return
	}
	dev := math.Max(distPointLine(c1, p0, p1), distPointLine(c2, p0, p1))
	if dev <= tol {
		*out = append(*out, p1)
		return
	}
	p01 := p0.Lerp(c1, 0.5)
	p12 := c1.Lerp(c2, 0.5)
	p23 := c2.Lerp(p1, 0.5)
	p012 := p01.Lerp(p12, 0.5)
	p123 := p12.Lerp(p23, 0.5)
	p0123 := p012.Lerp(p123, 0.5)
	flattenCubic(p0, p01, p012, p0123, tol, depth+1, out)
	flattenCubic(p0123, p123, p23, p1, tol, depth+1, out)
}

func distPointLine(p, a, b Point) float64 {
	ab := b.Sub(a)
	if ab.Len() < 1e-12 {
		return p.Sub(a).Len()
	}
	return math.Abs(ab.Cross(p.Sub(a))) / ab.Len()
}
