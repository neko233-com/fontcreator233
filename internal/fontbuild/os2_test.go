package fontbuild

import (
	"encoding/binary"
	"testing"
)

func TestOS2UnicodeRangeBits(t *testing.T) {
	cfg := testConfig()
	cfg.Glyphs = append(cfg.Glyphs,
		Glyph{Rune: '한', Advance: 1000},
		Glyph{Rune: 'あ', Advance: 1000},
		Glyph{Rune: '“', Advance: 500},
		Glyph{Rune: 0x3400, Advance: 1000},
		Glyph{Rune: 0x20000, Advance: 1000},
		Glyph{Rune: 0xE000, Advance: 600},
	)
	data, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	os2 := tableBytes(t, data, "OS/2")
	has := func(r rune, bit int) bool {
		t.Helper()
		off := 42 + (bit/32)*4
		word := binary.BigEndian.Uint32(os2[off:])
		return word&(1<<uint(bit%32)) != 0
	}
	if !has('中', 59) {
		t.Fatal("CJK Unified Ideographs must set bit 59")
	}
	if !has('한', 56) {
		t.Fatal("Hangul Syllables must set bit 56")
	}
	if !has('あ', 49) {
		t.Fatal("Hiragana must set bit 49")
	}
	if !has('“', 31) {
		t.Fatal("General Punctuation must set bit 31")
	}
	if !has(0x3400, 59) || !has(0x20000, 59) {
		t.Fatal("CJK Ext A/B must set bit 59")
	}
	if !has(0xE000, 60) {
		t.Fatal("Private Use Area must set bit 60")
	}
}
