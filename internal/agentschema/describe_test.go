package agentschema

import (
	"strings"
	"testing"
)

func TestDescribeOpenTypeContract(t *testing.T) {
	ot := Describe()["opentype"].(map[string]any)
	generated := strings.Join(ot["generated"].([]string), " ")
	for _, tag := range []string{"liga", "kern", "ccmp", "calt", "curs", "mark", "mkmk", "smcp", "onum", "zero", "sups", "subs"} {
		if !strings.Contains(generated, tag) {
			t.Fatalf("generated features missing %q: %s", tag, generated)
		}
	}
	tables := strings.Join(ot["tables"].([]string), " ")
	for _, want := range []string{"fvar", "gvar", "HVAR", "MVAR", "COLR", "CPAL", "SVG "} {
		if !strings.Contains(tables, want) {
			t.Fatalf("tables missing %q: %s", want, tables)
		}
	}
	if variable, ok := ot["variable"].(string); !ok || !strings.Contains(variable, "wght") || !strings.Contains(variable, "HVAR") || !strings.Contains(variable, "MVAR") {
		t.Fatalf("variable axis contract missing: %#v", ot["variable"])
	}
	if color, ok := ot["color"].(string); !ok || !strings.Contains(color, "COLR") || !strings.Contains(color, "openType.color") {
		t.Fatalf("color contract missing: %#v", ot["color"])
	}
	if contextual, ok := ot["contextual"].(string); !ok || !strings.Contains(contextual, "openType.contextual") {
		t.Fatalf("contextual contract missing: %#v", ot["contextual"])
	}
	if cursive, ok := ot["cursive"].(string); !ok || !strings.Contains(cursive, "GPOS") {
		t.Fatalf("cursive contract missing: %#v", ot["cursive"])
	}
	unsupported := strings.Join(ot["unsupported"].([]string), " ")
	for _, want := range []string{"variable", "color", "contextual"} {
		if !strings.Contains(unsupported, want) {
			t.Fatalf("unsupported list missing %q: %s", want, unsupported)
		}
	}
	if strings.Contains(unsupported, "cursive") {
		t.Fatalf("cursive must no longer be unsupported: %s", unsupported)
	}
	platforms := Describe()["platforms"].(map[string]any)
	if bm, ok := platforms["bmfont"].(string); !ok || !strings.Contains(bm, "BMFont") {
		t.Fatalf("bmfont platform contract missing: %#v", platforms["bmfont"])
	}
	md := Markdown()
	for _, want := range []string{"## OpenType", "unsupported", "ccmp", "mkmk", "bmfont"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}
