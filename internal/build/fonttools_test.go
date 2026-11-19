package build

import (
	"context"
	"os/exec"
	"testing"
)

func TestFontToolsVariable(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	if err := exec.Command(python, "-c", "import fontTools").Run(); err != nil {
		t.Skip("fontTools not available")
	}
	r := sampleRecipe(t)
	r.Output.Variable = true
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	script := `
import sys
from fontTools.ttLib import TTFont
f = TTFont(sys.argv[1])
if "fvar" not in f or "gvar" not in f:
    raise SystemExit("fvar/gvar missing")
axis = f["fvar"].axes[0]
if axis.axisTag != "wght" or not (axis.minValue < axis.defaultValue < axis.maxValue):
    raise SystemExit(f"bad wght axis: {axis}")
if "A" not in f["gvar"].variations:
    raise SystemExit("A has no gvar data")
coords = f["gvar"].variations["A"][0].coordinates
if not any(c is not None and (c[0] or c[1]) for c in coords):
    raise SystemExit("A has no non-zero variation delta")
print("ok", axis.minValue, axis.defaultValue, axis.maxValue)
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools variable: %v\n%s", err, out)
	}
}

func TestFontToolsCalt(t *testing.T) {
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
gsub = f["GSUB"].table
calt = [r for r in gsub.FeatureList.FeatureRecord if r.FeatureTag == "calt"]
if not calt or not calt[0].Feature.LookupListIndex:
    raise SystemExit("calt feature missing")
lookup = gsub.LookupList.Lookup[calt[0].Feature.LookupListIndex[0]]
if lookup.LookupType != 6:
    raise SystemExit(f"calt lookup type {lookup.LookupType}, want 6")
st = lookup.SubTable[0]
if st.Format != 3 or st.InputGlyphCount != 2 or st.SubstCount != 1:
    raise SystemExit(f"bad chain context: {st.Format} {st.InputGlyphCount} {st.SubstCount}")
if [c.glyphs for c in st.InputCoverage] != [["hyphen"], ["greater"]]:
    raise SystemExit(f"bad input coverages: {[c.glyphs for c in st.InputCoverage]}")
rec = st.SubstLookupRecord[0]
if rec.SequenceIndex != 0:
    raise SystemExit(f"bad sequence index: {rec.SequenceIndex}")
nested = gsub.LookupList.Lookup[rec.LookupListIndex]
if nested.LookupType != 4:
    raise SystemExit(f"nested lookup type {nested.LookupType}, want 4")
lig = nested.SubTable[0].ligatures["hyphen"][0]
if lig.Component != ["greater"]:
    raise SystemExit(f"bad ligature components: {lig.Component}")
print("ok", lookup.LookupType, rec.LookupListIndex)
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools calt: %v\n%s", err, out)
	}
}

func TestFontToolsColor(t *testing.T) {
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
if "COLR" not in f or "CPAL" not in f:
    raise SystemExit("COLR/CPAL missing")
layers = f["COLR"].ColorLayers
if "uniE000" not in layers or len(layers["uniE000"]) != 2:
    raise SystemExit(f"bad COLR layers: {layers}")
if layers["uniE000"][0].colorID != 0 or layers["uniE000"][1].colorID != 1:
    raise SystemExit(f"bad palette indices: {layers['uniE000']}")
palette = f["CPAL"].palettes[0]
if len(palette) < 3:
    raise SystemExit(f"bad CPAL palette: {palette}")
accent = palette[0]
if (accent.red, accent.green, accent.blue) != (0x5B, 0x8C, 0xFF):
    raise SystemExit(f"bad accent color: {accent.red},{accent.green},{accent.blue}")
print("ok", len(layers["uniE000"]), len(palette))
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools color: %v\n%s", err, out)
	}
}

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
if not {"case", "ccmp", "frac", "tnum", "ss01", "smcp", "onum", "zero", "sups", "subs"}.issubset({r.FeatureTag for r in f["GSUB"].table.FeatureList.FeatureRecord}):
    raise SystemExit("GSUB feature set incomplete")
gpos_tags = [r.FeatureTag for r in f["GPOS"].table.FeatureList.FeatureRecord] if "GPOS" in f else []
if "kern" not in gpos_tags or "mark" not in gpos_tags or "mkmk" not in gpos_tags:
    raise SystemExit(f"GPOS kern/mark/mkmk missing: {gpos_tags}")
if not any(l.LookupType == 6 for l in f["GPOS"].table.LookupList.Lookup):
    raise SystemExit("mkmk lookup missing")
ccmp = False
for lookup in f["GSUB"].table.LookupList.Lookup:
    if lookup.LookupType != 4:
        continue
    for st in lookup.SubTable:
        for first, ligs in st.ligatures.items():
            for lig in ligs:
                if first == "a" and lig.Component == ["acutecomb"]:
                    ccmp = True
if not ccmp:
    raise SystemExit("ccmp a+acute lookup missing")
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
