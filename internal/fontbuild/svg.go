package fontbuild

import (
	"fmt"
	"sort"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/svgout"
)

// SVGColorLayer is one visual layer of an SVG color glyph.
type SVGColorLayer struct {
	Path  geom.Path
	Color [3]byte // RGB
}

// SVGColorGlyph is one base glyph with its SVG layers.
type SVGColorGlyph struct {
	Base   string
	Layers []SVGColorLayer
}

// buildSVG writes an OpenType SVG table (version 0). Each color glyph maps to
// one uncompressed standalone SVG document. No external references are used.
func buildSVG(glyphs []Glyph, entries []SVGColorGlyph) []byte {
	ids := glyphIDs(glyphs)
	type doc struct {
		gid uint16
		xml []byte
	}
	var docs []doc
	for _, e := range entries {
		gid, ok := ids[e.Base]
		if !ok || len(e.Layers) == 0 {
			continue
		}
		var b strings.Builder
		b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -250 1000 1250" width="1000" height="1250">`)
		for _, l := range e.Layers {
			fmt.Fprintf(&b, `<path d="%s" fill="#%02X%02X%02X"/>`, svgout.PathData(l.Path), l.Color[0], l.Color[1], l.Color[2])
		}
		b.WriteString(`</svg>`)
		docs = append(docs, doc{gid: gid, xml: []byte(b.String())})
	}
	if len(docs) == 0 {
		return nil
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].gid < docs[j].gid })

	const header = 6 // version(2) + svgDocumentListOffset(4)
	listOff := header
	recordsOff := listOff + 2
	dataOff := recordsOff + 10*len(docs)
	total := dataOff
	offsets := make([]int, len(docs))
	for i, d := range docs {
		offsets[i] = total - listOff // offsets are relative to the document list
		total += len(d.xml)
	}
	b := make([]byte, total)
	putU16(b, 0, 0)                       // version 0
	putU32(b, 2, uint32(listOff))         // SVGDocumentListOffset
	putU16(b, listOff, uint16(len(docs))) // numSVGDocuments
	for i, d := range docs {
		o := recordsOff + 10*i
		putU16(b, o, d.gid)   // startGlyphID
		putU16(b, o+2, d.gid) // endGlyphID (one glyph per doc)
		putU32(b, o+4, uint32(offsets[i]))
		putU32(b, o+8, uint32(len(d.xml)))
		copy(b[dataOff:], d.xml)
		dataOff += len(d.xml)
	}
	return b
}
