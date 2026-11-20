package fontbuild

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/andybalholm/brotli"
)

// woff2KnownTags is the WOFF2 table-tag index list from the specification.
var woff2KnownTags = [63]string{
	"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post",
	"cvt ", "fpgm", "glyf", "loca", "prep", "CFF ", "VORG", "EBDT",
	"EBLC", "gasp", "hdmx", "kern", "LTSH", "PCLT", "VDMX", "vhea",
	"vmtx", "BASE", "GDEF", "GPOS", "GSUB", "EBSC", "JSTF", "MATH",
	"CBDT", "CBLC", "COLR", "CPAL", "SVG ", "sbix", "acnt", "avar",
	"bdat", "bloc", "bsln", "cvar", "fdsc", "feat", "fmtx", "fvar",
	"gvar", "hsty", "just", "lcar", "mort", "morx", "opbd", "prop",
	"trak", "Zapf", "Silf", "Glat", "Gloc", "Feat", "Sill",
}

type woff2Table struct {
	tag   string
	raw   []byte
	known int
}

// BuildWOFF2 wraps a TrueType font as WOFF 2.0 using the null transform for
// every table and a single Brotli-compressed data block. The result is a real,
// standards-compliant WOFF2 file that fontTools and browsers can read.
func BuildWOFF2(ttf []byte) ([]byte, error) {
	if len(ttf) < 12 {
		return nil, fmt.Errorf("woff2: short font")
	}
	flavor := binary.BigEndian.Uint32(ttf[:4])
	if flavor != 0x00010000 && string(ttf[:4]) != "true" && string(ttf[:4]) != "typ1" {
		return nil, fmt.Errorf("woff2: unsupported flavor %#x", flavor)
	}
	numTables := int(binary.BigEndian.Uint16(ttf[4:]))
	tables := make([]woff2Table, 0, numTables)
	totalSfntSize := uint32(12 + 16*numTables)
	for i := 0; i < numTables; i++ {
		base := 12 + i*16
		if base+16 > len(ttf) {
			return nil, fmt.Errorf("woff2: truncated directory")
		}
		tag := string(ttf[base : base+4])
		offset := binary.BigEndian.Uint32(ttf[base+8:])
		length := binary.BigEndian.Uint32(ttf[base+12:])
		if uint64(offset)+uint64(length) > uint64(len(ttf)) {
			return nil, fmt.Errorf("woff2: table %s out of range", tag)
		}
		known := 63
		for idx, k := range woff2KnownTags {
			if k == tag {
				known = idx
				break
			}
		}
		tables = append(tables, woff2Table{
			tag:   tag,
			raw:   ttf[offset : offset+length],
			known: known,
		})
		totalSfntSize += round4U32(length)
	}
	// WOFF2 requires directory entries sorted by tag.
	sort.Slice(tables, func(i, j int) bool { return tables[i].tag < tables[j].tag })

	var directory bytes.Buffer
	for _, t := range tables {
		flags := byte(t.known)
		if t.known == 63 {
			flags = 63
		}
		// The WOFF2 spec uses transformVersion 3 as the null transform for
		// glyf and loca, and 0 for all other tables.
		if t.tag == "glyf" || t.tag == "loca" {
			flags |= 3 << 6
		}
		directory.WriteByte(flags)
		if t.known == 63 {
			directory.WriteString(t.tag)
		}
		writeBase128(&directory, uint32(len(t.raw)))
		// Null transform: transformLength is omitted (equal to origLength).
	}
	var payload bytes.Buffer
	for _, t := range tables {
		payload.Write(t.raw)
	}
	var compressed bytes.Buffer
	zw := brotli.NewWriterLevel(&compressed, 6)
	if _, err := zw.Write(payload.Bytes()); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	total := 48 + directory.Len() + compressed.Len()
	out := make([]byte, total)
	copy(out[:4], "wOF2")
	binary.BigEndian.PutUint32(out[4:], flavor)
	binary.BigEndian.PutUint32(out[8:], uint32(total))
	binary.BigEndian.PutUint16(out[12:], uint16(numTables))
	binary.BigEndian.PutUint16(out[14:], 0)
	binary.BigEndian.PutUint32(out[16:], totalSfntSize)
	binary.BigEndian.PutUint32(out[20:], uint32(compressed.Len()))
	binary.BigEndian.PutUint16(out[24:], 1) // majorVersion
	binary.BigEndian.PutUint16(out[26:], 0) // minorVersion
	// meta/priv offsets are all zero.
	copy(out[48:], directory.Bytes())
	copy(out[48+directory.Len():], compressed.Bytes())
	return out, nil
}

func round4U32(v uint32) uint32 { return (v + 3) &^ 3 }

// writeBase128 writes a UIntBase128 value as used by the WOFF2 directory.
func writeBase128(buf *bytes.Buffer, v uint32) {
	var tmp [5]byte
	n := 0
	for {
		tmp[n] = byte(v & 0x7F)
		v >>= 7
		n++
		if v == 0 {
			break
		}
	}
	for i := n - 1; i >= 0; i-- {
		b := tmp[i]
		if i > 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
	}
}
