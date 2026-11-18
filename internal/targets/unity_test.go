package targets

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/integration"
	"github.com/neko233-com/fontcreator233/internal/recipe"
)

func TestUnityTarget(t *testing.T) {
	dir := t.TempDir()
	primary := filepath.Join(dir, "Aurora-Regular.ttf")
	split := filepath.Join(dir, "Aurora-CJK-01.ttf")
	for _, p := range []string{primary, split} {
		if err := os.WriteFile(p, []byte("font"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	shot := filepath.Join(dir, "Aurora-Regular.html")
	if err := os.WriteFile(shot, []byte("html"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := Build(context.Background(), Options{
		OutDir:         dir,
		Family:         "Aurora Grotesk",
		Style:          "Regular",
		RequestedStyle: "Regular",
		Version:        "1.0.0",
		License:        "OFL-1.1",
		Weight:         400,
		Targets:        []string{"unity"},
		SpecimenPath:   shot,
		TMP:            recipe.Default().Output.TMP,
		Fonts: []FontInput{
			{Path: primary, Primary: true, Runes: []rune{'A', '中', '한'}},
			{Path: split, Runes: []rune{'文', 'あ'}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths["unity"]) == 0 {
		t.Fatal("no unity paths")
	}
	unity := filepath.Join(dir, "unity")
	pkgRaw, err := os.ReadFile(filepath.Join(unity, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pkg map[string]any
	if err := json.Unmarshal(pkgRaw, &pkg); err != nil {
		t.Fatal(err)
	}
	if pkg["name"] != "com.neko233.fontcreator233.generated.aurora-grotesk" {
		t.Fatalf("package = %#v", pkg)
	}
	if pkg["unity"] != "2022.3" || pkg["version"] != "1.0.0" {
		t.Fatalf("package = %#v", pkg)
	}
	m, err := integration.ReadJSON(filepath.Join(unity, "fontcreator233.manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Fonts) != 2 || m.Fonts[0].Path != "Fonts/Aurora-Regular.ttf" {
		t.Fatalf("manifest = %#v", m)
	}
	if len(m.Fallbacks) != 2 || m.Fallbacks[1] != "Fonts/Aurora-CJK-01.ttf" {
		t.Fatalf("fallbacks = %#v", m.Fallbacks)
	}
	if m.Fonts[0].Charset != "Charsets/base.txt" || m.Fonts[1].Charset != "Charsets/cjk-01.txt" {
		t.Fatalf("charsets = %#v", m.Fonts)
	}
	for _, p := range []string{
		"Editor/FontCreator233TMPImporter.cs",
		"Fonts/Aurora-Regular.ttf",
		"Fonts/Aurora-CJK-01.ttf",
		"Charsets/all.txt",
		"Charsets/base.txt",
		"Charsets/cjk-01.txt",
		"Charsets/cjk.txt",
		"Charsets/hangul.txt",
		"OFL.txt",
	} {
		if _, err := os.Stat(filepath.Join(unity, p)); err != nil {
			t.Fatalf("missing %s: %v", p, err)
		}
	}
	cs, err := os.ReadFile(filepath.Join(unity, "Editor/FontCreator233TMPImporter.cs"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(cs)
	for _, want := range []string{
		"Tools/Fontcreator233/Create TMP Font Assets",
		"TMP_FontAsset.CreateFontAsset",
		"AtlasPopulationMode.Dynamic",
		"AtlasPopulationMode.Static",
		"fallbackFontAssetTable",
		"TryAddCharacters",
		"HasCharacter",
		"GlyphRenderMode.SDFAA",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("C# missing %q", want)
		}
	}
	if strings.Contains(text, "UNITY_INCLUDE_TESTS") {
		t.Fatal("unexpected test code")
	}
}
