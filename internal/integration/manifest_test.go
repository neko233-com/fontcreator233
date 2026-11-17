package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRangesAndScripts(t *testing.T) {
	ranges := Ranges([]rune{'A', 'B', 'C', 'E', '中', '文'})
	if len(ranges) != 4 {
		t.Fatalf("ranges = %#v", ranges)
	}
	if ranges[0] != (Range{'A', 'C'}) || ranges[1] != (Range{'E', 'E'}) {
		t.Fatalf("ranges = %#v", ranges)
	}
	scripts := Scripts([]rune{'A', 'é', 'Ω', 'Ж', '中', 'あ', '한', '€'})
	if strings.Join(scripts, ",") != "cjk,cyrillic,greek,hangul,kana,latin,symbols" {
		t.Fatalf("scripts = %v", scripts)
	}
	charsets := Charsets([]rune{'A', 'B', 'A', '中'})
	if charsets["latin"] != "AB" || charsets["cjk"] != "中" {
		t.Fatalf("charsets = %#v", charsets)
	}
}

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{
		Schema:    "fontcreator233/integration/v1",
		Generator: "test",
		Family:    "Test",
		Style:     "Regular",
		License:   "OFL-1.1",
		Fonts:     []Font{{Path: "Test.ttf", SHA256: "abc", Primary: true, Scripts: []string{"latin"}, Ranges: []Range{{65, 90}}}},
	}
	path := filepath.Join(dir, "manifest.json")
	if err := WriteJSON(path, m); err != nil {
		t.Fatal(err)
	}
	got, err := ReadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Family != "Test" || len(got.Fonts) != 1 || got.Fonts[0].Ranges[0] != (Range{65, 90}) {
		t.Fatalf("got = %#v", got)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"fontcreator233/integration/v1"`) {
		t.Fatalf("json = %s", data)
	}
}
