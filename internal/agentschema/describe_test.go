package agentschema

import (
	"strings"
	"testing"
)

func TestDescribeOpenTypeContract(t *testing.T) {
	ot := Describe()["opentype"].(map[string]any)
	generated := strings.Join(ot["generated"].([]string), " ")
	for _, tag := range []string{"liga", "kern", "ccmp", "mark", "mkmk", "smcp", "onum", "zero", "sups", "subs"} {
		if !strings.Contains(generated, tag) {
			t.Fatalf("generated features missing %q: %s", tag, generated)
		}
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
