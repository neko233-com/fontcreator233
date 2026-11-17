package fontbuild

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFontToolsValidation(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	cfg := testConfig()
	data, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.ttf")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	script := `
import sys
from fontTools.ttLib import TTFont
f = TTFont(sys.argv[1])
required = {"head","hhea","maxp","hmtx","cmap","loca","glyf","name","OS/2","post"}
missing = required - set(f.keys())
if missing:
    raise SystemExit(f"missing tables: {missing}")
for name in f.getGlyphOrder():
    f["glyf"][name]
print("ok", len(f.getGlyphOrder()))
`
	cmd := exec.Command(python, "-c", script, path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools: %v\n%s", err, out)
	}
}
