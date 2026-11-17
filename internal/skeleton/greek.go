package skeleton

func initGreek() {
	// Greek uppercase.
	reuse('Α', 'A', 667, ZoneUpper)
	reuse('Β', 'B', 667, ZoneUpper)
	reuse('Ε', 'E', 611, ZoneUpper)
	reuse('Ζ', 'Z', 611, ZoneUpper)
	reuse('Η', 'H', 722, ZoneUpper)
	reuse('Ι', 'I', 278, ZoneUpper)
	reuse('Κ', 'K', 667, ZoneUpper)
	reuse('Μ', 'M', 889, ZoneUpper)
	reuse('Ν', 'N', 722, ZoneUpper)
	reuse('Ο', 'O', 778, ZoneUpper)
	reuse('Ρ', 'P', 667, ZoneUpper)
	reuse('Τ', 'T', 611, ZoneUpper)
	reuse('Υ', 'Y', 667, ZoneUpper)
	reuse('Χ', 'X', 667, ZoneUpper)
	add(Def{Rune: 'Γ', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		bar(80, 550, 700),
		vstem(80, 0, 700),
	}})
	add(Def{Rune: 'Δ', Advance: 700, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 350 700 L 650 0"),
		sw(1, "M 350 700 L 50 0"),
		bar(50, 650, 0),
	}})
	add(Def{Rune: 'Θ', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		ellipse(389, 350, 300, 350),
		bar(120, 658, 350),
	}})
	add(Def{Rune: 'Λ', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 60 0 L 333 700"),
		sw(1, "M 333 700 L 606 0"),
	}})
	add(Def{Rune: 'Ξ', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		bar(80, 530, 700),
		bar(80, 530, 350),
		bar(80, 530, 0),
	}})
	add(Def{Rune: 'Π', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		bar(80, 642, 700),
		vstem(80, 0, 700),
		vstem(642, 0, 700),
	}})
	add(Def{Rune: 'Σ', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		bar(80, 550, 700),
		bar(80, 550, 0),
		sw(1, "M 550 700 L 280 350"),
		sw(1, "M 280 350 L 550 0"),
	}})
	add(Def{Rune: 'Φ', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		ellipse(389, 350, 240, 350),
		vstem(389, 0, 700),
	}})
	add(Def{Rune: 'Ψ', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(389, 0, 700),
		sw(1, "M 100 300 C 100 520 250 550 389 550 C 528 550 678 520 678 300"),
	}})
	add(Def{Rune: 'Ω', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 100 0 L 100 180 C 100 500 220 700 389 700 C 558 700 678 500 678 180 L 678 0"),
		bar(50, 250, 0),
		bar(528, 728, 0),
	}})

	// Greek lowercase.
	reuse('α', 'a', 556, ZoneLower)
	reuse('κ', 'k', 556, ZoneLower)
	reuse('ν', 'v', 556, ZoneLower)
	reuse('ο', 'o', 611, ZoneLower)
	reuse('υ', 'u', 556, ZoneLower)
	reuse('χ', 'x', 556, ZoneLower)
	add(Def{Rune: 'β', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstemPlain(100, -200, 740),
		ellipse(300, 500, 190, 200),
		ellipse(300, 150, 210, 170),
	}})
	add(Def{Rune: 'γ', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 100 500 C 150 200 250 0 300 -200"),
		sw(1, "M 500 500 C 450 300 350 150 280 0"),
	}})
	add(Def{Rune: 'δ', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(300, 200, 180, 180),
		sw(1, "M 120 420 C 120 550 400 650 480 500"),
	}})
	add(Def{Rune: 'ε', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 480 400 C 400 510 100 510 60 400 C 30 330 150 280 260 260"),
		sw(1, "M 260 260 C 400 240 500 180 460 80 C 410 -30 100 -10 60 90"),
		bar(100, 450, 260),
	}})
	add(Def{Rune: 'ζ', Advance: 500, Zone: ZoneLower, Strokes: []Stroke{
		bar(70, 430, 500),
		sw(1, "M 430 500 L 70 0"),
		bar(70, 430, 0),
		sw(1, "M 100 0 C 70 -120 120 -200 200 -220"),
	}})
	add(Def{Rune: 'η', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, -200, 500),
		sw(1, "M 100 380 C 100 500 220 510 300 510 C 400 510 460 440 460 340 L 460 -200"),
	}})
	add(Def{Rune: 'θ', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(300, 250, 200, 250),
		bar(100, 500, 250),
	}})
	add(Def{Rune: 'ι', Advance: 278, Zone: ZoneLower, Strokes: []Stroke{
		vstemPlain(139, 0, 500),
	}})
	add(Def{Rune: 'λ', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 60 0 L 278 500"),
		sw(1, "M 278 500 L 496 0"),
	}})
	add(Def{Rune: 'μ', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 100 500 L 100 -200"),
		sw(1, "M 100 380 C 100 500 220 510 300 510 C 400 510 460 440 460 340 L 460 0"),
	}})
	add(Def{Rune: 'ξ', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 100 600 C 150 700 450 700 500 600 C 540 520 400 450 300 430 C 200 410 100 380 100 320"),
		bar(150, 450, 300),
		sw(1, "M 100 320 C 100 200 250 180 350 160 C 550 120 550 -20 450 -80 C 350 -140 120 -120 60 -20"),
	}})
	add(Def{Rune: 'π', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		bar(60, 550, 500),
		vstemPlain(100, 0, 500),
		vstemPlain(500, 0, 500),
	}})
	add(Def{Rune: 'ρ', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstemPlain(100, -200, 500),
		ellipse(300, 300, 200, 200),
	}})
	add(Def{Rune: 'σ', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(280, 250, 180, 250),
		bar(440, 560, 420),
	}})
	add(Def{Rune: 'ς', Advance: 500, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 430 420 C 360 500 120 510 80 410 C 50 330 160 290 280 260 C 400 230 470 180 440 90 C 400 -20 120 -10 60 80 C 50 -40 100 -140 160 -200"),
	}})
	add(Def{Rune: 'τ', Advance: 500, Zone: ZoneLower, Strokes: []Stroke{
		bar(50, 450, 500),
		vstemPlain(250, 0, 500),
	}})
	add(Def{Rune: 'φ', Advance: 667, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(333, 350, 220, 250),
		vstemPlain(333, -200, 700),
	}})
	add(Def{Rune: 'ψ', Advance: 700, Zone: ZoneLower, Strokes: []Stroke{
		vstemPlain(350, -200, 700),
		sw(1, "M 100 300 C 100 480 220 520 350 520 C 480 520 600 480 600 300"),
	}})
	add(Def{Rune: 'ω', Advance: 778, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 60 400 C 60 100 150 0 250 0 C 330 0 380 80 389 250 C 398 80 448 0 528 0 C 628 0 718 100 718 400"),
	}})
}
