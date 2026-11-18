package fontlearn

import (
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/fontbuild"
	"github.com/neko233-com/fontcreator233/internal/geom"
)

func TestAnalyzeOpenTypeFont(t *testing.T) {
	cfg := fontbuild.Config{
		Family:      "Learn Test",
		Style:       "Regular",
		WeightClass: 550,
		Ascender:    800,
		Descender:   -200,
		CapHeight:   720,
		XHeight:     520,
		Glyphs: []fontbuild.Glyph{
			{Name: "uni0041", Rune: 'A', Advance: 600, Path: geom.RectPath(40, 0, 560, 700)},
			{Name: "uni0056", Rune: 'V', Advance: 600, Path: geom.RectPath(20, 0, 580, 700)},
			{Name: "uni0066", Rune: 'f', Advance: 300, Path: geom.RectPath(20, 0, 250, 700)},
			{Name: "uni0069", Rune: 'i', Advance: 260, Path: geom.RectPath(40, 0, 180, 700)},
			{Name: "f_i", NoCmap: true, Advance: 560, Path: geom.RectPath(20, 0, 500, 700)},
		},
		Kerning:   []fontbuild.KernPair{{Left: "uni0041", Right: "uni0056", Value: -80}},
		Ligatures: []fontbuild.Ligature{{Components: []string{"uni0066", "uni0069"}, Glyph: "f_i"}},
	}
	data, err := fontbuild.Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r, err := Analyze("learn-test.ttf", data)
	if err != nil {
		t.Fatal(err)
	}
	if r.Family != "Learn Test" || r.WeightClass != 550 {
		t.Fatalf("report = %#v", r)
	}
	if r.Kerning != 1 || len(r.SampleKern) != 1 {
		t.Fatalf("kerning = %#v", r.SampleKern)
	}
	hasLiga, hasKern := false, false
	for _, f := range r.Features {
		if f.Table == "GSUB" && f.Tag == "liga" && len(f.LookupType) == 1 && f.LookupType[0] == 4 {
			hasLiga = true
		}
		if f.Table == "GPOS" && f.Tag == "kern" && len(f.LookupType) == 1 && f.LookupType[0] == 2 {
			hasKern = true
		}
	}
	if !hasLiga || !hasKern {
		t.Fatalf("features = %#v", r.Features)
	}
	if !strings.Contains(r.Markdown(), "GSUB") || !strings.Contains(r.Markdown(), "Suggested recipe") {
		t.Fatal("markdown report missing sections")
	}
}
