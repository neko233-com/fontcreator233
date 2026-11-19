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

// ColorSet is a generalized COLR/CPAL payload. Palettes all have the same
// number of color entries. This is the recipe-driven, multi-glyph form.
type ColorSet struct {
	Glyphs   []ColorGlyph
	Palettes [][][4]byte // BGRA color records, one slice per palette
}

func accentRGB(accent string) [3]byte {
	s := strings.TrimSpace(strings.TrimPrefix(accent, "#"))
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return [3]byte{0x5B, 0x8C, 0xFF}
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return [3]byte{0x5B, 0x8C, 0xFF}
	}
	return [3]byte{byte(v >> 16), byte(v >> 8), byte(v)}
}

func colorRecord(c string) [4]byte {
	rgb := accentRGB(c)
	return [4]byte{rgb[2], rgb[1], rgb[0], 0xFF} // BGRA
}

// ParseColorRecord converts an #RGB/#RRGGBB string to a CPAL BGRA record.
func ParseColorRecord(c string) [4]byte { return colorRecord(c) }

// buildCPAL keeps the single-accent demo palette: accent, white, black.
func buildCPAL(accent string) []byte {
	rgb := accentRGB(accent)
	return buildCPALMulti([][][4]byte{{
		{rgb[2], rgb[1], rgb[0], 0xFF},
		{0xFF, 0xFF, 0xFF, 0xFF},
		{0x00, 0x00, 0x00, 0xFF},
	}})
}

// buildCPALMulti writes a real CPAL v0 table with multiple palettes. Each
// palette must have the same number of entries; color records are concatenated
// and each palette's colorRecordIndices points at its slice.
func buildCPALMulti(palettes [][][4]byte) []byte {
	if len(palettes) == 0 || len(palettes[0]) == 0 {
		return nil
	}
	entries := len(palettes[0])
	for _, p := range palettes {
		if len(p) != entries {
			return nil
		}
	}
	records := 0
	for _, p := range palettes {
		records += len(p)
	}
	recordOffset := 12 + 2*len(palettes)
	b := make([]byte, recordOffset+records*4)
	putU16(b, 0, 0)                     // version
	putU16(b, 2, uint16(entries))       // numPaletteEntries
	putU16(b, 4, uint16(len(palettes))) // numPalettes
	putU16(b, 6, uint16(records))       // numColorRecords
	putU32(b, 8, uint32(recordOffset))  // offsetFirstColorRecord
	idx := 0
	for p := range palettes {
		putU16(b, 12+2*p, uint16(idx)) // colorRecordIndices[p]
		idx += len(palettes[p])
	}
	off := recordOffset
	for _, p := range palettes {
		for _, c := range p {
			copy(b[off:], c[:])
			off += 4
		}
	}
	return b
}

// buildCOLR keeps the single-glyph demo path.
func buildCOLR(glyphs []Glyph, cg *ColorGlyph) []byte {
	if cg == nil {
		return nil
	}
	return buildCOLRMulti(glyphs, []ColorGlyph{*cg})
}

// buildCOLRMulti writes a COLR v0 table with multiple base glyphs. Layer
// records are laid out consecutively; each base glyph record stores its first
// layer index and layer count.
func buildCOLRMulti(glyphs []Glyph, cgs []ColorGlyph) []byte {
	ids := glyphIDs(glyphs)
	type layerRec struct {
		gid, palette uint16
	}
	type baseRec struct {
		gid, first, count uint16
	}
	var layers []layerRec
	var bases []baseRec
	for _, cg := range cgs {
		base, ok := ids[cg.Base]
		if !ok || len(cg.Layers) == 0 {
			return nil
		}
		first := uint16(len(layers))
		for _, layer := range cg.Layers {
			gid, ok := ids[layer.Glyph]
			if !ok {
				return nil
			}
			layers = append(layers, layerRec{gid: gid, palette: layer.PaletteIndex})
		}
		bases = append(bases, baseRec{gid: base, first: first, count: uint16(len(cg.Layers))})
	}
	if len(bases) == 0 {
		return nil
	}
	baseOffset := 14
	layerOffset := baseOffset + len(bases)*6
	b := make([]byte, layerOffset+len(layers)*4)
	putU16(b, 0, 0)                    // version
	putU16(b, 2, uint16(len(bases)))   // numBaseGlyphRecords
	putU32(b, 4, uint32(baseOffset))   // baseGlyphRecordsOffset
	putU32(b, 8, uint32(layerOffset))  // layerRecordsOffset
	putU16(b, 12, uint16(len(layers))) // numLayerRecords
	for i, br := range bases {
		o := baseOffset + i*6
		putU16(b, o, br.gid)
		putU16(b, o+2, br.first)
		putU16(b, o+4, br.count)
	}
	for i, lr := range layers {
		o := layerOffset + i*4
		putU16(b, o, lr.gid)
		putU16(b, o+2, lr.palette)
	}
	return b
}
