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

func TestUnrealTarget(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "Aurora-Regular.ttf")
	split := filepath.Join(dir, "Aurora-CJK-01.ttf")
	for _, p := range []string{primary, split} {
		if err := os.WriteFile(p, []byte("font"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := Build(context.Background(), Options{
		OutDir:  dir,
		Family:  "Aurora Grotesk",
		Style:   "Regular",
		Version: "1.0.0",
		License: "OFL-1.1",
		Weight:  400,
		Targets: []string{"unreal"},
		TMP:     recipe.Default().Output.TMP,
		Fonts: []FontInput{
			{Path: primary, Primary: true, Runes: []rune{'A', '中'}},
			{Path: split, Runes: []rune{'文', 'あ'}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths["unreal"]) == 0 {
		t.Fatal("no unreal paths")
	}
	unrealDir := filepath.Join(dir, "unreal")
	m, err := integration.ReadJSON(filepath.Join(unrealDir, "fontcreator233.manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m.UnrealComposite.Default != "Fonts/Aurora-Regular.ttf" || len(m.UnrealComposite.SubFonts) != 1 {
		t.Fatalf("composite = %#v", m.UnrealComposite)
	}
	if m.UnrealComposite.SubFonts[0].Path != "Fonts/Aurora-CJK-01.ttf" {
		t.Fatalf("subfont = %#v", m.UnrealComposite.SubFonts[0])
	}
	script, err := os.ReadFile(filepath.Join(unrealDir, "Scripts", "fontcreator233_import.py"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(script)
	for _, want := range []string{"AssetImportTask", "FontFactory", "CompositeFont", "unrealComposite", "ranges"} {
		if !strings.Contains(text, want) {
			t.Fatalf("python missing %q", want)
		}
	}
	readme, _ := os.ReadFile(filepath.Join(unrealDir, "README.md"))
	if !strings.Contains(string(readme), "CompositeFont") {
		t.Fatal("README missing CompositeFont")
	}
	for _, p := range []string{"Fonts/Aurora-Regular.ttf", "Fonts/Aurora-CJK-01.ttf", "Charsets/cjk.txt", "OFL.txt"} {
		if _, err := os.Stat(filepath.Join(unrealDir, p)); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
}
