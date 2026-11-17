package skeleton

func initCyrillic() {
	// Cyrillic uppercase.
	reuse('А', 'A', 667, ZoneUpper)
	reuse('В', 'B', 667, ZoneUpper)
	reuse('Е', 'E', 611, ZoneUpper)
	reuse('К', 'K', 667, ZoneUpper)
	reuse('М', 'M', 889, ZoneUpper)
	reuse('Н', 'H', 722, ZoneUpper)
	reuse('О', 'O', 778, ZoneUpper)
	reuse('Р', 'P', 667, ZoneUpper)
	reuse('С', 'C', 700, ZoneUpper)
	reuse('Т', 'T', 611, ZoneUpper)
	reuse('У', 'Y', 667, ZoneUpper)
	reuse('Х', 'X', 667, ZoneUpper)
	add(Def{Rune: 'Б', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		bar(100, 550, 700),
		sw(1, "M 100 350 C 430 350 560 280 560 175 C 560 70 430 0 100 0"),
	}})
	add(Def{Rune: 'Г', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		bar(80, 550, 700),
		vstem(80, 0, 700),
	}})
	add(Def{Rune: 'Д', Advance: 700, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 120 0 L 160 700 L 540 700 L 580 0"),
		bar(40, 660, 0),
	}})
	add(Def{Rune: 'Ж', Advance: 889, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(444, 0, 700),
		sw(1, "M 100 700 L 444 350"),
		sw(1, "M 444 350 L 788 0"),
		sw(1, "M 788 700 L 444 350"),
		sw(1, "M 444 350 L 100 0"),
	}})
	add(Def{Rune: 'З', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 100 580 C 180 700 520 720 540 580 C 560 460 380 380 300 370"),
		sw(1, "M 300 370 C 500 370 580 250 550 130 C 510 -20 160 -20 90 120"),
	}})
	add(Def{Rune: 'И', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(622, 0, 700),
		sw(1, "M 100 0 L 622 700"),
	}})
	add(Def{Rune: 'Л', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 100 0 L 200 700"),
		bar(200, 622, 700),
		vstem(622, 0, 700),
	}})
	add(Def{Rune: 'П', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		bar(80, 642, 700),
		vstem(80, 0, 700),
		vstem(642, 0, 700),
	}})
	add(Def{Rune: 'Ф', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		ellipse(389, 350, 240, 350),
		vstem(389, 0, 700),
	}})
	add(Def{Rune: 'Ц', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(622, 0, 700),
		bar(100, 700, 0),
		vstemPlain(700, -150, 0),
	}})
	add(Def{Rune: 'Ч', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 350, 700),
		vstem(622, 0, 700),
		bar(100, 622, 350),
	}})
	add(Def{Rune: 'Ш', Advance: 889, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(444, 0, 700),
		vstem(788, 0, 700),
		bar(100, 788, 0),
	}})
	add(Def{Rune: 'Щ', Advance: 944, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(444, 0, 700),
		vstem(788, 0, 700),
		bar(100, 850, 0),
		vstemPlain(850, -150, 0),
	}})
	add(Def{Rune: 'Ъ', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		bar(100, 550, 700),
		vstemPlain(550, 0, 350),
		sw(1, "M 100 350 C 450 350 580 280 580 175 C 580 70 450 0 100 0"),
	}})
	add(Def{Rune: 'Ы', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(700, 0, 700),
		sw(1, "M 100 350 C 450 350 580 280 580 175 C 580 70 450 0 100 0"),
	}})
	add(Def{Rune: 'Ь', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		sw(1, "M 100 350 C 430 350 560 280 560 175 C 560 70 430 0 100 0"),
	}})
	add(Def{Rune: 'Э', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 100 560 C 200 690 550 710 630 520 C 690 400 690 300 630 180 C 550 -10 200 10 100 140"),
		bar(350, 630, 350),
	}})
	add(Def{Rune: 'Ю', Advance: 889, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		ellipse(550, 350, 180, 350),
		bar(100, 370, 350),
	}})
	add(Def{Rune: 'Я', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(622, 0, 700),
		sw(1, "M 622 700 C 242 700 122 620 122 525 C 122 430 242 350 622 350"),
		sw(1, "M 372 350 L 72 0"),
	}})
	add(Def{Rune: 'Є', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 580 560 C 490 690 150 710 70 520 C 10 400 10 300 70 180 C 150 -10 490 10 580 140"),
		bar(300, 580, 350),
	}})
	reuse('І', 'I', 278, ZoneUpper)
	reuse('Ґ', 'Г', 611, ZoneUpper)

	// Cyrillic lowercase.
	reuse('а', 'a', 556, ZoneLower)
	reuse('е', 'e', 556, ZoneLower)
	reuse('о', 'o', 611, ZoneLower)
	reuse('р', 'p', 611, ZoneLower)
	reuse('с', 'c', 556, ZoneLower)
	reuse('у', 'y', 556, ZoneLower)
	reuse('х', 'x', 556, ZoneLower)
	reuse('к', 'k', 556, ZoneLower)
	reuse('м', 'm', 778, ZoneLower)
	reuse('н', 'H', 611, ZoneLower)
	reuse('т', 'm', 611, ZoneLower)
	add(Def{Rune: 'б', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(300, 220, 200, 220),
		sw(1, "M 100 480 C 150 700 500 720 540 560"),
		vstemPlain(500, 300, 560),
	}})
	add(Def{Rune: 'в', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		sw(1, "M 100 500 C 400 500 480 450 480 380 C 480 310 400 250 100 250"),
		sw(1, "M 100 250 C 430 250 520 190 520 125 C 520 60 430 0 100 0"),
	}})
	add(Def{Rune: 'г', Advance: 389, Zone: ZoneLower, Strokes: []Stroke{
		bar(80, 450, 500),
		vstem(80, 0, 500),
	}})
	add(Def{Rune: 'д', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 140 0 L 180 500 L 460 500 L 500 0"),
		bar(60, 540, 0),
	}})
	add(Def{Rune: 'ж', Advance: 778, Zone: ZoneLower, Strokes: []Stroke{
		vstem(389, 0, 500),
		sw(1, "M 100 500 L 389 250"),
		sw(1, "M 389 250 L 678 0"),
		sw(1, "M 678 500 L 389 250"),
		sw(1, "M 389 250 L 100 0"),
	}})
	add(Def{Rune: 'з', Advance: 500, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 80 410 C 150 520 440 520 460 410 C 480 330 320 280 250 270"),
		sw(1, "M 250 270 C 420 270 500 190 470 90 C 430 -30 120 -20 60 80"),
	}})
	add(Def{Rune: 'и', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		vstem(500, 0, 500),
		sw(1, "M 100 0 L 500 500"),
	}})
	add(Def{Rune: 'л', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 100 0 L 180 500"),
		bar(180, 500, 500),
		vstem(500, 0, 500),
	}})
	add(Def{Rune: 'п', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		bar(60, 520, 500),
		vstem(60, 0, 500),
		vstem(520, 0, 500),
	}})
	add(Def{Rune: 'ф', Advance: 667, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(333, 250, 200, 250),
		vstemPlain(333, -200, 700),
	}})
	add(Def{Rune: 'ц', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		vstem(500, 0, 500),
		bar(100, 580, 0),
		vstemPlain(580, -150, 0),
	}})
	add(Def{Rune: 'ч', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 250, 500),
		vstem(500, 0, 500),
		bar(100, 500, 250),
	}})
	add(Def{Rune: 'ш', Advance: 778, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		vstem(389, 0, 500),
		vstem(678, 0, 500),
		bar(100, 678, 0),
	}})
	add(Def{Rune: 'щ', Advance: 833, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		vstem(389, 0, 500),
		vstem(678, 0, 500),
		bar(100, 780, 0),
		vstemPlain(780, -150, 0),
	}})
	add(Def{Rune: 'ъ', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		bar(100, 420, 500),
		sw(1, "M 100 250 C 380 250 480 200 480 125 C 480 50 380 0 100 0"),
	}})
	add(Def{Rune: 'ы', Advance: 667, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		vstem(600, 0, 500),
		sw(1, "M 100 250 C 380 250 480 200 480 125 C 480 50 380 0 100 0"),
	}})
	add(Def{Rune: 'ь', Advance: 500, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		sw(1, "M 100 250 C 380 250 480 200 480 125 C 480 50 380 0 100 0"),
	}})
	add(Def{Rune: 'э', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 80 400 C 160 510 420 520 480 390 C 520 290 510 190 460 100 C 390 -20 160 -10 80 100"),
		bar(180, 450, 250),
	}})
	add(Def{Rune: 'ю', Advance: 778, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		ellipse(550, 250, 160, 250),
		bar(100, 370, 250),
	}})
	add(Def{Rune: 'я', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		vstem(500, 0, 500),
		sw(1, "M 500 500 C 200 500 100 450 100 380 C 100 310 200 250 500 250"),
		sw(1, "M 300 250 L 80 0"),
	}})
	add(Def{Rune: 'є', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 480 400 C 400 510 120 520 60 390 C 10 290 20 190 70 100 C 140 -20 400 -10 480 100"),
		bar(220, 480, 250),
	}})
	reuse('і', 0x0131, 278, ZoneLower)
	reuse('ґ', 'г', 389, ZoneLower)
}
