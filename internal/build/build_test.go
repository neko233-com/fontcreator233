package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/derive"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/unicodes"
	"golang.org/x/image/font/sfnt"
)

func sampleRecipe(t *testing.T) *recipe.Recipe {
	t.Helper()
	r := recipe.Default()
	r.Meta.Family = "Build Test"
	r.Meta.License = "OFL-1.1"
	r.Output.Formats = []string{"ttf"}
	r.Output.Targets = []string{"universal"}
	r.Output.Specimen = false
	r.Output.Open = false
	r.Coverage.Base.CJK = "sample"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return r
}

func readFile(path string) ([]byte, error) { return os.ReadFile(path) }

func TestBuildSample(t *testing.T) {
	r := sampleRecipe(t)
	dir := t.TempDir()
	res, err := Build(context.Background(), r, Options{OutDir: dir}, nil)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if res.FontPath == "" || res.SHA256 == "" || res.Bytes == 0 {
		t.Fatalf("result = %#v", res)
	}
	if res.Style != "Regular" {
		t.Fatalf("style = %q", res.Style)
	}
	data, err := readFile(res.FontPath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf sfnt.Buffer
	for _, rn := range []rune{'A', 'é', 'Ω', 'Ж', '中', 'あ', '한', '€', '→'} {
		gid, err := f.GlyphIndex(&buf, rn)
		if err != nil || gid == 0 {
			t.Fatalf("%U missing (gid=%d err=%v)", rn, gid, err)
		}
	}
	if res.Glyphs < 100 {
		t.Fatalf("glyphs = %d", res.Glyphs)
	}
	if res.Coverage["cjk"] == 0 || res.Coverage["hangul"] == 0 {
		t.Fatalf("coverage = %#v", res.Coverage)
	}
	name, _ := f.Name(&buf, sfnt.NameIDFamily)
	if name != r.Meta.Family {
		t.Fatalf("family = %q", name)
	}
}

func TestBuildDeterministic(t *testing.T) {
	r := sampleRecipe(t)
	a, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.SHA256 != b.SHA256 {
		t.Fatalf("hashes differ: %s %s", a.SHA256, b.SHA256)
	}
}

func TestBuildSpecimenOutput(t *testing.T) {
	r := sampleRecipe(t)
	r.Output.Specimen = true
	r.Output.Open = false
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.SpecimenPath == "" {
		t.Fatal("missing specimen path")
	}
	raw, err := readFile(res.SpecimenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "@font-face") || !strings.Contains(string(raw), "Build Test") {
		t.Fatal("specimen HTML incomplete")
	}
}

func TestItalicStyleResolution(t *testing.T) {
	r := sampleRecipe(t)
	r.Meta.Style = "Noir"
	r.Style.Slant = 8
	r.Output.Specimen = false
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Style != "Noir Italic" || res.RequestedStyle != "Noir" {
		t.Fatalf("styles = %q / %q", res.Style, res.RequestedStyle)
	}
	data, _ := readFile(res.FontPath)
	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	sub, _ := f.Name(&buf, sfnt.NameIDSubfamily)
	if sub != "Noir Italic" {
		t.Fatalf("subfamily = %q", sub)
	}
}

func TestCommonCoverageReport(t *testing.T) {
	if testing.Short() {
		t.Skip("common coverage is slow in -short")
	}
	r := sampleRecipe(t)
	r.Coverage.Base.CJK = "common"
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	st := res.CoverageReport["cjk"]
	if st.Found < 20976 || st.Missing <= 0 || st.Requested != st.Found+st.Missing {
		t.Fatalf("cjk stat = %#v", st)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "base coverage cjk") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing warning: %v", res.Warnings)
	}
}

func TestSplitCJK(t *testing.T) {
	r := sampleRecipe(t)
	dir := t.TempDir()
	res, err := Build(context.Background(), r, Options{OutDir: dir, SplitCJK: 32}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Split) < 2 {
		t.Fatalf("split = %v", res.Split)
	}
	requested, err := derive.CoverageRunes("sample")
	if err != nil {
		t.Fatal(err)
	}
	var cjk []rune
	for _, rn := range requested {
		if unicodes.IsCJK(rn) {
			cjk = append(cjk, rn)
		}
	}
	fonts := make([]*sfnt.Font, 0, len(res.Split))
	for _, path := range res.Split {
		data, _ := readFile(path)
		f, err := sfnt.Parse(data)
		if err != nil {
			t.Fatalf("parse split %s: %v", filepath.Base(path), err)
		}
		fonts = append(fonts, f)
	}
	var buf sfnt.Buffer
	for _, rn := range cjk {
		count := 0
		for _, f := range fonts {
			gid, _ := f.GlyphIndex(&buf, rn)
			if gid != 0 {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("%U appears in %d split fonts, want 1", rn, count)
		}
	}
}

func TestBuildCanceled(t *testing.T) {
	r := sampleRecipe(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Build(ctx, r, Options{OutDir: t.TempDir()}, nil)
	if err == nil {
		t.Fatal("expected cancellation")
	}
}
