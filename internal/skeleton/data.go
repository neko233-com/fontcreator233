package skeleton

import (
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

var glyphs = map[rune]*Def{}

func init() {
	// Order matters: scripts reuse Latin definitions and marks must exist
	// before composition runs.
	initLatin()
	initGreek()
	initCyrillic()
	initMarks()
}

func add(def Def) {
	d := def
	glyphs[d.Rune] = &d
}

func reuse(target, base rune, advance float64, zone Zone) {
	src := glyphs[base]
	if src == nil {
		panic(fmt.Sprintf("reuse source %U missing for %U", base, target))
	}
	d := *src
	d.Rune = target
	d.Advance = advance
	d.Zone = zone
	add(d)
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

// hook is a curved terminal stroke: it ends with a flat cut (a horizontal or
// vertical edge, depending on the exit tangent) and thins slightly toward the
// tip so the cut does not read as a ball terminal.
func hook(w float64, d string) Stroke {
	st := sw(w, d)
	st.CapEnd = "flat"
	st.TaperEnd = 0.12
	return st
}

// flatEnd is a stroke that ends with a flat cut and keeps its full width.
func flatEnd(w float64, d string) Stroke {
	st := sw(w, d)
	st.CapEnd = "flat"
	return st
}

// flatStart is a stroke that starts with a flat cut and keeps its full width.
func flatStart(w float64, d string) Stroke {
	st := sw(w, d)
	st.CapStart = "flat"
	return st
}

// roundEnd keeps an explicit round terminal, overriding the joint flattening
// the engine applies to shared endpoints.
func roundEnd(st Stroke) Stroke { st.CapEnd = "round"; return st }

// roundStart keeps an explicit round terminal at the start of the stroke.
func roundStart(st Stroke) Stroke { st.CapStart = "round"; return st }

// flatTop flattens the top of a stem that joins a shoulder or bowl. A round
// cap there pokes out as a small "ear" above the arch.
func flatTop(st Stroke) Stroke {
	st.CapEnd = "flat"
	return st
}

// flatBoth closes a continuous shoulder stroke with flat terminals, so a
// single path can run from baseline up over the arch and back down without
// a cap at the shoulder or a rounded cap below the baseline.
func flatBoth(st Stroke) Stroke {
	st.CapStart = "flat"
	st.CapEnd = "flat"
	return st
}

// makeSerifTop marks the end of a stroke for a top serif only, leaving the
// bottom (usually joined to another stroke) plain.
func makeSerifTop(st Stroke) Stroke {
	st.SerifEnd = true
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
