package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/skeleton"
	"github.com/neko233-com/fontcreator233/internal/style"
	"github.com/neko233-com/fontcreator233/internal/svgout"
	"github.com/neko233-com/fontcreator233/internal/unicodes"
)

func (a *app) cmdLibrary(args []string) int {
	if len(args) == 0 {
		return a.fail("library", 1, fmt.Errorf("usage: fontcreator library list|export"))
	}
	switch args[0] {
	case "list":
		return a.cmdLibraryList(args[1:])
	case "export":
		return a.cmdLibraryExport(args[1:])
	default:
		return a.fail("library", 1, fmt.Errorf("unknown library command %q", args[0]))
	}
}

func (a *app) cmdLibraryList(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("library list", &common)
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("library list", 1, err)
	}
	a.json = common.json
	counts := map[string]int{}
	runes := skeleton.Available()
	for _, r := range runes {
		counts[unicodes.ScriptOf(r)]++
	}
	if a.json {
		a.writeJSON(okMap("library list", map[string]any{"total": len(runes), "scripts": counts}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "built-in skeleton glyphs: %d\n", len(runes))
	for _, s := range []string{"latin", "greek", "cyrillic", "other"} {
		if counts[s] > 0 {
			fmt.Fprintf(a.io.Out, "  %-9s %d\n", s, counts[s])
		}
	}
	return 0
}

func (a *app) cmdLibraryExport(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("library export", &common)
	dir := fs.String("dir", "library", "output directory")
	recipePath := fs.String("recipe", "", "optional recipe for style")
	force := fs.Bool("force", false, "overwrite existing SVG files")
	values := map[string]bool{"dir": true, "recipe": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("library export", 1, err)
	}
	a.json = common.json
	r := recipe.Default()
	r.Meta.Family = "fontcreator233 Library"
	if *recipePath != "" {
		loaded, err := recipe.Load(*recipePath)
		if err != nil {
			return a.fail("library export", 1, err)
		}
		r = loaded
	}
	if entries, err := os.ReadDir(*dir); err == nil && len(entries) > 0 && !*force {
		return a.fail("library export", 1, fmt.Errorf("output directory %s is not empty (use --force)", *dir))
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return a.fail("library export", 1, err)
	}
	b := skeleton.NewBuilder(style.Resolve(r))
	count := 0
	for _, rn := range skeleton.Available() {
		p, adv, ok, err := b.Glyph(rn)
		if err != nil {
			return a.fail("library export", 1, err)
		}
		if !ok || len(p) == 0 {
			continue
		}
		name := fmt.Sprintf("U+%04X.svg", rn)
		path := filepath.Join(*dir, name)
		f, err := os.Create(path)
		if err != nil {
			return a.fail("library export", 1, err)
		}
		err = svgout.WriteGlyph(f, fmt.Sprintf("%s U+%04X", r.Meta.Family, rn), int(adv), p)
		_ = f.Close()
		if err != nil {
			return a.fail("library export", 1, err)
		}
		count++
	}
	if a.json {
		a.writeJSON(okMap("library export", map[string]any{"dir": *dir, "count": count}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "exported %d SVG glyphs to %s\n", count, *dir)
	return 0
}
