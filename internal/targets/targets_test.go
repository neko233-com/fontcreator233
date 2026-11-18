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
	if m.LicenseFile != "OFL.txt" {
		t.Fatalf("licenseFile = %q", m.LicenseFile)
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

func TestBuildApps(t *testing.T) {
	dir := t.TempDir()
	font := filepath.Join(dir, "Rock-Roll-Regular.ttf")
	if err := os.WriteFile(font, []byte("font"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := Build(context.Background(), Options{
		OutDir:  dir,
		Family:  "Rock & Roll",
		Style:   "Regular",
		License: "OFL-1.1",
		Weight:  400,
		Targets: []string{"apps"},
		Fonts:   []FontInput{{Path: font, Primary: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths["apps"]) < 10 {
		t.Fatalf("apps paths = %v", paths["apps"])
	}
	for _, rel := range []string{
		"apps/README.md", "apps/font-usage.json", "apps/photoshop.md",
		"apps/capcut.md", "apps/premiere.md", "apps/davinci.md",
		"apps/figma.md", "apps/blender.md", "apps/caption.srt", "apps/caption.ass",
		"apps/font/Rock-Roll-Regular.ttf",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	readme, _ := os.ReadFile(filepath.Join(dir, "apps", "README.md"))
	for _, want := range []string{"Photoshop", "剪映 / CapCut", "Premiere Pro", "Blender", "caption.srt"} {
		if !strings.Contains(string(readme), want) {
			t.Fatalf("README missing %q", want)
		}
	}
	ass, _ := os.ReadFile(filepath.Join(dir, "apps", "caption.ass"))
	if !strings.Contains(string(ass), "Rock & Roll") {
		t.Fatalf("ASS font name missing: %s", ass)
	}
	m, err := integration.ReadJSON(filepath.Join(dir, "fontcreator233.manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if m.Apps == nil || m.Apps.Usage != "apps/font-usage.json" || len(m.Apps.Guides) < 10 {
		t.Fatalf("manifest apps = %#v", m.Apps)
	}
	var usage struct {
		Schema   string `json:"schema"`
		Captions []string
		Apps     []struct {
			ID string `json:"id"`
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, "apps", "font-usage.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &usage); err != nil {
		t.Fatal(err)
	}
	if usage.Schema != "fontcreator233/apps/v1" || len(usage.Apps) < 16 || len(usage.Captions) != 2 {
		t.Fatalf("font-usage = %#v", usage)
	}
}

func TestWebSpecimenWithExternalAssets(t *testing.T) {
	dir := t.TempDir()
	font := filepath.Join(dir, "Test-Regular.ttf")
	if err := os.WriteFile(font, []byte("font"), 0o644); err != nil {
		t.Fatal(err)
	}
	assets := t.TempDir()
	if err := os.WriteFile(filepath.Join(assets, "specimen.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	html := filepath.Join(dir, "Test-Regular.html")
	rel, err := filepath.Rel(dir, assets)
	if err != nil {
		t.Fatal(err)
	}
	body := `<link rel="stylesheet" href="./` + filepath.ToSlash(rel) + `/specimen.css">`
	if err := os.WriteFile(html, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), Options{
		OutDir:       dir,
		Family:       "Test",
		Style:        "Regular",
		License:      "OFL-1.1",
		Weight:       400,
		Targets:      []string{"web"},
		SpecimenPath: html,
		AssetsDir:    assets,
		Fonts:        []FontInput{{Path: font, Primary: true}},
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "web", "Test-Regular.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"./assets/specimen.css"`) {
		t.Fatalf("html still points at external assets: %s", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "web", "assets", "specimen.css")); err != nil {
		t.Fatal(err)
	}
}

func TestExpandTargets(t *testing.T) {
	all, err := expandTargets([]string{"all"})
	if err != nil || len(all) != 6 {
		t.Fatalf("all = %v err=%v", all, err)
	}
	if _, err := expandTargets([]string{"nope"}); err == nil {
		t.Fatal("expected unknown target error")
	}
}
