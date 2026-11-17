package build

import (
	"context"
	"os/exec"
	"testing"
)

func TestFontToolsSample(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	r := sampleRecipe(t)
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	script := `
import sys
from fontTools.ttLib import TTFont
f = TTFont(sys.argv[1])
for name in f.getGlyphOrder():
    f["glyf"][name]
print("ok", len(f.getGlyphOrder()), len(f.getBestCmap()))
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools: %v\n%s", err, out)
	}
}
