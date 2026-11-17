package fontbuild

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
)

// BuildWOFF wraps a TrueType font as WOFF 1.0.
func BuildWOFF(ttf []byte) ([]byte, error) {
	if len(ttf) < 12 {
		return nil, fmt.Errorf("woff: short font")
	}
	flavor := binary.BigEndian.Uint32(ttf[0:])
	if flavor != 0x00010000 && string(ttf[:4]) != "true" && string(ttf[:4]) != "typ1" {
		return nil, fmt.Errorf("woff: unsupported flavor %#x", flavor)
	}
	numTables := int(binary.BigEndian.Uint16(ttf[4:]))
	type entry struct {
		tag        [4]byte
		checksum   uint32
		offset     uint32
		length     uint32
		comp       []byte
		compLength uint32
	}
	entries := make([]entry, 0, numTables)
	for i := 0; i < numTables; i++ {
		base := 12 + i*16
		if base+16 > len(ttf) {
			return nil, fmt.Errorf("woff: truncated directory")
		}
		var e entry
		copy(e.tag[:], ttf[base:base+4])
		e.checksum = binary.BigEndian.Uint32(ttf[base+4:])
		e.offset = binary.BigEndian.Uint32(ttf[base+8:])
		e.length = binary.BigEndian.Uint32(ttf[base+12:])
		if int(e.offset)+int(e.length) > len(ttf) {
			return nil, fmt.Errorf("woff: table %s out of range", string(e.tag[:]))
		}
		raw := ttf[e.offset : e.offset+e.length]
		var zbuf bytes.Buffer
		zw := zlib.NewWriter(&zbuf)
		if _, err := zw.Write(raw); err != nil {
			return nil, err
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}
		if zbuf.Len() < len(raw) {
			e.comp = zbuf.Bytes()
			e.compLength = uint32(zbuf.Len())
		} else {
			e.comp = append([]byte(nil), raw...)
			e.compLength = e.length
		}
		entries = append(entries, e)
	}
	headerLen := 44 + 20*numTables
	dataOffset := headerLen
	offsets := make([]uint32, len(entries))
	total := headerLen
	for i, e := range entries {
		offsets[i] = uint32(dataOffset)
		dataOffset += int(e.compLength)
		for dataOffset%4 != 0 {
			dataOffset++
		}
		total = dataOffset
	}
	out := make([]byte, total)
	copy(out[:4], "wOFF")
	binary.BigEndian.PutUint32(out[4:], flavor)
	binary.BigEndian.PutUint32(out[8:], uint32(total))
	binary.BigEndian.PutUint16(out[12:], uint16(numTables))
	binary.BigEndian.PutUint16(out[14:], 0)
	binary.BigEndian.PutUint32(out[16:], uint32(len(ttf)))
	binary.BigEndian.PutUint16(out[20:], 1)
	binary.BigEndian.PutUint16(out[22:], 0)
	for i, e := range entries {
		base := 44 + i*20
		copy(out[base:base+4], e.tag[:])
		binary.BigEndian.PutUint32(out[base+4:], offsets[i])
		binary.BigEndian.PutUint32(out[base+8:], e.compLength)
		binary.BigEndian.PutUint32(out[base+12:], e.length)
		binary.BigEndian.PutUint32(out[base+16:], e.checksum)
		copy(out[offsets[i]:], e.comp)
	}
	return out, nil
}
