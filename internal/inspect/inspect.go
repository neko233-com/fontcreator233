// Package inspect reads metadata and coverage from a compiled font.
package inspect

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/neko233-com/fontcreator233/internal/unicodes"
	"golang.org/x/image/font/sfnt"
)

// Info is a machine-readable font summary.
type Info struct {
	Family      string         `json:"family"`
	Style       string         `json:"style"`
	Version     string         `json:"version"`
	UnitsPerEm  int            `json:"unitsPerEm"`
	Glyphs      int            `json:"glyphs"`
	Tables      []string       `json:"tables"`
	SHA256      string         `json:"sha256"`
	Coverage    map[string]int `json:"coverage"`
	WeightClass int            `json:"weightClass"`
	ItalicAngle float64        `json:"italicAngle"`
}

// Inspect parses a TTF/OTF and summarizes it.
func Inspect(data []byte) (Info, error) {
	f, err := sfnt.Parse(data)
	if err != nil {
		return Info{}, err
	}
	info := Info{
		UnitsPerEm: int(f.UnitsPerEm()),
		Glyphs:     f.NumGlyphs(),
		Coverage:   map[string]int{},
		Tables:     tableTags(data),
	}
	sum := sha256.Sum256(data)
	info.SHA256 = hex.EncodeToString(sum[:])
	var buf sfnt.Buffer
	info.Family, _ = f.Name(&buf, sfnt.NameIDFamily)
	info.Style, _ = f.Name(&buf, sfnt.NameIDSubfamily)
	info.Version, _ = f.Name(&buf, sfnt.NameIDVersion)
	if pt := f.PostTable(); pt != nil {
		info.ItalicAngle = pt.ItalicAngle
	}
	if os2 := rawTable(data, "OS/2"); len(os2) >= 6 {
		info.WeightClass = int(binary.BigEndian.Uint16(os2[4:]))
	}
	for r := rune(0); r <= 0x323AF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		gid, err := f.GlyphIndex(&buf, r)
		if err != nil {
			return Info{}, fmt.Errorf("cmap %U: %w", r, err)
		}
		if gid != 0 {
			info.Coverage[unicodes.ScriptOf(r)]++
		}
	}
	return info, nil
}

func tableTags(data []byte) []string {
	if len(data) < 12 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	if 12+n*16 > len(data) {
		return nil
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, string(data[12+i*16:12+i*16+4]))
	}
	sort.Strings(out)
	return out
}

func rawTable(data []byte, tag string) []byte {
	if len(data) < 12 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := 0; i < n; i++ {
		base := 12 + i*16
		if base+16 > len(data) {
			return nil
		}
		if string(data[base:base+4]) != tag {
			continue
		}
		off := binary.BigEndian.Uint32(data[base+8:])
		length := binary.BigEndian.Uint32(data[base+12:])
		if int(off)+int(length) > len(data) {
			return nil
		}
		return data[off : off+length]
	}
	return nil
}
