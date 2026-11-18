package fontbuild

import "github.com/neko233-com/fontcreator233/internal/geom"

type glyphPoint struct {
	x, y int
	on   bool
}

func encodeGlyph(g Glyph) ([]byte, glyphBox, error) {
	var points []glyphPoint
	var endPts []uint16
	for _, contour := range g.Path {
		var cleaned []geom.Node
		for _, n := range contour {
			x, err := roundInt16(n.P.X)
			if err != nil {
				return nil, glyphBox{}, err
			}
			y, err := roundInt16(n.P.Y)
			if err != nil {
				return nil, glyphBox{}, err
			}
			p := geom.Node{P: geom.Point{X: float64(x), Y: float64(y)}, On: n.On}
			if len(cleaned) > 0 && cleaned[len(cleaned)-1].P == p.P {
				continue
			}
			cleaned = append(cleaned, p)
		}
		if len(cleaned) > 1 && cleaned[0].P == cleaned[len(cleaned)-1].P {
			cleaned = cleaned[:len(cleaned)-1]
		}
		if len(cleaned) == 0 {
			continue
		}
		for _, n := range cleaned {
			points = append(points, glyphPoint{x: int(n.P.X), y: int(n.P.Y), on: n.On})
		}
		if len(points) > 65535 {
			return nil, glyphBox{}, errIndexTooLarge
		}
		endPts = append(endPts, uint16(len(points)-1))
	}
	if len(points) == 0 {
		return nil, glyphBox{empty: true}, nil
	}
	if len(endPts) > 32767 {
		return nil, glyphBox{}, errIndexTooLarge
	}
	box := glyphBox{empty: false}
	box.xMin, box.yMin = int16(points[0].x), int16(points[0].y)
	box.xMax, box.yMax = box.xMin, box.yMin
	for _, p := range points[1:] {
		if int16(p.x) < box.xMin {
			box.xMin = int16(p.x)
		}
		if int16(p.y) < box.yMin {
			box.yMin = int16(p.y)
		}
		if int16(p.x) > box.xMax {
			box.xMax = int16(p.x)
		}
		if int16(p.y) > box.yMax {
			box.yMax = int16(p.y)
		}
	}

	var flags []byte
	var xData, yData []byte
	prevX, prevY := 0, 0
	for _, p := range points {
		dx, dy := p.x-prevX, p.y-prevY
		if dx < -32768 || dx > 32767 || dy < -32768 || dy > 32767 {
			return nil, glyphBox{}, errDeltaOverflow
		}
		prevX, prevY = p.x, p.y
		var flag byte
		if p.on {
			flag |= 0x01
		}
		if dx == 0 {
			flag |= 0x10
		} else if dx >= -255 && dx <= 255 {
			flag |= 0x02
			if dx > 0 {
				flag |= 0x10
			}
			xData = append(xData, byte(absInt(dx)))
		} else {
			xData = appendI16(xData, int16(dx))
		}
		if dy == 0 {
			flag |= 0x20
		} else if dy >= -255 && dy <= 255 {
			flag |= 0x04
			if dy > 0 {
				flag |= 0x20
			}
			yData = append(yData, byte(absInt(dy)))
		} else {
			yData = appendI16(yData, int16(dy))
		}
		flags = append(flags, flag)
	}

	var compressed []byte
	for i := 0; i < len(flags); {
		f := flags[i]
		run := 1
		for i+run < len(flags) && flags[i+run] == f && run < 256 {
			run++
		}
		if run > 1 {
			compressed = append(compressed, f|0x08, byte(run-1))
		} else {
			compressed = append(compressed, f)
		}
		i += run
	}

	var out []byte
	out = appendI16(out, int16(len(endPts)))
	out = appendI16(out, box.xMin)
	out = appendI16(out, box.yMin)
	out = appendI16(out, box.xMax)
	out = appendI16(out, box.yMax)
	for _, e := range endPts {
		out = appendU16(out, e)
	}
	out = appendU16(out, 0) // instructions
	out = append(out, compressed...)
	out = append(out, xData...)
	out = append(out, yData...)
	return out, box, nil
}

var (
	errIndexTooLarge = indexError("glyph has too many points or contours")
	errDeltaOverflow = indexError("glyph coordinate delta exceeds int16 range")
)

type indexError string

func (e indexError) Error() string { return string(e) }

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
