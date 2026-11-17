package skeleton

func initMarks() {
	// Combining marks are designed around x=-80..80, y=0..110. Composition
	// translates them above or below the base glyph.
	add(Def{Rune: 0x0300, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.6, "M -40 100 L 40 0")}})
	add(Def{Rune: 0x0301, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.6, "M -40 0 L 40 100")}})
	add(Def{Rune: 0x0302, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.6, "M -70 0 L 0 100 L 70 0")}})
	add(Def{Rune: 0x0303, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.55, "M -80 50 C -40 130 40 -30 80 50")}})
	add(Def{Rune: 0x0304, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{bar(-70, 70, 50)}})
	add(Def{Rune: 0x0306, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.6, "M -70 70 C -40 -10 40 -10 70 70")}})
	add(Def{Rune: 0x0307, Advance: 0, Zone: ZoneMark, Dots: []Dot{dot(0, 50, 26)}})
	add(Def{Rune: 0x0308, Advance: 0, Zone: ZoneMark, Dots: []Dot{dot(-40, 50, 22), dot(40, 50, 22)}})
	add(Def{Rune: 0x030A, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{func() Stroke {
		st := ellipse(0, 50, 32, 32)
		st.Width = 0.55
		return st
	}()}})
	add(Def{Rune: 0x030B, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{
		sw(0.6, "M -60 0 L -20 100"),
		sw(0.6, "M 20 0 L 60 100"),
	}})
	add(Def{Rune: 0x030C, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.6, "M -70 100 L 0 0 L 70 100")}})
	add(Def{Rune: 0x0323, Advance: 0, Zone: ZoneMark, Dots: []Dot{dot(0, -40, 26)}})
	add(Def{Rune: 0x0327, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.6, "M 0 0 C 0 -80 -60 -100 -80 -40")}})
	add(Def{Rune: 0x0328, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{sw(0.6, "M 0 0 C 30 -80 80 -100 60 -150")}})
	add(Def{Rune: 0x0331, Advance: 0, Zone: ZoneMark, Strokes: []Stroke{bar(-60, 60, -60)}})
}
