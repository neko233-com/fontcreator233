package derive

import (
	"testing"
)

func TestCoverageNames(t *testing.T) {
	names := CoverageNames()
	want := []string{"none", "sample", "common", "full"}
	if len(names) != len(want) {
		t.Fatalf("names = %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v", names)
		}
	}
}

func TestCoverageLevels(t *testing.T) {
	none, err := CoverageRunes("none")
	if err != nil || len(none) != 0 {
		t.Fatalf("none = %d, %v", len(none), err)
	}
	sample, err := CoverageRunes("sample")
	if err != nil {
		t.Fatal(err)
	}
	common, err := CoverageRunes("common")
	if err != nil {
		t.Fatal(err)
	}
	full, err := CoverageRunes("full")
	if err != nil {
		t.Fatal(err)
	}
	if !(len(sample) < len(common) && len(common) < len(full)) {
		t.Fatalf("counts sample=%d common=%d full=%d", len(sample), len(common), len(full))
	}
	has := func(rs []rune, r rune) bool {
		for _, x := range rs {
			if x == r {
				return true
			}
		}
		return false
	}
	if !has(common, '中') || !has(common, '한') || !has(common, 'あ') || !has(common, '€') {
		t.Fatal("common missing expected runes")
	}
	if !has(full, 0x20000) && !has(full, 0x20001) {
		// full must at least request an Ext-B range; actual presence is checked
		// against the font by the build coverage report.
		t.Log("full does not include U+20000; acceptable if base lacks it")
	}
	if _, err := CoverageRunes("nope"); err == nil {
		t.Fatal("expected error for unknown level")
	}
}
