package fontbuild

func buildOS2(cfg Config, glyphs []Glyph, fontBox glyphBox) []byte {
	b := make([]byte, 96)
	putU16(b, 0, 4)
	avg, n := 0, 0
	for _, g := range glyphs {
		if g.Rune == 0 {
			continue
		}
		avg += g.Advance
		n++
	}
	if n > 0 {
		putI16(b, 2, int16(avg/n))
	}
	putU16(b, 4, uint16(cfg.WeightClass))
	putU16(b, 6, uint16(cfg.WidthClass))
	putU16(b, 8, 0) // fsType: installable embedding
	putI16(b, 10, 650)
	putI16(b, 12, 600)
	putI16(b, 14, 0)
	putI16(b, 16, 75)
	putI16(b, 18, 650)
	putI16(b, 20, 600)
	putI16(b, 22, 0)
	putI16(b, 24, 350)
	putI16(b, 26, 50)
	putI16(b, 28, 250)
	putI16(b, 30, 0) // sFamilyClass
	panose := []byte{2, 0, 5, 5, 0, 0, 0, 0, 0, 0}
	copy(b[32:42], panose)

	unicodeBits := unicodeRangeBits(glyphs)
	putU32(b, 42, unicodeBits[0])
	putU32(b, 46, unicodeBits[1])
	putU32(b, 50, unicodeBits[2])
	putU32(b, 54, unicodeBits[3])
	copy(b[58:62], []byte("FC23"))
	var fsSelection uint16
	if cfg.ItalicAngle != 0 {
		fsSelection |= 1
	}
	if cfg.WeightClass >= 700 {
		fsSelection |= 1 << 5
	}
	if fsSelection == 0 {
		fsSelection |= 1 << 6
	}
	fsSelection |= 1 << 7 // use typo metrics
	putU16(b, 62, fsSelection)

	first, last := -1, -1
	for _, g := range glyphs {
		if g.Rune <= 0 || g.Rune > 0x10FFFF {
			continue
		}
		if g.Rune <= 0xFFFF {
			if first < 0 || int(g.Rune) < first {
				first = int(g.Rune)
			}
			if int(g.Rune) > last {
				last = int(g.Rune)
			}
		} else {
			last = 0xFFFF
		}
	}
	if first < 0 {
		first = 0xFFFF
	}
	if last < 0 {
		last = 0xFFFF
	}
	putU16(b, 64, uint16(first))
	putU16(b, 66, uint16(last))
	putI16(b, 68, int16(cfg.Ascender))
	putI16(b, 70, int16(cfg.Descender))
	putI16(b, 72, int16(cfg.LineGap))
	winAscent := cfg.Ascender
	winDescent := -cfg.Descender
	if !fontBox.empty && int(fontBox.yMax) > winAscent {
		winAscent = int(fontBox.yMax)
	}
	if !fontBox.empty && -int(fontBox.yMin) > winDescent {
		winDescent = -int(fontBox.yMin)
	}
	putU16(b, 74, uint16(winAscent))
	putU16(b, 76, uint16(winDescent))
	putU32(b, 78, codePageRange(glyphs))
	putU32(b, 82, 0)
	putI16(b, 86, int16(cfg.XHeight))
	putI16(b, 88, int16(cfg.CapHeight))
	putU16(b, 90, 0)
	putU16(b, 92, 32)
	putU16(b, 94, 0)
	return b
}

func unicodeRangeBits(glyphs []Glyph) [4]uint32 {
	var bits [4]uint32
	set := func(bit int) {
		if bit >= 0 && bit < 128 {
			bits[bit/32] |= 1 << (uint(bit) % 32)
		}
	}
	for _, g := range glyphs {
		r := g.Rune
		switch {
		case r <= 0x007F:
			set(0)
		case r <= 0x00FF:
			set(1)
		case r <= 0x017F:
			set(2)
		case r <= 0x024F:
			set(3)
		case r >= 0x0250 && r <= 0x02AF:
			set(4)
		case r >= 0x02B0 && r <= 0x02FF:
			set(5)
		case r >= 0x0300 && r <= 0x036F:
			set(6)
		case r >= 0x0370 && r <= 0x03FF:
			set(7)
		case r >= 0x0400 && r <= 0x04FF:
			set(9)
		case r >= 0x1E00 && r <= 0x1EFF:
			set(22)
		case r >= 0x1F00 && r <= 0x1FFF:
			set(23)
		case r >= 0x2000 && r <= 0x206F:
			set(24)
		case r >= 0x20A0 && r <= 0x20CF:
			set(26)
		case r >= 0x2190 && r <= 0x21FF:
			set(30)
		case r >= 0x2200 && r <= 0x22FF:
			set(31)
		case r >= 0x2500 && r <= 0x257F:
			set(36)
		case r >= 0x25A0 && r <= 0x25FF:
			set(38)
		case r >= 0x2600 && r <= 0x26FF:
			set(39)
		case r >= 0x3000 && r <= 0x303F:
			set(41)
		case r >= 0x3040 && r <= 0x309F:
			set(42)
		case r >= 0x30A0 && r <= 0x30FF:
			set(43)
		case r >= 0x3130 && r <= 0x318F:
			set(45)
		case r >= 0x4E00 && r <= 0x9FFF:
			set(53)
		case r >= 0xAC00 && r <= 0xD7AF:
			set(50)
		case r >= 0xFE30 && r <= 0xFE4F:
			set(59)
		case r >= 0xFF00 && r <= 0xFFEF:
			set(62)
		case r >= 0xF900 && r <= 0xFAFF:
			set(54)
		}
	}
	return bits
}

func codePageRange(glyphs []Glyph) uint32 {
	var bits uint32
	set := func(bit int) { bits |= 1 << uint(bit) }
	for _, g := range glyphs {
		r := g.Rune
		switch {
		case r <= 0x00FF:
			set(0)
		case r >= 0x0100 && r <= 0x024F:
			set(1)
		case r >= 0x0400 && r <= 0x04FF:
			set(2)
		case r >= 0x0370 && r <= 0x03FF:
			set(3)
		case r >= 0x1EA0 && r <= 0x1EF9:
			set(8)
		case r >= 0x3040 && r <= 0x30FF:
			set(17)
		case r >= 0x4E00 && r <= 0x9FFF:
			set(18)
		case r >= 0xAC00 && r <= 0xD7AF:
			set(19)
		case r >= 0x3100 && r <= 0x312F:
			set(20)
		}
	}
	return bits
}
