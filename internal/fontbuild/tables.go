package fontbuild

import "math"

func buildHead(cfg Config, fontBox glyphBox) []byte {
	b := make([]byte, 54)
	putU32(b, 0, 0x00010000)
	putU32(b, 4, 0x00010000)
	putU32(b, 8, 0)
	putU32(b, 12, 0x5F0F3CF5)
	putU16(b, 16, 0x000B)
	putU16(b, 18, 1000)
	secs := uint64(cfg.Created.Unix() + 2082844800)
	putU32(b, 20, uint32(secs>>32))
	putU32(b, 24, uint32(secs))
	putU32(b, 28, uint32(secs>>32))
	putU32(b, 32, uint32(secs))
	if !fontBox.empty {
		putI16(b, 36, fontBox.xMin)
		putI16(b, 38, fontBox.yMin)
		putI16(b, 40, fontBox.xMax)
		putI16(b, 42, fontBox.yMax)
	}
	var macStyle uint16
	if cfg.WeightClass >= 700 {
		macStyle |= 1
	}
	if cfg.ItalicAngle != 0 {
		macStyle |= 2
	}
	putU16(b, 44, macStyle)
	putU16(b, 46, 8)
	putI16(b, 48, 2)
	putI16(b, 50, 1)
	putI16(b, 52, 0)
	return b
}

func buildHhea(cfg Config, boxes []glyphBox, maxAdvance int) []byte {
	b := make([]byte, 36)
	putU32(b, 0, 0x00010000)
	putI16(b, 4, int16(cfg.Ascender))
	putI16(b, 6, int16(cfg.Descender))
	putI16(b, 8, int16(cfg.LineGap))
	putU16(b, 10, uint16(maxAdvance))
	minLSB, minRSB, xMaxExtent := 0, 0, 0
	first := true
	for _, box := range boxes {
		if box.empty {
			continue
		}
		lsb := int(box.xMin)
		rsb := box.advance - int(box.xMax)
		extent := lsb + int(box.xMax-box.xMin)
		if first {
			minLSB, minRSB, xMaxExtent = lsb, rsb, extent
			first = false
		} else {
			if lsb < minLSB {
				minLSB = lsb
			}
			if rsb < minRSB {
				minRSB = rsb
			}
			if extent > xMaxExtent {
				xMaxExtent = extent
			}
		}
	}
	putI16(b, 12, int16(minLSB))
	putI16(b, 14, int16(minRSB))
	putI16(b, 16, int16(xMaxExtent))
	if cfg.ItalicAngle != 0 {
		putI16(b, 18, 1000)
		putI16(b, 20, int16(math.Round(math.Tan(-cfg.ItalicAngle*math.Pi/180)*1000)))
	} else {
		putI16(b, 18, 1)
		putI16(b, 20, 0)
	}
	putI16(b, 22, 0)
	putI16(b, 32, 0)
	putU16(b, 34, uint16(len(boxes)))
	return b
}

func buildMaxp(numGlyphs, maxPoints, maxContours int) []byte {
	b := make([]byte, 32)
	putU32(b, 0, 0x00010000)
	putU16(b, 4, uint16(numGlyphs))
	putU16(b, 6, uint16(maxPoints))
	putU16(b, 8, uint16(maxContours))
	putU16(b, 10, 0)
	putU16(b, 12, 0)
	putU16(b, 14, 1)
	putU16(b, 16, 0)
	putU16(b, 18, 0)
	putU16(b, 20, 0)
	putU16(b, 22, 0)
	putU16(b, 24, 0)
	putU16(b, 26, 0)
	putU16(b, 28, 0)
	putU16(b, 30, 0)
	return b
}

func buildHmtx(boxes []glyphBox) []byte {
	b := make([]byte, 4*len(boxes))
	for i, box := range boxes {
		putU16(b, i*4, uint16(box.advance))
		putI16(b, i*4+2, int16(box.lsb))
	}
	return b
}

func buildPost(cfg Config) []byte {
	b := make([]byte, 32)
	putU32(b, 0, 0x00030000)
	angle := int32(math.Round(-cfg.ItalicAngle * 65536))
	putU32(b, 4, uint32(angle))
	putI16(b, 8, -100)
	putI16(b, 10, 50)
	if cfg.FixedPitch {
		putU32(b, 12, 1)
	}
	return b
}
