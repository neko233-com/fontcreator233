package fontbuild

import (
	"fmt"
	"sort"
)

type cmapPair struct {
	r   uint32
	gid uint16
}

type segment struct {
	start, end uint16
	delta      uint16
}

type group12 struct {
	start, end uint32
	gid        uint32
}

func buildCmap(glyphs []Glyph) ([]byte, error) {
	pairs := make([]cmapPair, 0, len(glyphs))
	for i, g := range glyphs {
		if g.Rune <= 0 || g.Rune > 0x10FFFF {
			continue
		}
		pairs = append(pairs, cmapPair{r: uint32(g.Rune), gid: uint16(i)})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].r < pairs[j].r })

	format4, err := buildFormat4(pairs)
	if err != nil {
		return nil, err
	}
	format12 := buildFormat12(pairs)
	const headerLen = 4 + 8*4
	off4 := headerLen
	off12 := off4 + len(format4)
	for off12%4 != 0 {
		format12 = append([]byte{0}, format12...)
		off12++
	}
	total := off12 + len(format12)
	out := make([]byte, total)
	putU16(out, 0, 0)
	putU16(out, 2, 4)
	records := []struct {
		platform, encoding uint16
		offset             int
	}{
		{0, 3, off4},
		{0, 4, off12},
		{3, 1, off4},
		{3, 10, off12},
	}
	for i, rec := range records {
		base := 4 + i*8
		putU16(out, base, rec.platform)
		putU16(out, base+2, rec.encoding)
		putU32(out, base+4, uint32(rec.offset))
	}
	copy(out[off4:], format4)
	copy(out[off12:], format12)
	return out, nil
}

func buildFormat4(pairs []cmapPair) ([]byte, error) {
	var bmp []cmapPair
	for _, p := range pairs {
		if p.r <= 0xFFFF {
			bmp = append(bmp, p)
		}
	}
	var segs []segment
	for i := 0; i < len(bmp); {
		j := i + 1
		for j < len(bmp) && bmp[j].r == bmp[j-1].r+1 && bmp[j].gid == bmp[j-1].gid+1 {
			j++
		}
		segs = append(segs, segment{
			start: uint16(bmp[i].r),
			end:   uint16(bmp[j-1].r),
			delta: uint16(bmp[i].gid - uint16(bmp[i].r)),
		})
		i = j
	}
	segs = append(segs, segment{start: 0xFFFF, end: 0xFFFF, delta: 1})
	segCount := len(segs)
	length := 16 + 8*segCount
	if length > 0xFFFF {
		return nil, fmt.Errorf("cmap format 4 needs %d segments (max 8189); use fewer non-contiguous BMP codepoints", segCount)
	}
	out := make([]byte, length)
	putU16(out, 0, 4)
	putU16(out, 2, uint16(length))
	putU16(out, 4, 0)
	putU16(out, 6, uint16(segCount*2))
	searchRange, entrySelector := 2, 0
	for (1 << (entrySelector + 1)) <= segCount {
		entrySelector++
		searchRange *= 2
	}
	putU16(out, 8, uint16(searchRange))
	putU16(out, 10, uint16(entrySelector))
	putU16(out, 12, uint16(segCount*2-searchRange))
	endOff := 14
	startOff := endOff + segCount*2 + 2
	deltaOff := startOff + segCount*2
	rangeOff := deltaOff + segCount*2
	for i, s := range segs {
		putU16(out, endOff+i*2, s.end)
		putU16(out, startOff+i*2, s.start)
		putU16(out, deltaOff+i*2, s.delta)
		putU16(out, rangeOff+i*2, 0)
	}
	return out, nil
}

func buildFormat12(pairs []cmapPair) []byte {
	var groups []group12
	for i := 0; i < len(pairs); {
		j := i + 1
		for j < len(pairs) && pairs[j].r == pairs[j-1].r+1 && pairs[j].gid == pairs[j-1].gid+1 {
			j++
		}
		groups = append(groups, group12{
			start: pairs[i].r,
			end:   pairs[j-1].r,
			gid:   uint32(pairs[i].gid),
		})
		i = j
	}
	length := 16 + 12*len(groups)
	out := make([]byte, length)
	putU16(out, 0, 12)
	putU16(out, 2, 0)
	putU32(out, 4, uint32(length))
	putU32(out, 8, 0)
	putU32(out, 12, uint32(len(groups)))
	for i, g := range groups {
		base := 16 + i*12
		putU32(out, base, g.start)
		putU32(out, base+4, g.end)
		putU32(out, base+8, g.gid)
	}
	return out
}
