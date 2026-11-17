package specimen

import (
	"github.com/neko233-com/fontcreator233/internal/unicodes"
	"golang.org/x/image/font/sfnt"
)

// GlyphsFromFont enumerates the cmap of a compiled font for the specimen grid.
func GlyphsFromFont(data []byte) ([]GlyphInfo, error) {
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	var buf sfnt.Buffer
	var out []GlyphInfo
	for r := rune(0); r <= 0x323AF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		gid, err := f.GlyphIndex(&buf, r)
		if err != nil {
			return nil, err
		}
		if gid != 0 {
			out = append(out, GlyphInfo{Rune: r, Script: unicodes.ScriptOf(r)})
		}
	}
	return out, nil
}
