package skeleton

func init() {
	// Uppercase.
	add(Def{Rune: 'A', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 60 0 L 333 700"),
		sw(1, "M 333 700 L 606 0"),
		bar(170, 496, 240),
	}})
	add(Def{Rune: 'B', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		sw(1, "M 100 700 C 430 700 580 620 580 525 C 580 430 430 350 100 350"),
		sw(1, "M 100 350 C 450 350 590 260 590 175 C 590 80 450 0 100 0"),
	}})
	add(Def{Rune: 'C', Advance: 700, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 600 560 C 500 690 150 710 70 520 C 10 400 10 300 70 180 C 150 -10 500 10 600 140"),
	}})
	add(Def{Rune: 'D', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		sw(1, "M 100 700 C 500 700 650 560 650 350 C 650 140 500 0 100 0"),
	}})
	add(Def{Rune: 'E', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		bar(100, 560, 700),
		bar(100, 500, 350),
		bar(100, 560, 0),
	}})
	add(Def{Rune: 'F', Advance: 556, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		bar(100, 510, 700),
		bar(100, 470, 350),
	}})
	add(Def{Rune: 'G', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 600 560 C 500 690 150 710 70 520 C 10 400 10 300 70 180 C 150 -10 500 10 600 140 L 600 230"),
		bar(440, 660, 350),
		vstemPlain(660, 230, 350),
	}})
	add(Def{Rune: 'H', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(622, 0, 700),
		bar(100, 622, 350),
	}})
	add(Def{Rune: 'I', Advance: 278, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(139, 0, 700),
	}})
	add(Def{Rune: 'J', Advance: 500, Zone: ZoneUpper, Strokes: []Stroke{
		vstemPlain(400, 120, 700),
		sw(1, "M 400 120 C 400 20 330 -10 240 -10 C 150 -10 80 40 60 140"),
	}})
	add(Def{Rune: 'K', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		sw(1, "M 100 350 L 560 700"),
		sw(1, "M 100 350 L 580 0"),
	}})
	add(Def{Rune: 'L', Advance: 556, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		bar(100, 500, 0),
	}})
	add(Def{Rune: 'M', Advance: 889, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(789, 0, 700),
		sw(1, "M 100 700 L 444 240"),
		sw(1, "M 444 240 L 789 700"),
	}})
	add(Def{Rune: 'N', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		vstem(622, 0, 700),
		sw(1, "M 100 700 L 622 0"),
	}})
	add(Def{Rune: 'O', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		ellipse(389, 350, 300, 350),
	}})
	add(Def{Rune: 'P', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		sw(1, "M 100 700 C 480 700 600 620 600 525 C 600 430 480 350 100 350"),
	}})
	add(Def{Rune: 'Q', Advance: 778, Zone: ZoneUpper, Strokes: []Stroke{
		ellipse(389, 350, 300, 350),
		sw(0.9, "M 480 100 L 700 -80"),
	}})
	add(Def{Rune: 'R', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		vstem(100, 0, 700),
		sw(1, "M 100 700 C 480 700 600 620 600 525 C 600 430 480 350 100 350"),
		sw(1, "M 350 350 L 650 0"),
	}})
	add(Def{Rune: 'S', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 590 570 C 500 680 180 710 90 560 C 20 440 180 380 350 350 C 520 320 650 250 600 120 C 540 -20 180 -10 80 130"),
	}})
	add(Def{Rune: 'T', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		bar(50, 561, 700),
		vstem(305, 0, 700),
	}})
	add(Def{Rune: 'U', Advance: 722, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 100 700 L 100 220 C 100 70 250 0 361 0 C 472 0 622 70 622 220 L 622 700"),
	}})
	add(Def{Rune: 'V', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 60 700 L 333 0 L 606 700"),
	}})
	add(Def{Rune: 'W', Advance: 944, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 60 700 L 280 0"),
		sw(1, "M 280 0 L 500 450"),
		sw(1, "M 500 450 L 720 0"),
		sw(1, "M 720 0 L 940 700"),
	}})
	add(Def{Rune: 'X', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 80 700 L 587 0"),
		sw(1, "M 587 700 L 80 0"),
	}})
	add(Def{Rune: 'Y', Advance: 667, Zone: ZoneUpper, Strokes: []Stroke{
		sw(1, "M 60 700 L 333 300 L 606 700"),
		vstemPlain(333, 0, 300),
	}})
	add(Def{Rune: 'Z', Advance: 611, Zone: ZoneUpper, Strokes: []Stroke{
		bar(60, 551, 700),
		sw(1, "M 551 700 L 60 0"),
		bar(60, 551, 0),
	}})

	// Lowercase.
	add(Def{Rune: 'a', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(270, 250, 190, 250),
		vstemPlain(460, 0, 500),
	}})
	add(Def{Rune: 'b', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 740),
		ellipse(320, 250, 210, 250),
	}})
	add(Def{Rune: 'c', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 480 400 C 400 510 120 520 60 390 C 10 290 20 190 70 100 C 140 -20 400 -10 480 100"),
	}})
	add(Def{Rune: 'd', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(290, 250, 210, 250),
		vstem(500, 0, 740),
	}})
	add(Def{Rune: 'e', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		bar(100, 500, 250),
		sw(1, "M 500 250 C 500 430 400 510 290 510 C 160 510 80 400 80 250"),
		sw(1, "M 80 250 C 80 100 160 -10 290 -10 C 400 -10 480 50 500 150"),
	}})
	add(Def{Rune: 'f', Advance: 333, Zone: ZoneLower, Strokes: []Stroke{
		vstemPlain(160, 0, 740),
		sw(1, "M 160 740 C 250 800 330 760 330 680"),
		bar(60, 300, 500),
	}})
	add(Def{Rune: 'g', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(270, 280, 190, 220),
		vstemPlain(460, -100, 280),
		sw(1, "M 460 -100 C 460 -200 380 -230 300 -210"),
	}})
	add(Def{Rune: 'h', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 740),
		sw(1, "M 100 380 C 100 500 220 510 300 510 C 400 510 460 440 460 340 L 460 0"),
	}})
	add(Def{Rune: 'i', Advance: 278, Zone: ZoneLower, Strokes: []Stroke{
		vstem(139, 0, 500),
	}, Dots: []Dot{dot(139, 640, 28)}})
	add(Def{Rune: 'j', Advance: 278, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 160 500 L 160 -80 C 160 -190 100 -220 40 -200"),
	}, Dots: []Dot{dot(160, 640, 28)}})
	add(Def{Rune: 'k', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 740),
		sw(1, "M 100 300 L 430 500"),
		sw(1, "M 100 300 L 450 0"),
	}})
	add(Def{Rune: 'l', Advance: 278, Zone: ZoneLower, Strokes: []Stroke{
		vstem(139, 0, 740),
	}})
	add(Def{Rune: 'm', Advance: 889, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		vstemPlain(444, 0, 500),
		vstemPlain(788, 0, 500),
		sw(1, "M 100 380 C 100 500 220 510 280 510 C 380 510 444 440 444 340 L 444 0"),
		sw(1, "M 444 380 C 444 500 550 510 620 510 C 720 510 788 440 788 340 L 788 0"),
	}})
	add(Def{Rune: 'n', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		sw(1, "M 100 380 C 100 500 220 510 300 510 C 400 510 460 440 460 340 L 460 0"),
	}})
	add(Def{Rune: 'o', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(305, 250, 205, 250),
	}})
	add(Def{Rune: 'p', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		vstemPlain(110, -200, 500),
		ellipse(320, 250, 210, 250),
	}})
	add(Def{Rune: 'q', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(290, 250, 210, 250),
		vstemPlain(500, -200, 500),
	}})
	add(Def{Rune: 'r', Advance: 389, Zone: ZoneLower, Strokes: []Stroke{
		vstem(100, 0, 500),
		sw(1, "M 100 380 C 100 500 220 510 300 510"),
	}})
	add(Def{Rune: 's', Advance: 500, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 430 420 C 360 500 120 510 80 410 C 50 330 160 290 280 260 C 400 230 470 180 440 90 C 400 -20 120 -10 60 80"),
	}})
	add(Def{Rune: 't', Advance: 333, Zone: ZoneLower, Strokes: []Stroke{
		vstemPlain(160, -20, 680),
		bar(60, 300, 500),
		sw(1, "M 160 60 C 160 0 220 -20 280 -20"),
	}})
	add(Def{Rune: 'u', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 100 500 L 100 160 C 100 40 220 0 305 0 C 390 0 500 40 500 160 L 500 500"),
	}})
	add(Def{Rune: 'v', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 60 500 L 278 0 L 496 500"),
	}})
	add(Def{Rune: 'w', Advance: 778, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 50 500 L 210 0"),
		sw(1, "M 210 0 L 389 350"),
		sw(1, "M 389 350 L 568 0"),
		sw(1, "M 568 0 L 728 500"),
	}})
	add(Def{Rune: 'x', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 70 500 L 486 0"),
		sw(1, "M 486 500 L 70 0"),
	}})
	add(Def{Rune: 'y', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		sw(1, "M 60 500 L 278 0"),
		sw(1, "M 496 500 L 278 0 L 180 -180"),
	}})
	add(Def{Rune: 'z', Advance: 500, Zone: ZoneLower, Strokes: []Stroke{
		bar(70, 430, 500),
		sw(1, "M 430 500 L 70 0"),
		bar(70, 430, 0),
	}})

	// Digits.
	add(Def{Rune: '0', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		ellipse(300, 350, 240, 350),
	}})
	add(Def{Rune: '1', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		vstem(300, 0, 700),
		sw(1, "M 180 560 L 300 700"),
		bar(160, 440, 0),
	}})
	add(Def{Rune: '2', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		sw(1, "M 100 530 C 100 650 500 750 540 560 C 570 420 300 250 100 0"),
		bar(100, 540, 0),
	}})
	add(Def{Rune: '3', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		sw(1, "M 100 580 C 180 700 520 720 540 580 C 560 460 380 380 300 370"),
		sw(1, "M 300 370 C 500 370 580 250 550 130 C 510 -20 160 -20 90 120"),
	}})
	add(Def{Rune: '4', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		sw(1, "M 380 700 L 80 200 L 560 200"),
		vstemPlain(380, 0, 700),
	}})
	add(Def{Rune: '5', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		bar(100, 500, 700),
		vstemPlain(100, 400, 700),
		sw(1, "M 100 400 C 400 430 580 320 560 180 C 530 20 160 -10 80 130"),
	}})
	add(Def{Rune: '6', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		ellipse(300, 240, 210, 240),
		sw(1, "M 100 480 C 150 680 500 720 540 560"),
	}})
	add(Def{Rune: '7', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		bar(80, 540, 700),
		sw(1, "M 540 700 L 200 0"),
	}})
	add(Def{Rune: '8', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		ellipse(300, 520, 190, 180),
		ellipse(300, 180, 220, 180),
	}})
	add(Def{Rune: '9', Advance: 600, Zone: ZoneDigit, Strokes: []Stroke{
		ellipse(300, 460, 210, 240),
		sw(1, "M 500 220 C 450 20 100 -20 60 140"),
	}})

	// Punctuation and symbols.
	add(Def{Rune: ' ', Advance: 280, Zone: ZoneSymbol})
	add(Def{Rune: '!', Advance: 260, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 110 700 L 110 120"),
	}, Dots: []Dot{dot(110, 40, 30)}})
	add(Def{Rune: '"', Advance: 300, Zone: ZoneSymbol, Strokes: []Stroke{
		vstemPlain(70, 520, 700),
		vstemPlain(150, 520, 700),
	}})
	add(Def{Rune: '#', Advance: 400, Zone: ZoneSymbol, Strokes: []Stroke{
		vstemPlain(120, 0, 500),
		vstemPlain(280, 0, 500),
		bar(50, 350, 160),
		bar(50, 350, 340),
	}})
	add(Def{Rune: '$', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 430 470 C 360 550 100 560 70 460 C 45 380 160 340 270 310 C 380 280 450 230 420 140 C 380 20 100 20 50 110"),
		vstemPlain(250, -50, 550),
	}})
	add(Def{Rune: '%', Advance: 460, Zone: ZoneSymbol, Strokes: []Stroke{
		ellipse(90, 580, 60, 60),
		ellipse(350, 120, 60, 60),
		sw(0.9, "M 60 0 L 400 700"),
	}})
	add(Def{Rune: '&', Advance: 600, Zone: ZoneSymbol, Strokes: []Stroke{
		ellipse(220, 180, 120, 120),
		sw(1, "M 100 450 C 100 600 300 700 360 560 C 400 470 300 420 220 380 C 120 330 60 250 100 150 C 150 20 330 50 430 180"),
	}})
	add(Def{Rune: '\'', Advance: 220, Zone: ZoneSymbol, Strokes: []Stroke{
		vstemPlain(100, 520, 700),
	}})
	add(Def{Rune: '(', Advance: 280, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 220 700 C 80 560 80 140 220 0"),
	}})
	add(Def{Rune: ')', Advance: 280, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 60 700 C 200 560 200 140 60 0"),
	}})
	add(Def{Rune: '*', Advance: 320, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(0.8, "M 80 500 L 240 700"),
		sw(0.8, "M 240 500 L 80 700"),
		bar(60, 260, 600),
	}})
	add(Def{Rune: '+', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{
		vstemPlain(250, 100, 500),
		bar(50, 450, 300),
	}})
	add(Def{Rune: ',', Advance: 260, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 100 40 C 120 -30 90 -90 50 -120"),
	}, Dots: []Dot{dot(100, 40, 30)}})
	add(Def{Rune: '-', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{bar(60, 440, 300)}})
	add(Def{Rune: '.', Advance: 300, Zone: ZoneSymbol, Dots: []Dot{dot(150, 40, 32)}})
	add(Def{Rune: '/', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{sw(1, "M 60 0 L 440 700")}})
	add(Def{Rune: ':', Advance: 280, Zone: ZoneSymbol, Dots: []Dot{dot(140, 40, 30), dot(140, 360, 30)}})
	add(Def{Rune: ';', Advance: 280, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 140 40 C 160 -30 130 -90 90 -120"),
	}, Dots: []Dot{dot(140, 360, 30), dot(140, 40, 30)}})
	add(Def{Rune: '<', Advance: 480, Zone: ZoneSymbol, Strokes: []Stroke{sw(1, "M 400 600 L 80 300 L 400 0")}})
	add(Def{Rune: '=', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{bar(60, 440, 220), bar(60, 440, 380)}})
	add(Def{Rune: '>', Advance: 480, Zone: ZoneSymbol, Strokes: []Stroke{sw(1, "M 80 600 L 400 300 L 80 0")}})
	add(Def{Rune: '?', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 80 560 C 100 700 420 720 440 560 C 450 450 300 420 260 330 L 260 220"),
		vstemPlain(260, 120, 220),
	}, Dots: []Dot{dot(260, 40, 30)}})
	add(Def{Rune: '@', Advance: 700, Zone: ZoneSymbol, Strokes: []Stroke{
		ellipse(300, 350, 260, 300),
		ellipse(290, 330, 110, 140),
		vstemPlain(400, 250, 330),
	}})
	add(Def{Rune: '[', Advance: 300, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 220 700 L 100 700 L 100 0 L 220 0"),
	}})
	add(Def{Rune: '\\', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{sw(1, "M 60 700 L 440 0")}})
	add(Def{Rune: ']', Advance: 300, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 80 700 L 200 700 L 200 0 L 80 0"),
	}})
	add(Def{Rune: '^', Advance: 320, Zone: ZoneSymbol, Strokes: []Stroke{sw(0.8, "M 60 500 L 160 700 L 260 500")}})
	add(Def{Rune: '_', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{bar(0, 500, -100)}})
	add(Def{Rune: '`', Advance: 250, Zone: ZoneSymbol, Strokes: []Stroke{sw(1, "M 80 700 L 160 600")}})
	add(Def{Rune: '{', Advance: 300, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 220 700 C 100 680 140 600 140 500 C 140 400 60 350 60 350 C 140 350 140 300 140 200 C 140 100 100 20 220 0"),
	}})
	add(Def{Rune: '|', Advance: 250, Zone: ZoneSymbol, Strokes: []Stroke{vstemPlain(100, -100, 700)}})
	add(Def{Rune: '}', Advance: 300, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(1, "M 80 700 C 200 680 160 600 160 500 C 160 400 240 350 240 350 C 160 350 160 300 160 200 C 160 100 200 20 80 0"),
	}})
	add(Def{Rune: '~', Advance: 500, Zone: ZoneSymbol, Strokes: []Stroke{
		sw(0.9, "M 60 300 C 120 400 180 400 240 320 C 300 240 360 240 440 340"),
	}})

	// Humanist/serif alternate letterforms.
	addVariant('a', "serif", Def{Rune: 'a', Advance: 556, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(250, 250, 180, 250),
		vstemPlain(430, 0, 500),
		sw(1, "M 430 500 C 360 540 250 540 210 480"),
	}})
	addVariant('g', "serif", Def{Rune: 'g', Advance: 611, Zone: ZoneLower, Strokes: []Stroke{
		ellipse(250, 300, 170, 200),
		sw(1, "M 420 500 C 420 560 350 590 280 580"),
		vstemPlain(420, -100, 500),
		sw(1, "M 420 -100 C 420 -210 340 -240 260 -220"),
	}})
}
