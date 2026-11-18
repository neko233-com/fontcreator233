package fontlearn

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/fontbuild"
	"github.com/neko233-com/fontcreator233/internal/geom"
)

func TestParseFvarAxis(t *testing.T) {
	data := make([]byte, 36)
	binary.BigEndian.PutUint32(data[0:], 0x00010000)
	binary.BigEndian.PutUint16(data[4:], 16) // offset to axis records
	binary.BigEndian.PutUint16(data[6:], 2)  // countSizePairs
	binary.BigEndian.PutUint16(data[8:], 1)  // axis count
	binary.BigEndian.PutUint16(data[10:], 20)
	copy(data[16:20], "wght")
	binary.BigEndian.PutUint32(data[20:], uint32(300<<16))
	binary.BigEndian.PutUint32(data[24:], uint32(400<<16))
	binary.BigEndian.PutUint32(data[28:], uint32(700<<16))
	binary.BigEndian.PutUint16(data[34:], 256)
	axes := parseFvar(data)
	if len(axes) != 1 {
		t.Fatalf("axes = %#v", axes)
	}
	a := axes[0]
	if a.Tag != "wght" || a.Min != 300 || a.Default != 400 || a.Max != 700 || a.NameID != 256 {
		t.Fatalf("axis = %#v", a)
	}
}

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
