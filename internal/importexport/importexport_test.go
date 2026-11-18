package importexport

import (
	"archive/zip"
	"context"
	"encoding/json"
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
	p, err := ImportFont(font, out, true, []rune{'A', 'B'}, false)
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

func TestImportFontScalesMetrics(t *testing.T) {
	out := t.TempDir()
	p, err := ImportFont(filepath.Join("testdata", "Fixture2048.ttf"), out, false, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if p.UnitsPerEm != 2048 {
		t.Fatalf("unitsPerEm = %d", p.UnitsPerEm)
	}
	if p.XHeight < 480 || p.XHeight > 520 {
		t.Fatalf("xHeight = %d, want 1000-scale ~500", p.XHeight)
	}
	if p.CapHeight < 690 || p.CapHeight > 740 {
		t.Fatalf("capHeight = %d, want 1000-scale ~716", p.CapHeight)
	}
	r, err := recipe.Load(p.RecipePath)
	if err != nil {
		t.Fatal(err)
	}
	if r.Metrics.XHeight < 0.48 || r.Metrics.XHeight > 0.52 {
		t.Fatalf("recipe xHeight = %v", r.Metrics.XHeight)
	}
	if r.Metrics.CapHeight < 0.69 || r.Metrics.CapHeight > 0.74 {
		t.Fatalf("recipe capHeight = %v", r.Metrics.CapHeight)
	}
}

func TestImportFontRequiresForceForNonEmptyDir(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "existing.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join("testdata", "Fixture2048.ttf")
	if _, err := ImportFont(fixture, out, false, nil, false); err == nil {
		t.Fatal("import overwrote a non-empty directory without force")
	}
	if _, err := ImportFont(fixture, out, false, nil, true); err != nil {
		t.Fatalf("forced import: %v", err)
	}
}

func TestExportGlyphsScalesNon1000UPM(t *testing.T) {
	dir := t.TempDir()
	paths, err := ExportGlyphsFromFont(filepath.Join("testdata", "Fixture2048.ttf"), dir, Filter{Unicodes: []rune{'A'}})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths = %v", paths)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `data-advance="667"`) {
		t.Fatalf("advance was not normalized to 1000 UPM:\n%s", data)
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
	imp, err := ImportBundle(context.Background(), zipPath, out, false, false)
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

func TestBundleIncludesGlyphOverrides(t *testing.T) {
	recipePath, _ := buildSample(t)
	recipeDir := filepath.Dir(recipePath)
	glyphDir := filepath.Join(recipeDir, "glyphs")
	if err := os.MkdirAll(glyphDir, 0o755); err != nil {
		t.Fatal(err)
	}
	glyph := `<svg xmlns="http://www.w3.org/2000/svg" data-advance="600"><path d="M 50 0 L 550 0 L 550 700 L 50 700 Z"/></svg>`
	if err := os.WriteFile(filepath.Join(glyphDir, "U+0041.svg"), []byte(glyph), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := recipe.Load(recipePath)
	if err != nil {
		t.Fatal(err)
	}
	r.Glyphs.Overrides = "glyphs"
	data, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recipePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "override.bundle.zip")
	if _, err := ExportBundle(context.Background(), recipePath, zipPath, []string{"universal"}); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	foundGlyph := false
	for _, f := range zr.File {
		if f.Name == "glyphs/U+0041.svg" {
			foundGlyph = true
		}
	}
	_ = zr.Close()
	if !foundGlyph {
		t.Fatal("bundle did not include glyphs/U+0041.svg")
	}
	out := t.TempDir()
	if _, err := ImportBundle(context.Background(), zipPath, out, true, false); err != nil {
		t.Fatal(err)
	}
	imported, err := recipe.Load(filepath.Join(out, "recipe.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if imported.Glyphs.Overrides != "glyphs" {
		t.Fatalf("overrides = %q", imported.Glyphs.Overrides)
	}
}

func TestBundleRejectsUnindexedFile(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "bad.zip")
	recipeData := []byte("apiVersion: fontcreator233/v1\nmeta:\n  family: Tiny\n  license: OFL-1.1\n")
	index := BundleIndex{
		Schema:  bundleSchema,
		License: "OFL-1.1",
		Files:   map[string]string{"recipe.yaml": hashBytes(recipeData)},
	}
	idxData, _ := json.Marshal(index)
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, data := range map[string][]byte{"bundle.json": idxData, "recipe.yaml": recipeData, "extra.txt": []byte("unindexed")} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if _, err := ImportBundle(context.Background(), zipPath, t.TempDir(), false, false); err == nil || !strings.Contains(err.Error(), "unindexed") {
		t.Fatalf("error = %v, want unindexed file", err)
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
	if _, err := ImportBundle(context.Background(), zipPath, t.TempDir(), false, false); err == nil {
		t.Fatal("zip slip accepted")
	}
}
