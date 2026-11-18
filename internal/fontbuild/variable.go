package fontbuild

import "math"

// VariableRange returns the wght axis range for a default weight. The axis is
// deliberately one-dimensional and opt-in; glyphs outside the Latin/symbol
// range stay at their default outlines.
func VariableRange(weight int) (min, def, max int) {
	def = weight
	if def < 100 {
		def = 100
	}
	if def > 900 {
		def = 900
	}
	min = def - 200
	max = def + 200
	if min < 100 {
		min = 100
	}
	if max > 900 {
		max = 900
	}
	if max <= min {
		min, max = 100, 900
	}
	return min, def, max
}

func shouldVary(g Glyph) bool {
	if len(g.Path) == 0 {
		return false
	}
	if g.NoCmap {
		return true
	}
	return g.Rune != 0 && g.Rune < 0x2E80
}

func variationDeltas(points []glyphPoint, box glyphBox, scale float64) ([]int16, []int16) {
	cx := (float64(box.xMin) + float64(box.xMax)) / 2
	cy := (float64(box.yMin) + float64(box.yMax)) / 2
	dx := make([]int16, len(points))
	dy := make([]int16, len(points))
	for i, p := range points {
		dx[i] = int16(math.Round((float64(p.x) - cx) * scale))
		dy[i] = int16(math.Round((float64(p.y) - cy) * scale))
	}
	// gvar point counts include four phantom points (advance and vertical
	// metrics). Keep them at zero so metrics stay stable across the axis.
	dx = append(dx, 0, 0, 0, 0)
	dy = append(dy, 0, 0, 0, 0)
	return dx, dy
}

// packDeltas uses the gvar packed-delta run format: zero runs, int8 runs, then
// int16 runs, each capped at 64 values.
func packDeltas(deltas []int16) []byte {
	var out []byte
	for i := 0; i < len(deltas); {
		if deltas[i] == 0 {
			run := 0
			for i+run < len(deltas) && deltas[i+run] == 0 && run < 64 {
				run++
			}
			out = append(out, 0x80|byte(run-1))
			i += run
			continue
		}
		if deltas[i] >= -128 && deltas[i] <= 127 {
			run := 0
			for i+run < len(deltas) && deltas[i+run] >= -128 && deltas[i+run] <= 127 && run < 64 {
				run++
			}
			out = append(out, byte(run-1))
			for j := 0; j < run; j++ {
				out = append(out, byte(int8(deltas[i+j])))
			}
			i += run
			continue
		}
		run := 0
		for i+run < len(deltas) && run < 64 {
			run++
		}
		out = append(out, 0x40|byte(run-1))
		for j := 0; j < run; j++ {
			out = appendI16(out, deltas[i+j])
		}
		i += run
	}
	return out
}

func buildFvar(cfg Config) []byte {
	min, def, max := VariableRange(cfg.WeightClass)
	b := make([]byte, 36)
	putU16(b, 0, 1)  // major
	putU16(b, 2, 0)  // minor
	putU16(b, 4, 16) // axes array offset
	putU16(b, 6, 2)  // reserved
	putU16(b, 8, 1)  // axis count
	putU16(b, 10, 20)
	putU16(b, 12, 0) // instance count
	putU16(b, 14, 8) // instance size: name ID + flags + one axis coordinate
	copy(b[16:20], "wght")
	putU32(b, 20, fixed16(min))
	putU32(b, 24, fixed16(def))
	putU32(b, 28, fixed16(max))
	putU16(b, 32, 0)   // flags
	putU16(b, 34, 256) // axis name ID
	return b
}

func fixed16(v int) uint32 { return uint32(int32(v * 65536)) }

func buildGvar(glyphs []Glyph) ([]byte, error) {
	datas := make([][]byte, len(glyphs))
	for i, g := range glyphs {
		if !shouldVary(g) {
			continue
		}
		points, _, box, err := flattenGlyph(g)
		if err != nil {
			return nil, err
		}
		if len(points) == 0 {
			continue
		}
		lightX, lightY := variationDeltas(points, box, -0.02)
		boldX, boldY := variationDeltas(points, box, 0.04)
		light := append(packDeltas(lightX), packDeltas(lightY)...)
		bold := append(packDeltas(boldX), packDeltas(boldY)...)
		if len(light) == 0 && len(bold) == 0 {
			continue
		}
		data := make([]byte, 0, 1+len(light)+len(bold))
		data = append(data, 0x00) // shared point numbers: all points
		data = append(data, light...)
		data = append(data, bold...)
		out := make([]byte, 0, 12+len(data)+1)
		out = appendU16(out, 2|0x8000) // two tuples, shared point numbers
		out = appendU16(out, 12)       // offset to serialized data
		out = appendU16(out, uint16(len(light)))
		out = appendU16(out, 0) // shared tuple 0: -1.0
		out = appendU16(out, uint16(len(bold)))
		out = appendU16(out, 1) // shared tuple 1: +1.0
		out = append(out, data...)
		if len(out)%2 != 0 {
			out = append(out, 0)
		}
		datas[i] = out
	}

	total := 0
	for _, d := range datas {
		total += len(d)
	}
	if total == 0 {
		return nil, nil
	}
	offsetsSize := 4 * (len(glyphs) + 1)
	offShared := 20 + offsetsSize
	offData := offShared + 4
	out := make([]byte, offData+total)
	putU16(out, 0, 1) // version
	putU16(out, 2, 0) // reserved
	putU16(out, 4, 1) // axis count
	putU16(out, 6, 2) // shared tuple count
	putU32(out, 8, uint32(offShared))
	putU16(out, 12, uint16(len(glyphs)))
	putU16(out, 14, 1) // long glyph variation offsets
	putU32(out, 16, uint32(offData))
	putI16(out, offShared, -16384)  // -1.0 F2DOT14
	putI16(out, offShared+2, 16384) // +1.0 F2DOT14

	pos := offData
	for i, d := range datas {
		putU32(out, 20+4*i, uint32(pos-offData))
		copy(out[pos:], d)
		pos += len(d)
	}
	putU32(out, 20+4*len(glyphs), uint32(pos-offData))
	return out, nil
}
