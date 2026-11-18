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
	if err := exec.Command(python, "-c", "import fontTools").Run(); err != nil {
		t.Skip("fontTools not available")
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
if "GSUB" not in f or "liga" not in [r.FeatureTag for r in f["GSUB"].table.FeatureList.FeatureRecord]:
    raise SystemExit("GSUB liga missing")
if not {"case", "frac", "tnum", "ss01", "smcp", "onum", "zero", "sups", "subs"}.issubset({r.FeatureTag for r in f["GSUB"].table.FeatureList.FeatureRecord}):
    raise SystemExit("GSUB feature set incomplete")
gpos_tags = [r.FeatureTag for r in f["GPOS"].table.FeatureList.FeatureRecord] if "GPOS" in f else []
if "kern" not in gpos_tags or "mark" not in gpos_tags:
    raise SystemExit(f"GPOS kern/mark missing: {gpos_tags}")
if "kern" not in f or not f["kern"].kernTables[0].kernTable:
    raise SystemExit("legacy kern missing")
if "GDEF" not in f:
    raise SystemExit("GDEF missing")
if not any(l.LookupType == 4 and "f" in l.SubTable[0].ligatures for l in f["GSUB"].table.LookupList.Lookup):
    raise SystemExit("liga lookup missing")
print("ok", len(f.getGlyphOrder()), len(f.getBestCmap()))
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools: %v\n%s", err, out)
	}
}
