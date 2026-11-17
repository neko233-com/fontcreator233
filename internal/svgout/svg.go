// Package svgout writes font-unit SVG outlines for glyph libraries and import.
package svgout

import (
	"fmt"
	"io"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

// PathData converts a font-unit outline to an SVG path. The coordinate system
// is y-up with baseline 0; consumers should use viewBox="0 -250 1000 1250".
func PathData(p geom.Path) string {
	var b strings.Builder
	for _, c := range p {
		if len(c) == 0 {
			continue
		}
		fmt.Fprintf(&b, "M %g %g", c[0].P.X, c[0].P.Y)
		for i := 1; i < len(c); i++ {
			n := c[i]
			if n.On {
				fmt.Fprintf(&b, " L %g %g", n.P.X, n.P.Y)
				continue
			}
			if i+1 < len(c) && c[i+1].On {
				fmt.Fprintf(&b, " Q %g %g %g %g", n.P.X, n.P.Y, c[i+1].P.X, c[i+1].P.Y)
				i++
				continue
			}
			mid := n.P.Lerp(c[i-1].P, 0.5)
			fmt.Fprintf(&b, " Q %g %g %g %g", n.P.X, n.P.Y, mid.X, mid.Y)
		}
		b.WriteString(" Z")
	}
	return b.String()
}

// WriteGlyph writes one glyph SVG with data-advance.
func WriteGlyph(w io.Writer, title string, advance int, p geom.Path) error {
	_, err := fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -250 1000 1250" width="1000" height="1250" data-advance="%d" role="img" aria-label="%s">
  <title>%s</title>
  <path d="%s" fill="currentColor"/>
</svg>
`, advance, escape(title), escape(title), PathData(p))
	return err
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}
