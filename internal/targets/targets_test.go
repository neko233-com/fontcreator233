package targets

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/integration"
	"github.com/neko233-com/fontcreator233/internal/recipe"
)

func TestBuildWebSystemManifest(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "Test-Regular.ttf")
	woff := filepath.Join(dir, "Test-Regular.woff")
	split := filepath.Join(dir, "Test-CJK-01.ttf")
	for path, data := range map[string]string{
		primary: "primary",
		woff:    "woff",
		split:   "split",
	} {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tmp := recipe.Default().Output.TMP
	paths, err := Build(context.Background(), Options{
		OutDir:       dir,
		Family:       "Test",
		Style:        "Regular",
		License:      "OFL-1.1",
		Weight:       400,
		Targets:      []string{"universal", "web", "system"},
		SpecimenPath: "",
		TMP:          tmp,
		Fonts: []FontInput{
			{Path: primary, WOFF: woff, Primary: true, Style: "Regular", Runes: []rune{'A', 'é', 'Ω', 'Ж', '€'}},
			{Path: split, Style: "Regular CJK 01", Runes: []rune{'中', '文'}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths["web"]) == 0 || len(paths["system"]) == 0 {
		t.Fatalf("paths = %#v", paths)
	}
	manifestPath := filepath.Join(dir, "fontcreator233.manifest.json")
	m, err := integration.ReadJSON(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if m.Schema != "fontcreator233/integration/v1" || len(m.Fonts) != 2 {
		t.Fatalf("manifest = %#v", m)
	}
	if m.Fallbacks[0] != "Test-Regular.ttf" || len(m.UnrealComposite.SubFonts) != 1 {
		t.Fatalf("fallbacks = %#v", m)
	}
	for _, charset := range []string{"Charsets/latin.txt", "Charsets/greek.txt", "Charsets/cyrillic.txt", "Charsets/cjk.txt", "Charsets/all.txt"} {
		if _, err := os.Stat(filepath.Join(dir, charset)); err != nil {
			t.Fatalf("charset %s: %v", charset, err)
		}
	}
	css, err := os.ReadFile(filepath.Join(dir, "web", "font.css"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(css)
	if strings.Count(text, "@font-face") != 2 {
		t.Fatalf("css = %s", text)
	}
	if !strings.Contains(text, "unicode-range: U+4E2D, U+6587") {
		t.Fatalf("css unicode-range missing: %s", text)
	}
	if strings.Count(text, "U+4E2D") != 1 {
		t.Fatalf("primary web font still claims split CJK: %s", text)
	}
	if !strings.Contains(text, `url("./Test-Regular.woff") format("woff")`) {
		t.Fatalf("css woff missing: %s", text)
	}
	for _, p := range []string{"web/Test-Regular.ttf", "web/Test-CJK-01.ttf", "system/Test-Regular.ttf", "system/INSTALL.md", "OFL.txt"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
}

func TestExpandTargets(t *testing.T) {
	all, err := expandTargets([]string{"all"})
	if err != nil || len(all) != 5 {
		t.Fatalf("all = %v err=%v", all, err)
	}
	if _, err := expandTargets([]string{"nope"}); err == nil {
		t.Fatal("expected unknown target error")
	}
}
