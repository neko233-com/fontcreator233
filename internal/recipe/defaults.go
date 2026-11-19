package recipe

// Default returns a complete, valid starting recipe.
func Default() *Recipe {
	return &Recipe{
		APIVersion: "fontcreator233/v1",
		Meta: Meta{
			Family:      "New Font",
			Style:       "Regular",
			Version:     "0.5.0",
			Designer:    "",
			Description: "Generated with fontcreator233.",
			License:     "OFL-1.1",
			Accent:      "#5B8CFF",
			SampleText:  "Hamburgefonstiv 字体创造器 0123",
		},
		Metrics: Metrics{
			Ascender:  800,
			Descender: -200,
			XHeight:   0.52,
			CapHeight: 0.72,
			LineGap:   0,
		},
		Style: Style{
			Skeleton:       "geometric",
			Weight:         400,
			Width:          100,
			Slant:          0,
			Contrast:       0.1,
			Terminal:       "round",
			Serif:          "none",
			SerifSize:      0,
			Overshoot:      0.012,
			Roughness:      0,
			JitterRotation: 0,
			Seed:           233,
		},
		Coverage: Coverage{
			Generated: []string{"latin", "greek", "cyrillic"},
			Base: BaseCoverage{
				CJK:     "common",
				Kana:    true,
				Hangul:  true,
				Symbols: true,
			},
		},
		Output: Output{
			Formats:  []string{"ttf", "woff"},
			Targets:  []string{"universal", "web", "system", "unity", "apps"},
			Specimen: true,
			Open:     true,
			SplitCJK: 0,
			TMP: TMP{
				Population:        "dynamic",
				SamplingPointSize: 90,
				Padding:           9,
				AtlasWidth:        1024,
				AtlasHeight:       1024,
				MultiAtlas:        true,
				RenderMode:        "SDFAA",
			},
		},
	}
}
