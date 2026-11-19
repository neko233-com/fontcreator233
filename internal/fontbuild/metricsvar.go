package fontbuild

import "math"

// varRegion is one normalized variation region on the wght axis. Coordinates
// are F2DOT14 (-1.0 = -16384, +1.0 = +16384).
type varRegion struct {
	start, peak, end int16
}

// wghtRegions mirrors the two gvar tuples: light at minimum weight, bold at
// maximum weight.
func wghtRegions() []varRegion {
	return []varRegion{
		{-16384, -16384, 0},
		{0, 16384, 16384},
	}
}

func clampI8(v int) int8 {
	if v < -127 {
		return -127
	}
	if v > 127 {
		return 127
	}
	return int8(v)
}

// buildVarStore writes a minimal ItemVariationStore with one axis, the given
// regions, and one delta per region per item. All deltas are int8
// (wordDeltaCount = 0), which is sufficient for metric adjustments.
func buildVarStore(regions []varRegion, items [][]int8) []byte {
	if len(regions) == 0 || len(items) == 0 {
		return nil
	}
	regionListOff := 12 + 4 // ItemVariationStore header + one data offset
	dataOff := regionListOff + 4 + len(regions)*6
	total := dataOff + 6 + 2*len(regions) + len(items)*len(regions)
	b := make([]byte, total)
	putU16(b, 0, 1)                     // format
	putU32(b, 2, uint32(regionListOff)) // variationRegionListOffset
	putU16(b, 6, 1)                     // itemVariationDataCount
	putU32(b, 8, uint32(dataOff))       // itemVariationDataOffsets[0]
	putU16(b, regionListOff, 1)         // axisCount
	putU16(b, regionListOff+2, uint16(len(regions)))
	for i, r := range regions {
		o := regionListOff + 4 + i*6
		putI16(b, o, r.start)
		putI16(b, o+2, r.peak)
		putI16(b, o+4, r.end)
	}
	putU16(b, dataOff, uint16(len(items)))     // itemCount
	putU16(b, dataOff+2, 0)                    // wordDeltaCount: all int8
	putU16(b, dataOff+4, uint16(len(regions))) // regionIndexCount
	for i := range regions {
		putU16(b, dataOff+6+2*i, uint16(i))
	}
	o := dataOff + 6 + 2*len(regions)
	for _, item := range items {
		for _, d := range item {
			b[o] = byte(d)
			o++
		}
	}
	return b
}

// buildHVAR writes advance-width deltas for the wght axis. A zero
// advanceWidthMapping means the delta set index is the glyph ID, so there is
// one two-value item per glyph in glyph order.
func buildHVAR(glyphs []Glyph) []byte {
	regions := wghtRegions()
	items := make([][]int8, len(glyphs))
	for i, g := range glyphs {
		if !shouldVary(g) || g.Advance <= 0 {
			items[i] = []int8{0, 0}
			continue
		}
		light := clampI8(-int(math.Round(float64(g.Advance) * 0.02)))
		bold := clampI8(int(math.Round(float64(g.Advance) * 0.04)))
		items[i] = []int8{light, bold}
	}
	store := buildVarStore(regions, items)
	if len(store) == 0 {
		return nil
	}
	const header = 20
	b := make([]byte, header+len(store))
	putU32(b, 0, 0x00010000)     // version 1.0
	putU32(b, 4, uint32(header)) // itemVariationStoreOffset
	putU32(b, 8, 0)              // advanceWidthMapping: identity
	putU32(b, 12, 0)             // lsbMapping
	putU32(b, 16, 0)             // rsbMapping
	copy(b[header:], store)
	return b
}

// buildMVAR writes font-wide metric deltas for the wght axis. Every value
// record's VarIdx points at one item in the store (outer index 0).
func buildMVAR(cfg Config) []byte {
	regions := wghtRegions()
	metrics := []struct {
		tag   string
		value float64
	}{
		{"hasc", float64(cfg.Ascender)},
		{"hdsc", float64(cfg.Descender)},
		{"hlgp", float64(cfg.LineGap)},
		{"xhgt", float64(cfg.XHeight)},
		{"cpht", float64(cfg.CapHeight)},
	}
	items := make([][]int8, len(metrics))
	for i, m := range metrics {
		items[i] = []int8{
			clampI8(-int(math.Round(m.value * 0.01))),
			clampI8(int(math.Round(m.value * 0.01))),
		}
	}
	store := buildVarStore(regions, items)
	if len(store) == 0 {
		return nil
	}
	const valueRecordSize = 8
	header := 12 // version(4) + reserved(2) + recordSize(2) + recordCount(2) + varStoreOffset(2)
	records := len(metrics) * valueRecordSize
	b := make([]byte, header+records+len(store))
	putU32(b, 0, 0x00010000)              // version 1.0
	putU16(b, 4, 0)                       // reserved
	putU16(b, 6, valueRecordSize)         // valueRecordSize
	putU16(b, 8, uint16(len(metrics)))    // valueRecordCount
	putU16(b, 10, uint16(header+records)) // varStoreOffset (Offset16)
	for i, m := range metrics {
		o := header + i*valueRecordSize
		copy(b[o:], m.tag)
		putU32(b, o+4, uint32(i)) // VarIdx: outer 0, inner i
	}
	copy(b[header+records:], store)
	return b
}
