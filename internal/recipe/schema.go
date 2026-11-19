package recipe

// Schema returns the JSON Schema for a fontcreator233/v1 recipe.
func Schema() map[string]any {
	obj := func(props map[string]any, required ...string) map[string]any {
		m := map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties":           props,
		}
		if len(required) > 0 {
			m["required"] = toAny(required)
		}
		return m
	}
	str := func(desc string) map[string]any {
		return map[string]any{"type": "string", "description": desc}
	}
	num := func(desc string, min, max float64) map[string]any {
		return map[string]any{"type": "number", "description": desc, "minimum": min, "maximum": max}
	}
	integer := func(desc string, min, max float64) map[string]any {
		return map[string]any{"type": "integer", "description": desc, "minimum": min, "maximum": max}
	}
	boolean := func(desc string) map[string]any {
		return map[string]any{"type": "boolean", "description": desc}
	}
	enum := func(desc string, values ...string) map[string]any {
		return map[string]any{"type": "string", "description": desc, "enum": toAny(values)}
	}

	meta := obj(map[string]any{
		"family":      str("Font family name; must not start with Noto"),
		"style":       str("Author-visible style name; resolved for italic naming"),
		"version":     str("Font version string"),
		"designer":    str("Designer name"),
		"description": str("Human-readable description"),
		"license":     str("OFL-1.1 is required when base glyphs are used"),
		"accent":      str("Specimen accent color, #RGB or #RRGGBB"),
		"sampleText":  str("Default specimen text"),
	}, "family", "style", "version", "license")

	metrics := obj(map[string]any{
		"ascender":  integer("Ascender in font units", 600, 1000),
		"descender": integer("Descender in font units", -400, -50),
		"xHeight":   num("x-height as a fraction of the em square", 0.40, 0.60),
		"capHeight": num("cap height as a fraction of the em square", 0.64, 0.78),
		"lineGap":   integer("Line gap in font units", 0, 300),
	}, "ascender", "descender", "xHeight", "capHeight", "lineGap")

	style := obj(map[string]any{
		"skeleton":       enum("Skeleton family", "geometric", "humanist", "serif"),
		"weight":         integer("Weight 100-900", 100, 900),
		"width":          num("Horizontal width percentage", 75, 125),
		"slant":          num("Slant in degrees, positive leans right", -15, 15),
		"contrast":       num("Stroke contrast 0-1", 0, 1),
		"terminal":       enum("Open stroke terminal", "flat", "round", "angled", "flare"),
		"serif":          enum("Serif style", "none", "slab", "bracketed", "hairline"),
		"serifSize":      num("Serif size relative to stem", 0, 1),
		"overshoot":      num("Round-letter overshoot fraction", 0, 0.1),
		"tracking":       num("Extra advance tracking as percent of em", -5, 10),
		"roughness":      num("Deterministic hand-drawn jitter 0-1", 0, 1),
		"jitterRotation": num("Per-glyph rotation jitter in degrees", 0, 2),
		"seed":           map[string]any{"type": "integer", "description": "Deterministic seed"},
	}, "skeleton", "weight", "width", "slant", "contrast", "terminal", "serif", "serifSize", "overshoot", "roughness", "jitterRotation", "seed")

	coverage := obj(map[string]any{
		"generated": map[string]any{
			"type":  "array",
			"items": enum("Generated script", "latin", "greek", "cyrillic"),
		},
		"base": obj(map[string]any{
			"cjk":     enum("CJK coverage", "none", "sample", "common", "full"),
			"kana":    boolean("Include kana"),
			"hangul":  boolean("Include Hangul"),
			"symbols": boolean("Include symbols/punctuation"),
		}, "cjk", "kana", "hangul", "symbols"),
	}, "generated", "base")

	glyphs := obj(map[string]any{
		"overrides": str("Directory of U+XXXX.svg glyph overrides"),
	})

	openType := obj(map[string]any{
		"contextual": map[string]any{
			"type": "array",
			"items": obj(map[string]any{
				"feature":   str("OpenType feature tag; defaults to calt"),
				"backtrack": str("Optional characters that must precede the input"),
				"input":     str("Input character sequence, for example ->"),
				"lookahead": str("Optional characters that must follow the input"),
				"target":    str("Single target character, for example →"),
			}, "input", "target"),
		},
		"color": obj(map[string]any{
			"palettes": map[string]any{
				"type":        "array",
				"description": "CPAL palettes; every palette has the same number of #RRGGBB colors",
				"items": map[string]any{
					"type":  "array",
					"items": str("Palette color as #RGB or #RRGGBB"),
				},
			},
			"glyphs": map[string]any{
				"type": "array",
				"items": obj(map[string]any{
					"codepoint": str("Unicode codepoint as hex, for example E000"),
					"advance":   integer("Glyph advance width in font units", 100, 4000),
					"layers": map[string]any{
						"type": "array",
						"items": obj(map[string]any{
							"shape": enum("Primitive shape", "circle", "square", "triangle", "star", "ring"),
							"color": integer("Palette entry index", 0, 63),
							"scale": num("Shape scale around glyph center", 0.1, 2),
							"x":     num("Horizontal offset from glyph center", -400, 400),
							"y":     num("Vertical offset from glyph center", -400, 400),
						}, "shape", "color"),
					},
				}, "codepoint", "layers"),
			},
		}, "palettes", "glyphs"),
	})

	output := obj(map[string]any{
		"formats": map[string]any{
			"type":  "array",
			"items": enum("Output format", "ttf", "woff", "woff2"),
		},
		"targets": map[string]any{
			"type":  "array",
			"items": enum("Platform target", "universal", "web", "system", "unity", "unreal", "apps", "bmfont", "all"),
		},
		"specimen": boolean("Generate the HTML specimen"),
		"open":     boolean("Open the specimen when the build completes"),
		"variable": boolean("Emit a wght variable axis with fvar/gvar for Latin and symbol glyphs"),
		"cursive":  boolean("Emit GPOS cursive attachment anchors for lowercase letters"),
		"bmfont": obj(map[string]any{
			"size":      integer("BMFont pixel size of the em square", 8, 256),
			"padding":   integer("BMFont per-glyph atlas padding in pixels", 0, 16),
			"atlasSize": integer("BMFont atlas page size in pixels", 64, 4096),
			"charset":   str("BMFont charset: latin, latin-ext, all, or an explicit string"),
		}),
		"splitCJK": integer("Split CJK/Hangul into chunks of this many codepoints; 0 disables", 0, 100000),
		"tmp": obj(map[string]any{
			"population":        enum("TMP atlas population mode", "dynamic", "static"),
			"samplingPointSize": integer("TMP sampling point size", 8, 200),
			"padding":           integer("TMP atlas padding", 0, 64),
			"atlasWidth":        integer("TMP atlas width (power of two)", 256, 4096),
			"atlasHeight":       integer("TMP atlas height (power of two)", 256, 4096),
			"multiAtlas":        boolean("Enable TMP multi-atlas"),
			"renderMode":        enum("TMP render mode", "SDFAA", "SDF8", "SDF16", "SDF32"),
		}, "population", "samplingPointSize", "padding", "atlasWidth", "atlasHeight", "multiAtlas", "renderMode"),
	}, "formats", "targets", "specimen", "open", "variable", "cursive", "bmfont", "splitCJK", "tmp")

	s := obj(map[string]any{
		"apiVersion": map[string]any{"type": "string", "const": "fontcreator233/v1"},
		"meta":       meta,
		"metrics":    metrics,
		"style":      style,
		"coverage":   coverage,
		"glyphs":     glyphs,
		"openType":   openType,
		"output":     output,
	}, "apiVersion", "meta", "metrics", "style", "coverage", "glyphs", "output")
	s["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	s["title"] = "fontcreator233 recipe"
	return s
}

func toAny(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
