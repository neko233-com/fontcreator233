package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/neko233-com/fontcreator233/internal/importexport"
	"github.com/neko233-com/fontcreator233/internal/recipe"
)

func (a *app) cmdExport(args []string) int {
	if len(args) == 0 {
		return a.fail("export", 1, fmt.Errorf("usage: fontcreator export glyphs|bundle ..."))
	}
	switch args[0] {
	case "glyphs":
		return a.cmdExportGlyphs(args[1:])
	case "bundle":
		return a.cmdExportBundle(args[1:])
	default:
		return a.fail("export", 1, fmt.Errorf("unknown export type %q", args[0]))
	}
}

func (a *app) cmdExportGlyphs(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("export glyphs", &common)
	dir := fs.String("dir", "glyphs", "output directory")
	recipePath := fs.String("recipe", "", "export glyphs from a recipe instead of a font")
	unicodes := fs.String("unicodes", "", "hex ranges, e.g. 0041-007A,4E00-9FFF")
	var scripts stringList
	fs.Var(&scripts, "scripts", "limit to latin|greek|cyrillic|cjk|kana|hangul|symbols")
	limit := fs.Int("limit", 0, "maximum number of glyphs")
	force := fs.Bool("force", false, "overwrite existing SVG files")
	values := map[string]bool{"dir": true, "recipe": true, "unicodes": true, "scripts": true, "limit": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("export glyphs", 1, err)
	}
	a.json = common.json
	filter := importexport.Filter{Scripts: scripts, Limit: *limit, Force: *force}
	var err error
	filter.Unicodes, err = importexport.ParseUnicodeSpec(*unicodes)
	if err != nil {
		return a.fail("export glyphs", 1, err)
	}
	var paths []string
	if *recipePath != "" {
		paths, err = importexport.ExportGlyphsFromRecipe(*recipePath, *dir, filter)
	} else {
		if fs.NArg() != 1 {
			return a.fail("export glyphs", 1, fmt.Errorf("usage: fontcreator export glyphs <font.ttf> --dir <dir>"))
		}
		paths, err = importexport.ExportGlyphsFromFont(fs.Arg(0), *dir, filter)
	}
	if err != nil {
		return a.fail("export glyphs", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("export glyphs", map[string]any{"dir": *dir, "count": len(paths), "files": paths}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "exported %d glyphs to %s\n", len(paths), *dir)
	return 0
}

func (a *app) cmdExportBundle(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("export bundle", &common)
	out := fs.String("out", "", "output zip path")
	force := fs.Bool("force", false, "overwrite an existing zip")
	var targets stringList
	fs.Var(&targets, "target", "universal|web|system|unity|unreal|all")
	values := map[string]bool{"out": true, "target": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("export bundle", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("export bundle", 1, fmt.Errorf("usage: fontcreator export bundle <recipe.yaml> --out <file.zip>"))
	}
	recipePath := fs.Arg(0)
	zipPath := *out
	if zipPath == "" {
		r, err := recipe.Load(recipePath)
		if err != nil {
			return a.fail("export bundle", 1, err)
		}
		stem := fontStemFromFamily(r.Meta.Family)
		zipPath = stem + ".bundle.zip"
	}
	if !*force {
		if _, err := os.Stat(zipPath); err == nil {
			return a.fail("export bundle", 1, fmt.Errorf("refusing to overwrite %s (use --force)", zipPath))
		}
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := importexport.ExportBundle(ctx, recipePath, zipPath, targets)
	if err != nil {
		return a.fail("export bundle", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("export bundle", map[string]any{"result": res}))
		return 0
	}
	abs, _ := filepath.Abs(res.Zip)
	fmt.Fprintf(a.io.Out, "bundle: %s (%d files, %d bytes)\n", abs, res.Files, res.Bytes)
	return 0
}

func fontStemFromFamily(family string) string {
	var b []rune
	for _, r := range family {
		if r == ' ' || r == '/' || r == '\\' || r == ':' {
			continue
		}
		b = append(b, r)
	}
	if len(b) == 0 {
		return "font"
	}
	return string(b)
}
