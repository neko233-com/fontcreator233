package cli

import (
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/importexport"
)

func (a *app) cmdImport(args []string) int {
	if len(args) == 0 {
		return a.fail("import", 1, fmt.Errorf("usage: fontcreator import font|bundle ..."))
	}
	switch args[0] {
	case "font":
		return a.cmdImportFont(args[1:])
	case "bundle":
		return a.cmdImportBundle(args[1:])
	default:
		return a.fail("import", 1, fmt.Errorf("unknown import type %q", args[0]))
	}
}

func (a *app) cmdImportFont(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("import font", &common)
	out := fs.String("out", "imported", "output directory")
	extract := fs.Bool("extract-glyphs", false, "export glyph SVGs and set glyphs.overrides")
	unicodes := fs.String("unicodes", "", "optional hex ranges, e.g. 0041-007A,4E00-9FFF")
	values := map[string]bool{"out": true, "unicodes": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("import font", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("import font", 1, fmt.Errorf("usage: fontcreator import font <font.ttf> --out <dir>"))
	}
	filter, err := importexport.ParseUnicodeSpec(*unicodes)
	if err != nil {
		return a.fail("import font", 1, err)
	}
	profile, err := importexport.ImportFont(fs.Arg(0), *out, *extract, filter)
	if err != nil {
		return a.fail("import font", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("import font", map[string]any{
			"profile": profile,
			"recipe":  profile.RecipePath,
			"glyphs":  profile.GlyphDir,
		}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "profile: %s\n", profile.RecipePath)
	fmt.Fprintf(a.io.Out, "  %s %s · %d glyphs · weight %d · italic %.1f\n", profile.Family, profile.Style, profile.Glyphs, profile.WeightClass, profile.ItalicAngle)
	if profile.GlyphDir != "" {
		fmt.Fprintf(a.io.Out, "  glyphs: %s (%d extracted)\n", profile.GlyphDir, len(profile.Extracted))
	}
	return 0
}

func (a *app) cmdImportBundle(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("import bundle", &common)
	out := fs.String("out", "work/imported", "output directory")
	buildAfter := fs.Bool("build", false, "rebuild the recipe after import")
	force := fs.Bool("force", false, "allow a non-empty output directory")
	values := map[string]bool{"out": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("import bundle", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("import bundle", 1, fmt.Errorf("usage: fontcreator import bundle <bundle.zip> --out <dir>"))
	}
	res, err := importexport.ImportBundle(fs.Arg(0), *out, *buildAfter, *force)
	if err != nil {
		return a.fail("import bundle", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("import bundle", map[string]any{"result": res}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "imported %d files to %s (built=%v)\n", len(res.Files), res.Dir, res.Built)
	return 0
}
