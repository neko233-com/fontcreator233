package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/recipe"
	"golang.org/x/image/font/sfnt"
)

func TestE2EThreeStyles(t *testing.T) {
	if testing.Short() {
		t.Skip("E2E builds common coverage and all platform packages")
	}
	paths := []string{
		"../../examples/aurora-grotesk.recipe.yaml",
		"../../examples/noir-didone.recipe.yaml",
		"../../examples/botanic-display.recipe.yaml",
	}
	seen := map[string]bool{}
	var results []Result
	for _, path := range paths {
		r, err := recipe.Load(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		out := t.TempDir()
		res, err := Build(context.Background(), r, Options{
			OutDir:  out,
			Targets: []string{"all"},
			Jobs:    4,
			NoOpen:  true,
		}, nil)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		results = append(results, res)
		if seen[res.SHA256] {
			t.Fatalf("duplicate font hash for %s", path)
		}
		seen[res.SHA256] = true
		if res.Glyphs < 30_000 || len(res.GlyphRunes) < 30_000 {
			t.Fatalf("%s: glyphs = %d", path, res.Glyphs)
		}
		if res.CoverageReport["cjk"].Found < 20_976 {
			t.Fatalf("%s: cjk coverage = %#v", path, res.CoverageReport["cjk"])
		}
		if !warnContains(res.Warnings, "base coverage cjk") {
			t.Fatalf("%s: missing CJK warning: %v", path, res.Warnings)
		}
		data, err := os.ReadFile(res.FontPath)
		if err != nil {
			t.Fatal(err)
		}
		f, err := sfnt.Parse(data)
		if err != nil {
			t.Fatalf("%s parse: %v", path, err)
		}
		var buf sfnt.Buffer
		for _, rn := range []rune{'A', 'a', '0', 'é', 'ñ', 'Ω', 'Ж', '中', 'あ', '한', '€', '→', '“', '”'} {
			gid, err := f.GlyphIndex(&buf, rn)
			if err != nil || gid == 0 {
				t.Fatalf("%s: missing %U (gid=%d err=%v)", path, rn, gid, err)
			}
		}
		if res.SpecimenPath == "" {
			t.Fatalf("%s: no specimen", path)
		}
		for _, rel := range []string{
			"fontcreator233.manifest.json",
			"web/font.css",
			"system/INSTALL.md",
			"unity/package.json",
			"unity/Editor/FontCreator233TMPImporter.cs",
			"unity/Fonts/" + filepath.Base(res.FontPath),
			"unity/Charsets/all.txt",
			"unreal/Scripts/fontcreator233_import.py",
			"unreal/fontcreator233.manifest.json",
			"apps/README.md",
			"apps/font-usage.json",
			"apps/photoshop.md",
			"apps/capcut.md",
			"apps/caption.srt",
			"apps/caption.ass",
		} {
			if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
				t.Fatalf("%s: missing %s: %v", path, rel, err)
			}
		}
		css, _ := os.ReadFile(filepath.Join(out, "web", "font.css"))
		if !strings.Contains(string(css), "@font-face") || !strings.Contains(string(css), "font-display: swap") {
			t.Fatalf("%s: css = %s", path, css)
		}
		if len(res.Targets["unity"]) == 0 || len(res.Targets["unreal"]) == 0 || len(res.Targets["apps"]) == 0 {
			t.Fatalf("%s: targets = %#v", path, res.Targets)
		}
	}
	if results[0].Style != "Regular" || results[1].Style != "Noir Italic" || results[2].Style != "Regular" {
		t.Fatalf("styles = %q %q %q", results[0].Style, results[1].Style, results[2].Style)
	}
	if results[1].Coverage["cjk"] == 0 {
		t.Fatalf("noir coverage = %#v", results[1].Coverage)
	}
}

func warnContains(warnings []string, sub string) bool {
	for _, w := range warnings {
		if strings.Contains(w, sub) {
			return true
		}
	}
	return false
}
