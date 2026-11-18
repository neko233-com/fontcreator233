package fontbuild

import (
	"strconv"
	"strings"
)

// ColorLayer is one COLR layer: a glyph plus a CPAL palette index.
type ColorLayer struct {
	Glyph        string
	PaletteIndex uint16
}

// ColorGlyph describes a COLR v0 base glyph and its layers.
type ColorGlyph struct {
	Base   string
	Rune   rune
	Layers []ColorLayer
}

func accentRGB(accent string) [3]byte {
	s := strings.TrimSpace(strings.TrimPrefix(accent, "#"))
	if len(s) != 6 {
		return [3]byte{0x5B, 0x8C, 0xFF}
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return [3]byte{0x5B, 0x8C, 0xFF}
	}
	return [3]byte{byte(v >> 16), byte(v >> 8), byte(v)}
}

func buildCPAL(accent string) []byte {
	rgb := accentRGB(accent)
	// Header: version, numPaletteEntries, numPalettes, numColorRecords,
	// offsetFirstColorRecord, then one palette index.
	b := make([]byte, 14+3*4)
	putU16(b, 0, 0)  // version
	putU16(b, 2, 3)  // palette entries
	putU16(b, 4, 1)  // palettes
	putU16(b, 6, 3)  // color records
	putU32(b, 8, 14) // offset to first color record
	putU16(b, 12, 0) // palette 0 starts at record 0
	colors := [][4]byte{
		{rgb[2], rgb[1], rgb[0], 0xFF}, // accent, BGRA
		{0xFF, 0xFF, 0xFF, 0xFF},       // white
		{0x00, 0x00, 0x00, 0xFF},       // black
	}
	for i, c := range colors {
		copy(b[14+i*4:], c[:])
	}
	return b
}

func buildCOLR(glyphs []Glyph, cg *ColorGlyph) []byte {
	if cg == nil || len(cg.Layers) == 0 {
		return nil
	}
	ids := glyphIDs(glyphs)
	base, ok := ids[cg.Base]
	if !ok {
		return nil
	}
	layers := make([]struct {
		gid     uint16
		palette uint16
	}, 0, len(cg.Layers))
	for _, layer := range cg.Layers {
		gid, ok := ids[layer.Glyph]
		if !ok {
			return nil
		}
		layers = append(layers, struct {
			gid     uint16
			palette uint16
		}{gid: gid, palette: layer.PaletteIndex})
	}
	baseOffset := 14
	layerOffset := baseOffset + 6
	total := layerOffset + len(layers)*4
	b := make([]byte, total)
	putU16(b, 0, 0) // version
	putU16(b, 2, 1) // base glyph records
	putU32(b, 4, uint32(baseOffset))
	putU32(b, 8, uint32(layerOffset))
	putU16(b, 12, uint16(len(layers)))
	putU16(b, baseOffset, base)
	putU16(b, baseOffset+2, 0)
	putU16(b, baseOffset+4, uint16(len(layers)))
	for i, layer := range layers {
		putU16(b, layerOffset+i*4, layer.gid)
		putU16(b, layerOffset+i*4+2, layer.palette)
	}
	return b
}
