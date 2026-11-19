package agentschema

import (
	"strings"
	"testing"
)

func TestDescribeOpenTypeContract(t *testing.T) {
	ot := Describe()["opentype"].(map[string]any)
	generated := strings.Join(ot["generated"].([]string), " ")
	for _, tag := range []string{"liga", "kern", "ccmp", "calt", "mark", "mkmk", "smcp", "onum", "zero", "sups", "subs"} {
		if !strings.Contains(generated, tag) {
			t.Fatalf("generated features missing %q: %s", tag, generated)
		}
	}
	tables := strings.Join(ot["tables"].([]string), " ")
	for _, want := range []string{"fvar", "gvar", "COLR", "CPAL"} {
		if !strings.Contains(tables, want) {
			t.Fatalf("tables missing %q: %s", want, tables)
		}
	}
	if variable, ok := ot["variable"].(string); !ok || !strings.Contains(variable, "wght") {
		t.Fatalf("variable axis contract missing: %#v", ot["variable"])
	}
	if color, ok := ot["color"].(string); !ok || !strings.Contains(color, "COLR") {
		t.Fatalf("color contract missing: %#v", ot["color"])
	}
	if contextual, ok := ot["contextual"].(string); !ok || !strings.Contains(contextual, "openType.contextual") {
		t.Fatalf("contextual contract missing: %#v", ot["contextual"])
	}
	unsupported := strings.Join(ot["unsupported"].([]string), " ")
	for _, want := range []string{"variable", "color", "contextual", "cursive"} {
		if !strings.Contains(unsupported, want) {
			t.Fatalf("unsupported list missing %q: %s", want, unsupported)
		}
	}
	md := Markdown()
	for _, want := range []string{"## OpenType", "unsupported", "ccmp", "mkmk"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q", want)
		}
	}
}
