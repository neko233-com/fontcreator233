package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func (a *app) cmdInit(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("init", &common)
	dir := fs.String("dir", ".", "output directory")
	template := fs.String("template", "geometric", "geometric|serif|display")
	family := fs.String("family", "", "font family name")
	force := fs.Bool("force", false, "overwrite an existing recipe")
	values := map[string]bool{"dir": true, "template": true, "family": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("init", 1, err)
	}
	a.json = common.json
	if *family == "" {
		switch *template {
		case "serif":
			*family = "Untitled Serif"
		case "display":
			*family = "Untitled Display"
		default:
			*family = "Untitled Grotesk"
		}
	}
	text, err := recipeTemplate(*template, *family)
	if err != nil {
		return a.fail("init", 1, err)
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return a.fail("init", 1, err)
	}
	path := filepath.Join(*dir, "font.recipe.yaml")
	if _, err := os.Stat(path); err == nil && !*force {
		return a.fail("init", 1, fmt.Errorf("%s exists; use --force to overwrite", path))
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return a.fail("init", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("init", map[string]any{"path": path, "template": *template, "family": *family}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "wrote %s (%s)\n", path, *template)
	return 0
}

func recipeTemplate(template, family string) (string, error) {
	style := `skeleton: geometric
  weight: 400
  width: 100
  slant: 0
  contrast: 0.10
  terminal: round
  serif: none
  serifSize: 0
  overshoot: 0.012
  roughness: 0
  jitterRotation: 0
  seed: 233`
	switch template {
	case "geometric":
	case "serif":
		style = `skeleton: serif
  weight: 500
  width: 98
  slant: 0
  contrast: 0.55
  terminal: angled
  serif: bracketed
  serifSize: 0.55
  overshoot: 0.012
  roughness: 0
  jitterRotation: 0
  seed: 233`
	case "display":
		style = `skeleton: geometric
  weight: 700
  width: 105
  slant: 0
  contrast: 0.30
  terminal: flare
  serif: none
  serifSize: 0
  overshoot: 0.016
  roughness: 0.10
  jitterRotation: 0.3
  seed: 233`
	default:
		return "", fmt.Errorf("unknown template %q (want geometric|serif|display)", template)
	}
	return fmt.Sprintf(`# fontcreator233 recipe
# Run: fontcreator build this-file.yaml --target all --no-open
apiVersion: fontcreator233/v1

meta:
  family: %q
  style: Regular
  version: 0.9.0
  designer: ""
  description: "Describe the reference and the intended mood."
  license: OFL-1.1
  accent: "#5B8CFF"
  sampleText: "Hamburgefonstiv 字体创造器 0123"

metrics:
  ascender: 800
  descender: -200
  xHeight: 0.52
  capHeight: 0.72
  lineGap: 0

style:
  %s

coverage:
  generated: [latin, greek, cyrillic]
  base:
    cjk: common
    kana: true
    hangul: true
    symbols: true

glyphs:
  overrides: ""

output:
  formats: [ttf, woff]
  targets: [universal, web, system, unity, apps]
  specimen: true
  open: true
  variable: true
  cursive: true
  splitCJK: 0
  tmp:
    population: dynamic
    samplingPointSize: 90
    padding: 9
    atlasWidth: 1024
    atlasHeight: 1024
    multiAtlas: true
    renderMode: SDFAA
`, family, style), nil
}
