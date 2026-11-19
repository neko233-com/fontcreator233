package build

import (
	"context"
	"os/exec"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/recipe"
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

func TestFontToolsContextualRecipe(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	if err := exec.Command(python, "-c", "import fontTools").Run(); err != nil {
		t.Skip("fontTools not available")
	}
	r := sampleRecipe(t)
	r.OpenType.Contextual = []recipe.ContextualRule{{
		Feature: "calt",
		Input:   "<-",
		Target:  "←",
	}}
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	script := `
import sys
from fontTools.ttLib import TTFont
f = TTFont(sys.argv[1])
gsub = f["GSUB"].table
calt = next(r for r in gsub.FeatureList.FeatureRecord if r.FeatureTag == "calt")
found = False
for li in calt.Feature.LookupListIndex:
    lookup = gsub.LookupList.Lookup[li]
    if lookup.LookupType != 6:
        continue
    st = lookup.SubTable[0]
    if [c.glyphs for c in st.InputCoverage] != [["less"], ["hyphen"]]:
        continue
    nested = gsub.LookupList.Lookup[st.SubstLookupRecord[0].LookupListIndex]
    lig = nested.SubTable[0].ligatures["less"][0]
    if lig.Component == ["hyphen"] and lig.LigGlyph == "arrowleft":
        found = True
if not found:
    raise SystemExit("recipe contextual rule missing")
print("ok", calt.Feature.LookupListIndex)
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools contextual recipe: %v\n%s", err, out)
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

func TestFontToolsCursive(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	if err := exec.Command(python, "-c", "import fontTools").Run(); err != nil {
		t.Skip("fontTools not available")
	}
	r := sampleRecipe(t)
	r.Output.Cursive = true
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	script := `
import sys
from fontTools.ttLib import TTFont
f = TTFont(sys.argv[1])
tags = [r.FeatureTag for r in f["GPOS"].table.FeatureList.FeatureRecord]
if "curs" not in tags:
    raise SystemExit(f"curs feature missing: {tags}")
curs = [l for l in f["GPOS"].table.LookupList.Lookup if l.LookupType == 3]
if len(curs) != 1:
    raise SystemExit(f"expected one cursive lookup, got {len(curs)}")
st = curs[0].SubTable[0]
if st.Format != 1 or st.EntryExitCount < 2:
    raise SystemExit(f"bad cursive subtable: format {st.Format}, count {st.EntryExitCount}")
glyphs = st.Coverage.glyphs
if "a" not in glyphs or "z" not in glyphs:
    raise SystemExit(f"cursive coverage incomplete: {glyphs[:5]}...{glyphs[-5:]}")
rec = st.EntryExitRecord[0]
entry, exit = rec.EntryAnchor, rec.ExitAnchor
if entry.Format != 1 or exit.Format != 1:
    raise SystemExit("expected format 1 anchors")
if entry.XCoordinate != 0 or entry.YCoordinate <= 0:
    raise SystemExit(f"bad entry anchor: {entry.XCoordinate},{entry.YCoordinate}")
if exit.XCoordinate <= 0 or exit.XCoordinate >= 1000 or exit.YCoordinate != entry.YCoordinate:
    raise SystemExit(f"bad exit anchor: {exit.XCoordinate},{exit.YCoordinate}")
print("ok", st.EntryExitCount, exit.XCoordinate, entry.YCoordinate)
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("fontTools cursive: %v\n%s", err, out)
	}
}

func TestHarfBuzzCursive(t *testing.T) {
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	if err := exec.Command(python, "-c", "import uharfbuzz").Run(); err != nil {
		t.Skip("uharfbuzz not available")
	}
	r := sampleRecipe(t)
	r.Output.Cursive = true
	res, err := Build(context.Background(), r, Options{OutDir: t.TempDir()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	script := `
import sys
import uharfbuzz as hb
p = sys.argv[1]
face = hb.Face(hb.Blob.from_file_path(p))
font = hb.Font(face)
def shape(text, curs):
    b = hb.Buffer()
    b.add_str(text)
    b.guess_segment_properties()
    hb.shape(font, b, {"curs": curs})
    return [(i.codepoint, pos.x_advance) for i, pos in zip(b.glyph_infos, b.glyph_positions)]
off = shape("abc", False)
on = shape("abc", True)
if on == off:
    raise SystemExit("curs feature had no shaping effect")
# Uppercase is outside cursive coverage and must keep its advance.
h_off = shape("H", False)
h_on = shape("H", True)
if h_off != h_on:
    raise SystemExit("curs unexpectedly changed uncovered glyph")
print("ok", off, on)
`
	out, err := exec.Command(python, "-c", script, res.FontPath).CombinedOutput()
	if err != nil {
		t.Fatalf("harfbuzz cursive: %v\n%s", err, out)
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
