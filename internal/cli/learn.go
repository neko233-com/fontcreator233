package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/neko233-com/fontcreator233/internal/fontlearn"
	"github.com/neko233-com/fontcreator233/internal/importexport"
)

func (a *app) cmdLearn(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("learn", &common)
	out := fs.String("out", "learned", "output directory for the report and starter recipe")
	glyphs := fs.Bool("glyphs", false, "also extract SVG glyphs and wire them into recipe.glyphs.overrides")
	force := fs.Bool("force", false, "allow a non-empty output directory")
	values := map[string]bool{"out": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("learn", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("learn", 1, fmt.Errorf("usage: fontcreator learn <font.ttf|font.otf> --out <dir> [--glyphs]"))
	}
	path := fs.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		return a.fail("learn", 1, err)
	}
	report, err := fontlearn.Analyze(filepath.Base(path), data)
	if err != nil {
		return a.fail("learn", 1, err)
	}
	profile, err := importexport.ImportFont(path, *out, *glyphs, nil, *force)
	if err != nil {
		return a.fail("learn", 1, err)
	}
	report.Warnings = append(report.Warnings, profile.Warnings...)
	reportData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return a.fail("learn", 1, err)
	}
	reportPath := filepath.Join(*out, "font-report.json")
	if err := os.WriteFile(reportPath, append(reportData, '\n'), 0o644); err != nil {
		return a.fail("learn", 1, err)
	}
	mdPath := filepath.Join(*out, "font-report.md")
	if err := os.WriteFile(mdPath, []byte(report.Markdown()), 0o644); err != nil {
		return a.fail("learn", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("learn", map[string]any{
			"report":   report,
			"json":     reportPath,
			"markdown": mdPath,
			"profile":  profile,
			"recipe":   profile.RecipePath,
		}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "learned %s %s · %d glyphs · %d features · %d kern pairs\n", report.Family, report.Style, report.Glyphs, len(report.Features), report.Kerning)
	fmt.Fprintf(a.io.Out, "  report: %s\n", mdPath)
	fmt.Fprintf(a.io.Out, "  json:   %s\n", reportPath)
	fmt.Fprintf(a.io.Out, "  recipe: %s\n", profile.RecipePath)
	for _, w := range report.Warnings {
		fmt.Fprintf(a.io.Err, "warning: %s\n", w)
	}
	return 0
}
