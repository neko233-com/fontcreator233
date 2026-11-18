package fontbuild

import (
	"sort"
	"unicode"
)

// buildMarkBaseSubtable creates a GPOS MarkBasePos subtable for combining
// marks (U+0300–U+036F) over letters and digits. It uses simple top-center
// anchors; this is intentionally not a full mark-shaping implementation.
func buildMarkBaseSubtable(glyphs []Glyph, capHeight, xHeight int) []byte {
	var marks, bases []uint16
	for i, g := range glyphs {
		if g.Rune >= 0x0300 && g.Rune <= 0x036F && len(g.Path) > 0 {
			marks = append(marks, uint16(i))
			continue
		}
		if g.NoCmap || g.Rune == 0 || len(g.Path) == 0 || g.Advance <= 0 {
			continue
		}
		inAlphabet := (g.Rune >= 0x0020 && g.Rune <= 0x052F) || (g.Rune >= 0x1E00 && g.Rune <= 0x1EFF)
		if inAlphabet && (unicode.IsLetter(g.Rune) || unicode.IsDigit(g.Rune)) {
			bases = append(bases, uint16(i))
		}
	}
	if len(marks) == 0 || len(bases) == 0 {
		return nil
	}
	sort.Slice(marks, func(i, j int) bool { return marks[i] < marks[j] })
	sort.Slice(bases, func(i, j int) bool { return bases[i] < bases[j] })
	markCoverage := buildCoverage(marks)
	baseCoverage := buildCoverage(bases)

	markArray := make([]byte, 2+4*len(marks))
	putU16(markArray, 0, uint16(len(marks)))
	markAnchorOff := 2 + 4*len(marks)
	anchors := make([]byte, 0, 6*len(marks))
	for i, gid := range marks {
		b := glyphs[gid].Path.Bounds()
		putU16(markArray, 2+4*i, 0)
		putU16(markArray, 4+4*i, uint16(markAnchorOff+6*i))
		anchors = append(anchors, makeAnchor(0, int16(b.MinY))...)
	}
	markArray = append(markArray, anchors...)

	baseHeader := 2 + 2*len(bases)
	baseAnchorOff := baseHeader
	baseArray := make([]byte, baseHeader)
	putU16(baseArray, 0, uint16(len(bases)))
	anchors = anchors[:0]
	for i, gid := range bases {
		g := glyphs[gid]
		y := xHeight
		if unicode.IsUpper(g.Rune) || unicode.IsDigit(g.Rune) {
			y = capHeight
		}
		putU16(baseArray, 2+2*i, uint16(baseAnchorOff+6*i))
		anchors = append(anchors, makeAnchor(int16(g.Advance/2), int16(y))...)
	}
	baseArray = append(baseArray, anchors...)

	header := 12
	total := header + len(markCoverage) + len(baseCoverage) + len(markArray) + len(baseArray)
	out := make([]byte, total)
	putU16(out, 0, 1)
	putU16(out, 2, uint16(header))
	putU16(out, 4, uint16(header+len(markCoverage)))
	putU16(out, 6, 1) // mark class count
	putU16(out, 8, uint16(header+len(markCoverage)+len(baseCoverage)))
	putU16(out, 10, uint16(header+len(markCoverage)+len(baseCoverage)+len(markArray)))
	off := header
	copy(out[off:], markCoverage)
	off += len(markCoverage)
	copy(out[off:], baseCoverage)
	off += len(baseCoverage)
	copy(out[off:], markArray)
	off += len(markArray)
	copy(out[off:], baseArray)
	return out
}

// buildMarkMarkSubtable creates a GPOS MarkMarkPos subtable so a second
// combining mark stacks on top of the first. Anchors are deliberately simple:
// mark-to-mark uses the top of mark 1 and the origin of mark 2.
func buildMarkMarkSubtable(glyphs []Glyph) []byte {
	var marks []uint16
	for i, g := range glyphs {
		if g.Rune >= 0x0300 && g.Rune <= 0x036F && len(g.Path) > 0 {
			marks = append(marks, uint16(i))
		}
	}
	if len(marks) < 2 {
		return nil
	}
	sort.Slice(marks, func(i, j int) bool { return marks[i] < marks[j] })
	coverage := buildCoverage(marks)

	mark1Header := 2 + 4*len(marks)
	mark1 := make([]byte, mark1Header)
	putU16(mark1, 0, uint16(len(marks)))
	anchors := make([]byte, 0, 6*len(marks))
	for i, gid := range marks {
		b := glyphs[gid].Path.Bounds()
		putU16(mark1, 2+4*i, 0) // mark class
		putU16(mark1, 4+4*i, uint16(mark1Header+6*i))
		// HarfBuzz subtracts the mark-1 anchor from the already-positioned
		// mark-1 origin, so a negative Y places mark 2 above mark 1.
		anchors = append(anchors, makeAnchor(0, -int16(b.MaxY))...)
	}
	mark1 = append(mark1, anchors...)

	mark2Header := 2 + 2*len(marks)
	mark2 := make([]byte, mark2Header)
	putU16(mark2, 0, uint16(len(marks)))
	anchors = anchors[:0]
	for i := range marks {
		putU16(mark2, 2+2*i, uint16(mark2Header+6*i))
		anchors = append(anchors, makeAnchor(0, 0)...)
	}
	mark2 = append(mark2, anchors...)

	header := 12
	total := header + 2*len(coverage) + len(mark1) + len(mark2)
	out := make([]byte, total)
	putU16(out, 0, 1)
	putU16(out, 2, uint16(header))
	putU16(out, 4, uint16(header+len(coverage)))
	putU16(out, 6, 1) // mark class count
	putU16(out, 8, uint16(header+2*len(coverage)))
	putU16(out, 10, uint16(header+2*len(coverage)+len(mark1)))
	off := header
	copy(out[off:], coverage)
	off += len(coverage)
	copy(out[off:], coverage)
	off += len(coverage)
	copy(out[off:], mark1)
	off += len(mark1)
	copy(out[off:], mark2)
	return out
}

func makeAnchor(x, y int16) []byte {
	b := make([]byte, 6)
	putU16(b, 0, 1)
	putI16(b, 2, x)
	putI16(b, 4, y)
	return b
}

func hasMarkGlyphs(glyphs []Glyph) bool {
	for _, g := range glyphs {
		if g.Rune >= 0x0300 && g.Rune <= 0x036F && len(g.Path) > 0 {
			return true
		}
	}
	return false
}

// buildGDEF marks combining marks as class 3 and non-cmap alternates as class
// 2; other letters/digits are class 1. It is deliberately conservative.
func buildGDEF(glyphs []Glyph) []byte {
	classes := make([]uint16, len(glyphs))
	for i, g := range glyphs {
		switch {
		case g.Rune >= 0x0300 && g.Rune <= 0x036F:
			classes[i] = 3
		case g.NoCmap:
			classes[i] = 2
		case g.Rune != 0 && (unicode.IsLetter(g.Rune) || unicode.IsDigit(g.Rune)):
			classes[i] = 1
		}
	}
	type classRange struct {
		start, end, class uint16
	}
	var ranges []classRange
	start := 0
	for i := 1; i <= len(classes); i++ {
		if i == len(classes) || classes[i] != classes[start] {
			if classes[start] != 0 {
				ranges = append(ranges, classRange{uint16(start), uint16(i - 1), classes[start]})
			}
			start = i
		}
	}
	classDef := make([]byte, 4+6*len(ranges))
	putU16(classDef, 0, 2)
	putU16(classDef, 2, uint16(len(ranges)))
	for i, r := range ranges {
		putU16(classDef, 4+6*i, r.start)
		putU16(classDef, 6+6*i, r.end)
		putU16(classDef, 8+6*i, r.class)
	}
	out := make([]byte, 12+len(classDef))
	putU32(out, 0, 0x00010000)
	putU16(out, 4, 12)
	copy(out[12:], classDef)
	return out
}
