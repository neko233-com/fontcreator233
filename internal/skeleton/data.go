package skeleton

import (
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

var glyphs = map[rune]*Def{}

func add(def Def) {
	d := def
	glyphs[d.Rune] = &d
}

func addVariant(base rune, key string, def Def) {
	g := glyphs[base]
	if g == nil {
		panic(fmt.Sprintf("variant for missing glyph %U", base))
	}
	if g.Variants == nil {
		g.Variants = map[string]*Def{}
	}
	d := def
	g.Variants[key] = &d
}

func mustParse(d string) []geom.Cmd {
	cmds, err := geom.ParseCommands(d)
	if err != nil {
		panic(err)
	}
	return cmds
}

func s(d string) Stroke {
	return Stroke{Cmds: mustParse(d), Width: 1}
}

func sw(w float64, d string) Stroke {
	return Stroke{Cmds: mustParse(d), Width: w}
}

func bar(x0, x1, y float64) Stroke {
	st := sw(0.8, fmt.Sprintf("M %g %g L %g %g", x0, y, x1, y))
	st.CapStart = "flat"
	st.CapEnd = "flat"
	return st
}

func vstem(x, y0, y1 float64) Stroke {
	st := sw(1, fmt.Sprintf("M %g %g L %g %g", x, y0, x, y1))
	st.SerifStart = true
	st.SerifEnd = true
	return st
}

func vstemPlain(x, y0, y1 float64) Stroke {
	return sw(1, fmt.Sprintf("M %g %g L %g %g", x, y0, x, y1))
}

func ellipsePath(cx, cy, rx, ry float64) string {
	const k = 0.5522847498307936
	kx, ky := rx*k, ry*k
	return fmt.Sprintf(
		"M %g %g C %g %g %g %g %g %g C %g %g %g %g %g %g C %g %g %g %g %g %g C %g %g %g %g %g %g Z",
		cx+rx, cy,
		cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry,
		cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy,
		cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry,
		cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy,
	)
}

func ellipse(cx, cy, rx, ry float64) Stroke {
	st := sw(1, ellipsePath(cx, cy, rx, ry))
	st.Closed = true
	return st
}

func dot(x, y, r float64) Dot { return Dot{X: x, Y: y, R: r} }
