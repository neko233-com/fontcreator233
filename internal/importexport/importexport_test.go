package importexport

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/build"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"gopkg.in/yaml.v3"
)

func buildSample(t *testing.T) (recipePath, fontPath string) {
	t.Helper()
	r := recipe.Default()
	r.Meta.Family = "IO Test"
	r.Meta.License = "OFL-1.1"
	r.Coverage.Base.CJK = "sample"
	r.Output.Formats = []string{"ttf"}
	r.Output.Targets = []string{"universal"}
	r.Output.Specimen = false
	r.Output.Open = false
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	res, err := build.Build(context.Background(), r, build.Options{OutDir: dir, Targets: []string{"universal"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "font.recipe.yaml")
	// Re-marshal with YAML for the bundle path.
	data, err := marshalRecipe(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, res.FontPath
}

func marshalRecipe(r *recipe.Recipe) ([]byte, error) {
	return yaml.Marshal(r)
}

func TestImportFontProfile(t *testing.T) {
	_, font := buildSample(t)
	out := t.TempDir()
	p, err := ImportFont(font, out, true, []rune{'A', 'B'})
	if err != nil {
		t.Fatal(err)
	}
	if p.Family != "IO Test" || p.Glyphs < 100 || p.Coverage["latin"] == 0 {
		t.Fatalf("profile = %#v", p)
	}
	if _, err := os.Stat(filepath.Join(out, "profile.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := recipe.Load(p.RecipePath); err != nil {
		t.Fatalf("recipe: %v", err)
	}
	if len(p.Extracted) != 2 {
		t.Fatalf("extracted = %v", p.Extracted)
	}
	for _, name := range p.Extracted {
		if _, err := os.Stat(filepath.Join(out, "glyphs", name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExportGlyphs(t *testing.T) {
	_, font := buildSample(t)
	dir := t.TempDir()
	paths, err := ExportGlyphsFromFont(font, dir, Filter{Scripts: []string{"latin"}, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 5 {
		t.Fatalf("paths = %v", paths)
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(data), "data-advance") {
			t.Fatalf("%s: %v", p, err)
		}
	}
}

func TestBundleRoundTrip(t *testing.T) {
	recipePath, _ := buildSample(t)
	zipPath := filepath.Join(t.TempDir(), "test.bundle.zip")
	res, err := ExportBundle(context.Background(), recipePath, zipPath, []string{"universal"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Files < 5 || res.Bytes == 0 {
		t.Fatalf("export = %#v", res)
	}
	out := t.TempDir()
	imp, err := ImportBundle(zipPath, out, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(imp.Files) < 5 {
		t.Fatalf("import = %#v", imp)
	}
	if _, err := os.Stat(filepath.Join(out, "recipe.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "OFL.txt")); err != nil {
		t.Fatal(err)
	}
	font, err := filepath.Glob(filepath.Join(out, "*.ttf"))
	if err != nil || len(font) == 0 {
		t.Fatalf("font = %v %v", font, err)
	}
}

func TestBundleZipSlip(t *testing.T) {
	zipPath := filepath.Join(t.TempDir(), "bad.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("../evil.txt")
	_, _ = w.Write([]byte("evil"))
	_ = zw.Close()
	_ = f.Close()
	if _, err := ImportBundle(zipPath, t.TempDir(), false, false); err == nil {
		t.Fatal("zip slip accepted")
	}
}
