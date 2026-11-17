package inspect

import (
	"context"
	"os"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/build"
	"github.com/neko233-com/fontcreator233/internal/recipe"
)

func TestInspectGeneratedFont(t *testing.T) {
	r := recipe.Default()
	r.Meta.Family = "Inspect Test"
	r.Meta.License = "OFL-1.1"
	r.Output.Formats = []string{"ttf"}
	r.Output.Targets = []string{"universal"}
	r.Output.Specimen = false
	r.Coverage.Base.CJK = "sample"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	res, err := build.Build(context.Background(), r, build.Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(res.FontPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := Inspect(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.Family != "Inspect Test" || info.Glyphs < 100 {
		t.Fatalf("info = %#v", info)
	}
	if info.UnitsPerEm != 1000 || info.SHA256 == "" {
		t.Fatalf("info = %#v", info)
	}
	for _, tag := range []string{"head", "glyf", "cmap", "name", "OS/2"} {
		found := false
		for _, t := range info.Tables {
			if t == tag {
				found = true
			}
		}
		if !found {
			t.Fatalf("table %s missing from %v", tag, info.Tables)
		}
	}
	if info.Coverage["cjk"] == 0 || info.Coverage["hangul"] == 0 {
		t.Fatalf("coverage = %#v", info.Coverage)
	}
}
