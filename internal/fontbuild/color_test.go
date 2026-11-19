package fontbuild

import (
	"encoding/binary"
	"testing"
)

func TestBuildCPALMulti(t *testing.T) {
	red := [4]byte{0x00, 0x00, 0xFF, 0xFF} // BGRA
	blue := [4]byte{0xFF, 0x00, 0x00, 0xFF}
	white := [4]byte{0xFF, 0xFF, 0xFF, 0xFF}
	b := buildCPALMulti([][][4]byte{
		{red, white},
		{blue, white},
	})
	if len(b) < 16 {
		t.Fatalf("CPAL too short: %d", len(b))
	}
	if got := binary.BigEndian.Uint16(b[2:]); got != 2 {
		t.Fatalf("numPaletteEntries = %d, want 2", got)
	}
	if got := binary.BigEndian.Uint16(b[4:]); got != 2 {
		t.Fatalf("numPalettes = %d, want 2", got)
	}
	if got := binary.BigEndian.Uint16(b[6:]); got != 4 {
		t.Fatalf("numColorRecords = %d, want 4", got)
	}
	recordOffset := int(binary.BigEndian.Uint32(b[8:]))
	if recordOffset != 16 {
		t.Fatalf("record offset = %d, want 16", recordOffset)
	}
	if got := binary.BigEndian.Uint16(b[12:]); got != 0 {
		t.Fatalf("palette 0 index = %d, want 0", got)
	}
	if got := binary.BigEndian.Uint16(b[14:]); got != 2 {
		t.Fatalf("palette 1 index = %d, want 2", got)
	}
	if b[recordOffset] != red[0] || b[recordOffset+4] != blue[0] {
		t.Fatalf("color records out of order: %v", b[recordOffset:recordOffset+8])
	}
	// Mismatched palette lengths must be rejected.
	if got := buildCPALMulti([][][4]byte{{red}, {blue, white}}); got != nil {
		t.Fatalf("mismatched palettes should fail, got %d bytes", len(got))
	}
}

func TestBuildCOLRMulti(t *testing.T) {
	glyphs := []Glyph{
		{Rune: 'A', Name: "A", Advance: 500, Path: nil},
		{Name: "b0", NoCmap: true, Advance: 500, Path: nil},
		{Name: "b1", NoCmap: true, Advance: 500, Path: nil},
		{Name: "c0", NoCmap: true, Advance: 500, Path: nil},
		{Name: "c1", NoCmap: true, Advance: 500, Path: nil},
		{Rune: 'B', Name: "B", Advance: 500, Path: nil},
	}
	b := buildCOLRMulti(glyphs, []ColorGlyph{
		{Base: "A", Layers: []ColorLayer{{Glyph: "b0", PaletteIndex: 0}, {Glyph: "b1", PaletteIndex: 1}}},
		{Base: "B", Layers: []ColorLayer{{Glyph: "c0", PaletteIndex: 1}, {Glyph: "c1", PaletteIndex: 0}}},
	})
	if len(b) == 0 {
		t.Fatal("COLR missing")
	}
	if got := binary.BigEndian.Uint16(b[2:]); got != 2 {
		t.Fatalf("numBaseGlyphRecords = %d, want 2", got)
	}
	if got := binary.BigEndian.Uint16(b[12:]); got != 4 {
		t.Fatalf("numLayerRecords = %d, want 4", got)
	}
	baseOffset := int(binary.BigEndian.Uint32(b[4:]))
	layerOffset := int(binary.BigEndian.Uint32(b[8:]))
	// First base: firstLayer=0, count=2. Second base: firstLayer=2, count=2.
	if got := binary.BigEndian.Uint16(b[baseOffset+2:]); got != 0 {
		t.Fatalf("base 0 first layer = %d", got)
	}
	if got := binary.BigEndian.Uint16(b[baseOffset+8:]); got != 2 {
		t.Fatalf("base 1 first layer = %d", got)
	}
	if got := binary.BigEndian.Uint16(b[layerOffset+2:]); got != 0 {
		t.Fatalf("layer 0 palette = %d", got)
	}
	if got := binary.BigEndian.Uint16(b[layerOffset+4*3+2:]); got != 0 {
		t.Fatalf("layer 3 palette = %d", got)
	}
}
