package specimen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func testOptions(t *testing.T) Options {
	t.Helper()
	dir := t.TempDir()
	font := filepath.Join(dir, "TestFont-Regular.ttf")
	if err := os.WriteFile(font, []byte("fake ttf for specimen"), 0o644); err != nil {
		t.Fatal(err)
	}
	return Options{
		OutDir:   dir,
		FontPath: font,
		FontFile: "TestFont-Regular.ttf",
		Data: Data{
			Family:      "Test Font",
			Style:       "Regular",
			Description: "Specimen test.",
			Version:     "1.0.0",
			SHA256:      "0123456789abcdef0123456789abcdef",
			RecipeName:  "test.recipe.yaml",
			Accent:      "#5B8CFF",
			SampleText:  "Aa 字体",
			Glyphs: []GlyphInfo{
				{Rune: 'A', Script: "latin"},
				{Rune: 'é', Script: "latin"},
				{Rune: '中', Script: "cjk"},
				{Rune: '한', Script: "hangul"},
				{Rune: 'あ', Script: "kana"},
				{Rune: '€', Script: "symbols"},
			},
			Coverage:     map[string]int{"latin": 3, "cjk": 1, "hangul": 1},
			HasKerning:   true,
			HasLigatures: true,
			HasFractions: true,
			HasCase:      true,
			HasTnum:      true,
			HasSS01:      true,
			KernPairs:    23,
		},
		SelfContained: false,
		Open:          false,
	}
}

func TestBuildSpecimen(t *testing.T) {
	o := testOptions(t)
	htmlPath, err := Build(o)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatal(err)
	}
	html := string(raw)
	for _, want := range []string{
		"@font-face", "Test Font", "Specimen",
		"./assets/specimen.css", "./assets/specimen.js",
		"./assets/vendor/tanstack-virtual.iife.js",
		"TestFont-Regular.glyphs.js", "data-accent=\"#5B8CFF\"",
		`id="asset-error"`, "__fontcreatorBoot",
		"IN YOUR APPS", "剪映 / CapCut", "CAPTION PREVIEW", "Download TTF",
		"OPENTYPE", "feature-liga", "feature-frac", "feature-case", "feature-tnum", "feature-ss01",
		"font-feature-settings",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML missing %q", want)
		}
	}
	if strings.Contains(html, `type="module"`) {
		t.Fatal("HTML must not use ES modules")
	}
	external := regexp.MustCompile(`(?i)(?:src|href)="(?:https?:)?//`)
	if external.MatchString(html) {
		t.Fatalf("HTML references external asset: %s", external.FindString(html))
	}
	glyphRaw, err := os.ReadFile(filepath.Join(o.OutDir, "TestFont-Regular.glyphs.js"))
	if err != nil {
		t.Fatal(err)
	}
	glyphJS := string(glyphRaw)
	if !strings.Contains(glyphJS, "window.FontCreatorData") || !strings.Contains(glyphJS, `"cps"`) {
		t.Fatalf("glyph JS = %s", glyphJS)
	}
	for _, rel := range []string{
		"assets/specimen.css", "assets/specimen.js",
		"assets/vendor/tanstack-virtual.iife.js",
		"assets/geist/Geist-Variable.woff2",
		"assets/geist/GeistMono-Variable.woff2",
		"assets/licenses/Geist-OFL.txt",
		"assets/licenses/TanStack-Virtual-MIT.txt",
		"assets/versions.json",
	} {
		if _, err := os.Stat(filepath.Join(o.OutDir, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing asset %s: %v", rel, err)
		}
	}
}

func TestEnsureAssetsIdempotent(t *testing.T) {
	dir := t.TempDir()
	v1, err := EnsureAssets(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	versions1, _ := os.ReadFile(filepath.Join(dir, "assets", "versions.json"))
	v2, err := EnsureAssets(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	versions2, _ := os.ReadFile(filepath.Join(dir, "assets", "versions.json"))
	if v1 != v2 || string(versions1) != string(versions2) {
		t.Fatal("assets are not idempotent")
	}
}

func TestSelfContained(t *testing.T) {
	o := testOptions(t)
	o.SelfContained = true
	htmlPath, err := Build(o)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(htmlPath)
	html := string(raw)
	if !strings.Contains(html, "data:font/ttf;base64,") {
		t.Fatal("missing inline font")
	}
	if strings.Contains(html, "./assets/specimen.css") {
		t.Fatal("self-contained HTML still references assets")
	}
	if !strings.Contains(html, "data:font/woff2;base64,") {
		t.Fatal("missing inline Geist font")
	}
}

func TestOpenInjectionAndCI(t *testing.T) {
	t.Setenv("CI", "")
	o := testOptions(t)
	o.Open = true
	called := ""
	o.OpenFunc = func(path string) error { called = path; return nil }
	path, err := Build(o)
	if err != nil {
		t.Fatal(err)
	}
	if called != path {
		t.Fatalf("opener called with %q, want %q", called, path)
	}

	t.Setenv("CI", "true")
	o = testOptions(t)
	o.Open = true
	called = ""
	o.OpenFunc = func(path string) error { called = path; return nil }
	if _, err := Build(o); err != nil {
		t.Fatal(err)
	}
	if called != "" {
		t.Fatal("opener called in CI")
	}
}

func TestOpenCommand(t *testing.T) {
	name, args := openCommand("windows", `C:\tmp\Rock & Roll.html`)
	if name != "rundll32.exe" || len(args) != 2 || args[0] != "url.dll,FileProtocolHandler" || !strings.HasPrefix(args[1], "file:///") {
		t.Fatalf("windows = %q %v", name, args)
	}
	if strings.Contains(args[1], "cmd") {
		t.Fatalf("windows command still uses a shell: %q", args[1])
	}
	name, args = openCommand("darwin", "/tmp/a.html")
	if name != "open" || len(args) != 1 {
		t.Fatalf("darwin = %q %v", name, args)
	}
	name, args = openCommand("linux", "/tmp/a.html")
	if name != "xdg-open" || len(args) != 1 {
		t.Fatalf("linux = %q %v", name, args)
	}
}
