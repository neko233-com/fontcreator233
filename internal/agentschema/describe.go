// Package agentschema exposes the machine-readable contract agents use.
package agentschema

import (
	"sort"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/derive"
	"github.com/neko233-com/fontcreator233/internal/recipe"
)

// Version is the public CLI contract version.
const Version = "0.1.0"

// Describe returns the complete JSON-friendly contract.
func Describe() map[string]any {
	return map[string]any{
		"name":        "fontcreator233",
		"version":     Version,
		"apiVersion":  "fontcreator233/v1",
		"description": "Deterministic offline font creator for agents. The agent reads reference art; the CLI only turns a recipe into fonts and platform packages.",
		"commands": []map[string]any{
			{"name": "build", "summary": "Build one recipe into TTF/WOFF/specimen/platform packages."},
			{"name": "build-all", "summary": "Build several recipes in parallel."},
			{"name": "validate", "summary": "Validate a recipe without building."},
			{"name": "inspect", "summary": "Inspect a compiled TTF/OTF."},
			{"name": "specimen", "summary": "Regenerate an offline HTML specimen for an existing font."},
			{"name": "init", "summary": "Write a starter recipe (geometric|serif|display)."},
			{"name": "describe", "summary": "Print this contract as JSON or Markdown."},
			{"name": "schema", "summary": "Print the recipe JSON Schema."},
			{"name": "library list", "summary": "List built-in skeleton coverage."},
			{"name": "library export", "summary": "Export built-in skeleton outlines as SVG."},
			{"name": "clean", "summary": "Remove state-tracked generated files safely."},
			{"name": "status", "summary": "Show workspace state, stale files, and reclaimable space."},
			{"name": "import font", "summary": "Profile a TTF/OTF and optionally extract glyph SVGs."},
			{"name": "import bundle", "summary": "Restore a fontcreator233 bundle zip."},
			{"name": "export glyphs", "summary": "Export glyph outlines from a font or recipe."},
			{"name": "export bundle", "summary": "Package a recipe, fonts, specimen, and targets as a zip."},
			{"name": "mcp", "summary": "Run the MCP stdio server."},
			{"name": "version", "summary": "Print the build version."},
		},
		"recipeSchema": recipe.Schema(),
		"styleGuide": map[string]any{
			"weight":    "100 very thin, 400 regular, 620 semibold, 900 black. Maps to stem width and OS/2 usWeightClass.",
			"width":     "75 condensed, 100 normal, 125 extended. Scales horizontal outlines and advances.",
			"slant":     "Positive leans right. Non-zero resolves a non-italic style name to Italic and sets post.italicAngle = -slant.",
			"contrast":  "0 uniform geometric strokes; 0.4 humanist; 0.85+ Didone/hairline. Modulates stroke width by tangent angle.",
			"serif":     "none|slab|bracketed|hairline. Applied to vertical stem endpoints.",
			"terminal":  "flat|round|angled|flare. Applied to open stroke endpoints; flare widens near the ends.",
			"tracking":  "Extra advance in percent of em, -5..10. Use 1-2% for flared/display faces.",
			"roughness": "0 clean; 0.15 hand-drawn; 0.5 loose brush. Deterministic via seed.",
			"seed":      "Any int64. Same seed + recipe + tool version = same bytes on the same platform.",
			"skeleton":  "geometric|humanist|serif; affects letterform variants (a/g) and style defaults.",
		},
		"referenceImageGuide": map[string]any{
			"thin_strokes":      "weight 200-350, contrast 0-0.15",
			"regular_strokes":   "weight 400-520, contrast 0.1-0.3",
			"bold_strokes":      "weight 620-780, contrast 0.2-0.5",
			"uniform_strokes":   "contrast 0-0.1, skeleton geometric",
			"high_contrast":     "contrast 0.7-0.95, serif hairline, weight 450-600",
			"slab_serifs":       "serif slab, serifSize 0.5-0.8",
			"organic_terminals": "terminal flare, contrast 0.4-0.7, roughness 0.1-0.3",
			"condensed_poster":  "width 82-92, weight 600-800",
			"wide_display":      "width 108-122, weight 500-700",
			"italic_reference":  "slant 6-12",
		},
		"coverage": map[string]any{
			"levels":   derive.CoverageNames(),
			"scripts":  []string{"latin", "greek", "cyrillic", "cjk", "kana", "hangul", "symbols"},
			"baseFont": "Embedded static OFL instances: fontcreator233 Base SC (CJK/kana/symbols) and Base KR (Hangul).",
			"note":     "Counts are actual cmap coverage, not a promise of every Unicode block.",
		},
		"platforms": map[string]any{
			"universal": "TTF + WOFF + HTML specimen + manifest",
			"web":       "font.css with local font files and unicode-range for split fonts",
			"system":    "TTF + install instructions",
			"unity":     "UPM package + Editor TMP importer + charsets + manifest (generation only, no Unity test)",
			"unreal":    "TTF + Editor Python import script + CompositeFont ranges",
		},
		"limits": map[string]any{
			"unitsPerEm": 1000,
			"maxGlyphs":  65535,
			"unsupportedScripts": []string{
				"Arabic", "Hebrew", "Indic scripts", "Thai", "vertical layout", "complex shaping",
			},
			"woff2":       "not generated; WOFF and TTF are generated",
			"determinism": "byte-identical within one fontcreator233 version, OS/arch, and Go version; jobs=1 and jobs=N match on the same platform",
		},
		"workflow": []string{
			"Analyze the reference image with your own vision; write a short style brief.",
			"Run `fontcreator init --template geometric --dir work` or write a recipe.",
			"Run `fontcreator validate work/font.recipe.yaml --json` and fix errors.",
			"Run `fontcreator build work/font.recipe.yaml --out dist --json --no-open`.",
			"Run `fontcreator specimen dist/<font>.ttf --open` for human review.",
			"Run `fontcreator inspect dist/<font>.ttf --json` and check coverage/warnings.",
			"Iterate at most 3 times, then report the recipe, font, specimen, and known limits.",
		},
	}
}

// Markdown renders the contract as a readable agent guide.
func Markdown() string {
	var b strings.Builder
	b.WriteString("# fontcreator233 agent contract\n\n")
	b.WriteString("Version " + Version + " · apiVersion fontcreator233/v1\n\n")
	b.WriteString("The agent owns visual understanding; the CLI owns deterministic synthesis.\n\n")
	b.WriteString("## Workflow\n\n")
	for _, s := range Describe()["workflow"].([]string) {
		b.WriteString("1. " + s + "\n")
	}
	b.WriteString("\n## Style mapping\n\n")
	guide := Describe()["styleGuide"].(map[string]any)
	keys := make([]string, 0, len(guide))
	for k := range guide {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString("- **" + k + "**: " + guide[k].(string) + "\n")
	}
	b.WriteString("\n## Commands\n\n")
	for _, c := range Describe()["commands"].([]map[string]any) {
		b.WriteString("- `" + c["name"].(string) + "` — " + c["summary"].(string) + "\n")
	}
	b.WriteString("\n## Limits\n\n")
	b.WriteString("- unitsPerEm: 1000\n- No Arabic/Hebrew/Indic/Thai/complex shaping\n- WOFF2 is not generated\n- Unity/UE packages are generated, not tested against their Editors\n")
	b.WriteString("\nRun `fontcreator schema` for the full recipe JSON Schema.\n")
	return b.String()
}
